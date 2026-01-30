package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/application/telegram/admin/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/telegram/admin"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// AlertCooldownMinutes is the cooldown period for alert deduplication
	AlertCooldownMinutes = 30
)

// TelegramMessageSender sends messages via Telegram (HTML format)
type TelegramMessageSender interface {
	SendMessage(chatID int64, text string) error
	SendMessageWithInlineKeyboard(chatID int64, text string, keyboard any) error
}

// NodeOfflineChecker defines the interface for checking offline nodes
type NodeOfflineChecker interface {
	// FindOfflineNodes returns nodes that haven't reported in the specified duration
	FindOfflineNodes(ctx context.Context, threshold time.Duration) ([]*node.Node, error)
}

// AgentOfflineChecker defines the interface for checking offline agents
type AgentOfflineChecker interface {
	// FindOfflineAgents returns agents that haven't reported in the specified duration
	FindOfflineAgents(ctx context.Context, threshold time.Duration) ([]*forward.ForwardAgent, error)
}

// CheckOfflineUseCase handles offline detection and alerting for nodes and agents
type CheckOfflineUseCase struct {
	bindingRepo       admin.AdminTelegramBindingRepository
	nodeRepo          node.NodeRepository
	agentRepo         forward.AgentRepository
	alertDeduplicator *cache.AlertDeduplicator
	botService        TelegramMessageSender
	logger            logger.Interface
}

// NewCheckOfflineUseCase creates a new CheckOfflineUseCase
func NewCheckOfflineUseCase(
	bindingRepo admin.AdminTelegramBindingRepository,
	nodeRepo node.NodeRepository,
	agentRepo forward.AgentRepository,
	alertDeduplicator *cache.AlertDeduplicator,
	botService TelegramMessageSender,
	logger logger.Interface,
) *CheckOfflineUseCase {
	return &CheckOfflineUseCase{
		bindingRepo:       bindingRepo,
		nodeRepo:          nodeRepo,
		agentRepo:         agentRepo,
		alertDeduplicator: alertDeduplicator,
		botService:        botService,
		logger:            logger,
	}
}

// CheckAndNotify checks for offline nodes and agents, then sends alerts
func (uc *CheckOfflineUseCase) CheckAndNotify(ctx context.Context) error {
	if uc.botService == nil {
		uc.logger.Debugw("offline check skipped: bot service not available")
		return nil
	}

	nodeCount, nodeErrors := uc.checkNodeOffline(ctx)
	agentCount, agentErrors := uc.checkAgentOffline(ctx)

	uc.logger.Infow("offline check completed",
		"node_alerts_sent", nodeCount,
		"agent_alerts_sent", agentCount,
		"node_errors", nodeErrors,
		"agent_errors", agentErrors,
	)

	return nil
}

func (uc *CheckOfflineUseCase) checkNodeOffline(ctx context.Context) (int, int) {
	alertsSent := 0
	errors := 0

	// Get bindings that want node offline notifications
	bindings, err := uc.bindingRepo.FindBindingsForNodeOfflineNotification(ctx)
	if err != nil {
		uc.logger.Errorw("failed to find bindings for node offline notification", "error", err)
		return 0, 1
	}

	if len(bindings) == 0 {
		return 0, 0
	}

	// Find the minimum threshold among all bindings to catch all potentially offline nodes
	minThreshold := time.Duration(bindings[0].OfflineThresholdMinutes()) * time.Minute
	for _, b := range bindings[1:] {
		t := time.Duration(b.OfflineThresholdMinutes()) * time.Minute
		if t < minThreshold {
			minThreshold = t
		}
	}

	// Find all nodes that are offline based on minimum threshold
	offlineNodes, err := uc.findOfflineNodes(ctx, minThreshold)
	if err != nil {
		uc.logger.Errorw("failed to find offline nodes", "error", err)
		return 0, 1
	}

	cooldown := time.Duration(AlertCooldownMinutes) * time.Minute

	for _, nodeInfo := range offlineNodes {
		// Skip if notification is muted for this node
		if nodeInfo.MuteNotification {
			uc.logger.Debugw("node offline notification skipped: muted",
				"node_sid", nodeInfo.SID,
				"node_name", nodeInfo.Name,
			)
			continue
		}

		// Atomically check and acquire alert lock to prevent duplicate alerts
		// in multi-instance deployments (TOCTOU-safe)
		acquired, err := uc.alertDeduplicator.TryAcquireAlertLock(ctx, cache.AlertTypeNodeOffline, nodeInfo.ID, cooldown)
		if err != nil {
			uc.logger.Errorw("failed to acquire alert lock", "error", err)
			errors++
			continue
		}

		if !acquired {
			continue // Skip - already alerted recently or another instance is handling
		}

		// Build message and keyboard once for this node
		message := BuildNodeOfflineMessageFromDTO(nodeInfo)
		keyboard := BuildMuteKeyboard("node", nodeInfo.SID)
		sentToAny := false

		// Send to ALL bindings whose threshold is met by this node's offline duration
		for i, binding := range bindings {
			if !binding.NotifyNodeOffline() {
				continue
			}

			// Check if this node's offline duration meets this binding's threshold
			bindingThreshold := time.Duration(binding.OfflineThresholdMinutes()) * time.Minute
			if nodeInfo.OfflineMinutes < int64(bindingThreshold.Minutes()) {
				continue // This binding's threshold is higher than node's offline time
			}

			if err := uc.botService.SendMessageWithInlineKeyboard(binding.TelegramUserID(), message, keyboard); err != nil {
				uc.logger.Errorw("failed to send node offline notification",
					"telegram_user_id", binding.TelegramUserID(),
					"node_id", nodeInfo.ID,
					"error", err,
				)
				errors++
				continue
			}

			sentToAny = true
			alertsSent++

			// Rate limiting between messages
			if i < len(bindings)-1 {
				time.Sleep(50 * time.Millisecond)
			}
		}

		// If failed to send to any binding, clear the lock so it can be retried
		if !sentToAny {
			_ = uc.alertDeduplicator.ClearAlert(ctx, cache.AlertTypeNodeOffline, nodeInfo.ID)
		}
	}

	return alertsSent, errors
}

func (uc *CheckOfflineUseCase) checkAgentOffline(ctx context.Context) (int, int) {
	alertsSent := 0
	errors := 0

	// Get bindings that want agent offline notifications
	bindings, err := uc.bindingRepo.FindBindingsForAgentOfflineNotification(ctx)
	if err != nil {
		uc.logger.Errorw("failed to find bindings for agent offline notification", "error", err)
		return 0, 1
	}

	if len(bindings) == 0 {
		return 0, 0
	}

	// Find the minimum threshold among all bindings to catch all potentially offline agents
	minThreshold := time.Duration(bindings[0].OfflineThresholdMinutes()) * time.Minute
	for _, b := range bindings[1:] {
		t := time.Duration(b.OfflineThresholdMinutes()) * time.Minute
		if t < minThreshold {
			minThreshold = t
		}
	}

	// Find all agents that are offline based on minimum threshold
	offlineAgents, err := uc.findOfflineAgents(ctx, minThreshold)
	if err != nil {
		uc.logger.Errorw("failed to find offline agents", "error", err)
		return 0, 1
	}

	cooldown := time.Duration(AlertCooldownMinutes) * time.Minute

	for _, agentInfo := range offlineAgents {
		// Skip if notification is muted for this agent
		if agentInfo.MuteNotification {
			uc.logger.Debugw("agent offline notification skipped: muted",
				"agent_sid", agentInfo.SID,
				"agent_name", agentInfo.Name,
			)
			continue
		}

		// Atomically check and acquire alert lock to prevent duplicate alerts
		// in multi-instance deployments (TOCTOU-safe)
		acquired, err := uc.alertDeduplicator.TryAcquireAlertLock(ctx, cache.AlertTypeAgentOffline, agentInfo.ID, cooldown)
		if err != nil {
			uc.logger.Errorw("failed to acquire alert lock", "error", err)
			errors++
			continue
		}

		if !acquired {
			continue // Skip - already alerted recently or another instance is handling
		}

		// Build message and keyboard once for this agent
		message := BuildAgentOfflineMessageFromDTO(agentInfo)
		keyboard := BuildMuteKeyboard("agent", agentInfo.SID)
		sentToAny := false

		// Send to ALL bindings whose threshold is met by this agent's offline duration
		for i, binding := range bindings {
			if !binding.NotifyAgentOffline() {
				continue
			}

			// Check if this agent's offline duration meets this binding's threshold
			bindingThreshold := time.Duration(binding.OfflineThresholdMinutes()) * time.Minute
			if agentInfo.OfflineMinutes < int64(bindingThreshold.Minutes()) {
				continue // This binding's threshold is higher than agent's offline time
			}

			if err := uc.botService.SendMessageWithInlineKeyboard(binding.TelegramUserID(), message, keyboard); err != nil {
				uc.logger.Errorw("failed to send agent offline notification",
					"telegram_user_id", binding.TelegramUserID(),
					"agent_id", agentInfo.ID,
					"error", err,
				)
				errors++
				continue
			}

			sentToAny = true
			alertsSent++

			// Rate limiting between messages
			if i < len(bindings)-1 {
				time.Sleep(50 * time.Millisecond)
			}
		}

		// If failed to send to any binding, clear the lock so it can be retried
		if !sentToAny {
			_ = uc.alertDeduplicator.ClearAlert(ctx, cache.AlertTypeAgentOffline, agentInfo.ID)
		}
	}

	return alertsSent, errors
}

func (uc *CheckOfflineUseCase) findOfflineNodes(ctx context.Context, threshold time.Duration) ([]dto.OfflineNodeInfo, error) {
	now := biztime.NowUTC()
	cutoff := now.Add(-threshold)

	// List all active nodes
	nodes, _, err := uc.nodeRepo.List(ctx, node.NodeFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	var offlineNodes []dto.OfflineNodeInfo
	for _, n := range nodes {
		// Only check nodes that have reported at least once
		lastSeen := n.LastSeenAt()
		if lastSeen == nil {
			continue
		}

		// Check if last seen is before cutoff
		if lastSeen.Before(cutoff) {
			offlineMinutes := int64(now.Sub(*lastSeen).Minutes())
			offlineNodes = append(offlineNodes, dto.OfflineNodeInfo{
				ID:               n.ID(),
				SID:              n.SID(),
				Name:             n.Name(),
				LastSeenAt:       lastSeen,
				OfflineMinutes:   offlineMinutes,
				MuteNotification: n.MuteNotification(),
			})
		}
	}

	return offlineNodes, nil
}

func (uc *CheckOfflineUseCase) findOfflineAgents(ctx context.Context, threshold time.Duration) ([]dto.OfflineAgentInfo, error) {
	now := biztime.NowUTC()
	cutoff := now.Add(-threshold)

	// List all enabled agents
	agents, _, err := uc.agentRepo.List(ctx, forward.AgentListFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	var offlineAgents []dto.OfflineAgentInfo
	for _, a := range agents {
		// Only check enabled agents
		if !a.IsEnabled() {
			continue
		}

		// Only check agents that have reported at least once
		lastSeen := a.LastSeenAt()
		if lastSeen == nil {
			continue
		}

		// Check if last seen is before cutoff
		if lastSeen.Before(cutoff) {
			offlineMinutes := int64(now.Sub(*lastSeen).Minutes())
			offlineAgents = append(offlineAgents, dto.OfflineAgentInfo{
				ID:               a.ID(),
				SID:              a.SID(),
				Name:             a.Name(),
				LastSeenAt:       lastSeen,
				OfflineMinutes:   offlineMinutes,
				MuteNotification: a.MuteNotification(),
			})
		}
	}

	return offlineAgents, nil
}

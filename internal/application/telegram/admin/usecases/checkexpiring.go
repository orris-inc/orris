package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/application/telegram/admin/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/telegram/admin"
	telegram "github.com/orris-inc/orris/internal/infrastructure/telegram"
	"github.com/orris-inc/orris/internal/infrastructure/telegram/i18n"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// CheckExpiringUseCase handles resource expiration detection and notification
type CheckExpiringUseCase struct {
	bindingRepo admin.AdminTelegramBindingRepository
	nodeRepo    node.NodeRepository
	agentRepo   forward.AgentRepository
	botService  TelegramMessageSender
	logger      logger.Interface
}

// NewCheckExpiringUseCase creates a new CheckExpiringUseCase
func NewCheckExpiringUseCase(
	bindingRepo admin.AdminTelegramBindingRepository,
	nodeRepo node.NodeRepository,
	agentRepo forward.AgentRepository,
	botService TelegramMessageSender,
	logger logger.Interface,
) *CheckExpiringUseCase {
	return &CheckExpiringUseCase{
		bindingRepo: bindingRepo,
		nodeRepo:    nodeRepo,
		agentRepo:   agentRepo,
		botService:  botService,
		logger:      logger,
	}
}

// CheckAndNotify checks for expiring resources and sends notifications
func (uc *CheckExpiringUseCase) CheckAndNotify(ctx context.Context) error {
	if uc.botService == nil {
		uc.logger.Debugw("expiring check skipped: bot service not available")
		return nil
	}

	// Get bindings that want resource expiring notifications and haven't been notified today
	bindings, err := uc.bindingRepo.FindBindingsForResourceExpiringNotification(ctx)
	if err != nil {
		uc.logger.Errorw("failed to find bindings for resource expiring notification", "error", err)
		return err
	}

	if len(bindings) == 0 {
		uc.logger.Debugw("no bindings need resource expiring notification")
		return nil
	}

	alertsSent := 0
	errors := 0

	for i, binding := range bindings {
		// Double check if notification can be sent (domain-level deduplication)
		if !binding.CanNotifyResourceExpiring() {
			continue
		}

		// Find expiring agents within the configured days threshold
		expiringAgents, err := uc.agentRepo.FindExpiringAgents(ctx, binding.ResourceExpiringDays())
		if err != nil {
			uc.logger.Errorw("failed to find expiring agents", "error", err)
			errors++
			continue
		}

		// Find expiring nodes within the configured days threshold
		expiringNodes, err := uc.nodeRepo.FindExpiringNodes(ctx, binding.ResourceExpiringDays())
		if err != nil {
			uc.logger.Errorw("failed to find expiring nodes", "error", err)
			errors++
			continue
		}

		// Skip if no expiring resources
		if len(expiringAgents) == 0 && len(expiringNodes) == 0 {
			uc.logger.Debugw("no expiring resources found",
				"binding_id", binding.ID(),
				"threshold_days", binding.ResourceExpiringDays(),
			)
			continue
		}

		// Convert to DTO with days remaining calculation
		now := biztime.NowUTC()
		agentInfos := make([]dto.ExpiringAgentInfo, 0, len(expiringAgents))
		for _, a := range expiringAgents {
			daysRemaining := calculateDaysRemaining(now, a.ExpiresAt)
			agentInfos = append(agentInfos, dto.ExpiringAgentInfo{
				ID:            a.ID,
				SID:           a.SID,
				Name:          a.Name,
				ExpiresAt:     a.ExpiresAt,
				DaysRemaining: daysRemaining,
				CostLabel:     a.CostLabel,
			})
		}

		nodeInfos := make([]dto.ExpiringNodeInfo, 0, len(expiringNodes))
		for _, n := range expiringNodes {
			daysRemaining := calculateDaysRemaining(now, n.ExpiresAt)
			nodeInfos = append(nodeInfos, dto.ExpiringNodeInfo{
				ID:            n.ID,
				SID:           n.SID,
				Name:          n.Name,
				ExpiresAt:     n.ExpiresAt,
				DaysRemaining: daysRemaining,
				CostLabel:     n.CostLabel,
			})
		}

		// Build and send message
		lang := i18n.ParseLang(binding.Language())
		message := i18n.BuildResourceExpiringMessage(lang, agentInfos, nodeInfos)
		if message == "" {
			continue
		}

		if err := uc.botService.SendMessage(binding.TelegramUserID(), message); err != nil {
			if telegram.IsBotBlocked(err) {
				uc.logger.Warnw("bot blocked by user, skipping notification",
					"telegram_user_id", binding.TelegramUserID())
				continue
			}
			uc.logger.Errorw("failed to send resource expiring notification",
				"telegram_user_id", binding.TelegramUserID(),
				"error", err,
			)
			errors++
			continue
		}

		// Record notification to prevent duplicate notifications today
		binding.RecordResourceExpiringNotification()
		if err := uc.bindingRepo.Update(ctx, binding); err != nil {
			uc.logger.Errorw("failed to update binding after notification",
				"binding_id", binding.ID(),
				"error", err,
			)
			// Don't count as error - notification was sent successfully
		}

		alertsSent++
		// Rate limiting between messages to avoid Telegram API throttling
		if i < len(bindings)-1 {
			time.Sleep(50 * time.Millisecond)
		}

		uc.logger.Infow("resource expiring notification sent",
			"telegram_user_id", binding.TelegramUserID(),
			"expiring_agents", len(agentInfos),
			"expiring_nodes", len(nodeInfos),
		)
	}

	uc.logger.Infow("expiring check completed",
		"alerts_sent", alertsSent,
		"errors", errors,
	)

	if errors > 0 {
		return fmt.Errorf("expiring check completed with %d errors out of %d bindings", errors, len(bindings))
	}

	return nil
}

// calculateDaysRemaining calculates the number of full days remaining until expiration.
// Uses calendar day boundary calculation to avoid floating point precision issues.
// Returns 0 if already expired or expires today.
func calculateDaysRemaining(now, expiresAt time.Time) int {
	// Truncate both times to day boundary in UTC for consistent calculation
	nowDay := now.UTC().Truncate(24 * time.Hour)
	expiresDay := expiresAt.UTC().Truncate(24 * time.Hour)

	// Calculate full days between the two dates using integer division
	// This avoids floating point precision issues from Hours()/24
	days := int(expiresDay.Sub(nowDay) / (24 * time.Hour))
	if days < 0 {
		return 0
	}
	return days
}

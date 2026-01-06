package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// MuteNotificationUseCase handles muting notifications for agents and nodes
type MuteNotificationUseCase struct {
	agentRepo forward.AgentRepository
	nodeRepo  node.NodeRepository
	logger    logger.Interface
}

// NewMuteNotificationUseCase creates a new MuteNotificationUseCase
func NewMuteNotificationUseCase(
	agentRepo forward.AgentRepository,
	nodeRepo node.NodeRepository,
	logger logger.Interface,
) *MuteNotificationUseCase {
	return &MuteNotificationUseCase{
		agentRepo: agentRepo,
		nodeRepo:  nodeRepo,
		logger:    logger,
	}
}

// MuteAgentNotification mutes offline notifications for an agent by SID
func (uc *MuteNotificationUseCase) MuteAgentNotification(ctx context.Context, agentSID string) error {
	agent, err := uc.agentRepo.GetBySID(ctx, agentSID)
	if err != nil {
		uc.logger.Errorw("failed to get agent by SID for muting",
			"agent_sid", agentSID,
			"error", err,
		)
		return fmt.Errorf("failed to get agent: %w", err)
	}

	if agent == nil {
		return fmt.Errorf("agent not found: %s", agentSID)
	}

	// Set mute notification flag
	agent.SetMuteNotification(true)

	// Save the agent
	if err := uc.agentRepo.Update(ctx, agent); err != nil {
		uc.logger.Errorw("failed to update agent mute notification",
			"agent_sid", agentSID,
			"error", err,
		)
		return fmt.Errorf("failed to update agent: %w", err)
	}

	uc.logger.Infow("agent notification muted",
		"agent_sid", agentSID,
		"agent_name", agent.Name(),
	)

	return nil
}

// MuteNodeNotification mutes offline notifications for a node by SID
func (uc *MuteNotificationUseCase) MuteNodeNotification(ctx context.Context, nodeSID string) error {
	n, err := uc.nodeRepo.GetBySID(ctx, nodeSID)
	if err != nil {
		uc.logger.Errorw("failed to get node by SID for muting",
			"node_sid", nodeSID,
			"error", err,
		)
		return fmt.Errorf("failed to get node: %w", err)
	}

	if n == nil {
		return fmt.Errorf("node not found: %s", nodeSID)
	}

	// Set mute notification flag
	n.SetMuteNotification(true)

	// Save the node
	if err := uc.nodeRepo.Update(ctx, n); err != nil {
		uc.logger.Errorw("failed to update node mute notification",
			"node_sid", nodeSID,
			"error", err,
		)
		return fmt.Errorf("failed to update node: %w", err)
	}

	uc.logger.Infow("node notification muted",
		"node_sid", nodeSID,
		"node_name", n.Name(),
	)

	return nil
}

// UnmuteAgentNotification unmutes offline notifications for an agent by SID
func (uc *MuteNotificationUseCase) UnmuteAgentNotification(ctx context.Context, agentSID string) error {
	agent, err := uc.agentRepo.GetBySID(ctx, agentSID)
	if err != nil {
		uc.logger.Errorw("failed to get agent by SID for unmuting",
			"agent_sid", agentSID,
			"error", err,
		)
		return fmt.Errorf("failed to get agent: %w", err)
	}

	if agent == nil {
		return fmt.Errorf("agent not found: %s", agentSID)
	}

	// Clear mute notification flag
	agent.SetMuteNotification(false)

	// Save the agent
	if err := uc.agentRepo.Update(ctx, agent); err != nil {
		uc.logger.Errorw("failed to update agent unmute notification",
			"agent_sid", agentSID,
			"error", err,
		)
		return fmt.Errorf("failed to update agent: %w", err)
	}

	uc.logger.Infow("agent notification unmuted",
		"agent_sid", agentSID,
		"agent_name", agent.Name(),
	)

	return nil
}

// UnmuteNodeNotification unmutes offline notifications for a node by SID
func (uc *MuteNotificationUseCase) UnmuteNodeNotification(ctx context.Context, nodeSID string) error {
	n, err := uc.nodeRepo.GetBySID(ctx, nodeSID)
	if err != nil {
		uc.logger.Errorw("failed to get node by SID for unmuting",
			"node_sid", nodeSID,
			"error", err,
		)
		return fmt.Errorf("failed to get node: %w", err)
	}

	if n == nil {
		return fmt.Errorf("node not found: %s", nodeSID)
	}

	// Clear mute notification flag
	n.SetMuteNotification(false)

	// Save the node
	if err := uc.nodeRepo.Update(ctx, n); err != nil {
		uc.logger.Errorw("failed to update node unmute notification",
			"node_sid", nodeSID,
			"error", err,
		)
		return fmt.Errorf("failed to update node: %w", err)
	}

	uc.logger.Infow("node notification unmuted",
		"node_sid", nodeSID,
		"node_name", n.Name(),
	)

	return nil
}

package services

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// SyncNotifier handles notification distribution and version management for config sync.
// It encapsulates the common patterns for sending sync messages to agents
// and tracking their sync versions.
type SyncNotifier struct {
	hub           SyncHub
	agentRepo     forward.AgentRepository
	agentVersions sync.Map
	globalVersion atomic.Uint64
	logger        logger.Interface
}

// NewSyncNotifier creates a new SyncNotifier instance.
func NewSyncNotifier(
	hub SyncHub,
	agentRepo forward.AgentRepository,
	log logger.Interface,
) *SyncNotifier {
	n := &SyncNotifier{
		hub:       hub,
		agentRepo: agentRepo,
		logger:    log,
	}
	// Initialize global version to 1
	n.globalVersion.Store(1)
	return n
}

// SendToAgent sends config sync message to a specific agent.
// It constructs the HubMessage envelope and sends it through the hub.
// Returns nil if the agent is offline (graceful skip).
func (n *SyncNotifier) SendToAgent(ctx context.Context, agentID uint, syncData *dto.ConfigSyncData) error {
	// Check if agent is online
	if !n.hub.IsAgentOnline(agentID) {
		n.logger.Debugw("agent offline, skipping sync notification",
			"agent_id", agentID,
			"version", syncData.Version,
		)
		return nil
	}

	// Get agent short ID for Stripe-style prefixed ID
	agent, err := n.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		n.logger.Errorw("failed to get agent for config sync",
			"agent_id", agentID,
			"error", err,
		)
		return err
	}
	if agent == nil {
		n.logger.Warnw("agent not found for config sync",
			"agent_id", agentID,
		)
		return forward.ErrAgentNotFound
	}

	// Build and send sync message
	msg := &dto.HubMessage{
		Type:      dto.MsgTypeConfigSync,
		AgentID:   agent.SID(),
		Timestamp: biztime.NowUTC().Unix(),
		Data:      syncData,
	}

	if err := n.hub.SendMessageToAgent(agentID, msg); err != nil {
		n.logger.Errorw("failed to send config sync message",
			"agent_id", agentID,
			"version", syncData.Version,
			"error", err,
		)
		return err
	}

	// Update agent version after successful send
	n.agentVersions.Store(agentID, syncData.Version)

	return nil
}

// IncrementVersion atomically increments and returns the new global version.
func (n *SyncNotifier) IncrementVersion() uint64 {
	return n.globalVersion.Add(1)
}

// GetAgentVersion returns the last synced version for an agent.
// Returns 0 if the agent has no recorded version.
func (n *SyncNotifier) GetAgentVersion(agentID uint) uint64 {
	if version, ok := n.agentVersions.Load(agentID); ok {
		return version.(uint64)
	}
	return 0
}

// UpdateAgentVersion updates the version for a specific agent.
// This is typically called when receiving config acknowledgment from an agent.
func (n *SyncNotifier) UpdateAgentVersion(agentID uint, version uint64) {
	n.agentVersions.Store(agentID, version)
}

// IsAgentOnline checks if an agent is currently connected to the hub.
func (n *SyncNotifier) IsAgentOnline(agentID uint) bool {
	return n.hub.IsAgentOnline(agentID)
}

// GetGlobalVersion returns the current global version.
func (n *SyncNotifier) GetGlobalVersion() uint64 {
	return n.globalVersion.Load()
}

package adapters

import (
	"context"

	"github.com/orris-inc/orris/internal/infrastructure/cache"
)

// AlertStateClearerAdapter adapts AlertStateManager to the use case interfaces
// for clearing alert state when resources are deleted.
type AlertStateClearerAdapter struct {
	manager *cache.AlertStateManager
}

// NewAlertStateClearerAdapter creates a new AlertStateClearerAdapter.
func NewAlertStateClearerAdapter(manager *cache.AlertStateManager) *AlertStateClearerAdapter {
	return &AlertStateClearerAdapter{manager: manager}
}

// ClearNodeAlertState implements nodeUsecases.AlertStateClearer.
func (a *AlertStateClearerAdapter) ClearNodeAlertState(ctx context.Context, nodeID uint) error {
	return a.manager.ClearState(ctx, cache.AlertResourceTypeNode, nodeID)
}

// ClearAgentAlertState implements forwardUsecases.AgentAlertStateClearer.
func (a *AlertStateClearerAdapter) ClearAgentAlertState(ctx context.Context, agentID uint) error {
	return a.manager.ClearState(ctx, cache.AlertResourceTypeAgent, agentID)
}

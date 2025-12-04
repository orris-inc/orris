package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GetExitAgentEndpointQuery represents the input for getting exit agent endpoint.
type GetExitAgentEndpointQuery struct {
	ExitAgentID uint
}

// GetExitAgentEndpointResult represents the output of getting exit agent endpoint.
type GetExitAgentEndpointResult struct {
	Address      string `json:"address"`
	WsListenPort uint16 `json:"ws_listen_port"`
}

// GetExitAgentEndpointUseCase handles retrieving exit agent connection information.
// This is used by Entry Agent to get the connection details of the Exit Agent.
type GetExitAgentEndpointUseCase struct {
	agentRepo forward.AgentRepository
	ruleRepo  forward.Repository
	logger    logger.Interface
}

// NewGetExitAgentEndpointUseCase creates a new GetExitAgentEndpointUseCase.
func NewGetExitAgentEndpointUseCase(
	agentRepo forward.AgentRepository,
	ruleRepo forward.Repository,
	logger logger.Interface,
) *GetExitAgentEndpointUseCase {
	return &GetExitAgentEndpointUseCase{
		agentRepo: agentRepo,
		ruleRepo:  ruleRepo,
		logger:    logger,
	}
}

// Execute retrieves the exit agent's address and WebSocket listen port.
func (uc *GetExitAgentEndpointUseCase) Execute(ctx context.Context, query GetExitAgentEndpointQuery) (*GetExitAgentEndpointResult, error) {
	uc.logger.Infow("executing get exit agent endpoint use case", "exit_agent_id", query.ExitAgentID)

	if query.ExitAgentID == 0 {
		return nil, errors.NewValidationError("exit agent ID is required")
	}

	// Get the exit agent to retrieve its public address
	agent, err := uc.agentRepo.GetByID(ctx, query.ExitAgentID)
	if err != nil {
		uc.logger.Errorw("failed to get exit agent", "exit_agent_id", query.ExitAgentID, "error", err)
		return nil, fmt.Errorf("failed to get exit agent: %w", err)
	}
	if agent == nil {
		return nil, errors.NewNotFoundError("exit agent", fmt.Sprintf("%d", query.ExitAgentID))
	}

	// Check if agent is enabled
	if !agent.IsEnabled() {
		return nil, errors.NewValidationError("exit agent is disabled")
	}

	// Validate public address exists
	publicAddress := agent.PublicAddress()
	if publicAddress == "" {
		return nil, errors.NewValidationError("exit agent public address is not configured")
	}

	// Get the exit rule for this agent to retrieve ws_listen_port
	exitRule, err := uc.ruleRepo.GetExitRuleByAgentID(ctx, query.ExitAgentID)
	if err != nil {
		uc.logger.Errorw("failed to get exit rule", "exit_agent_id", query.ExitAgentID, "error", err)
		return nil, fmt.Errorf("failed to get exit rule: %w", err)
	}
	if exitRule == nil {
		return nil, errors.NewNotFoundError("exit rule", fmt.Sprintf("for agent %d", query.ExitAgentID))
	}
	wsListenPort := exitRule.WsListenPort()
	if wsListenPort == 0 {
		return nil, errors.NewValidationError("exit rule ws_listen_port is not configured")
	}

	result := &GetExitAgentEndpointResult{
		Address:      publicAddress,
		WsListenPort: wsListenPort,
	}

	uc.logger.Infow("exit agent endpoint retrieved successfully",
		"exit_agent_id", query.ExitAgentID,
		"address", publicAddress,
		"ws_listen_port", wsListenPort)

	return result, nil
}

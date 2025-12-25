package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/shared/services"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ValidateForwardAgentTokenCommand contains the data needed to validate a forward agent token.
type ValidateForwardAgentTokenCommand struct {
	PlainToken string
	IPAddress  string
}

// ValidateForwardAgentTokenResult contains the result of token validation.
type ValidateForwardAgentTokenResult struct {
	AgentID       uint   // Internal database ID for downstream handlers
	AgentStripeID string // Stripe-style ID for logging and external display (e.g., "fa_xK9mP2vL3nQ")
	AgentName     string
}

// ValidateForwardAgentTokenUseCase handles the validation of forward agent tokens.
type ValidateForwardAgentTokenUseCase struct {
	agentRepo forward.AgentRepository
	logger    logger.Interface
}

// NewValidateForwardAgentTokenUseCase creates a new instance of ValidateForwardAgentTokenUseCase.
func NewValidateForwardAgentTokenUseCase(
	agentRepo forward.AgentRepository,
	logger logger.Interface,
) *ValidateForwardAgentTokenUseCase {
	return &ValidateForwardAgentTokenUseCase{
		agentRepo: agentRepo,
		logger:    logger,
	}
}

// Execute validates the provided forward agent token.
func (uc *ValidateForwardAgentTokenUseCase) Execute(
	ctx context.Context,
	cmd ValidateForwardAgentTokenCommand,
) (*ValidateForwardAgentTokenResult, error) {
	// Hash the plain token to look up the agent
	tokenHash := uc.hashToken(cmd.PlainToken)

	// Retrieve agent by token hash
	agent, err := uc.agentRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		uc.logger.Warnw("forward agent token not found", "error", err, "ip", cmd.IPAddress)
		return nil, fmt.Errorf("invalid token")
	}

	if agent == nil {
		uc.logger.Warnw("forward agent not found for token hash", "ip", cmd.IPAddress)
		return nil, fmt.Errorf("invalid token")
	}

	agentStripeID := agent.SID()

	// Verify token using constant-time comparison
	if !agent.VerifyAPIToken(cmd.PlainToken) {
		uc.logger.Warnw("forward agent token verification failed",
			"agent_id", agentStripeID,
			"ip", cmd.IPAddress,
		)
		return nil, fmt.Errorf("token verification failed")
	}

	// Check if agent is enabled
	if !agent.IsEnabled() {
		uc.logger.Warnw("forward agent is not enabled",
			"agent_id", agentStripeID,
			"status", agent.Status(),
			"ip", cmd.IPAddress,
		)
		return nil, fmt.Errorf("agent is not enabled")
	}

	uc.logger.Debugw("forward agent token validated successfully",
		"agent_id", agentStripeID,
		"agent_name", agent.Name(),
		"ip", cmd.IPAddress,
	)

	return &ValidateForwardAgentTokenResult{
		AgentID:       agent.ID(),
		AgentStripeID: agentStripeID,
		AgentName:     agent.Name(),
	}, nil
}

// hashToken computes the SHA256 hash of the plain token.
// This method uses the same hashing mechanism as the ForwardAgent domain.
func (uc *ValidateForwardAgentTokenUseCase) hashToken(plainToken string) string {
	// Use the token generator service for consistent hashing
	tokenGen := services.NewTokenGenerator()
	return tokenGen.HashToken(plainToken)
}

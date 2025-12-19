package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// CreateForwardAgentCommand represents the input for creating a forward agent.
type CreateForwardAgentCommand struct {
	Name          string
	PublicAddress string
	TunnelAddress string
	Remark        string
}

// CreateForwardAgentResult represents the output of creating a forward agent.
type CreateForwardAgentResult struct {
	ID            string `json:"id"` // Stripe-style prefixed ID (e.g., "fa_xK9mP2vL3nQ")
	Name          string `json:"name"`
	PublicAddress string `json:"public_address"`
	TunnelAddress string `json:"tunnel_address,omitempty"`
	Token         string `json:"token"`
	Status        string `json:"status"`
	Remark        string `json:"remark"`
	CreatedAt     string `json:"created_at"`
}

// CreateForwardAgentUseCase handles forward agent creation.
type CreateForwardAgentUseCase struct {
	repo     forward.AgentRepository
	tokenGen AgentTokenGenerator
	logger   logger.Interface
}

// NewCreateForwardAgentUseCase creates a new CreateForwardAgentUseCase.
func NewCreateForwardAgentUseCase(
	repo forward.AgentRepository,
	tokenGen AgentTokenGenerator,
	logger logger.Interface,
) *CreateForwardAgentUseCase {
	return &CreateForwardAgentUseCase{
		repo:     repo,
		tokenGen: tokenGen,
		logger:   logger,
	}
}

// Execute creates a new forward agent.
func (uc *CreateForwardAgentUseCase) Execute(ctx context.Context, cmd CreateForwardAgentCommand) (*CreateForwardAgentResult, error) {
	uc.logger.Infow("executing create forward agent use case", "name", cmd.Name)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid create forward agent command", "error", err)
		return nil, err
	}

	// Check if agent name already exists
	exists, err := uc.repo.ExistsByName(ctx, cmd.Name)
	if err != nil {
		uc.logger.Errorw("failed to check existing forward agent", "name", cmd.Name, "error", err)
		return nil, fmt.Errorf("failed to check existing agent: %w", err)
	}
	if exists {
		uc.logger.Warnw("agent name already exists", "name", cmd.Name)
		return nil, errors.NewConflictError("agent name already exists", cmd.Name)
	}

	// Create domain entity with HMAC-based token generator
	agent, err := forward.NewForwardAgent(cmd.Name, cmd.PublicAddress, cmd.TunnelAddress, cmd.Remark, id.NewForwardAgentID, uc.tokenGen.Generate)
	if err != nil {
		uc.logger.Errorw("failed to create forward agent entity", "error", err)
		return nil, fmt.Errorf("failed to create forward agent: %w", err)
	}

	// Persist
	if err := uc.repo.Create(ctx, agent); err != nil {
		uc.logger.Errorw("failed to persist forward agent", "error", err)
		return nil, fmt.Errorf("failed to save forward agent: %w", err)
	}

	// Get the plain token before it's cleared
	plainToken := agent.GetAPIToken()

	result := &CreateForwardAgentResult{
		ID:            agent.SID(),
		Name:          agent.Name(),
		PublicAddress: agent.PublicAddress(),
		TunnelAddress: agent.TunnelAddress(),
		Token:         plainToken,
		Status:        string(agent.Status()),
		Remark:        agent.Remark(),
		CreatedAt:     agent.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}

	uc.logger.Infow("forward agent created successfully", "id", result.ID, "name", cmd.Name)
	return result, nil
}

func (uc *CreateForwardAgentUseCase) validateCommand(cmd CreateForwardAgentCommand) error {
	if cmd.Name == "" {
		return errors.NewValidationError("agent name is required")
	}
	return nil
}

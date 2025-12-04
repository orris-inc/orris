package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// CreateForwardAgentCommand represents the input for creating a forward agent.
type CreateForwardAgentCommand struct {
	Name          string
	PublicAddress string
	Remark        string
}

// CreateForwardAgentResult represents the output of creating a forward agent.
type CreateForwardAgentResult struct {
	ID            uint   `json:"id"`
	Name          string `json:"name"`
	PublicAddress string `json:"public_address"`
	Token         string `json:"token"`
	Status        string `json:"status"`
	Remark        string `json:"remark"`
	CreatedAt     string `json:"created_at"`
}

// CreateForwardAgentUseCase handles forward agent creation.
type CreateForwardAgentUseCase struct {
	repo   forward.AgentRepository
	logger logger.Interface
}

// NewCreateForwardAgentUseCase creates a new CreateForwardAgentUseCase.
func NewCreateForwardAgentUseCase(
	repo forward.AgentRepository,
	logger logger.Interface,
) *CreateForwardAgentUseCase {
	return &CreateForwardAgentUseCase{
		repo:   repo,
		logger: logger,
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

	// Create domain entity
	agent, err := forward.NewForwardAgent(cmd.Name, cmd.PublicAddress, cmd.Remark)
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
		ID:            agent.ID(),
		Name:          agent.Name(),
		PublicAddress: agent.PublicAddress(),
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

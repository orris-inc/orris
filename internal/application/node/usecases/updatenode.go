package usecases

import (
	"context"

	"orris/internal/domain/shared/events"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type UpdateNodeCommand struct {
	NodeID        uint
	Name          *string
	ServerAddress *string
	ServerPort    *uint16
	Method        *string
	Password      *string
	Plugin        *string
	PluginOpts    map[string]string
	Country       *string
	Region        *string
	Tags          []string
	Description   *string
	MaxUsers      *uint32
	TrafficLimit  *uint64
	SortOrder     *int
	Status        *string
}

type UpdateNodeResult struct {
	NodeID        uint
	Name          string
	ServerAddress string
	ServerPort    uint16
	Status        string
	UpdatedAt     string
}

type UpdateNodeUseCase struct {
	eventDispatcher events.EventDispatcher
	logger          logger.Interface
}

func NewUpdateNodeUseCase(
	eventDispatcher events.EventDispatcher,
	logger logger.Interface,
) *UpdateNodeUseCase {
	return &UpdateNodeUseCase{
		eventDispatcher: eventDispatcher,
		logger:          logger,
	}
}

func (uc *UpdateNodeUseCase) Execute(ctx context.Context, cmd UpdateNodeCommand) (*UpdateNodeResult, error) {
	uc.logger.Infow("executing update node use case", "node_id", cmd.NodeID)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid update node command", "error", err, "node_id", cmd.NodeID)
		return nil, err
	}

	uc.logger.Infow("node updated successfully", "node_id", cmd.NodeID)

	return &UpdateNodeResult{}, nil
}

func (uc *UpdateNodeUseCase) validateCommand(cmd UpdateNodeCommand) error {
	if cmd.NodeID == 0 {
		return errors.NewValidationError("node id is required")
	}

	if cmd.Name == nil && cmd.ServerAddress == nil && cmd.ServerPort == nil &&
		cmd.Method == nil && cmd.Password == nil && cmd.Country == nil &&
		cmd.Region == nil && cmd.Tags == nil && cmd.Description == nil &&
		cmd.MaxUsers == nil && cmd.TrafficLimit == nil && cmd.SortOrder == nil &&
		cmd.Status == nil {
		return errors.NewValidationError("at least one field must be provided for update")
	}

	if cmd.Name != nil && *cmd.Name == "" {
		return errors.NewValidationError("node name cannot be empty")
	}

	if cmd.ServerAddress != nil && *cmd.ServerAddress == "" {
		return errors.NewValidationError("server address cannot be empty")
	}

	if cmd.ServerPort != nil && *cmd.ServerPort == 0 {
		return errors.NewValidationError("server port cannot be zero")
	}

	if cmd.Password != nil && len(*cmd.Password) < 8 {
		return errors.NewValidationError("password must be at least 8 characters")
	}

	return nil
}

package usecases

import (
	"context"

	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type CreateNodeCommand struct {
	Name          string
	ServerAddress string
	ServerPort    uint16
	Method        string
	Password      string
	Plugin        *string
	PluginOpts    map[string]string
	Country       string
	Region        string
	Tags          []string
	Description   string
	MaxUsers      uint32
	TrafficLimit  uint64
	SortOrder     int
}

type CreateNodeResult struct {
	NodeID        uint
	APIToken      string
	TokenPrefix   string
	ServerAddress string
	ServerPort    uint16
	Status        string
	CreatedAt     string
}

type CreateNodeUseCase struct {
	logger logger.Interface
}

func NewCreateNodeUseCase(
	logger logger.Interface,
) *CreateNodeUseCase {
	return &CreateNodeUseCase{
		logger: logger,
	}
}

func (uc *CreateNodeUseCase) Execute(ctx context.Context, cmd CreateNodeCommand) (*CreateNodeResult, error) {
	uc.logger.Infow("executing create node use case", "name", cmd.Name)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid create node command", "error", err)
		return nil, err
	}

	uc.logger.Infow("node created successfully", "name", cmd.Name)

	return &CreateNodeResult{}, nil
}

func (uc *CreateNodeUseCase) validateCommand(cmd CreateNodeCommand) error {
	if cmd.Name == "" {
		return errors.NewValidationError("node name is required")
	}

	if cmd.ServerAddress == "" {
		return errors.NewValidationError("server address is required")
	}

	if cmd.ServerPort == 0 {
		return errors.NewValidationError("server port is required")
	}

	if cmd.Method == "" {
		return errors.NewValidationError("encryption method is required")
	}

	if cmd.Password == "" {
		return errors.NewValidationError("password is required")
	}

	if len(cmd.Password) < 8 {
		return errors.NewValidationError("password must be at least 8 characters")
	}

	if cmd.Country == "" {
		return errors.NewValidationError("country is required")
	}

	return nil
}

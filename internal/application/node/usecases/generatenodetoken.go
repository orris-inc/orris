package usecases

import (
	"context"
	"time"

	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type GenerateNodeTokenCommand struct {
	NodeID    uint
	ExpiresAt *time.Time
}

type GenerateNodeTokenResult struct {
	NodeID      uint
	Token       string
	TokenPrefix string
	ExpiresAt   *time.Time
	CreatedAt   time.Time
}

type GenerateNodeTokenUseCase struct {
	logger          logger.Interface
}

func NewGenerateNodeTokenUseCase(
	logger logger.Interface,
) *GenerateNodeTokenUseCase {
	return &GenerateNodeTokenUseCase{
		logger:          logger,
	}
}

func (uc *GenerateNodeTokenUseCase) Execute(ctx context.Context, cmd GenerateNodeTokenCommand) (*GenerateNodeTokenResult, error) {
	uc.logger.Infow("executing generate node token use case", "node_id", cmd.NodeID)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid generate node token command", "error", err, "node_id", cmd.NodeID)
		return nil, err
	}

	if cmd.ExpiresAt != nil && cmd.ExpiresAt.Before(time.Now()) {
		uc.logger.Warnw("expiration time is in the past", "node_id", cmd.NodeID, "expires_at", cmd.ExpiresAt)
		return nil, errors.NewValidationError("expiration time cannot be in the past")
	}

	uc.logger.Infow("node token generated successfully", "node_id", cmd.NodeID)

	return &GenerateNodeTokenResult{
		NodeID:    cmd.NodeID,
		CreatedAt: time.Now(),
		ExpiresAt: cmd.ExpiresAt,
	}, nil
}

func (uc *GenerateNodeTokenUseCase) validateCommand(cmd GenerateNodeTokenCommand) error {
	if cmd.NodeID == 0 {
		return errors.NewValidationError("node id is required")
	}

	return nil
}

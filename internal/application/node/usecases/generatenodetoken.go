package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type GenerateNodeTokenCommand struct {
	ShortID   string // External API identifier
	ExpiresAt *time.Time
}

type GenerateNodeTokenResult struct {
	NodeID      uint       `json:"node_id"`
	Token       string     `json:"token"`
	TokenPrefix string     `json:"token_prefix"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type GenerateNodeTokenUseCase struct {
	nodeRepo node.NodeRepository
	logger   logger.Interface
}

func NewGenerateNodeTokenUseCase(
	nodeRepo node.NodeRepository,
	logger logger.Interface,
) *GenerateNodeTokenUseCase {
	return &GenerateNodeTokenUseCase{
		nodeRepo: nodeRepo,
		logger:   logger,
	}
}

func (uc *GenerateNodeTokenUseCase) Execute(ctx context.Context, cmd GenerateNodeTokenCommand) (*GenerateNodeTokenResult, error) {
	uc.logger.Infow("executing generate node token use case", "short_id", cmd.ShortID)

	// Validate command
	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid generate node token command", "error", err, "short_id", cmd.ShortID)
		return nil, err
	}

	if cmd.ExpiresAt != nil && cmd.ExpiresAt.Before(time.Now()) {
		uc.logger.Warnw("expiration time is in the past", "short_id", cmd.ShortID, "expires_at", cmd.ExpiresAt)
		return nil, errors.NewValidationError("expiration time cannot be in the past")
	}

	// Retrieve the node from repository
	n, err := uc.nodeRepo.GetByShortID(ctx, cmd.ShortID)
	if err != nil {
		uc.logger.Errorw("failed to get node by short ID", "short_id", cmd.ShortID, "error", err)
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	// Generate new API token using domain method
	plainToken, err := n.GenerateAPIToken()
	if err != nil {
		uc.logger.Errorw("failed to generate API token", "error", err, "short_id", cmd.ShortID)
		return nil, fmt.Errorf("failed to generate API token: %w", err)
	}

	// Extract token prefix (first 8 characters)
	tokenPrefix := plainToken
	if len(plainToken) > 8 {
		tokenPrefix = plainToken[:8]
	}

	// Update node in repository
	if err := uc.nodeRepo.Update(ctx, n); err != nil {
		uc.logger.Errorw("failed to update node", "error", err, "short_id", cmd.ShortID)
		return nil, fmt.Errorf("failed to update node: %w", err)
	}

	uc.logger.Infow("node token generated successfully", "short_id", cmd.ShortID)

	return &GenerateNodeTokenResult{
		NodeID:      n.ID(),
		Token:       plainToken,
		TokenPrefix: tokenPrefix,
		ExpiresAt:   cmd.ExpiresAt,
		CreatedAt:   time.Now(),
	}, nil
}

func (uc *GenerateNodeTokenUseCase) validateCommand(cmd GenerateNodeTokenCommand) error {
	if cmd.ShortID == "" {
		return errors.NewValidationError("short ID must be provided")
	}

	return nil
}

package usecases

import (
	"context"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type RegenerateUserNodeTokenCommand struct {
	UserID  uint
	NodeSID string
}

type RegenerateUserNodeTokenResult struct {
	NodeSID  string
	APIToken string
}

type RegenerateUserNodeTokenExecutor interface {
	Execute(ctx context.Context, cmd RegenerateUserNodeTokenCommand) (*RegenerateUserNodeTokenResult, error)
}

type RegenerateUserNodeTokenUseCase struct {
	nodeRepo node.NodeRepository
	logger   logger.Interface
}

func NewRegenerateUserNodeTokenUseCase(
	nodeRepo node.NodeRepository,
	logger logger.Interface,
) *RegenerateUserNodeTokenUseCase {
	return &RegenerateUserNodeTokenUseCase{
		nodeRepo: nodeRepo,
		logger:   logger,
	}
}

func (uc *RegenerateUserNodeTokenUseCase) Execute(ctx context.Context, cmd RegenerateUserNodeTokenCommand) (*RegenerateUserNodeTokenResult, error) {
	uc.logger.Infow("executing regenerate user node token use case", "user_id", cmd.UserID, "node_sid", cmd.NodeSID)

	nodeEntity, err := uc.nodeRepo.GetBySID(ctx, cmd.NodeSID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if !nodeEntity.IsOwnedBy(cmd.UserID) {
		return nil, errors.NewForbiddenError("access denied to this node")
	}

	// Generate new token
	newToken, err := nodeEntity.GenerateAPIToken()
	if err != nil {
		uc.logger.Errorw("failed to generate new token", "node_sid", cmd.NodeSID, "error", err)
		return nil, err
	}

	// Persist changes
	if err := uc.nodeRepo.Update(ctx, nodeEntity); err != nil {
		uc.logger.Errorw("failed to update node token", "node_sid", cmd.NodeSID, "error", err)
		return nil, err
	}

	uc.logger.Infow("user node token regenerated successfully", "user_id", cmd.UserID, "node_sid", cmd.NodeSID)
	return &RegenerateUserNodeTokenResult{
		NodeSID:  nodeEntity.SID(),
		APIToken: newToken,
	}, nil
}

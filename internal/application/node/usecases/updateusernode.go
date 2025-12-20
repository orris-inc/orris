package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/node/dto"
	"github.com/orris-inc/orris/internal/domain/node"
	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type UpdateUserNodeCommand struct {
	UserID           uint
	NodeSID          string
	Name             *string
	ServerAddress    *string
	AgentPort        *uint16
	SubscriptionPort *uint16
}

type UpdateUserNodeExecutor interface {
	Execute(ctx context.Context, cmd UpdateUserNodeCommand) (*dto.UserNodeDTO, error)
}

type UpdateUserNodeUseCase struct {
	nodeRepo node.NodeRepository
	logger   logger.Interface
}

func NewUpdateUserNodeUseCase(
	nodeRepo node.NodeRepository,
	logger logger.Interface,
) *UpdateUserNodeUseCase {
	return &UpdateUserNodeUseCase{
		nodeRepo: nodeRepo,
		logger:   logger,
	}
}

func (uc *UpdateUserNodeUseCase) Execute(ctx context.Context, cmd UpdateUserNodeCommand) (*dto.UserNodeDTO, error) {
	uc.logger.Infow("executing update user node use case", "user_id", cmd.UserID, "node_sid", cmd.NodeSID)

	nodeEntity, err := uc.nodeRepo.GetBySID(ctx, cmd.NodeSID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if !nodeEntity.IsOwnedBy(cmd.UserID) {
		return nil, errors.NewForbiddenError("access denied to this node")
	}

	// Update name if provided
	if cmd.Name != nil && *cmd.Name != "" {
		// Check uniqueness
		exists, err := uc.nodeRepo.ExistsByNameForUserExcluding(ctx, *cmd.Name, cmd.UserID, nodeEntity.ID())
		if err != nil {
			return nil, fmt.Errorf("failed to check name uniqueness: %w", err)
		}
		if exists {
			return nil, errors.NewConflictError("node with this name already exists")
		}
		if err := nodeEntity.UpdateName(*cmd.Name); err != nil {
			return nil, err
		}
	}

	// Update server address if provided
	if cmd.ServerAddress != nil && *cmd.ServerAddress != "" {
		serverAddress, err := vo.NewServerAddress(*cmd.ServerAddress)
		if err != nil {
			return nil, fmt.Errorf("invalid server address: %w", err)
		}
		if err := nodeEntity.UpdateServerAddress(serverAddress); err != nil {
			return nil, err
		}
	}

	// Update agent port if provided
	if cmd.AgentPort != nil {
		// Check address:port uniqueness
		addr := nodeEntity.ServerAddress().Value()
		if cmd.ServerAddress != nil {
			addr = *cmd.ServerAddress
		}
		exists, err := uc.nodeRepo.ExistsByAddressForUserExcluding(ctx, addr, int(*cmd.AgentPort), cmd.UserID, nodeEntity.ID())
		if err != nil {
			return nil, fmt.Errorf("failed to check address uniqueness: %w", err)
		}
		if exists {
			return nil, errors.NewConflictError("node with this address and port already exists")
		}
		if err := nodeEntity.UpdateAgentPort(*cmd.AgentPort); err != nil {
			return nil, err
		}
	}

	// Update subscription port if provided
	if cmd.SubscriptionPort != nil {
		if err := nodeEntity.UpdateSubscriptionPort(cmd.SubscriptionPort); err != nil {
			return nil, err
		}
	}

	// Persist changes
	if err := uc.nodeRepo.Update(ctx, nodeEntity); err != nil {
		uc.logger.Errorw("failed to update user node", "node_sid", cmd.NodeSID, "error", err)
		return nil, err
	}

	uc.logger.Infow("user node updated successfully", "user_id", cmd.UserID, "node_sid", cmd.NodeSID)
	return dto.ToUserNodeDTO(nodeEntity), nil
}

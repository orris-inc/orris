package usecases

import (
	"context"
	"fmt"

	"orris/internal/application/node/dto"
	domainNode "orris/internal/domain/node"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

// GetNodeQuery represents the query for getting a node
type GetNodeQuery struct {
	NodeID uint
}

// GetNodeResult represents the result of getting a node
type GetNodeResult struct {
	Node *dto.NodeDTO
}

// GetNodeUseCase handles the business logic for retrieving a node
type GetNodeUseCase struct {
	nodeRepo domainNode.NodeRepository
	logger   logger.Interface
}

// NewGetNodeUseCase creates a new get node use case
func NewGetNodeUseCase(
	nodeRepo domainNode.NodeRepository,
	logger logger.Interface,
) *GetNodeUseCase {
	return &GetNodeUseCase{
		nodeRepo: nodeRepo,
		logger:   logger,
	}
}

// Execute retrieves a node by ID
func (uc *GetNodeUseCase) Execute(ctx context.Context, query GetNodeQuery) (*GetNodeResult, error) {
	uc.logger.Infow("executing get node use case", "node_id", query.NodeID)

	// Validate query
	if err := uc.validateQuery(query); err != nil {
		uc.logger.Errorw("invalid get node query", "error", err)
		return nil, err
	}

	// Retrieve the node
	nodeEntity, err := uc.nodeRepo.GetByID(ctx, query.NodeID)
	if err != nil {
		uc.logger.Errorw("failed to get node", "node_id", query.NodeID, "error", err)
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	if nodeEntity == nil {
		uc.logger.Warnw("node not found", "node_id", query.NodeID)
		return nil, errors.NewNotFoundError("node not found")
	}

	// Map to DTO
	nodeDTO := dto.ToNodeDTO(nodeEntity)

	uc.logger.Infow("node retrieved successfully", "node_id", query.NodeID, "name", nodeEntity.Name())

	return &GetNodeResult{
		Node: nodeDTO,
	}, nil
}

// validateQuery validates the get node query
func (uc *GetNodeUseCase) validateQuery(query GetNodeQuery) error {
	if query.NodeID == 0 {
		return errors.NewValidationError("node ID cannot be zero")
	}

	return nil
}

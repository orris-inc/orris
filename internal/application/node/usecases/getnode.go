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

// NodeSystemStatusQuerier defines the interface for querying node system status
type NodeSystemStatusQuerier interface {
	GetNodeSystemStatus(ctx context.Context, nodeID uint) (*NodeSystemStatus, error)
}

// NodeSystemStatus represents node system status metrics
type NodeSystemStatus struct {
	CPU       string
	Memory    string
	Disk      string
	Uptime    int
	UpdatedAt int64
}

// GetNodeUseCase handles the business logic for retrieving a node
type GetNodeUseCase struct {
	nodeRepo      domainNode.NodeRepository
	statusQuerier NodeSystemStatusQuerier
	logger        logger.Interface
}

// NewGetNodeUseCase creates a new get node use case
func NewGetNodeUseCase(
	nodeRepo domainNode.NodeRepository,
	statusQuerier NodeSystemStatusQuerier,
	logger logger.Interface,
) *GetNodeUseCase {
	return &GetNodeUseCase{
		nodeRepo:      nodeRepo,
		statusQuerier: statusQuerier,
		logger:        logger,
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

	// Query system status from Redis
	systemStatus, err := uc.statusQuerier.GetNodeSystemStatus(ctx, query.NodeID)
	if err != nil {
		uc.logger.Warnw("failed to get node system status, continuing without it",
			"node_id", query.NodeID,
			"error", err,
		)
	} else if systemStatus != nil {
		// Add system status to DTO
		nodeDTO.SystemStatus = &dto.NodeSystemStatusDTO{
			CPU:       systemStatus.CPU,
			Memory:    systemStatus.Memory,
			Disk:      systemStatus.Disk,
			Uptime:    systemStatus.Uptime,
			UpdatedAt: systemStatus.UpdatedAt,
		}
	}

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

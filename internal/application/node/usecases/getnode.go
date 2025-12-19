package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/node/dto"
	domainNode "github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GetNodeQuery represents the query for getting a node
type GetNodeQuery struct {
	SID string // External API identifier
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
	CPU        string
	Memory     string
	Disk       string
	Uptime     int
	UpdatedAt  int64
	PublicIPv4 string
	PublicIPv6 string
}

// GetNodeUseCase handles the business logic for retrieving a node
type GetNodeUseCase struct {
	nodeRepo          domainNode.NodeRepository
	resourceGroupRepo resource.Repository
	statusQuerier     NodeSystemStatusQuerier
	logger            logger.Interface
}

// NewGetNodeUseCase creates a new get node use case
func NewGetNodeUseCase(
	nodeRepo domainNode.NodeRepository,
	resourceGroupRepo resource.Repository,
	statusQuerier NodeSystemStatusQuerier,
	logger logger.Interface,
) *GetNodeUseCase {
	return &GetNodeUseCase{
		nodeRepo:          nodeRepo,
		resourceGroupRepo: resourceGroupRepo,
		statusQuerier:     statusQuerier,
		logger:            logger,
	}
}

// Execute retrieves a node by SID
func (uc *GetNodeUseCase) Execute(ctx context.Context, query GetNodeQuery) (*GetNodeResult, error) {
	uc.logger.Infow("executing get node use case", "sid", query.SID)

	// Validate query
	if err := uc.validateQuery(query); err != nil {
		uc.logger.Errorw("invalid get node query", "error", err)
		return nil, err
	}

	// Retrieve the node
	nodeEntity, err := uc.nodeRepo.GetBySID(ctx, query.SID)
	if err != nil {
		uc.logger.Errorw("failed to get node by SID", "sid", query.SID, "error", err)
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	if nodeEntity == nil {
		uc.logger.Warnw("node not found", "sid", query.SID)
		return nil, errors.NewNotFoundError("node not found")
	}

	// Map to DTO
	nodeDTO := dto.ToNodeDTO(nodeEntity)

	// Resolve GroupIDs to GroupSIDs
	if len(nodeEntity.GroupIDs()) > 0 {
		groupSIDs := make([]string, 0, len(nodeEntity.GroupIDs()))
		for _, groupID := range nodeEntity.GroupIDs() {
			group, err := uc.resourceGroupRepo.GetByID(ctx, groupID)
			if err != nil {
				uc.logger.Warnw("failed to get resource group, skipping",
					"group_id", groupID,
					"error", err,
				)
				continue
			}
			if group != nil {
				groupSIDs = append(groupSIDs, group.SID())
			}
		}
		if len(groupSIDs) > 0 {
			nodeDTO.GroupSIDs = groupSIDs
		}
	}

	// Query system status from Redis using internal ID
	systemStatus, err := uc.statusQuerier.GetNodeSystemStatus(ctx, nodeEntity.ID())
	if err != nil {
		uc.logger.Warnw("failed to get node system status, continuing without it",
			"node_id", nodeEntity.ID(),
			"error", err,
		)
	} else if systemStatus != nil {
		// Add system status to DTO
		nodeDTO.SystemStatus = &dto.NodeSystemStatusDTO{
			CPU:        systemStatus.CPU,
			Memory:     systemStatus.Memory,
			Disk:       systemStatus.Disk,
			Uptime:     systemStatus.Uptime,
			UpdatedAt:  systemStatus.UpdatedAt,
			PublicIPv4: systemStatus.PublicIPv4,
			PublicIPv6: systemStatus.PublicIPv6,
		}
	}

	uc.logger.Infow("node retrieved successfully", "node_id", nodeEntity.ID(), "sid", nodeEntity.SID(), "name", nodeEntity.Name())

	return &GetNodeResult{
		Node: nodeDTO,
	}, nil
}

// validateQuery validates the get node query
func (uc *GetNodeUseCase) validateQuery(query GetNodeQuery) error {
	if query.SID == "" {
		return errors.NewValidationError("SID must be provided")
	}

	return nil
}

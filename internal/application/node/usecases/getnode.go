package usecases

import (
	"context"
	"fmt"
	"strings"

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

// NodeSystemStatus represents node system status metrics from Redis cache
type NodeSystemStatus struct {
	// System resources
	CPUPercent    float64
	MemoryPercent float64
	MemoryUsed    uint64
	MemoryTotal   uint64
	MemoryAvail   uint64
	DiskPercent   float64
	DiskUsed      uint64
	DiskTotal     uint64
	UptimeSeconds int64

	// System load
	LoadAvg1  float64
	LoadAvg5  float64
	LoadAvg15 float64

	// Network statistics
	NetworkRxBytes uint64
	NetworkTxBytes uint64
	NetworkRxRate  uint64
	NetworkTxRate  uint64

	// Connection statistics
	TCPConnections int
	UDPConnections int

	// Network info
	PublicIPv4 string
	PublicIPv6 string

	// Agent info
	AgentVersion string

	// Metadata
	UpdatedAt int64
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
	// Validate query
	if err := uc.validateQuery(query); err != nil {
		return nil, err
	}

	// Retrieve the node
	nodeEntity, err := uc.nodeRepo.GetBySID(ctx, query.SID)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	if nodeEntity == nil {
		return nil, errors.NewNotFoundError("node not found")
	}

	// Map to DTO
	nodeDTO := dto.ToNodeDTO(nodeEntity)

	// Resolve GroupIDs to GroupSIDs using batch query
	if len(nodeEntity.GroupIDs()) > 0 {
		groups, err := uc.resourceGroupRepo.GetByIDs(ctx, nodeEntity.GroupIDs())
		if err != nil {
			uc.logger.Warnw("failed to batch get resource groups, skipping",
				"group_ids", nodeEntity.GroupIDs(),
				"error", err,
			)
		} else {
			groupSIDs := make([]string, 0, len(groups))
			for _, group := range groups {
				groupSIDs = append(groupSIDs, group.SID())
			}
			if len(groupSIDs) > 0 {
				nodeDTO.GroupSIDs = groupSIDs
			}
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
		nodeDTO.SystemStatus = toNodeSystemStatusDTO(systemStatus)
		// Extract agent version to top-level field for easy display
		// Normalize version format by removing "v" prefix for consistency
		nodeDTO.AgentVersion = strings.TrimPrefix(systemStatus.AgentVersion, "v")
	}

	uc.logger.Debugw("node retrieved", "sid", nodeEntity.SID())

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

// toNodeSystemStatusDTO converts internal NodeSystemStatus to DTO
func toNodeSystemStatusDTO(status *NodeSystemStatus) *dto.NodeSystemStatusDTO {
	if status == nil {
		return nil
	}
	return &dto.NodeSystemStatusDTO{
		CPUPercent:     status.CPUPercent,
		MemoryPercent:  status.MemoryPercent,
		MemoryUsed:     status.MemoryUsed,
		MemoryTotal:    status.MemoryTotal,
		MemoryAvail:    status.MemoryAvail,
		DiskPercent:    status.DiskPercent,
		DiskUsed:       status.DiskUsed,
		DiskTotal:      status.DiskTotal,
		UptimeSeconds:  status.UptimeSeconds,
		LoadAvg1:       status.LoadAvg1,
		LoadAvg5:       status.LoadAvg5,
		LoadAvg15:      status.LoadAvg15,
		NetworkRxBytes: status.NetworkRxBytes,
		NetworkTxBytes: status.NetworkTxBytes,
		NetworkRxRate:  status.NetworkRxRate,
		NetworkTxRate:  status.NetworkTxRate,
		TCPConnections: status.TCPConnections,
		UDPConnections: status.UDPConnections,
		PublicIPv4:     status.PublicIPv4,
		PublicIPv6:     status.PublicIPv6,
		AgentVersion:   status.AgentVersion,
		UpdatedAt:      status.UpdatedAt,
	}
}

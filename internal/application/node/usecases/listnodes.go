package usecases

import (
	"context"

	"github.com/orris-inc/orris/internal/application/node/dto"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/logger"
	sharedquery "github.com/orris-inc/orris/internal/shared/query"
)

type ListNodesQuery struct {
	Status    *string
	GroupID   *uint
	Search    *string
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

// NodeListItem is deprecated - use dto.NodeDTO instead
// type NodeListItem struct {
// 	ID            uint
// 	Name          string
// 	ServerAddress string
// 	ServerPort    uint16
// 	Region        string
// 	Status        string
// 	SortOrder     int
// 	CreatedAt     string
// 	UpdatedAt     string
// }

type ListNodesResult struct {
	Nodes      []*dto.NodeDTO
	TotalCount int
	Limit      int
	Offset     int
}

// MultipleNodeSystemStatusQuerier defines the interface for querying multiple nodes' system status
type MultipleNodeSystemStatusQuerier interface {
	GetMultipleNodeSystemStatus(ctx context.Context, nodeIDs []uint) (map[uint]*NodeSystemStatus, error)
}

type ListNodesUseCase struct {
	nodeRepo      node.NodeRepository
	statusQuerier MultipleNodeSystemStatusQuerier
	logger        logger.Interface
}

func NewListNodesUseCase(
	nodeRepo node.NodeRepository,
	statusQuerier MultipleNodeSystemStatusQuerier,
	logger logger.Interface,
) *ListNodesUseCase {
	return &ListNodesUseCase{
		nodeRepo:      nodeRepo,
		statusQuerier: statusQuerier,
		logger:        logger,
	}
}

func (uc *ListNodesUseCase) Execute(ctx context.Context, query ListNodesQuery) (*ListNodesResult, error) {
	uc.logger.Infow("executing list nodes use case",
		"limit", query.Limit,
		"offset", query.Offset,
		"status", query.Status,
	)

	// Validate and normalize pagination parameters
	if query.Limit <= 0 {
		query.Limit = 20
	}

	if query.Limit > 100 {
		query.Limit = 100
	}

	if query.Offset < 0 {
		query.Offset = 0
	}

	// Validate and normalize sort parameters
	if query.SortBy == "" {
		query.SortBy = "sort_order"
	}

	if query.SortOrder == "" {
		query.SortOrder = "asc"
	}

	// Calculate page from offset and limit
	page := 1
	if query.Limit > 0 && query.Offset > 0 {
		page = (query.Offset / query.Limit) + 1
	}

	// Build domain filter from query parameters
	filter := node.NodeFilter{
		BaseFilter: sharedquery.NewBaseFilter(
			sharedquery.WithPage(page, query.Limit),
			sharedquery.WithSort(query.SortBy, query.SortOrder),
		),
		Name:   query.Search,
		Status: query.Status,
	}

	// Query nodes from repository
	nodes, totalCount, err := uc.nodeRepo.List(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to list nodes from repository", "error", err)
		return nil, err
	}

	// Convert domain entities to DTOs
	nodeDTOs := dto.ToNodeDTOList(nodes)

	// Collect node IDs for batch status query
	nodeIDs := make([]uint, 0, len(nodes))
	for _, n := range nodes {
		nodeIDs = append(nodeIDs, n.ID())
	}

	// Query system status for all nodes from Redis
	if len(nodeIDs) > 0 && uc.statusQuerier != nil {
		statusMap, err := uc.statusQuerier.GetMultipleNodeSystemStatus(ctx, nodeIDs)
		if err != nil {
			uc.logger.Warnw("failed to get nodes system status, continuing without it",
				"error", err,
			)
		} else {
			// Attach system status to each node DTO
			for _, nodeDTO := range nodeDTOs {
				if status, ok := statusMap[nodeDTO.ID]; ok && status != nil {
					nodeDTO.SystemStatus = &dto.NodeSystemStatusDTO{
						CPU:       status.CPU,
						Memory:    status.Memory,
						Disk:      status.Disk,
						Uptime:    status.Uptime,
						UpdatedAt: status.UpdatedAt,
					}
				}
			}
		}
	}

	uc.logger.Infow("nodes listed successfully",
		"count", len(nodeDTOs),
		"total", totalCount,
	)

	return &ListNodesResult{
		Nodes:      nodeDTOs,
		TotalCount: int(totalCount),
		Limit:      query.Limit,
		Offset:     query.Offset,
	}, nil
}

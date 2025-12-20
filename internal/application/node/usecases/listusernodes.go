package usecases

import (
	"context"

	"github.com/orris-inc/orris/internal/application/node/dto"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/query"
)

type ListUserNodesQuery struct {
	UserID    uint
	Status    *string
	Search    *string
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

type ListUserNodesResult struct {
	Nodes      []*dto.UserNodeDTO
	TotalCount int
	Limit      int
	Offset     int
}

type ListUserNodesExecutor interface {
	Execute(ctx context.Context, q ListUserNodesQuery) (*ListUserNodesResult, error)
}

type ListUserNodesUseCase struct {
	nodeRepo node.NodeRepository
	logger   logger.Interface
}

func NewListUserNodesUseCase(
	nodeRepo node.NodeRepository,
	logger logger.Interface,
) *ListUserNodesUseCase {
	return &ListUserNodesUseCase{
		nodeRepo: nodeRepo,
		logger:   logger,
	}
}

func (uc *ListUserNodesUseCase) Execute(ctx context.Context, q ListUserNodesQuery) (*ListUserNodesResult, error) {
	uc.logger.Debugw("executing list user nodes use case", "user_id", q.UserID)

	// Normalize pagination
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if q.Limit > 100 {
		q.Limit = 100
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
	if q.SortBy == "" {
		q.SortBy = "created_at"
	}
	if q.SortOrder == "" {
		q.SortOrder = "desc"
	}

	// Calculate page from offset
	page := 1
	if q.Limit > 0 && q.Offset > 0 {
		page = (q.Offset / q.Limit) + 1
	}

	// Build filter
	filter := node.NodeFilter{
		BaseFilter: query.NewBaseFilter(
			query.WithPage(page, q.Limit),
			query.WithSort(q.SortBy, q.SortOrder),
		),
		Name:   q.Search,
		Status: q.Status,
	}

	// Query user nodes
	nodes, totalCount, err := uc.nodeRepo.ListByUserID(ctx, q.UserID, filter)
	if err != nil {
		uc.logger.Errorw("failed to list user nodes", "user_id", q.UserID, "error", err)
		return nil, err
	}

	// Convert to DTOs
	nodeDTOs := dto.ToUserNodeDTOList(nodes)

	return &ListUserNodesResult{
		Nodes:      nodeDTOs,
		TotalCount: int(totalCount),
		Limit:      q.Limit,
		Offset:     q.Offset,
	}, nil
}

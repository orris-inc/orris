package usecases

import (
	"context"

	"orris/internal/shared/logger"
)

type ListNodesQuery struct {
	Status      *string
	Country     *string
	GroupID     *uint
	Search      *string
	Limit       int
	Offset      int
	SortBy      string
	SortOrder   string
}

type NodeListItem struct {
	ID            uint
	Name          string
	ServerAddress string
	ServerPort    uint16
	Country       string
	Region        string
	Status        string
	MaxUsers      uint32
	TrafficLimit  uint64
	TrafficUsed   uint64
	SortOrder     int
	CreatedAt     string
	UpdatedAt     string
}

type ListNodesResult struct {
	Nodes      []NodeListItem
	TotalCount int
	Limit      int
	Offset     int
}

type ListNodesUseCase struct {
	logger logger.Interface
}

func NewListNodesUseCase(
	logger logger.Interface,
) *ListNodesUseCase {
	return &ListNodesUseCase{
		logger: logger,
	}
}

func (uc *ListNodesUseCase) Execute(ctx context.Context, query ListNodesQuery) (*ListNodesResult, error) {
	uc.logger.Infow("executing list nodes use case",
		"limit", query.Limit,
		"offset", query.Offset,
		"status", query.Status,
		"country", query.Country,
	)

	if query.Limit <= 0 {
		query.Limit = 20
	}

	if query.Limit > 100 {
		query.Limit = 100
	}

	if query.Offset < 0 {
		query.Offset = 0
	}

	if query.SortBy == "" {
		query.SortBy = "sort_order"
	}

	if query.SortOrder == "" {
		query.SortOrder = "asc"
	}

	uc.logger.Infow("nodes listed successfully", "count", 0)

	return &ListNodesResult{
		Nodes:      []NodeListItem{},
		TotalCount: 0,
		Limit:      query.Limit,
		Offset:     query.Offset,
	}, nil
}

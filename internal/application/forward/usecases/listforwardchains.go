package usecases

import (
	"context"

	"orris/internal/application/forward/dto"
	"orris/internal/domain/forward"
	"orris/internal/shared/logger"
)

// ListForwardChainsQuery represents the input for listing forward chains.
type ListForwardChainsQuery struct {
	Page     int
	PageSize int
	Name     string
	Status   string
	OrderBy  string
	Order    string
}

// ListForwardChainsResult represents the output of listing forward chains.
type ListForwardChainsResult struct {
	Chains []*dto.ForwardChainDTO `json:"chains"`
	Total  int64                  `json:"total"`
}

// ListForwardChainsUseCase handles listing forward chains.
type ListForwardChainsUseCase struct {
	repo   forward.ChainRepository
	logger logger.Interface
}

// NewListForwardChainsUseCase creates a new ListForwardChainsUseCase.
func NewListForwardChainsUseCase(
	repo forward.ChainRepository,
	logger logger.Interface,
) *ListForwardChainsUseCase {
	return &ListForwardChainsUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute lists forward chains with filtering and pagination.
func (uc *ListForwardChainsUseCase) Execute(ctx context.Context, query ListForwardChainsQuery) (*ListForwardChainsResult, error) {
	uc.logger.Infow("executing list forward chains use case", "page", query.Page, "page_size", query.PageSize)

	// Set defaults
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}

	filter := forward.ChainListFilter{
		Page:     query.Page,
		PageSize: query.PageSize,
		Name:     query.Name,
		Status:   query.Status,
		OrderBy:  query.OrderBy,
		Order:    query.Order,
	}

	chains, total, err := uc.repo.List(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to list forward chains", "error", err)
		return nil, err
	}

	return &ListForwardChainsResult{
		Chains: dto.ToForwardChainDTOs(chains),
		Total:  total,
	}, nil
}

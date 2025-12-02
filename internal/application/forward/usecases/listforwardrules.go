package usecases

import (
	"context"
	"fmt"

	"orris/internal/application/forward/dto"
	"orris/internal/domain/forward"
	"orris/internal/shared/logger"
)

// ListForwardRulesQuery represents the input for listing forward rules.
type ListForwardRulesQuery struct {
	Page     int
	PageSize int
	Name     string
	Protocol string
	Status   string
	OrderBy  string
	Order    string
}

// ListForwardRulesResult represents the output of listing forward rules.
type ListForwardRulesResult struct {
	Rules []*dto.ForwardRuleDTO
	Total int64
	Page  int
	Pages int
}

// ListForwardRulesUseCase handles listing forward rules.
type ListForwardRulesUseCase struct {
	repo   forward.Repository
	logger logger.Interface
}

// NewListForwardRulesUseCase creates a new ListForwardRulesUseCase.
func NewListForwardRulesUseCase(
	repo forward.Repository,
	logger logger.Interface,
) *ListForwardRulesUseCase {
	return &ListForwardRulesUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute retrieves a list of forward rules.
func (uc *ListForwardRulesUseCase) Execute(ctx context.Context, query ListForwardRulesQuery) (*ListForwardRulesResult, error) {
	uc.logger.Infow("executing list forward rules use case", "page", query.Page, "page_size", query.PageSize)

	// Set defaults
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}

	filter := forward.ListFilter{
		Page:     query.Page,
		PageSize: query.PageSize,
		Name:     query.Name,
		Protocol: query.Protocol,
		Status:   query.Status,
		OrderBy:  query.OrderBy,
		Order:    query.Order,
	}

	rules, total, err := uc.repo.List(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to list forward rules", "error", err)
		return nil, fmt.Errorf("failed to list forward rules: %w", err)
	}

	// Calculate total pages
	pages := int(total) / query.PageSize
	if int(total)%query.PageSize > 0 {
		pages++
	}

	dtos := make([]*dto.ForwardRuleDTO, len(rules))
	for i, rule := range rules {
		dtos[i] = dto.ToForwardRuleDTO(rule)
	}

	return &ListForwardRulesResult{
		Rules: dtos,
		Total: total,
		Page:  query.Page,
		Pages: pages,
	}, nil
}

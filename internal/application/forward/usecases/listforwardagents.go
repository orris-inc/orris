package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ForwardAgentDTO represents the data transfer object for forward agents.
type ForwardAgentDTO struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Remark    string `json:"remark"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ListForwardAgentsQuery represents the input for listing forward agents.
type ListForwardAgentsQuery struct {
	Page     int
	PageSize int
	Name     string
	Status   string
	OrderBy  string
	Order    string
}

// ListForwardAgentsResult represents the output of listing forward agents.
type ListForwardAgentsResult struct {
	Agents []*ForwardAgentDTO
	Total  int64
	Page   int
	Pages  int
}

// ListForwardAgentsUseCase handles listing forward agents.
type ListForwardAgentsUseCase struct {
	repo   forward.AgentRepository
	logger logger.Interface
}

// NewListForwardAgentsUseCase creates a new ListForwardAgentsUseCase.
func NewListForwardAgentsUseCase(
	repo forward.AgentRepository,
	logger logger.Interface,
) *ListForwardAgentsUseCase {
	return &ListForwardAgentsUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute retrieves a list of forward agents.
func (uc *ListForwardAgentsUseCase) Execute(ctx context.Context, query ListForwardAgentsQuery) (*ListForwardAgentsResult, error) {
	uc.logger.Infow("executing list forward agents use case", "page", query.Page, "page_size", query.PageSize)

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

	filter := forward.AgentListFilter{
		Page:     query.Page,
		PageSize: query.PageSize,
		Name:     query.Name,
		Status:   query.Status,
		OrderBy:  query.OrderBy,
		Order:    query.Order,
	}

	agents, total, err := uc.repo.List(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to list forward agents", "error", err)
		return nil, fmt.Errorf("failed to list forward agents: %w", err)
	}

	// Calculate total pages
	pages := int(total) / query.PageSize
	if int(total)%query.PageSize > 0 {
		pages++
	}

	dtos := make([]*ForwardAgentDTO, len(agents))
	for i, agent := range agents {
		dtos[i] = &ForwardAgentDTO{
			ID:        agent.ID(),
			Name:      agent.Name(),
			Status:    string(agent.Status()),
			Remark:    agent.Remark(),
			CreatedAt: agent.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: agent.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return &ListForwardAgentsResult{
		Agents: dtos,
		Total:  total,
		Page:   query.Page,
		Pages:  pages,
	}, nil
}

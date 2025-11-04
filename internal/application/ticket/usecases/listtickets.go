package usecases

import (
	"context"

	"orris/internal/application/ticket/dto"
	"orris/internal/domain/ticket"
	vo "orris/internal/domain/ticket/value_objects"
	"orris/internal/shared/auth"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type ListTicketsQuery struct {
	UserID     uint
	UserRoles  []string
	Status     *string
	Priority   *string
	Category   *string
	AssigneeID *uint
	Page       int
	PageSize   int
	SortBy     string
	SortOrder  string
}

type ListTicketsResult struct {
	Tickets    []dto.TicketListItemDTO
	TotalCount int64
	Page       int
	PageSize   int
}

type ListTicketsExecutor interface {
	Execute(ctx context.Context, query ListTicketsQuery) (*ListTicketsResult, error)
}

type ListTicketsUseCase struct {
	ticketRepo ticket.TicketRepository
	logger     logger.Interface
}

func NewListTicketsUseCase(
	ticketRepo ticket.TicketRepository,
	logger logger.Interface,
) *ListTicketsUseCase {
	return &ListTicketsUseCase{
		ticketRepo: ticketRepo,
		logger:     logger,
	}
}

func (uc *ListTicketsUseCase) Execute(
	ctx context.Context,
	query ListTicketsQuery,
) (*ListTicketsResult, error) {
	uc.logger.Infow("executing list tickets use case",
		"user_id", query.UserID,
		"page", query.Page,
		"page_size", query.PageSize)

	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}
	if query.Page < 1 {
		query.Page = 1
	}

	filter := ticket.TicketFilter{}
	filter.Page = query.Page
	filter.PageSize = query.PageSize
	filter.SortBy = query.SortBy
	filter.SortOrder = query.SortOrder

	if query.SortBy == "" {
		filter.SortBy = "created_at"
	}
	if query.SortOrder == "" {
		filter.SortOrder = "desc"
	}

	if query.Status != nil {
		status, err := vo.NewTicketStatus(*query.Status)
		if err != nil {
			return nil, errors.NewValidationError("invalid status")
		}
		filter.Status = &status
	}

	if query.Priority != nil {
		priority, err := vo.NewPriority(*query.Priority)
		if err != nil {
			return nil, errors.NewValidationError("invalid priority")
		}
		filter.Priority = &priority
	}

	if query.Category != nil {
		category, err := vo.NewCategory(*query.Category)
		if err != nil {
			return nil, errors.NewValidationError("invalid category")
		}
		filter.Category = &category
	}

	if query.AssigneeID != nil {
		filter.AssigneeID = query.AssigneeID
	}

	isAdmin := auth.IsAdmin(query.UserRoles)
	isSupportAgent := auth.IsSupportAgent(query.UserRoles)

	if !isAdmin && !isSupportAgent {
		filter.CreatorID = &query.UserID
	}

	tickets, totalCount, err := uc.ticketRepo.List(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to list tickets", "error", err)
		return nil, errors.NewInternalError("failed to list tickets")
	}

	items := make([]dto.TicketListItemDTO, 0, len(tickets))
	for _, t := range tickets {
		if t.CanBeViewedBy(query.UserID, query.UserRoles) {
			items = append(items, dto.ToTicketListItemDTO(t))
		}
	}

	uc.logger.Infow("tickets listed successfully",
		"count", len(items),
		"total", totalCount)

	return &ListTicketsResult{
		Tickets:    items,
		TotalCount: totalCount,
		Page:       query.Page,
		PageSize:   query.PageSize,
	}, nil
}

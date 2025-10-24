package usecases

import (
	"context"

	"orris/internal/domain/ticket"
	vo "orris/internal/domain/ticket/value_objects"
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

type TicketListItem struct {
	ID           uint
	Number       string
	Title        string
	Status       string
	Priority     string
	Category     string
	CreatorID    uint
	AssigneeID   *uint
	SLADueTime   *string
	IsOverdue    bool
	CreatedAt    string
	UpdatedAt    string
}

type ListTicketsResult struct {
	Tickets    []TicketListItem
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


	filter := ticket.TicketFilter{
		Page:      query.Page,
		PageSize:    query.PageSize,
		SortBy:    query.SortBy,
		SortOrder: query.SortOrder,
	}

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

	isAdmin := uc.isAdmin(query.UserRoles)
	isSupportAgent := uc.isSupportAgent(query.UserRoles)

	if !isAdmin && !isSupportAgent {
		filter.CreatorID = &query.UserID
	}

	tickets, totalCount, err := uc.ticketRepo.List(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to list tickets", "error", err)
		return nil, errors.NewInternalError("failed to list tickets")
	}

	items := make([]TicketListItem, 0, len(tickets))
	for _, t := range tickets {
		if uc.canViewTicket(t, query.UserID, query.UserRoles) {
			item := TicketListItem{
				ID:         t.ID(),
				Number:     t.Number(),
				Title:      t.Title(),
				Status:     t.Status().String(),
				Priority:   t.Priority().String(),
				Category:   t.Category().String(),
				CreatorID:  t.CreatorID(),
				AssigneeID: t.AssigneeID(),
				IsOverdue:  t.IsOverdue(),
				CreatedAt:  t.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt:  t.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
			}

			if t.SLADueTime() != nil {
				slaTime := t.SLADueTime().Format("2006-01-02T15:04:05Z07:00")
				item.SLADueTime = &slaTime
			}

			items = append(items, item)
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

func (uc *ListTicketsUseCase) canViewTicket(t *ticket.Ticket, userID uint, userRoles []string) bool {
	return t.CanBeViewedBy(userID, userRoles)
}

func (uc *ListTicketsUseCase) isAdmin(roles []string) bool {
	for _, role := range roles {
		if role == "admin" {
			return true
		}
	}
	return false
}

func (uc *ListTicketsUseCase) isSupportAgent(roles []string) bool {
	for _, role := range roles {
		if role == "support_agent" {
			return true
		}
	}
	return false
}

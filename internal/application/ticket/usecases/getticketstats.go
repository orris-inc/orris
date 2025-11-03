package usecases

import (
	"context"
	"time"

	"orris/internal/domain/ticket"
	vo "orris/internal/domain/ticket/value_objects"
	"orris/internal/shared/logger"
)

type GetTicketStatsQuery struct {
	UserID    uint
	UserRoles []string
}

type GetTicketStatsResult struct {
	TotalTickets   int64
	OpenTickets    int64
	ClosedTickets  int64
	OverdueTickets int64
	ByStatus       map[string]int64
	ByPriority     map[string]int64
	ByCategory     map[string]int64
}

type GetTicketStatsExecutor interface {
	Execute(ctx context.Context, query GetTicketStatsQuery) (*GetTicketStatsResult, error)
}

type GetTicketStatsUseCase struct {
	ticketRepo ticket.TicketRepository
	logger     logger.Interface
}

func NewGetTicketStatsUseCase(
	ticketRepo ticket.TicketRepository,
	logger logger.Interface,
) *GetTicketStatsUseCase {
	return &GetTicketStatsUseCase{
		ticketRepo: ticketRepo,
		logger:     logger,
	}
}

func (uc *GetTicketStatsUseCase) Execute(
	ctx context.Context,
	query GetTicketStatsQuery,
) (*GetTicketStatsResult, error) {
	uc.logger.Infow("executing get ticket stats use case",
		"user_id", query.UserID)

	result := &GetTicketStatsResult{
		ByStatus:   make(map[string]int64),
		ByPriority: make(map[string]int64),
		ByCategory: make(map[string]int64),
	}

	isAdmin := uc.isAdmin(query.UserRoles)
	isSupportAgent := uc.isSupportAgent(query.UserRoles)

	filter := ticket.TicketFilter{}
	filter.Page = 1
	filter.PageSize = 10000

	if !isAdmin && !isSupportAgent {
		filter.CreatorID = &query.UserID
	}

	tickets, totalCount, err := uc.ticketRepo.List(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to get ticket stats", "error", err)
		return result, nil
	}

	result.TotalTickets = totalCount

	statuses := []vo.TicketStatus{
		vo.StatusNew,
		vo.StatusOpen,
		vo.StatusInProgress,
		vo.StatusPending,
		vo.StatusResolved,
		vo.StatusClosed,
		vo.StatusReopened,
	}

	priorities := []vo.Priority{
		vo.PriorityLow,
		vo.PriorityMedium,
		vo.PriorityHigh,
		vo.PriorityUrgent,
	}

	categories := []vo.Category{
		vo.CategoryTechnical,
		vo.CategoryAccount,
		vo.CategoryBilling,
		vo.CategoryFeature,
		vo.CategoryComplaint,
		vo.CategoryOther,
	}

	for _, status := range statuses {
		result.ByStatus[status.String()] = 0
	}

	for _, priority := range priorities {
		result.ByPriority[priority.String()] = 0
	}

	for _, category := range categories {
		result.ByCategory[category.String()] = 0
	}

	now := time.Now()

	for _, t := range tickets {
		if !uc.canViewTicket(t, query.UserID, query.UserRoles) {
			continue
		}

		result.ByStatus[t.Status().String()]++
		result.ByPriority[t.Priority().String()]++
		result.ByCategory[t.Category().String()]++

		if t.Status().IsClosed() {
			result.ClosedTickets++
		} else {
			result.OpenTickets++
		}

		if t.SLADueTime() != nil && !t.Status().IsClosed() && !t.Status().IsResolved() {
			if now.After(*t.SLADueTime()) {
				result.OverdueTickets++
			}
		}
	}

	uc.logger.Infow("ticket stats retrieved successfully",
		"total", result.TotalTickets,
		"open", result.OpenTickets,
		"closed", result.ClosedTickets,
		"overdue", result.OverdueTickets)

	return result, nil
}

func (uc *GetTicketStatsUseCase) canViewTicket(t *ticket.Ticket, userID uint, userRoles []string) bool {
	return t.CanBeViewedBy(userID, userRoles)
}

func (uc *GetTicketStatsUseCase) isAdmin(roles []string) bool {
	for _, role := range roles {
		if role == "admin" {
			return true
		}
	}
	return false
}

func (uc *GetTicketStatsUseCase) isSupportAgent(roles []string) bool {
	for _, role := range roles {
		if role == "support_agent" {
			return true
		}
	}
	return false
}

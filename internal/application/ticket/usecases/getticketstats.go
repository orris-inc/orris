package usecases

import (
	"context"

	"github.com/orris-inc/orris/internal/domain/ticket"
	vo "github.com/orris-inc/orris/internal/domain/ticket/valueobjects"
	"github.com/orris-inc/orris/internal/shared/auth"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
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

	isAdmin := auth.IsAdmin(query.UserRoles)
	isSupportAgent := auth.IsSupportAgent(query.UserRoles)

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

	now := biztime.NowUTC()

	for _, t := range tickets {
		if !t.CanBeViewedBy(query.UserID, query.UserRoles) {
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

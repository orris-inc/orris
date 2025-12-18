package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/ticket/dto"
	"github.com/orris-inc/orris/internal/domain/ticket"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type GetTicketQuery struct {
	TicketID  uint
	UserID    uint
	UserRoles []string
}

type GetTicketUseCase struct {
	ticketRepo  ticket.TicketRepository
	commentRepo ticket.CommentRepository
	logger      logger.Interface
}

func NewGetTicketUseCase(
	ticketRepo ticket.TicketRepository,
	commentRepo ticket.CommentRepository,
	logger logger.Interface,
) *GetTicketUseCase {
	return &GetTicketUseCase{
		ticketRepo:  ticketRepo,
		commentRepo: commentRepo,
		logger:      logger,
	}
}

func (uc *GetTicketUseCase) Execute(ctx context.Context, query GetTicketQuery) (*dto.TicketDTO, error) {
	uc.logger.Infow("executing get ticket use case", "ticket_id", query.TicketID, "user_id", query.UserID)

	t, err := uc.ticketRepo.GetByID(ctx, query.TicketID)
	if err != nil {
		uc.logger.Errorw("failed to load ticket", "ticket_id", query.TicketID, "error", err)
		return nil, fmt.Errorf("failed to load ticket: %w", err)
	}

	if !t.CanBeViewedBy(query.UserID, query.UserRoles) {
		uc.logger.Warnw("user cannot view ticket", "ticket_id", query.TicketID, "user_id", query.UserID)
		return nil, fmt.Errorf("permission denied: cannot view ticket")
	}

	comments, err := uc.commentRepo.GetByTicketID(ctx, query.TicketID)
	if err != nil {
		uc.logger.Errorw("failed to load comments", "ticket_id", query.TicketID, "error", err)
		return nil, fmt.Errorf("failed to load comments: %w", err)
	}

	isAgentOrAdmin := false
	for _, role := range query.UserRoles {
		if role == constants.RoleAdmin || role == constants.RoleSupportAgent {
			isAgentOrAdmin = true
			break
		}
	}

	result := dto.ToTicketDTO(t, comments, isAgentOrAdmin)

	uc.logger.Infow("ticket retrieved successfully", "ticket_id", query.TicketID)
	return result, nil
}

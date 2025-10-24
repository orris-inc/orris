package usecases

import (
	"context"

	"orris/internal/domain/ticket"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type DeleteTicketCommand struct {
	TicketID  uint
	DeletedBy uint
}

type DeleteTicketResult struct {
	TicketID uint
}

type DeleteTicketUseCase struct {
	ticketRepo ticket.TicketRepository
	logger     logger.Interface
}

func NewDeleteTicketUseCase(
	ticketRepo ticket.TicketRepository,
	logger logger.Interface,
) *DeleteTicketUseCase {
	return &DeleteTicketUseCase{
		ticketRepo: ticketRepo,
		logger:     logger,
	}
}

func (uc *DeleteTicketUseCase) Execute(ctx context.Context, cmd DeleteTicketCommand) (*DeleteTicketResult, error) {
	uc.logger.Infow("executing delete ticket use case", "ticket_id", cmd.TicketID)

	if cmd.TicketID == 0 {
		return nil, errors.NewValidationError("ticket ID is required")
	}

	if err := uc.ticketRepo.Delete(ctx, cmd.TicketID); err != nil {
		uc.logger.Errorw("failed to delete ticket", "ticket_id", cmd.TicketID, "error", err)
		return nil, errors.NewInternalError("failed to delete ticket")
	}

	uc.logger.Infow("ticket deleted successfully", "ticket_id", cmd.TicketID)

	return &DeleteTicketResult{
		TicketID: cmd.TicketID,
	}, nil
}

package usecases

import (
	"context"
	"fmt"

	"orris/internal/domain/shared/events"
	"orris/internal/domain/ticket"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type CloseTicketCommand struct {
	TicketID uint
	Reason   string
	ClosedBy uint
}

type CloseTicketResult struct {
	TicketID uint
	Status   string
	Reason   string
	ClosedAt string
}

type CloseTicketExecutor interface {
	Execute(ctx context.Context, cmd CloseTicketCommand) (*CloseTicketResult, error)
}

type CloseTicketUseCase struct {
	ticketRepo      ticket.TicketRepository
	eventDispatcher events.EventDispatcher
	logger          logger.Interface
}

func NewCloseTicketUseCase(
	ticketRepo ticket.TicketRepository,
	eventDispatcher events.EventDispatcher,
	logger logger.Interface,
) *CloseTicketUseCase {
	return &CloseTicketUseCase{
		ticketRepo:      ticketRepo,
		eventDispatcher: eventDispatcher,
		logger:          logger,
	}
}

func (uc *CloseTicketUseCase) Execute(ctx context.Context, cmd CloseTicketCommand) (*CloseTicketResult, error) {
	uc.logger.Infow("executing close ticket use case", "ticket_id", cmd.TicketID)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid close ticket command", "error", err)
		return nil, err
	}

	t, err := uc.ticketRepo.GetByID(ctx, cmd.TicketID)
	if err != nil {
		uc.logger.Errorw("failed to get ticket", "ticket_id", cmd.TicketID, "error", err)
		return nil, errors.NewNotFoundError(fmt.Sprintf("ticket %d not found", cmd.TicketID))
	}

	if err := t.Close(cmd.Reason, cmd.ClosedBy); err != nil {
		uc.logger.Errorw("failed to close ticket", "ticket_id", cmd.TicketID, "error", err)
		return nil, errors.NewValidationError(err.Error())
	}

	if err := uc.ticketRepo.Update(ctx, t); err != nil {
		uc.logger.Errorw("failed to update ticket", "ticket_id", cmd.TicketID, "error", err)
		return nil, errors.NewInternalError("failed to update ticket")
	}

	domainEvents := t.GetEvents()
	if len(domainEvents) > 0 {
		convertedEvents := make([]events.DomainEvent, 0, len(domainEvents))
		for _, evt := range domainEvents {
			if domainEvent, ok := evt.(events.DomainEvent); ok {
				convertedEvents = append(convertedEvents, domainEvent)
			}
		}
		if err := uc.eventDispatcher.PublishAll(convertedEvents); err != nil {
			uc.logger.Warnw("failed to publish events", "error", err)
		}
	}

	closedAt := ""
	if t.ClosedAt() != nil {
		closedAt = t.ClosedAt().Format("2006-01-02T15:04:05Z07:00")
	}

	uc.logger.Infow("ticket closed successfully", "ticket_id", cmd.TicketID)

	return &CloseTicketResult{
		TicketID: t.ID(),
		Status:   t.Status().String(),
		Reason:   cmd.Reason,
		ClosedAt: closedAt,
	}, nil
}

func (uc *CloseTicketUseCase) validateCommand(cmd CloseTicketCommand) error {
	if cmd.TicketID == 0 {
		return errors.NewValidationError("ticket ID is required")
	}

	if cmd.Reason == "" {
		return errors.NewValidationError("close reason is required")
	}

	if cmd.ClosedBy == 0 {
		return errors.NewValidationError("closed by user ID is required")
	}

	return nil
}

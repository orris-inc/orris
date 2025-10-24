package usecases

import (
	"context"
	"fmt"

	"orris/internal/domain/shared/events"
	"orris/internal/domain/ticket"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type ReopenTicketCommand struct {
	TicketID   uint
	Reason     string
	ReopenedBy uint
	UserRoles  []string
}

type ReopenTicketResult struct {
	TicketID   uint
	Status     string
	Reason     string
	ReopenedAt string
}

type ReopenTicketUseCase struct {
	ticketRepo      ticket.TicketRepository
	eventDispatcher events.EventDispatcher
	logger          logger.Interface
}

func NewReopenTicketUseCase(
	ticketRepo ticket.TicketRepository,
	eventDispatcher events.EventDispatcher,
	logger logger.Interface,
) *ReopenTicketUseCase {
	return &ReopenTicketUseCase{
		ticketRepo:      ticketRepo,
		eventDispatcher: eventDispatcher,
		logger:          logger,
	}
}

func (uc *ReopenTicketUseCase) Execute(ctx context.Context, cmd ReopenTicketCommand) (*ReopenTicketResult, error) {
	uc.logger.Infow("executing reopen ticket use case", "ticket_id", cmd.TicketID)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid reopen ticket command", "error", err)
		return nil, err
	}

	t, err := uc.ticketRepo.GetByID(ctx, cmd.TicketID)
	if err != nil {
		uc.logger.Errorw("failed to get ticket", "ticket_id", cmd.TicketID, "error", err)
		return nil, errors.NewNotFoundError(fmt.Sprintf("ticket %d not found", cmd.TicketID))
	}

	if err := uc.checkReopenPermission(t, cmd.ReopenedBy, cmd.UserRoles); err != nil {
		uc.logger.Errorw("permission denied to reopen ticket", "ticket_id", cmd.TicketID, "user_id", cmd.ReopenedBy, "error", err)
		return nil, err
	}

	if err := t.Reopen(cmd.Reason, cmd.ReopenedBy); err != nil {
		uc.logger.Errorw("failed to reopen ticket", "ticket_id", cmd.TicketID, "error", err)
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

	uc.logger.Infow("ticket reopened successfully", "ticket_id", cmd.TicketID)

	return &ReopenTicketResult{
		TicketID:   t.ID(),
		Status:     t.Status().String(),
		Reason:     cmd.Reason,
		ReopenedAt: t.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (uc *ReopenTicketUseCase) validateCommand(cmd ReopenTicketCommand) error {
	if cmd.TicketID == 0 {
		return errors.NewValidationError("ticket ID is required")
	}

	if cmd.Reason == "" {
		return errors.NewValidationError("reopen reason is required")
	}

	if cmd.ReopenedBy == 0 {
		return errors.NewValidationError("reopened by user ID is required")
	}

	return nil
}

func (uc *ReopenTicketUseCase) checkReopenPermission(t *ticket.Ticket, userID uint, userRoles []string) error {
	for _, role := range userRoles {
		if role == "admin" || role == "support_agent" {
			return nil
		}
	}

	if t.CreatorID() == userID {
		return nil
	}

	if t.AssigneeID() != nil && *t.AssigneeID() == userID {
		return nil
	}

	return errors.NewForbiddenError("user does not have permission to reopen this ticket")
}

package usecases

import (
	"context"
	"time"

	"orris/internal/domain/shared/events"
	"orris/internal/domain/ticket"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type UpdateTicketCommand struct {
	TicketID    uint
	Title       *string
	Description *string
	Tags        []string
	Metadata    map[string]interface{}
	UpdatedBy   uint
	UserRoles   []string
}

type UpdateTicketResult struct {
	TicketID  uint
	Title     string
	Status    string
	UpdatedAt time.Time
}

type UpdateTicketUseCase struct {
	ticketRepo      ticket.TicketRepository
	eventDispatcher events.EventDispatcher
	logger          logger.Interface
}

func NewUpdateTicketUseCase(
	ticketRepo ticket.TicketRepository,
	eventDispatcher events.EventDispatcher,
	logger logger.Interface,
) *UpdateTicketUseCase {
	return &UpdateTicketUseCase{
		ticketRepo:      ticketRepo,
		eventDispatcher: eventDispatcher,
		logger:          logger,
	}
}

func (uc *UpdateTicketUseCase) Execute(ctx context.Context, cmd UpdateTicketCommand) (*UpdateTicketResult, error) {
	uc.logger.Infow("executing update ticket use case", "ticket_id", cmd.TicketID, "updated_by", cmd.UpdatedBy)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid update ticket command", "error", err)
		return nil, err
	}

	existingTicket, err := uc.ticketRepo.GetByID(ctx, cmd.TicketID)
	if err != nil {
		uc.logger.Errorw("failed to get ticket", "error", err)
		return nil, err
	}

	if !existingTicket.CanBeViewedBy(cmd.UpdatedBy, cmd.UserRoles) {
		uc.logger.Warnw("user not authorized to view ticket", "ticket_id", cmd.TicketID, "user_id", cmd.UpdatedBy)
		return nil, errors.NewForbiddenError("user not authorized to view this ticket")
	}

	canUpdate := uc.canUserUpdate(existingTicket, cmd.UpdatedBy, cmd.UserRoles)
	if !canUpdate {
		uc.logger.Warnw("user not authorized to update ticket", "ticket_id", cmd.TicketID, "user_id", cmd.UpdatedBy)
		return nil, errors.NewForbiddenError("only creator, assignee, or admin can update ticket")
	}

	if err := uc.ticketRepo.Update(ctx, existingTicket); err != nil {
		uc.logger.Errorw("failed to update ticket", "error", err)
		return nil, err
	}

	domainEvents := existingTicket.GetEvents()
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

	uc.logger.Infow("ticket updated successfully", "ticket_id", existingTicket.ID())

	return &UpdateTicketResult{
		TicketID:  existingTicket.ID(),
		Title:     existingTicket.Title(),
		Status:    existingTicket.Status().String(),
		UpdatedAt: existingTicket.UpdatedAt(),
	}, nil
}

func (uc *UpdateTicketUseCase) validateCommand(cmd UpdateTicketCommand) error {
	if cmd.TicketID == 0 {
		return errors.NewValidationError("ticket ID is required")
	}

	if cmd.UpdatedBy == 0 {
		return errors.NewValidationError("updated by user ID is required")
	}

	if cmd.Title != nil {
		if len(*cmd.Title) == 0 {
			return errors.NewValidationError("title cannot be empty")
		}
		if len(*cmd.Title) > 200 {
			return errors.NewValidationError("title exceeds maximum length of 200 characters")
		}
	}

	if cmd.Description != nil {
		if len(*cmd.Description) == 0 {
			return errors.NewValidationError("description cannot be empty")
		}
		if len(*cmd.Description) > 5000 {
			return errors.NewValidationError("description exceeds maximum length of 5000 characters")
		}
	}

	return nil
}

func (uc *UpdateTicketUseCase) canUserUpdate(t *ticket.Ticket, userID uint, userRoles []string) bool {
	for _, role := range userRoles {
		if role == "admin" || role == "support_agent" {
			return true
		}
	}

	if t.CreatorID() == userID {
		return true
	}

	if assigneeID := t.AssigneeID(); assigneeID != nil && *assigneeID == userID {
		return true
	}

	return false
}

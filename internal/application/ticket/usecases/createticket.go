package usecases

import (
	"context"
	"time"

	"orris/internal/domain/shared/events"
	"orris/internal/domain/ticket"
	vo "orris/internal/domain/ticket/value_objects"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type CreateTicketCommand struct {
	Title       string
	Description string
	Category    string
	Priority    string
	CreatorID   uint
	Tags        []string
	Metadata    map[string]interface{}
}

type CreateTicketResult struct {
	TicketID  uint
	Number    string
	Status    string
	CreatedAt time.Time
}

type CreateTicketUseCase struct {
	ticketRepo      ticket.TicketRepository
	eventDispatcher events.EventDispatcher
	logger          logger.Interface
}

func NewCreateTicketUseCase(
	ticketRepo ticket.TicketRepository,
	eventDispatcher events.EventDispatcher,
	logger logger.Interface,
) *CreateTicketUseCase {
	return &CreateTicketUseCase{
		ticketRepo:      ticketRepo,
		eventDispatcher: eventDispatcher,
		logger:          logger,
	}
}

func (uc *CreateTicketUseCase) Execute(ctx context.Context, cmd CreateTicketCommand) (*CreateTicketResult, error) {
	uc.logger.Infow("executing create ticket use case", "title", cmd.Title, "creator_id", cmd.CreatorID)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid create ticket command", "error", err)
		return nil, err
	}

	category := vo.Category(cmd.Category)
	priority := vo.Priority(cmd.Priority)

	newTicket, err := ticket.NewTicket(
		cmd.Title,
		cmd.Description,
		category,
		priority,
		cmd.CreatorID,
	)
	if err != nil {
		uc.logger.Errorw("failed to create ticket entity", "error", err)
		return nil, errors.NewValidationError(err.Error())
	}

	if err := uc.ticketRepo.Save(ctx, newTicket); err != nil {
		uc.logger.Errorw("failed to save ticket", "error", err)
		return nil, err
	}

	domainEvents := newTicket.GetEvents()
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

	uc.logger.Infow("ticket created successfully", "ticket_id", newTicket.ID(), "number", newTicket.Number())

	return &CreateTicketResult{
		TicketID:  newTicket.ID(),
		Number:    newTicket.Number(),
		Status:    newTicket.Status().String(),
		CreatedAt: newTicket.CreatedAt(),
	}, nil
}

func (uc *CreateTicketUseCase) validateCommand(cmd CreateTicketCommand) error {
	if len(cmd.Title) == 0 {
		return errors.NewValidationError("title is required")
	}

	if len(cmd.Title) > 200 {
		return errors.NewValidationError("title exceeds maximum length of 200 characters")
	}

	if len(cmd.Description) == 0 {
		return errors.NewValidationError("description is required")
	}

	if len(cmd.Description) > 5000 {
		return errors.NewValidationError("description exceeds maximum length of 5000 characters")
	}

	if cmd.CreatorID == 0 {
		return errors.NewValidationError("creator ID is required")
	}

	category := vo.Category(cmd.Category)
	if !category.IsValid() {
		return errors.NewValidationError("invalid category")
	}

	priority := vo.Priority(cmd.Priority)
	if !priority.IsValid() {
		return errors.NewValidationError("invalid priority")
	}

	return nil
}

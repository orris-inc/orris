package usecases

import (
	"context"

	"orris/internal/domain/shared/events"
	"orris/internal/domain/ticket"
	"orris/internal/domain/user"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type AssignTicketCommand struct {
	TicketID   uint
	AssigneeID uint
	AssignedBy uint
}

type AssignTicketResult struct {
	TicketID   uint   `json:"ticket_id"`
	AssigneeID uint   `json:"assignee_id"`
	Status     string `json:"status"`
	UpdatedAt  string `json:"updated_at"`
}

type AssignTicketExecutor interface {
	Execute(ctx context.Context, cmd AssignTicketCommand) (*AssignTicketResult, error)
}

type AssignTicketUseCase struct {
	ticketRepo      ticket.TicketRepository
	userRepo        user.Repository
	eventDispatcher events.EventDispatcher
	logger          logger.Interface
}

func NewAssignTicketUseCase(
	ticketRepo ticket.TicketRepository,
	userRepo user.Repository,
	eventDispatcher events.EventDispatcher,
	logger logger.Interface,
) *AssignTicketUseCase {
	return &AssignTicketUseCase{
		ticketRepo:      ticketRepo,
		userRepo:        userRepo,
		eventDispatcher: eventDispatcher,
		logger:          logger,
	}
}

func (uc *AssignTicketUseCase) Execute(
	ctx context.Context,
	cmd AssignTicketCommand,
) (*AssignTicketResult, error) {
	uc.logger.Infow("executing assign ticket use case",
		"ticket_id", cmd.TicketID,
		"assignee_id", cmd.AssigneeID,
		"assigned_by", cmd.AssignedBy)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid assign ticket command", "error", err)
		return nil, err
	}

	assignee, err := uc.userRepo.GetByID(ctx, cmd.AssigneeID)
	if err != nil {
		uc.logger.Errorw("failed to find assignee", "error", err, "assignee_id", cmd.AssigneeID)
		return nil, errors.NewNotFoundError("assignee not found")
	}

	if !assignee.CanPerformActions() {
		uc.logger.Warnw("assignee cannot perform actions",
			"assignee_id", cmd.AssigneeID,
			"status", assignee.Status().String())
		return nil, errors.NewValidationError("assignee is not active and cannot be assigned tickets")
	}

	ticketAggregate, err := uc.ticketRepo.GetByID(ctx, cmd.TicketID)
	if err != nil {
		uc.logger.Errorw("failed to find ticket", "error", err, "ticket_id", cmd.TicketID)
		return nil, errors.NewNotFoundError("ticket not found")
	}

	if err := ticketAggregate.AssignTo(cmd.AssigneeID, cmd.AssignedBy); err != nil {
		uc.logger.Errorw("failed to assign ticket", "error", err)
		return nil, errors.NewValidationError(err.Error())
	}

	if err := uc.ticketRepo.Update(ctx, ticketAggregate); err != nil {
		uc.logger.Errorw("failed to update ticket", "error", err)
		return nil, errors.NewInternalError("failed to update ticket")
	}

	for _, event := range ticketAggregate.GetEvents() {
		if domainEvent, ok := event.(events.DomainEvent); ok {
			if err := uc.eventDispatcher.Publish(domainEvent); err != nil {
				uc.logger.Warnw("failed to dispatch event", "error", err)
			}
		}
	}

	uc.logger.Infow("ticket assigned successfully",
		"ticket_id", ticketAggregate.ID(),
		"assignee_id", cmd.AssigneeID)

	return &AssignTicketResult{
		TicketID:   ticketAggregate.ID(),
		AssigneeID: *ticketAggregate.AssigneeID(),
		Status:     ticketAggregate.Status().String(),
		UpdatedAt:  ticketAggregate.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (uc *AssignTicketUseCase) validateCommand(cmd AssignTicketCommand) error {
	if cmd.TicketID == 0 {
		return errors.NewValidationError("ticket ID is required")
	}
	if cmd.AssigneeID == 0 {
		return errors.NewValidationError("assignee ID is required")
	}
	if cmd.AssignedBy == 0 {
		return errors.NewValidationError("assigned by ID is required")
	}
	return nil
}

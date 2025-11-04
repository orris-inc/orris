package usecases

import (
	"context"
	"fmt"

	"orris/internal/domain/ticket"
	vo "orris/internal/domain/ticket/value_objects"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type ChangePriorityCommand struct {
	TicketID    uint
	NewPriority string
	ChangedBy   uint
}

type ChangePriorityResult struct {
	TicketID   uint   `json:"ticket_id"`
	Priority   string `json:"priority"`
	SLADueTime string `json:"sla_due_time"`
	UpdatedAt  string `json:"updated_at"`
}

type ChangePriorityExecutor interface {
	Execute(ctx context.Context, cmd ChangePriorityCommand) (*ChangePriorityResult, error)
}

type ChangePriorityUseCase struct {
	ticketRepo ticket.TicketRepository
	logger     logger.Interface
}

func NewChangePriorityUseCase(
	ticketRepo ticket.TicketRepository,
	logger logger.Interface,
) *ChangePriorityUseCase {
	return &ChangePriorityUseCase{
		ticketRepo: ticketRepo,
		logger:     logger,
	}
}

func (uc *ChangePriorityUseCase) Execute(
	ctx context.Context,
	cmd ChangePriorityCommand,
) (*ChangePriorityResult, error) {
	uc.logger.Infow("executing change priority use case",
		"ticket_id", cmd.TicketID,
		"new_priority", cmd.NewPriority,
		"changed_by", cmd.ChangedBy)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid change priority command", "error", err)
		return nil, err
	}

	priority, err := vo.NewPriority(cmd.NewPriority)
	if err != nil {
		return nil, errors.NewValidationError(fmt.Sprintf("invalid priority: %s", err))
	}

	ticketAggregate, err := uc.ticketRepo.GetByID(ctx, cmd.TicketID)
	if err != nil {
		uc.logger.Errorw("failed to find ticket", "error", err, "ticket_id", cmd.TicketID)
		return nil, errors.NewNotFoundError("ticket not found")
	}

	if err := ticketAggregate.ChangePriority(priority, cmd.ChangedBy); err != nil {
		uc.logger.Errorw("failed to change priority", "error", err)
		return nil, errors.NewValidationError(err.Error())
	}

	if err := uc.ticketRepo.Update(ctx, ticketAggregate); err != nil {
		uc.logger.Errorw("failed to update ticket", "error", err)
		return nil, errors.NewInternalError("failed to update ticket")
	}

	uc.logger.Infow("ticket priority changed successfully",
		"ticket_id", ticketAggregate.ID(),
		"new_priority", priority.String())

	result := &ChangePriorityResult{
		TicketID:  ticketAggregate.ID(),
		Priority:  ticketAggregate.Priority().String(),
		UpdatedAt: ticketAggregate.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}

	if ticketAggregate.SLADueTime() != nil {
		result.SLADueTime = ticketAggregate.SLADueTime().Format("2006-01-02T15:04:05Z07:00")
	}

	return result, nil
}

func (uc *ChangePriorityUseCase) validateCommand(cmd ChangePriorityCommand) error {
	if cmd.TicketID == 0 {
		return errors.NewValidationError("ticket ID is required")
	}
	if cmd.NewPriority == "" {
		return errors.NewValidationError("new priority is required")
	}
	if cmd.ChangedBy == 0 {
		return errors.NewValidationError("changed by ID is required")
	}
	return nil
}

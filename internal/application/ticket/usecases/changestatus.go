package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/ticket"
	vo "github.com/orris-inc/orris/internal/domain/ticket/valueobjects"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type ChangeStatusCommand struct {
	TicketID  uint
	NewStatus vo.TicketStatus
	ChangedBy uint
}

type ChangeStatusResult struct {
	TicketID  uint
	OldStatus string
	NewStatus string
	UpdatedAt string
}

type ChangeStatusUseCase struct {
	ticketRepo ticket.TicketRepository
	logger     logger.Interface
}

func NewChangeStatusUseCase(
	ticketRepo ticket.TicketRepository,
	logger logger.Interface,
) *ChangeStatusUseCase {
	return &ChangeStatusUseCase{
		ticketRepo: ticketRepo,
		logger:     logger,
	}
}

func (uc *ChangeStatusUseCase) Execute(ctx context.Context, cmd ChangeStatusCommand) (*ChangeStatusResult, error) {
	uc.logger.Infow("executing change status use case", "ticket_id", cmd.TicketID, "new_status", cmd.NewStatus)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid change status command", "error", err)
		return nil, err
	}

	t, err := uc.ticketRepo.GetByID(ctx, cmd.TicketID)
	if err != nil {
		uc.logger.Errorw("failed to get ticket", "ticket_id", cmd.TicketID, "error", err)
		return nil, errors.NewNotFoundError(fmt.Sprintf("ticket %d not found", cmd.TicketID))
	}

	oldStatus := t.Status()

	if err := t.ChangeStatus(cmd.NewStatus, cmd.ChangedBy); err != nil {
		uc.logger.Errorw("failed to change ticket status", "ticket_id", cmd.TicketID, "error", err)
		return nil, errors.NewValidationError(err.Error())
	}

	if err := uc.ticketRepo.Update(ctx, t); err != nil {
		uc.logger.Errorw("failed to update ticket", "ticket_id", cmd.TicketID, "error", err)
		return nil, errors.NewInternalError("failed to update ticket")
	}

	uc.logger.Infow("ticket status changed successfully", "ticket_id", cmd.TicketID, "old_status", oldStatus, "new_status", cmd.NewStatus)

	return &ChangeStatusResult{
		TicketID:  t.ID(),
		OldStatus: oldStatus.String(),
		NewStatus: t.Status().String(),
		UpdatedAt: t.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (uc *ChangeStatusUseCase) validateCommand(cmd ChangeStatusCommand) error {
	if cmd.TicketID == 0 {
		return errors.NewValidationError("ticket ID is required")
	}

	if !cmd.NewStatus.IsValid() {
		return errors.NewValidationError("invalid status")
	}

	if cmd.ChangedBy == 0 {
		return errors.NewValidationError("changed by user ID is required")
	}

	return nil
}

package usecases

import (
	"context"

	"orris/internal/application/ticket/dto"
)

type CreateTicketExecutor interface {
	Execute(ctx context.Context, cmd CreateTicketCommand) (*CreateTicketResult, error)
}

type UpdateTicketExecutor interface {
	Execute(ctx context.Context, cmd UpdateTicketCommand) (*UpdateTicketResult, error)
}

type DeleteTicketExecutor interface {
	Execute(ctx context.Context, cmd DeleteTicketCommand) (*DeleteTicketResult, error)
}

type GetTicketExecutor interface {
	Execute(ctx context.Context, query GetTicketQuery) (*dto.TicketDTO, error)
}

type AddCommentExecutor interface {
	Execute(ctx context.Context, cmd AddCommentCommand) (*AddCommentResult, error)
}

type ChangeStatusExecutor interface {
	Execute(ctx context.Context, cmd ChangeStatusCommand) (*ChangeStatusResult, error)
}

type UpdateTicketStatusExecutor = ChangeStatusExecutor

type UpdateTicketPriorityExecutor = ChangePriorityExecutor

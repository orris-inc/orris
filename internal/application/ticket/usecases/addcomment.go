package usecases

import (
	"context"
	"fmt"
	"time"

	"orris/internal/domain/shared/events"
	"orris/internal/domain/ticket"
	"orris/internal/shared/logger"
)

type AddCommentCommand struct {
	TicketID   uint
	UserID     uint
	UserRoles  []string
	Content    string
	IsInternal bool
}

type AddCommentResult struct {
	CommentID uint
	CreatedAt time.Time
}

type AddCommentUseCase struct {
	ticketRepo      ticket.TicketRepository
	commentRepo     ticket.CommentRepository
	eventDispatcher events.EventDispatcher
	logger          logger.Interface
}

func NewAddCommentUseCase(
	ticketRepo ticket.TicketRepository,
	commentRepo ticket.CommentRepository,
	eventDispatcher events.EventDispatcher,
	logger logger.Interface,
) *AddCommentUseCase {
	return &AddCommentUseCase{
		ticketRepo:      ticketRepo,
		commentRepo:     commentRepo,
		eventDispatcher: eventDispatcher,
		logger:          logger,
	}
}

func (uc *AddCommentUseCase) Execute(ctx context.Context, cmd AddCommentCommand) (*AddCommentResult, error) {
	uc.logger.Infow("executing add comment use case", "ticket_id", cmd.TicketID, "user_id", cmd.UserID)

	t, err := uc.ticketRepo.GetByID(ctx, cmd.TicketID)
	if err != nil {
		uc.logger.Errorw("failed to load ticket", "ticket_id", cmd.TicketID, "error", err)
		return nil, fmt.Errorf("failed to load ticket: %w", err)
	}

	if !t.CanBeViewedBy(cmd.UserID, cmd.UserRoles) {
		uc.logger.Warnw("user cannot view ticket", "ticket_id", cmd.TicketID, "user_id", cmd.UserID)
		return nil, fmt.Errorf("permission denied: cannot view ticket")
	}

	if cmd.IsInternal {
		hasPermission := false
		for _, role := range cmd.UserRoles {
			if role == "admin" || role == "support_agent" {
				hasPermission = true
				break
			}
		}
		if !hasPermission {
			uc.logger.Warnw("user cannot create internal comment", "user_id", cmd.UserID)
			return nil, fmt.Errorf("permission denied: only admin/support_agent can create internal comments")
		}
	}

	comment, err := ticket.NewComment(cmd.TicketID, cmd.UserID, cmd.Content, cmd.IsInternal)
	if err != nil {
		uc.logger.Errorw("failed to create comment", "error", err)
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}

	if err := uc.commentRepo.Save(ctx, comment); err != nil {
		uc.logger.Errorw("failed to save comment", "error", err)
		return nil, fmt.Errorf("failed to save comment: %w", err)
	}

	if err := t.AddComment(comment); err != nil {
		uc.logger.Errorw("failed to add comment to ticket", "error", err)
		return nil, fmt.Errorf("failed to add comment to ticket: %w", err)
	}

	if err := uc.ticketRepo.Update(ctx, t); err != nil {
		uc.logger.Errorw("failed to update ticket", "error", err)
		return nil, fmt.Errorf("failed to update ticket: %w", err)
	}

	domainEvents := t.GetEvents()
	for _, event := range domainEvents {
		if domainEvent, ok := event.(events.DomainEvent); ok {
			if err := uc.eventDispatcher.Publish(domainEvent); err != nil {
				uc.logger.Warnw("failed to publish event", "event_type", domainEvent.GetEventType(), "error", err)
			}
		}
	}

	result := &AddCommentResult{
		CommentID: comment.ID(),
		CreatedAt: comment.CreatedAt(),
	}

	uc.logger.Infow("comment added successfully", "comment_id", result.CommentID, "ticket_id", cmd.TicketID)
	return result, nil
}

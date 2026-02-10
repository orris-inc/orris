package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/domain/ticket"
	"github.com/orris-inc/orris/internal/shared/auth"
	"github.com/orris-inc/orris/internal/shared/db"
	"github.com/orris-inc/orris/internal/shared/logger"
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
	ticketRepo  ticket.TicketRepository
	commentRepo ticket.CommentRepository
	txMgr       *db.TransactionManager
	logger      logger.Interface
}

func NewAddCommentUseCase(
	ticketRepo ticket.TicketRepository,
	commentRepo ticket.CommentRepository,
	txMgr *db.TransactionManager,
	logger logger.Interface,
) *AddCommentUseCase {
	return &AddCommentUseCase{
		ticketRepo:  ticketRepo,
		commentRepo: commentRepo,
		txMgr:       txMgr,
		logger:      logger,
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
		if !auth.IsAdminOrAgent(cmd.UserRoles) {
			uc.logger.Warnw("user cannot create internal comment", "user_id", cmd.UserID)
			return nil, fmt.Errorf("permission denied: only admin/support_agent can create internal comments")
		}
	}

	comment, err := ticket.NewComment(cmd.TicketID, cmd.UserID, cmd.Content, cmd.IsInternal)
	if err != nil {
		uc.logger.Errorw("failed to create comment", "error", err)
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}

	// Use database transaction to ensure comment save + ticket update is atomic.
	// If any step fails, the entire operation is rolled back automatically.
	txErr := uc.txMgr.RunInTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.commentRepo.Save(txCtx, comment); err != nil {
			uc.logger.Errorw("failed to save comment", "error", err)
			return fmt.Errorf("failed to save comment: %w", err)
		}

		if err := t.AddComment(comment); err != nil {
			uc.logger.Errorw("failed to add comment to ticket", "error", err)
			return fmt.Errorf("failed to add comment to ticket: %w", err)
		}

		if err := uc.ticketRepo.Update(txCtx, t); err != nil {
			uc.logger.Errorw("failed to update ticket", "error", err)
			return fmt.Errorf("failed to update ticket: %w", err)
		}

		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	result := &AddCommentResult{
		CommentID: comment.ID(),
		CreatedAt: comment.CreatedAt(),
	}

	uc.logger.Infow("comment added successfully", "comment_id", result.CommentID, "ticket_id", cmd.TicketID)
	return result, nil
}

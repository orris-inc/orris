package usecases

import (
	"context"
	"fmt"
	"time"

	"orris/internal/domain/ticket"
	"orris/internal/shared/logger"
)

type GetTicketQuery struct {
	TicketID  uint
	UserID    uint
	UserRoles []string
}

type CommentDTO struct {
	ID         uint      `json:"id"`
	UserID     uint      `json:"user_id"`
	Content    string    `json:"content"`
	IsInternal bool      `json:"is_internal"`
	CreatedAt  time.Time `json:"created_at"`
}

type GetTicketResult struct {
	ID           uint                   `json:"id"`
	Number       string                 `json:"number"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	Category     string                 `json:"category"`
	Priority     string                 `json:"priority"`
	Status       string                 `json:"status"`
	CreatorID    uint                   `json:"creator_id"`
	AssigneeID   *uint                  `json:"assignee_id"`
	Tags         []string               `json:"tags"`
	Metadata     map[string]interface{} `json:"metadata"`
	SLADueTime   *time.Time             `json:"sla_due_time"`
	ResponseTime *time.Time             `json:"response_time"`
	ResolvedTime *time.Time             `json:"resolved_time"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	ClosedAt     *time.Time             `json:"closed_at"`
	Comments     []CommentDTO           `json:"comments"`
}

type GetTicketUseCase struct {
	ticketRepo  ticket.TicketRepository
	commentRepo ticket.CommentRepository
	logger      logger.Interface
}

func NewGetTicketUseCase(
	ticketRepo ticket.TicketRepository,
	commentRepo ticket.CommentRepository,
	logger logger.Interface,
) *GetTicketUseCase {
	return &GetTicketUseCase{
		ticketRepo:  ticketRepo,
		commentRepo: commentRepo,
		logger:      logger,
	}
}

func (uc *GetTicketUseCase) Execute(ctx context.Context, query GetTicketQuery) (*GetTicketResult, error) {
	uc.logger.Infow("executing get ticket use case", "ticket_id", query.TicketID, "user_id", query.UserID)

	t, err := uc.ticketRepo.GetByID(ctx, query.TicketID)
	if err != nil {
		uc.logger.Errorw("failed to load ticket", "ticket_id", query.TicketID, "error", err)
		return nil, fmt.Errorf("failed to load ticket: %w", err)
	}

	if !t.CanBeViewedBy(query.UserID, query.UserRoles) {
		uc.logger.Warnw("user cannot view ticket", "ticket_id", query.TicketID, "user_id", query.UserID)
		return nil, fmt.Errorf("permission denied: cannot view ticket")
	}

	comments, err := uc.commentRepo.GetByTicketID(ctx, query.TicketID)
	if err != nil {
		uc.logger.Errorw("failed to load comments", "ticket_id", query.TicketID, "error", err)
		return nil, fmt.Errorf("failed to load comments: %w", err)
	}

	isAgentOrAdmin := false
	for _, role := range query.UserRoles {
		if role == "admin" || role == "support_agent" {
			isAgentOrAdmin = true
			break
		}
	}

	commentDTOs := make([]CommentDTO, 0)
	for _, c := range comments {
		if c.IsInternal() && !isAgentOrAdmin {
			continue
		}
		commentDTOs = append(commentDTOs, CommentDTO{
			ID:         c.ID(),
			UserID:     c.UserID(),
			Content:    c.Content(),
			IsInternal: c.IsInternal(),
			CreatedAt:  c.CreatedAt(),
		})
	}

	result := &GetTicketResult{
		ID:           t.ID(),
		Number:       t.Number(),
		Title:        t.Title(),
		Description:  t.Description(),
		Category:     t.Category().String(),
		Priority:     t.Priority().String(),
		Status:       t.Status().String(),
		CreatorID:    t.CreatorID(),
		AssigneeID:   t.AssigneeID(),
		Tags:         t.Tags(),
		Metadata:     t.Metadata(),
		SLADueTime:   t.SLADueTime(),
		ResponseTime: t.ResponseTime(),
		ResolvedTime: t.ResolvedTime(),
		CreatedAt:    t.CreatedAt(),
		UpdatedAt:    t.UpdatedAt(),
		ClosedAt:     t.ClosedAt(),
		Comments:     commentDTOs,
	}

	uc.logger.Infow("ticket retrieved successfully", "ticket_id", query.TicketID)
	return result, nil
}

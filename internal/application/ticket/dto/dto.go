package dto

import (
	"time"

	"github.com/orris-inc/orris/internal/domain/ticket"
	"github.com/orris-inc/orris/internal/shared/mapper"
)

type TicketDTO struct {
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

type CommentDTO struct {
	ID         uint      `json:"id"`
	UserID     uint      `json:"user_id"`
	Content    string    `json:"content"`
	IsInternal bool      `json:"is_internal"`
	CreatedAt  time.Time `json:"created_at"`
}

type TicketListItemDTO struct {
	ID         uint    `json:"id"`
	Number     string  `json:"number"`
	Title      string  `json:"title"`
	Status     string  `json:"status"`
	Priority   string  `json:"priority"`
	Category   string  `json:"category"`
	CreatorID  uint    `json:"creator_id"`
	AssigneeID *uint   `json:"assignee_id"`
	SLADueTime *string `json:"sla_due_time"`
	IsOverdue  bool    `json:"is_overdue"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

func ToTicketDTO(t *ticket.Ticket, comments []*ticket.Comment, isAgentOrAdmin bool) *TicketDTO {
	if t == nil {
		return nil
	}

	commentDTOs := make([]CommentDTO, 0)
	for _, c := range comments {
		if c.IsInternal() && !isAgentOrAdmin {
			continue
		}
		commentDTOs = append(commentDTOs, ToCommentDTO(c))
	}

	return &TicketDTO{
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
}

func ToCommentDTO(c *ticket.Comment) CommentDTO {
	return CommentDTO{
		ID:         c.ID(),
		UserID:     c.UserID(),
		Content:    c.Content(),
		IsInternal: c.IsInternal(),
		CreatedAt:  c.CreatedAt(),
	}
}

func ToTicketListItemDTO(t *ticket.Ticket) TicketListItemDTO {
	item := TicketListItemDTO{
		ID:         t.ID(),
		Number:     t.Number(),
		Title:      t.Title(),
		Status:     t.Status().String(),
		Priority:   t.Priority().String(),
		Category:   t.Category().String(),
		CreatorID:  t.CreatorID(),
		AssigneeID: t.AssigneeID(),
		IsOverdue:  t.IsOverdue(),
		CreatedAt:  t.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:  t.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}

	if t.SLADueTime() != nil {
		slaTime := t.SLADueTime().Format("2006-01-02T15:04:05Z07:00")
		item.SLADueTime = &slaTime
	}

	return item
}

func ToTicketListItemDTOs(tickets []*ticket.Ticket) []TicketListItemDTO {
	return mapper.MapSlice(tickets, ToTicketListItemDTO)
}

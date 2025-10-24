package ticket

import (
	"context"

	vo "orris/internal/domain/ticket/value_objects"
)

type TicketRepository interface {
	Save(ctx context.Context, ticket *Ticket) error
	Update(ctx context.Context, ticket *Ticket) error
	Delete(ctx context.Context, ticketID uint) error
	GetByID(ctx context.Context, ticketID uint) (*Ticket, error)
	GetByNumber(ctx context.Context, number string) (*Ticket, error)
	List(ctx context.Context, filters TicketFilter) ([]*Ticket, int64, error)
	GetUserTickets(ctx context.Context, userID uint, filters TicketFilter) ([]*Ticket, int64, error)
	GetAssignedTickets(ctx context.Context, assigneeID uint, filters TicketFilter) ([]*Ticket, int64, error)
	GetOverdueTickets(ctx context.Context) ([]*Ticket, error)
}

type TicketFilter struct {
	Status     *vo.TicketStatus
	Priority   *vo.Priority
	Category   *vo.Category
	CreatorID  *uint
	AssigneeID *uint
	Tags       []string
	Overdue    *bool
	Page       int
	PageSize   int
	SortBy     string
	SortOrder  string
}

type CommentRepository interface {
	Save(ctx context.Context, comment *Comment) error
	GetByTicketID(ctx context.Context, ticketID uint) ([]*Comment, error)
	Delete(ctx context.Context, commentID uint) error
}

package ticket

import (
	"time"
)

type TicketCreatedEvent struct {
	TicketID  uint
	Number    string
	Title     string
	CreatorID uint
	Priority  string
	Category  string
	Timestamp time.Time
}

func NewTicketCreatedEvent(
	ticketID uint,
	number string,
	title string,
	creatorID uint,
	priority string,
	category string,
	timestamp time.Time,
) TicketCreatedEvent {
	return TicketCreatedEvent{
		TicketID:  ticketID,
		Number:    number,
		Title:     title,
		CreatorID: creatorID,
		Priority:  priority,
		Category:  category,
		Timestamp: timestamp,
	}
}

type TicketAssignedEvent struct {
	TicketID   uint
	AssigneeID uint
	AssignedBy uint
	Timestamp  time.Time
}

func NewTicketAssignedEvent(
	ticketID uint,
	assigneeID uint,
	assignedBy uint,
	timestamp time.Time,
) TicketAssignedEvent {
	return TicketAssignedEvent{
		TicketID:   ticketID,
		AssigneeID: assigneeID,
		AssignedBy: assignedBy,
		Timestamp:  timestamp,
	}
}

type TicketStatusChangedEvent struct {
	TicketID  uint
	OldStatus string
	NewStatus string
	ChangedBy uint
	Timestamp time.Time
}

func NewTicketStatusChangedEvent(
	ticketID uint,
	oldStatus string,
	newStatus string,
	changedBy uint,
	timestamp time.Time,
) TicketStatusChangedEvent {
	return TicketStatusChangedEvent{
		TicketID:  ticketID,
		OldStatus: oldStatus,
		NewStatus: newStatus,
		ChangedBy: changedBy,
		Timestamp: timestamp,
	}
}

type TicketPriorityChangedEvent struct {
	TicketID    uint
	OldPriority string
	NewPriority string
	ChangedBy   uint
	Timestamp   time.Time
}

func NewTicketPriorityChangedEvent(
	ticketID uint,
	oldPriority string,
	newPriority string,
	changedBy uint,
	timestamp time.Time,
) TicketPriorityChangedEvent {
	return TicketPriorityChangedEvent{
		TicketID:    ticketID,
		OldPriority: oldPriority,
		NewPriority: newPriority,
		ChangedBy:   changedBy,
		Timestamp:   timestamp,
	}
}

type TicketClosedEvent struct {
	TicketID  uint
	Reason    string
	ClosedBy  uint
	Timestamp time.Time
}

func NewTicketClosedEvent(
	ticketID uint,
	reason string,
	closedBy uint,
	timestamp time.Time,
) TicketClosedEvent {
	return TicketClosedEvent{
		TicketID:  ticketID,
		Reason:    reason,
		ClosedBy:  closedBy,
		Timestamp: timestamp,
	}
}

type TicketReopenedEvent struct {
	TicketID   uint
	Reason     string
	ReopenedBy uint
	Timestamp  time.Time
}

func NewTicketReopenedEvent(
	ticketID uint,
	reason string,
	reopenedBy uint,
	timestamp time.Time,
) TicketReopenedEvent {
	return TicketReopenedEvent{
		TicketID:   ticketID,
		Reason:     reason,
		ReopenedBy: reopenedBy,
		Timestamp:  timestamp,
	}
}

type CommentAddedEvent struct {
	TicketID   uint
	CommentID  uint
	UserID     uint
	IsInternal bool
	Timestamp  time.Time
}

func NewCommentAddedEvent(
	ticketID uint,
	commentID uint,
	userID uint,
	isInternal bool,
	timestamp time.Time,
) CommentAddedEvent {
	return CommentAddedEvent{
		TicketID:   ticketID,
		CommentID:  commentID,
		UserID:     userID,
		IsInternal: isInternal,
		Timestamp:  timestamp,
	}
}

type SLAViolatedEvent struct {
	TicketID  uint
	SLAType   string
	DueTime   time.Time
	Timestamp time.Time
}

func NewSLAViolatedEvent(
	ticketID uint,
	slaType string,
	dueTime time.Time,
	timestamp time.Time,
) SLAViolatedEvent {
	return SLAViolatedEvent{
		TicketID:  ticketID,
		SLAType:   slaType,
		DueTime:   dueTime,
		Timestamp: timestamp,
	}
}

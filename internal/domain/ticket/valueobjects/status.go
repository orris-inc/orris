package valueobjects

import "fmt"

type TicketStatus string

const (
	StatusNew        TicketStatus = "new"
	StatusOpen       TicketStatus = "open"
	StatusInProgress TicketStatus = "in_progress"
	StatusPending    TicketStatus = "pending"
	StatusResolved   TicketStatus = "resolved"
	StatusClosed     TicketStatus = "closed"
	StatusReopened   TicketStatus = "reopened"
)

var validTicketStatuses = map[TicketStatus]bool{
	StatusNew:        true,
	StatusOpen:       true,
	StatusInProgress: true,
	StatusPending:    true,
	StatusResolved:   true,
	StatusClosed:     true,
	StatusReopened:   true,
}

var ticketStatusTransitions = map[TicketStatus][]TicketStatus{
	StatusNew: {
		StatusOpen,
		StatusClosed,
	},
	StatusOpen: {
		StatusInProgress,
		StatusPending,
		StatusClosed,
	},
	StatusInProgress: {
		StatusPending,
		StatusResolved,
		StatusClosed,
	},
	StatusPending: {
		StatusInProgress,
		StatusResolved,
		StatusClosed,
	},
	StatusResolved: {
		StatusClosed,
		StatusReopened,
	},
	StatusClosed: {
		StatusReopened,
	},
	StatusReopened: {
		StatusOpen,
		StatusInProgress,
		StatusClosed,
	},
}

func (ts TicketStatus) String() string {
	return string(ts)
}

func (ts TicketStatus) IsValid() bool {
	return validTicketStatuses[ts]
}

func (ts TicketStatus) CanTransitionTo(newStatus TicketStatus) bool {
	allowedTransitions, ok := ticketStatusTransitions[ts]
	if !ok {
		return false
	}

	for _, allowed := range allowedTransitions {
		if allowed == newStatus {
			return true
		}
	}
	return false
}

func (ts TicketStatus) IsNew() bool {
	return ts == StatusNew
}

func (ts TicketStatus) IsOpen() bool {
	return ts == StatusOpen
}

func (ts TicketStatus) IsInProgress() bool {
	return ts == StatusInProgress
}

func (ts TicketStatus) IsPending() bool {
	return ts == StatusPending
}

func (ts TicketStatus) IsResolved() bool {
	return ts == StatusResolved
}

func (ts TicketStatus) IsClosed() bool {
	return ts == StatusClosed
}

func (ts TicketStatus) IsReopened() bool {
	return ts == StatusReopened
}

func NewTicketStatus(s string) (TicketStatus, error) {
	ts := TicketStatus(s)
	if !ts.IsValid() {
		return "", fmt.Errorf("invalid ticket status: %s", s)
	}
	return ts, nil
}

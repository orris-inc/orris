package value_objects

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTicketStatus(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    TicketStatus
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid new status",
			input:   "new",
			want:    StatusNew,
			wantErr: false,
		},
		{
			name:    "valid open status",
			input:   "open",
			want:    StatusOpen,
			wantErr: false,
		},
		{
			name:    "valid in_progress status",
			input:   "in_progress",
			want:    StatusInProgress,
			wantErr: false,
		},
		{
			name:    "valid pending status",
			input:   "pending",
			want:    StatusPending,
			wantErr: false,
		},
		{
			name:    "valid resolved status",
			input:   "resolved",
			want:    StatusResolved,
			wantErr: false,
		},
		{
			name:    "valid closed status",
			input:   "closed",
			want:    StatusClosed,
			wantErr: false,
		},
		{
			name:    "valid reopened status",
			input:   "reopened",
			want:    StatusReopened,
			wantErr: false,
		},
		{
			name:    "invalid status",
			input:   "invalid",
			wantErr: true,
			errMsg:  "invalid ticket status: invalid",
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
			errMsg:  "invalid ticket status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTicketStatus(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestTicketStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status TicketStatus
		want   bool
	}{
		{"new is valid", StatusNew, true},
		{"open is valid", StatusOpen, true},
		{"in_progress is valid", StatusInProgress, true},
		{"pending is valid", StatusPending, true},
		{"resolved is valid", StatusResolved, true},
		{"closed is valid", StatusClosed, true},
		{"reopened is valid", StatusReopened, true},
		{"invalid status", TicketStatus("invalid"), false},
		{"empty status", TicketStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.IsValid())
		})
	}
}

func TestTicketStatus_String(t *testing.T) {
	tests := []struct {
		name   string
		status TicketStatus
		want   string
	}{
		{"new", StatusNew, "new"},
		{"open", StatusOpen, "open"},
		{"in_progress", StatusInProgress, "in_progress"},
		{"pending", StatusPending, "pending"},
		{"resolved", StatusResolved, "resolved"},
		{"closed", StatusClosed, "closed"},
		{"reopened", StatusReopened, "reopened"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.String())
		})
	}
}

func TestTicketStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name          string
		currentStatus TicketStatus
		newStatus     TicketStatus
		want          bool
	}{
		{"new to open", StatusNew, StatusOpen, true},
		{"new to closed", StatusNew, StatusClosed, true},
		{"new to in_progress invalid", StatusNew, StatusInProgress, false},
		{"new to resolved invalid", StatusNew, StatusResolved, false},

		{"open to in_progress", StatusOpen, StatusInProgress, true},
		{"open to pending", StatusOpen, StatusPending, true},
		{"open to closed", StatusOpen, StatusClosed, true},
		{"open to new invalid", StatusOpen, StatusNew, false},
		{"open to resolved invalid", StatusOpen, StatusResolved, false},

		{"in_progress to pending", StatusInProgress, StatusPending, true},
		{"in_progress to resolved", StatusInProgress, StatusResolved, true},
		{"in_progress to closed", StatusInProgress, StatusClosed, true},
		{"in_progress to new invalid", StatusInProgress, StatusNew, false},
		{"in_progress to open invalid", StatusInProgress, StatusOpen, false},

		{"pending to in_progress", StatusPending, StatusInProgress, true},
		{"pending to resolved", StatusPending, StatusResolved, true},
		{"pending to closed", StatusPending, StatusClosed, true},
		{"pending to new invalid", StatusPending, StatusNew, false},
		{"pending to open invalid", StatusPending, StatusOpen, false},

		{"resolved to closed", StatusResolved, StatusClosed, true},
		{"resolved to reopened", StatusResolved, StatusReopened, true},
		{"resolved to new invalid", StatusResolved, StatusNew, false},
		{"resolved to open invalid", StatusResolved, StatusOpen, false},
		{"resolved to in_progress invalid", StatusResolved, StatusInProgress, false},

		{"closed to reopened", StatusClosed, StatusReopened, true},
		{"closed to new invalid", StatusClosed, StatusNew, false},
		{"closed to open invalid", StatusClosed, StatusOpen, false},
		{"closed to in_progress invalid", StatusClosed, StatusInProgress, false},
		{"closed to resolved invalid", StatusClosed, StatusResolved, false},

		{"reopened to open", StatusReopened, StatusOpen, true},
		{"reopened to in_progress", StatusReopened, StatusInProgress, true},
		{"reopened to closed", StatusReopened, StatusClosed, true},
		{"reopened to new invalid", StatusReopened, StatusNew, false},
		{"reopened to resolved invalid", StatusReopened, StatusResolved, false},

		{"invalid status cannot transition", TicketStatus("invalid"), StatusOpen, false},
		{"to invalid status", StatusOpen, TicketStatus("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.currentStatus.CanTransitionTo(tt.newStatus)
			assert.Equal(t, tt.want, result, "transition from %s to %s", tt.currentStatus, tt.newStatus)
		})
	}
}

func TestTicketStatus_StateCheckers(t *testing.T) {
	tests := []struct {
		name     string
		status   TicketStatus
		checker  string
		expected bool
	}{
		{"new is new", StatusNew, "IsNew", true},
		{"open is not new", StatusOpen, "IsNew", false},

		{"open is open", StatusOpen, "IsOpen", true},
		{"new is not open", StatusNew, "IsOpen", false},

		{"in_progress is in_progress", StatusInProgress, "IsInProgress", true},
		{"open is not in_progress", StatusOpen, "IsInProgress", false},

		{"pending is pending", StatusPending, "IsPending", true},
		{"open is not pending", StatusOpen, "IsPending", false},

		{"resolved is resolved", StatusResolved, "IsResolved", true},
		{"open is not resolved", StatusOpen, "IsResolved", false},

		{"closed is closed", StatusClosed, "IsClosed", true},
		{"open is not closed", StatusOpen, "IsClosed", false},

		{"reopened is reopened", StatusReopened, "IsReopened", true},
		{"open is not reopened", StatusOpen, "IsReopened", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result bool
			switch tt.checker {
			case "IsNew":
				result = tt.status.IsNew()
			case "IsOpen":
				result = tt.status.IsOpen()
			case "IsInProgress":
				result = tt.status.IsInProgress()
			case "IsPending":
				result = tt.status.IsPending()
			case "IsResolved":
				result = tt.status.IsResolved()
			case "IsClosed":
				result = tt.status.IsClosed()
			case "IsReopened":
				result = tt.status.IsReopened()
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTicketStatus_TransitionPaths(t *testing.T) {
	t.Run("typical workflow: new -> open -> in_progress -> resolved -> closed", func(t *testing.T) {
		current := StatusNew
		assert.True(t, current.CanTransitionTo(StatusOpen))

		current = StatusOpen
		assert.True(t, current.CanTransitionTo(StatusInProgress))

		current = StatusInProgress
		assert.True(t, current.CanTransitionTo(StatusResolved))

		current = StatusResolved
		assert.True(t, current.CanTransitionTo(StatusClosed))
	})

	t.Run("reopen workflow: closed -> reopened -> in_progress -> resolved -> closed", func(t *testing.T) {
		current := StatusClosed
		assert.True(t, current.CanTransitionTo(StatusReopened))

		current = StatusReopened
		assert.True(t, current.CanTransitionTo(StatusInProgress))

		current = StatusInProgress
		assert.True(t, current.CanTransitionTo(StatusResolved))

		current = StatusResolved
		assert.True(t, current.CanTransitionTo(StatusClosed))
	})

	t.Run("pending workflow: open -> pending -> in_progress -> resolved", func(t *testing.T) {
		current := StatusOpen
		assert.True(t, current.CanTransitionTo(StatusPending))

		current = StatusPending
		assert.True(t, current.CanTransitionTo(StatusInProgress))

		current = StatusInProgress
		assert.True(t, current.CanTransitionTo(StatusResolved))
	})

	t.Run("direct close: new -> closed", func(t *testing.T) {
		assert.True(t, StatusNew.CanTransitionTo(StatusClosed))
	})

	t.Run("direct close from open", func(t *testing.T) {
		assert.True(t, StatusOpen.CanTransitionTo(StatusClosed))
	})
}

func TestTicketStatus_AllStatusesAreValid(t *testing.T) {
	statuses := []TicketStatus{
		StatusNew,
		StatusOpen,
		StatusInProgress,
		StatusPending,
		StatusResolved,
		StatusClosed,
		StatusReopened,
	}

	for _, status := range statuses {
		t.Run(status.String(), func(t *testing.T) {
			assert.True(t, status.IsValid(), "status %s should be valid", status)
		})
	}
}

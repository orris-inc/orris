package ticket

import (
	"testing"
	"time"

	vo "orris/internal/domain/ticket/value_objects"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTicket(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		description string
		category    vo.Category
		priority    vo.Priority
		creatorID   uint
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "valid ticket",
			title:       "Test Ticket",
			description: "This is a test ticket description",
			category:    vo.CategoryTechnical,
			priority:    vo.PriorityMedium,
			creatorID:   1,
			wantErr:     false,
		},
		{
			name:        "empty title",
			title:       "",
			description: "Description",
			category:    vo.CategoryTechnical,
			priority:    vo.PriorityMedium,
			creatorID:   1,
			wantErr:     true,
			errMsg:      "title is required",
		},
		{
			name:        "title too long",
			title:       string(make([]byte, 201)),
			description: "Description",
			category:    vo.CategoryTechnical,
			priority:    vo.PriorityMedium,
			creatorID:   1,
			wantErr:     true,
			errMsg:      "title exceeds maximum length",
		},
		{
			name:        "empty description",
			title:       "Title",
			description: "",
			category:    vo.CategoryTechnical,
			priority:    vo.PriorityMedium,
			creatorID:   1,
			wantErr:     true,
			errMsg:      "description is required",
		},
		{
			name:        "description too long",
			title:       "Title",
			description: string(make([]byte, 5001)),
			category:    vo.CategoryTechnical,
			priority:    vo.PriorityMedium,
			creatorID:   1,
			wantErr:     true,
			errMsg:      "description exceeds maximum length",
		},
		{
			name:        "invalid category",
			title:       "Title",
			description: "Description",
			category:    vo.Category("invalid"),
			priority:    vo.PriorityMedium,
			creatorID:   1,
			wantErr:     true,
			errMsg:      "invalid category",
		},
		{
			name:        "invalid priority",
			title:       "Title",
			description: "Description",
			category:    vo.CategoryTechnical,
			priority:    vo.Priority("invalid"),
			creatorID:   1,
			wantErr:     true,
			errMsg:      "invalid priority",
		},
		{
			name:        "zero creator ID",
			title:       "Title",
			description: "Description",
			category:    vo.CategoryTechnical,
			priority:    vo.PriorityMedium,
			creatorID:   0,
			wantErr:     true,
			errMsg:      "creator ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket, err := NewTicket(
				tt.title,
				tt.description,
				tt.category,
				tt.priority,
				tt.creatorID,
			)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, ticket)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, ticket)
				assert.Equal(t, tt.title, ticket.Title())
				assert.Equal(t, tt.description, ticket.Description())
				assert.Equal(t, vo.StatusNew, ticket.Status())
				assert.NotNil(t, ticket.SLADueTime())
			}
		})
	}
}

func TestTicket_AssignTo(t *testing.T) {
	tests := []struct {
		name       string
		assigneeID uint
		assignedBy uint
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid assignment",
			assigneeID: 2,
			assignedBy: 3,
			wantErr:    false,
		},
		{
			name:       "zero assignee ID",
			assigneeID: 0,
			assignedBy: 3,
			wantErr:    true,
			errMsg:     "assignee ID cannot be zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket, err := createTestTicket()
			require.NoError(t, err)

			err = ticket.AssignTo(tt.assigneeID, tt.assignedBy)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.assigneeID, *ticket.AssigneeID())
				assert.Equal(t, vo.StatusOpen, ticket.Status())
			}
		})
	}
}

func TestTicket_ChangeStatus(t *testing.T) {
	tests := []struct {
		name          string
		setupStatuses []vo.TicketStatus
		newStatus     vo.TicketStatus
		changedBy     uint
		wantErr       bool
		checkResolved bool
		checkClosed   bool
		checkReopened bool
	}{
		{
			name:          "new to open",
			setupStatuses: []vo.TicketStatus{},
			newStatus:     vo.StatusOpen,
			changedBy:     1,
			wantErr:       false,
		},
		{
			name:          "open to in_progress",
			setupStatuses: []vo.TicketStatus{vo.StatusOpen},
			newStatus:     vo.StatusInProgress,
			changedBy:     1,
			wantErr:       false,
		},
		{
			name:          "in_progress to resolved",
			setupStatuses: []vo.TicketStatus{vo.StatusOpen, vo.StatusInProgress},
			newStatus:     vo.StatusResolved,
			changedBy:     1,
			wantErr:       false,
			checkResolved: true,
		},
		{
			name:          "resolved to closed",
			setupStatuses: []vo.TicketStatus{vo.StatusOpen, vo.StatusInProgress, vo.StatusResolved},
			newStatus:     vo.StatusClosed,
			changedBy:     1,
			wantErr:       false,
			checkClosed:   true,
		},
		{
			name:          "closed to reopened",
			setupStatuses: []vo.TicketStatus{vo.StatusOpen, vo.StatusInProgress, vo.StatusResolved, vo.StatusClosed},
			newStatus:     vo.StatusReopened,
			changedBy:     1,
			wantErr:       false,
			checkReopened: true,
		},
		{
			name:          "invalid transition: new to resolved",
			setupStatuses: []vo.TicketStatus{},
			newStatus:     vo.StatusResolved,
			changedBy:     1,
			wantErr:       true,
		},
		{
			name:          "invalid status",
			setupStatuses: []vo.TicketStatus{},
			newStatus:     vo.TicketStatus("invalid"),
			changedBy:     1,
			wantErr:       true,
		},
		{
			name:          "same status no change",
			setupStatuses: []vo.TicketStatus{vo.StatusOpen},
			newStatus:     vo.StatusOpen,
			changedBy:     1,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket, err := createTestTicket()
			require.NoError(t, err)

			for _, status := range tt.setupStatuses {
				_ = ticket.ChangeStatus(status, 1)
			}

			err = ticket.ChangeStatus(tt.newStatus, tt.changedBy)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.newStatus != ticket.Status() && len(tt.setupStatuses) > 0 && tt.setupStatuses[len(tt.setupStatuses)-1] == tt.newStatus {
					return
				}
				assert.Equal(t, tt.newStatus, ticket.Status())

				if tt.checkResolved {
					assert.NotNil(t, ticket.ResolvedTime())
				}
				if tt.checkClosed {
					assert.NotNil(t, ticket.ClosedAt())
				}
				if tt.checkReopened {
					assert.Nil(t, ticket.ClosedAt())
					assert.Nil(t, ticket.ResolvedTime())
				}
			}
		})
	}
}

func TestTicket_Close(t *testing.T) {
	tests := []struct {
		name     string
		reason   string
		closedBy uint
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid close",
			reason:   "Issue resolved",
			closedBy: 1,
			wantErr:  false,
		},
		{
			name:     "empty reason",
			reason:   "",
			closedBy: 1,
			wantErr:  true,
			errMsg:   "close reason is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket, err := createTestTicket()
			require.NoError(t, err)

			_ = ticket.ChangeStatus(vo.StatusOpen, 1)

			err = ticket.Close(tt.reason, tt.closedBy)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, vo.StatusClosed, ticket.Status())
				assert.NotNil(t, ticket.ClosedAt())
			}
		})
	}
}

func TestTicket_Reopen(t *testing.T) {
	tests := []struct {
		name       string
		reason     string
		reopenedBy uint
		setupFunc  func(*Ticket) error
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "reopen closed ticket",
			reason:     "Not actually fixed",
			reopenedBy: 1,
			setupFunc: func(tk *Ticket) error {
				_ = tk.ChangeStatus(vo.StatusOpen, 1)
				return tk.Close("Done", 1)
			},
			wantErr: false,
		},
		{
			name:       "reopen resolved ticket",
			reason:     "Issue persists",
			reopenedBy: 1,
			setupFunc: func(tk *Ticket) error {
				_ = tk.ChangeStatus(vo.StatusOpen, 1)
				_ = tk.ChangeStatus(vo.StatusInProgress, 1)
				return tk.ChangeStatus(vo.StatusResolved, 1)
			},
			wantErr: false,
		},
		{
			name:       "empty reason",
			reason:     "",
			reopenedBy: 1,
			setupFunc: func(tk *Ticket) error {
				_ = tk.ChangeStatus(vo.StatusOpen, 1)
				return tk.Close("Done", 1)
			},
			wantErr: true,
			errMsg:  "reopen reason is required",
		},
		{
			name:       "cannot reopen open ticket",
			reason:     "Test",
			reopenedBy: 1,
			setupFunc: func(tk *Ticket) error {
				return tk.ChangeStatus(vo.StatusOpen, 1)
			},
			wantErr: true,
			errMsg:  "only closed or resolved tickets can be reopened",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket, err := createTestTicket()
			require.NoError(t, err)

			if tt.setupFunc != nil {
				err = tt.setupFunc(ticket)
				require.NoError(t, err)
			}

			err = ticket.Reopen(tt.reason, tt.reopenedBy)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, vo.StatusReopened, ticket.Status())
				assert.Nil(t, ticket.ClosedAt())
			}
		})
	}
}

func TestTicket_AddComment(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(*Ticket) (*Comment, error)
		wantErr       bool
		errMsg        string
		checkResponse bool
	}{
		{
			name: "add valid comment",
			setupFunc: func(tk *Ticket) (*Comment, error) {
				tk.SetID(1)
				return NewComment(1, 2, "Test comment", false)
			},
			wantErr:       false,
			checkResponse: true,
		},
		{
			name: "add internal comment",
			setupFunc: func(tk *Ticket) (*Comment, error) {
				tk.SetID(1)
				return NewComment(1, 2, "Internal note", true)
			},
			wantErr:       false,
			checkResponse: false,
		},
		{
			name: "nil comment",
			setupFunc: func(tk *Ticket) (*Comment, error) {
				return nil, nil
			},
			wantErr: true,
			errMsg:  "comment cannot be nil",
		},
		{
			name: "comment ticket ID mismatch",
			setupFunc: func(tk *Ticket) (*Comment, error) {
				tk.SetID(1)
				return NewComment(999, 2, "Test", false)
			},
			wantErr: true,
			errMsg:  "comment ticket ID mismatch",
		},
		{
			name: "creator comment does not set response time",
			setupFunc: func(tk *Ticket) (*Comment, error) {
				tk.SetID(1)
				return NewComment(1, 1, "Creator's comment", false)
			},
			wantErr:       false,
			checkResponse: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket, err := createTestTicket()
			require.NoError(t, err)

			comment, err := tt.setupFunc(ticket)
			if err != nil {
				require.NoError(t, err)
			}

			err = ticket.AddComment(comment)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Len(t, ticket.Comments(), 1)
				if tt.checkResponse {
					assert.NotNil(t, ticket.ResponseTime())
				}
			}
		})
	}
}

func TestTicket_IsOverdue(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*Ticket)
		want      bool
	}{
		{
			name: "not overdue - future SLA",
			setupFunc: func(tk *Ticket) {
			},
			want: false,
		},
		{
			name: "overdue - past SLA",
			setupFunc: func(tk *Ticket) {
				past := time.Now().Add(-1 * time.Hour)
				tk.slaDueTime = &past
			},
			want: true,
		},
		{
			name: "closed ticket not overdue",
			setupFunc: func(tk *Ticket) {
				past := time.Now().Add(-1 * time.Hour)
				tk.slaDueTime = &past
				_ = tk.ChangeStatus(vo.StatusOpen, 1)
				_ = tk.Close("Done", 1)
			},
			want: false,
		},
		{
			name: "resolved ticket not overdue",
			setupFunc: func(tk *Ticket) {
				past := time.Now().Add(-1 * time.Hour)
				tk.slaDueTime = &past
				_ = tk.ChangeStatus(vo.StatusOpen, 1)
				_ = tk.ChangeStatus(vo.StatusInProgress, 1)
				_ = tk.ChangeStatus(vo.StatusResolved, 1)
			},
			want: false,
		},
		{
			name: "nil SLA time not overdue",
			setupFunc: func(tk *Ticket) {
				tk.slaDueTime = nil
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket, err := createTestTicket()
			require.NoError(t, err)

			if tt.setupFunc != nil {
				tt.setupFunc(ticket)
			}

			assert.Equal(t, tt.want, ticket.IsOverdue())
		})
	}
}

func TestTicket_ChangePriority(t *testing.T) {
	tests := []struct {
		name        string
		newPriority vo.Priority
		changedBy   uint
		wantErr     bool
		checkSLA    bool
	}{
		{
			name:        "change to high priority",
			newPriority: vo.PriorityHigh,
			changedBy:   1,
			wantErr:     false,
			checkSLA:    true,
		},
		{
			name:        "invalid priority",
			newPriority: vo.Priority("invalid"),
			changedBy:   1,
			wantErr:     true,
		},
		{
			name:        "same priority no change",
			newPriority: vo.PriorityMedium,
			changedBy:   1,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket, err := createTestTicket()
			require.NoError(t, err)

			oldSLA := ticket.SLADueTime()

			err = ticket.ChangePriority(tt.newPriority, tt.changedBy)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.newPriority, ticket.Priority())

				if tt.checkSLA && tt.newPriority != vo.PriorityMedium {
					assert.NotEqual(t, oldSLA, ticket.SLADueTime())
				}
			}
		})
	}
}

func TestTicket_SetID(t *testing.T) {
	tests := []struct {
		name    string
		id      uint
		twice   bool
		wantErr bool
		errMsg  string
	}{
		{
			name:    "set valid ID",
			id:      1,
			wantErr: false,
		},
		{
			name:    "set zero ID",
			id:      0,
			wantErr: true,
			errMsg:  "ticket ID cannot be zero",
		},
		{
			name:    "set ID twice",
			id:      1,
			twice:   true,
			wantErr: true,
			errMsg:  "ticket ID is already set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket, err := createTestTicket()
			require.NoError(t, err)

			if tt.twice {
				err = ticket.SetID(1)
				require.NoError(t, err)
			}

			err = ticket.SetID(tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.id, ticket.ID())
			}
		})
	}
}

func TestTicket_SetNumber(t *testing.T) {
	tests := []struct {
		name    string
		number  string
		twice   bool
		wantErr bool
		errMsg  string
	}{
		{
			name:    "set valid number",
			number:  "TK-001",
			wantErr: false,
		},
		{
			name:    "set empty number",
			number:  "",
			wantErr: true,
			errMsg:  "ticket number cannot be empty",
		},
		{
			name:    "set number twice",
			number:  "TK-001",
			twice:   true,
			wantErr: true,
			errMsg:  "ticket number is already set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket, err := createTestTicket()
			require.NoError(t, err)

			if tt.twice {
				err = ticket.SetNumber("TK-000")
				require.NoError(t, err)
			}

			err = ticket.SetNumber(tt.number)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.number, ticket.Number())
			}
		})
	}
}

func TestTicket_CanBeViewedBy(t *testing.T) {
	tests := []struct {
		name      string
		userID    uint
		userRoles []string
		want      bool
	}{
		{
			name:      "admin can view",
			userID:    999,
			userRoles: []string{"admin"},
			want:      true,
		},
		{
			name:      "support agent can view",
			userID:    999,
			userRoles: []string{"support_agent"},
			want:      true,
		},
		{
			name:      "creator can view",
			userID:    1,
			userRoles: []string{"user"},
			want:      true,
		},
		{
			name:      "assignee can view",
			userID:    2,
			userRoles: []string{"user"},
			want:      true,
		},
		{
			name:      "unrelated user cannot view",
			userID:    999,
			userRoles: []string{"user"},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket, err := createTestTicket()
			require.NoError(t, err)

			assignee := uint(2)
			ticket.assigneeID = &assignee

			result := ticket.CanBeViewedBy(tt.userID, tt.userRoles)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestTicket_Validate(t *testing.T) {
	tests := []struct {
		name    string
		ticket  func() *Ticket
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid ticket",
			ticket: func() *Ticket {
				tk, _ := createTestTicket()
				return tk
			},
			wantErr: false,
		},
		{
			name: "empty title",
			ticket: func() *Ticket {
				tk, _ := createTestTicket()
				tk.title = ""
				return tk
			},
			wantErr: true,
			errMsg:  "title is required",
		},
		{
			name: "empty description",
			ticket: func() *Ticket {
				tk, _ := createTestTicket()
				tk.description = ""
				return tk
			},
			wantErr: true,
			errMsg:  "description is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket := tt.ticket()
			err := ticket.Validate()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestReconstructTicket(t *testing.T) {
	now := time.Now()
	assigneeID := uint(2)

	tests := []struct {
		name     string
		id       uint
		number   string
		title    string
		category vo.Category
		priority vo.Priority
		status   vo.TicketStatus
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid reconstruction",
			id:       1,
			number:   "TK-001",
			title:    "Test",
			category: vo.CategoryTechnical,
			priority: vo.PriorityMedium,
			status:   vo.StatusOpen,
			wantErr:  false,
		},
		{
			name:     "zero ID",
			id:       0,
			number:   "TK-001",
			title:    "Test",
			category: vo.CategoryTechnical,
			priority: vo.PriorityMedium,
			status:   vo.StatusOpen,
			wantErr:  true,
			errMsg:   "ticket ID cannot be zero",
		},
		{
			name:     "empty number",
			id:       1,
			number:   "",
			title:    "Test",
			category: vo.CategoryTechnical,
			priority: vo.PriorityMedium,
			status:   vo.StatusOpen,
			wantErr:  true,
			errMsg:   "ticket number is required",
		},
		{
			name:     "empty title",
			id:       1,
			number:   "TK-001",
			title:    "",
			category: vo.CategoryTechnical,
			priority: vo.PriorityMedium,
			status:   vo.StatusOpen,
			wantErr:  true,
			errMsg:   "title is required",
		},
		{
			name:     "invalid category",
			id:       1,
			number:   "TK-001",
			title:    "Test",
			category: vo.Category("invalid"),
			priority: vo.PriorityMedium,
			status:   vo.StatusOpen,
			wantErr:  true,
			errMsg:   "invalid category",
		},
		{
			name:     "invalid priority",
			id:       1,
			number:   "TK-001",
			title:    "Test",
			category: vo.CategoryTechnical,
			priority: vo.Priority("invalid"),
			status:   vo.StatusOpen,
			wantErr:  true,
			errMsg:   "invalid priority",
		},
		{
			name:     "invalid status",
			id:       1,
			number:   "TK-001",
			title:    "Test",
			category: vo.CategoryTechnical,
			priority: vo.PriorityMedium,
			status:   vo.TicketStatus("invalid"),
			wantErr:  true,
			errMsg:   "invalid status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket, err := ReconstructTicket(
				tt.id,
				tt.number,
				tt.title,
				"Description",
				tt.category,
				tt.priority,
				tt.status,
				1,
				&assigneeID,
				[]string{"test"},
				map[string]interface{}{"key": "value"},
				&now,
				&now,
				&now,
				1,
				now,
				now,
				&now,
			)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, ticket)
				assert.Equal(t, tt.id, ticket.ID())
				assert.Equal(t, tt.number, ticket.Number())
			}
		})
	}
}

func createTestTicket() (*Ticket, error) {
	return NewTicket(
		"Test Ticket",
		"Test Description",
		vo.CategoryTechnical,
		vo.PriorityMedium,
		1,
	)
}

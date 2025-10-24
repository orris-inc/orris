package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"orris/internal/domain/ticket"
	vo "orris/internal/domain/ticket/value_objects"
	"orris/internal/domain/user"
	uservo "orris/internal/domain/user/value_objects"
)

type mockUser struct {
	id                uint
	email             string
	status            uservo.Status
	canPerformActions bool
}

func (m *mockUser) ID() uint {
	return m.id
}

func (m *mockUser) Email() string {
	return m.email
}

func (m *mockUser) Status() uservo.Status {
	return m.status
}

func (m *mockUser) CanPerformActions() bool {
	return m.canPerformActions
}

func TestAssignTicketUseCase_Execute_Success(t *testing.T) {
	ticketID := uint(1)
	assigneeID := uint(10)
	assignedBy := uint(5)

	existingTicket, err := ticket.ReconstructTicket(
		ticketID,
		"TKT-001",
		"Test ticket",
		"Test description",
		vo.CategoryTechnical,
		vo.PriorityMedium,
		vo.StatusNew,
		2,
		nil,
		[]string{},
		nil,
		nil,
		nil,
		nil,
		1,
		time.Now().Add(-1*time.Hour),
		time.Now().Add(-1*time.Hour),
		nil,
	)
	require.NoError(t, err)

	mockUserRepo := &mockUserRepository{
		GetByIDFunc: func(ctx context.Context, id uint) (*user.User, error) {
			if id == assigneeID {
				userEmail, _ := uservo.NewEmail("agent@example.com")
				userName, _ := uservo.NewName("Agent")
				u, _ := user.ReconstructUser(assigneeID, userEmail, userName, uservo.StatusActive, time.Now(), time.Now(), 1)
				return u, nil
			}
			return nil, errors.New("user not found")
		},
	}

	var updatedTicket *ticket.Ticket
	mockTicketRepo := &mockTicketRepository{
		GetByIDFunc: func(ctx context.Context, id uint) (*ticket.Ticket, error) {
			if id == ticketID {
				return existingTicket, nil
			}
			return nil, errors.New("ticket not found")
		},
		UpdateFunc: func(ctx context.Context, t *ticket.Ticket) error {
			updatedTicket = t
			return nil
		},
	}

	mockDispatcher := &mockEventDispatcher{}
	mockLog := &mockLogger{}

	useCase := NewAssignTicketUseCase(mockTicketRepo, mockUserRepo, mockDispatcher, mockLog)
	cmd := AssignTicketCommand{
		TicketID:   ticketID,
		AssigneeID: assigneeID,
		AssignedBy: assignedBy,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, ticketID, result.TicketID)
	assert.Equal(t, assigneeID, result.AssigneeID)
	assert.NotEmpty(t, result.Status)
	assert.NotEmpty(t, result.UpdatedAt)

	require.NotNil(t, updatedTicket)
	assert.NotNil(t, updatedTicket.AssigneeID())
	assert.Equal(t, assigneeID, *updatedTicket.AssigneeID())
}

func TestAssignTicketUseCase_Execute_ValidationErrors(t *testing.T) {
	tests := []struct {
		name          string
		command       AssignTicketCommand
		expectedError string
	}{
		{
			name: "missing ticket ID",
			command: AssignTicketCommand{
				TicketID:   0,
				AssigneeID: 10,
				AssignedBy: 5,
			},
			expectedError: "ticket ID is required",
		},
		{
			name: "missing assignee ID",
			command: AssignTicketCommand{
				TicketID:   1,
				AssigneeID: 0,
				AssignedBy: 5,
			},
			expectedError: "assignee ID is required",
		},
		{
			name: "missing assigned by",
			command: AssignTicketCommand{
				TicketID:   1,
				AssigneeID: 10,
				AssignedBy: 0,
			},
			expectedError: "assigned by ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTicketRepo := &mockTicketRepository{}
			mockUserRepo := &mockUserRepository{}
			mockDispatcher := &mockEventDispatcher{}
			mockLog := &mockLogger{}

			useCase := NewAssignTicketUseCase(mockTicketRepo, mockUserRepo, mockDispatcher, mockLog)
			result, err := useCase.Execute(context.Background(), tt.command)

			require.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestAssignTicketUseCase_Execute_AssigneeNotFound(t *testing.T) {
	mockUserRepo := &mockUserRepository{
		GetByIDFunc: func(ctx context.Context, id uint) (*user.User, error) {
			return nil, errors.New("user not found")
		},
	}

	mockTicketRepo := &mockTicketRepository{}
	mockDispatcher := &mockEventDispatcher{}
	mockLog := &mockLogger{}

	useCase := NewAssignTicketUseCase(mockTicketRepo, mockUserRepo, mockDispatcher, mockLog)
	cmd := AssignTicketCommand{
		TicketID:   1,
		AssigneeID: 10,
		AssignedBy: 5,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "assignee not found")
}

func TestAssignTicketUseCase_Execute_AssigneeNotActive(t *testing.T) {
	mockUserRepo := &mockUserRepository{
		GetByIDFunc: func(ctx context.Context, id uint) (*user.User, error) {
			userEmail, _ := uservo.NewEmail("inactive@example.com")
			userName, _ := uservo.NewName("Inactive User")
			u, _ := user.ReconstructUser(10, userEmail, userName, uservo.StatusInactive, time.Now(), time.Now(), 1)
			return u, nil
		},
	}

	mockTicketRepo := &mockTicketRepository{}
	mockDispatcher := &mockEventDispatcher{}
	mockLog := &mockLogger{}

	useCase := NewAssignTicketUseCase(mockTicketRepo, mockUserRepo, mockDispatcher, mockLog)
	cmd := AssignTicketCommand{
		TicketID:   1,
		AssigneeID: 10,
		AssignedBy: 5,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "assignee is not active")
}

func TestAssignTicketUseCase_Execute_TicketNotFound(t *testing.T) {
	mockUserRepo := &mockUserRepository{
		GetByIDFunc: func(ctx context.Context, id uint) (*user.User, error) {
			userEmail, _ := uservo.NewEmail("agent@example.com")
			userName, _ := uservo.NewName("Agent")
			u, _ := user.ReconstructUser(10, userEmail, userName, uservo.StatusActive, time.Now(), time.Now(), 1)
			return u, nil
		},
	}

	mockTicketRepo := &mockTicketRepository{
		GetByIDFunc: func(ctx context.Context, id uint) (*ticket.Ticket, error) {
			return nil, errors.New("ticket not found")
		},
	}

	mockDispatcher := &mockEventDispatcher{}
	mockLog := &mockLogger{}

	useCase := NewAssignTicketUseCase(mockTicketRepo, mockUserRepo, mockDispatcher, mockLog)
	cmd := AssignTicketCommand{
		TicketID:   1,
		AssigneeID: 10,
		AssignedBy: 5,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "ticket not found")
}

func TestAssignTicketUseCase_Execute_UpdateFailed(t *testing.T) {
	existingTicket, err := ticket.ReconstructTicket(
		1,
		"TKT-001",
		"Test ticket",
		"Test description",
		vo.CategoryTechnical,
		vo.PriorityMedium,
		vo.StatusNew,
		2,
		nil,
		[]string{},
		nil,
		nil,
		nil,
		nil,
		1,
		time.Now().Add(-1*time.Hour),
		time.Now().Add(-1*time.Hour),
		nil,
	)
	require.NoError(t, err)

	mockUserRepo := &mockUserRepository{
		GetByIDFunc: func(ctx context.Context, id uint) (*user.User, error) {
			userEmail, _ := uservo.NewEmail("agent@example.com")
			userName, _ := uservo.NewName("Agent")
			u, _ := user.ReconstructUser(10, userEmail, userName, uservo.StatusActive, time.Now(), time.Now(), 1)
			return u, nil
		},
	}

	mockTicketRepo := &mockTicketRepository{
		GetByIDFunc: func(ctx context.Context, id uint) (*ticket.Ticket, error) {
			return existingTicket, nil
		},
		UpdateFunc: func(ctx context.Context, t *ticket.Ticket) error {
			return errors.New("database error")
		},
	}

	mockDispatcher := &mockEventDispatcher{}
	mockLog := &mockLogger{}

	useCase := NewAssignTicketUseCase(mockTicketRepo, mockUserRepo, mockDispatcher, mockLog)
	cmd := AssignTicketCommand{
		TicketID:   1,
		AssigneeID: 10,
		AssignedBy: 5,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to update ticket")
}

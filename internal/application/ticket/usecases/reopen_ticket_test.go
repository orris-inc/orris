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
)

func TestReopenTicketUseCase_Execute_Success(t *testing.T) {
	tests := []struct {
		name      string
		userID    uint
		userRoles []string
	}{
		{
			name:      "admin can reopen",
			userID:    99,
			userRoles: []string{"admin"},
		},
		{
			name:      "support agent can reopen",
			userID:    88,
			userRoles: []string{"support_agent"},
		},
		{
			name:      "creator can reopen",
			userID:    10,
			userRoles: []string{"user"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticketID := uint(1)
			creatorID := uint(10)
			reason := "Customer requested reopen"
			closedAt := time.Now().Add(-1 * time.Hour)

			existingTicket, err := ticket.ReconstructTicket(
				ticketID,
				"TKT-001",
				"Test ticket",
				"Test description",
				vo.CategoryTechnical,
				vo.PriorityMedium,
				vo.StatusClosed,
				creatorID,
				nil,
				[]string{},
				nil,
				nil,
				nil,
				nil,
				1,
				time.Now().Add(-2*time.Hour),
				time.Now().Add(-1*time.Hour),
				&closedAt,
			)
			require.NoError(t, err)

			mockTicketRepo := &mockTicketRepository{
				GetByIDFunc: func(ctx context.Context, id uint) (*ticket.Ticket, error) {
					return existingTicket, nil
				},
				UpdateFunc: func(ctx context.Context, t *ticket.Ticket) error {
					return nil
				},
			}

			mockDispatcher := &mockEventDispatcher{}
			mockLog := &mockLogger{}

			useCase := NewReopenTicketUseCase(mockTicketRepo, mockDispatcher, mockLog)
			cmd := ReopenTicketCommand{
				TicketID:   ticketID,
				Reason:     reason,
				ReopenedBy: tt.userID,
				UserRoles:  tt.userRoles,
			}

			result, err := useCase.Execute(context.Background(), cmd)

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, ticketID, result.TicketID)
			assert.Equal(t, vo.StatusReopened.String(), result.Status)
			assert.Equal(t, reason, result.Reason)
		})
	}
}

func TestReopenTicketUseCase_Execute_ValidationErrors(t *testing.T) {
	tests := []struct {
		name          string
		command       ReopenTicketCommand
		expectedError string
	}{
		{
			name: "missing ticket ID",
			command: ReopenTicketCommand{
				TicketID:   0,
				Reason:     "Need to reopen",
				ReopenedBy: 5,
			},
			expectedError: "ticket ID is required",
		},
		{
			name: "missing reason",
			command: ReopenTicketCommand{
				TicketID:   1,
				Reason:     "",
				ReopenedBy: 5,
			},
			expectedError: "reopen reason is required",
		},
		{
			name: "missing reopened by",
			command: ReopenTicketCommand{
				TicketID:   1,
				Reason:     "Need to reopen",
				ReopenedBy: 0,
			},
			expectedError: "reopened by user ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTicketRepo := &mockTicketRepository{}
			mockDispatcher := &mockEventDispatcher{}
			mockLog := &mockLogger{}

			useCase := NewReopenTicketUseCase(mockTicketRepo, mockDispatcher, mockLog)
			result, err := useCase.Execute(context.Background(), tt.command)

			require.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestReopenTicketUseCase_Execute_PermissionDenied(t *testing.T) {
	ticketID := uint(1)
	creatorID := uint(10)
	otherUserID := uint(99)
	closedAt := time.Now().Add(-1 * time.Hour)

	existingTicket, err := ticket.ReconstructTicket(
		ticketID,
		"TKT-001",
		"Test ticket",
		"Test description",
		vo.CategoryTechnical,
		vo.PriorityMedium,
		vo.StatusClosed,
		creatorID,
		nil,
		[]string{},
		nil,
		nil,
		nil,
		nil,
		1,
		time.Now().Add(-2*time.Hour),
		time.Now().Add(-1*time.Hour),
		&closedAt,
	)
	require.NoError(t, err)

	mockTicketRepo := &mockTicketRepository{
		GetByIDFunc: func(ctx context.Context, id uint) (*ticket.Ticket, error) {
			return existingTicket, nil
		},
	}

	mockDispatcher := &mockEventDispatcher{}
	mockLog := &mockLogger{}

	useCase := NewReopenTicketUseCase(mockTicketRepo, mockDispatcher, mockLog)
	cmd := ReopenTicketCommand{
		TicketID:   ticketID,
		Reason:     "Unauthorized reopen",
		ReopenedBy: otherUserID,
		UserRoles:  []string{"user"},
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "permission")
}

func TestReopenTicketUseCase_Execute_TicketNotFound(t *testing.T) {
	mockTicketRepo := &mockTicketRepository{
		GetByIDFunc: func(ctx context.Context, id uint) (*ticket.Ticket, error) {
			return nil, errors.New("not found")
		},
	}

	mockDispatcher := &mockEventDispatcher{}
	mockLog := &mockLogger{}

	useCase := NewReopenTicketUseCase(mockTicketRepo, mockDispatcher, mockLog)
	cmd := ReopenTicketCommand{
		TicketID:   1,
		Reason:     "Need to reopen",
		ReopenedBy: 5,
		UserRoles:  []string{"admin"},
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
}

func TestReopenTicketUseCase_Execute_InvalidState(t *testing.T) {
	ticketID := uint(1)
	creatorID := uint(10)

	existingTicket, err := ticket.ReconstructTicket(
		ticketID,
		"TKT-001",
		"Test ticket",
		"Test description",
		vo.CategoryTechnical,
		vo.PriorityMedium,
		vo.StatusOpen,
		creatorID,
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

	mockTicketRepo := &mockTicketRepository{
		GetByIDFunc: func(ctx context.Context, id uint) (*ticket.Ticket, error) {
			return existingTicket, nil
		},
	}

	mockDispatcher := &mockEventDispatcher{}
	mockLog := &mockLogger{}

	useCase := NewReopenTicketUseCase(mockTicketRepo, mockDispatcher, mockLog)
	cmd := ReopenTicketCommand{
		TicketID:   ticketID,
		Reason:     "Try to reopen open ticket",
		ReopenedBy: creatorID,
		UserRoles:  []string{"user"},
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, result)
}

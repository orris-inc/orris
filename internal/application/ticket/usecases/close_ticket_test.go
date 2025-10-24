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

func TestCloseTicketUseCase_Execute_Success(t *testing.T) {
	ticketID := uint(1)
	closedBy := uint(5)
	reason := "Issue resolved"

	existingTicket, err := ticket.ReconstructTicket(
		ticketID,
		"TKT-001",
		"Test ticket",
		"Test description",
		vo.CategoryTechnical,
		vo.PriorityMedium,
		vo.StatusResolved,
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

	useCase := NewCloseTicketUseCase(mockTicketRepo, mockDispatcher, mockLog)
	cmd := CloseTicketCommand{
		TicketID: ticketID,
		Reason:   reason,
		ClosedBy: closedBy,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, ticketID, result.TicketID)
	assert.Equal(t, vo.StatusClosed.String(), result.Status)
	assert.Equal(t, reason, result.Reason)
	assert.NotEmpty(t, result.ClosedAt)
}

func TestCloseTicketUseCase_Execute_ValidationErrors(t *testing.T) {
	tests := []struct {
		name          string
		command       CloseTicketCommand
		expectedError string
	}{
		{
			name: "missing ticket ID",
			command: CloseTicketCommand{
				TicketID: 0,
				Reason:   "Resolved",
				ClosedBy: 5,
			},
			expectedError: "ticket ID is required",
		},
		{
			name: "missing reason",
			command: CloseTicketCommand{
				TicketID: 1,
				Reason:   "",
				ClosedBy: 5,
			},
			expectedError: "close reason is required",
		},
		{
			name: "missing closed by",
			command: CloseTicketCommand{
				TicketID: 1,
				Reason:   "Resolved",
				ClosedBy: 0,
			},
			expectedError: "closed by user ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTicketRepo := &mockTicketRepository{}
			mockDispatcher := &mockEventDispatcher{}
			mockLog := &mockLogger{}

			useCase := NewCloseTicketUseCase(mockTicketRepo, mockDispatcher, mockLog)
			result, err := useCase.Execute(context.Background(), tt.command)

			require.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestCloseTicketUseCase_Execute_TicketNotFound(t *testing.T) {
	mockTicketRepo := &mockTicketRepository{
		GetByIDFunc: func(ctx context.Context, id uint) (*ticket.Ticket, error) {
			return nil, errors.New("not found")
		},
	}

	mockDispatcher := &mockEventDispatcher{}
	mockLog := &mockLogger{}

	useCase := NewCloseTicketUseCase(mockTicketRepo, mockDispatcher, mockLog)
	cmd := CloseTicketCommand{
		TicketID: 1,
		Reason:   "Resolved",
		ClosedBy: 5,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
}

func TestCloseTicketUseCase_Execute_AlreadyClosed(t *testing.T) {
	ticketID := uint(1)
	closedBy := uint(5)
	closedAt := time.Now().Add(-1 * time.Hour)

	existingTicket, err := ticket.ReconstructTicket(
		ticketID,
		"TKT-001",
		"Test ticket",
		"Test description",
		vo.CategoryTechnical,
		vo.PriorityMedium,
		vo.StatusClosed,
		2,
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

	useCase := NewCloseTicketUseCase(mockTicketRepo, mockDispatcher, mockLog)
	cmd := CloseTicketCommand{
		TicketID: ticketID,
		Reason:   "Double close",
		ClosedBy: closedBy,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, vo.StatusClosed.String(), result.Status)
}

func TestCloseTicketUseCase_Execute_UpdateFailed(t *testing.T) {
	ticketID := uint(1)

	existingTicket, err := ticket.ReconstructTicket(
		ticketID,
		"TKT-001",
		"Test ticket",
		"Test description",
		vo.CategoryTechnical,
		vo.PriorityMedium,
		vo.StatusResolved,
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

	useCase := NewCloseTicketUseCase(mockTicketRepo, mockDispatcher, mockLog)
	cmd := CloseTicketCommand{
		TicketID: ticketID,
		Reason:   "Resolved",
		ClosedBy: 5,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to update ticket")
}

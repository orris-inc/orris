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

func TestChangeStatusUseCase_Execute_Success(t *testing.T) {
	tests := []struct {
		name      string
		oldStatus vo.TicketStatus
		newStatus vo.TicketStatus
	}{
		{
			name:      "new to open",
			oldStatus: vo.StatusNew,
			newStatus: vo.StatusOpen,
		},
		{
			name:      "open to in_progress",
			oldStatus: vo.StatusOpen,
			newStatus: vo.StatusInProgress,
		},
		{
			name:      "in_progress to resolved",
			oldStatus: vo.StatusInProgress,
			newStatus: vo.StatusResolved,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticketID := uint(1)
			changedBy := uint(5)

			existingTicket, err := ticket.ReconstructTicket(
				ticketID,
				"TKT-001",
				"Test ticket",
				"Test description",
				vo.CategoryTechnical,
				vo.PriorityMedium,
				tt.oldStatus,
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

			useCase := NewChangeStatusUseCase(mockTicketRepo, mockDispatcher, mockLog)
			cmd := ChangeStatusCommand{
				TicketID:  ticketID,
				NewStatus: tt.newStatus,
				ChangedBy: changedBy,
			}

			result, err := useCase.Execute(context.Background(), cmd)

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, ticketID, result.TicketID)
			assert.Equal(t, tt.oldStatus.String(), result.OldStatus)
			assert.Equal(t, tt.newStatus.String(), result.NewStatus)
		})
	}
}

func TestChangeStatusUseCase_Execute_ValidationErrors(t *testing.T) {
	tests := []struct {
		name          string
		command       ChangeStatusCommand
		expectedError string
	}{
		{
			name: "missing ticket ID",
			command: ChangeStatusCommand{
				TicketID:  0,
				NewStatus: vo.StatusOpen,
				ChangedBy: 5,
			},
			expectedError: "ticket ID is required",
		},
		{
			name: "invalid status",
			command: ChangeStatusCommand{
				TicketID:  1,
				NewStatus: vo.TicketStatus("invalid"),
				ChangedBy: 5,
			},
			expectedError: "invalid status",
		},
		{
			name: "missing changed by",
			command: ChangeStatusCommand{
				TicketID:  1,
				NewStatus: vo.StatusOpen,
				ChangedBy: 0,
			},
			expectedError: "changed by user ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTicketRepo := &mockTicketRepository{}
			mockDispatcher := &mockEventDispatcher{}
			mockLog := &mockLogger{}

			useCase := NewChangeStatusUseCase(mockTicketRepo, mockDispatcher, mockLog)
			result, err := useCase.Execute(context.Background(), tt.command)

			require.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestChangeStatusUseCase_Execute_IllegalTransition(t *testing.T) {
	ticketID := uint(1)
	changedBy := uint(5)

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

	useCase := NewChangeStatusUseCase(mockTicketRepo, mockDispatcher, mockLog)
	cmd := ChangeStatusCommand{
		TicketID:  ticketID,
		NewStatus: vo.StatusOpen,
		ChangedBy: changedBy,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "cannot transition")
}

func TestChangeStatusUseCase_Execute_TicketNotFound(t *testing.T) {
	mockTicketRepo := &mockTicketRepository{
		GetByIDFunc: func(ctx context.Context, id uint) (*ticket.Ticket, error) {
			return nil, errors.New("not found")
		},
	}

	mockDispatcher := &mockEventDispatcher{}
	mockLog := &mockLogger{}

	useCase := NewChangeStatusUseCase(mockTicketRepo, mockDispatcher, mockLog)
	cmd := ChangeStatusCommand{
		TicketID:  1,
		NewStatus: vo.StatusOpen,
		ChangedBy: 5,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
}

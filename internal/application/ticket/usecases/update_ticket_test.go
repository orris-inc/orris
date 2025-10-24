package usecases

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"orris/internal/domain/ticket"
	vo "orris/internal/domain/ticket/value_objects"
)

func TestUpdateTicketUseCase_Execute_Success(t *testing.T) {
	ticketID := uint(1)
	updatedBy := uint(5)
	newTitle := "Updated title"
	newDesc := "Updated description"

	existingTicket, err := ticket.ReconstructTicket(
		ticketID,
		"TKT-001",
		"Original title",
		"Original description",
		vo.CategoryTechnical,
		vo.PriorityMedium,
		vo.StatusOpen,
		updatedBy,
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

	useCase := NewUpdateTicketUseCase(mockTicketRepo, mockDispatcher, mockLog)
	cmd := UpdateTicketCommand{
		TicketID:    ticketID,
		Title:       &newTitle,
		Description: &newDesc,
		UpdatedBy:   updatedBy,
		UserRoles:   []string{"user"},
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, ticketID, result.TicketID)
}

func TestUpdateTicketUseCase_Execute_AsAdmin(t *testing.T) {
	ticketID := uint(1)
	creatorID := uint(10)
	adminID := uint(99)

	existingTicket, err := ticket.ReconstructTicket(
		ticketID,
		"TKT-001",
		"Original title",
		"Original description",
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
		UpdateFunc: func(ctx context.Context, t *ticket.Ticket) error {
			return nil
		},
	}

	mockDispatcher := &mockEventDispatcher{}
	mockLog := &mockLogger{}

	useCase := NewUpdateTicketUseCase(mockTicketRepo, mockDispatcher, mockLog)
	newTitle := "Admin updated"
	cmd := UpdateTicketCommand{
		TicketID:  ticketID,
		Title:     &newTitle,
		UpdatedBy: adminID,
		UserRoles: []string{"admin"},
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestUpdateTicketUseCase_Execute_ValidationErrors(t *testing.T) {
	tests := []struct {
		name          string
		command       UpdateTicketCommand
		expectedError string
	}{
		{
			name: "missing ticket ID",
			command: UpdateTicketCommand{
				TicketID:  0,
				UpdatedBy: 5,
			},
			expectedError: "ticket ID is required",
		},
		{
			name: "missing updated by",
			command: UpdateTicketCommand{
				TicketID:  1,
				UpdatedBy: 0,
			},
			expectedError: "updated by user ID is required",
		},
		{
			name: "empty title",
			command: UpdateTicketCommand{
				TicketID:  1,
				Title:     stringPtr(""),
				UpdatedBy: 5,
			},
			expectedError: "title cannot be empty",
		},
		{
			name: "title too long",
			command: UpdateTicketCommand{
				TicketID:  1,
				Title:     stringPtr(string(make([]byte, 201))),
				UpdatedBy: 5,
			},
			expectedError: "title exceeds maximum length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTicketRepo := &mockTicketRepository{}
			mockDispatcher := &mockEventDispatcher{}
			mockLog := &mockLogger{}

			useCase := NewUpdateTicketUseCase(mockTicketRepo, mockDispatcher, mockLog)
			result, err := useCase.Execute(context.Background(), tt.command)

			require.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestUpdateTicketUseCase_Execute_Forbidden(t *testing.T) {
	ticketID := uint(1)
	creatorID := uint(10)
	otherUserID := uint(99)

	existingTicket, err := ticket.ReconstructTicket(
		ticketID,
		"TKT-001",
		"Original title",
		"Original description",
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

	useCase := NewUpdateTicketUseCase(mockTicketRepo, mockDispatcher, mockLog)
	newTitle := "Hacker update"
	cmd := UpdateTicketCommand{
		TicketID:  ticketID,
		Title:     &newTitle,
		UpdatedBy: otherUserID,
		UserRoles: []string{"user"},
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not authorized")
}

func stringPtr(s string) *string {
	return &s
}

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

func TestGetTicketUseCase_Execute_Success(t *testing.T) {
	ticketID := uint(1)
	creatorID := uint(10)

	existingTicket, err := ticket.ReconstructTicket(
		ticketID,
		"TKT-001",
		"Test ticket",
		"Test description",
		vo.CategoryTechnical,
		vo.PriorityHigh,
		vo.StatusOpen,
		creatorID,
		nil,
		[]string{"bug", "urgent"},
		map[string]interface{}{"env": "production"},
		nil,
		nil,
		nil,
		1,
		time.Now().Add(-1*time.Hour),
		time.Now().Add(-1*time.Hour),
		nil,
	)
	require.NoError(t, err)

	comment1, err := ticket.ReconstructComment(
		1,
		ticketID,
		creatorID,
		"First comment",
		false,
		time.Now().Add(-30*time.Minute),
		time.Now().Add(-30*time.Minute),
	)
	require.NoError(t, err)

	comment2, err := ticket.ReconstructComment(
		2,
		ticketID,
		99,
		"Internal note",
		true,
		time.Now().Add(-20*time.Minute),
		time.Now().Add(-20*time.Minute),
	)
	require.NoError(t, err)

	mockTicketRepo := &mockTicketRepository{
		GetByIDFunc: func(ctx context.Context, id uint) (*ticket.Ticket, error) {
			return existingTicket, nil
		},
	}

	mockCommentRepo := &mockCommentRepository{
		GetByTicketIDFunc: func(ctx context.Context, tID uint) ([]*ticket.Comment, error) {
			return []*ticket.Comment{comment1, comment2}, nil
		},
	}

	mockLog := &mockLogger{}

	useCase := NewGetTicketUseCase(mockTicketRepo, mockCommentRepo, mockLog)

	t.Run("admin can see all comments", func(t *testing.T) {
		query := GetTicketQuery{
			TicketID:  ticketID,
			UserID:    99,
			UserRoles: []string{"admin"},
		}

		result, err := useCase.Execute(context.Background(), query)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, ticketID, result.ID)
		assert.Equal(t, "TKT-001", result.Number)
		assert.Equal(t, "Test ticket", result.Title)
		assert.Equal(t, 2, len(result.Comments))
	})

	t.Run("user cannot see internal comments", func(t *testing.T) {
		query := GetTicketQuery{
			TicketID:  ticketID,
			UserID:    creatorID,
			UserRoles: []string{"user"},
		}

		result, err := useCase.Execute(context.Background(), query)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, len(result.Comments))
		assert.Equal(t, "First comment", result.Comments[0].Content)
		assert.False(t, result.Comments[0].IsInternal)
	})
}

func TestGetTicketUseCase_Execute_TicketNotFound(t *testing.T) {
	mockTicketRepo := &mockTicketRepository{
		GetByIDFunc: func(ctx context.Context, id uint) (*ticket.Ticket, error) {
			return nil, errors.New("ticket not found")
		},
	}

	mockCommentRepo := &mockCommentRepository{}
	mockLog := &mockLogger{}

	useCase := NewGetTicketUseCase(mockTicketRepo, mockCommentRepo, mockLog)
	query := GetTicketQuery{
		TicketID:  1,
		UserID:    10,
		UserRoles: []string{"user"},
	}

	result, err := useCase.Execute(context.Background(), query)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to load ticket")
}

func TestGetTicketUseCase_Execute_PermissionDenied(t *testing.T) {
	ticketID := uint(1)
	creatorID := uint(10)
	otherUserID := uint(99)

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

	mockCommentRepo := &mockCommentRepository{}
	mockLog := &mockLogger{}

	useCase := NewGetTicketUseCase(mockTicketRepo, mockCommentRepo, mockLog)
	query := GetTicketQuery{
		TicketID:  ticketID,
		UserID:    otherUserID,
		UserRoles: []string{"user"},
	}

	result, err := useCase.Execute(context.Background(), query)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestGetTicketUseCase_Execute_LoadCommentsFailed(t *testing.T) {
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

	mockCommentRepo := &mockCommentRepository{
		GetByTicketIDFunc: func(ctx context.Context, tID uint) ([]*ticket.Comment, error) {
			return nil, errors.New("database error")
		},
	}

	mockLog := &mockLogger{}

	useCase := NewGetTicketUseCase(mockTicketRepo, mockCommentRepo, mockLog)
	query := GetTicketQuery{
		TicketID:  ticketID,
		UserID:    creatorID,
		UserRoles: []string{"user"},
	}

	result, err := useCase.Execute(context.Background(), query)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to load comments")
}

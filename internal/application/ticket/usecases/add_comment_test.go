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

func TestAddCommentUseCase_Execute_Success(t *testing.T) {
	tests := []struct {
		name       string
		userID     uint
		userRoles  []string
		isInternal bool
		shouldPass bool
	}{
		{
			name:       "user adds public comment",
			userID:     10,
			userRoles:  []string{"user"},
			isInternal: false,
			shouldPass: true,
		},
		{
			name:       "admin adds internal comment",
			userID:     99,
			userRoles:  []string{"admin"},
			isInternal: true,
			shouldPass: true,
		},
		{
			name:       "support agent adds internal comment",
			userID:     88,
			userRoles:  []string{"support_agent"},
			isInternal: true,
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
				UpdateFunc: func(ctx context.Context, t *ticket.Ticket) error {
					return nil
				},
			}

			var savedComment *ticket.Comment
			mockCommentRepo := &mockCommentRepository{
				SaveFunc: func(ctx context.Context, comment *ticket.Comment) error {
					err := comment.SetID(100)
					if err != nil {
						return err
					}
					savedComment = comment
					return nil
				},
			}

			mockDispatcher := &mockEventDispatcher{}
			mockLog := &mockLogger{}

			useCase := NewAddCommentUseCase(mockTicketRepo, mockCommentRepo, mockDispatcher, mockLog)
			cmd := AddCommentCommand{
				TicketID:   ticketID,
				UserID:     tt.userID,
				UserRoles:  tt.userRoles,
				Content:    "This is a test comment",
				IsInternal: tt.isInternal,
			}

			result, err := useCase.Execute(context.Background(), cmd)

			if tt.shouldPass {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, uint(100), result.CommentID)
				assert.NotZero(t, result.CreatedAt)
				require.NotNil(t, savedComment)
				assert.Equal(t, tt.isInternal, savedComment.IsInternal())
			}
		})
	}
}

func TestAddCommentUseCase_Execute_UserCannotAddInternalComment(t *testing.T) {
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

	mockCommentRepo := &mockCommentRepository{}
	mockDispatcher := &mockEventDispatcher{}
	mockLog := &mockLogger{}

	useCase := NewAddCommentUseCase(mockTicketRepo, mockCommentRepo, mockDispatcher, mockLog)
	cmd := AddCommentCommand{
		TicketID:   ticketID,
		UserID:     creatorID,
		UserRoles:  []string{"user"},
		Content:    "Trying to add internal comment",
		IsInternal: true,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestAddCommentUseCase_Execute_TicketNotFound(t *testing.T) {
	mockTicketRepo := &mockTicketRepository{
		GetByIDFunc: func(ctx context.Context, id uint) (*ticket.Ticket, error) {
			return nil, errors.New("ticket not found")
		},
	}

	mockCommentRepo := &mockCommentRepository{}
	mockDispatcher := &mockEventDispatcher{}
	mockLog := &mockLogger{}

	useCase := NewAddCommentUseCase(mockTicketRepo, mockCommentRepo, mockDispatcher, mockLog)
	cmd := AddCommentCommand{
		TicketID:   1,
		UserID:     10,
		UserRoles:  []string{"user"},
		Content:    "Test comment",
		IsInternal: false,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to load ticket")
}

func TestAddCommentUseCase_Execute_UserCannotViewTicket(t *testing.T) {
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
	mockDispatcher := &mockEventDispatcher{}
	mockLog := &mockLogger{}

	useCase := NewAddCommentUseCase(mockTicketRepo, mockCommentRepo, mockDispatcher, mockLog)
	cmd := AddCommentCommand{
		TicketID:   ticketID,
		UserID:     otherUserID,
		UserRoles:  []string{"user"},
		Content:    "Unauthorized comment",
		IsInternal: false,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestAddCommentUseCase_Execute_SaveCommentFailed(t *testing.T) {
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
		SaveFunc: func(ctx context.Context, comment *ticket.Comment) error {
			return errors.New("database error")
		},
	}

	mockDispatcher := &mockEventDispatcher{}
	mockLog := &mockLogger{}

	useCase := NewAddCommentUseCase(mockTicketRepo, mockCommentRepo, mockDispatcher, mockLog)
	cmd := AddCommentCommand{
		TicketID:   ticketID,
		UserID:     creatorID,
		UserRoles:  []string{"user"},
		Content:    "Test comment",
		IsInternal: false,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to save comment")
}

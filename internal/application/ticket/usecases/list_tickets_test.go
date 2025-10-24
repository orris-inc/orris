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

func TestListTicketsUseCase_Execute_Success(t *testing.T) {
	ticket1, err := ticket.ReconstructTicket(
		1,
		"TKT-001",
		"First ticket",
		"Description 1",
		vo.CategoryTechnical,
		vo.PriorityHigh,
		vo.StatusOpen,
		10,
		nil,
		[]string{},
		nil,
		nil,
		nil,
		nil,
		1,
		time.Now().Add(-2*time.Hour),
		time.Now().Add(-1*time.Hour),
		nil,
	)
	require.NoError(t, err)

	ticket2, err := ticket.ReconstructTicket(
		2,
		"TKT-002",
		"Second ticket",
		"Description 2",
		vo.CategoryBilling,
		vo.PriorityMedium,
		vo.StatusNew,
		10,
		nil,
		[]string{},
		nil,
		nil,
		nil,
		nil,
		1,
		time.Now().Add(-1*time.Hour),
		time.Now().Add(-30*time.Minute),
		nil,
	)
	require.NoError(t, err)

	tests := []struct {
		name          string
		query         ListTicketsQuery
		expectedCount int
	}{
		{
			name: "admin can see all tickets",
			query: ListTicketsQuery{
				UserID:    99,
				UserRoles: []string{"admin"},
				Page:      1,
				PageSize:  10,
			},
			expectedCount: 2,
		},
		{
			name: "support agent can see all tickets",
			query: ListTicketsQuery{
				UserID:    88,
				UserRoles: []string{"support_agent"},
				Page:      1,
				PageSize:  10,
			},
			expectedCount: 2,
		},
		{
			name: "user can only see their tickets",
			query: ListTicketsQuery{
				UserID:    10,
				UserRoles: []string{"user"},
				Page:      1,
				PageSize:  10,
			},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTicketRepo := &mockTicketRepository{
				ListFunc: func(ctx context.Context, filters ticket.TicketFilter) ([]*ticket.Ticket, int64, error) {
					return []*ticket.Ticket{ticket1, ticket2}, 2, nil
				},
			}

			mockLog := &mockLogger{}

			useCase := NewListTicketsUseCase(mockTicketRepo, mockLog)
			result, err := useCase.Execute(context.Background(), tt.query)

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.expectedCount, len(result.Tickets))
			assert.Equal(t, int64(2), result.TotalCount)
			assert.Equal(t, tt.query.Page, result.Page)
			assert.Equal(t, tt.query.PageSize, result.PageSize)
		})
	}
}

func TestListTicketsUseCase_Execute_WithFilters(t *testing.T) {
	ticket1, err := ticket.ReconstructTicket(
		1,
		"TKT-001",
		"High priority ticket",
		"Description",
		vo.CategoryTechnical,
		vo.PriorityHigh,
		vo.StatusOpen,
		10,
		nil,
		[]string{},
		nil,
		nil,
		nil,
		nil,
		1,
		time.Now().Add(-1*time.Hour),
		time.Now().Add(-30*time.Minute),
		nil,
	)
	require.NoError(t, err)

	mockTicketRepo := &mockTicketRepository{
		ListFunc: func(ctx context.Context, filters ticket.TicketFilter) ([]*ticket.Ticket, int64, error) {
			assert.NotNil(t, filters.Priority)
			assert.Equal(t, vo.PriorityHigh, *filters.Priority)
			return []*ticket.Ticket{ticket1}, 1, nil
		},
	}

	mockLog := &mockLogger{}

	useCase := NewListTicketsUseCase(mockTicketRepo, mockLog)
	priorityStr := string(vo.PriorityHigh)
	query := ListTicketsQuery{
		UserID:    99,
		UserRoles: []string{"admin"},
		Priority:  &priorityStr,
		Page:      1,
		PageSize:  10,
	}

	result, err := useCase.Execute(context.Background(), query)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, len(result.Tickets))
	assert.Equal(t, "High priority ticket", result.Tickets[0].Title)
}

func TestListTicketsUseCase_Execute_Pagination(t *testing.T) {
	mockTicketRepo := &mockTicketRepository{
		ListFunc: func(ctx context.Context, filters ticket.TicketFilter) ([]*ticket.Ticket, int64, error) {
			assert.Equal(t, 2, filters.Page)
			assert.Equal(t, 20, filters.PageSize)
			return []*ticket.Ticket{}, 50, nil
		},
	}

	mockLog := &mockLogger{}

	useCase := NewListTicketsUseCase(mockTicketRepo, mockLog)
	query := ListTicketsQuery{
		UserID:    99,
		UserRoles: []string{"admin"},
		Page:      2,
		PageSize:  20,
	}

	result, err := useCase.Execute(context.Background(), query)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2, result.Page)
	assert.Equal(t, 20, result.PageSize)
	assert.Equal(t, int64(50), result.TotalCount)
}

func TestListTicketsUseCase_Execute_DefaultPagination(t *testing.T) {
	mockTicketRepo := &mockTicketRepository{
		ListFunc: func(ctx context.Context, filters ticket.TicketFilter) ([]*ticket.Ticket, int64, error) {
			assert.Equal(t, 1, filters.Page)
			assert.Equal(t, 20, filters.PageSize)
			return []*ticket.Ticket{}, 0, nil
		},
	}

	mockLog := &mockLogger{}

	useCase := NewListTicketsUseCase(mockTicketRepo, mockLog)
	query := ListTicketsQuery{
		UserID:    99,
		UserRoles: []string{"admin"},
		Page:      0,
		PageSize:  0,
	}

	result, err := useCase.Execute(context.Background(), query)

	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestListTicketsUseCase_Execute_MaxPageSize(t *testing.T) {
	mockTicketRepo := &mockTicketRepository{
		ListFunc: func(ctx context.Context, filters ticket.TicketFilter) ([]*ticket.Ticket, int64, error) {
			assert.Equal(t, 100, filters.PageSize)
			return []*ticket.Ticket{}, 0, nil
		},
	}

	mockLog := &mockLogger{}

	useCase := NewListTicketsUseCase(mockTicketRepo, mockLog)
	query := ListTicketsQuery{
		UserID:    99,
		UserRoles: []string{"admin"},
		Page:      1,
		PageSize:  200,
	}

	result, err := useCase.Execute(context.Background(), query)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 100, result.PageSize)
}

func TestListTicketsUseCase_Execute_InvalidFilters(t *testing.T) {
	tests := []struct {
		name          string
		query         ListTicketsQuery
		expectedError string
	}{
		{
			name: "invalid status",
			query: ListTicketsQuery{
				UserID:    99,
				UserRoles: []string{"admin"},
				Status:    stringPtr("invalid_status"),
				Page:      1,
				PageSize:  10,
			},
			expectedError: "invalid status",
		},
		{
			name: "invalid priority",
			query: ListTicketsQuery{
				UserID:    99,
				UserRoles: []string{"admin"},
				Priority:  stringPtr("invalid_priority"),
				Page:      1,
				PageSize:  10,
			},
			expectedError: "invalid priority",
		},
		{
			name: "invalid category",
			query: ListTicketsQuery{
				UserID:    99,
				UserRoles: []string{"admin"},
				Category:  stringPtr("invalid_category"),
				Page:      1,
				PageSize:  10,
			},
			expectedError: "invalid category",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTicketRepo := &mockTicketRepository{}
			mockLog := &mockLogger{}

			useCase := NewListTicketsUseCase(mockTicketRepo, mockLog)
			result, err := useCase.Execute(context.Background(), tt.query)

			require.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestListTicketsUseCase_Execute_RepositoryError(t *testing.T) {
	mockTicketRepo := &mockTicketRepository{
		ListFunc: func(ctx context.Context, filters ticket.TicketFilter) ([]*ticket.Ticket, int64, error) {
			return nil, 0, errors.New("database error")
		},
	}

	mockLog := &mockLogger{}

	useCase := NewListTicketsUseCase(mockTicketRepo, mockLog)
	query := ListTicketsQuery{
		UserID:    99,
		UserRoles: []string{"admin"},
		Page:      1,
		PageSize:  10,
	}

	result, err := useCase.Execute(context.Background(), query)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to list tickets")
}

func TestListTicketsUseCase_Execute_UserOnlySeesOwnTickets(t *testing.T) {
	userID := uint(10)

	mockTicketRepo := &mockTicketRepository{
		ListFunc: func(ctx context.Context, filters ticket.TicketFilter) ([]*ticket.Ticket, int64, error) {
			assert.NotNil(t, filters.CreatorID)
			assert.Equal(t, userID, *filters.CreatorID)
			return []*ticket.Ticket{}, 0, nil
		},
	}

	mockLog := &mockLogger{}

	useCase := NewListTicketsUseCase(mockTicketRepo, mockLog)
	query := ListTicketsQuery{
		UserID:    userID,
		UserRoles: []string{"user"},
		Page:      1,
		PageSize:  10,
	}

	result, err := useCase.Execute(context.Background(), query)

	require.NoError(t, err)
	require.NotNil(t, result)
}

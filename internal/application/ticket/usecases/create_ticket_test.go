package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"orris/internal/domain/shared/events"
	"orris/internal/domain/ticket"
	vo "orris/internal/domain/ticket/value_objects"
)

func TestCreateTicketUseCase_Execute_Success(t *testing.T) {
	tests := []struct {
		name    string
		command CreateTicketCommand
	}{
		{
			name: "create technical ticket with high priority",
			command: CreateTicketCommand{
				Title:       "System crashes on login",
				Description: "Users experiencing crashes when attempting to login",
				Category:    string(vo.CategoryTechnical),
				Priority:    string(vo.PriorityHigh),
				CreatorID:   1,
				Tags:        []string{"crash", "login"},
				Metadata:    map[string]interface{}{"env": "production"},
			},
		},
		{
			name: "create billing ticket with low priority",
			command: CreateTicketCommand{
				Title:       "Invoice clarification needed",
				Description: "Need clarification on last month's invoice",
				Category:    string(vo.CategoryBilling),
				Priority:    string(vo.PriorityLow),
				CreatorID:   2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var savedTicket *ticket.Ticket
			mockRepo := &mockTicketRepository{
				SaveFunc: func(ctx context.Context, tkt *ticket.Ticket) error {
					err := tkt.SetID(100)
					if err != nil {
						return err
					}
					err = tkt.SetNumber("TKT-100")
					if err != nil {
						return err
					}
					savedTicket = tkt
					return nil
				},
			}
			mockDispatcher := &mockEventDispatcher{
				PublishAllFunc: func(events []events.DomainEvent) error {
					assert.NotEmpty(t, events)
					return nil
				},
			}
			mockLog := &mockLogger{}

			useCase := NewCreateTicketUseCase(mockRepo, mockDispatcher, mockLog)
			result, err := useCase.Execute(context.Background(), tt.command)

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, uint(100), result.TicketID)
			assert.Equal(t, "TKT-100", result.Number)
			assert.Equal(t, vo.StatusNew.String(), result.Status)
			assert.NotZero(t, result.CreatedAt)

			require.NotNil(t, savedTicket)
			assert.Equal(t, tt.command.Title, savedTicket.Title())
			assert.Equal(t, tt.command.Description, savedTicket.Description())
			assert.Equal(t, vo.Category(tt.command.Category), savedTicket.Category())
			assert.Equal(t, vo.Priority(tt.command.Priority), savedTicket.Priority())
		})
	}
}

func TestCreateTicketUseCase_Execute_ValidationErrors(t *testing.T) {
	tests := []struct {
		name          string
		command       CreateTicketCommand
		expectedError string
	}{
		{
			name: "empty title",
			command: CreateTicketCommand{
				Title:       "",
				Description: "Some description",
				Category:    string(vo.CategoryTechnical),
				Priority:    string(vo.PriorityMedium),
				CreatorID:   1,
			},
			expectedError: "title is required",
		},
		{
			name: "title too long",
			command: CreateTicketCommand{
				Title:       string(make([]byte, 201)),
				Description: "Some description",
				Category:    string(vo.CategoryTechnical),
				Priority:    string(vo.PriorityMedium),
				CreatorID:   1,
			},
			expectedError: "title exceeds maximum length",
		},
		{
			name: "empty description",
			command: CreateTicketCommand{
				Title:       "Valid title",
				Description: "",
				Category:    string(vo.CategoryTechnical),
				Priority:    string(vo.PriorityMedium),
				CreatorID:   1,
			},
			expectedError: "description is required",
		},
		{
			name: "description too long",
			command: CreateTicketCommand{
				Title:       "Valid title",
				Description: string(make([]byte, 5001)),
				Category:    string(vo.CategoryTechnical),
				Priority:    string(vo.PriorityMedium),
				CreatorID:   1,
			},
			expectedError: "description exceeds maximum length",
		},
		{
			name: "missing creator ID",
			command: CreateTicketCommand{
				Title:       "Valid title",
				Description: "Valid description",
				Category:    string(vo.CategoryTechnical),
				Priority:    string(vo.PriorityMedium),
				CreatorID:   0,
			},
			expectedError: "creator ID is required",
		},
		{
			name: "invalid category",
			command: CreateTicketCommand{
				Title:       "Valid title",
				Description: "Valid description",
				Category:    "invalid_category",
				Priority:    string(vo.PriorityMedium),
				CreatorID:   1,
			},
			expectedError: "invalid category",
		},
		{
			name: "invalid priority",
			command: CreateTicketCommand{
				Title:       "Valid title",
				Description: "Valid description",
				Category:    string(vo.CategoryTechnical),
				Priority:    "invalid_priority",
				CreatorID:   1,
			},
			expectedError: "invalid priority",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockTicketRepository{}
			mockDispatcher := &mockEventDispatcher{}
			mockLog := &mockLogger{}

			useCase := NewCreateTicketUseCase(mockRepo, mockDispatcher, mockLog)
			result, err := useCase.Execute(context.Background(), tt.command)

			require.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestCreateTicketUseCase_Execute_RepositoryError(t *testing.T) {
	mockRepo := &mockTicketRepository{
		SaveFunc: func(ctx context.Context, t *ticket.Ticket) error {
			return errors.New("database connection failed")
		},
	}
	mockDispatcher := &mockEventDispatcher{}
	mockLog := &mockLogger{}

	useCase := NewCreateTicketUseCase(mockRepo, mockDispatcher, mockLog)
	cmd := CreateTicketCommand{
		Title:       "Valid title",
		Description: "Valid description",
		Category:    string(vo.CategoryTechnical),
		Priority:    string(vo.PriorityMedium),
		CreatorID:   1,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "database connection failed")
}

func TestCreateTicketUseCase_Execute_EventPublishError(t *testing.T) {
	mockRepo := &mockTicketRepository{
		SaveFunc: func(ctx context.Context, tkt *ticket.Ticket) error {
			err := tkt.SetID(100)
			if err != nil {
				return err
			}
			err = tkt.SetNumber("TKT-100")
			if err != nil {
				return err
			}
			return nil
		},
	}
	mockDispatcher := &mockEventDispatcher{
		PublishAllFunc: func(events []events.DomainEvent) error {
			return errors.New("event publish failed")
		},
	}
	mockLog := &mockLogger{}

	useCase := NewCreateTicketUseCase(mockRepo, mockDispatcher, mockLog)
	cmd := CreateTicketCommand{
		Title:       "Valid title",
		Description: "Valid description",
		Category:    string(vo.CategoryTechnical),
		Priority:    string(vo.PriorityMedium),
		CreatorID:   1,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, uint(100), result.TicketID)
}

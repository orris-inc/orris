package usecases

import (
	"context"
	"testing"
	"time"

	"orris/internal/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGenerateNodeTokenUseCase_Execute_Success(t *testing.T) {
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	logger.On("Infow", "executing generate node token use case", mock.Anything).Return()
	logger.On("Infow", "node token generated successfully", mock.Anything).Return()

	uc := NewGenerateNodeTokenUseCase(dispatcher, logger)

	cmd := GenerateNodeTokenCommand{
		NodeID: 1,
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uint(1), result.NodeID)
	assert.False(t, result.CreatedAt.IsZero())
	logger.AssertExpectations(t)
}

func TestGenerateNodeTokenUseCase_Execute_WithExpiration(t *testing.T) {
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	logger.On("Infow", "executing generate node token use case", mock.Anything).Return()
	logger.On("Infow", "node token generated successfully", mock.Anything).Return()

	uc := NewGenerateNodeTokenUseCase(dispatcher, logger)

	expiresAt := time.Now().Add(24 * time.Hour)
	cmd := GenerateNodeTokenCommand{
		NodeID:    1,
		ExpiresAt: &expiresAt,
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uint(1), result.NodeID)
	assert.NotNil(t, result.ExpiresAt)
	logger.AssertExpectations(t)
}

func TestGenerateNodeTokenUseCase_Execute_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		cmd     GenerateNodeTokenCommand
		wantErr string
	}{
		{
			name: "missing node id",
			cmd: GenerateNodeTokenCommand{
				NodeID: 0,
			},
			wantErr: "node id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := new(mockEventDispatcher)
			logger := new(mockLogger)

			logger.On("Infow", "executing generate node token use case", mock.Anything).Return()
			logger.On("Errorw", "invalid generate node token command", mock.Anything).Return()

			uc := NewGenerateNodeTokenUseCase(dispatcher, logger)

			result, err := uc.Execute(context.Background(), tt.cmd)

			assert.Error(t, err)
			assert.Nil(t, result)

			appErr, ok := err.(*errors.AppError)
			assert.True(t, ok)
			assert.Equal(t, errors.ErrorTypeValidation, appErr.Type)
			assert.Equal(t, tt.wantErr, appErr.Message)
		})
	}
}

func TestGenerateNodeTokenUseCase_Execute_ExpirationInPast(t *testing.T) {
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	logger.On("Infow", "executing generate node token use case", mock.Anything).Return()
	logger.On("Warnw", "expiration time is in the past", mock.Anything).Return()

	uc := NewGenerateNodeTokenUseCase(dispatcher, logger)

	expiresAt := time.Now().Add(-24 * time.Hour)
	cmd := GenerateNodeTokenCommand{
		NodeID:    1,
		ExpiresAt: &expiresAt,
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.Error(t, err)
	assert.Nil(t, result)

	appErr, ok := err.(*errors.AppError)
	assert.True(t, ok)
	assert.Equal(t, errors.ErrorTypeValidation, appErr.Type)
	assert.Equal(t, "expiration time cannot be in the past", appErr.Message)
}

func TestGenerateNodeTokenUseCase_Execute_TokenUniqueness(t *testing.T) {
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	logger.On("Infow", "executing generate node token use case", mock.Anything).Return()
	logger.On("Infow", "node token generated successfully", mock.Anything).Return()

	uc := NewGenerateNodeTokenUseCase(dispatcher, logger)

	tokens := make(map[string]bool)
	numTokens := 10

	for i := 0; i < numTokens; i++ {
		cmd := GenerateNodeTokenCommand{
			NodeID: uint(i + 1),
		}

		result, err := uc.Execute(context.Background(), cmd)

		assert.NoError(t, err)
		assert.NotNil(t, result)

		if result.Token != "" {
			assert.False(t, tokens[result.Token], "token should be unique")
			tokens[result.Token] = true
		}
	}
}

func TestGenerateNodeTokenUseCase_Execute_MultipleNodes(t *testing.T) {
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	logger.On("Infow", "executing generate node token use case", mock.Anything).Return()
	logger.On("Infow", "node token generated successfully", mock.Anything).Return()

	uc := NewGenerateNodeTokenUseCase(dispatcher, logger)

	nodeIDs := []uint{1, 2, 3, 4, 5}

	for _, nodeID := range nodeIDs {
		cmd := GenerateNodeTokenCommand{
			NodeID: nodeID,
		}

		result, err := uc.Execute(context.Background(), cmd)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, nodeID, result.NodeID)
	}
}

func TestGenerateNodeTokenUseCase_Execute_TokenFormat(t *testing.T) {
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	logger.On("Infow", "executing generate node token use case", mock.Anything).Return()
	logger.On("Infow", "node token generated successfully", mock.Anything).Return()

	uc := NewGenerateNodeTokenUseCase(dispatcher, logger)

	cmd := GenerateNodeTokenCommand{
		NodeID: 1,
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGenerateNodeTokenUseCase_ValidateCommand(t *testing.T) {
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	uc := NewGenerateNodeTokenUseCase(dispatcher, logger)

	validCmd := GenerateNodeTokenCommand{
		NodeID: 1,
	}

	err := uc.validateCommand(validCmd)
	assert.NoError(t, err)

	invalidCmd := GenerateNodeTokenCommand{
		NodeID: 0,
	}

	err = uc.validateCommand(invalidCmd)
	assert.Error(t, err)
}

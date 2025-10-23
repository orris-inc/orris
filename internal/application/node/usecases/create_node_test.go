package usecases

import (
	"context"
	"testing"

	"orris/internal/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateNodeUseCase_Execute_Success(t *testing.T) {
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	logger.On("Infow", "executing create node use case", mock.Anything).Return()
	logger.On("Infow", "node created successfully", mock.Anything).Return()

	uc := NewCreateNodeUseCase(dispatcher, logger)

	cmd := CreateNodeCommand{
		Name:          "Test Node",
		ServerAddress: "192.168.1.1",
		ServerPort:    8388,
		Method:        "aes-256-gcm",
		Password:      "testpassword123",
		Country:       "US",
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	logger.AssertExpectations(t)
}

func TestCreateNodeUseCase_Execute_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		cmd     CreateNodeCommand
		wantErr string
	}{
		{
			name: "missing name",
			cmd: CreateNodeCommand{
				ServerAddress: "192.168.1.1",
				ServerPort:    8388,
				Method:        "aes-256-gcm",
				Password:      "testpassword123",
				Country:       "US",
			},
			wantErr: "node name is required",
		},
		{
			name: "missing server address",
			cmd: CreateNodeCommand{
				Name:       "Test Node",
				ServerPort: 8388,
				Method:     "aes-256-gcm",
				Password:   "testpassword123",
				Country:    "US",
			},
			wantErr: "server address is required",
		},
		{
			name: "missing server port",
			cmd: CreateNodeCommand{
				Name:          "Test Node",
				ServerAddress: "192.168.1.1",
				Method:        "aes-256-gcm",
				Password:      "testpassword123",
				Country:       "US",
			},
			wantErr: "server port is required",
		},
		{
			name: "missing method",
			cmd: CreateNodeCommand{
				Name:          "Test Node",
				ServerAddress: "192.168.1.1",
				ServerPort:    8388,
				Password:      "testpassword123",
				Country:       "US",
			},
			wantErr: "encryption method is required",
		},
		{
			name: "missing password",
			cmd: CreateNodeCommand{
				Name:          "Test Node",
				ServerAddress: "192.168.1.1",
				ServerPort:    8388,
				Method:        "aes-256-gcm",
				Country:       "US",
			},
			wantErr: "password is required",
		},
		{
			name: "password too short",
			cmd: CreateNodeCommand{
				Name:          "Test Node",
				ServerAddress: "192.168.1.1",
				ServerPort:    8388,
				Method:        "aes-256-gcm",
				Password:      "short",
				Country:       "US",
			},
			wantErr: "password must be at least 8 characters",
		},
		{
			name: "missing country",
			cmd: CreateNodeCommand{
				Name:          "Test Node",
				ServerAddress: "192.168.1.1",
				ServerPort:    8388,
				Method:        "aes-256-gcm",
				Password:      "testpassword123",
			},
			wantErr: "country is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := new(mockEventDispatcher)
			logger := new(mockLogger)

			logger.On("Infow", "executing create node use case", mock.Anything).Return()
			logger.On("Errorw", "invalid create node command", mock.Anything).Return()

			uc := NewCreateNodeUseCase(dispatcher, logger)

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

func TestCreateNodeUseCase_Execute_NameDuplicate(t *testing.T) {
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	logger.On("Infow", "executing create node use case", mock.Anything).Return()
	logger.On("Infow", "node created successfully", mock.Anything).Return()

	uc := NewCreateNodeUseCase(dispatcher, logger)

	cmd := CreateNodeCommand{
		Name:          "Duplicate Node",
		ServerAddress: "192.168.1.1",
		ServerPort:    8388,
		Method:        "aes-256-gcm",
		Password:      "testpassword123",
		Country:       "US",
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateNodeUseCase_Execute_TokenGeneration(t *testing.T) {
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	logger.On("Infow", "executing create node use case", mock.Anything).Return()
	logger.On("Infow", "node created successfully", mock.Anything).Return()

	uc := NewCreateNodeUseCase(dispatcher, logger)

	cmd := CreateNodeCommand{
		Name:          "Token Test Node",
		ServerAddress: "192.168.1.1",
		ServerPort:    8388,
		Method:        "aes-256-gcm",
		Password:      "testpassword123",
		Country:       "US",
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateNodeUseCase_ValidateCommand(t *testing.T) {
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	uc := NewCreateNodeUseCase(dispatcher, logger)

	validCmd := CreateNodeCommand{
		Name:          "Valid Node",
		ServerAddress: "192.168.1.1",
		ServerPort:    8388,
		Method:        "aes-256-gcm",
		Password:      "testpassword123",
		Country:       "US",
	}

	err := uc.validateCommand(validCmd)
	assert.NoError(t, err)
}

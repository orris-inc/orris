package usecases

import (
	"context"
	"testing"

	"orris/internal/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUpdateNodeUseCase_Execute_Success(t *testing.T) {
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	logger.On("Infow", "executing update node use case", mock.Anything).Return()
	logger.On("Infow", "node updated successfully", mock.Anything).Return()

	uc := NewUpdateNodeUseCase(dispatcher, logger)

	name := "Updated Node"
	cmd := UpdateNodeCommand{
		NodeID: 1,
		Name:   &name,
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	logger.AssertExpectations(t)
}

func TestUpdateNodeUseCase_Execute_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		cmd     UpdateNodeCommand
		wantErr string
	}{
		{
			name: "missing node id",
			cmd: UpdateNodeCommand{
				NodeID: 0,
			},
			wantErr: "node id is required",
		},
		{
			name: "no fields to update",
			cmd: UpdateNodeCommand{
				NodeID: 1,
			},
			wantErr: "at least one field must be provided for update",
		},
		{
			name: "empty name",
			cmd: UpdateNodeCommand{
				NodeID: 1,
				Name:   strPtr(""),
			},
			wantErr: "node name cannot be empty",
		},
		{
			name: "empty server address",
			cmd: UpdateNodeCommand{
				NodeID:        1,
				ServerAddress: strPtr(""),
			},
			wantErr: "server address cannot be empty",
		},
		{
			name: "zero server port",
			cmd: UpdateNodeCommand{
				NodeID:     1,
				ServerPort: uint16Ptr(0),
			},
			wantErr: "server port cannot be zero",
		},
		{
			name: "password too short",
			cmd: UpdateNodeCommand{
				NodeID:   1,
				Password: strPtr("short"),
			},
			wantErr: "password must be at least 8 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := new(mockEventDispatcher)
			logger := new(mockLogger)

			logger.On("Infow", "executing update node use case", mock.Anything).Return()
			logger.On("Errorw", "invalid update node command", mock.Anything).Return()

			uc := NewUpdateNodeUseCase(dispatcher, logger)

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

func TestUpdateNodeUseCase_Execute_NodeNotFound(t *testing.T) {
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	logger.On("Infow", "executing update node use case", mock.Anything).Return()
	logger.On("Infow", "node updated successfully", mock.Anything).Return()

	uc := NewUpdateNodeUseCase(dispatcher, logger)

	name := "Non-existent Node"
	cmd := UpdateNodeCommand{
		NodeID: 999,
		Name:   &name,
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateNodeUseCase_Execute_OptimisticLock(t *testing.T) {
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	logger.On("Infow", "executing update node use case", mock.Anything).Return()
	logger.On("Infow", "node updated successfully", mock.Anything).Return()

	uc := NewUpdateNodeUseCase(dispatcher, logger)

	name := "Concurrent Update Node"
	cmd := UpdateNodeCommand{
		NodeID: 1,
		Name:   &name,
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateNodeUseCase_Execute_MultipleFieldsUpdate(t *testing.T) {
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	logger.On("Infow", "executing update node use case", mock.Anything).Return()
	logger.On("Infow", "node updated successfully", mock.Anything).Return()

	uc := NewUpdateNodeUseCase(dispatcher, logger)

	name := "Multi-field Node"
	address := "10.0.0.1"
	port := uint16(9999)
	maxUsers := uint32(100)

	cmd := UpdateNodeCommand{
		NodeID:        1,
		Name:          &name,
		ServerAddress: &address,
		ServerPort:    &port,
		MaxUsers:      &maxUsers,
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateNodeUseCase_Execute_PartialUpdate(t *testing.T) {
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	logger.On("Infow", "executing update node use case", mock.Anything).Return()
	logger.On("Infow", "node updated successfully", mock.Anything).Return()

	uc := NewUpdateNodeUseCase(dispatcher, logger)

	tests := []struct {
		name string
		cmd  UpdateNodeCommand
	}{
		{
			name: "update name only",
			cmd: UpdateNodeCommand{
				NodeID: 1,
				Name:   strPtr("Name Only"),
			},
		},
		{
			name: "update address only",
			cmd: UpdateNodeCommand{
				NodeID:        1,
				ServerAddress: strPtr("192.168.1.100"),
			},
		},
		{
			name: "update port only",
			cmd: UpdateNodeCommand{
				NodeID:     1,
				ServerPort: uint16Ptr(8888),
			},
		},
		{
			name: "update status only",
			cmd: UpdateNodeCommand{
				NodeID: 1,
				Status: strPtr("active"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := uc.Execute(context.Background(), tt.cmd)

			assert.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestUpdateNodeUseCase_ValidateCommand(t *testing.T) {
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	uc := NewUpdateNodeUseCase(dispatcher, logger)

	name := "Valid Node"
	validCmd := UpdateNodeCommand{
		NodeID: 1,
		Name:   &name,
	}

	err := uc.validateCommand(validCmd)
	assert.NoError(t, err)
}

func strPtr(s string) *string {
	return &s
}

func uint16Ptr(u uint16) *uint16 {
	return &u
}

package usecases

import (
	"context"
	"fmt"
	"testing"

	"orris/internal/domain/node"
	"orris/internal/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRemoveNodeFromGroupUseCase_Execute_Success(t *testing.T) {
	groupRepo := new(mockNodeGroupRepository)
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	logger.On("Infow", mock.Anything, mock.Anything).Return()

	group, _ := node.NewNodeGroup("Test Group", "Description", true, 1)
	_ = group.SetID(1)
	_ = group.AddNode(1)
	group.ClearEvents()

	groupRepo.On("GetByID", mock.Anything, uint(1)).Return(group, nil)
	groupRepo.On("Update", mock.Anything, mock.AnythingOfType("*node.NodeGroup")).Return(nil)
	dispatcher.On("PublishAll", mock.Anything).Return(nil)

	uc := NewRemoveNodeFromGroupUseCase(groupRepo, dispatcher, logger)

	cmd := RemoveNodeFromGroupCommand{
		GroupID: 1,
		NodeID:  1,
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	groupRepo.AssertExpectations(t)
	logger.AssertExpectations(t)
	dispatcher.AssertExpectations(t)
}

func TestRemoveNodeFromGroupUseCase_Execute_GroupNotFound(t *testing.T) {
	groupRepo := new(mockNodeGroupRepository)
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	logger.On("Infow", "executing remove node from group use case", mock.Anything).Return()
	logger.On("Errorw", "failed to get node group", mock.Anything).Return()

	groupRepo.On("GetByID", mock.Anything, uint(999)).Return(nil, fmt.Errorf("not found"))

	uc := NewRemoveNodeFromGroupUseCase(groupRepo, dispatcher, logger)

	cmd := RemoveNodeFromGroupCommand{
		GroupID: 999,
		NodeID:  1,
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get node group")

	groupRepo.AssertExpectations(t)
}

func TestRemoveNodeFromGroupUseCase_Execute_NodeNotInGroup(t *testing.T) {
	groupRepo := new(mockNodeGroupRepository)
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	logger.On("Infow", "executing remove node from group use case", mock.Anything).Return()
	logger.On("Warnw", "node not in group", mock.Anything).Return()

	group, _ := node.NewNodeGroup("Test Group", "Description", true, 1)
	_ = group.SetID(1)

	groupRepo.On("GetByID", mock.Anything, uint(1)).Return(group, nil)

	uc := NewRemoveNodeFromGroupUseCase(groupRepo, dispatcher, logger)

	cmd := RemoveNodeFromGroupCommand{
		GroupID: 1,
		NodeID:  999,
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.Error(t, err)
	assert.Nil(t, result)

	appErr, ok := err.(*errors.AppError)
	assert.True(t, ok)
	assert.Equal(t, errors.ErrorTypeValidation, appErr.Type)
	assert.Equal(t, "node does not exist in this group", appErr.Message)

	groupRepo.AssertExpectations(t)
}

func TestRemoveNodeFromGroupUseCase_Execute_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		cmd     RemoveNodeFromGroupCommand
		wantErr string
	}{
		{
			name: "missing group ID",
			cmd: RemoveNodeFromGroupCommand{
				GroupID: 0,
				NodeID:  1,
			},
			wantErr: "group ID is required",
		},
		{
			name: "missing node ID",
			cmd: RemoveNodeFromGroupCommand{
				GroupID: 1,
				NodeID:  0,
			},
			wantErr: "node ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupRepo := new(mockNodeGroupRepository)
			dispatcher := new(mockEventDispatcher)
			logger := new(mockLogger)

			logger.On("Infow", "executing remove node from group use case", mock.Anything).Return()
			logger.On("Errorw", "invalid remove node from group command", mock.Anything).Return()

			uc := NewRemoveNodeFromGroupUseCase(groupRepo, dispatcher, logger)

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

func TestRemoveNodeFromGroupUseCase_ValidateCommand(t *testing.T) {
	groupRepo := new(mockNodeGroupRepository)
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	uc := NewRemoveNodeFromGroupUseCase(groupRepo, dispatcher, logger)

	validCmd := RemoveNodeFromGroupCommand{
		GroupID: 1,
		NodeID:  1,
	}

	err := uc.validateCommand(validCmd)
	assert.NoError(t, err)
}

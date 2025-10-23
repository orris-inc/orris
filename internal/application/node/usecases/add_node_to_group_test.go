package usecases

import (
	"context"
	"fmt"
	"testing"

	"orris/internal/domain/node"
	vo "orris/internal/domain/node/value_objects"
	"orris/internal/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAddNodeToGroupUseCase_Execute_Success(t *testing.T) {
	nodeRepo := new(mockNodeRepository)
	groupRepo := new(mockNodeGroupRepository)
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	logger.On("Infow", mock.Anything, mock.Anything).Return()

	serverAddr, _ := vo.NewServerAddress("192.168.1.1")
	encryptionConfig, _ := vo.NewEncryptionConfig("aes-256-gcm", "testpassword123")
	metadata := vo.NewNodeMetadata("US", "", nil, "")
	nodeEntity, _ := node.NewNode(
		"Test Node",
		serverAddr,
		8388,
		encryptionConfig,
		nil,
		metadata,
		100,
		0,
		1,
	)
	_ = nodeEntity.SetID(1)

	group, _ := node.NewNodeGroup("Test Group", "Description", true, 1)
	_ = group.SetID(1)
	group.ClearEvents()

	nodeRepo.On("GetByID", mock.Anything, uint(1)).Return(nodeEntity, nil)
	groupRepo.On("GetByID", mock.Anything, uint(1)).Return(group, nil)
	groupRepo.On("Update", mock.Anything, mock.AnythingOfType("*node.NodeGroup")).Return(nil)
	dispatcher.On("PublishAll", mock.Anything).Return(nil)

	uc := NewAddNodeToGroupUseCase(nodeRepo, groupRepo, dispatcher, logger)

	cmd := AddNodeToGroupCommand{
		GroupID: 1,
		NodeID:  1,
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uint(1), result.GroupID)
	assert.Equal(t, uint(1), result.NodeID)

	nodeRepo.AssertExpectations(t)
	groupRepo.AssertExpectations(t)
	logger.AssertExpectations(t)
	dispatcher.AssertExpectations(t)
}

func TestAddNodeToGroupUseCase_Execute_NodeNotFound(t *testing.T) {
	nodeRepo := new(mockNodeRepository)
	groupRepo := new(mockNodeGroupRepository)
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	logger.On("Infow", "executing add node to group use case", mock.Anything).Return()
	logger.On("Errorw", "failed to get node", mock.Anything).Return()

	nodeRepo.On("GetByID", mock.Anything, uint(999)).Return(nil, fmt.Errorf("not found"))

	uc := NewAddNodeToGroupUseCase(nodeRepo, groupRepo, dispatcher, logger)

	cmd := AddNodeToGroupCommand{
		GroupID: 1,
		NodeID:  999,
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.Error(t, err)
	assert.Nil(t, result)

	nodeRepo.AssertExpectations(t)
}

func TestAddNodeToGroupUseCase_Execute_GroupNotFound(t *testing.T) {
	nodeRepo := new(mockNodeRepository)
	groupRepo := new(mockNodeGroupRepository)
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	logger.On("Infow", "executing add node to group use case", mock.Anything).Return()
	logger.On("Errorw", "failed to get node group", mock.Anything).Return()

	serverAddr, _ := vo.NewServerAddress("192.168.1.1")
	encryptionConfig, _ := vo.NewEncryptionConfig("aes-256-gcm", "testpassword123")
	metadata := vo.NewNodeMetadata("US", "", nil, "")
	nodeEntity, _ := node.NewNode(
		"Test Node",
		serverAddr,
		8388,
		encryptionConfig,
		nil,
		metadata,
		100,
		0,
		1,
	)
	_ = nodeEntity.SetID(1)

	nodeRepo.On("GetByID", mock.Anything, uint(1)).Return(nodeEntity, nil)
	groupRepo.On("GetByID", mock.Anything, uint(999)).Return(nil, fmt.Errorf("not found"))

	uc := NewAddNodeToGroupUseCase(nodeRepo, groupRepo, dispatcher, logger)

	cmd := AddNodeToGroupCommand{
		GroupID: 999,
		NodeID:  1,
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get node group")

	nodeRepo.AssertExpectations(t)
	groupRepo.AssertExpectations(t)
}

func TestAddNodeToGroupUseCase_Execute_NodeAlreadyInGroup(t *testing.T) {
	nodeRepo := new(mockNodeRepository)
	groupRepo := new(mockNodeGroupRepository)
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	logger.On("Infow", "executing add node to group use case", mock.Anything).Return()

	serverAddr, _ := vo.NewServerAddress("192.168.1.1")
	encryptionConfig, _ := vo.NewEncryptionConfig("aes-256-gcm", "testpassword123")
	metadata := vo.NewNodeMetadata("US", "", nil, "")
	nodeEntity, _ := node.NewNode(
		"Test Node",
		serverAddr,
		8388,
		encryptionConfig,
		nil,
		metadata,
		100,
		0,
		1,
	)
	_ = nodeEntity.SetID(1)

	group, _ := node.NewNodeGroup("Test Group", "Description", true, 1)
	_ = group.SetID(1)
	_ = group.AddNode(1)

	nodeRepo.On("GetByID", mock.Anything, uint(1)).Return(nodeEntity, nil)
	groupRepo.On("GetByID", mock.Anything, uint(1)).Return(group, nil)

	uc := NewAddNodeToGroupUseCase(nodeRepo, groupRepo, dispatcher, logger)

	cmd := AddNodeToGroupCommand{
		GroupID: 1,
		NodeID:  1,
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.Error(t, err)
	assert.Nil(t, result)

	appErr, ok := err.(*errors.AppError)
	assert.True(t, ok)
	assert.Equal(t, errors.ErrorTypeValidation, appErr.Type)
	assert.Equal(t, "node already exists in this group", appErr.Message)

	nodeRepo.AssertExpectations(t)
	groupRepo.AssertExpectations(t)
}

func TestAddNodeToGroupUseCase_Execute_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		cmd     AddNodeToGroupCommand
		wantErr string
	}{
		{
			name: "missing group ID",
			cmd: AddNodeToGroupCommand{
				GroupID: 0,
				NodeID:  1,
			},
			wantErr: "group ID is required",
		},
		{
			name: "missing node ID",
			cmd: AddNodeToGroupCommand{
				GroupID: 1,
				NodeID:  0,
			},
			wantErr: "node ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodeRepo := new(mockNodeRepository)
			groupRepo := new(mockNodeGroupRepository)
			dispatcher := new(mockEventDispatcher)
			logger := new(mockLogger)

			logger.On("Infow", "executing add node to group use case", mock.Anything).Return()
			logger.On("Errorw", "invalid add node to group command", mock.Anything).Return()

			uc := NewAddNodeToGroupUseCase(nodeRepo, groupRepo, dispatcher, logger)

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

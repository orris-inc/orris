package usecases

import (
	"context"
	"testing"

	"orris/internal/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateNodeGroupUseCase_Execute_Success(t *testing.T) {
	repo := new(mockNodeGroupRepository)
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	logger.On("Infow", mock.Anything, mock.Anything).Return()

	repo.On("ExistsByName", mock.Anything, "Test Group").Return(false, nil)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*node.NodeGroup")).Return(nil)

	dispatcher.On("PublishAll", mock.Anything).Return(nil)

	uc := NewCreateNodeGroupUseCase(repo, dispatcher, logger)

	cmd := CreateNodeGroupCommand{
		Name:        "Test Group",
		Description: "Test Description",
		IsPublic:    true,
		SortOrder:   1,
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Test Group", result.Name)
	assert.Equal(t, "Test Description", result.Description)
	assert.True(t, result.IsPublic)
	assert.Equal(t, 1, result.SortOrder)

	repo.AssertExpectations(t)
	logger.AssertExpectations(t)
	dispatcher.AssertExpectations(t)
}

func TestCreateNodeGroupUseCase_Execute_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		cmd     CreateNodeGroupCommand
		wantErr string
	}{
		{
			name: "missing name",
			cmd: CreateNodeGroupCommand{
				Description: "Test",
				IsPublic:    true,
				SortOrder:   1,
			},
			wantErr: "node group name is required",
		},
		{
			name: "name too long",
			cmd: CreateNodeGroupCommand{
				Name:        string(make([]byte, 256)),
				Description: "Test",
				IsPublic:    true,
				SortOrder:   1,
			},
			wantErr: "node group name is too long",
		},
		{
			name: "negative sort order",
			cmd: CreateNodeGroupCommand{
				Name:        "Test Group",
				Description: "Test",
				IsPublic:    true,
				SortOrder:   -1,
			},
			wantErr: "sort order cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockNodeGroupRepository)
			dispatcher := new(mockEventDispatcher)
			logger := new(mockLogger)

			logger.On("Infow", "executing create node group use case", mock.Anything).Return()
			logger.On("Errorw", "invalid create node group command", mock.Anything).Return()

			uc := NewCreateNodeGroupUseCase(repo, dispatcher, logger)

			result, err := uc.Execute(context.Background(), tt.cmd)

			assert.Error(t, err)
			assert.Nil(t, result)

			appErr, ok := err.(*errors.AppError)
			assert.True(t, ok)
			assert.Equal(t, errors.ErrorTypeValidation, appErr.Type)
			assert.Contains(t, appErr.Message, tt.wantErr)
		})
	}
}

func TestCreateNodeGroupUseCase_Execute_NameAlreadyExists(t *testing.T) {
	repo := new(mockNodeGroupRepository)
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	logger.On("Infow", "executing create node group use case", mock.Anything).Return()

	repo.On("ExistsByName", mock.Anything, "Existing Group").Return(true, nil)

	uc := NewCreateNodeGroupUseCase(repo, dispatcher, logger)

	cmd := CreateNodeGroupCommand{
		Name:        "Existing Group",
		Description: "Test Description",
		IsPublic:    true,
		SortOrder:   1,
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.Error(t, err)
	assert.Nil(t, result)

	appErr, ok := err.(*errors.AppError)
	assert.True(t, ok)
	assert.Equal(t, errors.ErrorTypeValidation, appErr.Type)
	assert.Equal(t, "node group name already exists", appErr.Message)

	repo.AssertExpectations(t)
	logger.AssertExpectations(t)
}

func TestCreateNodeGroupUseCase_ValidateCommand(t *testing.T) {
	repo := new(mockNodeGroupRepository)
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	uc := NewCreateNodeGroupUseCase(repo, dispatcher, logger)

	validCmd := CreateNodeGroupCommand{
		Name:        "Valid Group",
		Description: "Test",
		IsPublic:    true,
		SortOrder:   0,
	}

	err := uc.validateCommand(validCmd)
	assert.NoError(t, err)
}

package usecases

import (
	"context"
	"fmt"
	"testing"
	"time"

	"orris/internal/domain/node"
	"orris/internal/domain/subscription"
	vo "orris/internal/domain/subscription/value_objects"
	"orris/internal/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAssociateGroupWithPlanUseCase_Execute_Success(t *testing.T) {
	groupRepo := new(mockNodeGroupRepository)
	planRepo := new(mockSubscriptionPlanRepository)
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	logger.On("Infow", "executing associate group with plan use case", mock.Anything).Return()
	logger.On("Infow", "group associated with plan successfully", mock.Anything).Return()

	group, _ := node.NewNodeGroup("Test Group", "Description", true, 1)
	_ = group.SetID(1)

	billingCycle, _ := vo.NewBillingCycle("monthly")

	plan, _ := subscription.NewSubscriptionPlan(
		"Test Plan",
		"test-plan",
		"Test Description",
		9999,
		"USD",
		*billingCycle,
		0,
	)
	_ = plan.SetID(1)

	groupRepo.On("GetByID", mock.Anything, uint(1)).Return(group, nil)
	planRepo.On("GetByID", mock.Anything, uint(1)).Return(plan, nil)
	groupRepo.On("Update", mock.Anything, mock.AnythingOfType("*node.NodeGroup")).Return(nil)
	dispatcher.On("PublishAll", mock.Anything).Return(nil)

	uc := NewAssociateGroupWithPlanUseCase(groupRepo, planRepo, dispatcher, logger)

	cmd := AssociateGroupWithPlanCommand{
		GroupID: 1,
		PlanID:  1,
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uint(1), result.GroupID)
	assert.Equal(t, uint(1), result.PlanID)

	groupRepo.AssertExpectations(t)
	planRepo.AssertExpectations(t)
	logger.AssertExpectations(t)
	dispatcher.AssertExpectations(t)
}

func TestAssociateGroupWithPlanUseCase_Execute_GroupNotFound(t *testing.T) {
	groupRepo := new(mockNodeGroupRepository)
	planRepo := new(mockSubscriptionPlanRepository)
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	logger.On("Infow", "executing associate group with plan use case", mock.Anything).Return()
	logger.On("Errorw", "failed to get node group", mock.Anything).Return()

	groupRepo.On("GetByID", mock.Anything, uint(999)).Return(nil, fmt.Errorf("not found"))

	uc := NewAssociateGroupWithPlanUseCase(groupRepo, planRepo, dispatcher, logger)

	cmd := AssociateGroupWithPlanCommand{
		GroupID: 999,
		PlanID:  1,
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get node group")

	groupRepo.AssertExpectations(t)
}

func TestAssociateGroupWithPlanUseCase_Execute_PlanNotFound(t *testing.T) {
	groupRepo := new(mockNodeGroupRepository)
	planRepo := new(mockSubscriptionPlanRepository)
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	logger.On("Infow", "executing associate group with plan use case", mock.Anything).Return()
	logger.On("Errorw", "failed to get subscription plan", mock.Anything).Return()

	group, _ := node.NewNodeGroup("Test Group", "Description", true, 1)
	_ = group.SetID(1)

	groupRepo.On("GetByID", mock.Anything, uint(1)).Return(group, nil)
	planRepo.On("GetByID", mock.Anything, uint(999)).Return(nil, fmt.Errorf("not found"))

	uc := NewAssociateGroupWithPlanUseCase(groupRepo, planRepo, dispatcher, logger)

	cmd := AssociateGroupWithPlanCommand{
		GroupID: 1,
		PlanID:  999,
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get subscription plan")

	groupRepo.AssertExpectations(t)
	planRepo.AssertExpectations(t)
}

func TestAssociateGroupWithPlanUseCase_Execute_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		cmd     AssociateGroupWithPlanCommand
		wantErr string
	}{
		{
			name: "missing group ID",
			cmd: AssociateGroupWithPlanCommand{
				GroupID: 0,
				PlanID:  1,
			},
			wantErr: "group ID is required",
		},
		{
			name: "missing plan ID",
			cmd: AssociateGroupWithPlanCommand{
				GroupID: 1,
				PlanID:  0,
			},
			wantErr: "plan ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupRepo := new(mockNodeGroupRepository)
			planRepo := new(mockSubscriptionPlanRepository)
			dispatcher := new(mockEventDispatcher)
			logger := new(mockLogger)

			logger.On("Infow", "executing associate group with plan use case", mock.Anything).Return()
			logger.On("Errorw", "invalid associate group with plan command", mock.Anything).Return()

			uc := NewAssociateGroupWithPlanUseCase(groupRepo, planRepo, dispatcher, logger)

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

func TestAssociateGroupWithPlanUseCase_Execute_PlanAlreadyAssociated(t *testing.T) {
	groupRepo := new(mockNodeGroupRepository)
	planRepo := new(mockSubscriptionPlanRepository)
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	logger.On("Infow", "executing associate group with plan use case", mock.Anything).Return()
	logger.On("Warnw", mock.Anything, mock.Anything).Return()

	group, _ := node.ReconstructNodeGroup(
		1,
		"Test Group",
		"Description",
		[]uint{},
		[]uint{1},
		true,
		1,
		make(map[string]interface{}),
		1,
		time.Now(),
		time.Now(),
	)

	billingCycle, _ := vo.NewBillingCycle("monthly")

	plan, _ := subscription.NewSubscriptionPlan(
		"Test Plan",
		"test-plan",
		"Test Description",
		9999,
		"USD",
		*billingCycle,
		0,
	)
	_ = plan.SetID(1)

	groupRepo.On("GetByID", mock.Anything, uint(1)).Return(group, nil)
	planRepo.On("GetByID", mock.Anything, uint(1)).Return(plan, nil)

	uc := NewAssociateGroupWithPlanUseCase(groupRepo, planRepo, dispatcher, logger)

	cmd := AssociateGroupWithPlanCommand{
		GroupID: 1,
		PlanID:  1,
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.Error(t, err)
	assert.Nil(t, result)

	appErr, ok := err.(*errors.AppError)
	assert.True(t, ok)
	assert.Equal(t, errors.ErrorTypeValidation, appErr.Type)
	assert.Equal(t, "node group already associated with this plan", appErr.Message)

	groupRepo.AssertExpectations(t)
	planRepo.AssertExpectations(t)
}

func TestAssociateGroupWithPlanUseCase_ValidateCommand(t *testing.T) {
	groupRepo := new(mockNodeGroupRepository)
	planRepo := new(mockSubscriptionPlanRepository)
	dispatcher := new(mockEventDispatcher)
	logger := new(mockLogger)

	uc := NewAssociateGroupWithPlanUseCase(groupRepo, planRepo, dispatcher, logger)

	validCmd := AssociateGroupWithPlanCommand{
		GroupID: 1,
		PlanID:  1,
	}

	err := uc.validateCommand(validCmd)
	assert.NoError(t, err)
}

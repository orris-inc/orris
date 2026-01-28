package mappers

import (
	"encoding/json"
	"fmt"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/subscription"
	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/mapper"
)

type SubscriptionMapper interface {
	ToEntity(model *models.SubscriptionModel) (*subscription.Subscription, error)
	ToModel(entity *subscription.Subscription) (*models.SubscriptionModel, error)
	ToEntities(models []*models.SubscriptionModel) ([]*subscription.Subscription, error)
	ToModels(entities []*subscription.Subscription) ([]*models.SubscriptionModel, error)
}

type SubscriptionMapperImpl struct{}

func NewSubscriptionMapper() SubscriptionMapper {
	return &SubscriptionMapperImpl{}
}

func (m *SubscriptionMapperImpl) ToEntity(model *models.SubscriptionModel) (*subscription.Subscription, error) {
	if model == nil {
		return nil, nil
	}

	status := vo.SubscriptionStatus(model.Status)
	if !vo.ValidStatuses[status] {
		return nil, fmt.Errorf("invalid subscription status: %s", model.Status)
	}

	var metadata map[string]interface{}
	if model.Metadata != nil {
		if err := json.Unmarshal(model.Metadata, &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	// Parse billing cycle from model
	var billingCycle *vo.BillingCycle
	if model.BillingCycle != nil && *model.BillingCycle != "" {
		bc, err := vo.NewBillingCycle(*model.BillingCycle)
		if err != nil {
			return nil, fmt.Errorf("failed to parse billing cycle: %w", err)
		}
		billingCycle = bc
	}

	entity, err := subscription.ReconstructSubscriptionWithSubject(
		model.ID,
		model.UserID,
		model.PlanID,
		model.SubjectType,
		model.SubjectID,
		model.SID,
		model.UUID,
		model.LinkToken,
		status,
		model.StartDate,
		model.EndDate,
		model.AutoRenew,
		model.CurrentPeriodStart,
		model.CurrentPeriodEnd,
		model.CancelledAt,
		model.CancelReason,
		metadata,
		model.Version,
		model.CreatedAt,
		model.UpdatedAt,
		billingCycle,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct subscription entity: %w", err)
	}

	return entity, nil
}

func (m *SubscriptionMapperImpl) ToModel(entity *subscription.Subscription) (*models.SubscriptionModel, error) {
	if entity == nil {
		return nil, nil
	}

	var metadataJSON datatypes.JSON
	if metadata := entity.Metadata(); len(metadata) > 0 {
		data, err := json.Marshal(metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = data
	}

	// Convert billing cycle to string pointer
	var billingCycleStr *string
	if bc := entity.BillingCycle(); bc != nil {
		s := bc.String()
		billingCycleStr = &s
	}

	model := &models.SubscriptionModel{
		ID:                 entity.ID(),
		SID:                entity.SID(),
		UUID:               entity.UUID(),
		LinkToken:          entity.LinkToken(),
		UserID:             entity.UserID(),
		SubjectType:        entity.SubjectType(),
		SubjectID:          entity.SubjectID(),
		PlanID:             entity.PlanID(),
		Status:             entity.Status().String(),
		StartDate:          entity.StartDate(),
		EndDate:            entity.EndDate(),
		AutoRenew:          entity.AutoRenew(),
		BillingCycle:       billingCycleStr,
		CurrentPeriodStart: entity.CurrentPeriodStart(),
		CurrentPeriodEnd:   entity.CurrentPeriodEnd(),
		CancelledAt:        entity.CancelledAt(),
		CancelReason:       entity.CancelReason(),
		Metadata:           metadataJSON,
		Version:            entity.Version(),
		CreatedAt:          entity.CreatedAt(),
		UpdatedAt:          entity.UpdatedAt(),
	}

	if entity.Status() == vo.StatusCancelled {
		now := entity.UpdatedAt()
		model.DeletedAt = gorm.DeletedAt{
			Time:  now,
			Valid: true,
		}
	}

	return model, nil
}

func (m *SubscriptionMapperImpl) ToEntities(modelList []*models.SubscriptionModel) ([]*subscription.Subscription, error) {
	return mapper.MapSlicePtrWithID(modelList, m.ToEntity, func(model *models.SubscriptionModel) uint { return model.ID })
}

func (m *SubscriptionMapperImpl) ToModels(entities []*subscription.Subscription) ([]*models.SubscriptionModel, error) {
	return mapper.MapSlicePtrWithID(entities, m.ToModel, func(entity *subscription.Subscription) uint { return entity.ID() })
}

package mappers

import (
	"github.com/orris-inc/orris/internal/domain/telegram"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// TelegramBindingMapper handles conversion between TelegramBinding domain and model.
type TelegramBindingMapper interface {
	// ToModel converts domain entity to GORM model.
	ToModel(binding *telegram.TelegramBinding) *models.TelegramBindingModel

	// ToDomain converts GORM model to domain entity.
	ToDomain(model *models.TelegramBindingModel) *telegram.TelegramBinding
}

// TelegramBindingMapperImpl is the concrete implementation of TelegramBindingMapper.
type TelegramBindingMapperImpl struct{}

// NewTelegramBindingMapper creates a new TelegramBindingMapper.
func NewTelegramBindingMapper() TelegramBindingMapper {
	return &TelegramBindingMapperImpl{}
}

// ToModel converts domain entity to GORM model
func (m *TelegramBindingMapperImpl) ToModel(binding *telegram.TelegramBinding) *models.TelegramBindingModel {
	return &models.TelegramBindingModel{
		ID:                   binding.ID(),
		SID:                  binding.SID(),
		UserID:               binding.UserID(),
		TelegramUserID:       binding.TelegramUserID(),
		TelegramUsername:     binding.TelegramUsername(),
		Language:             binding.Language(),
		NotifyExpiring:       binding.NotifyExpiring(),
		NotifyTraffic:        binding.NotifyTraffic(),
		ExpiringDays:         binding.ExpiringDays(),
		TrafficThreshold:     binding.TrafficThreshold(),
		LastExpiringNotifyAt: binding.LastExpiringNotifyAt(),
		LastTrafficNotifyAt:  binding.LastTrafficNotifyAt(),
		CreatedAt:            binding.CreatedAt(),
		UpdatedAt:            binding.UpdatedAt(),
	}
}

// ToDomain converts GORM model to domain entity
func (m *TelegramBindingMapperImpl) ToDomain(model *models.TelegramBindingModel) *telegram.TelegramBinding {
	return telegram.ReconstructTelegramBinding(
		model.ID,
		model.SID,
		model.UserID,
		model.TelegramUserID,
		model.TelegramUsername,
		model.Language,
		model.NotifyExpiring,
		model.NotifyTraffic,
		model.ExpiringDays,
		model.TrafficThreshold,
		model.LastExpiringNotifyAt,
		model.LastTrafficNotifyAt,
		model.CreatedAt,
		model.UpdatedAt,
	)
}

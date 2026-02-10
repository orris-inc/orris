package mappers

import (
	"github.com/orris-inc/orris/internal/domain/telegram/admin"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// AdminTelegramBindingMapper handles conversion between AdminTelegramBinding domain and model.
type AdminTelegramBindingMapper interface {
	// ToModel converts domain entity to GORM model.
	ToModel(binding *admin.AdminTelegramBinding) *models.AdminTelegramBindingModel

	// ToDomain converts GORM model to domain entity.
	ToDomain(model *models.AdminTelegramBindingModel) *admin.AdminTelegramBinding
}

// AdminTelegramBindingMapperImpl is the concrete implementation of AdminTelegramBindingMapper.
type AdminTelegramBindingMapperImpl struct{}

// NewAdminTelegramBindingMapper creates a new AdminTelegramBindingMapper.
func NewAdminTelegramBindingMapper() AdminTelegramBindingMapper {
	return &AdminTelegramBindingMapperImpl{}
}

// ToModel converts domain entity to GORM model
func (m *AdminTelegramBindingMapperImpl) ToModel(binding *admin.AdminTelegramBinding) *models.AdminTelegramBindingModel {
	return &models.AdminTelegramBindingModel{
		ID:                             binding.ID(),
		SID:                            binding.SID(),
		UserID:                         binding.UserID(),
		TelegramUserID:                 binding.TelegramUserID(),
		TelegramUsername:               binding.TelegramUsername(),
		Language:                       binding.Language(),
		NotifyNodeOffline:              binding.NotifyNodeOffline(),
		NotifyAgentOffline:             binding.NotifyAgentOffline(),
		NotifyNewUser:                  binding.NotifyNewUser(),
		NotifyPaymentSuccess:           binding.NotifyPaymentSuccess(),
		NotifyDailySummary:             binding.NotifyDailySummary(),
		NotifyWeeklySummary:            binding.NotifyWeeklySummary(),
		OfflineThresholdMinutes:        binding.OfflineThresholdMinutes(),
		NotifyResourceExpiring:         binding.NotifyResourceExpiring(),
		ResourceExpiringDays:           binding.ResourceExpiringDays(),
		DailySummaryHour:               binding.DailySummaryHour(),
		WeeklySummaryHour:              binding.WeeklySummaryHour(),
		WeeklySummaryWeekday:           binding.WeeklySummaryWeekday(),
		OfflineCheckIntervalMinutes:    binding.OfflineCheckIntervalMinutes(),
		LastNodeOfflineNotifyAt:        binding.LastNodeOfflineNotifyAt(),
		LastAgentOfflineNotifyAt:       binding.LastAgentOfflineNotifyAt(),
		LastDailySummaryAt:             binding.LastDailySummaryAt(),
		LastWeeklySummaryAt:            binding.LastWeeklySummaryAt(),
		LastResourceExpiringNotifyDate: binding.LastResourceExpiringNotifyDate(),
		CreatedAt:                      binding.CreatedAt(),
		UpdatedAt:                      binding.UpdatedAt(),
	}
}

// ToDomain converts GORM model to domain entity
func (m *AdminTelegramBindingMapperImpl) ToDomain(model *models.AdminTelegramBindingModel) *admin.AdminTelegramBinding {
	return admin.ReconstructAdminTelegramBinding(
		model.ID,
		model.SID,
		model.UserID,
		model.TelegramUserID,
		model.TelegramUsername,
		model.Language,
		model.NotifyNodeOffline,
		model.NotifyAgentOffline,
		model.NotifyNewUser,
		model.NotifyPaymentSuccess,
		model.NotifyDailySummary,
		model.NotifyWeeklySummary,
		model.OfflineThresholdMinutes,
		model.NotifyResourceExpiring,
		model.ResourceExpiringDays,
		model.DailySummaryHour,
		model.WeeklySummaryHour,
		model.WeeklySummaryWeekday,
		model.OfflineCheckIntervalMinutes,
		model.LastNodeOfflineNotifyAt,
		model.LastAgentOfflineNotifyAt,
		model.LastDailySummaryAt,
		model.LastWeeklySummaryAt,
		model.LastResourceExpiringNotifyDate,
		model.CreatedAt,
		model.UpdatedAt,
	)
}

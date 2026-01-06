package mappers

import (
	"github.com/orris-inc/orris/internal/domain/setting"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// SystemSettingMapper provides methods for converting between domain and model
type SystemSettingMapper interface {
	ToDomain(model *models.SystemSettingModel) *setting.SystemSetting
	ToModel(domain *setting.SystemSetting) *models.SystemSettingModel
	ToDomainList(modelList []*models.SystemSettingModel) []*setting.SystemSetting
}

// SystemSettingMapperImpl implements SystemSettingMapper
type SystemSettingMapperImpl struct{}

// NewSystemSettingMapper creates a new SystemSettingMapper
func NewSystemSettingMapper() SystemSettingMapper {
	return &SystemSettingMapperImpl{}
}

// ToDomain converts a SystemSettingModel to a SystemSetting domain entity
func (m *SystemSettingMapperImpl) ToDomain(model *models.SystemSettingModel) *setting.SystemSetting {
	if model == nil {
		return nil
	}

	return setting.ReconstructSystemSetting(
		model.ID,
		model.SID,
		model.Category,
		model.SettingKey,
		model.Value,
		setting.ValueType(model.ValueType),
		model.Description,
		model.UpdatedBy,
		model.Version,
		model.CreatedAt,
		model.UpdatedAt,
	)
}

// ToModel converts a SystemSetting domain entity to a SystemSettingModel
func (m *SystemSettingMapperImpl) ToModel(domain *setting.SystemSetting) *models.SystemSettingModel {
	if domain == nil {
		return nil
	}

	return &models.SystemSettingModel{
		ID:          domain.ID(),
		SID:         domain.SID(),
		Category:    domain.Category(),
		SettingKey:  domain.Key(),
		Value:       domain.Value(),
		ValueType:   string(domain.ValueType()),
		Description: domain.Description(),
		UpdatedBy:   domain.UpdatedBy(),
		Version:     domain.Version(),
		CreatedAt:   domain.CreatedAt(),
		UpdatedAt:   domain.UpdatedAt(),
	}
}

// ToDomainList converts a list of SystemSettingModel to a list of SystemSetting domain entities
func (m *SystemSettingMapperImpl) ToDomainList(modelList []*models.SystemSettingModel) []*setting.SystemSetting {
	if modelList == nil {
		return nil
	}

	domains := make([]*setting.SystemSetting, 0, len(modelList))
	for _, model := range modelList {
		if domain := m.ToDomain(model); domain != nil {
			domains = append(domains, domain)
		}
	}

	return domains
}

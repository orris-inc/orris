package mappers

import (
	"fmt"

	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// VLESSConfigMapper handles the conversion between VLESSConfig value objects and persistence models
type VLESSConfigMapper interface {
	// ToValueObject converts a persistence model to a domain value object
	// UUID is passed separately as it's derived from subscription, not stored in DB
	ToValueObject(model *models.VLESSConfigModel) (*vo.VLESSConfig, error)

	// ToModel converts a domain value object to a persistence model
	ToModel(nodeID uint, config *vo.VLESSConfig) (*models.VLESSConfigModel, error)
}

// VLESSConfigMapperImpl is the concrete implementation of VLESSConfigMapper
type VLESSConfigMapperImpl struct{}

// NewVLESSConfigMapper creates a new VLESS config mapper
func NewVLESSConfigMapper() VLESSConfigMapper {
	return &VLESSConfigMapperImpl{}
}

// ToValueObject converts a persistence model to a domain value object
// Note: UUID is not stored in database, it's derived from subscription
func (m *VLESSConfigMapperImpl) ToValueObject(model *models.VLESSConfigModel) (*vo.VLESSConfig, error) {
	if model == nil {
		return nil, nil
	}

	config, err := vo.NewVLESSConfig(
		model.TransportType,
		model.Flow,
		model.Security,
		model.SNI,
		model.Fingerprint,
		model.AllowInsecure,
		model.Host,
		model.Path,
		model.ServiceName,
		model.PublicKey,
		model.ShortID,
		model.SpiderX,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create VLESS config value object: %w", err)
	}

	return &config, nil
}

// ToModel converts a domain value object to a persistence model
func (m *VLESSConfigMapperImpl) ToModel(nodeID uint, config *vo.VLESSConfig) (*models.VLESSConfigModel, error) {
	if config == nil {
		return nil, nil
	}

	return &models.VLESSConfigModel{
		NodeID:        nodeID,
		TransportType: config.TransportType(),
		Flow:          config.Flow(),
		Security:      config.Security(),
		SNI:           config.SNI(),
		Fingerprint:   config.Fingerprint(),
		AllowInsecure: config.AllowInsecure(),
		Host:          config.Host(),
		Path:          config.Path(),
		ServiceName:   config.ServiceName(),
		PublicKey:     config.PublicKey(),
		ShortID:       config.ShortID(),
		SpiderX:       config.SpiderX(),
	}, nil
}

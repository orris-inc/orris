package mappers

import (
	"fmt"

	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// VMessConfigMapper handles the conversion between VMessConfig value objects and persistence models
type VMessConfigMapper interface {
	// ToValueObject converts a persistence model to a domain value object
	// UUID is passed separately as it's derived from subscription, not stored in DB
	ToValueObject(model *models.VMessConfigModel, uuid string) (*vo.VMessConfig, error)

	// ToModel converts a domain value object to a persistence model
	ToModel(nodeID uint, config *vo.VMessConfig) (*models.VMessConfigModel, error)
}

// VMessConfigMapperImpl is the concrete implementation of VMessConfigMapper
type VMessConfigMapperImpl struct{}

// NewVMessConfigMapper creates a new VMess config mapper
func NewVMessConfigMapper() VMessConfigMapper {
	return &VMessConfigMapperImpl{}
}

// ToValueObject converts a persistence model to a domain value object
// UUID is passed separately as it's derived from subscription UUID, not stored in DB
func (m *VMessConfigMapperImpl) ToValueObject(model *models.VMessConfigModel, uuid string) (*vo.VMessConfig, error) {
	if model == nil {
		return nil, nil
	}

	// UUID is not used in the value object construction
	// It will be used when generating URI
	config, err := vo.NewVMessConfig(
		model.AlterID,
		model.Security,
		model.TransportType,
		model.Host,
		model.Path,
		model.ServiceName,
		model.TLS,
		model.SNI,
		model.AllowInsecure,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create vmess config value object: %w", err)
	}

	return &config, nil
}

// ToModel converts a domain value object to a persistence model
func (m *VMessConfigMapperImpl) ToModel(nodeID uint, config *vo.VMessConfig) (*models.VMessConfigModel, error) {
	if config == nil {
		return nil, nil
	}

	return &models.VMessConfigModel{
		NodeID:        nodeID,
		AlterID:       config.AlterID(),
		Security:      config.Security(),
		TransportType: config.TransportType(),
		Host:          config.Host(),
		Path:          config.Path(),
		ServiceName:   config.ServiceName(),
		TLS:           config.TLS(),
		SNI:           config.SNI(),
		AllowInsecure: config.AllowInsecure(),
	}, nil
}

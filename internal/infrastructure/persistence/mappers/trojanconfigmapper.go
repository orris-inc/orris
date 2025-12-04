package mappers

import (
	"fmt"

	vo "github.com/orris-inc/orris/internal/domain/node/value_objects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// TrojanConfigMapper handles the conversion between TrojanConfig value objects and persistence models
type TrojanConfigMapper interface {
	// ToValueObject converts a persistence model to a domain value object
	ToValueObject(model *models.TrojanConfigModel, password string) (*vo.TrojanConfig, error)

	// ToModel converts a domain value object to a persistence model
	ToModel(nodeID uint, config *vo.TrojanConfig) (*models.TrojanConfigModel, error)
}

// TrojanConfigMapperImpl is the concrete implementation of TrojanConfigMapper
type TrojanConfigMapperImpl struct{}

// NewTrojanConfigMapper creates a new trojan config mapper
func NewTrojanConfigMapper() TrojanConfigMapper {
	return &TrojanConfigMapperImpl{}
}

// ToValueObject converts a persistence model to a domain value object
// Password is passed separately as it's derived from subscription UUID, not stored in DB
func (m *TrojanConfigMapperImpl) ToValueObject(model *models.TrojanConfigModel, password string) (*vo.TrojanConfig, error) {
	if model == nil {
		return nil, nil
	}

	// Use placeholder password if not provided (for node entity reconstruction)
	if password == "" {
		password = "placeholder"
	}

	config, err := vo.NewTrojanConfig(
		password,
		model.TransportProtocol,
		model.Host,
		model.Path,
		model.AllowInsecure,
		model.SNI,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trojan config value object: %w", err)
	}

	return &config, nil
}

// ToModel converts a domain value object to a persistence model
func (m *TrojanConfigMapperImpl) ToModel(nodeID uint, config *vo.TrojanConfig) (*models.TrojanConfigModel, error) {
	if config == nil {
		return nil, nil
	}

	return &models.TrojanConfigModel{
		NodeID:            nodeID,
		TransportProtocol: config.TransportProtocol(),
		Host:              config.Host(),
		Path:              config.Path(),
		SNI:               config.SNI(),
		AllowInsecure:     config.AllowInsecure(),
	}, nil
}

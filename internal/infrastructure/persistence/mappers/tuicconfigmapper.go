package mappers

import (
	"fmt"

	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// TUICConfigMapper handles the conversion between TUICConfig value objects and persistence models
type TUICConfigMapper interface {
	// ToValueObject converts a persistence model to a domain value object
	ToValueObject(model *models.TUICConfigModel, uuid, password string) (*vo.TUICConfig, error)

	// ToModel converts a domain value object to a persistence model
	ToModel(nodeID uint, config *vo.TUICConfig) (*models.TUICConfigModel, error)
}

// TUICConfigMapperImpl is the concrete implementation of TUICConfigMapper
type TUICConfigMapperImpl struct{}

// NewTUICConfigMapper creates a new TUIC config mapper
func NewTUICConfigMapper() TUICConfigMapper {
	return &TUICConfigMapperImpl{}
}

// ToValueObject converts a persistence model to a domain value object
// UUID and password are passed separately as they're derived from subscription, not stored in DB
func (m *TUICConfigMapperImpl) ToValueObject(model *models.TUICConfigModel, uuid, password string) (*vo.TUICConfig, error) {
	if model == nil {
		return nil, nil
	}

	// Use placeholder values if not provided (for node entity reconstruction)
	if uuid == "" {
		uuid = PlaceholderUUID
	}
	if password == "" {
		password = PlaceholderPassword
	}

	config, err := vo.NewTUICConfig(
		uuid,
		password,
		model.CongestionControl,
		model.UDPRelayMode,
		model.ALPN,
		model.SNI,
		model.AllowInsecure,
		model.DisableSNI,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create TUIC config value object: %w", err)
	}

	return &config, nil
}

// ToModel converts a domain value object to a persistence model
// Note: UUID and password are not stored in the database
func (m *TUICConfigMapperImpl) ToModel(nodeID uint, config *vo.TUICConfig) (*models.TUICConfigModel, error) {
	if config == nil {
		return nil, nil
	}

	return &models.TUICConfigModel{
		NodeID:            nodeID,
		CongestionControl: config.CongestionControl(),
		UDPRelayMode:      config.UDPRelayMode(),
		ALPN:              config.ALPN(),
		SNI:               config.SNI(),
		AllowInsecure:     config.AllowInsecure(),
		DisableSNI:        config.DisableSNI(),
	}, nil
}

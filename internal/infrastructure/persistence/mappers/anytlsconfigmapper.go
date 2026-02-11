package mappers

import (
	"fmt"

	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// AnyTLSConfigMapper handles the conversion between AnyTLSConfig value objects and persistence models
type AnyTLSConfigMapper interface {
	// ToValueObject converts a persistence model to a domain value object
	ToValueObject(model *models.AnyTLSConfigModel, password string) (*vo.AnyTLSConfig, error)

	// ToModel converts a domain value object to a persistence model
	ToModel(nodeID uint, config *vo.AnyTLSConfig) (*models.AnyTLSConfigModel, error)
}

// AnyTLSConfigMapperImpl is the concrete implementation of AnyTLSConfigMapper
type AnyTLSConfigMapperImpl struct{}

// NewAnyTLSConfigMapper creates a new AnyTLS config mapper
func NewAnyTLSConfigMapper() AnyTLSConfigMapper {
	return &AnyTLSConfigMapperImpl{}
}

// ToValueObject converts a persistence model to a domain value object
// Password is passed separately as it's derived from subscription UUID, not stored in DB
func (m *AnyTLSConfigMapperImpl) ToValueObject(model *models.AnyTLSConfigModel, password string) (*vo.AnyTLSConfig, error) {
	if model == nil {
		return nil, nil
	}

	// Use placeholder password if not provided (for node entity reconstruction)
	if password == "" {
		password = PlaceholderPassword
	}

	config, err := vo.NewAnyTLSConfig(
		password,
		model.SNI,
		model.AllowInsecure,
		model.Fingerprint,
		model.IdleSessionCheckInterval,
		model.IdleSessionTimeout,
		model.MinIdleSession,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create anytls config value object: %w", err)
	}

	return &config, nil
}

// ToModel converts a domain value object to a persistence model
func (m *AnyTLSConfigMapperImpl) ToModel(nodeID uint, config *vo.AnyTLSConfig) (*models.AnyTLSConfigModel, error) {
	if config == nil {
		return nil, nil
	}

	return &models.AnyTLSConfigModel{
		NodeID:                   nodeID,
		SNI:                      config.SNI(),
		AllowInsecure:            config.AllowInsecure(),
		Fingerprint:              config.Fingerprint(),
		IdleSessionCheckInterval: config.IdleSessionCheckInterval(),
		IdleSessionTimeout:       config.IdleSessionTimeout(),
		MinIdleSession:           config.MinIdleSession(),
	}, nil
}

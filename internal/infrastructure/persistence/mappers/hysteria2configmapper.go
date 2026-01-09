package mappers

import (
	"fmt"
	"math"

	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// Hysteria2ConfigMapper handles the conversion between Hysteria2Config value objects and persistence models
type Hysteria2ConfigMapper interface {
	// ToValueObject converts a persistence model to a domain value object
	ToValueObject(model *models.Hysteria2ConfigModel, password string) (*vo.Hysteria2Config, error)

	// ToModel converts a domain value object to a persistence model
	ToModel(nodeID uint, config *vo.Hysteria2Config) (*models.Hysteria2ConfigModel, error)
}

// Hysteria2ConfigMapperImpl is the concrete implementation of Hysteria2ConfigMapper
type Hysteria2ConfigMapperImpl struct{}

// NewHysteria2ConfigMapper creates a new hysteria2 config mapper
func NewHysteria2ConfigMapper() Hysteria2ConfigMapper {
	return &Hysteria2ConfigMapperImpl{}
}

// ToValueObject converts a persistence model to a domain value object
// Password is passed separately as it's derived from subscription UUID, not stored in DB
func (m *Hysteria2ConfigMapperImpl) ToValueObject(model *models.Hysteria2ConfigModel, password string) (*vo.Hysteria2Config, error) {
	if model == nil {
		return nil, nil
	}

	// Use placeholder password if not provided (for node entity reconstruction)
	if password == "" {
		password = placeholderPassword
	}

	// Convert uint* to int* for bandwidth limits with overflow check
	var upMbps, downMbps *int
	if model.UpMbps != nil {
		if *model.UpMbps > uint(math.MaxInt) {
			return nil, fmt.Errorf("up_mbps value %d exceeds maximum safe integer value", *model.UpMbps)
		}
		v := int(*model.UpMbps)
		upMbps = &v
	}
	if model.DownMbps != nil {
		if *model.DownMbps > uint(math.MaxInt) {
			return nil, fmt.Errorf("down_mbps value %d exceeds maximum safe integer value", *model.DownMbps)
		}
		v := int(*model.DownMbps)
		downMbps = &v
	}

	config, err := vo.NewHysteria2Config(
		password,
		model.CongestionControl,
		model.Obfs,
		model.ObfsPassword,
		upMbps,
		downMbps,
		model.SNI,
		model.AllowInsecure,
		model.Fingerprint,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create hysteria2 config value object: %w", err)
	}

	return &config, nil
}

// ToModel converts a domain value object to a persistence model
func (m *Hysteria2ConfigMapperImpl) ToModel(nodeID uint, config *vo.Hysteria2Config) (*models.Hysteria2ConfigModel, error) {
	if config == nil {
		return nil, nil
	}

	// Convert int* to uint* for bandwidth limits with negative value check
	var upMbps, downMbps *uint
	if config.UpMbps() != nil {
		if *config.UpMbps() < 0 {
			return nil, fmt.Errorf("up_mbps value %d cannot be negative", *config.UpMbps())
		}
		v := uint(*config.UpMbps())
		upMbps = &v
	}
	if config.DownMbps() != nil {
		if *config.DownMbps() < 0 {
			return nil, fmt.Errorf("down_mbps value %d cannot be negative", *config.DownMbps())
		}
		v := uint(*config.DownMbps())
		downMbps = &v
	}

	return &models.Hysteria2ConfigModel{
		NodeID:            nodeID,
		CongestionControl: config.CongestionControl(),
		Obfs:              config.Obfs(),
		ObfsPassword:      config.ObfsPassword(),
		UpMbps:            upMbps,
		DownMbps:          downMbps,
		SNI:               config.SNI(),
		AllowInsecure:     config.AllowInsecure(),
		Fingerprint:       config.Fingerprint(),
	}, nil
}

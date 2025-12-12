package mappers

import (
	"encoding/json"
	"fmt"

	"gorm.io/datatypes"

	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// ShadowsocksConfigMapper handles the conversion between Shadowsocks config value objects and persistence models
type ShadowsocksConfigMapper interface {
	// ToValueObjects converts a persistence model to domain value objects (EncryptionConfig + PluginConfig)
	ToValueObjects(model *models.ShadowsocksConfigModel) (vo.EncryptionConfig, *vo.PluginConfig, error)

	// ToModel converts domain value objects to a persistence model
	ToModel(nodeID uint, encryptionConfig vo.EncryptionConfig, pluginConfig *vo.PluginConfig) (*models.ShadowsocksConfigModel, error)
}

// ShadowsocksConfigMapperImpl is the concrete implementation of ShadowsocksConfigMapper
type ShadowsocksConfigMapperImpl struct{}

// NewShadowsocksConfigMapper creates a new shadowsocks config mapper
func NewShadowsocksConfigMapper() ShadowsocksConfigMapper {
	return &ShadowsocksConfigMapperImpl{}
}

// ToValueObjects converts a persistence model to domain value objects
func (m *ShadowsocksConfigMapperImpl) ToValueObjects(model *models.ShadowsocksConfigModel) (vo.EncryptionConfig, *vo.PluginConfig, error) {
	if model == nil {
		return vo.EncryptionConfig{}, nil, nil
	}

	// Create EncryptionConfig
	encryptionConfig, err := vo.NewEncryptionConfig(model.EncryptionMethod)
	if err != nil {
		return vo.EncryptionConfig{}, nil, fmt.Errorf("failed to create encryption config: %w", err)
	}

	// Create PluginConfig if present
	var pluginConfig *vo.PluginConfig
	if model.Plugin != nil && *model.Plugin != "" {
		var opts map[string]string
		if model.PluginOpts != nil {
			if err := json.Unmarshal(model.PluginOpts, &opts); err != nil {
				return vo.EncryptionConfig{}, nil, fmt.Errorf("failed to unmarshal plugin opts: %w", err)
			}
		}
		pluginConfig, err = vo.NewPluginConfig(*model.Plugin, opts)
		if err != nil {
			return vo.EncryptionConfig{}, nil, fmt.Errorf("failed to create plugin config: %w", err)
		}
	}

	return encryptionConfig, pluginConfig, nil
}

// ToModel converts domain value objects to a persistence model
func (m *ShadowsocksConfigMapperImpl) ToModel(nodeID uint, encryptionConfig vo.EncryptionConfig, pluginConfig *vo.PluginConfig) (*models.ShadowsocksConfigModel, error) {
	model := &models.ShadowsocksConfigModel{
		NodeID:           nodeID,
		EncryptionMethod: encryptionConfig.Method(),
	}

	// Handle plugin config
	if pluginConfig != nil {
		pluginName := pluginConfig.Plugin()
		model.Plugin = &pluginName

		opts := pluginConfig.Opts()
		if len(opts) > 0 {
			optsBytes, err := json.Marshal(opts)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal plugin opts: %w", err)
			}
			model.PluginOpts = datatypes.JSON(optsBytes)
		}
	}

	return model, nil
}

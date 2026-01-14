package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/node"
	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// allowedNodeOrderByFields defines the whitelist of allowed ORDER BY fields
// to prevent SQL injection attacks.
var allowedNodeOrderByFields = map[string]bool{
	"id":             true,
	"sid":            true,
	"name":           true,
	"server_address": true,
	"agent_port":     true,
	"protocol":       true,
	"status":         true,
	"user_id":        true,
	"region":         true,
	"sort_order":     true,
	"last_seen_at":   true,
	"created_at":     true,
	"updated_at":     true,
}

// NodeRepositoryImpl implements the node.NodeRepository interface
type NodeRepositoryImpl struct {
	db                    *gorm.DB
	mapper                mappers.NodeMapper
	trojanConfigRepo      *TrojanConfigRepository
	shadowsocksConfigRepo *ShadowsocksConfigRepository
	vlessConfigRepo       *VLESSConfigRepository
	vmessConfigRepo       *VMessConfigRepository
	hysteria2ConfigRepo   *Hysteria2ConfigRepository
	tuicConfigRepo        *TUICConfigRepository
	logger                logger.Interface
}

// NewNodeRepository creates a new node repository instance
func NewNodeRepository(db *gorm.DB, logger logger.Interface) node.NodeRepository {
	return &NodeRepositoryImpl{
		db:                    db,
		mapper:                mappers.NewNodeMapper(),
		trojanConfigRepo:      NewTrojanConfigRepository(db, logger),
		shadowsocksConfigRepo: NewShadowsocksConfigRepository(db, logger),
		vlessConfigRepo:       NewVLESSConfigRepository(db, logger),
		vmessConfigRepo:       NewVMessConfigRepository(db, logger),
		hysteria2ConfigRepo:   NewHysteria2ConfigRepository(db, logger),
		tuicConfigRepo:        NewTUICConfigRepository(db, logger),
		logger:                logger,
	}
}

// Create creates a new node in the database
// Uses transaction to ensure node and protocol-specific configs are created atomically
func (r *NodeRepositoryImpl) Create(ctx context.Context, nodeEntity *node.Node) error {
	model, err := r.mapper.ToModel(nodeEntity)
	if err != nil {
		r.logger.Errorw("failed to map node entity to model", "error", err)
		return fmt.Errorf("failed to map node entity: %w", err)
	}

	// Use transaction to create node and protocol config atomically
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create node
		if err := tx.Create(model).Error; err != nil {
			if strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "duplicate key") {
				if strings.Contains(err.Error(), "name") {
					return errors.NewConflictError("node with this name already exists")
				}
				if strings.Contains(err.Error(), "token_hash") {
					return errors.NewConflictError("node with this token already exists")
				}
				return errors.NewConflictError("node already exists")
			}
			return fmt.Errorf("failed to create node: %w", err)
		}

		// Create protocol-specific config based on protocol type
		switch nodeEntity.Protocol() {
		case vo.ProtocolShadowsocks:
			if err := r.shadowsocksConfigRepo.CreateInTx(tx, model.ID, nodeEntity.EncryptionConfig(), nodeEntity.PluginConfig()); err != nil {
				return fmt.Errorf("failed to create shadowsocks config: %w", err)
			}
		case vo.ProtocolTrojan:
			if nodeEntity.TrojanConfig() != nil {
				if err := r.trojanConfigRepo.CreateInTx(tx, model.ID, nodeEntity.TrojanConfig()); err != nil {
					return fmt.Errorf("failed to create trojan config: %w", err)
				}
			}
		case vo.ProtocolVLESS:
			if nodeEntity.VLESSConfig() != nil {
				if err := r.vlessConfigRepo.CreateInTx(tx, model.ID, nodeEntity.VLESSConfig()); err != nil {
					return fmt.Errorf("failed to create vless config: %w", err)
				}
			}
		case vo.ProtocolVMess:
			if nodeEntity.VMessConfig() != nil {
				if err := r.vmessConfigRepo.CreateInTx(tx, model.ID, nodeEntity.VMessConfig()); err != nil {
					return fmt.Errorf("failed to create vmess config: %w", err)
				}
			}
		case vo.ProtocolHysteria2:
			if nodeEntity.Hysteria2Config() != nil {
				if err := r.hysteria2ConfigRepo.CreateInTx(tx, model.ID, nodeEntity.Hysteria2Config()); err != nil {
					return fmt.Errorf("failed to create hysteria2 config: %w", err)
				}
			}
		case vo.ProtocolTUIC:
			if nodeEntity.TUICConfig() != nil {
				if err := r.tuicConfigRepo.CreateInTx(tx, model.ID, nodeEntity.TUICConfig()); err != nil {
					return fmt.Errorf("failed to create tuic config: %w", err)
				}
			}
		}

		return nil
	})

	if err != nil {
		r.logger.Errorw("failed to create node in database", "error", err)
		return err
	}

	if err := nodeEntity.SetID(model.ID); err != nil {
		r.logger.Errorw("failed to set node ID", "error", err)
		return fmt.Errorf("failed to set node ID: %w", err)
	}

	r.logger.Infow("node created successfully", "id", model.ID, "name", model.Name)
	return nil
}

// GetByID retrieves a node by its ID
func (r *NodeRepositoryImpl) GetByID(ctx context.Context, id uint) (*node.Node, error) {
	var model models.NodeModel

	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("node not found")
		}
		r.logger.Errorw("failed to get node by ID", "id", id, "error", err)
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	// Load protocol-specific config
	var trojanConfig *vo.TrojanConfig
	var encryptionConfig vo.EncryptionConfig
	var pluginConfig *vo.PluginConfig
	var vlessConfig *vo.VLESSConfig
	var vmessConfig *vo.VMessConfig
	var hysteria2Config *vo.Hysteria2Config
	var tuicConfig *vo.TUICConfig

	switch model.Protocol {
	case "shadowsocks":
		var err error
		encryptionConfig, pluginConfig, err = r.shadowsocksConfigRepo.GetByNodeID(ctx, id)
		if err != nil {
			r.logger.Errorw("failed to get shadowsocks config", "node_id", id, "error", err)
			return nil, fmt.Errorf("failed to get shadowsocks config: %w", err)
		}
	case "trojan":
		var err error
		trojanConfig, err = r.trojanConfigRepo.GetByNodeID(ctx, id)
		if err != nil {
			r.logger.Errorw("failed to get trojan config", "node_id", id, "error", err)
			return nil, fmt.Errorf("failed to get trojan config: %w", err)
		}
	case "vless":
		var err error
		vlessConfig, err = r.vlessConfigRepo.GetByNodeID(ctx, id)
		if err != nil {
			r.logger.Errorw("failed to get vless config", "node_id", id, "error", err)
			return nil, fmt.Errorf("failed to get vless config: %w", err)
		}
	case "vmess":
		var err error
		vmessConfig, err = r.vmessConfigRepo.GetByNodeID(ctx, id)
		if err != nil {
			r.logger.Errorw("failed to get vmess config", "node_id", id, "error", err)
			return nil, fmt.Errorf("failed to get vmess config: %w", err)
		}
	case "hysteria2":
		var err error
		hysteria2Config, err = r.hysteria2ConfigRepo.GetByNodeID(ctx, id)
		if err != nil {
			r.logger.Errorw("failed to get hysteria2 config", "node_id", id, "error", err)
			return nil, fmt.Errorf("failed to get hysteria2 config: %w", err)
		}
	case "tuic":
		var err error
		tuicConfig, err = r.tuicConfigRepo.GetByNodeID(ctx, id)
		if err != nil {
			r.logger.Errorw("failed to get tuic config", "node_id", id, "error", err)
			return nil, fmt.Errorf("failed to get tuic config: %w", err)
		}
	}

	entity, err := r.mapper.ToEntity(&model, encryptionConfig, pluginConfig, trojanConfig, vlessConfig, vmessConfig, hysteria2Config, tuicConfig)
	if err != nil {
		r.logger.Errorw("failed to map node model to entity", "id", id, "error", err)
		return nil, fmt.Errorf("failed to map node: %w", err)
	}

	return entity, nil
}

// GetBySID retrieves a node by its SID
func (r *NodeRepositoryImpl) GetBySID(ctx context.Context, sid string) (*node.Node, error) {
	var model models.NodeModel

	if err := r.db.WithContext(ctx).Where("sid = ?", sid).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("node not found")
		}
		r.logger.Errorw("failed to get node by SID", "sid", sid, "error", err)
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	// Load protocol-specific config
	var trojanConfig *vo.TrojanConfig
	var encryptionConfig vo.EncryptionConfig
	var pluginConfig *vo.PluginConfig
	var vlessConfig *vo.VLESSConfig
	var vmessConfig *vo.VMessConfig
	var hysteria2Config *vo.Hysteria2Config
	var tuicConfig *vo.TUICConfig

	switch model.Protocol {
	case "shadowsocks":
		var err error
		encryptionConfig, pluginConfig, err = r.shadowsocksConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get shadowsocks config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get shadowsocks config: %w", err)
		}
	case "trojan":
		var err error
		trojanConfig, err = r.trojanConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get trojan config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get trojan config: %w", err)
		}
	case "vless":
		var err error
		vlessConfig, err = r.vlessConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get vless config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get vless config: %w", err)
		}
	case "vmess":
		var err error
		vmessConfig, err = r.vmessConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get vmess config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get vmess config: %w", err)
		}
	case "hysteria2":
		var err error
		hysteria2Config, err = r.hysteria2ConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get hysteria2 config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get hysteria2 config: %w", err)
		}
	case "tuic":
		var err error
		tuicConfig, err = r.tuicConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get tuic config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get tuic config: %w", err)
		}
	}

	entity, err := r.mapper.ToEntity(&model, encryptionConfig, pluginConfig, trojanConfig, vlessConfig, vmessConfig, hysteria2Config, tuicConfig)
	if err != nil {
		r.logger.Errorw("failed to map node model to entity", "sid", sid, "error", err)
		return nil, fmt.Errorf("failed to map node: %w", err)
	}

	return entity, nil
}

// GetBySIDs retrieves nodes by their SIDs
func (r *NodeRepositoryImpl) GetBySIDs(ctx context.Context, sids []string) ([]*node.Node, error) {
	if len(sids) == 0 {
		return []*node.Node{}, nil
	}

	var nodeModels []*models.NodeModel
	if err := r.db.WithContext(ctx).Where("sid IN ?", sids).Find(&nodeModels).Error; err != nil {
		r.logger.Errorw("failed to get nodes by SIDs", "sids", sids, "error", err)
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	// Collect node IDs for batch loading protocol configs
	nodeIDs := make([]uint, len(nodeModels))
	for i, m := range nodeModels {
		nodeIDs[i] = m.ID
	}

	// Load shadowsocks configs
	ssConfigsRaw, err := r.shadowsocksConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load shadowsocks configs", "error", err)
		ssConfigsRaw = make(map[uint]*ShadowsocksConfigData)
	}

	// Convert to mapper format
	ssConfigs := make(map[uint]*mappers.ShadowsocksConfigData)
	for nodeID, data := range ssConfigsRaw {
		ssConfigs[nodeID] = &mappers.ShadowsocksConfigData{
			EncryptionConfig: data.EncryptionConfig,
			PluginConfig:     data.PluginConfig,
		}
	}

	// Load trojan configs
	trojanConfigs, err := r.trojanConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load trojan configs", "error", err)
		trojanConfigs = make(map[uint]*vo.TrojanConfig)
	}

	// Load vless configs
	vlessConfigs, err := r.vlessConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load vless configs", "error", err)
		vlessConfigs = make(map[uint]*vo.VLESSConfig)
	}

	// Load vmess configs
	vmessConfigs, err := r.vmessConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load vmess configs", "error", err)
		vmessConfigs = make(map[uint]*vo.VMessConfig)
	}

	// Load hysteria2 configs
	hysteria2Configs, err := r.hysteria2ConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load hysteria2 configs", "error", err)
		hysteria2Configs = make(map[uint]*vo.Hysteria2Config)
	}

	// Load tuic configs
	tuicConfigs, err := r.tuicConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load tuic configs", "error", err)
		tuicConfigs = make(map[uint]*vo.TUICConfig)
	}

	// Convert to entities
	entities, err := r.mapper.ToEntities(nodeModels, ssConfigs, trojanConfigs, vlessConfigs, vmessConfigs, hysteria2Configs, tuicConfigs)
	if err != nil {
		r.logger.Errorw("failed to map node models to entities", "error", err)
		return nil, fmt.Errorf("failed to map nodes: %w", err)
	}

	return entities, nil
}

// GetByIDs retrieves nodes by their IDs
func (r *NodeRepositoryImpl) GetByIDs(ctx context.Context, ids []uint) ([]*node.Node, error) {
	if len(ids) == 0 {
		return []*node.Node{}, nil
	}

	var nodeModels []*models.NodeModel
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&nodeModels).Error; err != nil {
		r.logger.Errorw("failed to get nodes by IDs", "ids", ids, "error", err)
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	// Collect node IDs for batch loading protocol configs
	nodeIDs := make([]uint, len(nodeModels))
	for i, m := range nodeModels {
		nodeIDs[i] = m.ID
	}

	// Load shadowsocks configs
	ssConfigsRaw, err := r.shadowsocksConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load shadowsocks configs", "error", err)
		ssConfigsRaw = make(map[uint]*ShadowsocksConfigData)
	}

	// Convert to mapper format
	ssConfigs := make(map[uint]*mappers.ShadowsocksConfigData)
	for nodeID, data := range ssConfigsRaw {
		ssConfigs[nodeID] = &mappers.ShadowsocksConfigData{
			EncryptionConfig: data.EncryptionConfig,
			PluginConfig:     data.PluginConfig,
		}
	}

	// Load trojan configs
	trojanConfigs, err := r.trojanConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load trojan configs", "error", err)
		trojanConfigs = make(map[uint]*vo.TrojanConfig)
	}

	// Load vless configs
	vlessConfigs, err := r.vlessConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load vless configs", "error", err)
		vlessConfigs = make(map[uint]*vo.VLESSConfig)
	}

	// Load vmess configs
	vmessConfigs, err := r.vmessConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load vmess configs", "error", err)
		vmessConfigs = make(map[uint]*vo.VMessConfig)
	}

	// Load hysteria2 configs
	hysteria2Configs, err := r.hysteria2ConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load hysteria2 configs", "error", err)
		hysteria2Configs = make(map[uint]*vo.Hysteria2Config)
	}

	// Load tuic configs
	tuicConfigs, err := r.tuicConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load tuic configs", "error", err)
		tuicConfigs = make(map[uint]*vo.TUICConfig)
	}

	// Convert to entities
	entities, err := r.mapper.ToEntities(nodeModels, ssConfigs, trojanConfigs, vlessConfigs, vmessConfigs, hysteria2Configs, tuicConfigs)
	if err != nil {
		r.logger.Errorw("failed to map node models to entities", "error", err)
		return nil, fmt.Errorf("failed to map nodes: %w", err)
	}

	return entities, nil
}

// GetByToken retrieves a node by its API token hash
func (r *NodeRepositoryImpl) GetByToken(ctx context.Context, tokenHash string) (*node.Node, error) {
	var model models.NodeModel

	if err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("node not found")
		}
		r.logger.Errorw("failed to get node by token", "error", err)
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	// Load protocol-specific config
	var trojanConfig *vo.TrojanConfig
	var encryptionConfig vo.EncryptionConfig
	var pluginConfig *vo.PluginConfig
	var vlessConfig *vo.VLESSConfig
	var vmessConfig *vo.VMessConfig
	var hysteria2Config *vo.Hysteria2Config
	var tuicConfig *vo.TUICConfig

	switch model.Protocol {
	case "shadowsocks":
		var err error
		encryptionConfig, pluginConfig, err = r.shadowsocksConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get shadowsocks config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get shadowsocks config: %w", err)
		}
	case "trojan":
		var err error
		trojanConfig, err = r.trojanConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get trojan config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get trojan config: %w", err)
		}
	case "vless":
		var err error
		vlessConfig, err = r.vlessConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get vless config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get vless config: %w", err)
		}
	case "vmess":
		var err error
		vmessConfig, err = r.vmessConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get vmess config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get vmess config: %w", err)
		}
	case "hysteria2":
		var err error
		hysteria2Config, err = r.hysteria2ConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get hysteria2 config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get hysteria2 config: %w", err)
		}
	case "tuic":
		var err error
		tuicConfig, err = r.tuicConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get tuic config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get tuic config: %w", err)
		}
	}

	entity, err := r.mapper.ToEntity(&model, encryptionConfig, pluginConfig, trojanConfig, vlessConfig, vmessConfig, hysteria2Config, tuicConfig)
	if err != nil {
		r.logger.Errorw("failed to map node model to entity", "token_hash", tokenHash, "error", err)
		return nil, fmt.Errorf("failed to map node: %w", err)
	}

	return entity, nil
}

// Update updates an existing node with optimistic locking
// Uses transaction to ensure node and protocol-specific configs are updated atomically
func (r *NodeRepositoryImpl) Update(ctx context.Context, nodeEntity *node.Node) error {
	model, err := r.mapper.ToModel(nodeEntity)
	if err != nil {
		r.logger.Errorw("failed to map node entity to model", "error", err)
		return fmt.Errorf("failed to map node entity: %w", err)
	}

	// Use the original version from when the entity was loaded for optimistic locking.
	// This handles the case where multiple properties are updated in one operation,
	// each incrementing the domain version, but we need to check against the DB version.
	expectedVersion := nodeEntity.OriginalVersion()
	if expectedVersion < 1 {
		expectedVersion = 1
	}

	// Use transaction to update node and protocol config atomically
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Use Select to explicitly specify fields to update, including nullable fields like group_ids
		// This ensures GORM updates NULL values correctly (without Select, GORM ignores nil values in map)
		// Use optimistic locking: WHERE id = ? AND version = expectedVersion
		result := tx.Model(&models.NodeModel{}).
			Where("id = ? AND version = ?", model.ID, expectedVersion).
			Select(
				"name", "server_address", "agent_port", "subscription_port",
				"protocol", "status", "region", "tags", "sort_order",
				"maintenance_reason", "token_hash", "api_token", "group_ids", "route_config", "mute_notification", "version", "updated_at",
			).
			Updates(model)

		if result.Error != nil {
			if strings.Contains(result.Error.Error(), "Duplicate entry") || strings.Contains(result.Error.Error(), "duplicate key") {
				if strings.Contains(result.Error.Error(), "name") {
					return errors.NewConflictError("node with this name already exists")
				}
				return errors.NewConflictError("node already exists")
			}
			return fmt.Errorf("failed to update node: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			// Check if the record exists to distinguish between not found and version conflict
			var count int64
			if err := tx.Model(&models.NodeModel{}).Where("id = ?", model.ID).Count(&count).Error; err == nil && count > 0 {
				return errors.NewConflictError("node was modified by another request, please retry")
			}
			return errors.NewNotFoundError("node not found", fmt.Sprintf("id=%d", model.ID))
		}

		// Update protocol-specific config based on protocol type
		// Delete all other protocol configs when updating (handles protocol change)
		switch nodeEntity.Protocol() {
		case vo.ProtocolShadowsocks:
			if err := r.shadowsocksConfigRepo.UpdateInTx(tx, model.ID, nodeEntity.EncryptionConfig(), nodeEntity.PluginConfig()); err != nil {
				return fmt.Errorf("failed to update shadowsocks config: %w", err)
			}
			// Delete other protocol configs if they exist (protocol changed)
			if err := r.trojanConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete trojan config: %w", err)
			}
			if err := r.vlessConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete vless config: %w", err)
			}
			if err := r.vmessConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete vmess config: %w", err)
			}
			if err := r.hysteria2ConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete hysteria2 config: %w", err)
			}
			if err := r.tuicConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete tuic config: %w", err)
			}
		case vo.ProtocolTrojan:
			if err := r.trojanConfigRepo.UpdateInTx(tx, model.ID, nodeEntity.TrojanConfig()); err != nil {
				return fmt.Errorf("failed to update trojan config: %w", err)
			}
			// Delete other protocol configs if they exist (protocol changed)
			if err := r.shadowsocksConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete shadowsocks config: %w", err)
			}
			if err := r.vlessConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete vless config: %w", err)
			}
			if err := r.vmessConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete vmess config: %w", err)
			}
			if err := r.hysteria2ConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete hysteria2 config: %w", err)
			}
			if err := r.tuicConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete tuic config: %w", err)
			}
		case vo.ProtocolVLESS:
			if err := r.vlessConfigRepo.UpdateInTx(tx, model.ID, nodeEntity.VLESSConfig()); err != nil {
				return fmt.Errorf("failed to update vless config: %w", err)
			}
			// Delete other protocol configs if they exist (protocol changed)
			if err := r.shadowsocksConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete shadowsocks config: %w", err)
			}
			if err := r.trojanConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete trojan config: %w", err)
			}
			if err := r.vmessConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete vmess config: %w", err)
			}
			if err := r.hysteria2ConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete hysteria2 config: %w", err)
			}
			if err := r.tuicConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete tuic config: %w", err)
			}
		case vo.ProtocolVMess:
			if err := r.vmessConfigRepo.UpdateInTx(tx, model.ID, nodeEntity.VMessConfig()); err != nil {
				return fmt.Errorf("failed to update vmess config: %w", err)
			}
			// Delete other protocol configs if they exist (protocol changed)
			if err := r.shadowsocksConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete shadowsocks config: %w", err)
			}
			if err := r.trojanConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete trojan config: %w", err)
			}
			if err := r.vlessConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete vless config: %w", err)
			}
			if err := r.hysteria2ConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete hysteria2 config: %w", err)
			}
			if err := r.tuicConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete tuic config: %w", err)
			}
		case vo.ProtocolHysteria2:
			if err := r.hysteria2ConfigRepo.UpdateInTx(tx, model.ID, nodeEntity.Hysteria2Config()); err != nil {
				return fmt.Errorf("failed to update hysteria2 config: %w", err)
			}
			// Delete other protocol configs if they exist (protocol changed)
			if err := r.shadowsocksConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete shadowsocks config: %w", err)
			}
			if err := r.trojanConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete trojan config: %w", err)
			}
			if err := r.vlessConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete vless config: %w", err)
			}
			if err := r.vmessConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete vmess config: %w", err)
			}
			if err := r.tuicConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete tuic config: %w", err)
			}
		case vo.ProtocolTUIC:
			if err := r.tuicConfigRepo.UpdateInTx(tx, model.ID, nodeEntity.TUICConfig()); err != nil {
				return fmt.Errorf("failed to update tuic config: %w", err)
			}
			// Delete other protocol configs if they exist (protocol changed)
			if err := r.shadowsocksConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete shadowsocks config: %w", err)
			}
			if err := r.trojanConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete trojan config: %w", err)
			}
			if err := r.vlessConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete vless config: %w", err)
			}
			if err := r.vmessConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete vmess config: %w", err)
			}
			if err := r.hysteria2ConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete hysteria2 config: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		r.logger.Errorw("failed to update node", "id", model.ID, "error", err)
		return err
	}

	r.logger.Infow("node updated successfully", "id", model.ID, "name", model.Name)
	return nil
}

// Delete permanently deletes a node and its associated protocol configs from the database.
func (r *NodeRepositoryImpl) Delete(ctx context.Context, id uint) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete protocol configs first
		if err := r.shadowsocksConfigRepo.DeleteInTx(tx, id); err != nil {
			return fmt.Errorf("failed to delete shadowsocks config: %w", err)
		}
		if err := r.trojanConfigRepo.DeleteInTx(tx, id); err != nil {
			return fmt.Errorf("failed to delete trojan config: %w", err)
		}
		if err := r.vlessConfigRepo.DeleteInTx(tx, id); err != nil {
			return fmt.Errorf("failed to delete vless config: %w", err)
		}
		if err := r.vmessConfigRepo.DeleteInTx(tx, id); err != nil {
			return fmt.Errorf("failed to delete vmess config: %w", err)
		}
		if err := r.hysteria2ConfigRepo.DeleteInTx(tx, id); err != nil {
			return fmt.Errorf("failed to delete hysteria2 config: %w", err)
		}
		if err := r.tuicConfigRepo.DeleteInTx(tx, id); err != nil {
			return fmt.Errorf("failed to delete tuic config: %w", err)
		}

		// Hard delete node using Unscoped() to bypass soft delete
		result := tx.Unscoped().Delete(&models.NodeModel{}, id)
		if result.Error != nil {
			return fmt.Errorf("failed to delete node: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			return errors.NewNotFoundError("node not found")
		}

		return nil
	})

	if err != nil {
		r.logger.Errorw("failed to delete node", "id", id, "error", err)
		return err
	}

	r.logger.Infow("node deleted successfully", "id", id)
	return nil
}

// List retrieves a paginated list of nodes with filtering
func (r *NodeRepositoryImpl) List(ctx context.Context, filter node.NodeFilter) ([]*node.Node, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.NodeModel{})

	// Apply filters
	if filter.AdminOnly != nil && *filter.AdminOnly {
		query = query.Where("user_id IS NULL")
	} else if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.Name != nil && *filter.Name != "" {
		query = query.Where("name LIKE ?", "%"+*filter.Name+"%")
	}
	if filter.Status != nil && *filter.Status != "" {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.Tag != nil && *filter.Tag != "" {
		// Search in JSON tags array using proper JSON encoding to handle special characters
		tagJSON, _ := json.Marshal(*filter.Tag)
		query = query.Where("JSON_CONTAINS(tags, ?)", string(tagJSON))
	}
	if len(filter.GroupIDs) > 0 {
		// Use JSON_OVERLAPS to check if group_ids array contains any of the filter group IDs
		// JSON_OVERLAPS returns true if two JSON arrays have at least one element in common
		// Use json.Marshal for safe JSON array construction instead of string formatting
		groupIDsJSON, _ := json.Marshal(filter.GroupIDs)
		query = query.Where("JSON_OVERLAPS(group_ids, ?)", string(groupIDsJSON))
	}

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		r.logger.Errorw("failed to count nodes", "error", err)
		return nil, 0, fmt.Errorf("failed to count nodes: %w", err)
	}

	// Apply sorting with whitelist validation to prevent SQL injection
	sortBy := strings.ToLower(filter.SortFilter.SortBy)
	if sortBy != "" && allowedNodeOrderByFields[sortBy] {
		order := "ASC"
		if filter.SortFilter.IsDescending() {
			order = "DESC"
		}
		query = query.Order(fmt.Sprintf("%s %s", sortBy, order))
	} else {
		query = query.Order("sort_order ASC, created_at DESC")
	}

	// Apply pagination
	offset := filter.PageFilter.Offset()
	limit := filter.PageFilter.Limit()
	query = query.Offset(offset).Limit(limit)

	// Execute query
	var nodeModels []*models.NodeModel
	if err := query.Find(&nodeModels).Error; err != nil {
		r.logger.Errorw("failed to list nodes", "error", err)
		return nil, 0, fmt.Errorf("failed to list nodes: %w", err)
	}

	// Collect node IDs by protocol
	var ssNodeIDs, trojanNodeIDs, vlessNodeIDs, vmessNodeIDs, hysteria2NodeIDs, tuicNodeIDs []uint
	for _, m := range nodeModels {
		switch m.Protocol {
		case "shadowsocks":
			ssNodeIDs = append(ssNodeIDs, m.ID)
		case "trojan":
			trojanNodeIDs = append(trojanNodeIDs, m.ID)
		case "vless":
			vlessNodeIDs = append(vlessNodeIDs, m.ID)
		case "vmess":
			vmessNodeIDs = append(vmessNodeIDs, m.ID)
		case "hysteria2":
			hysteria2NodeIDs = append(hysteria2NodeIDs, m.ID)
		case "tuic":
			tuicNodeIDs = append(tuicNodeIDs, m.ID)
		}
	}

	// Load protocol-specific configs in parallel
	var (
		ssConfigsRaw     map[uint]*ShadowsocksConfigData
		trojanConfigs    map[uint]*vo.TrojanConfig
		vlessConfigs     map[uint]*vo.VLESSConfig
		vmessConfigs     map[uint]*vo.VMessConfig
		hysteria2Configs map[uint]*vo.Hysteria2Config
		tuicConfigs      map[uint]*vo.TUICConfig
		mu               sync.Mutex
	)

	g, gctx := errgroup.WithContext(ctx)

	// Shadowsocks configs
	if len(ssNodeIDs) > 0 {
		g.Go(func() error {
			configs, err := r.shadowsocksConfigRepo.GetByNodeIDs(gctx, ssNodeIDs)
			if err != nil {
				r.logger.Errorw("failed to get shadowsocks configs", "error", err)
				return fmt.Errorf("failed to get shadowsocks configs: %w", err)
			}
			mu.Lock()
			ssConfigsRaw = configs
			mu.Unlock()
			return nil
		})
	}

	// Trojan configs
	if len(trojanNodeIDs) > 0 {
		g.Go(func() error {
			configs, err := r.trojanConfigRepo.GetByNodeIDs(gctx, trojanNodeIDs)
			if err != nil {
				r.logger.Errorw("failed to get trojan configs", "error", err)
				return fmt.Errorf("failed to get trojan configs: %w", err)
			}
			mu.Lock()
			trojanConfigs = configs
			mu.Unlock()
			return nil
		})
	}

	// VLESS configs
	if len(vlessNodeIDs) > 0 {
		g.Go(func() error {
			configs, err := r.vlessConfigRepo.GetByNodeIDs(gctx, vlessNodeIDs)
			if err != nil {
				r.logger.Errorw("failed to get vless configs", "error", err)
				return fmt.Errorf("failed to get vless configs: %w", err)
			}
			mu.Lock()
			vlessConfigs = configs
			mu.Unlock()
			return nil
		})
	}

	// VMess configs
	if len(vmessNodeIDs) > 0 {
		g.Go(func() error {
			configs, err := r.vmessConfigRepo.GetByNodeIDs(gctx, vmessNodeIDs)
			if err != nil {
				r.logger.Errorw("failed to get vmess configs", "error", err)
				return fmt.Errorf("failed to get vmess configs: %w", err)
			}
			mu.Lock()
			vmessConfigs = configs
			mu.Unlock()
			return nil
		})
	}

	// Hysteria2 configs
	if len(hysteria2NodeIDs) > 0 {
		g.Go(func() error {
			configs, err := r.hysteria2ConfigRepo.GetByNodeIDs(gctx, hysteria2NodeIDs)
			if err != nil {
				r.logger.Errorw("failed to get hysteria2 configs", "error", err)
				return fmt.Errorf("failed to get hysteria2 configs: %w", err)
			}
			mu.Lock()
			hysteria2Configs = configs
			mu.Unlock()
			return nil
		})
	}

	// TUIC configs
	if len(tuicNodeIDs) > 0 {
		g.Go(func() error {
			configs, err := r.tuicConfigRepo.GetByNodeIDs(gctx, tuicNodeIDs)
			if err != nil {
				r.logger.Errorw("failed to get tuic configs", "error", err)
				return fmt.Errorf("failed to get tuic configs: %w", err)
			}
			mu.Lock()
			tuicConfigs = configs
			mu.Unlock()
			return nil
		})
	}

	// Wait for all goroutines to complete
	if err := g.Wait(); err != nil {
		return nil, 0, err
	}

	// Convert shadowsocks configs to mapper format
	ssConfigs := make(map[uint]*mappers.ShadowsocksConfigData)
	for nodeID, data := range ssConfigsRaw {
		ssConfigs[nodeID] = &mappers.ShadowsocksConfigData{
			EncryptionConfig: data.EncryptionConfig,
			PluginConfig:     data.PluginConfig,
		}
	}

	// Convert models to entities
	entities, err := r.mapper.ToEntities(nodeModels, ssConfigs, trojanConfigs, vlessConfigs, vmessConfigs, hysteria2Configs, tuicConfigs)
	if err != nil {
		r.logger.Errorw("failed to map node models to entities", "error", err)
		return nil, 0, fmt.Errorf("failed to map nodes: %w", err)
	}

	return entities, total, nil
}

// ExistsByName checks if a node with the given name exists.
func (r *NodeRepositoryImpl) ExistsByName(ctx context.Context, name string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Where("name = ?", name).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to check node existence by name", "name", name, "error", err)
		return false, fmt.Errorf("failed to check node existence: %w", err)
	}
	return count > 0, nil
}

// ExistsByNameExcluding checks if a node with the given name exists, excluding a specific node ID.
func (r *NodeRepositoryImpl) ExistsByNameExcluding(ctx context.Context, name string, excludeID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Where("name = ? AND id != ?", name, excludeID).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to check node existence by name", "name", name, "exclude_id", excludeID, "error", err)
		return false, fmt.Errorf("failed to check node existence: %w", err)
	}
	return count > 0, nil
}

// ExistsByAddress checks if a node with the given address and port exists.
func (r *NodeRepositoryImpl) ExistsByAddress(ctx context.Context, address string, port int) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Where("server_address = ? AND agent_port = ?", address, port).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to check node existence by address", "address", address, "port", port, "error", err)
		return false, fmt.Errorf("failed to check node existence: %w", err)
	}
	return count > 0, nil
}

// ExistsByAddressExcluding checks if a node with the given address and port exists, excluding a specific node ID.
func (r *NodeRepositoryImpl) ExistsByAddressExcluding(ctx context.Context, address string, port int, excludeID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Where("server_address = ? AND agent_port = ? AND id != ?", address, port, excludeID).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to check node existence by address", "address", address, "port", port, "exclude_id", excludeID, "error", err)
		return false, fmt.Errorf("failed to check node existence: %w", err)
	}
	return count > 0, nil
}

// IncrementTraffic atomically increments the traffic_used field
func (r *NodeRepositoryImpl) IncrementTraffic(ctx context.Context, nodeID uint, amount uint64) error {
	if amount == 0 {
		return nil
	}

	result := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Where("id = ?", nodeID).
		UpdateColumn("traffic_used", gorm.Expr("traffic_used + ?", amount))

	if result.Error != nil {
		r.logger.Errorw("failed to increment traffic", "node_id", nodeID, "amount", amount, "error", result.Error)
		return fmt.Errorf("failed to increment traffic: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("node not found")
	}

	r.logger.Debugw("traffic incremented successfully", "node_id", nodeID, "amount", amount)
	return nil
}

// UpdateLastSeenAt updates the last_seen_at timestamp, public IPs, and agent info for a node
// Uses conditional update to avoid race conditions: only updates if last_seen_at is NULL
// or older than the threshold (2 minutes). This moves the throttling logic to the database
// layer for atomic operation.
func (r *NodeRepositoryImpl) UpdateLastSeenAt(ctx context.Context, nodeID uint, publicIPv4, publicIPv6, agentVersion, platform, arch string) error {
	now := biztime.NowUTC()
	threshold := now.Add(-2 * time.Minute)

	updates := map[string]interface{}{
		"last_seen_at": now,
	}

	// Only update public IPs if provided
	if publicIPv4 != "" {
		updates["public_ipv4"] = publicIPv4
	}
	if publicIPv6 != "" {
		updates["public_ipv6"] = publicIPv6
	}

	// Only update agent info if provided
	if agentVersion != "" {
		updates["agent_version"] = agentVersion
	}
	if platform != "" {
		updates["platform"] = platform
	}
	if arch != "" {
		updates["arch"] = arch
	}

	// Use conditional update to prevent race conditions
	// Only update if last_seen_at is NULL or older than 2 minutes
	result := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Where("id = ? AND (last_seen_at IS NULL OR last_seen_at < ?)", nodeID, threshold).
		Updates(updates)

	if result.Error != nil {
		r.logger.Errorw("failed to update last_seen_at", "node_id", nodeID, "error", result.Error)
		return fmt.Errorf("failed to update last_seen_at: %w", result.Error)
	}

	// RowsAffected == 0 is normal when throttled, not an error
	if result.RowsAffected > 0 {
		r.logger.Debugw("last_seen_at updated successfully",
			"node_id", nodeID,
			"public_ipv4", publicIPv4,
			"public_ipv6", publicIPv6,
			"agent_version", agentVersion,
		)
	}
	return nil
}

// GetLastSeenAt retrieves just the last_seen_at timestamp for a node (lightweight query)
// Returns NotFoundError if the node does not exist
func (r *NodeRepositoryImpl) GetLastSeenAt(ctx context.Context, nodeID uint) (*time.Time, error) {
	var model models.NodeModel
	err := r.db.WithContext(ctx).
		Select("id", "last_seen_at").
		Where("id = ?", nodeID).
		First(&model).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("node not found")
		}
		r.logger.Errorw("failed to get last_seen_at", "node_id", nodeID, "error", err)
		return nil, fmt.Errorf("failed to get last_seen_at: %w", err)
	}

	return model.LastSeenAt, nil
}

// ListByUserID returns nodes owned by a specific user
func (r *NodeRepositoryImpl) ListByUserID(ctx context.Context, userID uint, filter node.NodeFilter) ([]*node.Node, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.NodeModel{}).Where("user_id = ?", userID)

	// Apply filters
	if filter.Name != nil && *filter.Name != "" {
		query = query.Where("name LIKE ?", "%"+*filter.Name+"%")
	}
	if filter.Status != nil && *filter.Status != "" {
		query = query.Where("status = ?", *filter.Status)
	}

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		r.logger.Errorw("failed to count user nodes", "user_id", userID, "error", err)
		return nil, 0, fmt.Errorf("failed to count user nodes: %w", err)
	}

	// Apply sorting with whitelist validation to prevent SQL injection
	sortBy := strings.ToLower(filter.SortFilter.SortBy)
	if sortBy != "" && allowedNodeOrderByFields[sortBy] {
		order := "ASC"
		if filter.SortFilter.IsDescending() {
			order = "DESC"
		}
		query = query.Order(fmt.Sprintf("%s %s", sortBy, order))
	} else {
		query = query.Order("created_at DESC")
	}

	// Apply pagination
	offset := filter.PageFilter.Offset()
	limit := filter.PageFilter.Limit()
	query = query.Offset(offset).Limit(limit)

	// Execute query
	var nodeModels []*models.NodeModel
	if err := query.Find(&nodeModels).Error; err != nil {
		r.logger.Errorw("failed to list user nodes", "user_id", userID, "error", err)
		return nil, 0, fmt.Errorf("failed to list user nodes: %w", err)
	}

	// Collect node IDs by protocol
	var ssNodeIDs, trojanNodeIDs, vlessNodeIDs, vmessNodeIDs, hysteria2NodeIDs, tuicNodeIDs []uint
	for _, m := range nodeModels {
		switch m.Protocol {
		case "shadowsocks":
			ssNodeIDs = append(ssNodeIDs, m.ID)
		case "trojan":
			trojanNodeIDs = append(trojanNodeIDs, m.ID)
		case "vless":
			vlessNodeIDs = append(vlessNodeIDs, m.ID)
		case "vmess":
			vmessNodeIDs = append(vmessNodeIDs, m.ID)
		case "hysteria2":
			hysteria2NodeIDs = append(hysteria2NodeIDs, m.ID)
		case "tuic":
			tuicNodeIDs = append(tuicNodeIDs, m.ID)
		}
	}

	// Load protocol-specific configs
	ssConfigsRaw, err := r.shadowsocksConfigRepo.GetByNodeIDs(ctx, ssNodeIDs)
	if err != nil {
		r.logger.Errorw("failed to get shadowsocks configs", "error", err)
		return nil, 0, fmt.Errorf("failed to get shadowsocks configs: %w", err)
	}

	ssConfigs := make(map[uint]*mappers.ShadowsocksConfigData)
	for nodeID, data := range ssConfigsRaw {
		ssConfigs[nodeID] = &mappers.ShadowsocksConfigData{
			EncryptionConfig: data.EncryptionConfig,
			PluginConfig:     data.PluginConfig,
		}
	}

	trojanConfigs, err := r.trojanConfigRepo.GetByNodeIDs(ctx, trojanNodeIDs)
	if err != nil {
		r.logger.Errorw("failed to get trojan configs", "error", err)
		return nil, 0, fmt.Errorf("failed to get trojan configs: %w", err)
	}

	vlessConfigs, err := r.vlessConfigRepo.GetByNodeIDs(ctx, vlessNodeIDs)
	if err != nil {
		r.logger.Errorw("failed to get vless configs", "error", err)
		return nil, 0, fmt.Errorf("failed to get vless configs: %w", err)
	}

	vmessConfigs, err := r.vmessConfigRepo.GetByNodeIDs(ctx, vmessNodeIDs)
	if err != nil {
		r.logger.Errorw("failed to get vmess configs", "error", err)
		return nil, 0, fmt.Errorf("failed to get vmess configs: %w", err)
	}

	hysteria2Configs, err := r.hysteria2ConfigRepo.GetByNodeIDs(ctx, hysteria2NodeIDs)
	if err != nil {
		r.logger.Errorw("failed to get hysteria2 configs", "error", err)
		return nil, 0, fmt.Errorf("failed to get hysteria2 configs: %w", err)
	}

	tuicConfigs, err := r.tuicConfigRepo.GetByNodeIDs(ctx, tuicNodeIDs)
	if err != nil {
		r.logger.Errorw("failed to get tuic configs", "error", err)
		return nil, 0, fmt.Errorf("failed to get tuic configs: %w", err)
	}

	entities, err := r.mapper.ToEntities(nodeModels, ssConfigs, trojanConfigs, vlessConfigs, vmessConfigs, hysteria2Configs, tuicConfigs)
	if err != nil {
		r.logger.Errorw("failed to map node models to entities", "error", err)
		return nil, 0, fmt.Errorf("failed to map nodes: %w", err)
	}

	return entities, total, nil
}

// CountByUserID counts nodes owned by a specific user.
func (r *NodeRepositoryImpl) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to count user nodes", "user_id", userID, "error", err)
		return 0, fmt.Errorf("failed to count user nodes: %w", err)
	}
	return count, nil
}

// ExistsByNameForUser checks if a node with the given name exists for a specific user.
func (r *NodeRepositoryImpl) ExistsByNameForUser(ctx context.Context, name string, userID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Where("name = ? AND user_id = ?", name, userID).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to check node existence by name for user", "name", name, "user_id", userID, "error", err)
		return false, fmt.Errorf("failed to check node existence: %w", err)
	}
	return count > 0, nil
}

// ExistsByNameForUserExcluding checks if a node with the given name exists for a user, excluding a specific node.
func (r *NodeRepositoryImpl) ExistsByNameForUserExcluding(ctx context.Context, name string, userID uint, excludeID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Where("name = ? AND user_id = ? AND id != ?", name, userID, excludeID).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to check node existence by name for user", "name", name, "user_id", userID, "exclude_id", excludeID, "error", err)
		return false, fmt.Errorf("failed to check node existence: %w", err)
	}
	return count > 0, nil
}

// ExistsByAddressForUser checks if a node with the given address and port exists for a specific user.
func (r *NodeRepositoryImpl) ExistsByAddressForUser(ctx context.Context, address string, port int, userID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Where("server_address = ? AND agent_port = ? AND user_id = ?", address, port, userID).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to check node existence by address for user", "address", address, "port", port, "user_id", userID, "error", err)
		return false, fmt.Errorf("failed to check node existence: %w", err)
	}
	return count > 0, nil
}

// ExistsByAddressForUserExcluding checks if a node with the given address and port exists for a user, excluding a specific node.
func (r *NodeRepositoryImpl) ExistsByAddressForUserExcluding(ctx context.Context, address string, port int, userID uint, excludeID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Where("server_address = ? AND agent_port = ? AND user_id = ? AND id != ?", address, port, userID, excludeID).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to check node existence by address for user", "address", address, "port", port, "user_id", userID, "exclude_id", excludeID, "error", err)
		return false, fmt.Errorf("failed to check node existence: %w", err)
	}
	return count > 0, nil
}

// GetPublicIPs retrieves the current public IPs for a node.
// Returns (ipv4, ipv6, error) - empty string if IP is not set
func (r *NodeRepositoryImpl) GetPublicIPs(ctx context.Context, nodeID uint) (string, string, error) {
	var model models.NodeModel
	err := r.db.WithContext(ctx).
		Select("id", "public_ipv4", "public_ipv6").
		Where("id = ?", nodeID).
		First(&model).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", "", errors.NewNotFoundError("node not found")
		}
		r.logger.Errorw("failed to get public IPs", "node_id", nodeID, "error", err)
		return "", "", fmt.Errorf("failed to get public IPs: %w", err)
	}

	ipv4 := ""
	ipv6 := ""
	if model.PublicIPv4 != nil {
		ipv4 = *model.PublicIPv4
	}
	if model.PublicIPv6 != nil {
		ipv6 = *model.PublicIPv6
	}

	return ipv4, ipv6, nil
}

// UpdatePublicIP updates the public IP for a node (immediate, no throttling).
// Pass empty string to skip updating that IP version.
func (r *NodeRepositoryImpl) UpdatePublicIP(ctx context.Context, nodeID uint, ipv4, ipv6 string) error {
	updates := make(map[string]interface{})
	if ipv4 != "" {
		updates["public_ipv4"] = ipv4
	}
	if ipv6 != "" {
		updates["public_ipv6"] = ipv6
	}

	if len(updates) == 0 {
		return nil
	}

	result := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Where("id = ?", nodeID).
		Updates(updates)

	if result.Error != nil {
		r.logger.Errorw("failed to update public IP", "node_id", nodeID, "error", result.Error)
		return fmt.Errorf("failed to update public IP: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("node not found")
	}

	r.logger.Infow("node public IP updated", "node_id", nodeID, "ipv4", ipv4, "ipv6", ipv6)
	return nil
}

// ValidateNodeSIDsForUser checks if all given node SIDs exist and belong to the specified user.
// Returns slice of invalid SIDs (not found or not owned by user).
func (r *NodeRepositoryImpl) ValidateNodeSIDsForUser(ctx context.Context, sids []string, userID uint) ([]string, error) {
	if len(sids) == 0 {
		return nil, nil
	}

	var existingSIDs []string
	err := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Where("sid IN ? AND user_id = ?", sids, userID).
		Pluck("sid", &existingSIDs).Error
	if err != nil {
		r.logger.Errorw("failed to validate node SIDs for user", "sids", sids, "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to validate node SIDs: %w", err)
	}

	// Find invalid SIDs (not in existing set)
	existingSet := make(map[string]bool)
	for _, sid := range existingSIDs {
		existingSet[sid] = true
	}

	var invalidSIDs []string
	for _, sid := range sids {
		if !existingSet[sid] {
			invalidSIDs = append(invalidSIDs, sid)
		}
	}

	return invalidSIDs, nil
}

// ValidateNodeSIDsExist checks if all given node SIDs exist (for admin nodes).
// Returns slice of invalid SIDs (not found).
func (r *NodeRepositoryImpl) ValidateNodeSIDsExist(ctx context.Context, sids []string) ([]string, error) {
	if len(sids) == 0 {
		return nil, nil
	}

	var existingSIDs []string
	err := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Where("sid IN ?", sids).
		Pluck("sid", &existingSIDs).Error
	if err != nil {
		r.logger.Errorw("failed to validate node SIDs existence", "sids", sids, "error", err)
		return nil, fmt.Errorf("failed to validate node SIDs: %w", err)
	}

	// Find invalid SIDs (not in existing set)
	existingSet := make(map[string]bool)
	for _, sid := range existingSIDs {
		existingSet[sid] = true
	}

	var invalidSIDs []string
	for _, sid := range sids {
		if !existingSet[sid] {
			invalidSIDs = append(invalidSIDs, sid)
		}
	}

	return invalidSIDs, nil
}

// GetAllMetadata returns lightweight metadata for all nodes.
// Only queries id, sid, name fields without loading protocol configs.
func (r *NodeRepositoryImpl) GetAllMetadata(ctx context.Context) ([]*node.NodeMetadata, error) {
	var results []struct {
		ID   uint   `gorm:"column:id"`
		SID  string `gorm:"column:sid"`
		Name string `gorm:"column:name"`
	}

	if err := r.db.WithContext(ctx).
		Model(&models.NodeModel{}).
		Select("id, sid, name").
		Find(&results).Error; err != nil {
		r.logger.Errorw("failed to get all node metadata", "error", err)
		return nil, fmt.Errorf("failed to get node metadata: %w", err)
	}

	metadata := make([]*node.NodeMetadata, len(results))
	for i, res := range results {
		metadata[i] = &node.NodeMetadata{
			ID:   res.ID,
			SID:  res.SID,
			Name: res.Name,
		}
	}

	return metadata, nil
}

// GetMetadataBySIDs returns lightweight metadata for nodes by SIDs.
// Only queries id, sid, name fields without loading protocol configs.
func (r *NodeRepositoryImpl) GetMetadataBySIDs(ctx context.Context, sids []string) ([]*node.NodeMetadata, error) {
	if len(sids) == 0 {
		return []*node.NodeMetadata{}, nil
	}

	var results []struct {
		ID   uint   `gorm:"column:id"`
		SID  string `gorm:"column:sid"`
		Name string `gorm:"column:name"`
	}

	if err := r.db.WithContext(ctx).
		Model(&models.NodeModel{}).
		Select("id, sid, name").
		Where("sid IN ?", sids).
		Find(&results).Error; err != nil {
		r.logger.Errorw("failed to get node metadata by SIDs", "sids", sids, "error", err)
		return nil, fmt.Errorf("failed to get node metadata: %w", err)
	}

	metadata := make([]*node.NodeMetadata, len(results))
	for i, res := range results {
		metadata[i] = &node.NodeMetadata{
			ID:   res.ID,
			SID:  res.SID,
			Name: res.Name,
		}
	}

	return metadata, nil
}

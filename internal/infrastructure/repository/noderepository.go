package repository

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/node"
	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
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
	anytlsConfigRepo      *AnyTLSConfigRepository
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
		anytlsConfigRepo:      NewAnyTLSConfigRepository(db, logger),
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
		case vo.ProtocolAnyTLS:
			if nodeEntity.AnyTLSConfig() != nil {
				if err := r.anytlsConfigRepo.CreateInTx(tx, model.ID, nodeEntity.AnyTLSConfig()); err != nil {
					return fmt.Errorf("failed to create anytls config: %w", err)
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
				"maintenance_reason", "token_hash", "api_token", "group_ids", "route_config", "mute_notification",
				"expires_at", "cost_label", "version", "updated_at",
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
			if err := r.anytlsConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete anytls config: %w", err)
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
			if err := r.anytlsConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete anytls config: %w", err)
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
			if err := r.anytlsConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete anytls config: %w", err)
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
			if err := r.anytlsConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete anytls config: %w", err)
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
			if err := r.anytlsConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete anytls config: %w", err)
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
			if err := r.anytlsConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete anytls config: %w", err)
			}
		case vo.ProtocolAnyTLS:
			if err := r.anytlsConfigRepo.UpdateInTx(tx, model.ID, nodeEntity.AnyTLSConfig()); err != nil {
				return fmt.Errorf("failed to update anytls config: %w", err)
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
			if err := r.tuicConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete tuic config: %w", err)
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
		if err := r.anytlsConfigRepo.DeleteInTx(tx, id); err != nil {
			return fmt.Errorf("failed to delete anytls config: %w", err)
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

package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/orris-inc/orris/internal/domain/node"
	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

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

	// Collect node IDs by protocol (pre-allocate with upper bound capacity)
	protoCapacity := len(nodeModels)
	ssNodeIDs := make([]uint, 0, protoCapacity)
	trojanNodeIDs := make([]uint, 0, protoCapacity)
	vlessNodeIDs := make([]uint, 0, protoCapacity)
	vmessNodeIDs := make([]uint, 0, protoCapacity)
	hysteria2NodeIDs := make([]uint, 0, protoCapacity)
	tuicNodeIDs := make([]uint, 0, protoCapacity)
	anytlsNodeIDs := make([]uint, 0, protoCapacity)
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
		case "anytls":
			anytlsNodeIDs = append(anytlsNodeIDs, m.ID)
		}
	}

	// Load protocol-specific configs in parallel.
	// Each goroutine writes to a distinct variable and errgroup.Wait() provides
	// a happens-before guarantee, so no mutex is needed.
	var (
		ssConfigsRaw     map[uint]*ShadowsocksConfigData
		trojanConfigs    map[uint]*vo.TrojanConfig
		vlessConfigs     map[uint]*vo.VLESSConfig
		vmessConfigs     map[uint]*vo.VMessConfig
		hysteria2Configs map[uint]*vo.Hysteria2Config
		tuicConfigs      map[uint]*vo.TUICConfig
		anytlsConfigs    map[uint]*vo.AnyTLSConfig
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
			ssConfigsRaw = configs
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
			trojanConfigs = configs
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
			vlessConfigs = configs
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
			vmessConfigs = configs
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
			hysteria2Configs = configs
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
			tuicConfigs = configs
			return nil
		})
	}

	// AnyTLS configs
	if len(anytlsNodeIDs) > 0 {
		g.Go(func() error {
			configs, err := r.anytlsConfigRepo.GetByNodeIDs(gctx, anytlsNodeIDs)
			if err != nil {
				r.logger.Errorw("failed to get anytls configs", "error", err)
				return fmt.Errorf("failed to get anytls configs: %w", err)
			}
			anytlsConfigs = configs
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
	entities, err := r.mapper.ToEntities(nodeModels, ssConfigs, trojanConfigs, vlessConfigs, vmessConfigs, hysteria2Configs, tuicConfigs, anytlsConfigs)
	if err != nil {
		r.logger.Errorw("failed to map node models to entities", "error", err)
		return nil, 0, fmt.Errorf("failed to map nodes: %w", err)
	}

	return entities, total, nil
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

	entities, err := r.loadProtocolConfigsAndConvert(ctx, nodeModels)
	if err != nil {
		return nil, 0, err
	}

	return entities, total, nil
}

// loadProtocolConfigsAndConvert loads protocol-specific configs for node models
// in parallel and converts them to domain entities.
// This is a shared helper to avoid duplicating protocol config loading logic.
func (r *NodeRepositoryImpl) loadProtocolConfigsAndConvert(ctx context.Context, nodeModels []*models.NodeModel) ([]*node.Node, error) {
	// Collect node IDs by protocol
	protoCapacity := len(nodeModels)
	ssNodeIDs := make([]uint, 0, protoCapacity)
	trojanNodeIDs := make([]uint, 0, protoCapacity)
	vlessNodeIDs := make([]uint, 0, protoCapacity)
	vmessNodeIDs := make([]uint, 0, protoCapacity)
	hysteria2NodeIDs := make([]uint, 0, protoCapacity)
	tuicNodeIDs := make([]uint, 0, protoCapacity)
	anytlsNodeIDs := make([]uint, 0, protoCapacity)
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
		case "anytls":
			anytlsNodeIDs = append(anytlsNodeIDs, m.ID)
		}
	}

	// Load protocol-specific configs in parallel.
	// Each goroutine writes to a distinct variable and errgroup.Wait() provides
	// a happens-before guarantee, so no mutex is needed.
	var (
		ssConfigsRaw     map[uint]*ShadowsocksConfigData
		trojanConfigs    map[uint]*vo.TrojanConfig
		vlessConfigs     map[uint]*vo.VLESSConfig
		vmessConfigs     map[uint]*vo.VMessConfig
		hysteria2Configs map[uint]*vo.Hysteria2Config
		tuicConfigs      map[uint]*vo.TUICConfig
		anytlsConfigs    map[uint]*vo.AnyTLSConfig
	)

	g, gctx := errgroup.WithContext(ctx)

	if len(ssNodeIDs) > 0 {
		g.Go(func() error {
			configs, err := r.shadowsocksConfigRepo.GetByNodeIDs(gctx, ssNodeIDs)
			if err != nil {
				return fmt.Errorf("failed to get shadowsocks configs: %w", err)
			}
			ssConfigsRaw = configs
			return nil
		})
	}
	if len(trojanNodeIDs) > 0 {
		g.Go(func() error {
			configs, err := r.trojanConfigRepo.GetByNodeIDs(gctx, trojanNodeIDs)
			if err != nil {
				return fmt.Errorf("failed to get trojan configs: %w", err)
			}
			trojanConfigs = configs
			return nil
		})
	}
	if len(vlessNodeIDs) > 0 {
		g.Go(func() error {
			configs, err := r.vlessConfigRepo.GetByNodeIDs(gctx, vlessNodeIDs)
			if err != nil {
				return fmt.Errorf("failed to get vless configs: %w", err)
			}
			vlessConfigs = configs
			return nil
		})
	}
	if len(vmessNodeIDs) > 0 {
		g.Go(func() error {
			configs, err := r.vmessConfigRepo.GetByNodeIDs(gctx, vmessNodeIDs)
			if err != nil {
				return fmt.Errorf("failed to get vmess configs: %w", err)
			}
			vmessConfigs = configs
			return nil
		})
	}
	if len(hysteria2NodeIDs) > 0 {
		g.Go(func() error {
			configs, err := r.hysteria2ConfigRepo.GetByNodeIDs(gctx, hysteria2NodeIDs)
			if err != nil {
				return fmt.Errorf("failed to get hysteria2 configs: %w", err)
			}
			hysteria2Configs = configs
			return nil
		})
	}
	if len(tuicNodeIDs) > 0 {
		g.Go(func() error {
			configs, err := r.tuicConfigRepo.GetByNodeIDs(gctx, tuicNodeIDs)
			if err != nil {
				return fmt.Errorf("failed to get tuic configs: %w", err)
			}
			tuicConfigs = configs
			return nil
		})
	}
	if len(anytlsNodeIDs) > 0 {
		g.Go(func() error {
			configs, err := r.anytlsConfigRepo.GetByNodeIDs(gctx, anytlsNodeIDs)
			if err != nil {
				return fmt.Errorf("failed to get anytls configs: %w", err)
			}
			anytlsConfigs = configs
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		r.logger.Errorw("failed to load protocol configs", "error", err)
		return nil, err
	}

	// Convert shadowsocks configs to mapper format
	ssConfigs := make(map[uint]*mappers.ShadowsocksConfigData)
	for nodeID, data := range ssConfigsRaw {
		ssConfigs[nodeID] = &mappers.ShadowsocksConfigData{
			EncryptionConfig: data.EncryptionConfig,
			PluginConfig:     data.PluginConfig,
		}
	}

	entities, err := r.mapper.ToEntities(nodeModels, ssConfigs, trojanConfigs, vlessConfigs, vmessConfigs, hysteria2Configs, tuicConfigs, anytlsConfigs)
	if err != nil {
		r.logger.Errorw("failed to map node models to entities", "error", err)
		return nil, fmt.Errorf("failed to map nodes: %w", err)
	}

	return entities, nil
}

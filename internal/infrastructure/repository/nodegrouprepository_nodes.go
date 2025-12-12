package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// AddNode adds a node to a node group
func (r *NodeGroupRepositoryImpl) AddNode(ctx context.Context, groupID, nodeID uint) error {
	association := &models.NodeGroupNodeModel{
		NodeGroupID: groupID,
		NodeID:      nodeID,
	}

	if err := r.db.WithContext(ctx).Create(association).Error; err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "duplicate key") {
			// Already associated, not an error
			return nil
		}
		r.logger.Errorw("failed to add node to group", "group_id", groupID, "node_id", nodeID, "error", err)
		return fmt.Errorf("failed to add node to group: %w", err)
	}

	r.logger.Infow("node added to group successfully", "group_id", groupID, "node_id", nodeID)
	return nil
}

// RemoveNode removes a node from a node group
func (r *NodeGroupRepositoryImpl) RemoveNode(ctx context.Context, groupID, nodeID uint) error {
	result := r.db.WithContext(ctx).
		Where("node_group_id = ? AND node_id = ?", groupID, nodeID).
		Delete(&models.NodeGroupNodeModel{})

	if result.Error != nil {
		r.logger.Errorw("failed to remove node from group", "group_id", groupID, "node_id", nodeID, "error", result.Error)
		return fmt.Errorf("failed to remove node from group: %w", result.Error)
	}

	r.logger.Infow("node removed from group successfully", "group_id", groupID, "node_id", nodeID)
	return nil
}

// GetNodesByGroupID retrieves all nodes in a node group
func (r *NodeGroupRepositoryImpl) GetNodesByGroupID(ctx context.Context, groupID uint) ([]*node.Node, error) {
	var nodeModels []*models.NodeModel

	err := r.db.WithContext(ctx).
		Joins("JOIN node_group_nodes ON nodes.id = node_group_nodes.node_id").
		Where("node_group_nodes.node_group_id = ?", groupID).
		Order("nodes.sort_order ASC").
		Find(&nodeModels).Error

	if err != nil {
		r.logger.Errorw("failed to get nodes by group ID", "group_id", groupID, "error", err)
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	// Collect node IDs by protocol
	var ssNodeIDs, trojanNodeIDs []uint
	for _, m := range nodeModels {
		switch m.Protocol {
		case "shadowsocks":
			ssNodeIDs = append(ssNodeIDs, m.ID)
		case "trojan":
			trojanNodeIDs = append(trojanNodeIDs, m.ID)
		}
	}

	// Load protocol-specific configs
	ssConfigsRaw, err := r.shadowsocksConfigRepo.GetByNodeIDs(ctx, ssNodeIDs)
	if err != nil {
		r.logger.Errorw("failed to get shadowsocks configs", "error", err)
		return nil, fmt.Errorf("failed to get shadowsocks configs: %w", err)
	}

	// Convert to mapper format
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
		return nil, fmt.Errorf("failed to get trojan configs: %w", err)
	}

	// Convert models to entities
	entities, err := r.nodeMapper.ToEntities(nodeModels, ssConfigs, trojanConfigs)
	if err != nil {
		r.logger.Errorw("failed to map node models to entities", "error", err)
		return nil, fmt.Errorf("failed to map nodes: %w", err)
	}

	return entities, nil
}

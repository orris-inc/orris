package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/node"
	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/errors"
)

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

// UpdateLastSeenAt updates the last_seen_at timestamp and public IPs for a node
func (r *NodeRepositoryImpl) UpdateLastSeenAt(ctx context.Context, nodeID uint, publicIPv4, publicIPv6 string) error {
	updates := map[string]interface{}{
		"last_seen_at": gorm.Expr("NOW()"),
	}

	// Only update public IPs if provided
	if publicIPv4 != "" {
		updates["public_ipv4"] = publicIPv4
	}
	if publicIPv6 != "" {
		updates["public_ipv6"] = publicIPv6
	}

	result := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Where("id = ?", nodeID).
		Updates(updates)

	if result.Error != nil {
		r.logger.Errorw("failed to update last_seen_at", "node_id", nodeID, "error", result.Error)
		return fmt.Errorf("failed to update last_seen_at: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("node not found")
	}

	r.logger.Debugw("last_seen_at updated successfully",
		"node_id", nodeID,
		"public_ipv4", publicIPv4,
		"public_ipv6", publicIPv6,
	)
	return nil
}

// GetLastSeenAt retrieves just the last_seen_at timestamp for a node (lightweight query)
func (r *NodeRepositoryImpl) GetLastSeenAt(ctx context.Context, nodeID uint) (*node.Node, error) {
	var model models.NodeModel
	err := r.db.WithContext(ctx).
		Select("id", "last_seen_at").
		Where("id = ?", nodeID).
		First(&model).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get last_seen_at", "node_id", nodeID, "error", err)
		return nil, fmt.Errorf("failed to get last_seen_at: %w", err)
	}

	// Create a minimal node entity with just the lastSeenAt field
	// We use empty/default values for other fields since we only need lastSeenAt
	serverAddr, _ := vo.NewServerAddress("0.0.0.0")
	nodeEntity, _ := node.ReconstructNode(
		model.ID,
		model.ShortID,
		"",
		serverAddr,
		1,   // agentPort (placeholder, just needs to be non-zero)
		nil, // subscriptionPort
		vo.Protocol("shadowsocks"),
		vo.EncryptionConfig{},
		nil,
		nil,
		vo.NodeStatusInactive,
		vo.NewNodeMetadata("", nil, ""),
		"placeholder", // tokenHash
		"",            // apiToken
		0,
		nil,
		model.LastSeenAt,
		nil, // publicIPv4
		nil, // publicIPv6
		0,
		model.CreatedAt,
		model.UpdatedAt,
	)

	return nodeEntity, nil
}

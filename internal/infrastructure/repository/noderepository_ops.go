package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/errors"
)

// --- Existence checks ---

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

// --- Traffic, status, and batch operations ---

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

// BatchUpdateGroupIDs updates group_ids for multiple nodes using a single CASE WHEN SQL.
// This is optimized for resource group membership changes where only group_ids needs to be updated.
func (r *NodeRepositoryImpl) BatchUpdateGroupIDs(ctx context.Context, nodeGroupIDs map[uint][]uint) (int, error) {
	if len(nodeGroupIDs) == 0 {
		return 0, nil
	}

	// Pre-serialize all group IDs to JSON
	type nodeGroupJSON struct {
		nodeID    uint
		jsonBytes []byte
	}
	entries := make([]nodeGroupJSON, 0, len(nodeGroupIDs))
	for nodeID, groupIDs := range nodeGroupIDs {
		var groupIDsJSON []byte
		var err error
		if len(groupIDs) == 0 {
			groupIDsJSON = []byte("[]")
		} else {
			groupIDsJSON, err = json.Marshal(groupIDs)
			if err != nil {
				return 0, fmt.Errorf("failed to marshal group IDs for node %d: %w", nodeID, err)
			}
		}
		entries = append(entries, nodeGroupJSON{nodeID: nodeID, jsonBytes: groupIDsJSON})
	}

	// Build CASE WHEN SQL:
	// UPDATE nodes SET group_ids = CASE id WHEN ? THEN ? ... END, updated_at = ? WHERE id IN (?,...）
	var sb strings.Builder
	sb.WriteString("UPDATE nodes SET group_ids = CASE id ")

	// args: CASE WHEN pairs + updated_at + WHERE IN ids
	args := make([]interface{}, 0, len(entries)*2+1+len(entries))
	ids := make([]interface{}, 0, len(entries))

	for _, e := range entries {
		sb.WriteString("WHEN ? THEN ? ")
		args = append(args, e.nodeID, string(e.jsonBytes))
		ids = append(ids, e.nodeID)
	}

	sb.WriteString("END, updated_at = ? WHERE id IN (")
	args = append(args, biztime.NowUTC())

	for i := range ids {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString("?")
	}
	sb.WriteString(")")
	args = append(args, ids...)

	var updated int
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Exec(sb.String(), args...)
		if result.Error != nil {
			return fmt.Errorf("failed to batch update group IDs: %w", result.Error)
		}
		updated = int(result.RowsAffected)
		return nil
	})

	if err != nil {
		r.logger.Errorw("failed to batch update group IDs", "error", err, "node_count", len(nodeGroupIDs))
		return 0, err
	}

	r.logger.Infow("batch updated group IDs", "updated_count", updated, "total_count", len(nodeGroupIDs))
	return updated, nil
}

// CountByLastSeenAfter counts nodes whose last_seen_at is after the given threshold.
func (r *NodeRepositoryImpl) CountByLastSeenAfter(ctx context.Context, threshold time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Where("last_seen_at > ?", threshold).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to count nodes by last_seen_at", "threshold", threshold, "error", err)
		return 0, fmt.Errorf("failed to count nodes by last_seen_at: %w", err)
	}
	return count, nil
}

// GetIDsByGroupID returns all node IDs that belong to the specified resource group.
// This method has no pagination limit — it returns only IDs for efficiency.
func (r *NodeRepositoryImpl) GetIDsByGroupID(ctx context.Context, groupID uint) ([]uint, error) {
	groupIDsJSON, _ := json.Marshal([]uint{groupID})

	var ids []uint
	if err := r.db.WithContext(ctx).
		Model(&models.NodeModel{}).
		Where("JSON_OVERLAPS(group_ids, ?)", string(groupIDsJSON)).
		Pluck("id", &ids).Error; err != nil {
		r.logger.Errorw("failed to get node IDs by group ID", "group_id", groupID, "error", err)
		return nil, fmt.Errorf("failed to get node IDs by group ID: %w", err)
	}

	return ids, nil
}

// FindExpiringNodes returns active nodes that will expire within the specified days.
// Only returns nodes that have expires_at set, are active, and are not already expired.
func (r *NodeRepositoryImpl) FindExpiringNodes(ctx context.Context, withinDays int) ([]*node.ExpiringNodeInfo, error) {
	now := biztime.NowUTC()
	threshold := now.AddDate(0, 0, withinDays)

	var results []struct {
		ID        uint       `gorm:"column:id"`
		SID       string     `gorm:"column:sid"`
		Name      string     `gorm:"column:name"`
		ExpiresAt *time.Time `gorm:"column:expires_at"`
		CostLabel *string    `gorm:"column:cost_label"`
	}

	if err := r.db.WithContext(ctx).
		Model(&models.NodeModel{}).
		Select("id, sid, name, expires_at, cost_label").
		Where("status = ?", "active").       // Only active nodes need expiring notification
		Where("expires_at IS NOT NULL").
		Where("expires_at > ?", now).        // Not already expired
		Where("expires_at <= ?", threshold). // Within threshold
		Order("expires_at ASC").
		Find(&results).Error; err != nil {
		r.logger.Errorw("failed to find expiring nodes", "within_days", withinDays, "error", err)
		return nil, fmt.Errorf("failed to find expiring nodes: %w", err)
	}

	nodes := make([]*node.ExpiringNodeInfo, 0, len(results))
	for _, res := range results {
		if res.ExpiresAt == nil {
			continue
		}
		nodes = append(nodes, &node.ExpiringNodeInfo{
			ID:        res.ID,
			SID:       res.SID,
			Name:      res.Name,
			ExpiresAt: res.ExpiresAt.UTC(),
			CostLabel: res.CostLabel,
		})
	}

	return nodes, nil
}

package repository

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/db"
	"github.com/orris-inc/orris/internal/shared/errors"
)

// ExistsByListenPort checks if a rule with the given listen port exists (excluding soft-deleted records).
func (r *ForwardRuleRepositoryImpl) ExistsByListenPort(ctx context.Context, port uint16) (bool, error) {
	var count int64
	tx := db.GetTxFromContext(ctx, r.db)
	err := tx.Model(&models.ForwardRuleModel{}).
		Scopes(db.NotDeleted()).
		Where("listen_port = ?", port).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to check forward rule existence by listen port", "port", port, "error", err)
		return false, fmt.Errorf("failed to check forward rule existence: %w", err)
	}
	return count > 0, nil
}

// ExistsByAgentIDAndListenPort checks if a rule with the given agent ID and listen port exists.
// This is used for auto-assigning ports within an agent's scope.
func (r *ForwardRuleRepositoryImpl) ExistsByAgentIDAndListenPort(ctx context.Context, agentID uint, port uint16) (bool, error) {
	var count int64
	tx := db.GetTxFromContext(ctx, r.db)
	err := tx.Model(&models.ForwardRuleModel{}).
		Scopes(db.NotDeleted()).
		Where("agent_id = ? AND listen_port = ?", agentID, port).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to check forward rule existence by agent and port", "agent_id", agentID, "port", port, "error", err)
		return false, fmt.Errorf("failed to check forward rule existence: %w", err)
	}
	return count > 0, nil
}

// IsPortInUseByAgent checks if a port is in use by the specified agent across all rules.
// This includes both main rule ports (agent_id + listen_port) and chain_port_config entries.
func (r *ForwardRuleRepositoryImpl) IsPortInUseByAgent(ctx context.Context, agentID uint, port uint16, excludeRuleID uint) (bool, error) {
	var count int64
	tx := db.GetTxFromContext(ctx, r.db)

	// Build query to check both:
	// 1. Main rule: agent_id = ? AND listen_port = ?
	// 2. Chain port config: JSON_EXTRACT(chain_port_config, '$."<agent_id>"') = port
	// Note: MySQL JSON keys are strings, so we use the agent ID as a string key
	query := tx.Model(&models.ForwardRuleModel{}).
		Scopes(db.NotDeleted())

	// Exclude specific rule if provided (for update scenarios)
	if excludeRuleID > 0 {
		query = query.Where("id != ?", excludeRuleID)
	}

	// Check main rule ports OR chain_port_config entries
	// Use CAST for explicit type conversion to ensure correct comparison
	// JSON_EXTRACT returns JSON value, CAST converts it to unsigned integer for comparison
	err := query.Where(
		"(agent_id = ? AND listen_port = ?) OR (chain_port_config IS NOT NULL AND CAST(JSON_EXTRACT(chain_port_config, CONCAT('$.\"', ?, '\"')) AS UNSIGNED) = ?)",
		agentID, port, agentID, port,
	).Count(&count).Error

	if err != nil {
		r.logger.Errorw("failed to check port in use by agent",
			"agent_id", agentID,
			"port", port,
			"exclude_rule_id", excludeRuleID,
			"error", err,
		)
		return false, fmt.Errorf("failed to check port in use: %w", err)
	}

	return count > 0, nil
}

// UpdateTraffic updates the traffic counters for a rule.
func (r *ForwardRuleRepositoryImpl) UpdateTraffic(ctx context.Context, id uint, upload, download int64) error {
	tx := db.GetTxFromContext(ctx, r.db)
	result := tx.Model(&models.ForwardRuleModel{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"upload_bytes":   gorm.Expr("upload_bytes + ?", upload),
			"download_bytes": gorm.Expr("download_bytes + ?", download),
		})

	if result.Error != nil {
		r.logger.Errorw("failed to update forward rule traffic", "id", id, "error", result.Error)
		return fmt.Errorf("failed to update traffic: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("forward rule", fmt.Sprintf("%d", id))
	}

	return nil
}

// CountByUserID returns the total count of forward rules for a specific user (excluding soft-deleted records).
func (r *ForwardRuleRepositoryImpl) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	var count int64
	tx := db.GetTxFromContext(ctx, r.db)
	err := tx.Model(&models.ForwardRuleModel{}).
		Scopes(db.NotDeleted()).
		Where("user_id = ?", userID).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to count forward rules by user ID", "user_id", userID, "error", err)
		return 0, fmt.Errorf("failed to count forward rules by user ID: %w", err)
	}
	return count, nil
}

// CountBySubscriptionID returns the total count of forward rules for a specific subscription (excluding soft-deleted records).
func (r *ForwardRuleRepositoryImpl) CountBySubscriptionID(ctx context.Context, subscriptionID uint) (int64, error) {
	var count int64
	tx := db.GetTxFromContext(ctx, r.db)
	err := tx.Model(&models.ForwardRuleModel{}).
		Scopes(db.NotDeleted()).
		Where("subscription_id = ?", subscriptionID).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to count forward rules by subscription ID", "subscription_id", subscriptionID, "error", err)
		return 0, fmt.Errorf("failed to count forward rules by subscription ID: %w", err)
	}
	return count, nil
}

// GetTotalTrafficByUserID returns the total traffic (upload + download) for all rules owned by a user (excluding soft-deleted records).
func (r *ForwardRuleRepositoryImpl) GetTotalTrafficByUserID(ctx context.Context, userID uint) (int64, error) {
	var result struct {
		TotalTraffic int64
	}

	tx := db.GetTxFromContext(ctx, r.db)
	err := tx.Model(&models.ForwardRuleModel{}).
		Scopes(db.NotDeleted()).
		Select("COALESCE(SUM(upload_bytes + download_bytes), 0) as total_traffic").
		Where("user_id = ?", userID).
		Scan(&result).Error

	if err != nil {
		r.logger.Errorw("failed to get total traffic by user ID", "user_id", userID, "error", err)
		return 0, fmt.Errorf("failed to get total traffic by user ID: %w", err)
	}

	return result.TotalTraffic, nil
}

// UpdateSortOrders batch updates sort_order for multiple rules using a single CASE WHEN SQL.
func (r *ForwardRuleRepositoryImpl) UpdateSortOrders(ctx context.Context, ruleOrders map[uint]int) error {
	if len(ruleOrders) == 0 {
		return nil
	}

	// Build CASE WHEN SQL: UPDATE forward_rules SET sort_order = CASE id WHEN ? THEN ? ... END WHERE id IN (?,...)
	var sb strings.Builder
	sb.WriteString("UPDATE forward_rules SET sort_order = CASE id ")

	args := make([]interface{}, 0, len(ruleOrders)*2+len(ruleOrders))
	ids := make([]interface{}, 0, len(ruleOrders))

	for id, sortOrder := range ruleOrders {
		sb.WriteString("WHEN ? THEN ? ")
		args = append(args, id, sortOrder)
		ids = append(ids, id)
	}

	sb.WriteString("END WHERE id IN (")
	for i := range ids {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString("?")
	}
	sb.WriteString(")")
	args = append(args, ids...)

	tx := db.GetTxFromContext(ctx, r.db)
	if err := tx.Exec(sb.String(), args...).Error; err != nil {
		r.logger.Errorw("failed to batch update sort orders", "error", err, "count", len(ruleOrders))
		return fmt.Errorf("failed to batch update sort orders: %w", err)
	}

	r.logger.Infow("sort orders updated successfully", "count", len(ruleOrders))
	return nil
}

// AddGroupIDAtomically adds a group ID to a rule's group_ids array atomically.
// Returns true if the group ID was added, false if it already exists.
// Uses a single UPDATE statement with conditional logic to avoid TOCTOU race conditions.
func (r *ForwardRuleRepositoryImpl) AddGroupIDAtomically(ctx context.Context, ruleID uint, groupID uint) (bool, error) {
	tx := db.GetTxFromContext(ctx, r.db)

	// Single atomic UPDATE that:
	// 1. Only updates if the group ID doesn't already exist (via WHERE clause)
	// 2. Creates new array if NULL, otherwise appends
	// This avoids TOCTOU race conditions by combining check and update in one statement
	updateQuery := `
		UPDATE forward_rules
		SET group_ids = CASE
			WHEN group_ids IS NULL THEN JSON_ARRAY(?)
			ELSE JSON_ARRAY_APPEND(group_ids, '$', CAST(? AS UNSIGNED))
		END,
		updated_at = NOW()
		WHERE id = ? AND deleted_at IS NULL
		AND (group_ids IS NULL OR NOT JSON_CONTAINS(group_ids, CAST(? AS JSON)))
	`
	result := tx.Exec(updateQuery, groupID, groupID, ruleID, groupID)
	if result.Error != nil {
		r.logger.Errorw("failed to add group ID to rule atomically", "rule_id", ruleID, "group_id", groupID, "error", result.Error)
		return false, fmt.Errorf("failed to add group ID atomically: %w", result.Error)
	}

	// RowsAffected == 0 means either:
	// 1. Rule not found / deleted
	// 2. Group ID already exists in the array
	// We need to distinguish these cases
	if result.RowsAffected == 0 {
		// Check if rule exists
		var exists bool
		if err := tx.Raw("SELECT EXISTS(SELECT 1 FROM forward_rules WHERE id = ? AND deleted_at IS NULL)", ruleID).Scan(&exists).Error; err != nil {
			return false, fmt.Errorf("failed to check rule existence: %w", err)
		}
		if !exists {
			return false, fmt.Errorf("rule not found or already deleted")
		}
		// Rule exists but group ID already in array
		return false, nil
	}

	return true, nil
}

// RemoveGroupIDAtomically removes a group ID from a rule's group_ids array atomically.
// Returns true if the group ID was removed, false if it was not found.
// Uses JSON_TABLE (MySQL 8.0+) to rebuild the array excluding the target element,
// which correctly handles numeric values unlike JSON_SEARCH.
func (r *ForwardRuleRepositoryImpl) RemoveGroupIDAtomically(ctx context.Context, ruleID uint, groupID uint) (bool, error) {
	tx := db.GetTxFromContext(ctx, r.db)

	// Single atomic UPDATE that rebuilds the array excluding the target group ID
	// JSON_TABLE extracts array elements, we filter out the target and rebuild with JSON_ARRAYAGG
	// The WHERE clause ensures we only update if the group ID exists
	updateQuery := `
		UPDATE forward_rules fr
		SET fr.group_ids = (
			SELECT JSON_ARRAYAGG(jt.gid)
			FROM JSON_TABLE(fr.group_ids, '$[*]' COLUMNS(gid INT PATH '$')) AS jt
			WHERE jt.gid != ?
		),
		fr.updated_at = NOW()
		WHERE fr.id = ? AND fr.deleted_at IS NULL
		AND fr.group_ids IS NOT NULL
		AND JSON_CONTAINS(fr.group_ids, CAST(? AS JSON))
	`
	result := tx.Exec(updateQuery, groupID, ruleID, groupID)
	if result.Error != nil {
		r.logger.Errorw("failed to remove group ID from rule atomically", "rule_id", ruleID, "group_id", groupID, "error", result.Error)
		return false, fmt.Errorf("failed to remove group ID atomically: %w", result.Error)
	}

	return result.RowsAffected > 0, nil
}

// RemoveGroupIDFromAllRules removes a group ID from all rules that contain it.
// This is used when deleting a resource group to clean up orphaned references.
// Uses JSON_TABLE (MySQL 8.0+) to correctly handle numeric array values.
func (r *ForwardRuleRepositoryImpl) RemoveGroupIDFromAllRules(ctx context.Context, groupID uint) (int64, error) {
	tx := db.GetTxFromContext(ctx, r.db)

	// Use a subquery with JSON_TABLE to rebuild arrays excluding the target group ID
	// This correctly handles numeric values in JSON arrays
	updateQuery := `
		UPDATE forward_rules fr
		SET fr.group_ids = (
			SELECT JSON_ARRAYAGG(jt.gid)
			FROM JSON_TABLE(fr.group_ids, '$[*]' COLUMNS(gid INT PATH '$')) AS jt
			WHERE jt.gid != ?
		),
		fr.updated_at = NOW()
		WHERE fr.deleted_at IS NULL
		AND fr.group_ids IS NOT NULL
		AND JSON_CONTAINS(fr.group_ids, CAST(? AS JSON))
	`
	result := tx.Exec(updateQuery, groupID, groupID)
	if result.Error != nil {
		r.logger.Errorw("failed to remove group ID from all rules", "group_id", groupID, "error", result.Error)
		return 0, fmt.Errorf("failed to remove group ID from all rules: %w", result.Error)
	}

	r.logger.Infow("removed group ID from rules", "group_id", groupID, "affected_rows", result.RowsAffected)
	return result.RowsAffected, nil
}

// BatchAddGroupID adds a group ID to multiple rules atomically in a single transaction.
// Returns the number of rules that were updated (excludes rules that already had the group ID).
func (r *ForwardRuleRepositoryImpl) BatchAddGroupID(ctx context.Context, ruleIDs []uint, groupID uint) (int, error) {
	if len(ruleIDs) == 0 {
		return 0, nil
	}

	updated := 0
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Single UPDATE statement that affects all matching rules
		// Uses IN clause for efficiency
		updateQuery := `
			UPDATE forward_rules
			SET group_ids = CASE
				WHEN group_ids IS NULL THEN JSON_ARRAY(?)
				ELSE JSON_ARRAY_APPEND(group_ids, '$', CAST(? AS UNSIGNED))
			END,
			updated_at = NOW()
			WHERE id IN ? AND deleted_at IS NULL
			AND (group_ids IS NULL OR NOT JSON_CONTAINS(group_ids, CAST(? AS JSON)))
		`
		result := tx.Exec(updateQuery, groupID, groupID, ruleIDs, groupID)
		if result.Error != nil {
			return fmt.Errorf("failed to batch add group ID: %w", result.Error)
		}
		updated = int(result.RowsAffected)
		return nil
	})

	if err != nil {
		r.logger.Errorw("failed to batch add group ID to rules", "group_id", groupID, "rule_count", len(ruleIDs), "error", err)
		return 0, err
	}

	r.logger.Infow("batch added group ID to rules", "group_id", groupID, "updated_count", updated, "total_count", len(ruleIDs))
	return updated, nil
}

// BatchRemoveGroupID removes a group ID from multiple rules atomically in a single transaction.
// Returns the number of rules that were updated.
func (r *ForwardRuleRepositoryImpl) BatchRemoveGroupID(ctx context.Context, ruleIDs []uint, groupID uint) (int, error) {
	if len(ruleIDs) == 0 {
		return 0, nil
	}

	updated := 0
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Single UPDATE statement that affects all matching rules
		updateQuery := `
			UPDATE forward_rules fr
			SET fr.group_ids = (
				SELECT JSON_ARRAYAGG(jt.gid)
				FROM JSON_TABLE(fr.group_ids, '$[*]' COLUMNS(gid INT PATH '$')) AS jt
				WHERE jt.gid != ?
			),
			fr.updated_at = NOW()
			WHERE fr.id IN ? AND fr.deleted_at IS NULL
			AND fr.group_ids IS NOT NULL
			AND JSON_CONTAINS(fr.group_ids, CAST(? AS JSON))
		`
		result := tx.Exec(updateQuery, groupID, ruleIDs, groupID)
		if result.Error != nil {
			return fmt.Errorf("failed to batch remove group ID: %w", result.Error)
		}
		updated = int(result.RowsAffected)
		return nil
	})

	if err != nil {
		r.logger.Errorw("failed to batch remove group ID from rules", "group_id", groupID, "rule_count", len(ruleIDs), "error", err)
		return 0, err
	}

	r.logger.Infow("batch removed group ID from rules", "group_id", groupID, "updated_count", updated, "total_count", len(ruleIDs))
	return updated, nil
}

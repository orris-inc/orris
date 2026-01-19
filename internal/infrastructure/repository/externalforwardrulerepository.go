package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/externalforward"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ExternalForwardRuleRepository implements the externalforward.Repository interface.
type ExternalForwardRuleRepository struct {
	db     *gorm.DB
	mapper *mappers.ExternalForwardRuleMapper
	logger logger.Interface
}

// NewExternalForwardRuleRepository creates a new repository.
func NewExternalForwardRuleRepository(db *gorm.DB, log logger.Interface) *ExternalForwardRuleRepository {
	return &ExternalForwardRuleRepository{
		db:     db,
		mapper: mappers.NewExternalForwardRuleMapper(),
		logger: log,
	}
}

// Create persists a new external forward rule.
func (r *ExternalForwardRuleRepository) Create(ctx context.Context, rule *externalforward.ExternalForwardRule) error {
	model, err := r.mapper.ToModel(rule)
	if err != nil {
		return fmt.Errorf("failed to map rule to model: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create external forward rule: %w", err)
	}

	if err := rule.SetID(model.ID); err != nil {
		return fmt.Errorf("failed to set rule ID: %w", err)
	}

	return nil
}

// GetByID retrieves an external forward rule by ID.
func (r *ExternalForwardRuleRepository) GetByID(ctx context.Context, id uint) (*externalforward.ExternalForwardRule, error) {
	var model models.ExternalForwardRuleModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("external forward rule", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("failed to get external forward rule: %w", err)
	}

	return r.mapper.ToDomain(&model)
}

// GetBySID retrieves an external forward rule by SID.
func (r *ExternalForwardRuleRepository) GetBySID(ctx context.Context, sid string) (*externalforward.ExternalForwardRule, error) {
	var model models.ExternalForwardRuleModel
	if err := r.db.WithContext(ctx).Where("sid = ?", sid).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("external forward rule", sid)
		}
		return nil, fmt.Errorf("failed to get external forward rule: %w", err)
	}

	return r.mapper.ToDomain(&model)
}

// Update updates an existing external forward rule.
func (r *ExternalForwardRuleRepository) Update(ctx context.Context, rule *externalforward.ExternalForwardRule) error {
	model, err := r.mapper.ToModel(rule)
	if err != nil {
		return fmt.Errorf("failed to map rule to model: %w", err)
	}

	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return fmt.Errorf("failed to update external forward rule: %w", err)
	}

	return nil
}

// Delete removes an external forward rule.
func (r *ExternalForwardRuleRepository) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&models.ExternalForwardRuleModel{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete external forward rule: %w", err)
	}
	return nil
}

// ListBySubscriptionID returns all external forward rules for a specific subscription.
func (r *ExternalForwardRuleRepository) ListBySubscriptionID(ctx context.Context, subscriptionID uint) ([]*externalforward.ExternalForwardRule, error) {
	var modelList []*models.ExternalForwardRuleModel
	if err := r.db.WithContext(ctx).
		Where("subscription_id = ?", subscriptionID).
		Order("sort_order ASC, id ASC").
		Find(&modelList).Error; err != nil {
		return nil, fmt.Errorf("failed to list external forward rules: %w", err)
	}

	return r.mapper.ToDomainList(modelList)
}

// ListEnabledBySubscriptionID returns all enabled external forward rules for a specific subscription.
func (r *ExternalForwardRuleRepository) ListEnabledBySubscriptionID(ctx context.Context, subscriptionID uint) ([]*externalforward.ExternalForwardRule, error) {
	var modelList []*models.ExternalForwardRuleModel
	if err := r.db.WithContext(ctx).
		Where("subscription_id = ? AND status = ?", subscriptionID, "enabled").
		Order("sort_order ASC, id ASC").
		Find(&modelList).Error; err != nil {
		return nil, fmt.Errorf("failed to list enabled external forward rules: %w", err)
	}

	return r.mapper.ToDomainList(modelList)
}

// ListBySubscriptionIDWithPagination returns external forward rules for a subscription with filtering and pagination.
func (r *ExternalForwardRuleRepository) ListBySubscriptionIDWithPagination(ctx context.Context, subscriptionID uint, filter externalforward.ListFilter) ([]*externalforward.ExternalForwardRule, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.ExternalForwardRuleModel{}).Where("subscription_id = ?", subscriptionID)

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count external forward rules: %w", err)
	}

	// Apply ordering with whitelist validation to prevent SQL injection
	orderBy := filter.OrderBy
	allowedOrderBy := map[string]bool{
		"sort_order": true,
		"created_at": true,
		"updated_at": true,
		"name":       true,
		"status":     true,
	}
	if !allowedOrderBy[orderBy] {
		orderBy = "sort_order"
	}
	order := filter.Order
	if order != "DESC" && order != "desc" {
		order = "ASC"
	}
	query = query.Order(fmt.Sprintf("%s %s, id ASC", orderBy, order))

	// Apply pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	var modelList []*models.ExternalForwardRuleModel
	if err := query.Find(&modelList).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list external forward rules: %w", err)
	}

	rules, err := r.mapper.ToDomainList(modelList)
	if err != nil {
		return nil, 0, err
	}

	return rules, total, nil
}

// ListByUserID returns external forward rules for a specific user with filtering and pagination.
func (r *ExternalForwardRuleRepository) ListByUserID(ctx context.Context, userID uint, filter externalforward.ListFilter) ([]*externalforward.ExternalForwardRule, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.ExternalForwardRuleModel{}).Where("user_id = ?", userID)

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count external forward rules: %w", err)
	}

	// Apply ordering with whitelist validation to prevent SQL injection
	orderBy := filter.OrderBy
	allowedOrderBy := map[string]bool{
		"sort_order": true,
		"created_at": true,
		"updated_at": true,
		"name":       true,
		"status":     true,
	}
	if !allowedOrderBy[orderBy] {
		orderBy = "sort_order"
	}
	order := filter.Order
	if order != "DESC" && order != "desc" {
		order = "ASC"
	}
	query = query.Order(fmt.Sprintf("%s %s, id ASC", orderBy, order))

	// Apply pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	var modelList []*models.ExternalForwardRuleModel
	if err := query.Find(&modelList).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list external forward rules: %w", err)
	}

	rules, err := r.mapper.ToDomainList(modelList)
	if err != nil {
		return nil, 0, err
	}

	return rules, total, nil
}

// CountBySubscriptionID returns the total count of external forward rules for a specific subscription.
func (r *ExternalForwardRuleRepository) CountBySubscriptionID(ctx context.Context, subscriptionID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.ExternalForwardRuleModel{}).
		Where("subscription_id = ?", subscriptionID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count external forward rules: %w", err)
	}
	return count, nil
}

// UpdateSortOrders batch updates sort_order for multiple rules.
func (r *ExternalForwardRuleRepository) UpdateSortOrders(ctx context.Context, ruleOrders map[uint]int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for ruleID, order := range ruleOrders {
			if err := tx.Model(&models.ExternalForwardRuleModel{}).
				Where("id = ?", ruleID).
				Update("sort_order", order).Error; err != nil {
				return fmt.Errorf("failed to update sort order for rule %d: %w", ruleID, err)
			}
		}
		return nil
	})
}

// ListWithPagination returns external forward rules with optional filters and pagination (for admin use).
func (r *ExternalForwardRuleRepository) ListWithPagination(ctx context.Context, filter externalforward.AdminListFilter) ([]*externalforward.ExternalForwardRule, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.ExternalForwardRuleModel{})

	// Apply optional filters with whitelist validation for status
	if filter.SubscriptionID != nil {
		query = query.Where("subscription_id = ?", *filter.SubscriptionID)
	}

	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}

	// Whitelist validation for status to prevent invalid queries
	if filter.Status != "" {
		allowedStatus := map[string]bool{
			"enabled":  true,
			"disabled": true,
		}
		if allowedStatus[filter.Status] {
			query = query.Where("status = ?", filter.Status)
		}
	}

	// Filter by external source (allow any value as it's user-defined)
	if filter.ExternalSource != "" {
		query = query.Where("external_source = ?", filter.ExternalSource)
	}

	// Count total before pagination
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count external forward rules: %w", err)
	}

	// Apply ordering with whitelist validation to prevent SQL injection
	orderBy := filter.OrderBy
	allowedOrderBy := map[string]bool{
		"created_at":  true,
		"updated_at":  true,
		"name":        true,
		"status":      true,
		"listen_port": true,
		"sort_order":  true,
	}
	if !allowedOrderBy[orderBy] {
		orderBy = "created_at"
	}
	order := filter.Order
	if order != "ASC" && order != "asc" {
		order = "DESC"
	}
	query = query.Order(fmt.Sprintf("%s %s, id DESC", orderBy, order))

	// Apply pagination with safe defaults to prevent full table scan
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize
	query = query.Offset(offset).Limit(pageSize)

	var modelList []*models.ExternalForwardRuleModel
	if err := query.Find(&modelList).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list external forward rules: %w", err)
	}

	rules, err := r.mapper.ToDomainList(modelList)
	if err != nil {
		return nil, 0, err
	}

	return rules, total, nil
}

// ListByGroupID returns all external forward rules that belong to the specified resource group.
func (r *ExternalForwardRuleRepository) ListByGroupID(ctx context.Context, groupID uint, page, pageSize int) ([]*externalforward.ExternalForwardRule, int64, error) {
	// Build base query using CAST(? AS JSON) for proper numeric comparison
	baseQuery := r.db.WithContext(ctx).Model(&models.ExternalForwardRuleModel{}).
		Where("JSON_CONTAINS(group_ids, CAST(? AS JSON))", groupID)

	// Count total records
	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		r.logger.Errorw("failed to count external forward rules by group ID", "group_id", groupID, "error", err)
		return nil, 0, fmt.Errorf("failed to count external forward rules by group ID: %w", err)
	}

	// Build paginated query with sorting: sort_order ASC, created_at DESC
	var modelList []*models.ExternalForwardRuleModel
	query := r.db.WithContext(ctx).
		Where("JSON_CONTAINS(group_ids, CAST(? AS JSON))", groupID).
		Order("sort_order ASC, created_at DESC")

	// Apply pagination if specified
	if page > 0 && pageSize > 0 {
		offset := (page - 1) * pageSize
		query = query.Offset(offset).Limit(pageSize)
	}

	if err := query.Find(&modelList).Error; err != nil {
		r.logger.Errorw("failed to list external forward rules by group ID", "group_id", groupID, "error", err)
		return nil, 0, fmt.Errorf("failed to list external forward rules by group ID: %w", err)
	}

	rules, err := r.mapper.ToDomainList(modelList)
	if err != nil {
		return nil, 0, err
	}

	return rules, total, nil
}

// ListEnabledByGroupIDs returns all enabled external forward rules for the given resource groups.
func (r *ExternalForwardRuleRepository) ListEnabledByGroupIDs(ctx context.Context, groupIDs []uint) ([]*externalforward.ExternalForwardRule, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}

	// Build OR conditions for each group ID
	var modelList []*models.ExternalForwardRuleModel
	query := r.db.WithContext(ctx).
		Where("status = ?", "enabled")

	// Build JSON_CONTAINS conditions for any of the group IDs
	conditions := make([]string, len(groupIDs))
	args := make([]interface{}, len(groupIDs))
	for i, gid := range groupIDs {
		conditions[i] = "JSON_CONTAINS(group_ids, CAST(? AS JSON))"
		args[i] = gid
	}

	// Combine with OR
	orCondition := "(" + conditions[0]
	for i := 1; i < len(conditions); i++ {
		orCondition += " OR " + conditions[i]
	}
	orCondition += ")"

	query = query.Where(orCondition, args...).
		Order("sort_order ASC, id ASC")

	if err := query.Find(&modelList).Error; err != nil {
		r.logger.Errorw("failed to list enabled external forward rules by group IDs", "group_ids", groupIDs, "error", err)
		return nil, fmt.Errorf("failed to list enabled external forward rules by group IDs: %w", err)
	}

	return r.mapper.ToDomainList(modelList)
}

// GetBySIDs returns external forward rules for the given SIDs.
func (r *ExternalForwardRuleRepository) GetBySIDs(ctx context.Context, sids []string) (map[string]*externalforward.ExternalForwardRule, error) {
	if len(sids) == 0 {
		return make(map[string]*externalforward.ExternalForwardRule), nil
	}

	var modelList []*models.ExternalForwardRuleModel
	if err := r.db.WithContext(ctx).Where("sid IN ?", sids).Find(&modelList).Error; err != nil {
		r.logger.Errorw("failed to get external forward rules by SIDs", "count", len(sids), "error", err)
		return nil, fmt.Errorf("failed to get external forward rules by SIDs: %w", err)
	}

	result := make(map[string]*externalforward.ExternalForwardRule, len(modelList))
	for _, model := range modelList {
		entity, err := r.mapper.ToDomain(model)
		if err != nil {
			r.logger.Errorw("failed to map external forward rule model to entity", "sid", model.SID, "error", err)
			return nil, fmt.Errorf("failed to map external forward rule: %w", err)
		}
		result[model.SID] = entity
	}

	return result, nil
}

// AddGroupIDAtomically adds a group ID to a rule's group_ids array atomically using JSON_ARRAY_APPEND.
func (r *ExternalForwardRuleRepository) AddGroupIDAtomically(ctx context.Context, ruleID, groupID uint) (bool, error) {
	// Single atomic UPDATE that:
	// 1. Only updates if the group ID doesn't already exist (via WHERE clause)
	// 2. Creates new array if NULL, otherwise appends
	updateQuery := `
		UPDATE external_forward_rules
		SET group_ids = CASE
			WHEN group_ids IS NULL THEN JSON_ARRAY(?)
			ELSE JSON_ARRAY_APPEND(group_ids, '$', CAST(? AS UNSIGNED))
		END,
		updated_at = NOW()
		WHERE id = ? AND deleted_at IS NULL
		AND (group_ids IS NULL OR NOT JSON_CONTAINS(group_ids, CAST(? AS JSON)))
	`
	result := r.db.WithContext(ctx).Exec(updateQuery, groupID, groupID, ruleID, groupID)
	if result.Error != nil {
		r.logger.Errorw("failed to add group ID to external forward rule atomically", "rule_id", ruleID, "group_id", groupID, "error", result.Error)
		return false, fmt.Errorf("failed to add group ID atomically: %w", result.Error)
	}

	// RowsAffected == 0 means either rule not found or group ID already exists
	if result.RowsAffected == 0 {
		// Check if rule exists
		var exists bool
		if err := r.db.WithContext(ctx).Raw("SELECT EXISTS(SELECT 1 FROM external_forward_rules WHERE id = ? AND deleted_at IS NULL)", ruleID).Scan(&exists).Error; err != nil {
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

// RemoveGroupIDAtomically removes a group ID from a rule's group_ids array atomically using JSON_REMOVE.
func (r *ExternalForwardRuleRepository) RemoveGroupIDAtomically(ctx context.Context, ruleID, groupID uint) (bool, error) {
	// Single atomic UPDATE that rebuilds the array excluding the target group ID
	updateQuery := `
		UPDATE external_forward_rules fr
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
	result := r.db.WithContext(ctx).Exec(updateQuery, groupID, ruleID, groupID)
	if result.Error != nil {
		r.logger.Errorw("failed to remove group ID from external forward rule atomically", "rule_id", ruleID, "group_id", groupID, "error", result.Error)
		return false, fmt.Errorf("failed to remove group ID atomically: %w", result.Error)
	}

	return result.RowsAffected > 0, nil
}

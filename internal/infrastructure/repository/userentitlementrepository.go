package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/entitlement"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// UserEntitlementRepositoryImpl implements the entitlement.Repository interface
type UserEntitlementRepositoryImpl struct {
	db     *gorm.DB
	mapper mappers.EntitlementMapper
	logger logger.Interface
}

// NewUserEntitlementRepository creates a new user entitlement repository instance
func NewUserEntitlementRepository(db *gorm.DB, logger logger.Interface) entitlement.Repository {
	return &UserEntitlementRepositoryImpl{
		db:     db,
		mapper: mappers.NewEntitlementMapper(),
		logger: logger,
	}
}

// Create creates a new entitlement
func (r *UserEntitlementRepositoryImpl) Create(ctx context.Context, e *entitlement.Entitlement) error {
	// Convert domain entity to persistence model
	model, err := r.mapper.ToModel(e)
	if err != nil {
		r.logger.Errorw("failed to map entitlement entity to model", "error", err)
		return fmt.Errorf("failed to map entitlement entity: %w", err)
	}

	// Create in database
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.Errorw("failed to create entitlement in database",
			"subject_type", e.SubjectType(),
			"subject_id", e.SubjectID(),
			"resource_type", e.ResourceType(),
			"resource_id", e.ResourceID(),
			"error", err)
		return fmt.Errorf("failed to create entitlement: %w", err)
	}

	// Set the ID back to the entity
	if err := e.SetID(model.ID); err != nil {
		r.logger.Errorw("failed to set entitlement ID", "error", err)
		return fmt.Errorf("failed to set entitlement ID: %w", err)
	}

	r.logger.Infow("entitlement created successfully",
		"id", model.ID,
		"subject_type", model.SubjectType,
		"subject_id", model.SubjectID,
		"resource_type", model.ResourceType,
		"resource_id", model.ResourceID)

	return nil
}

// Update updates an existing entitlement
func (r *UserEntitlementRepositoryImpl) Update(ctx context.Context, e *entitlement.Entitlement) error {
	// Convert domain entity to persistence model
	model, err := r.mapper.ToModel(e)
	if err != nil {
		r.logger.Errorw("failed to map entitlement entity to model", "id", e.ID(), "error", err)
		return fmt.Errorf("failed to map entitlement entity: %w", err)
	}

	// Update in database
	result := r.db.WithContext(ctx).Model(&models.EntitlementModel{}).
		Where("id = ? AND version = ?", model.ID, model.Version-1).
		Updates(map[string]interface{}{
			"status":     model.Status,
			"expires_at": model.ExpiresAt,
			"metadata":   model.Metadata,
			"updated_at": model.UpdatedAt,
			"version":    model.Version,
		})

	if result.Error != nil {
		r.logger.Errorw("failed to update entitlement", "id", model.ID, "error", result.Error)
		return fmt.Errorf("failed to update entitlement: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("entitlement not found or version mismatch (optimistic lock)")
	}

	r.logger.Infow("entitlement updated successfully", "id", model.ID)
	return nil
}

// Delete deletes an entitlement by ID
func (r *UserEntitlementRepositoryImpl) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.EntitlementModel{}, id)

	if result.Error != nil {
		r.logger.Errorw("failed to delete entitlement", "id", id, "error", result.Error)
		return fmt.Errorf("failed to delete entitlement: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("entitlement not found")
	}

	r.logger.Infow("entitlement deleted successfully", "id", id)
	return nil
}

// GetByID retrieves an entitlement by ID
func (r *UserEntitlementRepositoryImpl) GetByID(ctx context.Context, id uint) (*entitlement.Entitlement, error) {
	var model models.EntitlementModel

	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get entitlement by ID", "id", id, "error", err)
		return nil, fmt.Errorf("failed to get entitlement: %w", err)
	}

	// Convert persistence model to domain entity
	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		r.logger.Errorw("failed to map entitlement model to entity", "id", id, "error", err)
		return nil, fmt.Errorf("failed to map entitlement: %w", err)
	}

	return entity, nil
}

// GetBySubject retrieves all entitlements for a subject
func (r *UserEntitlementRepositoryImpl) GetBySubject(ctx context.Context, subjectType entitlement.SubjectType, subjectID uint) ([]*entitlement.Entitlement, error) {
	var modelList []*models.EntitlementModel

	if err := r.db.WithContext(ctx).
		Where("subject_type = ? AND subject_id = ?", subjectType.String(), subjectID).
		Find(&modelList).Error; err != nil {
		r.logger.Errorw("failed to get entitlements by subject",
			"subject_type", subjectType,
			"subject_id", subjectID,
			"error", err)
		return nil, fmt.Errorf("failed to get entitlements by subject: %w", err)
	}

	// Convert models to entities
	entities, err := r.mapper.ToEntities(modelList)
	if err != nil {
		r.logger.Errorw("failed to map entitlement models to entities", "error", err)
		return nil, fmt.Errorf("failed to map entitlements: %w", err)
	}

	return entities, nil
}

// GetActiveBySubject retrieves all active entitlements for a subject
func (r *UserEntitlementRepositoryImpl) GetActiveBySubject(ctx context.Context, subjectType entitlement.SubjectType, subjectID uint) ([]*entitlement.Entitlement, error) {
	var modelList []*models.EntitlementModel

	now := time.Now()
	if err := r.db.WithContext(ctx).
		Where("subject_type = ? AND subject_id = ?", subjectType.String(), subjectID).
		Where("status = ?", entitlement.EntitlementStatusActive).
		Where("(expires_at IS NULL OR expires_at > ?)", now).
		Find(&modelList).Error; err != nil {
		r.logger.Errorw("failed to get active entitlements by subject",
			"subject_type", subjectType,
			"subject_id", subjectID,
			"error", err)
		return nil, fmt.Errorf("failed to get active entitlements: %w", err)
	}

	// Convert models to entities
	entities, err := r.mapper.ToEntities(modelList)
	if err != nil {
		r.logger.Errorw("failed to map entitlement models to entities", "error", err)
		return nil, fmt.Errorf("failed to map entitlements: %w", err)
	}

	return entities, nil
}

// GetByResource retrieves all entitlements for a specific resource
func (r *UserEntitlementRepositoryImpl) GetByResource(ctx context.Context, resourceType entitlement.ResourceType, resourceID uint) ([]*entitlement.Entitlement, error) {
	var modelList []*models.EntitlementModel

	if err := r.db.WithContext(ctx).
		Where("resource_type = ? AND resource_id = ?", resourceType.String(), resourceID).
		Find(&modelList).Error; err != nil {
		r.logger.Errorw("failed to get entitlements by resource",
			"resource_type", resourceType,
			"resource_id", resourceID,
			"error", err)
		return nil, fmt.Errorf("failed to get entitlements by resource: %w", err)
	}

	// Convert models to entities
	entities, err := r.mapper.ToEntities(modelList)
	if err != nil {
		r.logger.Errorw("failed to map entitlement models to entities", "error", err)
		return nil, fmt.Errorf("failed to map entitlements: %w", err)
	}

	return entities, nil
}

// GetBySource retrieves all entitlements from a specific source
func (r *UserEntitlementRepositoryImpl) GetBySource(ctx context.Context, sourceType entitlement.SourceType, sourceID uint) ([]*entitlement.Entitlement, error) {
	var modelList []*models.EntitlementModel

	if err := r.db.WithContext(ctx).
		Where("source_type = ? AND source_id = ?", sourceType.String(), sourceID).
		Find(&modelList).Error; err != nil {
		r.logger.Errorw("failed to get entitlements by source",
			"source_type", sourceType,
			"source_id", sourceID,
			"error", err)
		return nil, fmt.Errorf("failed to get entitlements by source: %w", err)
	}

	// Convert models to entities
	entities, err := r.mapper.ToEntities(modelList)
	if err != nil {
		r.logger.Errorw("failed to map entitlement models to entities", "error", err)
		return nil, fmt.Errorf("failed to map entitlements: %w", err)
	}

	return entities, nil
}

// Exists checks if an entitlement exists for a subject-resource pair
func (r *UserEntitlementRepositoryImpl) Exists(ctx context.Context, subjectType entitlement.SubjectType, subjectID uint,
	resourceType entitlement.ResourceType, resourceID uint) (bool, error) {
	var count int64

	if err := r.db.WithContext(ctx).
		Model(&models.EntitlementModel{}).
		Where("subject_type = ? AND subject_id = ? AND resource_type = ? AND resource_id = ?",
			subjectType.String(), subjectID, resourceType.String(), resourceID).
		Count(&count).Error; err != nil {
		r.logger.Errorw("failed to check entitlement existence",
			"subject_type", subjectType,
			"subject_id", subjectID,
			"resource_type", resourceType,
			"resource_id", resourceID,
			"error", err)
		return false, fmt.Errorf("failed to check entitlement existence: %w", err)
	}

	return count > 0, nil
}

// BatchCreate creates multiple entitlements in a single transaction
func (r *UserEntitlementRepositoryImpl) BatchCreate(ctx context.Context, entitlements []*entitlement.Entitlement) error {
	if len(entitlements) == 0 {
		return nil
	}

	// Convert entities to models
	modelList, err := r.mapper.ToModels(entitlements)
	if err != nil {
		r.logger.Errorw("failed to map entitlement entities to models", "error", err)
		return fmt.Errorf("failed to map entitlement entities: %w", err)
	}

	// Create in database
	if err := r.db.WithContext(ctx).Create(&modelList).Error; err != nil {
		r.logger.Errorw("failed to batch create entitlements", "count", len(modelList), "error", err)
		return fmt.Errorf("failed to batch create entitlements: %w", err)
	}

	// Set IDs back to entities
	for i, model := range modelList {
		if err := entitlements[i].SetID(model.ID); err != nil {
			r.logger.Warnw("failed to set entitlement ID after batch create", "index", i, "error", err)
		}
	}

	r.logger.Infow("entitlements batch created successfully", "count", len(modelList))
	return nil
}

// BatchUpdateStatus updates the status of multiple entitlements
func (r *UserEntitlementRepositoryImpl) BatchUpdateStatus(ctx context.Context, ids []uint, status entitlement.EntitlementStatus) error {
	if len(ids) == 0 {
		return nil
	}

	result := r.db.WithContext(ctx).
		Model(&models.EntitlementModel{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"status":     status.String(),
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		r.logger.Errorw("failed to batch update entitlement status", "ids", ids, "status", status, "error", result.Error)
		return fmt.Errorf("failed to batch update entitlement status: %w", result.Error)
	}

	r.logger.Infow("entitlement status batch updated successfully", "count", result.RowsAffected, "status", status)
	return nil
}

// RevokeBySource revokes all entitlements from a specific source
func (r *UserEntitlementRepositoryImpl) RevokeBySource(ctx context.Context, sourceType entitlement.SourceType, sourceID uint) error {
	result := r.db.WithContext(ctx).
		Model(&models.EntitlementModel{}).
		Where("source_type = ? AND source_id = ?", sourceType.String(), sourceID).
		Where("status = ?", entitlement.EntitlementStatusActive).
		Updates(map[string]interface{}{
			"status":     entitlement.EntitlementStatusRevoked.String(),
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		r.logger.Errorw("failed to revoke entitlements by source",
			"source_type", sourceType,
			"source_id", sourceID,
			"error", result.Error)
		return fmt.Errorf("failed to revoke entitlements by source: %w", result.Error)
	}

	r.logger.Infow("entitlements revoked by source",
		"source_type", sourceType,
		"source_id", sourceID,
		"count", result.RowsAffected)

	return nil
}

// GetExpiredEntitlements retrieves all entitlements that have passed their expiration time
// but haven't been marked as expired yet
func (r *UserEntitlementRepositoryImpl) GetExpiredEntitlements(ctx context.Context) ([]*entitlement.Entitlement, error) {
	var modelList []*models.EntitlementModel

	now := time.Now()
	if err := r.db.WithContext(ctx).
		Where("status = ?", entitlement.EntitlementStatusActive).
		Where("expires_at IS NOT NULL AND expires_at <= ?", now).
		Find(&modelList).Error; err != nil {
		r.logger.Errorw("failed to get expired entitlements", "error", err)
		return nil, fmt.Errorf("failed to get expired entitlements: %w", err)
	}

	// Convert models to entities
	entities, err := r.mapper.ToEntities(modelList)
	if err != nil {
		r.logger.Errorw("failed to map entitlement models to entities", "error", err)
		return nil, fmt.Errorf("failed to map entitlements: %w", err)
	}

	return entities, nil
}

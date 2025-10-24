package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"orris/internal/domain/notification"
	vo "orris/internal/domain/notification/value_objects"
	"orris/internal/infrastructure/persistence/mappers"
	"orris/internal/infrastructure/persistence/models"
	"orris/internal/shared/errors"
)

type AnnouncementRepositoryImpl struct {
	db     *gorm.DB
	mapper mappers.AnnouncementMapper
}

func NewAnnouncementRepository(db *gorm.DB) notification.AnnouncementRepository {
	return &AnnouncementRepositoryImpl{
		db:     db,
		mapper: mappers.NewAnnouncementMapper(),
	}
}

func (r *AnnouncementRepositoryImpl) Create(ctx context.Context, announcement *notification.Announcement) error {
	model, err := r.mapper.ToModel(announcement)
	if err != nil {
		return fmt.Errorf("failed to map announcement entity to model: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create announcement: %w", err)
	}

	if err := announcement.SetID(model.ID); err != nil {
		return fmt.Errorf("failed to set announcement ID: %w", err)
	}

	return nil
}

func (r *AnnouncementRepositoryImpl) GetByID(ctx context.Context, id uint) (*notification.Announcement, error) {
	var model models.AnnouncementModel

	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get announcement by ID: %w", err)
	}

	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		return nil, fmt.Errorf("failed to map announcement model to entity: %w", err)
	}

	return entity, nil
}

func (r *AnnouncementRepositoryImpl) Update(ctx context.Context, announcement *notification.Announcement) error {
	model, err := r.mapper.ToModel(announcement)
	if err != nil {
		return fmt.Errorf("failed to map announcement entity to model: %w", err)
	}

	result := r.db.WithContext(ctx).Save(model)
	if result.Error != nil {
		return fmt.Errorf("failed to update announcement: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("announcement not found")
	}

	return nil
}

func (r *AnnouncementRepositoryImpl) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.AnnouncementModel{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete announcement: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("announcement not found")
	}

	return nil
}

func (r *AnnouncementRepositoryImpl) List(ctx context.Context, limit, offset int) ([]*notification.Announcement, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&models.AnnouncementModel{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count announcements: %w", err)
	}

	var modelList []*models.AnnouncementModel
	query := r.db.WithContext(ctx).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&modelList).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list announcements: %w", err)
	}

	entities, err := r.mapper.ToEntities(modelList)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to map announcement models to entities: %w", err)
	}

	return entities, total, nil
}

func (r *AnnouncementRepositoryImpl) FindBySpecification(
	ctx context.Context,
	spec notification.Specification,
	limit, offset int,
) ([]*notification.Announcement, int64, error) {
	var modelList []*models.AnnouncementModel

	query := r.db.WithContext(ctx).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&modelList).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to find announcements by specification: %w", err)
	}

	entities, err := r.mapper.ToEntities(modelList)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to map announcement models to entities: %w", err)
	}

	filtered := make([]*notification.Announcement, 0)
	for _, entity := range entities {
		if spec.IsSatisfiedBy(entity) {
			filtered = append(filtered, entity)
		}
	}

	return filtered, int64(len(filtered)), nil
}

func (r *AnnouncementRepositoryImpl) IncrementViewCount(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Model(&models.AnnouncementModel{}).
		Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + ?", 1))

	if result.Error != nil {
		return fmt.Errorf("failed to increment view count: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("announcement not found")
	}

	return nil
}

func (r *AnnouncementRepositoryImpl) FindByStatus(
	ctx context.Context,
	status vo.AnnouncementStatus,
	limit, offset int,
) ([]*notification.Announcement, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&models.AnnouncementModel{}).Where("status = ?", status.String())

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count announcements by status: %w", err)
	}

	var modelList []*models.AnnouncementModel
	query = query.Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&modelList).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to find announcements by status: %w", err)
	}

	entities, err := r.mapper.ToEntities(modelList)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to map announcement models to entities: %w", err)
	}

	return entities, total, nil
}

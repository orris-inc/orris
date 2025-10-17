package repository

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"orris/internal/domain/user"
	"orris/internal/domain/user/specifications"
	domainEvents "orris/internal/domain/shared/events"
	"orris/internal/infrastructure/persistence/mappers"
	"orris/internal/infrastructure/persistence/models"
	"orris/internal/shared/logger"
)

// UserRepositoryDDD implements the user repository interface with DDD patterns
type UserRepositoryDDD struct {
	db              *gorm.DB
	mapper          mappers.UserMapper
	eventDispatcher domainEvents.EventDispatcher
	logger          logger.Interface
}

// NewUserRepositoryDDD creates a new DDD user repository
func NewUserRepositoryDDD(db *gorm.DB, eventDispatcher domainEvents.EventDispatcher, logger logger.Interface) user.RepositoryWithSpecifications {
	return &UserRepositoryDDD{
		db:              db,
		mapper:          mappers.NewUserMapper(),
		eventDispatcher: eventDispatcher,
		logger:          logger,
	}
}

// Create creates a new user
func (r *UserRepositoryDDD) Create(ctx context.Context, userEntity *user.User) error {
	// Convert domain entity to persistence model
	model, err := r.mapper.ToModel(userEntity)
	if err != nil {
		r.logger.Errorw("failed to map user entity to model", zap.Error(err))
		return fmt.Errorf("failed to map user entity: %w", err)
	}

	// Create in database
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.Errorw("failed to create user in database", zap.Error(err))
		return fmt.Errorf("failed to create user: %w", err)
	}

	// Set the ID back to the entity
	if err := userEntity.SetID(model.ID); err != nil {
		r.logger.Errorw("failed to set user ID", "error", err)
		return fmt.Errorf("failed to set user ID: %w", err)
	}

	// Publish domain events
	if r.eventDispatcher != nil {
		events := userEntity.GetEvents()
		for _, event := range events {
			if domainEvent, ok := event.(domainEvents.DomainEvent); ok {
				if err := r.eventDispatcher.Publish(domainEvent); err != nil {
					r.logger.Errorw("failed to publish domain event", "event_type", domainEvent.GetEventType(), "error", err)
					// Don't fail the creation due to event publishing failure
					// In production, consider using outbox pattern
				}
			}
		}
	}

	r.logger.Infow("user created successfully", "id", model.ID, "email", model.Email)
	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepositoryDDD) GetByID(ctx context.Context, id uint) (*user.User, error) {
	var model models.UserModel
	
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get user by ID", "id", id, "error", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Convert persistence model to domain entity
	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		r.logger.Errorw("failed to map user model to entity", "id", id, "error", err)
		return nil, fmt.Errorf("failed to map user: %w", err)
	}

	return entity, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepositoryDDD) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	var model models.UserModel
	
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get user by email", "email", email, "error", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Convert persistence model to domain entity
	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		r.logger.Errorw("failed to map user model to entity", "email", email, zap.Error(err))
		return nil, fmt.Errorf("failed to map user: %w", err)
	}

	return entity, nil
}

// Update updates an existing user
func (r *UserRepositoryDDD) Update(ctx context.Context, userEntity *user.User) error {
	// Convert domain entity to persistence model
	model, err := r.mapper.ToModel(userEntity)
	if err != nil {
		r.logger.Errorw("failed to map user entity to model", "id", userEntity.ID(), zap.Error(err))
		return fmt.Errorf("failed to map user entity: %w", err)
	}

	// Wrap update and event publishing in a transaction
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Update with optimistic locking
		result := tx.Model(model).
			Where("id = ? AND version = ?", model.ID, model.Version).
			Updates(map[string]interface{}{
				"email":      model.Email,
				"name":       model.Name,
				"status":     model.Status,
				"version":    model.Version + 1,
				"updated_at": model.UpdatedAt,
			})

		if result.Error != nil {
			r.logger.Errorw("failed to update user", "id", model.ID, "error", result.Error)
			return fmt.Errorf("failed to update user: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			return fmt.Errorf("user not found or version mismatch (optimistic lock failed)")
		}

		// Publish domain events after successful update
		if r.eventDispatcher != nil {
			events := userEntity.GetEvents()
			for _, event := range events {
				if domainEvent, ok := event.(domainEvents.DomainEvent); ok {
					if err := r.eventDispatcher.Publish(domainEvent); err != nil {
						r.logger.Errorw("failed to publish domain event", "event_type", domainEvent.GetEventType(), "error", err)
						// Don't fail the transaction due to event publishing failure
						// In production, consider using outbox pattern
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	r.logger.Infow("user updated successfully", "id", model.ID)
	return nil
}

// Delete soft deletes a user
func (r *UserRepositoryDDD) Delete(ctx context.Context, id uint) error {
	// First get the user to trigger domain events
	userEntity, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if userEntity == nil {
		return fmt.Errorf("user not found")
	}

	// Mark as deleted in domain
	if err := userEntity.Delete(); err != nil {
		return fmt.Errorf("failed to delete user in domain: %w", err)
	}

	// Soft delete in database
	if err := r.db.WithContext(ctx).Delete(&models.UserModel{}, id).Error; err != nil {
		r.logger.Errorw("failed to delete user", "id", id, zap.Error(err))
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// Publish domain events
	if r.eventDispatcher != nil {
		entityEvents := userEntity.GetEvents()
		for _, event := range entityEvents {
			if domainEvent, ok := event.(domainEvents.DomainEvent); ok {
				if err := r.eventDispatcher.Publish(domainEvent); err != nil {
					r.logger.Errorw("failed to publish domain event", "error", err)
				}
			}
		}
	}

	r.logger.Infow("user deleted successfully", "id", id)
	return nil
}

// List retrieves a paginated list of users
func (r *UserRepositoryDDD) List(ctx context.Context, filter user.ListFilter) ([]*user.User, int64, error) {
	var models []*models.UserModel
	var total int64

	query := r.db.WithContext(ctx).Table("users")

	// Apply filters
	if filter.Email != "" {
		query = query.Where("email LIKE ?", "%"+filter.Email+"%")
	}
	if filter.Name != "" {
		query = query.Where("name LIKE ?", "%"+filter.Name+"%")
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		r.logger.Errorw("failed to count users", zap.Error(err))
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Apply sorting
	if filter.OrderBy != "" {
		order := "ASC"
		if filter.Order == "desc" {
			order = "DESC"
		}
		query = query.Order(fmt.Sprintf("%s %s", filter.OrderBy, order))
	} else {
		query = query.Order("created_at DESC")
	}

	// Apply pagination
	offset := (filter.Page - 1) * filter.PageSize
	query = query.Offset(offset).Limit(filter.PageSize)

	// Execute query
	if err := query.Find(&models).Error; err != nil {
		r.logger.Errorw("failed to list users", zap.Error(err))
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	// Convert models to entities
	entities, err := r.mapper.ToEntities(models)
	if err != nil {
		r.logger.Errorw("failed to map user models to entities", zap.Error(err))
		return nil, 0, fmt.Errorf("failed to map users: %w", err)
	}

	return entities, total, nil
}

// Exists checks if a user exists by ID
func (r *UserRepositoryDDD) Exists(ctx context.Context, id uint) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.UserModel{}).Where("id = ?", id).Count(&count).Error; err != nil {
		r.logger.Errorw("failed to check user existence", "id", id, zap.Error(err))
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return count > 0, nil
}

// ExistsByEmail checks if a user exists by email
func (r *UserRepositoryDDD) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.UserModel{}).Where("email = ?", email).Count(&count).Error; err != nil {
		r.logger.Errorw("failed to check user existence by email", "email", email, zap.Error(err))
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return count > 0, nil
}

// FindBySpecification finds users by specification
func (r *UserRepositoryDDD) FindBySpecification(ctx context.Context, spec interface{}, limit int) ([]*user.User, error) {
	query := r.db.WithContext(ctx).Table("users")
	
	// Apply specification to query
	if spec != nil {
		if specification, ok := spec.(specifications.Specification); ok {
			sql, args := specification.ToSQL()
			query = query.Where(sql, args...)
		}
	}
	
	// Apply limit
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	var models []*models.UserModel
	if err := query.Find(&models).Error; err != nil {
		r.logger.Errorw("failed to find users by specification", zap.Error(err))
		return nil, fmt.Errorf("failed to find users: %w", err)
	}
	
	// Convert models to entities
	entities, err := r.mapper.ToEntities(models)
	if err != nil {
		r.logger.Errorw("failed to map user models to entities", zap.Error(err))
		return nil, fmt.Errorf("failed to map users: %w", err)
	}
	
	return entities, nil
}
// GetByVerificationToken retrieves a user by email verification token
func (r *UserRepositoryDDD) GetByVerificationToken(ctx context.Context, token string) (*user.User, error) {
	var model models.UserModel
	
	if err := r.db.WithContext(ctx).Where("email_verification_token = ?", token).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		r.logger.Errorw("failed to get user by verification token", "error", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		r.logger.Errorw("failed to map user model to entity", "error", err)
		return nil, fmt.Errorf("failed to map user: %w", err)
	}

	return entity, nil
}

// GetByPasswordResetToken retrieves a user by password reset token
func (r *UserRepositoryDDD) GetByPasswordResetToken(ctx context.Context, token string) (*user.User, error) {
	var model models.UserModel
	
	if err := r.db.WithContext(ctx).Where("password_reset_token = ?", token).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		r.logger.Errorw("failed to get user by reset token", "error", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		r.logger.Errorw("failed to map user model to entity", "error", err)
		return nil, fmt.Errorf("failed to map user: %w", err)
	}

	return entity, nil
}

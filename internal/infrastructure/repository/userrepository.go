package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/db"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// allowedUserOrderByFields defines the whitelist of allowed ORDER BY fields
// to prevent SQL injection attacks.
var allowedUserOrderByFields = map[string]bool{
	"id":         true,
	"sid":        true,
	"email":      true,
	"name":       true,
	"role":       true,
	"status":     true,
	"created_at": true,
	"updated_at": true,
}

// UserRepository implements the user repository interface with DDD patterns
type UserRepository struct {
	db     *gorm.DB
	mapper mappers.UserMapper
	logger logger.Interface
}

// NewUserRepository creates a new DDD user repository
func NewUserRepository(db *gorm.DB, logger logger.Interface) user.Repository {
	return &UserRepository{
		db:     db,
		mapper: mappers.NewUserMapper(),
		logger: logger,
	}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, userEntity *user.User) error {
	// Convert domain entity to persistence model
	model, err := r.mapper.ToModel(userEntity)
	if err != nil {
		r.logger.Errorw("failed to map user entity to model", "error", err)
		return fmt.Errorf("failed to map user entity: %w", err)
	}

	// Create in database
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.Errorw("failed to create user in database", "error", err)
		return fmt.Errorf("failed to create user: %w", err)
	}

	// Set the ID back to the entity
	if err := userEntity.SetID(model.ID); err != nil {
		r.logger.Errorw("failed to set user ID", "error", err)
		return fmt.Errorf("failed to set user ID: %w", err)
	}

	r.logger.Infow("user created successfully", "id", model.ID, "email", model.Email)
	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uint) (*user.User, error) {
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

// GetByIDs retrieves multiple users by internal IDs
func (r *UserRepository) GetByIDs(ctx context.Context, ids []uint) ([]*user.User, error) {
	if len(ids) == 0 {
		return []*user.User{}, nil
	}

	var userModels []*models.UserModel
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&userModels).Error; err != nil {
		r.logger.Errorw("failed to get users by IDs", "ids", ids, "error", err)
		return nil, fmt.Errorf("failed to get users by IDs: %w", err)
	}

	users := make([]*user.User, 0, len(userModels))
	for _, model := range userModels {
		entity, err := r.mapper.ToEntity(model)
		if err != nil {
			r.logger.Warnw("failed to map user model to entity, skipping", "id", model.ID, "error", err)
			continue
		}
		users = append(users, entity)
	}

	return users, nil
}

// GetBySID retrieves a user by external SID (Stripe-style ID)
func (r *UserRepository) GetBySID(ctx context.Context, sid string) (*user.User, error) {
	var model models.UserModel

	if err := r.db.WithContext(ctx).Where("sid = ?", sid).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get user by SID", "sid", sid, "error", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Convert persistence model to domain entity
	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		r.logger.Errorw("failed to map user model to entity", "sid", sid, "error", err)
		return nil, fmt.Errorf("failed to map user: %w", err)
	}

	return entity, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
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
		r.logger.Errorw("failed to map user model to entity", "email", email, "error", err)
		return nil, fmt.Errorf("failed to map user: %w", err)
	}

	return entity, nil
}

// Update updates an existing user
func (r *UserRepository) Update(ctx context.Context, userEntity *user.User) error {
	// Convert domain entity to persistence model
	model, err := r.mapper.ToModel(userEntity)
	if err != nil {
		r.logger.Errorw("failed to map user entity to model", "id", userEntity.ID(), "error", err)
		return fmt.Errorf("failed to map user entity: %w", err)
	}

	result := r.db.WithContext(ctx).Model(&models.UserModel{}).
		Where("id = ?", model.ID).
		Updates(map[string]interface{}{
			"email":                         model.Email,
			"name":                          model.Name,
			"role":                          model.Role,
			"status":                        model.Status,
			"version":                       model.Version,
			"updated_at":                    model.UpdatedAt,
			"password_hash":                 model.PasswordHash,
			"email_verified":                model.EmailVerified,
			"email_verification_token":      model.EmailVerificationToken,
			"email_verification_expires_at": model.EmailVerificationExpiresAt,
			"password_reset_token":          model.PasswordResetToken,
			"password_reset_expires_at":     model.PasswordResetExpiresAt,
			"last_password_change_at":       model.LastPasswordChangeAt,
			"failed_login_attempts":         model.FailedLoginAttempts,
			"locked_until":                  model.LockedUntil,
		})

	if result.Error != nil {
		r.logger.Errorw("failed to update user", "id", model.ID, "error", result.Error)
		return fmt.Errorf("failed to update user: %w", result.Error)
	}

	// Note: RowsAffected may be 0 when updated values are identical to existing values.
	// This is expected MySQL behavior and should not be treated as "user not found".

	r.logger.Infow("user updated successfully", "id", model.ID)
	return nil
}

// Delete soft deletes a user by internal ID
func (r *UserRepository) Delete(ctx context.Context, id uint) error {
	// Soft delete in database - updates both status and deleted_at
	result := r.db.WithContext(ctx).
		Model(&models.UserModel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     "deleted",
			"deleted_at": r.db.NowFunc(),
		})

	if result.Error != nil {
		r.logger.Errorw("failed to delete user", "id", id, "error", result.Error)
		return fmt.Errorf("failed to delete user: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	r.logger.Infow("user deleted successfully", "id", id)
	return nil
}

// DeleteBySID soft deletes a user by external SID
func (r *UserRepository) DeleteBySID(ctx context.Context, sid string) error {
	// Soft delete in database - updates both status and deleted_at
	result := r.db.WithContext(ctx).
		Model(&models.UserModel{}).
		Where("sid = ?", sid).
		Updates(map[string]interface{}{
			"status":     "deleted",
			"deleted_at": r.db.NowFunc(),
		})

	if result.Error != nil {
		r.logger.Errorw("failed to delete user", "sid", sid, "error", result.Error)
		return fmt.Errorf("failed to delete user: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	r.logger.Infow("user deleted successfully", "sid", sid)
	return nil
}

// List retrieves a paginated list of users
func (r *UserRepository) List(ctx context.Context, filter user.ListFilter) ([]*user.User, int64, error) {
	var userModels []*models.UserModel
	var total int64

	// Use Model() instead of Table() to ensure soft delete filtering works
	query := r.db.WithContext(ctx).Model(&models.UserModel{})

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
	if filter.Role != "" {
		query = query.Where("role = ?", filter.Role)
	}

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		r.logger.Errorw("failed to count users", "error", err)
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Apply sorting with whitelist validation to prevent SQL injection
	orderBy := filter.OrderBy
	if orderBy == "" || !allowedUserOrderByFields[orderBy] {
		query = query.Order("created_at DESC")
	} else {
		order := strings.ToUpper(filter.Order)
		if order != "ASC" && order != "DESC" {
			order = "DESC"
		}
		query = query.Order(fmt.Sprintf("%s %s", orderBy, order))
	}

	// Apply pagination
	offset := (filter.Page - 1) * filter.PageSize
	query = query.Offset(offset).Limit(filter.PageSize)

	// Execute query
	if err := query.Find(&userModels).Error; err != nil {
		r.logger.Errorw("failed to list users", "error", err)
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	// Convert models to entities
	entities, err := r.mapper.ToEntities(userModels)
	if err != nil {
		r.logger.Errorw("failed to map user models to entities", "error", err)
		return nil, 0, fmt.Errorf("failed to map users: %w", err)
	}

	return entities, total, nil
}

// Exists checks if a user exists by ID (excluding soft-deleted records)
func (r *UserRepository) Exists(ctx context.Context, id uint) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.UserModel{}).
		Scopes(db.NotDeleted()).
		Where("id = ?", id).
		Count(&count).Error; err != nil {
		r.logger.Errorw("failed to check user existence", "id", id, "error", err)
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return count > 0, nil
}

// ExistsByEmail checks if a user exists by email (excluding soft-deleted records)
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.UserModel{}).
		Scopes(db.NotDeleted()).
		Where("email = ?", email).
		Count(&count).Error; err != nil {
		r.logger.Errorw("failed to check user existence by email", "email", email, "error", err)
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return count > 0, nil
}

// GetByVerificationToken retrieves a user by email verification token
func (r *UserRepository) GetByVerificationToken(ctx context.Context, token string) (*user.User, error) {
	var model models.UserModel

	// Hash the token before querying database
	tokenHash := hashToken(token)

	if err := r.db.WithContext(ctx).Where("email_verification_token = ?", tokenHash).First(&model).Error; err != nil {
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
func (r *UserRepository) GetByPasswordResetToken(ctx context.Context, token string) (*user.User, error) {
	var model models.UserModel

	// Hash the token before querying database
	tokenHash := hashToken(token)

	if err := r.db.WithContext(ctx).Where("password_reset_token = ?", tokenHash).First(&model).Error; err != nil {
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

// hashToken computes SHA256 hash of the token for database lookup
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

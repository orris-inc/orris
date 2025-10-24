package repository

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"orris/internal/domain/user"
	"orris/internal/shared/constants"
	"orris/internal/shared/errors"
)

// UserRepository implements the user.Repository interface
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) user.Repository {
	return &UserRepository{
		db: db,
	}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	if err := r.db.WithContext(ctx).Create(u).Error; err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") && strings.Contains(err.Error(), "email") {
			return errors.NewConflictError("User with this email already exists")
		}
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uint) (*user.User, error) {
	var u user.User
	err := r.db.WithContext(ctx).First(&u, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}
	return &u, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	var u user.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return &u, nil
}

// Update updates an existing user
func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	result := r.db.WithContext(ctx).Save(u)
	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "Duplicate entry") && strings.Contains(result.Error.Error(), "email") {
			return errors.NewConflictError("User with this email already exists")
		}
		return fmt.Errorf("failed to update user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("User not found")
	}
	return nil
}

// Delete soft deletes a user
func (r *UserRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&user.User{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("User not found")
	}
	return nil
}

// List retrieves a paginated list of users
func (r *UserRepository) List(ctx context.Context, filter user.ListFilter) ([]*user.User, int64, error) {
	query := r.db.WithContext(ctx).Model(&user.User{})

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
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Apply ordering
	orderBy := "created_at"
	order := "desc"
	if filter.OrderBy != "" {
		orderBy = filter.OrderBy
	}
	if filter.Order != "" && (strings.ToLower(filter.Order) == "asc" || strings.ToLower(filter.Order) == "desc") {
		order = strings.ToLower(filter.Order)
	}
	query = query.Order(fmt.Sprintf("%s %s", orderBy, order))

	// Apply pagination
	page := filter.Page
	if page < 1 {
		page = constants.DefaultPage
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = constants.DefaultPageSize
	}
	if pageSize > constants.MaxPageSize {
		pageSize = constants.MaxPageSize
	}

	offset := (page - 1) * pageSize
	query = query.Offset(offset).Limit(pageSize)

	// Execute query
	var users []*user.User
	if err := query.Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	return users, total, nil
}

// Exists checks if a user exists by ID
func (r *UserRepository) Exists(ctx context.Context, id uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&user.User{}).Where("id = ?", id).Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}
	return count > 0, nil
}

// ExistsByEmail checks if a user exists by email
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&user.User{}).Where("email = ?", email).Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check user existence by email: %w", err)
	}
	return count > 0, nil
}

// GetByVerificationToken retrieves a user by email verification token
func (r *UserRepository) GetByVerificationToken(ctx context.Context, token string) (*user.User, error) {
	var u user.User
	err := r.db.WithContext(ctx).Where("email_verification_token = ?", token).First(&u).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("User not found")
		}
		return nil, fmt.Errorf("failed to get user by verification token: %w", err)
	}
	return &u, nil
}

// GetByPasswordResetToken retrieves a user by password reset token
func (r *UserRepository) GetByPasswordResetToken(ctx context.Context, token string) (*user.User, error) {
	var u user.User
	err := r.db.WithContext(ctx).Where("password_reset_token = ?", token).First(&u).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("User not found")
		}
		return nil, fmt.Errorf("failed to get user by password reset token: %w", err)
	}
	return &u, nil
}

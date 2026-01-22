package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// PasskeyCredentialRepository implements the passkey credential repository interface
type PasskeyCredentialRepository struct {
	db     *gorm.DB
	mapper mappers.PasskeyCredentialMapper
	logger logger.Interface
}

// NewPasskeyCredentialRepository creates a new passkey credential repository
func NewPasskeyCredentialRepository(db *gorm.DB, logger logger.Interface) user.PasskeyCredentialRepository {
	return &PasskeyCredentialRepository{
		db:     db,
		mapper: mappers.NewPasskeyCredentialMapper(),
		logger: logger,
	}
}

// Create creates a new passkey credential
func (r *PasskeyCredentialRepository) Create(ctx context.Context, credential *user.PasskeyCredential) error {
	model, err := r.mapper.ToModel(credential)
	if err != nil {
		r.logger.Errorw("failed to map passkey credential entity to model", "error", err)
		return fmt.Errorf("failed to map passkey credential entity: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.Errorw("failed to create passkey credential in database", "error", err)
		return fmt.Errorf("failed to create passkey credential: %w", err)
	}

	if err := credential.SetID(model.ID); err != nil {
		r.logger.Errorw("failed to set passkey credential ID", "error", err)
		return fmt.Errorf("failed to set passkey credential ID: %w", err)
	}

	r.logger.Infow("passkey credential created successfully", "id", model.ID, "user_id", model.UserID)
	return nil
}

// GetByID retrieves a passkey credential by internal ID
func (r *PasskeyCredentialRepository) GetByID(ctx context.Context, id uint) (*user.PasskeyCredential, error) {
	var model models.PasskeyCredentialModel

	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get passkey credential by ID", "id", id, "error", err)
		return nil, fmt.Errorf("failed to get passkey credential: %w", err)
	}

	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		r.logger.Errorw("failed to map passkey credential model to entity", "id", id, "error", err)
		return nil, fmt.Errorf("failed to map passkey credential: %w", err)
	}

	return entity, nil
}

// GetBySID retrieves a passkey credential by external SID (pk_xxx)
func (r *PasskeyCredentialRepository) GetBySID(ctx context.Context, sid string) (*user.PasskeyCredential, error) {
	var model models.PasskeyCredentialModel

	if err := r.db.WithContext(ctx).Where("sid = ?", sid).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get passkey credential by SID", "sid", sid, "error", err)
		return nil, fmt.Errorf("failed to get passkey credential: %w", err)
	}

	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		r.logger.Errorw("failed to map passkey credential model to entity", "sid", sid, "error", err)
		return nil, fmt.Errorf("failed to map passkey credential: %w", err)
	}

	return entity, nil
}

// GetByCredentialID retrieves a passkey credential by WebAuthn credential ID
func (r *PasskeyCredentialRepository) GetByCredentialID(ctx context.Context, credentialID []byte) (*user.PasskeyCredential, error) {
	var model models.PasskeyCredentialModel

	if err := r.db.WithContext(ctx).Where("credential_id = ?", credentialID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get passkey credential by credential ID", "error", err)
		return nil, fmt.Errorf("failed to get passkey credential: %w", err)
	}

	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		r.logger.Errorw("failed to map passkey credential model to entity", "error", err)
		return nil, fmt.Errorf("failed to map passkey credential: %w", err)
	}

	return entity, nil
}

// GetByUserID retrieves all passkey credentials for a user
func (r *PasskeyCredentialRepository) GetByUserID(ctx context.Context, userID uint) ([]*user.PasskeyCredential, error) {
	var credentialModels []*models.PasskeyCredentialModel

	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&credentialModels).Error; err != nil {
		r.logger.Errorw("failed to get passkey credentials by user ID", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to get passkey credentials: %w", err)
	}

	credentials, err := r.mapper.ToEntities(credentialModels)
	if err != nil {
		r.logger.Errorw("failed to map passkey credential models to entities", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to map passkey credentials: %w", err)
	}

	return credentials, nil
}

// Update updates an existing passkey credential
func (r *PasskeyCredentialRepository) Update(ctx context.Context, credential *user.PasskeyCredential) error {
	model, err := r.mapper.ToModel(credential)
	if err != nil {
		r.logger.Errorw("failed to map passkey credential entity to model", "error", err)
		return fmt.Errorf("failed to map passkey credential entity: %w", err)
	}

	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		r.logger.Errorw("failed to update passkey credential in database", "id", model.ID, "error", err)
		return fmt.Errorf("failed to update passkey credential: %w", err)
	}

	r.logger.Infow("passkey credential updated successfully", "id", model.ID)
	return nil
}

// Delete deletes a passkey credential by internal ID
func (r *PasskeyCredentialRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.PasskeyCredentialModel{}, id)
	if result.Error != nil {
		r.logger.Errorw("failed to delete passkey credential", "id", id, "error", result.Error)
		return fmt.Errorf("failed to delete passkey credential: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		r.logger.Warnw("passkey credential not found for deletion", "id", id)
		return nil
	}

	r.logger.Infow("passkey credential deleted successfully", "id", id)
	return nil
}

// DeleteBySID deletes a passkey credential by external SID
func (r *PasskeyCredentialRepository) DeleteBySID(ctx context.Context, sid string) error {
	result := r.db.WithContext(ctx).Where("sid = ?", sid).Delete(&models.PasskeyCredentialModel{})
	if result.Error != nil {
		r.logger.Errorw("failed to delete passkey credential by SID", "sid", sid, "error", result.Error)
		return fmt.Errorf("failed to delete passkey credential: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		r.logger.Warnw("passkey credential not found for deletion by SID", "sid", sid)
		return nil
	}

	r.logger.Infow("passkey credential deleted successfully by SID", "sid", sid)
	return nil
}

// CountByUserID returns the count of passkey credentials for a user
func (r *PasskeyCredentialRepository) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	var count int64

	if err := r.db.WithContext(ctx).Model(&models.PasskeyCredentialModel{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		r.logger.Errorw("failed to count passkey credentials by user ID", "user_id", userID, "error", err)
		return 0, fmt.Errorf("failed to count passkey credentials: %w", err)
	}

	return count, nil
}

// ExistsByCredentialID checks if a credential with the given WebAuthn credential ID exists
func (r *PasskeyCredentialRepository) ExistsByCredentialID(ctx context.Context, credentialID []byte) (bool, error) {
	var count int64

	if err := r.db.WithContext(ctx).Model(&models.PasskeyCredentialModel{}).Where("credential_id = ?", credentialID).Count(&count).Error; err != nil {
		r.logger.Errorw("failed to check if passkey credential exists by credential ID", "error", err)
		return false, fmt.Errorf("failed to check passkey credential existence: %w", err)
	}

	return count > 0, nil
}

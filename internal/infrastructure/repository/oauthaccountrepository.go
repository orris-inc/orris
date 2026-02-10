package repository

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/errors"
)

// OAuthAccountRepository implements the user.OAuthAccountRepository interface
// using GORM with Model/Mapper separation.
type OAuthAccountRepository struct {
	db     *gorm.DB
	mapper mappers.OAuthAccountMapper
}

// NewOAuthAccountRepository creates a new OAuthAccountRepository.
func NewOAuthAccountRepository(db *gorm.DB) user.OAuthAccountRepository {
	return &OAuthAccountRepository{
		db:     db,
		mapper: mappers.NewOAuthAccountMapper(),
	}
}

func (r *OAuthAccountRepository) Create(account *user.OAuthAccount) error {
	model := r.mapper.ToModel(account)
	if err := r.db.Create(model).Error; err != nil {
		return fmt.Errorf("failed to create oauth account: %w", err)
	}
	// Sync auto-generated ID back to the domain entity
	account.ID = model.ID
	return nil
}

func (r *OAuthAccountRepository) GetByID(id uint) (*user.OAuthAccount, error) {
	var model models.OAuthAccountModel
	err := r.db.First(&model, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("oauth account not found")
		}
		return nil, fmt.Errorf("failed to get oauth account by ID: %w", err)
	}
	return r.mapper.ToDomain(&model), nil
}

func (r *OAuthAccountRepository) GetByProviderAndUserID(provider, providerUserID string) (*user.OAuthAccount, error) {
	var model models.OAuthAccountModel
	err := r.db.Where("provider = ? AND provider_user_id = ?", provider, providerUserID).
		First(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get oauth account: %w", err)
	}
	return r.mapper.ToDomain(&model), nil
}

func (r *OAuthAccountRepository) GetByUserID(userID uint) ([]*user.OAuthAccount, error) {
	var accountModels []*models.OAuthAccountModel
	err := r.db.Where("user_id = ?", userID).Find(&accountModels).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get oauth accounts by user ID: %w", err)
	}
	return r.mapper.ToDomainList(accountModels), nil
}

func (r *OAuthAccountRepository) Update(account *user.OAuthAccount) error {
	model := r.mapper.ToModel(account)
	result := r.db.Save(model)
	if result.Error != nil {
		return fmt.Errorf("failed to update oauth account: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("oauth account not found")
	}
	return nil
}

func (r *OAuthAccountRepository) Delete(id uint) error {
	result := r.db.Delete(&models.OAuthAccountModel{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete oauth account: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("oauth account not found")
	}
	return nil
}

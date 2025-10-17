package repository

import (
	"fmt"

	"gorm.io/gorm"

	"orris/internal/domain/user"
	"orris/internal/shared/errors"
)

type OAuthAccountRepository struct {
	db *gorm.DB
}

func NewOAuthAccountRepository(db *gorm.DB) user.OAuthAccountRepository {
	return &OAuthAccountRepository{db: db}
}

func (r *OAuthAccountRepository) Create(account *user.OAuthAccount) error {
	if err := r.db.Create(account).Error; err != nil {
		return fmt.Errorf("failed to create oauth account: %w", err)
	}
	return nil
}

func (r *OAuthAccountRepository) GetByID(id uint) (*user.OAuthAccount, error) {
	var account user.OAuthAccount
	err := r.db.First(&account, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("oauth account not found")
		}
		return nil, fmt.Errorf("failed to get oauth account by ID: %w", err)
	}
	return &account, nil
}

func (r *OAuthAccountRepository) GetByProviderAndUserID(provider, providerUserID string) (*user.OAuthAccount, error) {
	var account user.OAuthAccount
	err := r.db.Where("provider = ? AND provider_user_id = ?", provider, providerUserID).
		First(&account).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get oauth account: %w", err)
	}
	return &account, nil
}

func (r *OAuthAccountRepository) GetByUserID(userID uint) ([]*user.OAuthAccount, error) {
	var accounts []*user.OAuthAccount
	err := r.db.Where("user_id = ?", userID).Find(&accounts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get oauth accounts by user ID: %w", err)
	}
	return accounts, nil
}

func (r *OAuthAccountRepository) Update(account *user.OAuthAccount) error {
	result := r.db.Save(account)
	if result.Error != nil {
		return fmt.Errorf("failed to update oauth account: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("oauth account not found")
	}
	return nil
}

func (r *OAuthAccountRepository) Delete(id uint) error {
	result := r.db.Delete(&user.OAuthAccount{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete oauth account: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("oauth account not found")
	}
	return nil
}

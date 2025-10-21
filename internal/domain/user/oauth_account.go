package user

import (
	"fmt"
	"time"
)

type OAuthAccount struct {
	ID                uint
	UserID            uint
	Provider          string
	ProviderUserID    string
	ProviderEmail     string
	ProviderUsername  string
	ProviderAvatarURL string
	RawUserInfo       *string
	LastLoginAt       *time.Time
	LoginCount        uint
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (OAuthAccount) TableName() string {
	return "oauth_accounts"
}

func NewOAuthAccount(userID uint, provider, providerUserID, providerEmail string) (*OAuthAccount, error) {
	if userID == 0 {
		return nil, fmt.Errorf("user ID is required")
	}
	if provider == "" {
		return nil, fmt.Errorf("provider is required")
	}
	if providerUserID == "" {
		return nil, fmt.Errorf("provider user ID is required")
	}

	now := time.Now()
	return &OAuthAccount{
		UserID:         userID,
		Provider:       provider,
		ProviderUserID: providerUserID,
		ProviderEmail:  providerEmail,
		LoginCount:     1,
		LastLoginAt:    &now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (o *OAuthAccount) RecordLogin() {
	o.LoginCount++
	now := time.Now()
	o.LastLoginAt = &now
	o.UpdatedAt = now
}

type OAuthAccountRepository interface {
	Create(account *OAuthAccount) error
	GetByID(id uint) (*OAuthAccount, error)
	GetByProviderAndUserID(provider, providerUserID string) (*OAuthAccount, error)
	GetByUserID(userID uint) ([]*OAuthAccount, error)
	Update(account *OAuthAccount) error
	Delete(id uint) error
}

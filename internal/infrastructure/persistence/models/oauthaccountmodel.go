package models

import "time"

// OAuthAccountModel represents the database persistence model for OAuth accounts.
type OAuthAccountModel struct {
	ID                uint       `gorm:"primarykey"`
	UserID            uint       `gorm:"not null;index:idx_oauth_user_id"`
	Provider          string     `gorm:"not null;size:50;uniqueIndex:idx_provider_user"`
	ProviderUserID    string     `gorm:"not null;size:255;uniqueIndex:idx_provider_user;column:provider_user_id"`
	ProviderEmail     string     `gorm:"size:255"`
	ProviderUsername  string     `gorm:"size:100"`
	ProviderAvatarURL string     `gorm:"size:500;column:provider_avatar_url"`
	RawUserInfo       *string    `gorm:"type:text"`
	LastLoginAt       *time.Time
	LoginCount        uint       `gorm:"default:0"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// TableName specifies the table name for GORM
func (OAuthAccountModel) TableName() string {
	return "oauth_accounts"
}

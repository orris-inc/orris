package models

import (
	"time"

	"gorm.io/gorm"

	"orris/internal/shared/constants"
)

// UserModel represents the database persistence model for users
// This is the anti-corruption layer between domain and database
type UserModel struct {
	ID                         uint    `gorm:"primarykey"`
	Email                      string  `gorm:"uniqueIndex;not null;size:255"`
	Name                       string  `gorm:"not null;size:100"`
	AvatarURL                  *string `gorm:"size:500"`
	EmailVerified              bool    `gorm:"default:false;index:idx_email_verified"`
	Locale                     string  `gorm:"size:10;default:en"`
	Status                     string  `gorm:"not null;default:pending;size:20"`
	Version                    int     `gorm:"not null;default:1"`
	PasswordHash               *string `gorm:"size:255"`
	EmailVerificationToken     *string `gorm:"size:255;index:idx_email_verification_token"`
	EmailVerificationExpiresAt *time.Time
	PasswordResetToken         *string `gorm:"size:255;index:idx_password_reset_token"`
	PasswordResetExpiresAt     *time.Time
	LastPasswordChangeAt       *time.Time
	FailedLoginAttempts        int `gorm:"default:0"`
	LockedUntil                *time.Time
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
	DeletedAt                  gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name for GORM
func (UserModel) TableName() string {
	return constants.TableUsers
}

// BeforeCreate hook for GORM
func (u *UserModel) BeforeCreate(tx *gorm.DB) error {
	if u.Status == "" {
		u.Status = "pending"
	}
	if u.Version == 0 {
		u.Version = 1
	}
	return nil
}

// BeforeUpdate hook for GORM
func (u *UserModel) BeforeUpdate(tx *gorm.DB) error {
	// Increment version for optimistic locking
	tx.Statement.SetColumn("version", u.Version+1)
	return nil
}

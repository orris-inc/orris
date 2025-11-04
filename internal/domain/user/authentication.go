package user

import (
	"fmt"
	"time"

	vo "orris/internal/domain/user/value_objects"
)

type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, hash string) error
}

func (u *User) SetPassword(password *vo.Password, hasher PasswordHasher) error {
	if password == nil {
		return fmt.Errorf("password cannot be nil")
	}

	hash, err := hasher.Hash(password.String())
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	u.passwordHash = &hash
	u.lastPasswordChangeAt = timePtr(time.Now())
	u.updatedAt = time.Now()
	u.version++

	return nil
}

func (u *User) VerifyPassword(plainPassword string, hasher PasswordHasher) error {
	if u.passwordHash == nil || *u.passwordHash == "" {
		return fmt.Errorf("user has no password set")
	}

	if err := hasher.Verify(plainPassword, *u.passwordHash); err != nil {
		u.recordFailedLogin()
		return fmt.Errorf("invalid password")
	}

	u.resetFailedLoginAttempts()
	return nil
}

func (u *User) GenerateEmailVerificationToken() (*vo.Token, error) {
	token, err := vo.GenerateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate verification token: %w", err)
	}

	u.emailVerificationToken = stringPtr(token.Hash())
	u.emailVerificationExpiresAt = timePtr(time.Now().Add(24 * time.Hour))
	u.updatedAt = time.Now()

	return token, nil
}

func (u *User) VerifyEmail(plainToken string) error {
	if u.emailVerified {
		return fmt.Errorf("email is already verified")
	}

	if u.emailVerificationToken == nil || *u.emailVerificationToken == "" {
		return fmt.Errorf("no verification token found")
	}

	if u.emailVerificationExpiresAt == nil || time.Now().After(*u.emailVerificationExpiresAt) {
		return fmt.Errorf("verification token has expired")
	}

	token, err := vo.NewTokenFromValue(plainToken)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	if !token.Verify(plainToken) || token.Hash() != *u.emailVerificationToken {
		return fmt.Errorf("invalid verification token")
	}

	u.emailVerified = true
	u.emailVerificationToken = nil
	u.emailVerificationExpiresAt = nil
	u.updatedAt = time.Now()
	u.version++

	return nil
}

func (u *User) GeneratePasswordResetToken() (*vo.Token, error) {
	token, err := vo.GenerateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate reset token: %w", err)
	}

	u.passwordResetToken = stringPtr(token.Hash())
	u.passwordResetExpiresAt = timePtr(time.Now().Add(30 * time.Minute))
	u.updatedAt = time.Now()

	return token, nil
}

func (u *User) ResetPassword(plainToken string, newPassword *vo.Password, hasher PasswordHasher) error {
	if u.passwordResetToken == nil || *u.passwordResetToken == "" {
		return fmt.Errorf("no password reset token found")
	}

	if u.passwordResetExpiresAt == nil || time.Now().After(*u.passwordResetExpiresAt) {
		return fmt.Errorf("password reset token has expired")
	}

	token, err := vo.NewTokenFromValue(plainToken)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	if !token.Verify(plainToken) || token.Hash() != *u.passwordResetToken {
		return fmt.Errorf("invalid reset token")
	}

	if err := u.SetPassword(newPassword, hasher); err != nil {
		return fmt.Errorf("failed to set new password: %w", err)
	}

	u.passwordResetToken = nil
	u.passwordResetExpiresAt = nil
	u.failedLoginAttempts = 0
	u.lockedUntil = nil

	return nil
}

func (u *User) RecordFailedLogin() {
	u.recordFailedLogin()
}

func (u *User) recordFailedLogin() {
	u.failedLoginAttempts++
	u.updatedAt = time.Now()

	const maxAttempts = 5
	if u.failedLoginAttempts >= maxAttempts {
		lockDuration := 30 * time.Minute
		u.lockedUntil = timePtr(time.Now().Add(lockDuration))
	}
}

func (u *User) resetFailedLoginAttempts() {
	if u.failedLoginAttempts > 0 {
		u.failedLoginAttempts = 0
		u.lockedUntil = nil
		u.updatedAt = time.Now()
	}
}

func (u *User) IsLocked() bool {
	if u.lockedUntil == nil {
		return false
	}
	return time.Now().Before(*u.lockedUntil)
}

func (u *User) HasPassword() bool {
	return u.passwordHash != nil && *u.passwordHash != ""
}

func (u *User) IsEmailVerified() bool {
	return u.emailVerified
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func stringPtr(s string) *string {
	return &s
}

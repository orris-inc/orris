package user

import (
	"fmt"
	"time"

	"orris/internal/shared/authorization"
	vo "orris/internal/domain/user/value_objects"
)

// User represents the user aggregate root (pure domain model without persistence concerns)
type User struct {
	id                         uint
	email                      *vo.Email
	name                       *vo.Name
	role                       authorization.UserRole
	status                     vo.Status
	createdAt                  time.Time
	updatedAt                  time.Time
	version                    int
	passwordHash               *string
	emailVerified              bool
	emailVerificationToken     *string
	emailVerificationExpiresAt *time.Time
	passwordResetToken         *string
	passwordResetExpiresAt     *time.Time
	lastPasswordChangeAt       *time.Time
	failedLoginAttempts        int
	lockedUntil                *time.Time
}

// NewUser creates a new user aggregate with initial values
func NewUser(email *vo.Email, name *vo.Name) (*User, error) {
	if email == nil {
		return nil, fmt.Errorf("email is required")
	}
	if name == nil {
		return nil, fmt.Errorf("name is required")
	}

	now := time.Now()
	user := &User{
		email:     email,
		name:      name,
		role:      authorization.RoleUser,
		status:    vo.StatusPending,
		createdAt: now,
		updatedAt: now,
		version:   1,
	}

	return user, nil
}

// ReconstructUser reconstructs a user from persistence
func ReconstructUser(id uint, email *vo.Email, name *vo.Name, role authorization.UserRole, status vo.Status, createdAt, updatedAt time.Time, version int) (*User, error) {
	if id == 0 {
		return nil, fmt.Errorf("user ID cannot be zero")
	}
	if email == nil {
		return nil, fmt.Errorf("email is required")
	}
	if name == nil {
		return nil, fmt.Errorf("name is required")
	}

	return &User{
		id:        id,
		email:     email,
		name:      name,
		role:      role,
		status:    status,
		createdAt: createdAt,
		updatedAt: updatedAt,
		version:   version,
	}, nil
}

type UserAuthData struct {
	PasswordHash               *string
	EmailVerified              bool
	EmailVerificationToken     *string
	EmailVerificationExpiresAt *time.Time
	PasswordResetToken         *string
	PasswordResetExpiresAt     *time.Time
	LastPasswordChangeAt       *time.Time
	FailedLoginAttempts        int
	LockedUntil                *time.Time
}

func ReconstructUserWithAuth(id uint, email *vo.Email, name *vo.Name, role authorization.UserRole, status vo.Status, createdAt, updatedAt time.Time, version int, authData *UserAuthData) (*User, error) {
	u, err := ReconstructUser(id, email, name, role, status, createdAt, updatedAt, version)
	if err != nil {
		return nil, err
	}

	if authData != nil {
		u.passwordHash = authData.PasswordHash
		u.emailVerified = authData.EmailVerified
		u.emailVerificationToken = authData.EmailVerificationToken
		u.emailVerificationExpiresAt = authData.EmailVerificationExpiresAt
		u.passwordResetToken = authData.PasswordResetToken
		u.passwordResetExpiresAt = authData.PasswordResetExpiresAt
		u.lastPasswordChangeAt = authData.LastPasswordChangeAt
		u.failedLoginAttempts = authData.FailedLoginAttempts
		u.lockedUntil = authData.LockedUntil
	}

	return u, nil
}

func (u *User) GetAuthData() *UserAuthData {
	return &UserAuthData{
		PasswordHash:               u.passwordHash,
		EmailVerified:              u.emailVerified,
		EmailVerificationToken:     u.emailVerificationToken,
		EmailVerificationExpiresAt: u.emailVerificationExpiresAt,
		PasswordResetToken:         u.passwordResetToken,
		PasswordResetExpiresAt:     u.passwordResetExpiresAt,
		LastPasswordChangeAt:       u.lastPasswordChangeAt,
		FailedLoginAttempts:        u.failedLoginAttempts,
		LockedUntil:                u.lockedUntil,
	}
}

// ID returns the user ID
func (u *User) ID() uint {
	return u.id
}

// Email returns the user's email
func (u *User) Email() *vo.Email {
	return u.email
}

// Name returns the user's name
func (u *User) Name() *vo.Name {
	return u.name
}

// Role returns the user's role
func (u *User) Role() authorization.UserRole {
	return u.role
}

// IsAdmin returns true if the user has admin role
func (u *User) IsAdmin() bool {
	return u.role.IsAdmin()
}

// SetRole sets the user's role
func (u *User) SetRole(role authorization.UserRole) {
	u.role = role
	u.updatedAt = time.Now()
}

// Status returns the user's status
func (u *User) Status() vo.Status {
	return u.status
}

// CreatedAt returns when the user was created
func (u *User) CreatedAt() time.Time {
	return u.createdAt
}

// UpdatedAt returns when the user was last updated
func (u *User) UpdatedAt() time.Time {
	return u.updatedAt
}

// Version returns the aggregate version for optimistic locking
func (u *User) Version() int {
	return u.version
}

// SetID sets the user ID (only for persistence layer use)
func (u *User) SetID(id uint) error {
	if u.id != 0 {
		return fmt.Errorf("user ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("user ID cannot be zero")
	}
	u.id = id
	return nil
}

// UpdateEmail updates the user's email address
func (u *User) UpdateEmail(newEmail *vo.Email) error {
	if newEmail == nil {
		return fmt.Errorf("email cannot be nil")
	}

	if u.email.Equals(newEmail) {
		return nil // No change needed
	}

	u.email = newEmail
	u.updatedAt = time.Now()
	u.version++

	return nil
}

// UpdateName updates the user's name
func (u *User) UpdateName(newName *vo.Name) error {
	if newName == nil {
		return fmt.Errorf("name cannot be nil")
	}

	if u.name.Equals(newName) {
		return nil // No change needed
	}

	u.name = newName
	u.updatedAt = time.Now()
	u.version++

	return nil
}

// Activate activates a pending or inactive user
func (u *User) Activate() error {
	if u.status.IsActive() {
		return nil // Already active
	}

	if !u.status.CanTransitionTo(vo.StatusActive) {
		return fmt.Errorf("cannot activate user with status %s", u.status.String())
	}

	u.status = vo.StatusActive
	u.updatedAt = time.Now()
	u.version++

	return nil
}

// Deactivate deactivates an active user
func (u *User) Deactivate(reason string) error {
	if u.status.IsInactive() {
		return nil // Already inactive
	}

	if !u.status.CanTransitionTo(vo.StatusInactive) {
		return fmt.Errorf("cannot deactivate user with status %s", u.status.String())
	}

	u.status = vo.StatusInactive
	u.updatedAt = time.Now()
	u.version++

	if reason == "" {
		reason = "User deactivated"
	}

	return nil
}

// Suspend suspends a user (typically for policy violations)
func (u *User) Suspend(reason string) error {
	if u.status.IsSuspended() {
		return nil // Already suspended
	}

	if !u.status.CanTransitionTo(vo.StatusSuspended) {
		return fmt.Errorf("cannot suspend user with status %s", u.status.String())
	}

	if reason == "" {
		return fmt.Errorf("suspension reason is required")
	}

	u.status = vo.StatusSuspended
	u.updatedAt = time.Now()
	u.version++

	return nil
}

// Delete marks the user as deleted (soft delete)
func (u *User) Delete() error {
	if u.status.IsDeleted() {
		return nil // Already deleted
	}

	if !u.status.CanTransitionTo(vo.StatusDeleted) {
		return fmt.Errorf("cannot delete user with status %s", u.status.String())
	}

	u.status = vo.StatusDeleted
	u.updatedAt = time.Now()
	u.version++

	return nil
}

// CanPerformActions checks if the user can perform actions
func (u *User) CanPerformActions() bool {
	return u.status.CanPerformActions()
}

// RequiresVerification checks if the user requires verification
func (u *User) RequiresVerification() bool {
	return u.status.RequiresVerification()
}

// IsBusinessEmail checks if the user has a business email
func (u *User) IsBusinessEmail() bool {
	return u.email.IsBusinessEmail()
}

// GetDisplayInfo returns formatted display information
func (u *User) GetDisplayInfo() UserDisplayInfo {
	return UserDisplayInfo{
		ID:          u.id,
		Email:       u.email.String(),
		DisplayName: u.name.DisplayName(),
		Initials:    u.name.Initials(),
		Status:      u.status.String(),
		CreatedAt:   u.createdAt,
	}
}

// UserDisplayInfo represents user information for display purposes
type UserDisplayInfo struct {
	ID          uint      `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Initials    string    `json:"initials"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// ChangePassword changes the user's password after verifying the old password
// It updates the password hash and records the change timestamp
func (u *User) ChangePassword(oldPassword, newPassword *vo.Password, hasher PasswordHasher) error {
	if u.passwordHash == nil {
		return fmt.Errorf("user has no password set")
	}

	// Verify old password
	if err := hasher.Verify(oldPassword.String(), *u.passwordHash); err != nil {
		return fmt.Errorf("old password is incorrect")
	}

	// Hash new password
	newHash, err := hasher.Hash(newPassword.String())
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update password
	u.passwordHash = &newHash
	now := time.Now()
	u.lastPasswordChangeAt = &now
	u.updatedAt = now
	u.version++

	return nil
}

// Validate performs domain-level validation
func (u *User) Validate() error {
	if u.email == nil {
		return fmt.Errorf("email is required")
	}
	if u.name == nil {
		return fmt.Errorf("name is required")
	}
	if !vo.ValidStatuses[u.status] {
		return fmt.Errorf("invalid status: %s", u.status)
	}
	return nil
}

package user

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	vo "github.com/orris-inc/orris/internal/domain/user/valueobjects"
	"github.com/orris-inc/orris/internal/shared/authorization"
)

// =============================================================================
// Test helpers
// =============================================================================

// mockShortIDGenerator generates a predictable short ID for testing.
func mockShortIDGenerator() func() (string, error) {
	counter := 0
	return func() (string, error) {
		counter++
		return fmt.Sprintf("usr_test_%d", counter), nil
	}
}

// failingShortIDGenerator returns a generator that always fails.
func failingShortIDGenerator() func() (string, error) {
	return func() (string, error) {
		return "", fmt.Errorf("short ID generation failed")
	}
}

// validEmail creates a valid Email value object for testing.
func validEmail(t *testing.T) *vo.Email {
	t.Helper()
	email, err := vo.NewEmail("test@example.com")
	require.NoError(t, err)
	return email
}

// validEmailWithAddr creates a valid Email value object with a specific address.
func validEmailWithAddr(t *testing.T, addr string) *vo.Email {
	t.Helper()
	email, err := vo.NewEmail(addr)
	require.NoError(t, err)
	return email
}

// validName creates a valid Name value object for testing.
func validName(t *testing.T) *vo.Name {
	t.Helper()
	name, err := vo.NewName("John Doe")
	require.NoError(t, err)
	return name
}

// validNameWithValue creates a valid Name value object with a specific value.
func validNameWithValue(t *testing.T, value string) *vo.Name {
	t.Helper()
	name, err := vo.NewName(value)
	require.NoError(t, err)
	return name
}

// validPassword creates a valid Password value object for testing.
func validPassword(t *testing.T, pw string) *vo.Password {
	t.Helper()
	password, err := vo.NewPassword(pw)
	require.NoError(t, err)
	return password
}

// newTestUser creates a new user with valid defaults for testing.
func newTestUser(t *testing.T) *User {
	t.Helper()
	gen := mockShortIDGenerator()
	u, err := NewUser(validEmail(t), validName(t), gen)
	require.NoError(t, err)
	require.NotNil(t, u)
	return u
}

// mockPasswordHasher is a simple password hasher for testing.
type mockPasswordHasher struct {
	hashPrefix string
}

func (h *mockPasswordHasher) Hash(password string) (string, error) {
	return h.hashPrefix + ":" + password, nil
}

func (h *mockPasswordHasher) Verify(password, hash string) error {
	expected := h.hashPrefix + ":" + password
	if expected != hash {
		return fmt.Errorf("password mismatch")
	}
	return nil
}

// failingPasswordHasher always fails hashing.
type failingPasswordHasher struct{}

func (h *failingPasswordHasher) Hash(_ string) (string, error) {
	return "", fmt.Errorf("hash failure")
}

func (h *failingPasswordHasher) Verify(_, _ string) error {
	return fmt.Errorf("verify failure")
}

// reconstructActiveUser reconstructs a user from persistence with active status.
func reconstructActiveUser(t *testing.T) *User {
	t.Helper()
	u, err := ReconstructUser(
		1, "usr_abc123",
		validEmail(t), validName(t),
		authorization.RoleUser, vo.StatusActive,
		time.Now().Add(-24*time.Hour), time.Now(),
		1,
	)
	require.NoError(t, err)
	return u
}

// reconstructUserWithStatus reconstructs a user with a specific status.
func reconstructUserWithStatus(t *testing.T, status vo.Status) *User {
	t.Helper()
	u, err := ReconstructUser(
		1, "usr_abc123",
		validEmail(t), validName(t),
		authorization.RoleUser, status,
		time.Now().Add(-24*time.Hour), time.Now(),
		1,
	)
	require.NoError(t, err)
	return u
}

// reconstructUserWithPassword reconstructs a user with a password set.
func reconstructUserWithPassword(t *testing.T, hasher *mockPasswordHasher) *User {
	t.Helper()
	hash, err := hasher.Hash("OldPass123")
	require.NoError(t, err)
	u, err := ReconstructUserWithAuth(
		1, "usr_abc123",
		validEmail(t), validName(t),
		authorization.RoleUser, vo.StatusActive,
		time.Now().Add(-24*time.Hour), time.Now(),
		1,
		&UserAuthData{
			PasswordHash: &hash,
		},
	)
	require.NoError(t, err)
	return u
}

// =============================================================================
// NewUser - Constructor Tests
// =============================================================================

// TestNewUser_ValidInput verifies that NewUser creates a user with valid parameters.
func TestNewUser_ValidInput(t *testing.T) {
	email := validEmail(t)
	name := validName(t)
	gen := mockShortIDGenerator()

	u, err := NewUser(email, name, gen)

	require.NoError(t, err)
	require.NotNil(t, u)
	assert.Equal(t, "usr_test_1", u.SID())
	assert.Equal(t, email, u.Email())
	assert.Equal(t, name, u.Name())
	assert.Equal(t, authorization.RoleUser, u.Role())
	assert.Equal(t, vo.StatusActive, u.Status())
	assert.Equal(t, 1, u.Version())
	assert.False(t, u.IsAdmin())
	assert.True(t, u.CanPerformActions())
	assert.Zero(t, u.ID(), "new user should have zero ID before persistence")
}

// TestNewUser_InvalidEmail verifies that NewUser rejects a nil email.
func TestNewUser_InvalidEmail(t *testing.T) {
	gen := mockShortIDGenerator()

	u, err := NewUser(nil, validName(t), gen)

	assert.Error(t, err)
	assert.Nil(t, u)
	assert.Contains(t, err.Error(), "email is required")
}

// TestNewUser_EmptyName verifies that NewUser rejects a nil name.
func TestNewUser_EmptyName(t *testing.T) {
	gen := mockShortIDGenerator()

	u, err := NewUser(validEmail(t), nil, gen)

	assert.Error(t, err)
	assert.Nil(t, u)
	assert.Contains(t, err.Error(), "name is required")
}

// TestNewUser_ShortIDGeneratorFailure verifies that NewUser fails when the ID generator fails.
func TestNewUser_ShortIDGeneratorFailure(t *testing.T) {
	gen := failingShortIDGenerator()

	u, err := NewUser(validEmail(t), validName(t), gen)

	assert.Error(t, err)
	assert.Nil(t, u)
	assert.Contains(t, err.Error(), "failed to generate short ID")
}

// TestNewUser_BothNilEmailAndName verifies that NewUser fails when both email and name are nil.
func TestNewUser_BothNilEmailAndName(t *testing.T) {
	gen := mockShortIDGenerator()

	u, err := NewUser(nil, nil, gen)

	assert.Error(t, err)
	assert.Nil(t, u)
	// Should fail on email check first
	assert.Contains(t, err.Error(), "email is required")
}

// TestNewUser_MultipleCreations verifies that each user gets a unique SID.
func TestNewUser_MultipleCreations(t *testing.T) {
	gen := mockShortIDGenerator()

	u1, err := NewUser(validEmail(t), validName(t), gen)
	require.NoError(t, err)

	u2, err := NewUser(validEmailWithAddr(t, "other@example.com"), validName(t), gen)
	require.NoError(t, err)

	assert.NotEqual(t, u1.SID(), u2.SID(), "each user should have a unique SID")
}

// =============================================================================
// ReconstructUser Tests
// =============================================================================

// TestReconstructUser_Valid verifies successful reconstruction from persistence.
func TestReconstructUser_Valid(t *testing.T) {
	email := validEmail(t)
	name := validName(t)
	now := time.Now()
	created := now.Add(-24 * time.Hour)

	u, err := ReconstructUser(
		42, "usr_xyz789",
		email, name,
		authorization.RoleAdmin, vo.StatusActive,
		created, now,
		5,
	)

	require.NoError(t, err)
	require.NotNil(t, u)
	assert.Equal(t, uint(42), u.ID())
	assert.Equal(t, "usr_xyz789", u.SID())
	assert.Equal(t, email, u.Email())
	assert.Equal(t, name, u.Name())
	assert.Equal(t, authorization.RoleAdmin, u.Role())
	assert.Equal(t, vo.StatusActive, u.Status())
	assert.Equal(t, created, u.CreatedAt())
	assert.Equal(t, now, u.UpdatedAt())
	assert.Equal(t, 5, u.Version())
	assert.True(t, u.IsAdmin())
}

// TestReconstructUser_ZeroID verifies that reconstruction fails with zero ID.
func TestReconstructUser_ZeroID(t *testing.T) {
	u, err := ReconstructUser(
		0, "usr_abc",
		validEmail(t), validName(t),
		authorization.RoleUser, vo.StatusActive,
		time.Now(), time.Now(),
		1,
	)

	assert.Error(t, err)
	assert.Nil(t, u)
	assert.Contains(t, err.Error(), "user ID cannot be zero")
}

// TestReconstructUser_EmptySID verifies that reconstruction fails with empty SID.
func TestReconstructUser_EmptySID(t *testing.T) {
	u, err := ReconstructUser(
		1, "",
		validEmail(t), validName(t),
		authorization.RoleUser, vo.StatusActive,
		time.Now(), time.Now(),
		1,
	)

	assert.Error(t, err)
	assert.Nil(t, u)
	assert.Contains(t, err.Error(), "user SID is required")
}

// TestReconstructUser_NilEmail verifies that reconstruction fails with nil email.
func TestReconstructUser_NilEmail(t *testing.T) {
	u, err := ReconstructUser(
		1, "usr_abc",
		nil, validName(t),
		authorization.RoleUser, vo.StatusActive,
		time.Now(), time.Now(),
		1,
	)

	assert.Error(t, err)
	assert.Nil(t, u)
	assert.Contains(t, err.Error(), "email is required")
}

// TestReconstructUser_NilName verifies that reconstruction fails with nil name.
func TestReconstructUser_NilName(t *testing.T) {
	u, err := ReconstructUser(
		1, "usr_abc",
		validEmail(t), nil,
		authorization.RoleUser, vo.StatusActive,
		time.Now(), time.Now(),
		1,
	)

	assert.Error(t, err)
	assert.Nil(t, u)
	assert.Contains(t, err.Error(), "name is required")
}

// =============================================================================
// ReconstructUserWithAuth Tests
// =============================================================================

// TestReconstructUserWithAuth_ValidWithAuthData verifies reconstruction with auth data.
func TestReconstructUserWithAuth_ValidWithAuthData(t *testing.T) {
	hash := "hashed_password"
	lastPwChange := time.Now().Add(-1 * time.Hour)
	lockedUntil := time.Now().Add(15 * time.Minute)
	readAt := time.Now().Add(-30 * time.Minute)

	u, err := ReconstructUserWithAuth(
		1, "usr_abc",
		validEmail(t), validName(t),
		authorization.RoleUser, vo.StatusActive,
		time.Now(), time.Now(),
		1,
		&UserAuthData{
			PasswordHash:           &hash,
			EmailVerified:          true,
			FailedLoginAttempts:    3,
			LastPasswordChangeAt:   &lastPwChange,
			LockedUntil:            &lockedUntil,
			AnnouncementsReadAt:    &readAt,
		},
	)

	require.NoError(t, err)
	require.NotNil(t, u)
	assert.True(t, u.HasPassword())
	assert.True(t, u.IsEmailVerified())
	authData := u.GetAuthData()
	assert.Equal(t, 3, authData.FailedLoginAttempts)
	assert.NotNil(t, authData.LockedUntil)
	assert.NotNil(t, authData.AnnouncementsReadAt)
}

// TestReconstructUserWithAuth_NilAuthData verifies reconstruction with nil auth data.
func TestReconstructUserWithAuth_NilAuthData(t *testing.T) {
	u, err := ReconstructUserWithAuth(
		1, "usr_abc",
		validEmail(t), validName(t),
		authorization.RoleUser, vo.StatusActive,
		time.Now(), time.Now(),
		1,
		nil,
	)

	require.NoError(t, err)
	require.NotNil(t, u)
	assert.False(t, u.HasPassword())
	assert.False(t, u.IsEmailVerified())
}

// =============================================================================
// SetID Tests
// =============================================================================

// TestUser_SetID_Valid verifies setting the ID on a new user.
func TestUser_SetID_Valid(t *testing.T) {
	u := newTestUser(t)

	err := u.SetID(42)

	require.NoError(t, err)
	assert.Equal(t, uint(42), u.ID())
}

// TestUser_SetID_Zero verifies that setting zero ID fails.
func TestUser_SetID_Zero(t *testing.T) {
	u := newTestUser(t)

	err := u.SetID(0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user ID cannot be zero")
}

// TestUser_SetID_AlreadySet verifies that setting ID twice fails.
func TestUser_SetID_AlreadySet(t *testing.T) {
	u := newTestUser(t)
	require.NoError(t, u.SetID(1))

	err := u.SetID(2)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user ID is already set")
}

// =============================================================================
// State Transition Tests - Activate
// =============================================================================

// TestUser_Activate_FromPending verifies activating a pending user.
func TestUser_Activate_FromPending(t *testing.T) {
	u := reconstructUserWithStatus(t, vo.StatusPending)
	origVersion := u.Version()

	err := u.Activate()

	require.NoError(t, err)
	assert.Equal(t, vo.StatusActive, u.Status())
	assert.Equal(t, origVersion+1, u.Version())
}

// TestUser_Activate_FromInactive verifies activating an inactive user.
func TestUser_Activate_FromInactive(t *testing.T) {
	u := reconstructUserWithStatus(t, vo.StatusInactive)

	err := u.Activate()

	require.NoError(t, err)
	assert.Equal(t, vo.StatusActive, u.Status())
}

// TestUser_Activate_FromSuspended verifies activating a suspended user.
func TestUser_Activate_FromSuspended(t *testing.T) {
	u := reconstructUserWithStatus(t, vo.StatusSuspended)

	err := u.Activate()

	require.NoError(t, err)
	assert.Equal(t, vo.StatusActive, u.Status())
}

// TestUser_Activate_Idempotent verifies that activating an already active user is a no-op.
func TestUser_Activate_Idempotent(t *testing.T) {
	u := reconstructActiveUser(t)
	origVersion := u.Version()
	origUpdatedAt := u.UpdatedAt()

	err := u.Activate()

	require.NoError(t, err)
	assert.Equal(t, vo.StatusActive, u.Status())
	assert.Equal(t, origVersion, u.Version(), "version should not change for idempotent activation")
	assert.Equal(t, origUpdatedAt, u.UpdatedAt(), "updatedAt should not change for idempotent activation")
}

// TestUser_Activate_FromDeleted verifies that activating a deleted user fails.
func TestUser_Activate_FromDeleted(t *testing.T) {
	u := reconstructUserWithStatus(t, vo.StatusDeleted)

	err := u.Activate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot activate user with status deleted")
	assert.Equal(t, vo.StatusDeleted, u.Status(), "status should remain deleted")
}

// =============================================================================
// State Transition Tests - Deactivate
// =============================================================================

// TestUser_Deactivate_FromActive verifies deactivating an active user.
func TestUser_Deactivate_FromActive(t *testing.T) {
	u := reconstructActiveUser(t)
	origVersion := u.Version()

	err := u.Deactivate("user request")

	require.NoError(t, err)
	assert.Equal(t, vo.StatusInactive, u.Status())
	assert.Equal(t, origVersion+1, u.Version())
}

// TestUser_Deactivate_FromPending verifies deactivating a pending user.
func TestUser_Deactivate_FromPending(t *testing.T) {
	u := reconstructUserWithStatus(t, vo.StatusPending)

	err := u.Deactivate("admin decision")

	require.NoError(t, err)
	assert.Equal(t, vo.StatusInactive, u.Status())
}

// TestUser_Deactivate_Idempotent verifies that deactivating an already inactive user is a no-op.
func TestUser_Deactivate_Idempotent(t *testing.T) {
	u := reconstructUserWithStatus(t, vo.StatusInactive)
	origVersion := u.Version()

	err := u.Deactivate("reason")

	require.NoError(t, err)
	assert.Equal(t, vo.StatusInactive, u.Status())
	assert.Equal(t, origVersion, u.Version(), "version should not change for idempotent deactivation")
}

// TestUser_Deactivate_EmptyReason verifies that deactivating with empty reason still works
// (the method sets a default reason internally).
func TestUser_Deactivate_EmptyReason(t *testing.T) {
	u := reconstructActiveUser(t)

	err := u.Deactivate("")

	require.NoError(t, err)
	assert.Equal(t, vo.StatusInactive, u.Status())
}

// TestUser_Deactivate_FromDeleted verifies that deactivating a deleted user fails.
func TestUser_Deactivate_FromDeleted(t *testing.T) {
	u := reconstructUserWithStatus(t, vo.StatusDeleted)

	err := u.Deactivate("reason")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot deactivate user with status deleted")
}

// =============================================================================
// State Transition Tests - Suspend
// =============================================================================

// TestUser_Suspend_FromActive verifies suspending an active user.
func TestUser_Suspend_FromActive(t *testing.T) {
	u := reconstructActiveUser(t)
	origVersion := u.Version()

	err := u.Suspend("policy violation")

	require.NoError(t, err)
	assert.Equal(t, vo.StatusSuspended, u.Status())
	assert.Equal(t, origVersion+1, u.Version())
}

// TestUser_Suspend_EmptyReason verifies that suspending without a reason fails.
func TestUser_Suspend_EmptyReason(t *testing.T) {
	u := reconstructActiveUser(t)

	err := u.Suspend("")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "suspension reason is required")
	assert.Equal(t, vo.StatusActive, u.Status(), "status should remain active")
}

// TestUser_Suspend_Idempotent verifies that suspending an already suspended user is a no-op.
func TestUser_Suspend_Idempotent(t *testing.T) {
	u := reconstructUserWithStatus(t, vo.StatusSuspended)
	origVersion := u.Version()

	err := u.Suspend("reason")

	require.NoError(t, err)
	assert.Equal(t, vo.StatusSuspended, u.Status())
	assert.Equal(t, origVersion, u.Version())
}

// TestUser_Suspend_FromPending verifies that suspending a pending user fails
// (pending -> suspended is not in the transition table).
func TestUser_Suspend_FromPending(t *testing.T) {
	u := reconstructUserWithStatus(t, vo.StatusPending)

	err := u.Suspend("reason")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot suspend user with status pending")
}

// TestUser_Suspend_FromInactive verifies that suspending an inactive user fails
// (inactive -> suspended is not in the transition table).
func TestUser_Suspend_FromInactive(t *testing.T) {
	u := reconstructUserWithStatus(t, vo.StatusInactive)

	err := u.Suspend("reason")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot suspend user with status inactive")
}

// TestUser_Suspend_FromDeleted verifies that suspending a deleted user fails.
func TestUser_Suspend_FromDeleted(t *testing.T) {
	u := reconstructUserWithStatus(t, vo.StatusDeleted)

	err := u.Suspend("reason")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot suspend user with status deleted")
}

// =============================================================================
// State Transition Tests - Delete
// =============================================================================

// TestUser_Delete_FromActive verifies soft deleting an active user.
func TestUser_Delete_FromActive(t *testing.T) {
	u := reconstructActiveUser(t)
	origVersion := u.Version()

	err := u.Delete()

	require.NoError(t, err)
	assert.Equal(t, vo.StatusDeleted, u.Status())
	assert.Equal(t, origVersion+1, u.Version())
}

// TestUser_Delete_FromPending verifies soft deleting a pending user.
func TestUser_Delete_FromPending(t *testing.T) {
	u := reconstructUserWithStatus(t, vo.StatusPending)

	err := u.Delete()

	require.NoError(t, err)
	assert.Equal(t, vo.StatusDeleted, u.Status())
}

// TestUser_Delete_FromInactive verifies soft deleting an inactive user.
func TestUser_Delete_FromInactive(t *testing.T) {
	u := reconstructUserWithStatus(t, vo.StatusInactive)

	err := u.Delete()

	require.NoError(t, err)
	assert.Equal(t, vo.StatusDeleted, u.Status())
}

// TestUser_Delete_FromSuspended verifies soft deleting a suspended user.
func TestUser_Delete_FromSuspended(t *testing.T) {
	u := reconstructUserWithStatus(t, vo.StatusSuspended)

	err := u.Delete()

	require.NoError(t, err)
	assert.Equal(t, vo.StatusDeleted, u.Status())
}

// TestUser_Delete_Idempotent verifies that deleting an already deleted user is a no-op.
func TestUser_Delete_Idempotent(t *testing.T) {
	u := reconstructUserWithStatus(t, vo.StatusDeleted)
	origVersion := u.Version()

	err := u.Delete()

	require.NoError(t, err)
	assert.Equal(t, vo.StatusDeleted, u.Status())
	assert.Equal(t, origVersion, u.Version())
}

// TestUser_Delete_Terminal verifies that a deleted user cannot transition to any other state.
func TestUser_Delete_Terminal(t *testing.T) {
	u := reconstructUserWithStatus(t, vo.StatusDeleted)

	tests := []struct {
		name   string
		action func() error
	}{
		{"activate", func() error { return u.Activate() }},
		{"deactivate", func() error { return u.Deactivate("reason") }},
		{"suspend", func() error { return u.Suspend("reason") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.action()
			assert.Error(t, err)
			assert.Equal(t, vo.StatusDeleted, u.Status())
		})
	}
}

// =============================================================================
// State Transition Tests - All transitions table-driven
// =============================================================================

// TestUser_StateTransitions_Comprehensive validates all status transitions with a table-driven test.
func TestUser_StateTransitions_Comprehensive(t *testing.T) {
	type transition struct {
		from    vo.Status
		to      string // operation name
		wantErr bool
	}

	tests := []transition{
		// From Pending
		{vo.StatusPending, "activate", false},
		{vo.StatusPending, "deactivate", false},
		{vo.StatusPending, "suspend", true},   // not allowed
		{vo.StatusPending, "delete", false},

		// From Active
		{vo.StatusActive, "activate", false},   // idempotent
		{vo.StatusActive, "deactivate", false},
		{vo.StatusActive, "suspend", false},
		{vo.StatusActive, "delete", false},

		// From Inactive
		{vo.StatusInactive, "activate", false},
		{vo.StatusInactive, "deactivate", false}, // idempotent
		{vo.StatusInactive, "suspend", true},      // not allowed
		{vo.StatusInactive, "delete", false},

		// From Suspended
		{vo.StatusSuspended, "activate", false},
		{vo.StatusSuspended, "deactivate", false},
		{vo.StatusSuspended, "suspend", false}, // idempotent
		{vo.StatusSuspended, "delete", false},

		// From Deleted (terminal)
		{vo.StatusDeleted, "activate", true},
		{vo.StatusDeleted, "deactivate", true},
		{vo.StatusDeleted, "suspend", true},
		{vo.StatusDeleted, "delete", false}, // idempotent
	}

	for _, tt := range tests {
		name := fmt.Sprintf("%s_to_%s", tt.from, tt.to)
		t.Run(name, func(t *testing.T) {
			u := reconstructUserWithStatus(t, tt.from)

			var err error
			switch tt.to {
			case "activate":
				err = u.Activate()
			case "deactivate":
				err = u.Deactivate("reason")
			case "suspend":
				err = u.Suspend("policy violation")
			case "delete":
				err = u.Delete()
			}

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Business Logic Tests - ChangePassword
// =============================================================================

// TestUser_ChangePassword_Valid verifies successful password change.
func TestUser_ChangePassword_Valid(t *testing.T) {
	hasher := &mockPasswordHasher{hashPrefix: "bcrypt"}
	u := reconstructUserWithPassword(t, hasher)
	origVersion := u.Version()

	oldPw := validPassword(t, "OldPass123")
	newPw := validPassword(t, "NewPass456")

	err := u.ChangePassword(oldPw, newPw, hasher)

	require.NoError(t, err)
	assert.Equal(t, origVersion+1, u.Version())
	assert.True(t, u.HasPassword())
	authData := u.GetAuthData()
	assert.NotNil(t, authData.LastPasswordChangeAt)

	// Verify new password works
	err = u.VerifyPassword("NewPass456", hasher)
	assert.NoError(t, err)
}

// TestUser_ChangePassword_NoPasswordSet verifies that changing password fails when no password is set.
func TestUser_ChangePassword_NoPasswordSet(t *testing.T) {
	u := newTestUser(t) // No password set
	hasher := &mockPasswordHasher{hashPrefix: "bcrypt"}
	oldPw := validPassword(t, "OldPass123")
	newPw := validPassword(t, "NewPass456")

	err := u.ChangePassword(oldPw, newPw, hasher)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user has no password set")
}

// TestUser_ChangePassword_WrongOldPassword verifies that wrong old password fails.
func TestUser_ChangePassword_WrongOldPassword(t *testing.T) {
	hasher := &mockPasswordHasher{hashPrefix: "bcrypt"}
	u := reconstructUserWithPassword(t, hasher)

	wrongOldPw := validPassword(t, "WrongPass1")
	newPw := validPassword(t, "NewPass456")

	err := u.ChangePassword(wrongOldPw, newPw, hasher)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "old password is incorrect")
}

// TestUser_ChangePassword_HasherFailure verifies that hash failure is propagated.
func TestUser_ChangePassword_HasherFailure(t *testing.T) {
	setupHasher := &mockPasswordHasher{hashPrefix: "bcrypt"}
	u := reconstructUserWithPassword(t, setupHasher)

	oldPw := validPassword(t, "OldPass123")
	newPw := validPassword(t, "NewPass456")

	// Use a special hasher that verifies OK but fails on hash
	hackHasher := &verifyOnlyHasher{
		verifyHasher: setupHasher,
	}

	err := u.ChangePassword(oldPw, newPw, hackHasher)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to hash new password")
}

// verifyOnlyHasher can verify but fails on hash.
type verifyOnlyHasher struct {
	verifyHasher PasswordHasher
}

func (h *verifyOnlyHasher) Hash(_ string) (string, error) {
	return "", fmt.Errorf("hash failure")
}

func (h *verifyOnlyHasher) Verify(password, hash string) error {
	return h.verifyHasher.Verify(password, hash)
}

// =============================================================================
// Business Logic Tests - SetPassword
// =============================================================================

// TestUser_SetPassword_Valid verifies setting a password on a user.
func TestUser_SetPassword_Valid(t *testing.T) {
	u := newTestUser(t)
	hasher := &mockPasswordHasher{hashPrefix: "bcrypt"}
	pw := validPassword(t, "MyPass123")

	err := u.SetPassword(pw, hasher)

	require.NoError(t, err)
	assert.True(t, u.HasPassword())
	authData := u.GetAuthData()
	assert.NotNil(t, authData.LastPasswordChangeAt)
}

// TestUser_SetPassword_NilPassword verifies that setting nil password fails.
func TestUser_SetPassword_NilPassword(t *testing.T) {
	u := newTestUser(t)
	hasher := &mockPasswordHasher{hashPrefix: "bcrypt"}

	err := u.SetPassword(nil, hasher)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "password cannot be nil")
}

// TestUser_SetPassword_HasherFailure verifies that hash failure is propagated.
func TestUser_SetPassword_HasherFailure(t *testing.T) {
	u := newTestUser(t)
	pw := validPassword(t, "MyPass123")

	err := u.SetPassword(pw, &failingPasswordHasher{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to hash password")
}

// =============================================================================
// Business Logic Tests - VerifyPassword
// =============================================================================

// TestUser_VerifyPassword_Valid verifies successful password verification.
func TestUser_VerifyPassword_Valid(t *testing.T) {
	hasher := &mockPasswordHasher{hashPrefix: "bcrypt"}
	u := reconstructUserWithPassword(t, hasher)

	err := u.VerifyPassword("OldPass123", hasher)

	assert.NoError(t, err)
}

// TestUser_VerifyPassword_Invalid verifies that wrong password fails.
func TestUser_VerifyPassword_Invalid(t *testing.T) {
	hasher := &mockPasswordHasher{hashPrefix: "bcrypt"}
	u := reconstructUserWithPassword(t, hasher)

	err := u.VerifyPassword("WrongPassword1", hasher)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid password")
}

// TestUser_VerifyPassword_NoPasswordSet verifies that verification fails when no password is set.
func TestUser_VerifyPassword_NoPasswordSet(t *testing.T) {
	u := newTestUser(t)
	hasher := &mockPasswordHasher{hashPrefix: "bcrypt"}

	err := u.VerifyPassword("AnyPass123", hasher)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user has no password set")
}

// =============================================================================
// Business Logic Tests - UpdateEmail
// =============================================================================

// TestUser_UpdateEmail_Valid verifies successful email update.
func TestUser_UpdateEmail_Valid(t *testing.T) {
	u := newTestUser(t)
	origVersion := u.Version()
	newEmail := validEmailWithAddr(t, "new@example.com")

	err := u.UpdateEmail(newEmail)

	require.NoError(t, err)
	assert.Equal(t, newEmail, u.Email())
	assert.Equal(t, origVersion+1, u.Version())
}

// TestUser_UpdateEmail_NilEmail verifies that nil email update fails.
func TestUser_UpdateEmail_NilEmail(t *testing.T) {
	u := newTestUser(t)

	err := u.UpdateEmail(nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "email cannot be nil")
}

// TestUser_UpdateEmail_SameEmail verifies that updating to the same email is a no-op.
func TestUser_UpdateEmail_SameEmail(t *testing.T) {
	u := newTestUser(t)
	origVersion := u.Version()
	sameEmail := validEmailWithAddr(t, "test@example.com")

	err := u.UpdateEmail(sameEmail)

	require.NoError(t, err)
	assert.Equal(t, origVersion, u.Version(), "version should not change when email is the same")
}

// =============================================================================
// Business Logic Tests - UpdateName
// =============================================================================

// TestUser_UpdateName_Valid verifies successful name update.
func TestUser_UpdateName_Valid(t *testing.T) {
	u := newTestUser(t)
	origVersion := u.Version()
	newName := validNameWithValue(t, "Jane Smith")

	err := u.UpdateName(newName)

	require.NoError(t, err)
	assert.Equal(t, newName, u.Name())
	assert.Equal(t, origVersion+1, u.Version())
}

// TestUser_UpdateName_NilName verifies that nil name update fails.
func TestUser_UpdateName_NilName(t *testing.T) {
	u := newTestUser(t)

	err := u.UpdateName(nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name cannot be nil")
}

// TestUser_UpdateName_SameName verifies that updating to the same name is a no-op
// (Name.Equals is case-insensitive).
func TestUser_UpdateName_SameName(t *testing.T) {
	u := newTestUser(t)
	origVersion := u.Version()
	sameName := validNameWithValue(t, "john doe") // different case but equal

	err := u.UpdateName(sameName)

	require.NoError(t, err)
	assert.Equal(t, origVersion, u.Version(), "version should not change when name is the same (case-insensitive)")
}

// =============================================================================
// Business Logic Tests - CanPerformActions / RequiresVerification
// =============================================================================

// TestUser_CanPerformActions verifies that only active users can perform actions.
func TestUser_CanPerformActions(t *testing.T) {
	tests := []struct {
		status vo.Status
		canAct bool
	}{
		{vo.StatusActive, true},
		{vo.StatusInactive, false},
		{vo.StatusPending, false},
		{vo.StatusSuspended, false},
		{vo.StatusDeleted, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			u := reconstructUserWithStatus(t, tt.status)
			assert.Equal(t, tt.canAct, u.CanPerformActions())
		})
	}
}

// TestUser_RequiresVerification verifies that only pending users require verification.
func TestUser_RequiresVerification(t *testing.T) {
	tests := []struct {
		status   vo.Status
		requires bool
	}{
		{vo.StatusPending, true},
		{vo.StatusActive, false},
		{vo.StatusInactive, false},
		{vo.StatusSuspended, false},
		{vo.StatusDeleted, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			u := reconstructUserWithStatus(t, tt.status)
			assert.Equal(t, tt.requires, u.RequiresVerification())
		})
	}
}

// =============================================================================
// Business Logic Tests - Email Verification
// =============================================================================

// TestUser_GenerateEmailVerificationToken verifies token generation.
func TestUser_GenerateEmailVerificationToken(t *testing.T) {
	u := newTestUser(t)

	token, err := u.GenerateEmailVerificationToken()

	require.NoError(t, err)
	require.NotNil(t, token)
	assert.NotEmpty(t, token.Value())
	assert.NotEmpty(t, token.Hash())

	// Auth data should contain the token hash and expiry
	authData := u.GetAuthData()
	assert.NotNil(t, authData.EmailVerificationToken)
	assert.NotNil(t, authData.EmailVerificationExpiresAt)
}

// TestUser_VerifyEmail_Valid verifies successful email verification with valid token.
func TestUser_VerifyEmail_Valid(t *testing.T) {
	u := newTestUser(t)
	token, err := u.GenerateEmailVerificationToken()
	require.NoError(t, err)
	origVersion := u.Version()

	err = u.VerifyEmail(token.Value())

	require.NoError(t, err)
	assert.True(t, u.IsEmailVerified())
	assert.Equal(t, origVersion+1, u.Version())

	// Token should be cleared after verification
	authData := u.GetAuthData()
	assert.Nil(t, authData.EmailVerificationToken)
	assert.Nil(t, authData.EmailVerificationExpiresAt)
}

// TestUser_VerifyEmail_AlreadyVerified verifies that re-verifying fails.
func TestUser_VerifyEmail_AlreadyVerified(t *testing.T) {
	u := newTestUser(t)
	token, err := u.GenerateEmailVerificationToken()
	require.NoError(t, err)
	require.NoError(t, u.VerifyEmail(token.Value()))

	// Generate a new token would normally be needed, but since email is already verified
	err = u.VerifyEmail(token.Value())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "email is already verified")
}

// TestUser_VerifyEmail_NoTokenGenerated verifies that verifying without a token fails.
func TestUser_VerifyEmail_NoTokenGenerated(t *testing.T) {
	u := newTestUser(t)

	err := u.VerifyEmail("some_token_value_that_is_at_least_32_chars_long")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no verification token found")
}

// TestUser_VerifyEmail_WrongToken verifies that wrong token fails.
func TestUser_VerifyEmail_WrongToken(t *testing.T) {
	u := newTestUser(t)
	_, err := u.GenerateEmailVerificationToken()
	require.NoError(t, err)

	// Create a different valid token
	wrongToken, err := vo.GenerateToken()
	require.NoError(t, err)

	err = u.VerifyEmail(wrongToken.Value())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid verification token")
}

// =============================================================================
// Business Logic Tests - Password Reset
// =============================================================================

// TestUser_GeneratePasswordResetToken verifies token generation.
func TestUser_GeneratePasswordResetToken(t *testing.T) {
	u := newTestUser(t)

	token, err := u.GeneratePasswordResetToken()

	require.NoError(t, err)
	require.NotNil(t, token)
	assert.NotEmpty(t, token.Value())

	authData := u.GetAuthData()
	assert.NotNil(t, authData.PasswordResetToken)
	assert.NotNil(t, authData.PasswordResetExpiresAt)
}

// TestUser_ResetPassword_Valid verifies successful password reset.
func TestUser_ResetPassword_Valid(t *testing.T) {
	hasher := &mockPasswordHasher{hashPrefix: "bcrypt"}
	u := reconstructUserWithPassword(t, hasher)

	token, err := u.GeneratePasswordResetToken()
	require.NoError(t, err)

	newPw := validPassword(t, "ResetPass789")

	err = u.ResetPassword(token.Value(), newPw, hasher)

	require.NoError(t, err)
	assert.True(t, u.HasPassword())

	// Reset token should be cleared
	authData := u.GetAuthData()
	assert.Nil(t, authData.PasswordResetToken)
	assert.Nil(t, authData.PasswordResetExpiresAt)
	assert.Equal(t, 0, authData.FailedLoginAttempts)
	assert.Nil(t, authData.LockedUntil)

	// New password should work
	err = u.VerifyPassword("ResetPass789", hasher)
	assert.NoError(t, err)
}

// TestUser_ResetPassword_NoToken verifies that reset without token fails.
func TestUser_ResetPassword_NoToken(t *testing.T) {
	hasher := &mockPasswordHasher{hashPrefix: "bcrypt"}
	u := reconstructUserWithPassword(t, hasher)

	newPw := validPassword(t, "ResetPass789")

	err := u.ResetPassword("some_token_value_that_is_at_least_32_chars_long", newPw, hasher)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no password reset token found")
}

// TestUser_ResetPassword_WrongToken verifies that wrong token fails.
func TestUser_ResetPassword_WrongToken(t *testing.T) {
	hasher := &mockPasswordHasher{hashPrefix: "bcrypt"}
	u := reconstructUserWithPassword(t, hasher)

	_, err := u.GeneratePasswordResetToken()
	require.NoError(t, err)

	wrongToken, err := vo.GenerateToken()
	require.NoError(t, err)

	newPw := validPassword(t, "ResetPass789")

	err = u.ResetPassword(wrongToken.Value(), newPw, hasher)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid reset token")
}

// =============================================================================
// Business Logic Tests - AdminResetPassword
// =============================================================================

// TestUser_AdminResetPassword_Valid verifies admin password reset clears security state.
func TestUser_AdminResetPassword_Valid(t *testing.T) {
	hasher := &mockPasswordHasher{hashPrefix: "bcrypt"}

	// Create user with password, failed logins, and a lock
	hash, err := hasher.Hash("OldPass123")
	require.NoError(t, err)
	lockedUntil := time.Now().Add(15 * time.Minute)
	u, err := ReconstructUserWithAuth(
		1, "usr_abc",
		validEmail(t), validName(t),
		authorization.RoleUser, vo.StatusActive,
		time.Now(), time.Now(),
		1,
		&UserAuthData{
			PasswordHash:        &hash,
			FailedLoginAttempts: 5,
			LockedUntil:         &lockedUntil,
		},
	)
	require.NoError(t, err)

	newPw := validPassword(t, "AdminReset1")
	err = u.AdminResetPassword(newPw, hasher)

	require.NoError(t, err)
	authData := u.GetAuthData()
	assert.Nil(t, authData.PasswordResetToken)
	assert.Nil(t, authData.PasswordResetExpiresAt)
	assert.Equal(t, 0, authData.FailedLoginAttempts)
	assert.Nil(t, authData.LockedUntil)

	// New password should work
	err = u.VerifyPassword("AdminReset1", hasher)
	assert.NoError(t, err)
}

// =============================================================================
// Business Logic Tests - Failed Login Attempts / Lockout
// =============================================================================

// TestUser_RecordFailedLoginWithPolicy verifies failed login counting and lockout.
func TestUser_RecordFailedLoginWithPolicy(t *testing.T) {
	u := reconstructActiveUser(t)
	policy := &SecurityPolicy{
		MaxLoginAttempts:       3,
		LockoutDurationMinutes: 10,
	}

	// Record failures below threshold
	u.RecordFailedLoginWithPolicy(policy)
	assert.False(t, u.IsLocked())
	assert.Equal(t, 1, u.GetAuthData().FailedLoginAttempts)

	u.RecordFailedLoginWithPolicy(policy)
	assert.False(t, u.IsLocked())
	assert.Equal(t, 2, u.GetAuthData().FailedLoginAttempts)

	// Third failure should trigger lockout
	u.RecordFailedLoginWithPolicy(policy)
	assert.True(t, u.IsLocked())
	assert.Equal(t, 3, u.GetAuthData().FailedLoginAttempts)
	assert.NotNil(t, u.GetAuthData().LockedUntil)
}

// TestUser_RecordFailedLoginWithPolicy_NilPolicy verifies that nil policy uses defaults.
func TestUser_RecordFailedLoginWithPolicy_NilPolicy(t *testing.T) {
	u := reconstructActiveUser(t)

	// Should not panic with nil policy
	u.RecordFailedLoginWithPolicy(nil)

	assert.Equal(t, 1, u.GetAuthData().FailedLoginAttempts)
}

// TestUser_IsLocked_NotLocked verifies unlocked state.
func TestUser_IsLocked_NotLocked(t *testing.T) {
	u := reconstructActiveUser(t)
	assert.False(t, u.IsLocked())
}

// TestUser_IsLocked_ExpiredLock verifies that an expired lock is not locked.
func TestUser_IsLocked_ExpiredLock(t *testing.T) {
	pastLock := time.Now().Add(-1 * time.Hour)
	u, err := ReconstructUserWithAuth(
		1, "usr_abc",
		validEmail(t), validName(t),
		authorization.RoleUser, vo.StatusActive,
		time.Now(), time.Now(),
		1,
		&UserAuthData{
			LockedUntil: &pastLock,
		},
	)
	require.NoError(t, err)

	assert.False(t, u.IsLocked(), "expired lock should not be considered locked")
}

// TestUser_VerifyPassword_ResetsFailedAttempts verifies that successful login resets counter.
func TestUser_VerifyPassword_ResetsFailedAttempts(t *testing.T) {
	hasher := &mockPasswordHasher{hashPrefix: "bcrypt"}
	hash, err := hasher.Hash("GoodPass123")
	require.NoError(t, err)
	u, err := ReconstructUserWithAuth(
		1, "usr_abc",
		validEmail(t), validName(t),
		authorization.RoleUser, vo.StatusActive,
		time.Now(), time.Now(),
		1,
		&UserAuthData{
			PasswordHash:        &hash,
			FailedLoginAttempts: 3,
		},
	)
	require.NoError(t, err)

	err = u.VerifyPassword("GoodPass123", hasher)

	require.NoError(t, err)
	assert.Equal(t, 0, u.GetAuthData().FailedLoginAttempts)
}

// =============================================================================
// Business Logic Tests - SetRole
// =============================================================================

// TestUser_SetRole verifies role change.
func TestUser_SetRole(t *testing.T) {
	u := newTestUser(t)
	assert.Equal(t, authorization.RoleUser, u.Role())
	assert.False(t, u.IsAdmin())
	origVersion := u.Version()

	u.SetRole(authorization.RoleAdmin)

	assert.Equal(t, authorization.RoleAdmin, u.Role())
	assert.True(t, u.IsAdmin())
	assert.Equal(t, origVersion+1, u.Version())
}

// =============================================================================
// Business Logic Tests - Display Info
// =============================================================================

// TestUser_GetDisplayInfo verifies display info formatting.
func TestUser_GetDisplayInfo(t *testing.T) {
	u := reconstructActiveUser(t)

	info := u.GetDisplayInfo()

	assert.Equal(t, u.SID(), info.ID)
	assert.Equal(t, "test@example.com", info.Email)
	assert.NotEmpty(t, info.DisplayName)
	assert.NotEmpty(t, info.Initials)
	assert.Equal(t, "user", info.Role)
	assert.Equal(t, "active", info.Status)
}

// =============================================================================
// Business Logic Tests - IsBusinessEmail
// =============================================================================

// TestUser_IsBusinessEmail verifies business email detection.
func TestUser_IsBusinessEmail(t *testing.T) {
	tests := []struct {
		email      string
		isBusiness bool
	}{
		{"user@company.com", true},
		{"user@myorg.io", true},
		{"user@gmail.com", false},
		{"user@yahoo.com", false},
		{"user@hotmail.com", false},
		{"user@outlook.com", false},
		{"user@qq.com", false},
		{"user@163.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			gen := mockShortIDGenerator()
			email := validEmailWithAddr(t, tt.email)
			u, err := NewUser(email, validName(t), gen)
			require.NoError(t, err)
			assert.Equal(t, tt.isBusiness, u.IsBusinessEmail())
		})
	}
}

// =============================================================================
// Business Logic Tests - AnnouncementsReadAt
// =============================================================================

// TestUser_MarkAnnouncementsAsRead verifies announcement read timestamp.
func TestUser_MarkAnnouncementsAsRead(t *testing.T) {
	u := newTestUser(t)
	assert.Nil(t, u.AnnouncementsReadAt())

	u.MarkAnnouncementsAsRead()

	readAt := u.AnnouncementsReadAt()
	require.NotNil(t, readAt)
	// Verify it's a recent timestamp
	assert.WithinDuration(t, time.Now(), *readAt, 5*time.Second)
}

// =============================================================================
// Validate Tests
// =============================================================================

// TestUser_Validate_Valid verifies that a valid user passes validation.
func TestUser_Validate_Valid(t *testing.T) {
	u := newTestUser(t)
	assert.NoError(t, u.Validate())
}

// =============================================================================
// Edge Cases / Boundary Values
// =============================================================================

// TestNewEmail_Boundary verifies email boundary conditions.
func TestNewEmail_Boundary(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"valid simple", "a@b.co", false},
		{"empty", "", true},
		{"no at sign", "invalid", true},
		{"no domain", "user@", true},
		{"no local part", "@domain.com", true},
		{"spaces only", "   ", true},
		{"max length", strings.Repeat("a", 243) + "@example.com", false}, // 255 total
		{"exceeds max length", strings.Repeat("a", 244) + "@example.com", true}, // 256 total
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := vo.NewEmail(tt.email)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestNewName_Boundary verifies name boundary conditions.
func TestNewName_Boundary(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid two chars", "AB", false},
		{"single char", "A", true},
		{"empty", "", true},
		{"spaces only", "   ", true},
		{"consecutive spaces", "John  Doe", true},
		{"100 chars", strings.Repeat("A", 100), false},
		{"101 chars", strings.Repeat("A", 101), true},
		{"with hyphen", "Mary-Jane", false},
		{"with apostrophe", "O'Brien", false},
		{"with period", "Dr. Smith", false},
		{"unicode CJK", "Zhang San", false},
		{"special chars", "John@Doe", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := vo.NewName(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestNewPassword_Boundary verifies password boundary conditions.
func TestNewPassword_Boundary(t *testing.T) {
	tests := []struct {
		name    string
		pw      string
		wantErr bool
	}{
		{"valid 8 chars", "Abcdefg1", false},
		{"too short", "Ab1defg", true},
		{"no letters", "12345678", true},
		{"no numbers", "Abcdefgh", true},
		{"72 chars valid", strings.Repeat("Ab1", 24), false},
		{"73 chars too long", strings.Repeat("Ab1", 24) + "x", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := vo.NewPassword(tt.pw)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestUser_HasPassword verifies HasPassword for various states.
func TestUser_HasPassword(t *testing.T) {
	// User without password
	u := newTestUser(t)
	assert.False(t, u.HasPassword())

	// User with password
	hasher := &mockPasswordHasher{hashPrefix: "bcrypt"}
	pw := validPassword(t, "TestPass1")
	require.NoError(t, u.SetPassword(pw, hasher))
	assert.True(t, u.HasPassword())
}

// TestUser_HasPassword_EmptyHash verifies that an empty hash is treated as no password.
func TestUser_HasPassword_EmptyHash(t *testing.T) {
	emptyHash := ""
	u, err := ReconstructUserWithAuth(
		1, "usr_abc",
		validEmail(t), validName(t),
		authorization.RoleUser, vo.StatusActive,
		time.Now(), time.Now(),
		1,
		&UserAuthData{
			PasswordHash: &emptyHash,
		},
	)
	require.NoError(t, err)

	assert.False(t, u.HasPassword(), "empty hash string should be treated as no password")
}

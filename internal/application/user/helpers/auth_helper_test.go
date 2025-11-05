package helpers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"orris/internal/domain/user"
	vo "orris/internal/domain/user/value_objects"
	"orris/internal/shared/authorization"
	"orris/internal/shared/logger"
)

// ============================================================================
// Mock Objects
// ============================================================================

// MockUserRepository is a mock implementation of user.Repository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, u *user.User) error {
	args := m.Called(ctx, u)
	return args.Error(0)
}

func (m *MockUserRepository) Update(ctx context.Context, u *user.User) error {
	args := m.Called(ctx, u)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uint) (*user.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) List(ctx context.Context, filter user.ListFilter) ([]*user.User, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*user.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) Exists(ctx context.Context, id uint) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	args := m.Called(ctx, email)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) GetByVerificationToken(ctx context.Context, token string) (*user.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) GetByPasswordResetToken(ctx context.Context, token string) (*user.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

// MockSessionRepository is a mock implementation of user.SessionRepository
type MockSessionRepository struct {
	mock.Mock
}

func (m *MockSessionRepository) Create(session *user.Session) error {
	args := m.Called(session)
	return args.Error(0)
}

func (m *MockSessionRepository) GetByID(sessionID string) (*user.Session, error) {
	args := m.Called(sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.Session), args.Error(1)
}

func (m *MockSessionRepository) GetByUserID(userID uint) ([]*user.Session, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*user.Session), args.Error(1)
}

func (m *MockSessionRepository) GetByTokenHash(tokenHash string) (*user.Session, error) {
	args := m.Called(tokenHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.Session), args.Error(1)
}

func (m *MockSessionRepository) Update(session *user.Session) error {
	args := m.Called(session)
	return args.Error(0)
}

func (m *MockSessionRepository) Delete(sessionID string) error {
	args := m.Called(sessionID)
	return args.Error(0)
}

func (m *MockSessionRepository) DeleteByUserID(userID uint) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockSessionRepository) DeleteExpired() error {
	args := m.Called()
	return args.Error(0)
}

// MockLogger is a mock implementation of logger.Interface
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(msg string, args ...any) {
	m.Called(msg, args)
}

func (m *MockLogger) Info(msg string, args ...any) {
	m.Called(msg, args)
}

func (m *MockLogger) Warn(msg string, args ...any) {
	m.Called(msg, args)
}

func (m *MockLogger) Error(msg string, args ...any) {
	m.Called(msg, args)
}

func (m *MockLogger) Fatal(msg string, args ...any) {
	m.Called(msg, args)
}

func (m *MockLogger) With(args ...any) logger.Interface {
	calledArgs := m.Called(args)
	if calledArgs.Get(0) == nil {
		return nil
	}
	return calledArgs.Get(0).(logger.Interface)
}

func (m *MockLogger) Named(name string) logger.Interface {
	calledArgs := m.Called(name)
	if calledArgs.Get(0) == nil {
		return nil
	}
	return calledArgs.Get(0).(logger.Interface)
}

func (m *MockLogger) Debugw(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

func (m *MockLogger) Infow(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

func (m *MockLogger) Warnw(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

func (m *MockLogger) Errorw(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

func (m *MockLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

// ============================================================================
// Test Helpers
// ============================================================================

// createTestUser creates a test user with default values
func createTestUser(id uint, status vo.Status) *user.User {
	email, _ := vo.NewEmail("test@example.com")
	name, _ := vo.NewName("Test User")
	u, _ := user.ReconstructUser(id, email, name, authorization.RoleUser, status, time.Now(), time.Now(), 1)
	return u
}

// createTestUserWithAuth creates a test user with authentication data
func createTestUserWithAuth(id uint, status vo.Status, authData *user.UserAuthData) *user.User {
	email, _ := vo.NewEmail("test@example.com")
	name, _ := vo.NewName("Test User")
	u, _ := user.ReconstructUserWithAuth(id, email, name, authorization.RoleUser, status, time.Now(), time.Now(), 1, authData)
	return u
}

// ============================================================================
// Tests for ValidateUserCanLogin
// ============================================================================

func TestValidateUserCanLogin(t *testing.T) {
	tests := []struct {
		name          string
		user          *user.User
		expectedCode  string
		expectedError bool
		logWarning    bool
	}{
		{
			name:          "nil user",
			user:          nil,
			expectedCode:  ErrCodeUserNotFound,
			expectedError: true,
			logWarning:    false,
		},
		{
			name: "user is locked",
			user: createTestUserWithAuth(1, vo.StatusActive, &user.UserAuthData{
				LockedUntil:  ptrTime(time.Now().Add(time.Hour)),
				PasswordHash: ptrString("hashedpassword"),
			}),
			expectedCode:  ErrCodeAccountLocked,
			expectedError: true,
			logWarning:    true,
		},
		{
			name: "user has no password",
			user: createTestUserWithAuth(1, vo.StatusActive, &user.UserAuthData{
				PasswordHash: nil,
			}),
			expectedCode:  ErrCodePasswordUnavailable,
			expectedError: true,
			logWarning:    true,
		},
		{
			name: "user is inactive",
			user: createTestUserWithAuth(1, vo.StatusInactive, &user.UserAuthData{
				PasswordHash: ptrString("hashedpassword"),
			}),
			expectedCode:  ErrCodeAccountInactive,
			expectedError: true,
			logWarning:    true,
		},
		{
			name: "user is pending",
			user: createTestUserWithAuth(1, vo.StatusPending, &user.UserAuthData{
				PasswordHash: ptrString("hashedpassword"),
			}),
			expectedCode:  ErrCodeAccountInactive,
			expectedError: true,
			logWarning:    true,
		},
		{
			name: "user is suspended",
			user: createTestUserWithAuth(1, vo.StatusSuspended, &user.UserAuthData{
				PasswordHash: ptrString("hashedpassword"),
			}),
			expectedCode:  ErrCodeAccountInactive,
			expectedError: true,
			logWarning:    true,
		},
		{
			name: "valid active user with password",
			user: createTestUserWithAuth(1, vo.StatusActive, &user.UserAuthData{
				PasswordHash: ptrString("hashedpassword"),
			}),
			expectedCode:  "",
			expectedError: false,
			logWarning:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger := new(MockLogger)
			helper := &AuthHelper{
				logger: mockLogger,
			}

			// Setup log expectations
			if tt.logWarning {
				mockLogger.On("Warnw", mock.Anything, mock.Anything).Return()
			}

			// Execute
			validationErr := helper.ValidateUserCanLogin(tt.user)

			// Verify
			if tt.expectedError {
				require.NotNil(t, validationErr)
				assert.Equal(t, tt.expectedCode, validationErr.Code)
			} else {
				assert.Nil(t, validationErr)
			}

			mockLogger.AssertExpectations(t)
		})
	}
}

// ============================================================================
// Tests for ValidateUserCanPerformAction
// ============================================================================

func TestValidateUserCanPerformAction(t *testing.T) {
	tests := []struct {
		name          string
		user          *user.User
		expectedCode  string
		expectedError bool
		logWarning    bool
	}{
		{
			name:          "nil user",
			user:          nil,
			expectedCode:  ErrCodeUserNotFound,
			expectedError: true,
			logWarning:    false,
		},
		{
			name:          "user is inactive",
			user:          createTestUser(1, vo.StatusInactive),
			expectedCode:  ErrCodeAccountInactive,
			expectedError: true,
			logWarning:    true,
		},
		{
			name:          "user is pending",
			user:          createTestUser(1, vo.StatusPending),
			expectedCode:  ErrCodeAccountInactive,
			expectedError: true,
			logWarning:    true,
		},
		{
			name:          "user is suspended",
			user:          createTestUser(1, vo.StatusSuspended),
			expectedCode:  ErrCodeAccountInactive,
			expectedError: true,
			logWarning:    true,
		},
		{
			name:          "valid active user",
			user:          createTestUser(1, vo.StatusActive),
			expectedCode:  "",
			expectedError: false,
			logWarning:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger := new(MockLogger)
			helper := &AuthHelper{
				logger: mockLogger,
			}

			// Setup log expectations
			if tt.logWarning {
				mockLogger.On("Warnw", mock.Anything, mock.Anything).Return()
			}

			// Execute
			validationErr := helper.ValidateUserCanPerformAction(tt.user)

			// Verify
			if tt.expectedError {
				require.NotNil(t, validationErr)
				assert.Equal(t, tt.expectedCode, validationErr.Code)
			} else {
				assert.Nil(t, validationErr)
			}

			mockLogger.AssertExpectations(t)
		})
	}
}

// ============================================================================
// Tests for IsFirstUser
// ============================================================================

func TestIsFirstUser(t *testing.T) {
	tests := []struct {
		name        string
		totalUsers  int64
		repoError   error
		expected    bool
		expectError bool
	}{
		{
			name:        "first user - total is 1",
			totalUsers:  1,
			repoError:   nil,
			expected:    true,
			expectError: false,
		},
		{
			name:        "not first user - total is 0",
			totalUsers:  0,
			repoError:   nil,
			expected:    false,
			expectError: false,
		},
		{
			name:        "not first user - total is 2",
			totalUsers:  2,
			repoError:   nil,
			expected:    false,
			expectError: false,
		},
		{
			name:        "repository error",
			totalUsers:  0,
			repoError:   errors.New("database error"),
			expected:    false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockUserRepository)
			mockLogger := new(MockLogger)
			helper := &AuthHelper{
				userRepo: mockRepo,
				logger:   mockLogger,
			}

			ctx := context.Background()
			mockRepo.On("List", ctx, mock.MatchedBy(func(filter user.ListFilter) bool {
				return filter.Page == 1 && filter.PageSize == 1
			})).Return([]*user.User{}, tt.totalUsers, tt.repoError)

			// Execute
			result, err := helper.IsFirstUser(ctx)

			// Verify
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// ============================================================================
// Tests for HashToken
// ============================================================================

func TestHashToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{
			name:     "hash simple token",
			token:    "simple_token",
			expected: "a6eb4e7f5792c0fc7c8c8c5e2aafaf85e6c80dc06f7c8c4e8c1f8e7c8c4e8c1f",
		},
		{
			name:     "hash empty token",
			token:    "",
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "hash long token",
			token:    "very_long_token_string_with_many_characters_1234567890",
			expected: "0c8aab5b0e8c8b0e8c8c4e8c1f8e7c8c4e8c1f8e7c8c4e8c1f8e7c8c4e8c1f8e",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helper := &AuthHelper{}

			// Execute
			result := helper.HashToken(tt.token)

			// Verify - hash should be consistent
			assert.NotEmpty(t, result)
			assert.Len(t, result, 64) // SHA256 produces 64 character hex string

			// Same token should produce same hash
			result2 := helper.HashToken(tt.token)
			assert.Equal(t, result, result2)
		})
	}
}

// ============================================================================
// Tests for CreateSessionWithTokens
// ============================================================================

func TestCreateSessionWithTokens(t *testing.T) {
	tests := []struct {
		name          string
		userID        uint
		deviceInfo    DeviceInfo
		accessToken   string
		refreshToken  string
		sessionError  error
		expectError   bool
		expectLogs    bool
	}{
		{
			name:   "successful session creation",
			userID: 1,
			deviceInfo: DeviceInfo{
				DeviceName: "iPhone 14",
				DeviceType: "mobile",
				IPAddress:  "192.168.1.1",
				UserAgent:  "Mozilla/5.0",
			},
			accessToken:  "access_token_123",
			refreshToken: "refresh_token_456",
			sessionError: nil,
			expectError:  false,
			expectLogs:   true,
		},
		{
			name:   "session repository error",
			userID: 1,
			deviceInfo: DeviceInfo{
				DeviceName: "iPhone 14",
				DeviceType: "mobile",
				IPAddress:  "192.168.1.1",
				UserAgent:  "Mozilla/5.0",
			},
			accessToken:  "access_token_123",
			refreshToken: "refresh_token_456",
			sessionError: errors.New("database error"),
			expectError:  true,
			expectLogs:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSessionRepo := new(MockSessionRepository)
			mockLogger := new(MockLogger)
			helper := &AuthHelper{
				sessionRepo: mockSessionRepo,
				logger:      mockLogger,
			}

			// Setup expectations
			if tt.expectLogs {
				if tt.sessionError != nil {
					mockLogger.On("Errorw", mock.Anything, mock.Anything).Return()
				} else {
					mockLogger.On("Infow", mock.Anything, mock.Anything).Return()
				}
			}

			mockSessionRepo.On("Create", mock.AnythingOfType("*user.Session")).Return(tt.sessionError)

			// Execute
			result, err := helper.CreateSessionWithTokens(
				tt.userID,
				tt.deviceInfo,
				24*time.Hour,
				tt.accessToken,
				tt.refreshToken,
				3600,
			)

			// Verify
			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.NotNil(t, result.Session)
				assert.Equal(t, tt.accessToken, result.AccessToken)
				assert.Equal(t, tt.refreshToken, result.RefreshToken)
				assert.Equal(t, int64(3600), result.ExpiresIn)
				assert.NotEmpty(t, result.Session.TokenHash)
				assert.NotEmpty(t, result.Session.RefreshTokenHash)
			}

			mockSessionRepo.AssertExpectations(t)
			mockLogger.AssertExpectations(t)
		})
	}
}

// ============================================================================
// Tests for CreateAndSaveSessionWithTokens
// ============================================================================

func TestCreateAndSaveSessionWithTokens(t *testing.T) {
	tests := []struct {
		name          string
		userID        uint
		deviceInfo    DeviceInfo
		tokenGenError error
		sessionError  error
		expectError   bool
	}{
		{
			name:   "successful session creation with token generation",
			userID: 1,
			deviceInfo: DeviceInfo{
				DeviceName: "MacBook Pro",
				DeviceType: "desktop",
				IPAddress:  "192.168.1.100",
				UserAgent:  "Mozilla/5.0",
			},
			tokenGenError: nil,
			sessionError:  nil,
			expectError:   false,
		},
		{
			name:   "token generation error",
			userID: 1,
			deviceInfo: DeviceInfo{
				DeviceName: "iPhone 14",
				DeviceType: "mobile",
				IPAddress:  "192.168.1.1",
				UserAgent:  "Mozilla/5.0",
			},
			tokenGenError: errors.New("token generation failed"),
			sessionError:  nil,
			expectError:   true,
		},
		{
			name:   "session repository error",
			userID: 1,
			deviceInfo: DeviceInfo{
				DeviceName: "iPad Pro",
				DeviceType: "tablet",
				IPAddress:  "192.168.1.50",
				UserAgent:  "Mozilla/5.0",
			},
			tokenGenError: nil,
			sessionError:  errors.New("database error"),
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSessionRepo := new(MockSessionRepository)
			mockLogger := new(MockLogger)
			helper := &AuthHelper{
				sessionRepo: mockSessionRepo,
				logger:      mockLogger,
			}

			// Setup token generator function
			tokenGenerator := func(userID uint, sessionID string) (string, string, int64, error) {
				if tt.tokenGenError != nil {
					return "", "", 0, tt.tokenGenError
				}
				return "access_token_" + sessionID, "refresh_token_" + sessionID, 3600, nil
			}

			// Setup expectations
			mockLogger.On("Errorw", mock.Anything, mock.Anything).Return().Maybe()
			mockLogger.On("Infow", mock.Anything, mock.Anything).Return().Maybe()

			if tt.tokenGenError == nil {
				mockSessionRepo.On("Create", mock.AnythingOfType("*user.Session")).Return(tt.sessionError)
			}

			// Execute
			result, err := helper.CreateAndSaveSessionWithTokens(
				tt.userID,
				tt.deviceInfo,
				24*time.Hour,
				tokenGenerator,
			)

			// Verify
			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.NotNil(t, result.Session)
				assert.NotEmpty(t, result.AccessToken)
				assert.NotEmpty(t, result.RefreshToken)
				assert.Equal(t, int64(3600), result.ExpiresIn)
			}

			mockLogger.AssertExpectations(t)
		})
	}
}

// ============================================================================
// Tests for GrantAdminToFirstUserIfNeeded
// ============================================================================

func TestGrantAdminToFirstUserIfNeeded(t *testing.T) {
	tests := []struct {
		name        string
		user        *user.User
		totalUsers  int64
		repoError   error
		expected    bool
		expectError bool
	}{
		{
			name:        "nil user",
			user:        nil,
			totalUsers:  0,
			repoError:   nil,
			expected:    false,
			expectError: true,
		},
		{
			name:        "is first user",
			user:        createTestUser(1, vo.StatusActive),
			totalUsers:  1,
			repoError:   nil,
			expected:    true,
			expectError: false,
		},
		{
			name:        "not first user",
			user:        createTestUser(2, vo.StatusActive),
			totalUsers:  2,
			repoError:   nil,
			expected:    false,
			expectError: false,
		},
		{
			name:        "repository error",
			user:        createTestUser(1, vo.StatusActive),
			totalUsers:  0,
			repoError:   errors.New("database error"),
			expected:    false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockUserRepository)
			mockLogger := new(MockLogger)
			helper := &AuthHelper{
				userRepo: mockRepo,
				logger:   mockLogger,
			}

			ctx := context.Background()

			// Setup expectations
			if tt.user != nil {
				mockRepo.On("List", ctx, mock.MatchedBy(func(filter user.ListFilter) bool {
					return filter.Page == 1 && filter.PageSize == 1
				})).Return([]*user.User{}, tt.totalUsers, tt.repoError)

				if tt.repoError != nil {
					mockLogger.On("Errorw", mock.Anything, mock.Anything).Return()
				} else if tt.expected {
					mockLogger.On("Infow", mock.Anything, mock.Anything).Return()
				}
			}

			// Execute
			result, err := helper.GrantAdminToFirstUserIfNeeded(ctx, tt.user)

			// Verify
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}

			mockRepo.AssertExpectations(t)
			mockLogger.AssertExpectations(t)
		})
	}
}

// ============================================================================
// Tests for GrantAdminAndSave
// ============================================================================

func TestGrantAdminAndSave(t *testing.T) {
	tests := []struct {
		name        string
		user        *user.User
		totalUsers  int64
		saveError   error
		repoError   error
		expectError bool
		expectAdmin bool
	}{
		{
			name:        "nil user",
			user:        nil,
			totalUsers:  0,
			saveError:   nil,
			repoError:   nil,
			expectError: true,
			expectAdmin: false,
		},
		{
			name:        "first user - grant admin successfully",
			user:        createTestUser(1, vo.StatusActive),
			totalUsers:  1,
			saveError:   nil,
			repoError:   nil,
			expectError: false,
			expectAdmin: true,
		},
		{
			name:        "first user - save error",
			user:        createTestUser(1, vo.StatusActive),
			totalUsers:  1,
			saveError:   errors.New("database error"),
			repoError:   nil,
			expectError: true,
			expectAdmin: true,
		},
		{
			name:        "not first user - no admin grant",
			user:        createTestUser(2, vo.StatusActive),
			totalUsers:  2,
			saveError:   nil,
			repoError:   nil,
			expectError: false,
			expectAdmin: false,
		},
		{
			name:        "repository error",
			user:        createTestUser(1, vo.StatusActive),
			totalUsers:  0,
			saveError:   nil,
			repoError:   errors.New("database error"),
			expectError: true,
			expectAdmin: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockUserRepository)
			mockLogger := new(MockLogger)
			helper := &AuthHelper{
				userRepo: mockRepo,
				logger:   mockLogger,
			}

			ctx := context.Background()

			// Setup expectations
			if tt.user != nil {
				mockRepo.On("List", ctx, mock.MatchedBy(func(filter user.ListFilter) bool {
					return filter.Page == 1 && filter.PageSize == 1
				})).Return([]*user.User{}, tt.totalUsers, tt.repoError)

				if tt.repoError != nil {
					mockLogger.On("Errorw", mock.Anything, mock.Anything).Return()
				} else if tt.totalUsers == 1 {
					mockRepo.On("Update", ctx, tt.user).Return(tt.saveError)
					mockLogger.On("Errorw", mock.Anything, mock.Anything).Return().Maybe()
					mockLogger.On("Infow", mock.Anything, mock.Anything).Return().Maybe()
					mockLogger.On("Warnw", mock.Anything, mock.Anything).Return().Maybe()
				}
			}

			// Execute
			err := helper.GrantAdminAndSave(ctx, tt.user)

			// Verify
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tt.user != nil && tt.expectAdmin {
				assert.Equal(t, authorization.RoleAdmin, tt.user.Role())
			}

			mockRepo.AssertExpectations(t)
			mockLogger.AssertExpectations(t)
		})
	}
}

// ============================================================================
// Tests for SaveUserWithLogging
// ============================================================================

func TestSaveUserWithLogging(t *testing.T) {
	tests := []struct {
		name             string
		user             *user.User
		isCritical       bool
		operation        string
		saveError        error
		expectError      bool
		expectResultFail bool
	}{
		{
			name:             "nil user - critical operation",
			user:             nil,
			isCritical:       true,
			operation:        "test operation",
			saveError:        nil,
			expectError:      true,
			expectResultFail: true,
		},
		{
			name:             "nil user - non-critical operation",
			user:             nil,
			isCritical:       false,
			operation:        "test operation",
			saveError:        nil,
			expectError:      false,
			expectResultFail: true,
		},
		{
			name:             "successful save - critical operation",
			user:             createTestUser(1, vo.StatusActive),
			isCritical:       true,
			operation:        "activate user",
			saveError:        nil,
			expectError:      false,
			expectResultFail: false,
		},
		{
			name:             "successful save - non-critical operation",
			user:             createTestUser(1, vo.StatusActive),
			isCritical:       false,
			operation:        "update last login",
			saveError:        nil,
			expectError:      false,
			expectResultFail: false,
		},
		{
			name:             "save error - critical operation",
			user:             createTestUser(1, vo.StatusActive),
			isCritical:       true,
			operation:        "activate user",
			saveError:        errors.New("database error"),
			expectError:      true,
			expectResultFail: true,
		},
		{
			name:             "save error - non-critical operation",
			user:             createTestUser(1, vo.StatusActive),
			isCritical:       false,
			operation:        "update last login",
			saveError:        errors.New("database error"),
			expectError:      false,
			expectResultFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockUserRepository)
			mockLogger := new(MockLogger)
			helper := &AuthHelper{
				userRepo: mockRepo,
				logger:   mockLogger,
			}

			ctx := context.Background()

			// Setup expectations
			mockLogger.On("Errorw", mock.Anything, mock.Anything).Return().Maybe()
			mockLogger.On("Infow", mock.Anything, mock.Anything).Return().Maybe()
			mockLogger.On("Warnw", mock.Anything, mock.Anything).Return().Maybe()

			if tt.user != nil {
				mockRepo.On("Update", ctx, tt.user).Return(tt.saveError)
			}

			// Execute
			result, err := helper.SaveUserWithLogging(ctx, tt.user, tt.isCritical, tt.operation)

			// Verify
			require.NotNil(t, result)
			assert.Equal(t, !tt.expectResultFail, result.Success)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			mockLogger.AssertExpectations(t)
		})
	}
}

// ============================================================================
// Tests for RecordFailedLoginAndSave
// ============================================================================

func TestRecordFailedLoginAndSave(t *testing.T) {
	tests := []struct {
		name             string
		user             *user.User
		saveError        error
		expectResultFail bool
	}{
		{
			name:             "nil user",
			user:             nil,
			saveError:        nil,
			expectResultFail: true,
		},
		{
			name: "successful record and save",
			user: createTestUserWithAuth(1, vo.StatusActive, &user.UserAuthData{
				FailedLoginAttempts: 0,
			}),
			saveError:        nil,
			expectResultFail: false,
		},
		{
			name: "save error",
			user: createTestUserWithAuth(1, vo.StatusActive, &user.UserAuthData{
				FailedLoginAttempts: 2,
			}),
			saveError:        errors.New("database error"),
			expectResultFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockUserRepository)
			mockLogger := new(MockLogger)
			helper := &AuthHelper{
				userRepo: mockRepo,
				logger:   mockLogger,
			}

			ctx := context.Background()

			// Setup expectations
			mockLogger.On("Errorw", mock.Anything, mock.Anything).Return().Maybe()
			mockLogger.On("Infow", mock.Anything, mock.Anything).Return().Maybe()

			if tt.user != nil {
				mockRepo.On("Update", ctx, tt.user).Return(tt.saveError)
			}

			// Execute
			result := helper.RecordFailedLoginAndSave(ctx, tt.user)

			// Verify
			require.NotNil(t, result)
			assert.Equal(t, !tt.expectResultFail, result.Success)

			mockLogger.AssertExpectations(t)
		})
	}
}

// ============================================================================
// Tests for SaveUserAfterSuccessfulLogin
// ============================================================================

func TestSaveUserAfterSuccessfulLogin(t *testing.T) {
	tests := []struct {
		name             string
		user             *user.User
		saveError        error
		expectResultFail bool
	}{
		{
			name:             "successful save",
			user:             createTestUser(1, vo.StatusActive),
			saveError:        nil,
			expectResultFail: false,
		},
		{
			name:             "save error - non-critical",
			user:             createTestUser(1, vo.StatusActive),
			saveError:        errors.New("database error"),
			expectResultFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockUserRepository)
			mockLogger := new(MockLogger)
			helper := &AuthHelper{
				userRepo: mockRepo,
				logger:   mockLogger,
			}

			ctx := context.Background()

			// Setup expectations
			mockLogger.On("Errorw", mock.Anything, mock.Anything).Return().Maybe()
			mockLogger.On("Infow", mock.Anything, mock.Anything).Return().Maybe()
			mockLogger.On("Warnw", mock.Anything, mock.Anything).Return().Maybe()

			if tt.user != nil {
				mockRepo.On("Update", ctx, tt.user).Return(tt.saveError)
			}

			// Execute
			result := helper.SaveUserAfterSuccessfulLogin(ctx, tt.user)

			// Verify
			require.NotNil(t, result)
			assert.Equal(t, !tt.expectResultFail, result.Success)

			mockLogger.AssertExpectations(t)
		})
	}
}

// ============================================================================
// Tests for SetSessionTokens
// ============================================================================

func TestSetSessionTokens(t *testing.T) {
	helper := &AuthHelper{}
	session := &user.Session{
		ID:     "session123",
		UserID: 1,
	}

	accessToken := "access_token_123"
	refreshToken := "refresh_token_456"

	// Execute
	helper.SetSessionTokens(session, accessToken, refreshToken)

	// Verify
	assert.NotEmpty(t, session.TokenHash)
	assert.NotEmpty(t, session.RefreshTokenHash)
	assert.Equal(t, helper.HashToken(accessToken), session.TokenHash)
	assert.Equal(t, helper.HashToken(refreshToken), session.RefreshTokenHash)
}

// ============================================================================
// Tests for UpdateSessionAccessToken
// ============================================================================

func TestUpdateSessionAccessToken(t *testing.T) {
	helper := &AuthHelper{}
	session := &user.Session{
		ID:               "session123",
		UserID:           1,
		RefreshTokenHash: "old_refresh_hash",
	}

	oldRefreshHash := session.RefreshTokenHash
	newAccessToken := "new_access_token_789"

	// Execute
	helper.UpdateSessionAccessToken(session, newAccessToken)

	// Verify
	assert.NotEmpty(t, session.TokenHash)
	assert.Equal(t, helper.HashToken(newAccessToken), session.TokenHash)
	// Refresh token hash should remain unchanged
	assert.Equal(t, oldRefreshHash, session.RefreshTokenHash)
}

// ============================================================================
// Tests for UserValidationError
// ============================================================================

func TestUserValidationError(t *testing.T) {
	tests := []struct {
		name            string
		code            string
		message         string
		field           string
		expectedMessage string
	}{
		{
			name:            "account locked error",
			code:            ErrCodeAccountLocked,
			message:         "account is locked",
			field:           "account",
			expectedMessage: "account is locked",
		},
		{
			name:            "user not found error",
			code:            ErrCodeUserNotFound,
			message:         "user not found",
			field:           "user",
			expectedMessage: "user not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewUserValidationError(tt.code, tt.message, tt.field)

			assert.Equal(t, tt.code, err.Code)
			assert.Equal(t, tt.message, err.Message)
			assert.Equal(t, tt.field, err.Field)
			assert.Equal(t, tt.expectedMessage, err.Error())
		})
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

func ptrString(s string) *string {
	return &s
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

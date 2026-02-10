package handlers

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/orris-inc/orris/internal/application/user/usecases"
	"github.com/orris-inc/orris/internal/domain/user"
	uservo "github.com/orris-inc/orris/internal/domain/user/valueobjects"
	"github.com/orris-inc/orris/internal/interfaces/http/handlers/testutil"
	"github.com/orris-inc/orris/internal/shared/authorization"
	"github.com/orris-inc/orris/internal/shared/config"
	"github.com/orris-inc/orris/internal/shared/errors"
)

// =====================================================================
// Mock use cases
// =====================================================================

type mockRegisterUC struct {
	result *user.User
	err    error
}

func (m *mockRegisterUC) Execute(ctx context.Context, cmd usecases.RegisterWithPasswordCommand) (*user.User, error) {
	return m.result, m.err
}

type mockLoginUC struct {
	result *usecases.LoginWithPasswordResult
	err    error
}

func (m *mockLoginUC) Execute(ctx context.Context, cmd usecases.LoginWithPasswordCommand) (*usecases.LoginWithPasswordResult, error) {
	return m.result, m.err
}

type mockVerifyEmailUC struct {
	err error
}

func (m *mockVerifyEmailUC) Execute(ctx context.Context, cmd usecases.VerifyEmailCommand) error {
	return m.err
}

type mockRequestResetUC struct {
	err error
}

func (m *mockRequestResetUC) Execute(ctx context.Context, cmd usecases.RequestPasswordResetCommand) error {
	return m.err
}

type mockResetPasswordUC struct {
	err error
}

func (m *mockResetPasswordUC) Execute(ctx context.Context, cmd usecases.ResetPasswordCommand) error {
	return m.err
}

type mockInitiateOAuthUC struct {
	result *usecases.InitiateOAuthLoginResult
	err    error
}

func (m *mockInitiateOAuthUC) Execute(cmd usecases.InitiateOAuthLoginCommand) (*usecases.InitiateOAuthLoginResult, error) {
	return m.result, m.err
}

type mockHandleOAuthUC struct {
	result *usecases.HandleOAuthCallbackResult
	err    error
}

func (m *mockHandleOAuthUC) Execute(ctx context.Context, cmd usecases.HandleOAuthCallbackCommand) (*usecases.HandleOAuthCallbackResult, error) {
	return m.result, m.err
}

type mockRefreshTokenUC struct {
	result *usecases.RefreshTokenResult
	err    error
}

func (m *mockRefreshTokenUC) Execute(ctx context.Context, cmd usecases.RefreshTokenCommand) (*usecases.RefreshTokenResult, error) {
	return m.result, m.err
}

type mockLogoutUC struct {
	err error
}

func (m *mockLogoutUC) Execute(cmd usecases.LogoutCommand) error {
	return m.err
}

// =====================================================================
// Mock repository
// =====================================================================

type mockUserRepo struct {
	user *user.User
	err  error
}

func (m *mockUserRepo) GetByID(ctx context.Context, id uint) (*user.User, error) {
	return m.user, m.err
}

func (m *mockUserRepo) GetBySID(ctx context.Context, sid string) (*user.User, error) {
	return m.user, m.err
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	return m.user, m.err
}

func (m *mockUserRepo) Create(ctx context.Context, u *user.User) error {
	return m.err
}

func (m *mockUserRepo) Update(ctx context.Context, u *user.User) error {
	return m.err
}

func (m *mockUserRepo) ListUsers(ctx context.Context, page, pageSize int, filters map[string]interface{}) ([]*user.User, int, error) {
	return nil, 0, m.err
}

func (m *mockUserRepo) Delete(ctx context.Context, id uint) error {
	return m.err
}

func (m *mockUserRepo) DeleteBySID(ctx context.Context, sid string) error {
	return m.err
}

func (m *mockUserRepo) Exists(ctx context.Context, id uint) (bool, error) {
	return m.user != nil, m.err
}

func (m *mockUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return m.user != nil, m.err
}

func (m *mockUserRepo) GetByIDs(ctx context.Context, ids []uint) ([]*user.User, error) {
	if m.user != nil {
		return []*user.User{m.user}, m.err
	}
	return nil, m.err
}

func (m *mockUserRepo) List(ctx context.Context, filter user.ListFilter) ([]*user.User, int64, error) {
	if m.user != nil {
		return []*user.User{m.user}, 1, m.err
	}
	return nil, 0, m.err
}

func (m *mockUserRepo) GetByVerificationToken(ctx context.Context, token string) (*user.User, error) {
	return m.user, m.err
}

func (m *mockUserRepo) GetByPasswordResetToken(ctx context.Context, token string) (*user.User, error) {
	return m.user, m.err
}

// =====================================================================
// Mock email checker
// =====================================================================

type mockEmailChecker struct {
	configured bool
}

func (m *mockEmailChecker) IsConfigured() bool {
	return m.configured
}

// =====================================================================
// Test helpers
// =====================================================================

func createTestUser() *user.User {
	email, _ := uservo.NewEmail("test@example.com")
	name, _ := uservo.NewName("Test User")
	now := time.Now().UTC()

	u, _ := user.ReconstructUser(
		1, "usr_test123",
		email, name,
		authorization.RoleUser, uservo.StatusActive,
		now, now,
		1,
	)
	return u
}

func newTestAuthHandler(
	registerUC registerUseCase,
	loginUC loginUseCase,
	verifyEmailUC verifyEmailUseCase,
	requestResetUC requestPasswordResetUseCase,
	resetPasswordUC resetPasswordUseCase,
	initiateOAuthUC initiateOAuthUseCase,
	handleOAuthUC handleOAuthCallbackUseCase,
	refreshTokenUC refreshTokenUseCase,
	logoutUC logoutUseCase,
	userRepo user.Repository,
	emailChecker EmailConfigChecker,
) *AuthHandler {
	return NewAuthHandler(
		registerUC, loginUC, verifyEmailUC, requestResetUC, resetPasswordUC,
		initiateOAuthUC, handleOAuthUC, refreshTokenUC, logoutUC,
		userRepo, testutil.NewMockLogger(),
		config.CookieConfig{}, config.JWTConfig{}, config.SessionConfig{},
		"http://localhost:3000/callback", nil, emailChecker,
	)
}

// =====================================================================
// TestAuthHandler_Register
// =====================================================================

func TestAuthHandler_Register_Success(t *testing.T) {
	testUser := createTestUser()
	mockUC := &mockRegisterUC{result: testUser}
	handler := newTestAuthHandler(mockUC, nil, nil, nil, nil, nil, nil, nil, nil, nil, &mockEmailChecker{configured: false})

	reqBody := RegisterRequest{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/auth/register", reqBody)

	handler.Register(c)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	var data RegisterResponse
	err = json.Unmarshal(resp.Data, &data)
	require.NoError(t, err)
	assert.Equal(t, testUser.SID(), data.UserID)
	assert.Equal(t, testUser.Email().String(), data.Email)
	assert.False(t, data.RequiresEmailVerification)
}

func TestAuthHandler_Register_InvalidRequest(t *testing.T) {
	handler := newTestAuthHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := map[string]string{"email": "test@example.com"} // missing name, password
	c, w := testutil.NewTestContext(http.MethodPost, "/auth/register", reqBody)

	handler.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.NotNil(t, resp.Error)
}

func TestAuthHandler_Register_UseCaseError(t *testing.T) {
	mockUC := &mockRegisterUC{err: errors.NewConflictError("email already exists", "")}
	handler := newTestAuthHandler(mockUC, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := RegisterRequest{
		Email:    "test@example.com",
		Name:     "Test",
		Password: "password123",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/auth/register", reqBody)

	handler.Register(c)

	assert.NotEqual(t, http.StatusCreated, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestAuthHandler_Register_WithEmailVerificationRequired(t *testing.T) {
	testUser := createTestUser()
	mockUC := &mockRegisterUC{result: testUser}
	handler := newTestAuthHandler(mockUC, nil, nil, nil, nil, nil, nil, nil, nil, nil, &mockEmailChecker{configured: true})

	reqBody := RegisterRequest{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/auth/register", reqBody)

	handler.Register(c)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	var data RegisterResponse
	err = json.Unmarshal(resp.Data, &data)
	require.NoError(t, err)
	assert.True(t, data.RequiresEmailVerification)
}

// =====================================================================
// TestAuthHandler_Login
// =====================================================================

func TestAuthHandler_Login_Success(t *testing.T) {
	mockResult := &usecases.LoginWithPasswordResult{
		AccessToken:  "access_token_xxx",
		RefreshToken: "refresh_token_xxx",
		ExpiresIn:    3600,
		User:         createTestUser(),
	}
	mockUC := &mockLoginUC{result: mockResult}
	handler := newTestAuthHandler(nil, mockUC, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := LoginRequest{
		Email:      "test@example.com",
		Password:   "password123",
		RememberMe: false,
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/auth/login", reqBody)

	handler.Login(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	mockUC := &mockLoginUC{err: errors.NewUnauthorizedError("invalid credentials", "")}
	handler := newTestAuthHandler(nil, mockUC, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := LoginRequest{
		Email:    "test@example.com",
		Password: "wrong_password",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/auth/login", reqBody)

	handler.Login(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestAuthHandler_Login_InvalidRequest(t *testing.T) {
	handler := newTestAuthHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := map[string]string{"email": "test@example.com"} // missing password
	c, w := testutil.NewTestContext(http.MethodPost, "/auth/login", reqBody)

	handler.Login(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestAuthHandler_VerifyEmail
// =====================================================================

func TestAuthHandler_VerifyEmail_Success(t *testing.T) {
	mockUC := &mockVerifyEmailUC{err: nil}
	handler := newTestAuthHandler(nil, nil, mockUC, nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := VerifyEmailRequest{
		Token: "valid_verification_token_123456789012345678901234567890",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/auth/verify-email", reqBody)

	handler.VerifyEmail(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestAuthHandler_VerifyEmail_InvalidToken(t *testing.T) {
	mockUC := &mockVerifyEmailUC{err: errors.NewValidationError("invalid token", "")}
	handler := newTestAuthHandler(nil, nil, mockUC, nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := VerifyEmailRequest{
		Token: "invalid_token_12345678901234567890123456789012",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/auth/verify-email", reqBody)

	handler.VerifyEmail(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestAuthHandler_VerifyEmail_MissingToken(t *testing.T) {
	handler := newTestAuthHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := VerifyEmailRequest{} // missing token
	c, w := testutil.NewTestContext(http.MethodPost, "/auth/verify-email", reqBody)

	handler.VerifyEmail(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestAuthHandler_RequestPasswordReset
// =====================================================================

func TestAuthHandler_RequestPasswordReset_Success(t *testing.T) {
	mockUC := &mockRequestResetUC{err: nil}
	handler := newTestAuthHandler(nil, nil, nil, mockUC, nil, nil, nil, nil, nil, nil, nil)

	reqBody := ForgotPasswordRequest{
		Email: "test@example.com",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/auth/forgot-password", reqBody)

	handler.ForgotPassword(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestAuthHandler_RequestPasswordReset_InvalidEmail(t *testing.T) {
	handler := newTestAuthHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := ForgotPasswordRequest{
		Email: "invalid-email",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/auth/forgot-password", reqBody)

	handler.ForgotPassword(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestAuthHandler_ResetPassword
// =====================================================================

func TestAuthHandler_ResetPassword_Success(t *testing.T) {
	mockUC := &mockResetPasswordUC{err: nil}
	handler := newTestAuthHandler(nil, nil, nil, nil, mockUC, nil, nil, nil, nil, nil, nil)

	reqBody := ResetPasswordRequest{
		Token:    "valid_reset_token_1234567890123456789012345678901234",
		Password: "newpassword123",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/auth/reset-password", reqBody)

	handler.ResetPassword(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestAuthHandler_ResetPassword_InvalidToken(t *testing.T) {
	mockUC := &mockResetPasswordUC{err: errors.NewValidationError("invalid or expired token", "")}
	handler := newTestAuthHandler(nil, nil, nil, nil, mockUC, nil, nil, nil, nil, nil, nil)

	reqBody := ResetPasswordRequest{
		Token:    "invalid_token_123456789012345678901234567890123",
		Password: "newpassword123",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/auth/reset-password", reqBody)

	handler.ResetPassword(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestAuthHandler_RefreshToken
// =====================================================================

func TestAuthHandler_RefreshToken_Success(t *testing.T) {
	mockResult := &usecases.RefreshTokenResult{
		AccessToken:  "new_access_token_xxx",
		RefreshToken: "new_refresh_token_xxx",
		ExpiresIn:    3600,
	}
	mockUC := &mockRefreshTokenUC{result: mockResult}
	handler := newTestAuthHandler(nil, nil, nil, nil, nil, nil, nil, mockUC, nil, nil, nil)

	reqBody := RefreshTokenRequest{
		RefreshToken: "old_refresh_token_xxx",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/auth/refresh", reqBody)

	handler.RefreshToken(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestAuthHandler_RefreshToken_MissingToken(t *testing.T) {
	handler := newTestAuthHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodPost, "/auth/refresh", nil)

	handler.RefreshToken(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestAuthHandler_RefreshToken_InvalidToken(t *testing.T) {
	mockUC := &mockRefreshTokenUC{err: errors.NewUnauthorizedError("invalid refresh token", "")}
	handler := newTestAuthHandler(nil, nil, nil, nil, nil, nil, nil, mockUC, nil, nil, nil)

	reqBody := RefreshTokenRequest{
		RefreshToken: "invalid_token_xxx",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/auth/refresh", reqBody)

	handler.RefreshToken(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestAuthHandler_Logout
// =====================================================================

func TestAuthHandler_Logout_Success(t *testing.T) {
	mockUC := &mockLogoutUC{err: nil}
	handler := newTestAuthHandler(nil, nil, nil, nil, nil, nil, nil, nil, mockUC, nil, nil)

	c, w := testutil.NewTestContext(http.MethodPost, "/auth/logout", nil)
	c.Set("session_id", "test-session-id")

	handler.Logout(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestAuthHandler_Logout_NoSession(t *testing.T) {
	handler := newTestAuthHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodPost, "/auth/logout", nil)
	// No session_id set

	handler.Logout(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestAuthHandler_Logout_UseCaseError(t *testing.T) {
	mockUC := &mockLogoutUC{err: stderrors.New("session cleanup failed")}
	handler := newTestAuthHandler(nil, nil, nil, nil, nil, nil, nil, nil, mockUC, nil, nil)

	c, w := testutil.NewTestContext(http.MethodPost, "/auth/logout", nil)
	c.Set("session_id", "test-session-id")

	handler.Logout(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestAuthHandler_GetCurrentUser
// =====================================================================

func TestAuthHandler_GetCurrentUser_Success(t *testing.T) {
	testUser := createTestUser()
	mockRepo := &mockUserRepo{user: testUser}
	handler := newTestAuthHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, mockRepo, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/auth/me", nil)
	c.Set("user_id", uint(1))

	handler.GetCurrentUser(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestAuthHandler_GetCurrentUser_NotAuthenticated(t *testing.T) {
	handler := newTestAuthHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/auth/me", nil)
	// No user_id set

	handler.GetCurrentUser(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestAuthHandler_GetCurrentUser_UserNotFound(t *testing.T) {
	mockRepo := &mockUserRepo{user: nil, err: errors.NewNotFoundError("user not found", "")}
	handler := newTestAuthHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, mockRepo, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/auth/me", nil)
	c.Set("user_id", uint(999))

	handler.GetCurrentUser(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestAuthHandler_InitiateOAuthLogin
// =====================================================================

func TestAuthHandler_InitiateOAuthLogin_Success(t *testing.T) {
	mockResult := &usecases.InitiateOAuthLoginResult{
		AuthURL: "https://oauth-provider.com/authorize?state=xxx",
		State:   "state_xxx",
	}
	mockUC := &mockInitiateOAuthUC{result: mockResult}
	handler := newTestAuthHandler(nil, nil, nil, nil, nil, mockUC, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/auth/oauth/google", nil)
	c.Params = append(c.Params, gin.Param{Key: "provider", Value: "google"})

	handler.InitiateOAuth(c)

	// InitiateOAuth returns a redirect (307), not JSON
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "https://oauth-provider.com/authorize")
}

func TestAuthHandler_InitiateOAuthLogin_UnsupportedProvider(t *testing.T) {
	mockUC := &mockInitiateOAuthUC{err: errors.NewValidationError("unsupported OAuth provider", "")}
	handler := newTestAuthHandler(nil, nil, nil, nil, nil, mockUC, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/auth/oauth/unsupported", nil)
	c.Params = append(c.Params, gin.Param{Key: "provider", Value: "unsupported"})

	handler.InitiateOAuth(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"orris/internal/application/user/usecases"
	"orris/internal/domain/user"
	"orris/internal/shared/config"
	"orris/internal/shared/constants"
	"orris/internal/shared/logger"
	"orris/internal/shared/utils"
)

type AuthHandler struct {
	registerUseCase      *usecases.RegisterWithPasswordUseCase
	loginUseCase         *usecases.LoginWithPasswordUseCase
	verifyEmailUseCase   *usecases.VerifyEmailUseCase
	requestResetUseCase  *usecases.RequestPasswordResetUseCase
	resetPasswordUseCase *usecases.ResetPasswordUseCase
	initiateOAuthUseCase *usecases.InitiateOAuthLoginUseCase
	handleOAuthUseCase   *usecases.HandleOAuthCallbackUseCase
	refreshTokenUseCase  *usecases.RefreshTokenUseCase
	logoutUseCase        *usecases.LogoutUseCase
	userRepo             user.Repository
	logger               logger.Interface
	cookieConfig         config.CookieConfig
	jwtConfig            config.JWTConfig
	frontendCallbackURL  string
	allowedOrigins       []string
}

func NewAuthHandler(
	registerUC *usecases.RegisterWithPasswordUseCase,
	loginUC *usecases.LoginWithPasswordUseCase,
	verifyEmailUC *usecases.VerifyEmailUseCase,
	requestResetUC *usecases.RequestPasswordResetUseCase,
	resetPasswordUC *usecases.ResetPasswordUseCase,
	initiateOAuthUC *usecases.InitiateOAuthLoginUseCase,
	handleOAuthUC *usecases.HandleOAuthCallbackUseCase,
	refreshTokenUC *usecases.RefreshTokenUseCase,
	logoutUC *usecases.LogoutUseCase,
	userRepo user.Repository,
	logger logger.Interface,
	cookieConfig config.CookieConfig,
	jwtConfig config.JWTConfig,
	frontendCallbackURL string,
	allowedOrigins []string,
) *AuthHandler {
	return &AuthHandler{
		registerUseCase:      registerUC,
		loginUseCase:         loginUC,
		verifyEmailUseCase:   verifyEmailUC,
		requestResetUseCase:  requestResetUC,
		resetPasswordUseCase: resetPasswordUC,
		initiateOAuthUseCase: initiateOAuthUC,
		handleOAuthUseCase:   handleOAuthUC,
		refreshTokenUseCase:  refreshTokenUC,
		logoutUseCase:        logoutUC,
		userRepo:             userRepo,
		logger:               logger,
		cookieConfig:         cookieConfig,
		jwtConfig:            jwtConfig,
		frontendCallbackURL:  frontendCallbackURL,
		allowedOrigins:       allowedOrigins,
	}
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Name     string `json:"name" binding:"required,min=2,max=100"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required"`
	RememberMe bool   `json:"remember_me"`
}

type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// Register godoc
//
//	@Summary		Register a new user
//	@Description	Register a new user with email and password
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		RegisterRequest	true	"Registration request"
//	@Success		201		{object}	utils.APIResponse
//	@Failure		400		{object}	utils.APIResponse
//	@Router			/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	cmd := usecases.RegisterWithPasswordCommand{
		Email:    req.Email,
		Name:     req.Name,
		Password: req.Password,
	}

	newUser, err := h.registerUseCase.Execute(c.Request.Context(), cmd)
	if err != nil {
		h.logger.Errorw("registration failed", "error", err)
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "registration successful, please verify your email", gin.H{
		"user_id": newUser.ID(),
		"email":   newUser.Email().String(),
	})
}

// Login godoc
//
//	@Summary		Login with email and password
//	@Description	Authenticate user with email and password, sets JWT tokens in HttpOnly cookies
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		LoginRequest	true	"Login credentials"
//	@Success		200		{object}	utils.APIResponse
//	@Failure		400		{object}	utils.APIResponse
//	@Failure		401		{object}	utils.APIResponse
//	@Router			/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	cmd := usecases.LoginWithPasswordCommand{
		Email:      req.Email,
		Password:   req.Password,
		RememberMe: req.RememberMe,
		DeviceName: c.GetHeader("User-Agent"),
		DeviceType: "web",
		IPAddress:  c.ClientIP(),
		UserAgent:  c.GetHeader("User-Agent"),
	}

	result, err := h.loginUseCase.Execute(c.Request.Context(), cmd)
	if err != nil {
		h.logger.Errorw("login failed", "error", err, "email", req.Email)
		utils.ErrorResponse(c, http.StatusUnauthorized, err.Error())
		return
	}

	// Calculate cookie max age in seconds
	accessMaxAge := h.jwtConfig.AccessExpMinutes * 60
	refreshMaxAge := h.jwtConfig.RefreshExpDays * 24 * 60 * 60

	// Set tokens in HttpOnly cookies
	utils.SetAuthCookies(c, h.cookieConfig, result.AccessToken, result.RefreshToken, accessMaxAge, refreshMaxAge)

	utils.SuccessResponse(c, http.StatusOK, "login successful", gin.H{
		"user":       result.User.GetDisplayInfo(),
		"expires_in": result.ExpiresIn,
	})
}

// VerifyEmail godoc
//
//	@Summary		Verify email address
//	@Description	Verify user email with token from email
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			token	query		string				false	"Verification token (can also be in body)"
//	@Param			request	body		VerifyEmailRequest	false	"Verification request"
//	@Success		200		{object}	utils.APIResponse
//	@Failure		400		{object}	utils.APIResponse
//	@Router			/auth/verify-email [post]
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		var req VerifyEmailRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "token is required")
			return
		}
		token = req.Token
	}

	cmd := usecases.VerifyEmailCommand{Token: token}

	if err := h.verifyEmailUseCase.Execute(c.Request.Context(), cmd); err != nil {
		h.logger.Errorw("email verification failed", "error", err)
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "email verified successfully", nil)
}

// ForgotPassword godoc
//
//	@Summary		Request password reset
//	@Description	Request a password reset email
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		ForgotPasswordRequest	true	"Email address"
//	@Success		200		{object}	utils.APIResponse
//	@Failure		400		{object}	utils.APIResponse
//	@Router			/auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	cmd := usecases.RequestPasswordResetCommand{Email: req.Email}

	if err := h.requestResetUseCase.Execute(c.Request.Context(), cmd); err != nil {
		h.logger.Errorw("password reset request failed", "error", err)
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "if the email exists, a password reset link has been sent", nil)
}

// ResetPassword godoc
//
//	@Summary		Reset password
//	@Description	Reset password with token from email
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		ResetPasswordRequest	true	"Reset password request"
//	@Success		200		{object}	utils.APIResponse
//	@Failure		400		{object}	utils.APIResponse
//	@Router			/auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	cmd := usecases.ResetPasswordCommand{
		Token:       req.Token,
		NewPassword: req.Password,
	}

	if err := h.resetPasswordUseCase.Execute(c.Request.Context(), cmd); err != nil {
		h.logger.Errorw("password reset failed", "error", err)
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "password reset successfully", nil)
}

// InitiateOAuth godoc
//
//	@Summary		Initiate OAuth login
//	@Description	Redirect to OAuth provider (google or github)
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			provider	path	string	true	"OAuth provider (google or github)"
//	@Success		307			"Redirect to OAuth provider"
//	@Failure		400			{object}	utils.APIResponse
//	@Router			/auth/oauth/{provider} [get]
func (h *AuthHandler) InitiateOAuth(c *gin.Context) {
	provider := c.Param("provider")

	cmd := usecases.InitiateOAuthLoginCommand{Provider: provider}

	result, err := h.initiateOAuthUseCase.Execute(cmd)
	if err != nil {
		h.logger.Errorw("OAuth initiation failed", "error", err, "provider", provider)
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, result.AuthURL)
}

// HandleOAuthCallback godoc
//
//	@Summary		Handle OAuth callback
//	@Description	Handle OAuth provider callback and login/register user
//	@Tags			auth
//	@Accept			json
//	@Produce		html
//	@Param			provider			path		string	true	"OAuth provider (google or github)"
//	@Param			code				query		string	false	"Authorization code from OAuth provider"
//	@Param			state				query		string	false	"State parameter for CSRF protection"
//	@Param			error				query		string	false	"OAuth error code"
//	@Param			error_description	query		string	false	"OAuth error description"
//	@Success		200					{string}	string	"HTML page with postMessage script"
//	@Failure		400					{string}	string	"HTML error page"
//	@Router			/auth/oauth/{provider}/callback [get]
func (h *AuthHandler) HandleOAuthCallback(c *gin.Context) {
	provider := c.Param("provider")
	code := c.Query("code")
	state := c.Query("state")

	// Check OAuth provider errors
	if errParam := c.Query("error"); errParam != "" {
		errorDesc := c.Query("error_description")

		userMsg := constants.GetOAuthErrorMessageFromString(errParam)

		h.logger.Warnw("OAuth provider returned error",
			"provider", provider,
			"error_code", errParam,
			"error_description", errorDesc,
		)

		h.renderOAuthError(c, userMsg)
		return
	}

	// Check missing parameters
	if code == "" {
		h.logger.Warnw("OAuth callback missing code", "provider", provider)
		h.renderOAuthError(c, constants.GetOAuthErrorMessage(constants.OAuthErrorMissingCode))
		return
	}

	if state == "" {
		h.logger.Warnw("OAuth callback missing state", "provider", provider)
		h.renderOAuthError(c, constants.GetOAuthErrorMessage(constants.OAuthErrorMissingState))
		return
	}

	cmd := usecases.HandleOAuthCallbackCommand{
		Provider:   provider,
		Code:       code,
		State:      state,
		DeviceName: c.GetHeader("User-Agent"),
		DeviceType: detectDeviceType(c.GetHeader("User-Agent")),
		IPAddress:  c.ClientIP(),
		UserAgent:  c.GetHeader("User-Agent"),
	}

	result, err := h.handleOAuthUseCase.Execute(c.Request.Context(), cmd)
	if err != nil {
		h.logger.Errorw("OAuth callback failed", "error", err, "provider", provider)

		// Map error to user-friendly message
		var userMsg string
		if strings.Contains(err.Error(), "invalid or expired state") {
			userMsg = constants.GetOAuthErrorMessage(constants.OAuthErrorInvalidState)
		} else if strings.Contains(err.Error(), "exchange") {
			userMsg = constants.GetOAuthErrorMessage(constants.OAuthErrorExchangeFailed)
		} else if strings.Contains(err.Error(), "user info") {
			userMsg = constants.GetOAuthErrorMessage(constants.OAuthErrorUserInfoFailed)
		} else {
			userMsg = constants.GetOAuthErrorMessage("") // default message
		}

		h.renderOAuthError(c, userMsg)
		return
	}

	// Calculate cookie max age in seconds
	accessMaxAge := h.jwtConfig.AccessExpMinutes * 60
	refreshMaxAge := h.jwtConfig.RefreshExpDays * 24 * 60 * 60

	// Set tokens in HttpOnly cookies
	utils.SetAuthCookies(c, h.cookieConfig, result.AccessToken, result.RefreshToken, accessMaxAge, refreshMaxAge)

	h.renderOAuthSuccess(c, result)
}

// RefreshToken godoc
//
//	@Summary		Refresh access token
//	@Description	Get new access token using refresh token from HttpOnly cookie or request body
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		RefreshTokenRequest	false	"Refresh token (optional if using cookies)"
//	@Success		200		{object}	utils.APIResponse
//	@Failure		400		{object}	utils.APIResponse
//	@Failure		401		{object}	utils.APIResponse
//	@Router			/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	// Try to get refresh token from cookie first
	refreshToken := utils.GetTokenFromCookie(c, utils.RefreshTokenCookie)

	// If not in cookie, try request body (backward compatibility)
	if refreshToken == "" {
		var req RefreshTokenRequest
		if err := c.ShouldBindJSON(&req); err == nil {
			refreshToken = req.RefreshToken
		}
	}

	if refreshToken == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "refresh token is required")
		return
	}

	cmd := usecases.RefreshTokenCommand{RefreshToken: refreshToken}

	result, err := h.refreshTokenUseCase.Execute(cmd)
	if err != nil {
		h.logger.Errorw("token refresh failed", "error", err)
		utils.ErrorResponse(c, http.StatusUnauthorized, err.Error())
		return
	}

	// Calculate cookie max age in seconds
	accessMaxAge := h.jwtConfig.AccessExpMinutes * 60
	refreshMaxAge := h.jwtConfig.RefreshExpDays * 24 * 60 * 60

	// Update cookies with new tokens
	utils.SetAuthCookies(c, h.cookieConfig, result.AccessToken, refreshToken, accessMaxAge, refreshMaxAge)

	utils.SuccessResponse(c, http.StatusOK, "token refreshed successfully", gin.H{
		"expires_in": result.ExpiresIn,
	})
}

// Logout godoc
//
//	@Summary		Logout user
//	@Description	Logout current user, invalidate session and clear auth cookies
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	utils.APIResponse
//	@Failure		401	{object}	utils.APIResponse
//	@Failure		500	{object}	utils.APIResponse
//	@Router			/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	sessionID, exists := c.Get("session_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "session not found")
		return
	}

	cmd := usecases.LogoutCommand{SessionID: sessionID.(string)}

	if err := h.logoutUseCase.Execute(cmd); err != nil {
		h.logger.Errorw("logout failed", "error", err)
		utils.ErrorResponse(c, http.StatusInternalServerError, "logout failed")
		return
	}

	// Clear auth cookies
	utils.ClearAuthCookies(c, h.cookieConfig)

	utils.SuccessResponse(c, http.StatusOK, "logout successful", nil)
}

// GetCurrentUser godoc
//
//	@Summary		Get current user
//	@Description	Get current authenticated user information
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	utils.APIResponse
//	@Failure		401	{object}	utils.APIResponse
//	@Failure		404	{object}	utils.APIResponse
//	@Router			/auth/me [get]
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	currentUser, err := h.userRepo.GetByID(c.Request.Context(), userID.(uint))
	if err != nil || currentUser == nil {
		h.logger.Errorw("failed to get current user", "error", err, "user_id", userID)
		utils.ErrorResponse(c, http.StatusNotFound, "user not found")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "success", currentUser.GetDisplayInfo())
}

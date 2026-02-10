package usecases

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/go-webauthn/webauthn/protocol"

	"github.com/orris-inc/orris/internal/application/user/helpers"
	"github.com/orris-inc/orris/internal/domain/user"
	vo "github.com/orris-inc/orris/internal/domain/user/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/auth"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/shared/config"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// FinishPasskeySignupCommand represents the command to finish passkey signup
type FinishPasskeySignupCommand struct {
	SessionToken string
	Challenge    string
	Response     *protocol.ParsedCredentialCreationData
	DeviceName   string
	DeviceType   string
	IPAddress    string
	UserAgent    string
}

// FinishPasskeySignupResult represents the result of finishing passkey signup
type FinishPasskeySignupResult struct {
	User         *user.User
	Credential   *user.PasskeyCredential
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

// FinishPasskeySignupUseCase handles the completion of passkey signup ceremony
type FinishPasskeySignupUseCase struct {
	userRepo           user.Repository
	passkeyRepo        user.PasskeyCredentialRepository
	sessionRepo        user.SessionRepository
	webAuthnService    *auth.WebAuthnService
	challengeStore     *cache.PasskeyChallengeStore
	signupSessionStore *cache.PasskeySignupSessionStore
	jwtService         JWTService
	authHelper         *helpers.AuthHelper
	sessionConfig      config.SessionConfig
	logger             logger.Interface
}

// NewFinishPasskeySignupUseCase creates a new FinishPasskeySignupUseCase
func NewFinishPasskeySignupUseCase(
	userRepo user.Repository,
	passkeyRepo user.PasskeyCredentialRepository,
	sessionRepo user.SessionRepository,
	webAuthnService *auth.WebAuthnService,
	challengeStore *cache.PasskeyChallengeStore,
	signupSessionStore *cache.PasskeySignupSessionStore,
	jwtService JWTService,
	authHelper *helpers.AuthHelper,
	sessionConfig config.SessionConfig,
	logger logger.Interface,
) *FinishPasskeySignupUseCase {
	return &FinishPasskeySignupUseCase{
		userRepo:           userRepo,
		passkeyRepo:        passkeyRepo,
		sessionRepo:        sessionRepo,
		webAuthnService:    webAuthnService,
		challengeStore:     challengeStore,
		signupSessionStore: signupSessionStore,
		jwtService:         jwtService,
		authHelper:         authHelper,
		sessionConfig:      sessionConfig,
		logger:             logger,
	}
}

// Execute completes the passkey signup ceremony and creates a new user
func (uc *FinishPasskeySignupUseCase) Execute(ctx context.Context, cmd FinishPasskeySignupCommand) (*FinishPasskeySignupResult, error) {
	// Get and validate signup session (one-time use via GETDEL)
	signupSession, err := uc.signupSessionStore.Get(ctx, cmd.SessionToken)
	if err != nil {
		uc.logger.Errorw("failed to get signup session", "error", err)
		return nil, fmt.Errorf("invalid or expired signup session: %w", err)
	}

	// Get WebAuthn session data (one-time use via GETDEL)
	sessionData, err := uc.challengeStore.Get(ctx, cmd.Challenge)
	if err != nil {
		uc.logger.Errorw("failed to get passkey challenge", "error", err)
		return nil, fmt.Errorf("invalid or expired challenge: %w", err)
	}

	// Verify session data belongs to the signup session (prevent challenge hijacking)
	if !bytes.Equal(sessionData.UserID, signupSession.TempUserID) {
		uc.logger.Errorw("challenge user mismatch",
			"expected_user_id", signupSession.TempUserID,
			"session_user_id", sessionData.UserID)
		return nil, fmt.Errorf("invalid challenge")
	}

	// Double-check email is still not registered (race condition protection)
	exists, err := uc.userRepo.ExistsByEmail(ctx, signupSession.Email)
	if err != nil {
		uc.logger.Errorw("failed to check email existence", "email", signupSession.Email, "error", err)
		return nil, fmt.Errorf("failed to check email existence: %w", err)
	}
	if exists {
		uc.logger.Warnw("email registered during signup flow", "email", signupSession.Email)
		return nil, fmt.Errorf("email already registered")
	}

	// Recreate temporary WebAuthn user for verification
	tempUser := helpers.NewTempWebAuthnUser(signupSession.TempUserID, signupSession.Email, signupSession.Name)

	// Finish registration ceremony
	credential, err := uc.webAuthnService.FinishRegistration(tempUser, *sessionData, cmd.Response)
	if err != nil {
		uc.logger.Errorw("failed to finish passkey registration", "email", signupSession.Email, "error", err)
		return nil, fmt.Errorf("failed to verify passkey registration: %w", err)
	}

	// Check if credential already exists
	credExists, err := uc.passkeyRepo.ExistsByCredentialID(ctx, credential.ID)
	if err != nil {
		uc.logger.Errorw("failed to check credential existence", "error", err)
		return nil, fmt.Errorf("failed to check credential existence: %w", err)
	}
	if credExists {
		return nil, fmt.Errorf("credential already registered")
	}

	// Create value objects for user
	email, err := vo.NewEmail(signupSession.Email)
	if err != nil {
		return nil, err
	}

	name, err := vo.NewName(signupSession.Name)
	if err != nil {
		return nil, err
	}

	// Create new user (active status, no password)
	newUser, err := user.NewUser(email, name, id.NewUserID)
	if err != nil {
		uc.logger.Errorw("failed to create user entity", "email", signupSession.Email, "error", err)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Save user to database
	if err := uc.userRepo.Create(ctx, newUser); err != nil {
		uc.logger.Errorw("failed to save user", "email", signupSession.Email, "error", err)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Grant admin role to first user if applicable
	if err := uc.authHelper.GrantAdminAndSave(ctx, newUser); err != nil {
		uc.logger.Warnw("failed to grant admin role to first user", "error", err, "user_id", newUser.ID())
		// Continue despite error as user is already created
	}

	// Set device name for passkey
	deviceName := cmd.DeviceName
	if deviceName == "" {
		deviceName = "Passkey"
	}

	// Create passkey credential domain entity
	passkeyCredential, err := user.NewPasskeyCredentialFromWebAuthn(
		newUser.ID(),
		credential,
		deviceName,
		id.NewPasskeyCredentialID,
	)
	if err != nil {
		uc.logger.Errorw("failed to create passkey credential entity", "user_id", newUser.ID(), "error", err)
		return nil, fmt.Errorf("failed to create passkey credential: %w", err)
	}

	// Save passkey to database
	if err := uc.passkeyRepo.Create(ctx, passkeyCredential); err != nil {
		uc.logger.Errorw("failed to save passkey credential", "user_id", newUser.ID(), "error", err)
		return nil, fmt.Errorf("failed to save passkey credential: %w", err)
	}

	// Create session with tokens (passkey is trusted auth, use remember duration)
	sessionDuration := time.Duration(uc.sessionConfig.RememberExpDays) * 24 * time.Hour

	sessionWithTokens, err := uc.authHelper.CreateAndSaveSessionWithTokens(
		newUser.ID(),
		newUser.SID(),
		helpers.DeviceInfo{
			DeviceName: deviceName,
			DeviceType: cmd.DeviceType,
			IPAddress:  cmd.IPAddress,
			UserAgent:  cmd.UserAgent,
		},
		sessionDuration,
		true, // rememberMe: passkey is trusted auth, default to persistent cookie
		func(userUUID string, sessionID string) (string, string, int64, error) {
			tokens, err := uc.jwtService.Generate(userUUID, sessionID, newUser.Role())
			if err != nil {
				return "", "", 0, err
			}
			return tokens.AccessToken, tokens.RefreshToken, tokens.ExpiresIn, nil
		},
	)
	if err != nil {
		return nil, err
	}

	uc.logger.Infow("user registered via passkey",
		"user_id", newUser.ID(),
		"email", signupSession.Email,
		"credential_sid", passkeyCredential.SID(),
		"session_id", sessionWithTokens.Session.ID)

	return &FinishPasskeySignupResult{
		User:         newUser,
		Credential:   passkeyCredential,
		AccessToken:  sessionWithTokens.AccessToken,
		RefreshToken: sessionWithTokens.RefreshToken,
		ExpiresIn:    sessionWithTokens.ExpiresIn,
	}, nil
}

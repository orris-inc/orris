package usecases

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/orris-inc/orris/internal/application/user/helpers"
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/infrastructure/auth"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/shared/config"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// FinishPasskeyAuthenticationCommand represents the command to finish passkey authentication
type FinishPasskeyAuthenticationCommand struct {
	Challenge  string
	Response   *protocol.ParsedCredentialAssertionData
	DeviceName string
	DeviceType string
	IPAddress  string
	UserAgent  string
}

// FinishPasskeyAuthenticationResult represents the result of finishing passkey authentication
type FinishPasskeyAuthenticationResult struct {
	User         *user.User
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

// FinishPasskeyAuthenticationUseCase handles the completion of passkey authentication ceremony
type FinishPasskeyAuthenticationUseCase struct {
	userRepo        user.Repository
	passkeyRepo     user.PasskeyCredentialRepository
	sessionRepo     user.SessionRepository
	webAuthnService *auth.WebAuthnService
	challengeStore  *cache.PasskeyChallengeStore
	jwtService      JWTService
	authHelper      *helpers.AuthHelper
	sessionConfig   config.SessionConfig
	logger          logger.Interface
}

// NewFinishPasskeyAuthenticationUseCase creates a new FinishPasskeyAuthenticationUseCase
func NewFinishPasskeyAuthenticationUseCase(
	userRepo user.Repository,
	passkeyRepo user.PasskeyCredentialRepository,
	sessionRepo user.SessionRepository,
	webAuthnService *auth.WebAuthnService,
	challengeStore *cache.PasskeyChallengeStore,
	jwtService JWTService,
	authHelper *helpers.AuthHelper,
	sessionConfig config.SessionConfig,
	logger logger.Interface,
) *FinishPasskeyAuthenticationUseCase {
	return &FinishPasskeyAuthenticationUseCase{
		userRepo:        userRepo,
		passkeyRepo:     passkeyRepo,
		sessionRepo:     sessionRepo,
		webAuthnService: webAuthnService,
		challengeStore:  challengeStore,
		jwtService:      jwtService,
		authHelper:      authHelper,
		sessionConfig:   sessionConfig,
		logger:          logger,
	}
}

// Execute completes the passkey authentication ceremony
func (uc *FinishPasskeyAuthenticationUseCase) Execute(ctx context.Context, cmd FinishPasskeyAuthenticationCommand) (*FinishPasskeyAuthenticationResult, error) {
	// Get session data from challenge store
	sessionData, err := uc.challengeStore.Get(ctx, cmd.Challenge)
	if err != nil {
		uc.logger.Errorw("failed to get passkey challenge", "error", err)
		return nil, fmt.Errorf("invalid or expired challenge: %w", err)
	}

	// Handle discoverable vs non-discoverable login
	var authenticatedUser *user.User
	var matchingCredential *user.PasskeyCredential

	if len(sessionData.UserID) == 0 {
		// Discoverable login - find user by credential ID
		credential, err := uc.webAuthnService.FinishDiscoverableLogin(
			func(rawID, userHandle []byte) (webauthn.User, error) {
				// Find user by WebAuthn user handle (which is our user ID in bytes)
				userID := helpers.ParseUserIDFromBytes(userHandle)
				if userID == 0 {
					return nil, fmt.Errorf("invalid user handle")
				}

				existingUser, err := uc.userRepo.GetByID(ctx, userID)
				if err != nil {
					return nil, fmt.Errorf("failed to get user: %w", err)
				}
				if existingUser == nil {
					return nil, fmt.Errorf("user not found")
				}

				credentials, err := uc.passkeyRepo.GetByUserID(ctx, userID)
				if err != nil {
					return nil, fmt.Errorf("failed to get passkeys: %w", err)
				}

				// Find the specific credential being used
				for _, cred := range credentials {
					if bytes.Equal(cred.CredentialID(), rawID) {
						matchingCredential = cred
						break
					}
				}

				authenticatedUser = existingUser
				return helpers.NewWebAuthnUser(existingUser, credentials), nil
			},
			*sessionData,
			cmd.Response,
		)
		if err != nil {
			uc.logger.Errorw("failed to finish discoverable passkey login", "error", err)
			return nil, fmt.Errorf("failed to verify passkey: %w", err)
		}

		// Update sign count - check for credential cloning
		if matchingCredential != nil {
			if err := matchingCredential.UpdateSignCount(credential.Authenticator.SignCount); err != nil {
				uc.logger.Errorw("possible credential cloning detected", "error", err, "credential_id", matchingCredential.SID())
				return nil, fmt.Errorf("authentication failed: credential verification error")
			}
			matchingCredential.UpdateLastUsed()
			if err := uc.passkeyRepo.Update(ctx, matchingCredential); err != nil {
				uc.logger.Warnw("failed to update passkey sign count", "error", err)
				// Non-critical error, continue
			}
		} else {
			// This should not happen if webauthn validation passed
			uc.logger.Warnw("credential not found in database after successful webauthn validation",
				"user_id", authenticatedUser.ID())
		}
	} else {
		// Non-discoverable login
		userID := helpers.ParseUserIDFromBytes(sessionData.UserID)
		if userID == 0 {
			return nil, fmt.Errorf("invalid session data")
		}

		existingUser, err := uc.userRepo.GetByID(ctx, userID)
		if err != nil {
			uc.logger.Errorw("failed to get user", "user_id", userID, "error", err)
			return nil, fmt.Errorf("failed to get user: %w", err)
		}
		if existingUser == nil {
			return nil, fmt.Errorf("user not found")
		}

		credentials, err := uc.passkeyRepo.GetByUserID(ctx, userID)
		if err != nil {
			uc.logger.Errorw("failed to get user passkeys", "user_id", userID, "error", err)
			return nil, fmt.Errorf("failed to get passkeys: %w", err)
		}

		webAuthnUser := helpers.NewWebAuthnUser(existingUser, credentials)

		credential, err := uc.webAuthnService.FinishLogin(webAuthnUser, *sessionData, cmd.Response)
		if err != nil {
			uc.logger.Errorw("failed to finish passkey login", "user_id", userID, "error", err)
			return nil, fmt.Errorf("failed to verify passkey: %w", err)
		}

		// Find and update the matching credential - check for credential cloning
		for _, cred := range credentials {
			if bytes.Equal(cred.CredentialID(), credential.ID) {
				if err := cred.UpdateSignCount(credential.Authenticator.SignCount); err != nil {
					uc.logger.Errorw("possible credential cloning detected", "error", err, "credential_id", cred.SID())
					return nil, fmt.Errorf("authentication failed: credential verification error")
				}
				cred.UpdateLastUsed()
				if err := uc.passkeyRepo.Update(ctx, cred); err != nil {
					uc.logger.Warnw("failed to update passkey sign count", "error", err)
				}
				break
			}
		}

		authenticatedUser = existingUser
	}

	if authenticatedUser == nil {
		return nil, fmt.Errorf("authentication failed")
	}

	// Validate user can login
	if validationErr := uc.authHelper.ValidateUserCanLogin(authenticatedUser); validationErr != nil {
		return nil, validationErr
	}

	// Create session with tokens (passkey is trusted auth, use remember duration)
	sessionDuration := time.Duration(uc.sessionConfig.RememberExpDays) * 24 * time.Hour

	sessionWithTokens, err := uc.authHelper.CreateAndSaveSessionWithTokens(
		authenticatedUser.ID(),
		authenticatedUser.SID(),
		helpers.DeviceInfo{
			DeviceName: cmd.DeviceName,
			DeviceType: cmd.DeviceType,
			IPAddress:  cmd.IPAddress,
			UserAgent:  cmd.UserAgent,
		},
		sessionDuration,
		true, // rememberMe: passkey is trusted auth, default to persistent cookie
		func(userUUID string, sessionID string) (string, string, int64, error) {
			tokens, err := uc.jwtService.Generate(userUUID, sessionID, authenticatedUser.Role())
			if err != nil {
				return "", "", 0, err
			}
			return tokens.AccessToken, tokens.RefreshToken, tokens.ExpiresIn, nil
		},
	)
	if err != nil {
		return nil, err
	}

	uc.logger.Infow("user logged in via passkey", "user_id", authenticatedUser.ID(), "session_id", sessionWithTokens.Session.ID)

	return &FinishPasskeyAuthenticationResult{
		User:         authenticatedUser,
		AccessToken:  sessionWithTokens.AccessToken,
		RefreshToken: sessionWithTokens.RefreshToken,
		ExpiresIn:    sessionWithTokens.ExpiresIn,
	}, nil
}

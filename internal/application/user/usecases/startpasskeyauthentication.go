package usecases

import (
	"context"
	"fmt"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/orris-inc/orris/internal/application/user/helpers"
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/infrastructure/auth"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// StartPasskeyAuthenticationCommand represents the command to start passkey authentication
type StartPasskeyAuthenticationCommand struct {
	// Email is optional - if provided, it will be used for non-discoverable login
	// If empty, a discoverable (passwordless) login will be initiated
	Email string
}

// StartPasskeyAuthenticationResult represents the result of starting passkey authentication
type StartPasskeyAuthenticationResult struct {
	Options *protocol.CredentialAssertion
}

// StartPasskeyAuthenticationUseCase handles the start of passkey authentication ceremony
type StartPasskeyAuthenticationUseCase struct {
	userRepo        user.Repository
	passkeyRepo     user.PasskeyCredentialRepository
	webAuthnService *auth.WebAuthnService
	challengeStore  *cache.PasskeyChallengeStore
	logger          logger.Interface
}

// NewStartPasskeyAuthenticationUseCase creates a new StartPasskeyAuthenticationUseCase
func NewStartPasskeyAuthenticationUseCase(
	userRepo user.Repository,
	passkeyRepo user.PasskeyCredentialRepository,
	webAuthnService *auth.WebAuthnService,
	challengeStore *cache.PasskeyChallengeStore,
	logger logger.Interface,
) *StartPasskeyAuthenticationUseCase {
	return &StartPasskeyAuthenticationUseCase{
		userRepo:        userRepo,
		passkeyRepo:     passkeyRepo,
		webAuthnService: webAuthnService,
		challengeStore:  challengeStore,
		logger:          logger,
	}
}

// Execute starts the passkey authentication ceremony
func (uc *StartPasskeyAuthenticationUseCase) Execute(ctx context.Context, cmd StartPasskeyAuthenticationCommand) (*StartPasskeyAuthenticationResult, error) {
	var options *protocol.CredentialAssertion
	var sessionData *webauthn.SessionData

	if cmd.Email == "" {
		// Discoverable login (passwordless)
		opts, session, err := uc.webAuthnService.BeginDiscoverableLogin()
		if err != nil {
			uc.logger.Errorw("failed to begin discoverable passkey login", "error", err)
			return nil, fmt.Errorf("failed to begin passkey login: %w", err)
		}
		options = opts
		sessionData = session
	} else {
		// Non-discoverable login with email
		existingUser, err := uc.userRepo.GetByEmail(ctx, cmd.Email)
		if err != nil {
			uc.logger.Errorw("failed to get user by email", "email", cmd.Email, "error", err)
			return nil, fmt.Errorf("failed to get user: %w", err)
		}
		if existingUser == nil {
			// Return generic error to prevent email enumeration
			return nil, fmt.Errorf("invalid credentials")
		}

		// Get user's passkey credentials
		credentials, err := uc.passkeyRepo.GetByUserID(ctx, existingUser.ID())
		if err != nil {
			uc.logger.Errorw("failed to get user passkeys", "user_id", existingUser.ID(), "error", err)
			return nil, fmt.Errorf("failed to get passkeys: %w", err)
		}

		if len(credentials) == 0 {
			// Return generic error to prevent account enumeration
			return nil, fmt.Errorf("invalid credentials")
		}

		// Create WebAuthn user adapter
		webAuthnUser := helpers.NewWebAuthnUser(existingUser, credentials)

		opts, session, err := uc.webAuthnService.BeginLogin(webAuthnUser)
		if err != nil {
			uc.logger.Errorw("failed to begin passkey login", "user_id", existingUser.ID(), "error", err)
			return nil, fmt.Errorf("failed to begin passkey login: %w", err)
		}
		options = opts
		sessionData = session
	}

	// Store session data for later verification
	if err := uc.challengeStore.Store(ctx, sessionData); err != nil {
		uc.logger.Errorw("failed to store passkey challenge", "error", err)
		return nil, fmt.Errorf("failed to store passkey challenge: %w", err)
	}

	uc.logger.Infow("passkey authentication started", "discoverable", cmd.Email == "")

	return &StartPasskeyAuthenticationResult{
		Options: options,
	}, nil
}

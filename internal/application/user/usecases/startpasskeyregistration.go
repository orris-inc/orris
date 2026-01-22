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

// StartPasskeyRegistrationCommand represents the command to start passkey registration
type StartPasskeyRegistrationCommand struct {
	UserID uint
}

// StartPasskeyRegistrationResult represents the result of starting passkey registration
type StartPasskeyRegistrationResult struct {
	Options *protocol.CredentialCreation
}

// StartPasskeyRegistrationUseCase handles the start of passkey registration ceremony
type StartPasskeyRegistrationUseCase struct {
	userRepo        user.Repository
	passkeyRepo     user.PasskeyCredentialRepository
	webAuthnService *auth.WebAuthnService
	challengeStore  *cache.PasskeyChallengeStore
	logger          logger.Interface
}

// NewStartPasskeyRegistrationUseCase creates a new StartPasskeyRegistrationUseCase
func NewStartPasskeyRegistrationUseCase(
	userRepo user.Repository,
	passkeyRepo user.PasskeyCredentialRepository,
	webAuthnService *auth.WebAuthnService,
	challengeStore *cache.PasskeyChallengeStore,
	logger logger.Interface,
) *StartPasskeyRegistrationUseCase {
	return &StartPasskeyRegistrationUseCase{
		userRepo:        userRepo,
		passkeyRepo:     passkeyRepo,
		webAuthnService: webAuthnService,
		challengeStore:  challengeStore,
		logger:          logger,
	}
}

// Execute starts the passkey registration ceremony
func (uc *StartPasskeyRegistrationUseCase) Execute(ctx context.Context, cmd StartPasskeyRegistrationCommand) (*StartPasskeyRegistrationResult, error) {
	// Get user
	existingUser, err := uc.userRepo.GetByID(ctx, cmd.UserID)
	if err != nil {
		uc.logger.Errorw("failed to get user", "user_id", cmd.UserID, "error", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if existingUser == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Get existing credentials to exclude them
	existingCredentials, err := uc.passkeyRepo.GetByUserID(ctx, cmd.UserID)
	if err != nil {
		uc.logger.Errorw("failed to get existing passkeys", "user_id", cmd.UserID, "error", err)
		return nil, fmt.Errorf("failed to get existing passkeys: %w", err)
	}

	// Create WebAuthn user adapter
	webAuthnUser := helpers.NewWebAuthnUser(existingUser, existingCredentials)

	// Build exclude credentials list
	var excludeCredentials []protocol.CredentialDescriptor
	for _, cred := range existingCredentials {
		excludeCredentials = append(excludeCredentials, protocol.CredentialDescriptor{
			Type:         protocol.PublicKeyCredentialType,
			CredentialID: cred.CredentialID(),
		})
	}

	// Start registration ceremony
	// Note: AuthenticatorAttachment is not specified to allow both:
	// - Platform authenticators (Touch ID, Face ID, Windows Hello)
	// - Cross-platform authenticators (USB security keys, phone via QR code)
	options, sessionData, err := uc.webAuthnService.BeginRegistration(
		webAuthnUser,
		webauthn.WithExclusions(excludeCredentials),
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			UserVerification: protocol.VerificationPreferred,
			ResidentKey:      protocol.ResidentKeyRequirementPreferred,
		}),
	)
	if err != nil {
		uc.logger.Errorw("failed to begin passkey registration", "user_id", cmd.UserID, "error", err)
		return nil, fmt.Errorf("failed to begin passkey registration: %w", err)
	}

	// Store session data for later verification
	if err := uc.challengeStore.Store(ctx, sessionData); err != nil {
		uc.logger.Errorw("failed to store passkey challenge", "user_id", cmd.UserID, "error", err)
		return nil, fmt.Errorf("failed to store passkey challenge: %w", err)
	}

	uc.logger.Infow("passkey registration started", "user_id", cmd.UserID)

	return &StartPasskeyRegistrationResult{
		Options: options,
	}, nil
}

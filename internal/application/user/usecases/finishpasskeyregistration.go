package usecases

import (
	"bytes"
	"context"
	"fmt"

	"github.com/go-webauthn/webauthn/protocol"

	"github.com/orris-inc/orris/internal/application/user/helpers"
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/infrastructure/auth"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// FinishPasskeyRegistrationCommand represents the command to finish passkey registration
type FinishPasskeyRegistrationCommand struct {
	UserID     uint
	Challenge  string
	Response   *protocol.ParsedCredentialCreationData
	DeviceName string
}

// FinishPasskeyRegistrationResult represents the result of finishing passkey registration
type FinishPasskeyRegistrationResult struct {
	Credential *user.PasskeyCredential
}

// FinishPasskeyRegistrationUseCase handles the completion of passkey registration ceremony
type FinishPasskeyRegistrationUseCase struct {
	userRepo        user.Repository
	passkeyRepo     user.PasskeyCredentialRepository
	webAuthnService *auth.WebAuthnService
	challengeStore  *cache.PasskeyChallengeStore
	logger          logger.Interface
}

// NewFinishPasskeyRegistrationUseCase creates a new FinishPasskeyRegistrationUseCase
func NewFinishPasskeyRegistrationUseCase(
	userRepo user.Repository,
	passkeyRepo user.PasskeyCredentialRepository,
	webAuthnService *auth.WebAuthnService,
	challengeStore *cache.PasskeyChallengeStore,
	logger logger.Interface,
) *FinishPasskeyRegistrationUseCase {
	return &FinishPasskeyRegistrationUseCase{
		userRepo:        userRepo,
		passkeyRepo:     passkeyRepo,
		webAuthnService: webAuthnService,
		challengeStore:  challengeStore,
		logger:          logger,
	}
}

// Execute completes the passkey registration ceremony
func (uc *FinishPasskeyRegistrationUseCase) Execute(ctx context.Context, cmd FinishPasskeyRegistrationCommand) (*FinishPasskeyRegistrationResult, error) {
	// Get user
	existingUser, err := uc.userRepo.GetByID(ctx, cmd.UserID)
	if err != nil {
		uc.logger.Errorw("failed to get user", "user_id", cmd.UserID, "error", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if existingUser == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Get session data from challenge store
	sessionData, err := uc.challengeStore.Get(ctx, cmd.Challenge)
	if err != nil {
		uc.logger.Errorw("failed to get passkey challenge", "user_id", cmd.UserID, "error", err)
		return nil, fmt.Errorf("invalid or expired challenge: %w", err)
	}

	// Verify session data belongs to the requesting user (prevent challenge hijacking)
	expectedUserID := helpers.NewWebAuthnUser(existingUser, nil).WebAuthnID()
	if !bytes.Equal(sessionData.UserID, expectedUserID) {
		uc.logger.Errorw("challenge user mismatch", "user_id", cmd.UserID, "expected_user_id", expectedUserID, "session_user_id", sessionData.UserID)
		return nil, fmt.Errorf("invalid challenge")
	}

	// Get existing credentials
	existingCredentials, err := uc.passkeyRepo.GetByUserID(ctx, cmd.UserID)
	if err != nil {
		uc.logger.Errorw("failed to get existing passkeys", "user_id", cmd.UserID, "error", err)
		return nil, fmt.Errorf("failed to get existing passkeys: %w", err)
	}

	// Check passkey count limit (max 10 passkeys per user)
	const maxPasskeysPerUser = 10
	if len(existingCredentials) >= maxPasskeysPerUser {
		uc.logger.Warnw("user reached maximum passkey limit", "user_id", cmd.UserID, "count", len(existingCredentials))
		return nil, fmt.Errorf("maximum number of passkeys reached (limit: %d)", maxPasskeysPerUser)
	}

	// Create WebAuthn user adapter
	webAuthnUser := helpers.NewWebAuthnUser(existingUser, existingCredentials)

	// Finish registration ceremony
	credential, err := uc.webAuthnService.FinishRegistration(webAuthnUser, *sessionData, cmd.Response)
	if err != nil {
		uc.logger.Errorw("failed to finish passkey registration", "user_id", cmd.UserID, "error", err)
		return nil, fmt.Errorf("failed to verify passkey registration: %w", err)
	}

	// Check if credential already exists
	exists, err := uc.passkeyRepo.ExistsByCredentialID(ctx, credential.ID)
	if err != nil {
		uc.logger.Errorw("failed to check credential existence", "user_id", cmd.UserID, "error", err)
		return nil, fmt.Errorf("failed to check credential existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("credential already registered")
	}

	// Set device name
	deviceName := cmd.DeviceName
	if deviceName == "" {
		deviceName = "Passkey"
	}

	// Create passkey credential domain entity
	passkeyCredential, err := user.NewPasskeyCredentialFromWebAuthn(
		cmd.UserID,
		credential,
		deviceName,
		id.NewPasskeyCredentialID,
	)
	if err != nil {
		uc.logger.Errorw("failed to create passkey credential entity", "user_id", cmd.UserID, "error", err)
		return nil, fmt.Errorf("failed to create passkey credential: %w", err)
	}

	// Save to database
	if err := uc.passkeyRepo.Create(ctx, passkeyCredential); err != nil {
		uc.logger.Errorw("failed to save passkey credential", "user_id", cmd.UserID, "error", err)
		return nil, fmt.Errorf("failed to save passkey credential: %w", err)
	}

	uc.logger.Infow("passkey registration completed", "user_id", cmd.UserID, "credential_sid", passkeyCredential.SID())

	return &FinishPasskeyRegistrationResult{
		Credential: passkeyCredential,
	}, nil
}

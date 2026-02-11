package usecases

import (
	"context"
	"fmt"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/orris-inc/orris/internal/application/user/helpers"
	"github.com/orris-inc/orris/internal/domain/user"
	vo "github.com/orris-inc/orris/internal/domain/user/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/auth"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// StartPasskeySignupCommand represents the command to start passkey signup
type StartPasskeySignupCommand struct {
	Email string
	Name  string
}

// StartPasskeySignupResult represents the result of starting passkey signup
type StartPasskeySignupResult struct {
	Options      *protocol.CredentialCreation
	SessionToken string
}

// StartPasskeySignupUseCase handles the start of passkey signup ceremony for new users
type StartPasskeySignupUseCase struct {
	userRepo           user.Repository
	webAuthnService    *auth.WebAuthnService
	challengeStore     *cache.PasskeyChallengeStore
	signupSessionStore *cache.PasskeySignupSessionStore
	logger             logger.Interface
}

// NewStartPasskeySignupUseCase creates a new StartPasskeySignupUseCase
func NewStartPasskeySignupUseCase(
	userRepo user.Repository,
	webAuthnService *auth.WebAuthnService,
	challengeStore *cache.PasskeyChallengeStore,
	signupSessionStore *cache.PasskeySignupSessionStore,
	logger logger.Interface,
) *StartPasskeySignupUseCase {
	return &StartPasskeySignupUseCase{
		userRepo:           userRepo,
		webAuthnService:    webAuthnService,
		challengeStore:     challengeStore,
		signupSessionStore: signupSessionStore,
		logger:             logger,
	}
}

// Execute starts the passkey signup ceremony for a new user
func (uc *StartPasskeySignupUseCase) Execute(ctx context.Context, cmd StartPasskeySignupCommand) (*StartPasskeySignupResult, error) {
	// Validate email format
	email, err := vo.NewEmail(cmd.Email)
	if err != nil {
		uc.logger.Warnw("invalid email format in passkey signup", "email", cmd.Email, "error", err)
		return nil, err
	}

	// Check if email is already registered
	exists, err := uc.userRepo.ExistsByEmail(ctx, email.String())
	if err != nil {
		uc.logger.Errorw("failed to check email existence", "email", cmd.Email, "error", err)
		return nil, fmt.Errorf("failed to check email existence: %w", err)
	}
	if exists {
		// Use generic error message to prevent email enumeration attacks
		uc.logger.Infow("passkey signup attempt with existing email", "email", cmd.Email)
		return nil, fmt.Errorf("registration failed, please check your information or try logging in")
	}

	// Validate name
	name, err := vo.NewName(cmd.Name)
	if err != nil {
		uc.logger.Warnw("invalid name in passkey signup", "name", cmd.Name, "error", err)
		return nil, err
	}

	// Generate temporary user ID for WebAuthn
	tempUserID, err := helpers.GenerateTempUserID()
	if err != nil {
		uc.logger.Errorw("failed to generate temp user ID", "error", err)
		return nil, fmt.Errorf("failed to generate temp user ID: %w", err)
	}

	// Generate session token
	sessionToken, err := cache.GenerateSessionToken()
	if err != nil {
		uc.logger.Errorw("failed to generate session token", "error", err)
		return nil, fmt.Errorf("failed to generate session token: %w", err)
	}

	// Create temporary WebAuthn user
	tempUser := helpers.NewTempWebAuthnUser(tempUserID, email.String(), name.DisplayName())

	// Start registration ceremony
	options, sessionData, err := uc.webAuthnService.BeginRegistration(
		tempUser,
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			UserVerification: protocol.VerificationPreferred,
			ResidentKey:      protocol.ResidentKeyRequirementPreferred,
		}),
	)
	if err != nil {
		uc.logger.Errorw("failed to begin passkey registration", "email", cmd.Email, "error", err)
		return nil, fmt.Errorf("failed to begin passkey registration: %w", err)
	}

	// Store WebAuthn session data
	if err := uc.challengeStore.Store(ctx, sessionData); err != nil {
		uc.logger.Errorw("failed to store passkey challenge", "email", cmd.Email, "error", err)
		return nil, fmt.Errorf("failed to store passkey challenge: %w", err)
	}

	// Store signup session data
	// Note: Store DisplayName to match WebAuthn user creation above
	signupSession := &cache.PasskeySignupSession{
		SessionToken: sessionToken,
		Email:        email.String(),
		Name:         name.DisplayName(),
		TempUserID:   tempUserID,
		CreatedAt:    biztime.NowUTC().UnixMilli(),
	}
	if err := uc.signupSessionStore.Store(ctx, signupSession); err != nil {
		uc.logger.Errorw("failed to store signup session", "email", cmd.Email, "error", err)
		return nil, fmt.Errorf("failed to store signup session: %w", err)
	}

	uc.logger.Infow("passkey signup started", "email", cmd.Email)

	return &StartPasskeySignupResult{
		Options:      options,
		SessionToken: sessionToken,
	}, nil
}

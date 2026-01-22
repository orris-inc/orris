package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// DeletePasskeyCommand represents the command to delete a passkey
type DeletePasskeyCommand struct {
	UserID     uint
	PasskeySID string // pk_xxx format
}

// DeletePasskeyUseCase handles deleting a user's passkey credential
type DeletePasskeyUseCase struct {
	passkeyRepo user.PasskeyCredentialRepository
	logger      logger.Interface
}

// NewDeletePasskeyUseCase creates a new DeletePasskeyUseCase
func NewDeletePasskeyUseCase(
	passkeyRepo user.PasskeyCredentialRepository,
	logger logger.Interface,
) *DeletePasskeyUseCase {
	return &DeletePasskeyUseCase{
		passkeyRepo: passkeyRepo,
		logger:      logger,
	}
}

// Execute deletes a passkey credential
func (uc *DeletePasskeyUseCase) Execute(ctx context.Context, cmd DeletePasskeyCommand) error {
	// Get the passkey to verify ownership
	passkey, err := uc.passkeyRepo.GetBySID(ctx, cmd.PasskeySID)
	if err != nil {
		uc.logger.Errorw("failed to get passkey", "passkey_sid", cmd.PasskeySID, "error", err)
		return fmt.Errorf("failed to get passkey: %w", err)
	}

	if passkey == nil {
		return fmt.Errorf("passkey not found")
	}

	// Verify ownership
	if passkey.UserID() != cmd.UserID {
		uc.logger.Warnw("unauthorized passkey deletion attempt", "user_id", cmd.UserID, "passkey_user_id", passkey.UserID())
		return fmt.Errorf("passkey not found") // Return generic error to prevent information disclosure
	}

	// Check if this is the user's only passkey
	count, err := uc.passkeyRepo.CountByUserID(ctx, cmd.UserID)
	if err != nil {
		uc.logger.Errorw("failed to count user passkeys", "user_id", cmd.UserID, "error", err)
		return fmt.Errorf("failed to count passkeys: %w", err)
	}

	// Allow deletion even if it's the last passkey (user can still login with password or OAuth)
	// But log a warning for monitoring
	if count == 1 {
		uc.logger.Warnw("user deleting their last passkey", "user_id", cmd.UserID, "passkey_sid", cmd.PasskeySID)
	}

	// Delete the passkey
	if err := uc.passkeyRepo.DeleteBySID(ctx, cmd.PasskeySID); err != nil {
		uc.logger.Errorw("failed to delete passkey", "passkey_sid", cmd.PasskeySID, "error", err)
		return fmt.Errorf("failed to delete passkey: %w", err)
	}

	uc.logger.Infow("passkey deleted successfully", "user_id", cmd.UserID, "passkey_sid", cmd.PasskeySID)

	return nil
}

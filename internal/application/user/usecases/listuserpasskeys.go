package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ListUserPasskeysCommand represents the command to list user's passkeys
type ListUserPasskeysCommand struct {
	UserID uint
}

// ListUserPasskeysResult represents the result of listing user's passkeys
type ListUserPasskeysResult struct {
	Passkeys []user.PasskeyCredentialDisplayInfo
}

// ListUserPasskeysUseCase handles listing a user's passkey credentials
type ListUserPasskeysUseCase struct {
	passkeyRepo user.PasskeyCredentialRepository
	logger      logger.Interface
}

// NewListUserPasskeysUseCase creates a new ListUserPasskeysUseCase
func NewListUserPasskeysUseCase(
	passkeyRepo user.PasskeyCredentialRepository,
	logger logger.Interface,
) *ListUserPasskeysUseCase {
	return &ListUserPasskeysUseCase{
		passkeyRepo: passkeyRepo,
		logger:      logger,
	}
}

// Execute lists all passkey credentials for a user
func (uc *ListUserPasskeysUseCase) Execute(ctx context.Context, cmd ListUserPasskeysCommand) (*ListUserPasskeysResult, error) {
	credentials, err := uc.passkeyRepo.GetByUserID(ctx, cmd.UserID)
	if err != nil {
		uc.logger.Errorw("failed to get user passkeys", "user_id", cmd.UserID, "error", err)
		return nil, fmt.Errorf("failed to get passkeys: %w", err)
	}

	displayInfos := make([]user.PasskeyCredentialDisplayInfo, len(credentials))
	for i, cred := range credentials {
		displayInfos[i] = cred.GetDisplayInfo()
	}

	uc.logger.Debugw("listed user passkeys", "user_id", cmd.UserID, "count", len(credentials))

	return &ListUserPasskeysResult{
		Passkeys: displayInfos,
	}, nil
}

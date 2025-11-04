package helpers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"orris/internal/domain/user"
)

// AuthHelper provides common authentication helper methods
type AuthHelper struct {
	userRepo user.Repository
}

// NewAuthHelper creates a new AuthHelper instance
func NewAuthHelper(userRepo user.Repository) *AuthHelper {
	return &AuthHelper{userRepo: userRepo}
}

// IsFirstUser checks if this is the first user in the system
// First user is automatically granted admin privileges
func (h *AuthHelper) IsFirstUser(ctx context.Context) (bool, error) {
	filter := user.ListFilter{Page: 1, PageSize: 1}
	_, total, err := h.userRepo.List(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to count users: %w", err)
	}
	return total == 1, nil
}

// HashToken generates SHA256 hash of a token for secure storage
func (h *AuthHelper) HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

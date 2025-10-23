package node

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type NodeToken struct {
	tokenHash string
	expiresAt *time.Time
}

func GenerateNodeToken() (plainToken string, token *NodeToken, err error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", nil, fmt.Errorf("failed to generate random token: %w", err)
	}

	plainToken = "node_" + base64.URLEncoding.EncodeToString(tokenBytes)
	tokenHash := hashToken(plainToken)

	token = &NodeToken{
		tokenHash: tokenHash,
		expiresAt: nil,
	}

	return plainToken, token, nil
}

func GenerateNodeTokenWithExpiry(expiresAt time.Time) (plainToken string, token *NodeToken, err error) {
	plainToken, token, err = GenerateNodeToken()
	if err != nil {
		return "", nil, err
	}

	token.expiresAt = &expiresAt
	return plainToken, token, nil
}

func NewNodeToken(tokenHash string) (*NodeToken, error) {
	if tokenHash == "" {
		return nil, fmt.Errorf("token hash cannot be empty")
	}

	if len(tokenHash) != 64 {
		return nil, fmt.Errorf("invalid token hash length (expected 64 hex characters)")
	}

	if !isHexString(tokenHash) {
		return nil, fmt.Errorf("token hash must be a valid hexadecimal string")
	}

	return &NodeToken{
		tokenHash: tokenHash,
		expiresAt: nil,
	}, nil
}

func NewNodeTokenWithExpiry(tokenHash string, expiresAt time.Time) (*NodeToken, error) {
	token, err := NewNodeToken(tokenHash)
	if err != nil {
		return nil, err
	}

	token.expiresAt = &expiresAt
	return token, nil
}

func (nt *NodeToken) Hash() string {
	return nt.tokenHash
}

func (nt *NodeToken) ExpiresAt() *time.Time {
	if nt.expiresAt == nil {
		return nil
	}
	expiry := *nt.expiresAt
	return &expiry
}

func (nt *NodeToken) Verify(plainToken string) bool {
	if !strings.HasPrefix(plainToken, "node_") {
		return false
	}

	tokenHash := hashToken(plainToken)
	return subtle.ConstantTimeCompare([]byte(nt.tokenHash), []byte(tokenHash)) == 1
}

func (nt *NodeToken) IsExpired() bool {
	if nt.expiresAt == nil {
		return false
	}
	return time.Now().After(*nt.expiresAt)
}

func (nt *NodeToken) IsValid(plainToken string) bool {
	return nt.Verify(plainToken) && !nt.IsExpired()
}

func (nt *NodeToken) RemainingTime() *time.Duration {
	if nt.expiresAt == nil {
		return nil
	}

	remaining := time.Until(*nt.expiresAt)
	if remaining < 0 {
		zero := time.Duration(0)
		return &zero
	}

	return &remaining
}

func (nt *NodeToken) WithExpiry(expiresAt time.Time) *NodeToken {
	return &NodeToken{
		tokenHash: nt.tokenHash,
		expiresAt: &expiresAt,
	}
}

func (nt *NodeToken) WithoutExpiry() *NodeToken {
	return &NodeToken{
		tokenHash: nt.tokenHash,
		expiresAt: nil,
	}
}

func (nt *NodeToken) Equals(other *NodeToken) bool {
	if nt == nil || other == nil {
		return nt == other
	}

	if nt.tokenHash != other.tokenHash {
		return false
	}

	if (nt.expiresAt == nil) != (other.expiresAt == nil) {
		return false
	}

	if nt.expiresAt != nil && !nt.expiresAt.Equal(*other.expiresAt) {
		return false
	}

	return true
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func isHexString(s string) bool {
	for _, char := range s {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}

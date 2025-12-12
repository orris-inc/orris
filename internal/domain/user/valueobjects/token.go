package valueobjects

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type Token struct {
	value string
	hash  string
}

func GenerateToken() (*Token, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return nil, fmt.Errorf("failed to generate random token: %w", err)
	}

	value := hex.EncodeToString(bytes)
	hash := hashToken(value)

	return &Token{
		value: value,
		hash:  hash,
	}, nil
}

func NewTokenFromValue(value string) (*Token, error) {
	if err := validateToken(value); err != nil {
		return nil, err
	}

	hash := hashToken(value)
	return &Token{
		value: value,
		hash:  hash,
	}, nil
}

func (t *Token) Value() string {
	return t.value
}

func (t *Token) Hash() string {
	return t.hash
}

func (t *Token) Verify(plainToken string) bool {
	return hashToken(plainToken) == t.hash
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func validateToken(token string) error {
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	if len(token) < 32 {
		return fmt.Errorf("token must be at least 32 characters long")
	}

	if !isHexString(token) {
		return fmt.Errorf("token must be a valid hexadecimal string")
	}

	return nil
}

func isHexString(s string) bool {
	for _, char := range s {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}

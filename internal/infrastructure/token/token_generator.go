package token

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
)

const (
	PrefixLive = "sk_live_"
	PrefixTest = "sk_test_"
)

const tokenRandomBytes = 32

type TokenGenerator interface {
	Generate(prefix string) (plainToken string, hash string, err error)
	Hash(plainToken string) string
	Verify(plainToken, hash string) bool
}

type tokenGenerator struct{}

func NewTokenGenerator() TokenGenerator {
	return &tokenGenerator{}
}

func (g *tokenGenerator) Generate(prefix string) (string, string, error) {
	randomBytes := make([]byte, tokenRandomBytes)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	plainToken := prefix + hex.EncodeToString(randomBytes)
	tokenHash := g.Hash(plainToken)

	return plainToken, tokenHash, nil
}

func (g *tokenGenerator) Hash(plainToken string) string {
	hash := sha256.Sum256([]byte(plainToken))
	return hex.EncodeToString(hash[:])
}

func (g *tokenGenerator) Verify(plainToken, hash string) bool {
	computedHash := g.Hash(plainToken)
	return subtle.ConstantTimeCompare([]byte(computedHash), []byte(hash)) == 1
}

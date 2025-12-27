package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/orris-inc/orris/internal/shared/biztime"
)

type TokenGenerator interface {
	GenerateAPIToken(prefix string) (plainToken string, tokenHash string, err error)
	HashToken(plainToken string) string
	VerifyToken(plainToken, tokenHash string) bool
}

type DefaultTokenGenerator struct{}

func NewTokenGenerator() TokenGenerator {
	return &DefaultTokenGenerator{}
}

func (g *DefaultTokenGenerator) GenerateAPIToken(prefix string) (string, string, error) {
	tokenBytes := make([]byte, 32)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	plainToken := prefix + "_" + base64.RawURLEncoding.EncodeToString(tokenBytes)
	tokenHash := g.HashToken(plainToken)

	return plainToken, tokenHash, nil
}

func (g *DefaultTokenGenerator) HashToken(plainToken string) string {
	hash := sha256.Sum256([]byte(plainToken))
	return hex.EncodeToString(hash[:])
}

func (g *DefaultTokenGenerator) VerifyToken(plainToken, tokenHash string) bool {
	computedHash := g.HashToken(plainToken)
	return computedHash == tokenHash
}

type OrderNumberGenerator interface {
	Generate(prefix string) string
}

type DefaultOrderNumberGenerator struct{}

func NewOrderNumberGenerator() OrderNumberGenerator {
	return &DefaultOrderNumberGenerator{}
}

func (g *DefaultOrderNumberGenerator) Generate(prefix string) string {
	now := biztime.NowUTC()
	return fmt.Sprintf("%s%s%06d",
		prefix,
		now.Format("20060102150405"),
		now.Nanosecond()%1000000,
	)
}

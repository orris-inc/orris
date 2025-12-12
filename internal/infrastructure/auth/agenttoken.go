package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

const (
	// AgentTokenPrefix is the prefix for forward agent tokens.
	AgentTokenPrefix = "fwd"
)

// AgentTokenService handles generation and verification of HMAC-based agent tokens.
// Token format: fwd_<short_id>_<signature>
// Signature: base64url(HMAC-SHA256(secret, "fwd_" + short_id)[:16])
type AgentTokenService struct {
	secret []byte
}

// NewAgentTokenService creates a new AgentTokenService with the given signing secret.
func NewAgentTokenService(secret string) *AgentTokenService {
	return &AgentTokenService{
		secret: []byte(secret),
	}
}

// Generate creates a token for the given agent short ID.
// Returns the plain token and its hash (for storage).
func (s *AgentTokenService) Generate(shortID string) (plainToken string, tokenHash string) {
	signature := s.computeSignature(shortID)
	plainToken = fmt.Sprintf("%s_%s_%s", AgentTokenPrefix, shortID, signature)
	// Hash the token for storage (maintains compatibility with existing verification)
	hash := sha256.Sum256([]byte(plainToken))
	tokenHash = fmt.Sprintf("%x", hash)
	return plainToken, tokenHash
}

// Verify validates a token and returns the agent short ID if valid.
// This can be done locally without server round-trip.
func (s *AgentTokenService) Verify(token string) (shortID string, err error) {
	if token == "" {
		return "", errors.New("token cannot be empty")
	}

	// Parse token: fwd_<short_id>_<signature>
	// Use SplitN to limit splits since signature may contain '_' (base64url encoding)
	parts := strings.SplitN(token, "_", 3)
	if len(parts) != 3 {
		return "", errors.New("invalid token format")
	}

	prefix := parts[0]
	shortID = parts[1]
	providedSig := parts[2]

	if prefix != AgentTokenPrefix {
		return "", errors.New("invalid token prefix")
	}

	if shortID == "" {
		return "", errors.New("invalid short ID in token")
	}

	// Compute expected signature and compare
	expectedSig := s.computeSignature(shortID)
	if !hmac.Equal([]byte(providedSig), []byte(expectedSig)) {
		return "", errors.New("invalid token signature")
	}

	return shortID, nil
}

// computeSignature computes the HMAC signature for a given short ID.
func (s *AgentTokenService) computeSignature(shortID string) string {
	data := fmt.Sprintf("%s_%s", AgentTokenPrefix, shortID)
	h := hmac.New(sha256.New, s.secret)
	h.Write([]byte(data))
	sig := h.Sum(nil)
	// Truncate to 16 bytes and encode
	return base64.RawURLEncoding.EncodeToString(sig[:16])
}

// HashToken computes SHA256 hash of a token (for storage compatibility).
func (s *AgentTokenService) HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", hash)
}

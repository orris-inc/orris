package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// generatePKCEParams generates code_verifier and code_challenge for PKCE flow
func generatePKCEParams() (codeVerifier, codeChallenge string, err error) {
	// Generate 32 bytes of random data
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Base64 URL encode without padding
	codeVerifier = base64.RawURLEncoding.EncodeToString(verifierBytes)

	// Calculate SHA256 hash
	hash := sha256.Sum256([]byte(codeVerifier))

	// Base64 URL encode the hash
	codeChallenge = base64.RawURLEncoding.EncodeToString(hash[:])

	return codeVerifier, codeChallenge, nil
}

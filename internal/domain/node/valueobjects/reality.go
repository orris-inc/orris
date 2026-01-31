package valueobjects

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

// RealityKeyPair represents an X25519 key pair for VLESS Reality protocol
type RealityKeyPair struct {
	PrivateKey string // Base64-encoded private key
	PublicKey  string // Base64-encoded public key
}

// GenerateRealityKeyPair generates a new X25519 key pair for VLESS Reality protocol.
// The keys are base64-encoded as required by sing-box.
func GenerateRealityKeyPair() (*RealityKeyPair, error) {
	// Generate a random 32-byte private key
	var privateKey [32]byte
	if _, err := rand.Read(privateKey[:]); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Clamp the private key for X25519 (following RFC 7748)
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64

	// Derive the public key from the private key
	var publicKey [32]byte
	curve25519.ScalarBaseMult(&publicKey, &privateKey)

	return &RealityKeyPair{
		PrivateKey: base64.RawURLEncoding.EncodeToString(privateKey[:]),
		PublicKey:  base64.RawURLEncoding.EncodeToString(publicKey[:]),
	}, nil
}

// GenerateRealityShortID generates a random short ID for VLESS Reality protocol.
// The short ID is a hex-encoded string (up to 16 characters / 8 bytes).
func GenerateRealityShortID() (string, error) {
	// Generate 8 random bytes (will produce 16 hex characters)
	shortIDBytes := make([]byte, 8)
	if _, err := rand.Read(shortIDBytes); err != nil {
		return "", fmt.Errorf("failed to generate short ID: %w", err)
	}

	return hex.EncodeToString(shortIDBytes), nil
}

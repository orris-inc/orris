package id

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

const (
	// Base62 alphabet: 0-9, A-Z, a-z
	alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

	// DefaultLength is the default length for generated short IDs
	DefaultLength = 12
)

// Prefixes for different entity types (Stripe-style)
const (
	PrefixForwardAgent = "fa"
	PrefixForwardRule  = "fr"
)

// Generate creates a random short ID with the specified length using Base62 encoding.
// The generated ID is cryptographically random and URL-safe.
func Generate(length int) (string, error) {
	if length <= 0 {
		length = DefaultLength
	}

	result := make([]byte, length)
	alphabetLen := big.NewInt(int64(len(alphabet)))

	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, alphabetLen)
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %w", err)
		}
		result[i] = alphabet[num.Int64()]
	}

	return string(result), nil
}

// MustGenerate creates a random short ID and panics on error.
// Use this only when you're certain the generation won't fail.
func MustGenerate(length int) string {
	id, err := Generate(length)
	if err != nil {
		panic(err)
	}
	return id
}

// GenerateWithPrefix creates a prefixed ID in the format "prefix_randomstring".
// This follows the Stripe-style ID pattern for human-readable identifiers.
func GenerateWithPrefix(prefix string, length int) (string, error) {
	id, err := Generate(length)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s_%s", prefix, id), nil
}

// MustGenerateWithPrefix creates a prefixed ID and panics on error.
func MustGenerateWithPrefix(prefix string, length int) string {
	id, err := GenerateWithPrefix(prefix, length)
	if err != nil {
		panic(err)
	}
	return id
}

// FormatWithPrefix adds a prefix to an existing short ID.
// Example: FormatWithPrefix("fa", "xK9mP2vL3nQ") returns "fa_xK9mP2vL3nQ"
func FormatWithPrefix(prefix, shortID string) string {
	if shortID == "" {
		return ""
	}
	return fmt.Sprintf("%s_%s", prefix, shortID)
}

// ParsePrefixedID extracts the prefix and short ID from a prefixed ID string.
// Example: ParsePrefixedID("fa_xK9mP2vL3nQ") returns ("fa", "xK9mP2vL3nQ", nil)
func ParsePrefixedID(prefixedID string) (prefix, shortID string, err error) {
	parts := strings.SplitN(prefixedID, "_", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid prefixed ID format: %s", prefixedID)
	}
	return parts[0], parts[1], nil
}

// ValidatePrefix checks if the prefixed ID has the expected prefix.
func ValidatePrefix(prefixedID, expectedPrefix string) error {
	prefix, _, err := ParsePrefixedID(prefixedID)
	if err != nil {
		return err
	}
	if prefix != expectedPrefix {
		return fmt.Errorf("invalid prefix: expected %s, got %s", expectedPrefix, prefix)
	}
	return nil
}

// ExtractShortID extracts the short ID from a prefixed ID, validating the prefix.
// Example: ExtractShortID("fa_xK9mP2vL3nQ", "fa") returns "xK9mP2vL3nQ"
func ExtractShortID(prefixedID, expectedPrefix string) (string, error) {
	if err := ValidatePrefix(prefixedID, expectedPrefix); err != nil {
		return "", err
	}
	_, shortID, _ := ParsePrefixedID(prefixedID)
	return shortID, nil
}

// NewForwardAgentID generates a new Forward Agent ID.
func NewForwardAgentID() (string, error) {
	return Generate(DefaultLength)
}

// NewForwardRuleID generates a new Forward Rule ID.
func NewForwardRuleID() (string, error) {
	return Generate(DefaultLength)
}

// FormatForwardAgentID formats a short ID as a Forward Agent prefixed ID.
func FormatForwardAgentID(shortID string) string {
	return FormatWithPrefix(PrefixForwardAgent, shortID)
}

// FormatForwardRuleID formats a short ID as a Forward Rule prefixed ID.
func FormatForwardRuleID(shortID string) string {
	return FormatWithPrefix(PrefixForwardRule, shortID)
}

// ParseForwardAgentID extracts the short ID from a Forward Agent prefixed ID.
func ParseForwardAgentID(prefixedID string) (string, error) {
	return ExtractShortID(prefixedID, PrefixForwardAgent)
}

// ParseForwardRuleID extracts the short ID from a Forward Rule prefixed ID.
func ParseForwardRuleID(prefixedID string) (string, error) {
	return ExtractShortID(prefixedID, PrefixForwardRule)
}

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
	PrefixForwardAgent      = "fa"
	PrefixForwardRule       = "fr"
	PrefixNode              = "node"
	PrefixUser              = "usr"
	PrefixSubscription      = "sub"
	PrefixPlan              = "plan"
	PrefixSubscriptionToken = "stoken"
	PrefixSubscriptionUsage = "usage"
	PrefixPlanPricing       = "price"
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

// NewNodeID generates a new Node ID with prefix.
func NewNodeID() (string, error) {
	return GenerateWithPrefix(PrefixNode, DefaultLength)
}

// FormatNodeID formats a short ID as a Node prefixed ID.
func FormatNodeID(shortID string) string {
	return FormatWithPrefix(PrefixNode, shortID)
}

// ParseNodeID extracts the short ID from a Node prefixed ID.
func ParseNodeID(prefixedID string) (string, error) {
	return ExtractShortID(prefixedID, PrefixNode)
}

// NewUserID generates a new User ID (without prefix, for internal use).
func NewUserID() (string, error) {
	return Generate(DefaultLength)
}

// NewUserIDWithPrefix generates a new User ID with usr_ prefix (for storage).
func NewUserIDWithPrefix() (string, error) {
	return GenerateWithPrefix(PrefixUser, DefaultLength)
}

// FormatUserID formats a short ID as a User prefixed ID.
func FormatUserID(shortID string) string {
	return FormatWithPrefix(PrefixUser, shortID)
}

// ParseUserID extracts the short ID from a User prefixed ID.
func ParseUserID(prefixedID string) (string, error) {
	return ExtractShortID(prefixedID, PrefixUser)
}

// NewSubscriptionID generates a new Subscription ID.
func NewSubscriptionID() (string, error) {
	return Generate(DefaultLength)
}

// FormatSubscriptionID formats a short ID as a Subscription prefixed ID.
func FormatSubscriptionID(shortID string) string {
	return FormatWithPrefix(PrefixSubscription, shortID)
}

// ParseSubscriptionID extracts the short ID from a Subscription prefixed ID.
func ParseSubscriptionID(prefixedID string) (string, error) {
	return ExtractShortID(prefixedID, PrefixSubscription)
}

// NewPlanID generates a new Plan ID.
func NewPlanID() (string, error) {
	return Generate(DefaultLength)
}

// FormatPlanID formats a short ID as a Plan prefixed ID.
func FormatPlanID(shortID string) string {
	return FormatWithPrefix(PrefixPlan, shortID)
}

// ParsePlanID extracts the short ID from a Plan prefixed ID.
func ParsePlanID(prefixedID string) (string, error) {
	return ExtractShortID(prefixedID, PrefixPlan)
}

// NewSubscriptionTokenID generates a new Subscription Token ID.
func NewSubscriptionTokenID() (string, error) {
	return Generate(DefaultLength)
}

// FormatSubscriptionTokenID formats a short ID as a Subscription Token prefixed ID.
func FormatSubscriptionTokenID(shortID string) string {
	return FormatWithPrefix(PrefixSubscriptionToken, shortID)
}

// ParseSubscriptionTokenID extracts the short ID from a Subscription Token prefixed ID.
func ParseSubscriptionTokenID(prefixedID string) (string, error) {
	return ExtractShortID(prefixedID, PrefixSubscriptionToken)
}

// NewSubscriptionUsageID generates a new Subscription Usage ID.
func NewSubscriptionUsageID() (string, error) {
	return Generate(DefaultLength)
}

// FormatSubscriptionUsageID formats a short ID as a Subscription Usage prefixed ID.
func FormatSubscriptionUsageID(shortID string) string {
	return FormatWithPrefix(PrefixSubscriptionUsage, shortID)
}

// ParseSubscriptionUsageID extracts the short ID from a Subscription Usage prefixed ID.
func ParseSubscriptionUsageID(prefixedID string) (string, error) {
	return ExtractShortID(prefixedID, PrefixSubscriptionUsage)
}

// NewPlanPricingID generates a new Plan Pricing ID.
func NewPlanPricingID() (string, error) {
	return Generate(DefaultLength)
}

// FormatPlanPricingID formats a short ID as a Plan Pricing prefixed ID.
func FormatPlanPricingID(shortID string) string {
	return FormatWithPrefix(PrefixPlanPricing, shortID)
}

// ParsePlanPricingID extracts the short ID from a Plan Pricing prefixed ID.
func ParsePlanPricingID(prefixedID string) (string, error) {
	return ExtractShortID(prefixedID, PrefixPlanPricing)
}

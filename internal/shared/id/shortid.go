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

// NewSID generates a new Stripe-style ID with the given prefix.
// All SIDs are stored with prefix in database (e.g., "fa_xxx", "fr_xxx", "node_xxx").
// This is the unified ID generation method for all entities.
func NewSID(prefix string) (string, error) {
	return GenerateWithPrefix(prefix, DefaultLength)
}

// MustNewSID generates a new Stripe-style ID and panics on error.
func MustNewSID(prefix string) string {
	sid, err := NewSID(prefix)
	if err != nil {
		panic(err)
	}
	return sid
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

// NewForwardAgentID generates a new Forward Agent SID (fa_xxx).
func NewForwardAgentID() (string, error) {
	return NewSID(PrefixForwardAgent)
}

// NewForwardRuleID generates a new Forward Rule SID (fr_xxx).
func NewForwardRuleID() (string, error) {
	return NewSID(PrefixForwardRule)
}

// ParseForwardAgentID extracts the short ID from a Forward Agent prefixed ID.
func ParseForwardAgentID(prefixedID string) (string, error) {
	return ExtractShortID(prefixedID, PrefixForwardAgent)
}

// ParseForwardRuleID extracts the short ID from a Forward Rule prefixed ID.
func ParseForwardRuleID(prefixedID string) (string, error) {
	return ExtractShortID(prefixedID, PrefixForwardRule)
}

// NewNodeID generates a new Node SID (node_xxx).
func NewNodeID() (string, error) {
	return NewSID(PrefixNode)
}

// ParseNodeID extracts the short ID from a Node prefixed ID.
func ParseNodeID(prefixedID string) (string, error) {
	return ExtractShortID(prefixedID, PrefixNode)
}

// NewUserID generates a new User SID (usr_xxx).
func NewUserID() (string, error) {
	return NewSID(PrefixUser)
}

// ParseUserID extracts the short ID from a User prefixed ID.
func ParseUserID(prefixedID string) (string, error) {
	return ExtractShortID(prefixedID, PrefixUser)
}

// NewSubscriptionID generates a new Subscription SID (sub_xxx).
func NewSubscriptionID() (string, error) {
	return NewSID(PrefixSubscription)
}

// ParseSubscriptionID extracts the short ID from a Subscription prefixed ID.
func ParseSubscriptionID(prefixedID string) (string, error) {
	return ExtractShortID(prefixedID, PrefixSubscription)
}

// NewPlanID generates a new Plan SID (plan_xxx).
func NewPlanID() (string, error) {
	return NewSID(PrefixPlan)
}

// ParsePlanID extracts the short ID from a Plan prefixed ID.
func ParsePlanID(prefixedID string) (string, error) {
	return ExtractShortID(prefixedID, PrefixPlan)
}

// NewSubscriptionTokenID generates a new Subscription Token SID (stoken_xxx).
func NewSubscriptionTokenID() (string, error) {
	return NewSID(PrefixSubscriptionToken)
}

// ParseSubscriptionTokenID extracts the short ID from a Subscription Token prefixed ID.
func ParseSubscriptionTokenID(prefixedID string) (string, error) {
	return ExtractShortID(prefixedID, PrefixSubscriptionToken)
}

// NewSubscriptionUsageID generates a new Subscription Usage SID (usage_xxx).
func NewSubscriptionUsageID() (string, error) {
	return NewSID(PrefixSubscriptionUsage)
}

// ParseSubscriptionUsageID extracts the short ID from a Subscription Usage prefixed ID.
func ParseSubscriptionUsageID(prefixedID string) (string, error) {
	return ExtractShortID(prefixedID, PrefixSubscriptionUsage)
}

// NewPlanPricingID generates a new Plan Pricing SID (price_xxx).
func NewPlanPricingID() (string, error) {
	return NewSID(PrefixPlanPricing)
}

// ParsePlanPricingID extracts the short ID from a Plan Pricing prefixed ID.
func ParsePlanPricingID(prefixedID string) (string, error) {
	return ExtractShortID(prefixedID, PrefixPlanPricing)
}

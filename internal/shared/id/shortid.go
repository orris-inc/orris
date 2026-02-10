package id

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sort"
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
	PrefixForwardAgent           = "fa"
	PrefixForwardRule            = "fr"
	PrefixNode                   = "node"
	PrefixUser                   = "usr"
	PrefixSubscription           = "sub"
	PrefixPlan                   = "plan"
	PrefixSubscriptionToken      = "stoken"
	PrefixSubscriptionUsage      = "usage"
	PrefixPlanPricing            = "price"
	PrefixResourceGroup          = "rg"
	PrefixTelegramBinding        = "tg_bind"
	PrefixAdminTelegramBinding   = "atg_bind"
	PrefixSetting                = "setting"
	PrefixSubscriptionUsageStats = "usagestat"
	PrefixPasskeyCredential      = "pk"
	PrefixAnnouncement           = "ann"
)

// knownPrefixes is a list of all known prefixes sorted by length (longest first)
// to ensure correct matching for prefixes containing underscores (e.g., "tg_bind", "atg_bind").
// Sorted at init time so new prefixes only need to be added to the slice, not manually ordered.
var knownPrefixes []string

func init() {
	knownPrefixes = []string{
		PrefixAdminTelegramBinding,
		PrefixSubscriptionUsageStats,
		PrefixSubscriptionToken,
		PrefixTelegramBinding,
		PrefixSubscriptionUsage,
		PrefixPasskeyCredential,
		PrefixPlanPricing,
		PrefixResourceGroup,
		PrefixAnnouncement,
		PrefixForwardAgent,
		PrefixForwardRule,
		PrefixSubscription,
		PrefixSetting,
		PrefixNode,
		PrefixPlan,
		PrefixUser,
	}
	// Sort by length descending so longest prefixes are matched first.
	sort.Slice(knownPrefixes, func(i, j int) bool {
		return len(knownPrefixes[i]) > len(knownPrefixes[j])
	})
}

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
// It tries matching known prefixes first (longest match) to correctly handle
// multi-segment prefixes like "tg_bind" and "atg_bind", then falls back to
// splitting on the first underscore for unknown prefixes.
// Example: ParsePrefixedID("fa_xK9mP2vL3nQ") returns ("fa", "xK9mP2vL3nQ", nil)
// Example: ParsePrefixedID("tg_bind_xK9mP2vL3nQ") returns ("tg_bind", "xK9mP2vL3nQ", nil)
func ParsePrefixedID(prefixedID string) (prefix, shortID string, err error) {
	// Try matching known prefixes (longest first to handle multi-segment prefixes like "tg_bind")
	for _, p := range knownPrefixes {
		pfx := p + "_"
		if strings.HasPrefix(prefixedID, pfx) {
			shortID = prefixedID[len(pfx):]
			if len(shortID) != DefaultLength {
				return "", "", fmt.Errorf("invalid short ID length: expected %d, got %d", DefaultLength, len(shortID))
			}
			for _, c := range shortID {
				if !strings.ContainsRune(alphabet, c) {
					return "", "", fmt.Errorf("invalid character in short ID: %c", c)
				}
			}
			return p, shortID, nil
		}
	}

	// Fallback for unknown prefixes: split on first underscore
	parts := strings.SplitN(prefixedID, "_", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid prefixed ID format: %s", prefixedID)
	}

	shortID = parts[1]

	// Validate shortID length (should match DefaultLength)
	if len(shortID) != DefaultLength {
		return "", "", fmt.Errorf("invalid short ID length: expected %d, got %d", DefaultLength, len(shortID))
	}

	// Validate shortID charset (must be base62)
	for _, c := range shortID {
		if !strings.ContainsRune(alphabet, c) {
			return "", "", fmt.Errorf("invalid character in short ID: %c", c)
		}
	}

	return parts[0], shortID, nil
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

// NewResourceGroupID generates a new Resource Group SID (rg_xxx).
func NewResourceGroupID() (string, error) {
	return NewSID(PrefixResourceGroup)
}

// ParseResourceGroupID extracts the short ID from a Resource Group prefixed ID.
func ParseResourceGroupID(prefixedID string) (string, error) {
	return ExtractShortID(prefixedID, PrefixResourceGroup)
}

// NewTelegramBindingID generates a new Telegram Binding SID (tg_bind_xxx).
func NewTelegramBindingID() (string, error) {
	return NewSID(PrefixTelegramBinding)
}

// ParseTelegramBindingID extracts the short ID from a Telegram Binding prefixed ID.
func ParseTelegramBindingID(prefixedID string) (string, error) {
	return ExtractShortID(prefixedID, PrefixTelegramBinding)
}

// NewAdminTelegramBindingID generates a new Admin Telegram Binding SID (atg_bind_xxx).
func NewAdminTelegramBindingID() (string, error) {
	return NewSID(PrefixAdminTelegramBinding)
}

// ParseAdminTelegramBindingID extracts the short ID from an Admin Telegram Binding prefixed ID.
func ParseAdminTelegramBindingID(prefixedID string) (string, error) {
	return ExtractShortID(prefixedID, PrefixAdminTelegramBinding)
}

// NewSettingID generates a new Setting SID (setting_xxx).
func NewSettingID() (string, error) {
	return NewSID(PrefixSetting)
}

// ParseSettingID extracts the short ID from a Setting prefixed ID.
func ParseSettingID(prefixedID string) (string, error) {
	return ExtractShortID(prefixedID, PrefixSetting)
}

// NewSubscriptionUsageStatsID generates a new Subscription Usage Stats SID (usagestat_xxx).
func NewSubscriptionUsageStatsID() (string, error) {
	return NewSID(PrefixSubscriptionUsageStats)
}

// ParseSubscriptionUsageStatsID extracts the short ID from a Subscription Usage Stats prefixed ID.
func ParseSubscriptionUsageStatsID(prefixedID string) (string, error) {
	return ExtractShortID(prefixedID, PrefixSubscriptionUsageStats)
}

// NewPasskeyCredentialID generates a new Passkey Credential SID (pk_xxx).
func NewPasskeyCredentialID() (string, error) {
	return NewSID(PrefixPasskeyCredential)
}

// ParsePasskeyCredentialID extracts the short ID from a Passkey Credential prefixed ID.
func ParsePasskeyCredentialID(prefixedID string) (string, error) {
	return ExtractShortID(prefixedID, PrefixPasskeyCredential)
}

// NewAnnouncementID generates a new Announcement SID (ann_xxx).
func NewAnnouncementID() (string, error) {
	return NewSID(PrefixAnnouncement)
}

// ParseAnnouncementID extracts the short ID from an Announcement prefixed ID.
func ParseAnnouncementID(prefixedID string) (string, error) {
	return ExtractShortID(prefixedID, PrefixAnnouncement)
}

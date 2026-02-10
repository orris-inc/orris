package id

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// FuzzParsePrefixedID tests the ParsePrefixedID function with random inputs
func FuzzParsePrefixedID(f *testing.F) {
	// Seed corpus with valid and invalid cases
	seeds := []string{
		"fa_xK9mP2vL3nQ",
		"fr_abc123",
		"node_test",
		"usr_user123",
		"sub_subscription",
		"plan_plan123",
		"tg_bind_xK9mP2vL3n",
		"atg_bind_K9mP2vL3nQ",
		"",
		"nounderscore",
		"_leadingunderscore",
		"trailing_",
		"multiple_under_scores_here",
		"__double__underscore__",
		"a_b",
		"*_special",
		strings.Repeat("a", 1000) + "_" + strings.Repeat("b", 1000),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// Skip invalid UTF-8
		if !utf8.ValidString(input) {
			return
		}

		prefix, shortID, err := ParsePrefixedID(input)

		// If no underscore, should return error
		if !strings.Contains(input, "_") {
			if err == nil {
				t.Errorf("ParsePrefixedID(%q) should return error for input without underscore", input)
			}
			return
		}

		// If has underscore, check the parsing is correct
		if err == nil {
			// Verify the parsed values can reconstruct the original
			if !strings.HasPrefix(input, prefix+"_") {
				t.Errorf("ParsePrefixedID(%q) returned prefix=%q which doesn't match input", input, prefix)
			}
			// Verify prefix + "_" + shortID == input
			reconstructed := prefix + "_" + shortID
			if reconstructed != input {
				t.Errorf("ParsePrefixedID(%q) returned prefix=%q, shortID=%q which don't reconstruct input (got %q)", input, prefix, shortID, reconstructed)
			}
		}
	})
}

// FuzzValidatePrefix tests the ValidatePrefix function
func FuzzValidatePrefix(f *testing.F) {
	// Seed corpus
	seeds := []struct {
		prefixedID     string
		expectedPrefix string
	}{
		{"fa_test", "fa"},
		{"fa_test", "fr"},
		{"node_abc", "node"},
		{"node_abc", "usr"},
		{"", "fa"},
		{"nounderscore", "fa"},
		{"fa_", "fa"},
		{"_test", ""},
		{"tg_bind_abc123", "tg_bind"},
		{"atg_bind_abc123", "atg_bind"},
	}

	for _, seed := range seeds {
		f.Add(seed.prefixedID, seed.expectedPrefix)
	}

	f.Fuzz(func(t *testing.T, prefixedID, expectedPrefix string) {
		// Skip invalid UTF-8
		if !utf8.ValidString(prefixedID) || !utf8.ValidString(expectedPrefix) {
			return
		}

		err := ValidatePrefix(prefixedID, expectedPrefix)

		// If the ID doesn't have underscore, should error
		if !strings.Contains(prefixedID, "_") {
			if err == nil {
				t.Errorf("ValidatePrefix(%q, %q) should return error for ID without underscore", prefixedID, expectedPrefix)
			}
			return
		}

		// If parsing succeeds, verify prefix matching logic
		parsedPrefix, _, parseErr := ParsePrefixedID(prefixedID)
		if parseErr != nil {
			// If parsing fails, ValidatePrefix should also fail
			if err == nil {
				t.Errorf("ValidatePrefix(%q, %q) should return error when ParsePrefixedID fails", prefixedID, expectedPrefix)
			}
			return
		}

		// If parsed prefix matches expected, should not error
		if parsedPrefix == expectedPrefix && err != nil {
			t.Errorf("ValidatePrefix(%q, %q) returned unexpected error: %v", prefixedID, expectedPrefix, err)
		}

		// If parsed prefix doesn't match expected, should error
		if parsedPrefix != expectedPrefix && err == nil {
			t.Errorf("ValidatePrefix(%q, %q) should return error for wrong prefix (parsed: %q)", prefixedID, expectedPrefix, parsedPrefix)
		}
	})
}

// FuzzFormatWithPrefix tests the FormatWithPrefix function
func FuzzFormatWithPrefix(f *testing.F) {
	seeds := []struct {
		prefix  string
		shortID string
	}{
		{"fa", "abc123"},
		{"", "abc123"},
		{"fa", ""},
		{"", ""},
		{"node", "test_with_underscore"},
		{"*special*", "id"},
	}

	for _, seed := range seeds {
		f.Add(seed.prefix, seed.shortID)
	}

	f.Fuzz(func(t *testing.T, prefix, shortID string) {
		// Skip invalid UTF-8
		if !utf8.ValidString(prefix) || !utf8.ValidString(shortID) {
			return
		}

		result := FormatWithPrefix(prefix, shortID)

		// Empty shortID should return empty string
		if shortID == "" {
			if result != "" {
				t.Errorf("FormatWithPrefix(%q, %q) = %q, expected empty string", prefix, shortID, result)
			}
			return
		}

		// Non-empty shortID should return prefix_shortID format
		expected := prefix + "_" + shortID
		if result != expected {
			t.Errorf("FormatWithPrefix(%q, %q) = %q, expected %q", prefix, shortID, result, expected)
		}
	})
}

// FuzzGenerate tests the Generate function
func FuzzGenerate(f *testing.F) {
	// Seed with various lengths
	lengths := []int{0, 1, 2, 5, 10, 12, 20, 50, 100}
	for _, l := range lengths {
		f.Add(l)
	}

	f.Fuzz(func(t *testing.T, length int) {
		// Generate should handle any length
		result, err := Generate(length)

		// Should not return error
		if err != nil {
			t.Errorf("Generate(%d) returned error: %v", length, err)
			return
		}

		// If length <= 0, should use default length
		expectedLen := length
		if expectedLen <= 0 {
			expectedLen = DefaultLength
		}

		if len(result) != expectedLen {
			t.Errorf("Generate(%d) returned string of length %d, expected %d", length, len(result), expectedLen)
		}

		// All characters should be from the alphabet
		for _, c := range result {
			if !strings.ContainsRune(alphabet, c) {
				t.Errorf("Generate(%d) returned invalid character %q", length, c)
			}
		}
	})
}

// TestGenerateUniqueness tests that generated IDs are unique
func TestGenerateUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	iterations := 10000

	for i := 0; i < iterations; i++ {
		id, err := Generate(DefaultLength)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if seen[id] {
			t.Errorf("Generate produced duplicate ID: %s", id)
		}
		seen[id] = true
	}
}

// TestNewSIDFormats tests that NewSID produces correct formats for all prefix types,
// including multi-segment prefixes like "tg_bind" and "atg_bind".
func TestNewSIDFormats(t *testing.T) {
	tests := []struct {
		name      string
		generator func() (string, error)
		prefix    string
	}{
		{"ForwardAgent", NewForwardAgentID, PrefixForwardAgent},
		{"ForwardRule", NewForwardRuleID, PrefixForwardRule},
		{"Node", NewNodeID, PrefixNode},
		{"User", NewUserID, PrefixUser},
		{"Subscription", NewSubscriptionID, PrefixSubscription},
		{"Plan", NewPlanID, PrefixPlan},
		{"TelegramBinding", NewTelegramBindingID, PrefixTelegramBinding},
		{"AdminTelegramBinding", NewAdminTelegramBindingID, PrefixAdminTelegramBinding},
		{"Setting", NewSettingID, PrefixSetting},
		{"SubscriptionToken", NewSubscriptionTokenID, PrefixSubscriptionToken},
		{"SubscriptionUsage", NewSubscriptionUsageID, PrefixSubscriptionUsage},
		{"PlanPricing", NewPlanPricingID, PrefixPlanPricing},
		{"ResourceGroup", NewResourceGroupID, PrefixResourceGroup},
		{"SubscriptionUsageStats", NewSubscriptionUsageStatsID, PrefixSubscriptionUsageStats},
		{"PasskeyCredential", NewPasskeyCredentialID, PrefixPasskeyCredential},
		{"Announcement", NewAnnouncementID, PrefixAnnouncement},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := tt.generator()
			if err != nil {
				t.Fatalf("generator failed: %v", err)
			}

			if !strings.HasPrefix(id, tt.prefix+"_") {
				t.Errorf("generated ID %q doesn't have expected prefix %q_", id, tt.prefix)
			}

			// Verify the format can be parsed back correctly
			parsedPrefix, shortID, err := ParsePrefixedID(id)
			if err != nil {
				t.Errorf("failed to parse generated ID %q: %v", id, err)
			}
			if parsedPrefix != tt.prefix {
				t.Errorf("parsed prefix %q doesn't match expected %q", parsedPrefix, tt.prefix)
			}
			if len(shortID) != DefaultLength {
				t.Errorf("short ID length %d doesn't match default %d", len(shortID), DefaultLength)
			}

			// Verify round-trip: prefix + "_" + shortID == original id
			reconstructed := parsedPrefix + "_" + shortID
			if reconstructed != id {
				t.Errorf("round-trip failed: original=%q, reconstructed=%q", id, reconstructed)
			}
		})
	}
}

// TestParsePrefixedIDMultiSegment specifically tests multi-segment prefix parsing
func TestParsePrefixedIDMultiSegment(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedPrefix string
		expectError    bool
	}{
		{
			name:           "tg_bind prefix parses correctly",
			input:          "tg_bind_xK9mP2vL3nQw",
			expectedPrefix: "tg_bind",
		},
		{
			name:           "atg_bind prefix parses correctly",
			input:          "atg_bind_xK9mP2vL3nQw",
			expectedPrefix: "atg_bind",
		},
		{
			name:           "simple prefix still works",
			input:          "fa_xK9mP2vL3nQw",
			expectedPrefix: "fa",
		},
		{
			name:        "tg_bind with wrong length short ID",
			input:       "tg_bind_short",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, shortID, err := ParsePrefixedID(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for input %q, got prefix=%q shortID=%q", tt.input, prefix, shortID)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for input %q: %v", tt.input, err)
			}
			if prefix != tt.expectedPrefix {
				t.Errorf("prefix mismatch for %q: got %q, want %q", tt.input, prefix, tt.expectedPrefix)
			}
			if len(shortID) != DefaultLength {
				t.Errorf("shortID length mismatch for %q: got %d, want %d", tt.input, len(shortID), DefaultLength)
			}
		})
	}
}

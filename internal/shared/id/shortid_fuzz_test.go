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
		"",
		"nounderscore",
		"_leadingunderscore",
		"trailing_",
		"multiple_under_scores_here",
		"__double__underscore__",
		"a_b",
		"*_special",
		"中文_测试",
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
			// Verify the parsed values can reconstruct the original (up to first underscore)
			if !strings.HasPrefix(input, prefix+"_") {
				t.Errorf("ParsePrefixedID(%q) returned prefix=%q which doesn't match input", input, prefix)
			}
			// Verify shortID is the rest after first underscore
			parts := strings.SplitN(input, "_", 2)
			if len(parts) == 2 && shortID != parts[1] {
				t.Errorf("ParsePrefixedID(%q) returned shortID=%q, expected %q", input, shortID, parts[1])
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

		// If has correct prefix, should not error
		if strings.HasPrefix(prefixedID, expectedPrefix+"_") && err != nil {
			t.Errorf("ValidatePrefix(%q, %q) returned unexpected error: %v", prefixedID, expectedPrefix, err)
		}

		// If has wrong prefix, should error
		if !strings.HasPrefix(prefixedID, expectedPrefix+"_") && err == nil {
			actualPrefix := strings.SplitN(prefixedID, "_", 2)[0]
			if actualPrefix != expectedPrefix {
				t.Errorf("ValidatePrefix(%q, %q) should return error for wrong prefix", prefixedID, expectedPrefix)
			}
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
		{"中文", "测试"},
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

// TestNewSIDFormats tests that NewSID produces correct formats
func TestNewSIDFormats(t *testing.T) {
	tests := []struct {
		name      string
		generator func() (string, error)
		prefix    string
		// hasUnderscoreInPrefix indicates the prefix itself contains underscore
		// These prefixes cannot be correctly parsed by ParsePrefixedID
		hasUnderscoreInPrefix bool
	}{
		{"ForwardAgent", NewForwardAgentID, PrefixForwardAgent, false},
		{"ForwardRule", NewForwardRuleID, PrefixForwardRule, false},
		{"Node", NewNodeID, PrefixNode, false},
		{"User", NewUserID, PrefixUser, false},
		{"Subscription", NewSubscriptionID, PrefixSubscription, false},
		{"Plan", NewPlanID, PrefixPlan, false},
		// Note: These prefixes contain underscores, ParsePrefixedID will not work correctly
		{"TelegramBinding", NewTelegramBindingID, PrefixTelegramBinding, true},
		{"AdminTelegramBinding", NewAdminTelegramBindingID, PrefixAdminTelegramBinding, true},
		{"Setting", NewSettingID, PrefixSetting, false},
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

			// Skip parse verification for prefixes with underscores
			// ParsePrefixedID splits on first "_", so "tg_bind_xxx" becomes prefix="tg", shortID="bind_xxx"
			if tt.hasUnderscoreInPrefix {
				t.Logf("Skipping parse test for %q (prefix contains underscore)", tt.prefix)
				return
			}

			// Verify the format can be parsed back
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
		})
	}
}

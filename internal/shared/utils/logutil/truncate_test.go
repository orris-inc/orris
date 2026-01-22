package logutil

import "testing"

func TestTruncateForLog(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		// Empty string cases
		{
			name:     "empty string with positive maxLen",
			input:    "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "empty string with zero maxLen",
			input:    "",
			maxLen:   0,
			expected: "...",
		},

		// String shorter than maxLen (no truncation)
		{
			name:     "string shorter than maxLen",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},

		// String equal to maxLen (no truncation)
		{
			name:     "string equal to maxLen",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},

		// String longer than maxLen (truncation with "...")
		{
			name:     "string longer than maxLen",
			input:    "hello world",
			maxLen:   5,
			expected: "hello...",
		},
		{
			name:     "long token truncated",
			input:    "sk_live_abcdefghijklmnop",
			maxLen:   8,
			expected: "sk_live_...",
		},

		// Boundary cases for maxLen
		{
			name:     "maxLen is zero",
			input:    "hello",
			maxLen:   0,
			expected: "...",
		},
		{
			name:     "maxLen is negative",
			input:    "hello",
			maxLen:   -1,
			expected: "...",
		},
		{
			name:     "maxLen is negative large number",
			input:    "hello",
			maxLen:   -100,
			expected: "...",
		},

		// Edge case: maxLen is 1
		{
			name:     "maxLen is 1 with longer string",
			input:    "hello",
			maxLen:   1,
			expected: "h...",
		},
		{
			name:     "maxLen is 1 with single char string",
			input:    "h",
			maxLen:   1,
			expected: "h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateForLog(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("TruncateForLog(%q, %d) = %q, want %q",
					tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

package node

import (
	"testing"
)

func TestDetectFormatFromUserAgent(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		expected  string
	}{
		{
			name:      "Empty User-Agent returns base64",
			userAgent: "",
			expected:  "base64",
		},
		{
			name:      "Clash client returns clash format",
			userAgent: "Clash/1.0.0",
			expected:  "clash",
		},
		{
			name:      "Clash Meta client returns clash format",
			userAgent: "ClashMeta/1.14.0",
			expected:  "clash",
		},
		{
			name:      "Clash for Windows returns clash format",
			userAgent: "Mozilla/5.0 Clash",
			expected:  "clash",
		},
		{
			name:      "Surge client returns surge format",
			userAgent: "Surge/4.0.0",
			expected:  "surge",
		},
		{
			name:      "Surge iOS returns surge format",
			userAgent: "Surge iOS/1.0",
			expected:  "surge",
		},
		{
			name:      "Quantumult client returns base64 format",
			userAgent: "Quantumult/1.0.0",
			expected:  "base64",
		},
		{
			name:      "Quantumult X client returns base64 format",
			userAgent: "Quantumult%20X/1.0.0",
			expected:  "base64",
		},
		{
			name:      "Shadowrocket client returns base64 format",
			userAgent: "Shadowrocket/1.0.0",
			expected:  "base64",
		},
		{
			name:      "V2RayN client returns base64 format (supports all protocols)",
			userAgent: "v2rayN/1.0.0",
			expected:  "base64",
		},
		{
			name:      "V2RayNG client returns base64 format (supports all protocols)",
			userAgent: "V2RayNG/1.0.0",
			expected:  "base64",
		},
		{
			name:      "Unknown browser returns base64 format",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
			expected:  "base64",
		},
		{
			name:      "curl returns base64 format",
			userAgent: "curl/7.68.0",
			expected:  "base64",
		},
		{
			name:      "Case insensitive - CLASH returns clash format",
			userAgent: "CLASH/1.0.0",
			expected:  "clash",
		},
		{
			name:      "Case insensitive - SURGE returns surge format",
			userAgent: "SURGE/1.0.0",
			expected:  "surge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectFormatFromUserAgent(tt.userAgent)
			if result != tt.expected {
				t.Errorf("detectFormatFromUserAgent(%q) = %q, expected %q", tt.userAgent, result, tt.expected)
			}
		})
	}
}

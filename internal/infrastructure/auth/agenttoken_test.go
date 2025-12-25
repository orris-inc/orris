package auth

import (
	"testing"
)

func TestAgentTokenService_VerifyWithUnderscore(t *testing.T) {
	service := NewAgentTokenService("test-secret")

	tests := []struct {
		name    string
		shortID string
	}{
		{"simple id", "abc123"},
		{"with prefix fa_", "fa_abc123xyz"},
		{"with prefix node_", "node_xyz789"},
		{"multiple underscores", "fa_abc_123_xyz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, _ := service.Generate(tt.shortID)
			verified, err := service.Verify(token)
			if err != nil {
				t.Errorf("Verify() error = %v", err)
				return
			}
			if verified != tt.shortID {
				t.Errorf("Verify() = %v, want %v", verified, tt.shortID)
			}
		})
	}
}

func TestAgentTokenService_VerifyInvalidTokens(t *testing.T) {
	service := NewAgentTokenService("test-secret")

	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"wrong prefix", "abc_fa_xxx_signature12345678"},
		{"too short", "fwd_x"},
		{"no separator before sig", "fwd_fa_xxxsignature1234567890"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Verify(tt.token)
			if err == nil {
				t.Errorf("Verify() expected error for token %q", tt.token)
			}
		})
	}
}

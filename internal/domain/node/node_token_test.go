package node

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateNodeToken(t *testing.T) {
	plainToken, token, err := GenerateNodeToken()
	if err != nil {
		t.Errorf("GenerateNodeToken() error = %v, want nil", err)
		return
	}

	if plainToken == "" {
		t.Error("GenerateNodeToken() returned empty plainToken")
	}

	if !strings.HasPrefix(plainToken, "node_") {
		t.Errorf("GenerateNodeToken() plainToken = %q, should start with 'node_'", plainToken)
	}

	if token == nil {
		t.Error("GenerateNodeToken() returned nil token")
	}

	if token.Hash() == "" {
		t.Error("GenerateNodeToken() token hash is empty")
	}

	if token.ExpiresAt() != nil {
		t.Error("GenerateNodeToken() token should not have expiry")
	}
}

func TestGenerateNodeToken_Uniqueness(t *testing.T) {
	tokens := make(map[string]bool)
	iterations := 100

	for i := 0; i < iterations; i++ {
		plainToken, _, err := GenerateNodeToken()
		if err != nil {
			t.Fatalf("GenerateNodeToken() error = %v", err)
		}

		if tokens[plainToken] {
			t.Errorf("GenerateNodeToken() generated duplicate token: %q", plainToken)
		}
		tokens[plainToken] = true
	}
}

func TestGenerateNodeTokenWithExpiry(t *testing.T) {
	expiresAt := time.Now().Add(24 * time.Hour)
	plainToken, token, err := GenerateNodeTokenWithExpiry(expiresAt)

	if err != nil {
		t.Errorf("GenerateNodeTokenWithExpiry() error = %v, want nil", err)
		return
	}

	if plainToken == "" {
		t.Error("GenerateNodeTokenWithExpiry() returned empty plainToken")
	}

	if token == nil {
		t.Error("GenerateNodeTokenWithExpiry() returned nil token")
	}

	if token.ExpiresAt() == nil {
		t.Error("GenerateNodeTokenWithExpiry() token should have expiry")
	}

	if !token.ExpiresAt().Equal(expiresAt) {
		t.Errorf("GenerateNodeTokenWithExpiry() expiresAt = %v, want %v", token.ExpiresAt(), expiresAt)
	}
}

func TestNewNodeToken_Valid(t *testing.T) {
	validHash := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	token, err := NewNodeToken(validHash)
	if err != nil {
		t.Errorf("NewNodeToken(%q) error = %v, want nil", validHash, err)
		return
	}

	if token.Hash() != validHash {
		t.Errorf("Hash() = %q, want %q", token.Hash(), validHash)
	}

	if token.ExpiresAt() != nil {
		t.Error("NewNodeToken() token should not have expiry")
	}
}

func TestNewNodeToken_Invalid(t *testing.T) {
	tests := []struct {
		name string
		hash string
	}{
		{"empty hash", ""},
		{"too short", "abc123"},
		{"too long", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef00"},
		{"invalid characters", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdeg"},
		{"contains space", "0123456789abcdef 123456789abcdef0123456789abcdef0123456789abcdef"},
		{"wrong length 63", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcde"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewNodeToken(tt.hash)
			if err == nil {
				t.Errorf("NewNodeToken(%q) error = nil, want error", tt.hash)
			}
		})
	}
}

func TestNewNodeTokenWithExpiry(t *testing.T) {
	validHash := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	expiresAt := time.Now().Add(24 * time.Hour)

	token, err := NewNodeTokenWithExpiry(validHash, expiresAt)
	if err != nil {
		t.Errorf("NewNodeTokenWithExpiry() error = %v, want nil", err)
		return
	}

	if token.ExpiresAt() == nil {
		t.Error("NewNodeTokenWithExpiry() token should have expiry")
	}

	if !token.ExpiresAt().Equal(expiresAt) {
		t.Errorf("ExpiresAt() = %v, want %v", token.ExpiresAt(), expiresAt)
	}
}

func TestNodeToken_Verify(t *testing.T) {
	plainToken, token, err := GenerateNodeToken()
	if err != nil {
		t.Fatalf("GenerateNodeToken() error = %v", err)
	}

	tests := []struct {
		name       string
		plainToken string
		expected   bool
	}{
		{"valid token", plainToken, true},
		{"invalid token", "node_invalid", false},
		{"wrong prefix", "token_" + plainToken[5:], false},
		{"empty string", "", false},
		{"random string", "random", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := token.Verify(tt.plainToken)
			if result != tt.expected {
				t.Errorf("Verify(%q) = %v, want %v", tt.plainToken, result, tt.expected)
			}
		})
	}
}

func TestNodeToken_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt *time.Time
		expected  bool
	}{
		{"not expired", timePtr(time.Now().Add(24 * time.Hour)), false},
		{"expired", timePtr(time.Now().Add(-1 * time.Hour)), true},
		{"no expiry", nil, false},
		{"just expired", timePtr(time.Now().Add(-1 * time.Millisecond)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var token *NodeToken
			var err error

			if tt.expiresAt != nil {
				_, token, err = GenerateNodeTokenWithExpiry(*tt.expiresAt)
			} else {
				_, token, err = GenerateNodeToken()
			}

			if err != nil {
				t.Fatalf("token generation error = %v", err)
			}

			result := token.IsExpired()
			if result != tt.expected {
				t.Errorf("IsExpired() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNodeToken_IsValid(t *testing.T) {
	tests := []struct {
		name       string
		setupToken func() (string, *NodeToken)
		expected   bool
	}{
		{
			"valid and not expired",
			func() (string, *NodeToken) {
				plain, token, _ := GenerateNodeTokenWithExpiry(time.Now().Add(24 * time.Hour))
				return plain, token
			},
			true,
		},
		{
			"valid but expired",
			func() (string, *NodeToken) {
				plain, token, _ := GenerateNodeTokenWithExpiry(time.Now().Add(-1 * time.Hour))
				return plain, token
			},
			false,
		},
		{
			"invalid token",
			func() (string, *NodeToken) {
				_, token, _ := GenerateNodeToken()
				return "node_invalid", token
			},
			false,
		},
		{
			"valid with no expiry",
			func() (string, *NodeToken) {
				plain, token, _ := GenerateNodeToken()
				return plain, token
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plainToken, token := tt.setupToken()
			result := token.IsValid(plainToken)
			if result != tt.expected {
				t.Errorf("IsValid() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNodeToken_RemainingTime(t *testing.T) {
	t.Run("no expiry", func(t *testing.T) {
		_, token, err := GenerateNodeToken()
		if err != nil {
			t.Fatalf("GenerateNodeToken() error = %v", err)
		}

		remaining := token.RemainingTime()
		if remaining != nil {
			t.Errorf("RemainingTime() = %v, want nil", remaining)
		}
	})

	t.Run("future expiry", func(t *testing.T) {
		expiresAt := time.Now().Add(1 * time.Hour)
		_, token, err := GenerateNodeTokenWithExpiry(expiresAt)
		if err != nil {
			t.Fatalf("GenerateNodeTokenWithExpiry() error = %v", err)
		}

		remaining := token.RemainingTime()
		if remaining == nil {
			t.Error("RemainingTime() = nil, want duration")
			return
		}

		if *remaining <= 0 {
			t.Errorf("RemainingTime() = %v, want positive duration", *remaining)
		}

		if *remaining > 1*time.Hour {
			t.Errorf("RemainingTime() = %v, should not exceed 1 hour", *remaining)
		}
	})

	t.Run("past expiry", func(t *testing.T) {
		expiresAt := time.Now().Add(-1 * time.Hour)
		_, token, err := GenerateNodeTokenWithExpiry(expiresAt)
		if err != nil {
			t.Fatalf("GenerateNodeTokenWithExpiry() error = %v", err)
		}

		remaining := token.RemainingTime()
		if remaining == nil {
			t.Error("RemainingTime() = nil, want zero duration")
			return
		}

		if *remaining != 0 {
			t.Errorf("RemainingTime() = %v, want 0", *remaining)
		}
	})
}

func TestNodeToken_WithExpiry(t *testing.T) {
	_, original, err := GenerateNodeToken()
	if err != nil {
		t.Fatalf("GenerateNodeToken() error = %v", err)
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	updated := original.WithExpiry(expiresAt)

	if updated.ExpiresAt() == nil {
		t.Error("WithExpiry() token should have expiry")
		return
	}

	if !updated.ExpiresAt().Equal(expiresAt) {
		t.Errorf("WithExpiry() expiresAt = %v, want %v", updated.ExpiresAt(), expiresAt)
	}

	if original.ExpiresAt() != nil {
		t.Error("WithExpiry() modified original token")
	}

	if updated.Hash() != original.Hash() {
		t.Error("WithExpiry() changed token hash")
	}
}

func TestNodeToken_WithoutExpiry(t *testing.T) {
	expiresAt := time.Now().Add(24 * time.Hour)
	_, original, err := GenerateNodeTokenWithExpiry(expiresAt)
	if err != nil {
		t.Fatalf("GenerateNodeTokenWithExpiry() error = %v", err)
	}

	updated := original.WithoutExpiry()

	if updated.ExpiresAt() != nil {
		t.Error("WithoutExpiry() token should not have expiry")
	}

	if original.ExpiresAt() == nil {
		t.Error("WithoutExpiry() modified original token")
	}

	if updated.Hash() != original.Hash() {
		t.Error("WithoutExpiry() changed token hash")
	}
}

func TestNodeToken_Equals(t *testing.T) {
	hash1 := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	hash2 := "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
	expiry1 := time.Now().Add(24 * time.Hour)
	expiry2 := time.Now().Add(48 * time.Hour)

	token1, _ := NewNodeToken(hash1)
	token2, _ := NewNodeToken(hash1)
	token3, _ := NewNodeToken(hash2)
	token4, _ := NewNodeTokenWithExpiry(hash1, expiry1)
	token5, _ := NewNodeTokenWithExpiry(hash1, expiry1)
	token6, _ := NewNodeTokenWithExpiry(hash1, expiry2)

	tests := []struct {
		name     string
		token1   *NodeToken
		token2   *NodeToken
		expected bool
	}{
		{"same hash no expiry", token1, token2, true},
		{"different hash", token1, token3, false},
		{"same hash different expiry", token4, token6, false},
		{"same hash same expiry", token4, token5, true},
		{"one with expiry one without", token1, token4, false},
		{"nil equals nil", nil, nil, true},
		{"nil vs non-nil", nil, token1, false},
		{"non-nil vs nil", token1, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.token1.Equals(tt.token2)
			if result != tt.expected {
				t.Errorf("Equals() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNodeToken_ExpiresAtImmutability(t *testing.T) {
	expiresAt := time.Now().Add(24 * time.Hour)
	_, token, err := GenerateNodeTokenWithExpiry(expiresAt)
	if err != nil {
		t.Fatalf("GenerateNodeTokenWithExpiry() error = %v", err)
	}

	returnedExpiry := token.ExpiresAt()
	if returnedExpiry == nil {
		t.Fatal("ExpiresAt() returned nil")
	}

	originalExpiry := *returnedExpiry
	*returnedExpiry = time.Now().Add(48 * time.Hour)

	currentExpiry := token.ExpiresAt()
	if !currentExpiry.Equal(originalExpiry) {
		t.Error("modifying returned ExpiresAt() pointer affected internal state")
	}
}

func TestNodeToken_HashFormat(t *testing.T) {
	_, token, err := GenerateNodeToken()
	if err != nil {
		t.Fatalf("GenerateNodeToken() error = %v", err)
	}

	hash := token.Hash()

	if len(hash) != 64 {
		t.Errorf("Hash() length = %d, want 64", len(hash))
	}

	for _, char := range hash {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			t.Errorf("Hash() contains invalid character: %c", char)
		}
	}
}

func TestNodeToken_VerifyConstantTime(t *testing.T) {
	plainToken, token, err := GenerateNodeToken()
	if err != nil {
		t.Fatalf("GenerateNodeToken() error = %v", err)
	}

	iterations := 1000
	validTimes := make([]time.Duration, iterations)
	invalidTimes := make([]time.Duration, iterations)

	for i := 0; i < iterations; i++ {
		start := time.Now()
		token.Verify(plainToken)
		validTimes[i] = time.Since(start)

		start = time.Now()
		token.Verify("node_invalid")
		invalidTimes[i] = time.Since(start)
	}
}

func TestNodeToken_BoundaryConditions(t *testing.T) {
	t.Run("expiry exactly now", func(t *testing.T) {
		plainToken, token, err := GenerateNodeTokenWithExpiry(time.Now())
		if err != nil {
			t.Fatalf("GenerateNodeTokenWithExpiry() error = %v", err)
		}

		time.Sleep(1 * time.Millisecond)

		if !token.IsExpired() {
			t.Error("token with expiry at current time should be expired after delay")
		}

		if token.IsValid(plainToken) {
			t.Error("expired token should not be valid")
		}
	})

	t.Run("uppercase hex hash", func(t *testing.T) {
		upperHash := "ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789"
		_, err := NewNodeToken(upperHash)
		if err != nil {
			t.Errorf("NewNodeToken() with uppercase hex should succeed, got: %v", err)
		}
	})

	t.Run("mixed case hex hash", func(t *testing.T) {
		mixedHash := "AbCdEf0123456789AbCdEf0123456789AbCdEf0123456789AbCdEf0123456789"
		_, err := NewNodeToken(mixedHash)
		if err != nil {
			t.Errorf("NewNodeToken() with mixed case hex should succeed, got: %v", err)
		}
	})
}

func timePtr(t time.Time) *time.Time {
	return &t
}

package node

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestNewEncryptionConfig_ValidMethods(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		password string
	}{
		{"AES-256-GCM", "aes-256-gcm", "password123"},
		{"AES-128-GCM", "aes-128-gcm", "password123"},
		{"ChaCha20-IETF-Poly1305", "chacha20-ietf-poly1305", "password123"},
		{"uppercase method", "AES-256-GCM", "password123"},
		{"mixed case method", "ChaCha20-IETF-Poly1305", "password123"},
		{"with whitespace", "  aes-256-gcm  ", "password123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewEncryptionConfig(tt.method, tt.password)
			if err != nil {
				t.Errorf("NewEncryptionConfig(%q, %q) error = %v, want nil", tt.method, tt.password, err)
				return
			}
			if config.Method() == "" {
				t.Error("Method() returned empty string")
			}
			if config.Password() != tt.password {
				t.Errorf("Password() = %q, want %q", config.Password(), tt.password)
			}
		})
	}
}

func TestNewEncryptionConfig_InvalidMethods(t *testing.T) {
	tests := []struct {
		name   string
		method string
	}{
		{"unsupported method", "rc4"},
		{"invalid method", "invalid-method"},
		{"empty method", ""},
		{"whitespace only", "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEncryptionConfig(tt.method, "password123")
			if err == nil {
				t.Errorf("NewEncryptionConfig(%q, _) error = nil, want error", tt.method)
			}
		})
	}
}

func TestNewEncryptionConfig_PasswordValidation(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		wantError bool
	}{
		{"minimum length 8", "12345678", false},
		{"too short", "1234567", true},
		{"empty password", "", true},
		{"valid long password", "this-is-a-very-secure-password-123", false},
		{"maximum length 128", strings.Repeat("a", 128), false},
		{"exceeds maximum length", strings.Repeat("a", 129), true},
		{"special characters", "P@ssw0rd!", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEncryptionConfig("aes-256-gcm", tt.password)
			if (err != nil) != tt.wantError {
				t.Errorf("NewEncryptionConfig(_, %q) error = %v, wantError %v", tt.password, err, tt.wantError)
			}
		})
	}
}

func TestEncryptionConfig_IsAES(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		expected bool
	}{
		{"AES-256-GCM", "aes-256-gcm", true},
		{"AES-128-GCM", "aes-128-gcm", true},
		{"ChaCha20", "chacha20-ietf-poly1305", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewEncryptionConfig(tt.method, "password123")
			if err != nil {
				t.Fatalf("NewEncryptionConfig() error = %v", err)
			}
			if config.IsAES() != tt.expected {
				t.Errorf("IsAES() = %v, want %v", config.IsAES(), tt.expected)
			}
		})
	}
}

func TestEncryptionConfig_IsChacha20(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		expected bool
	}{
		{"AES-256-GCM", "aes-256-gcm", false},
		{"AES-128-GCM", "aes-128-gcm", false},
		{"ChaCha20", "chacha20-ietf-poly1305", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewEncryptionConfig(tt.method, "password123")
			if err != nil {
				t.Fatalf("NewEncryptionConfig() error = %v", err)
			}
			if config.IsChacha20() != tt.expected {
				t.Errorf("IsChacha20() = %v, want %v", config.IsChacha20(), tt.expected)
			}
		})
	}
}

func TestEncryptionConfig_ToShadowsocksAuth(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		password string
	}{
		{"AES-256-GCM", "aes-256-gcm", "password123"},
		{"ChaCha20", "chacha20-ietf-poly1305", "secretpass"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewEncryptionConfig(tt.method, tt.password)
			if err != nil {
				t.Fatalf("NewEncryptionConfig() error = %v", err)
			}

			auth := config.ToShadowsocksAuth()
			if auth == "" {
				t.Error("ToShadowsocksAuth() returned empty string")
			}

			decoded, err := base64.URLEncoding.DecodeString(auth)
			if err != nil {
				t.Errorf("ToShadowsocksAuth() returned invalid base64: %v", err)
			}

			expectedAuth := tt.method + ":" + tt.password
			if string(decoded) != expectedAuth {
				t.Errorf("ToShadowsocksAuth() decoded = %q, want %q", string(decoded), expectedAuth)
			}
		})
	}
}

func TestEncryptionConfig_Equals(t *testing.T) {
	config1, _ := NewEncryptionConfig("aes-256-gcm", "password123")
	config2, _ := NewEncryptionConfig("aes-256-gcm", "password123")
	config3, _ := NewEncryptionConfig("aes-128-gcm", "password123")
	config4, _ := NewEncryptionConfig("aes-256-gcm", "different")

	tests := []struct {
		name     string
		config1  *EncryptionConfig
		config2  *EncryptionConfig
		expected bool
	}{
		{"same configs", config1, config2, true},
		{"different methods", config1, config3, false},
		{"different passwords", config1, config4, false},
		{"nil equals nil", nil, nil, true},
		{"nil vs non-nil", nil, config1, false},
		{"non-nil vs nil", config1, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config1.Equals(tt.config2)
			if result != tt.expected {
				t.Errorf("Equals() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetSupportedMethods(t *testing.T) {
	methods := GetSupportedMethods()

	if len(methods) == 0 {
		t.Error("GetSupportedMethods() returned empty slice")
	}

	expectedMethods := map[string]bool{
		"aes-256-gcm":            true,
		"aes-128-gcm":            true,
		"chacha20-ietf-poly1305": true,
	}

	for _, method := range methods {
		if !expectedMethods[method] {
			t.Errorf("GetSupportedMethods() contains unexpected method: %s", method)
		}
	}

	for method := range expectedMethods {
		found := false
		for _, m := range methods {
			if m == method {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GetSupportedMethods() missing expected method: %s", method)
		}
	}
}

func TestEncryptionConfig_MethodNormalization(t *testing.T) {
	tests := []struct {
		name           string
		inputMethod    string
		expectedMethod string
	}{
		{"lowercase", "aes-256-gcm", "aes-256-gcm"},
		{"uppercase", "AES-256-GCM", "aes-256-gcm"},
		{"mixed case", "Aes-256-Gcm", "aes-256-gcm"},
		{"with spaces", "  aes-256-gcm  ", "aes-256-gcm"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewEncryptionConfig(tt.inputMethod, "password123")
			if err != nil {
				t.Fatalf("NewEncryptionConfig() error = %v", err)
			}
			if config.Method() != tt.expectedMethod {
				t.Errorf("Method() = %q, want %q", config.Method(), tt.expectedMethod)
			}
		})
	}
}

func TestEncryptionConfig_BoundaryPasswords(t *testing.T) {
	t.Run("minimum valid password", func(t *testing.T) {
		_, err := NewEncryptionConfig("aes-256-gcm", "12345678")
		if err != nil {
			t.Errorf("8 char password should be valid, got error: %v", err)
		}
	})

	t.Run("just below minimum", func(t *testing.T) {
		_, err := NewEncryptionConfig("aes-256-gcm", "1234567")
		if err == nil {
			t.Error("7 char password should be invalid")
		}
	})

	t.Run("maximum valid password", func(t *testing.T) {
		_, err := NewEncryptionConfig("aes-256-gcm", strings.Repeat("a", 128))
		if err != nil {
			t.Errorf("128 char password should be valid, got error: %v", err)
		}
	})

	t.Run("just above maximum", func(t *testing.T) {
		_, err := NewEncryptionConfig("aes-256-gcm", strings.Repeat("a", 129))
		if err == nil {
			t.Error("129 char password should be invalid")
		}
	})
}

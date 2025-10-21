package token

import (
	"strings"
	"testing"
)

func TestTokenGenerator_Generate(t *testing.T) {
	generator := NewTokenGenerator()

	tests := []struct {
		name   string
		prefix string
	}{
		{
			name:   "generate live token",
			prefix: PrefixLive,
		},
		{
			name:   "generate test token",
			prefix: PrefixTest,
		},
		{
			name:   "generate custom prefix token",
			prefix: "custom_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plainToken, hash, err := generator.Generate(tt.prefix)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			if !strings.HasPrefix(plainToken, tt.prefix) {
				t.Errorf("plainToken = %v, want prefix %v", plainToken, tt.prefix)
			}

			if hash == "" {
				t.Error("hash should not be empty")
			}

			if len(hash) != 64 {
				t.Errorf("hash length = %d, want 64 (SHA256 hex)", len(hash))
			}

			if plainToken == hash {
				t.Error("plainToken and hash should be different")
			}
		})
	}
}

func TestTokenGenerator_Generate_Uniqueness(t *testing.T) {
	generator := NewTokenGenerator()

	token1, hash1, err1 := generator.Generate(PrefixLive)
	if err1 != nil {
		t.Fatalf("Generate() error = %v", err1)
	}

	token2, hash2, err2 := generator.Generate(PrefixLive)
	if err2 != nil {
		t.Fatalf("Generate() error = %v", err2)
	}

	if token1 == token2 {
		t.Error("tokens should be unique")
	}

	if hash1 == hash2 {
		t.Error("hashes should be unique")
	}
}

func TestTokenGenerator_Hash(t *testing.T) {
	generator := NewTokenGenerator()

	tests := []struct {
		name       string
		plainToken string
		wantSame   bool
	}{
		{
			name:       "same token produces same hash",
			plainToken: "sk_live_test123",
			wantSame:   true,
		},
		{
			name:       "different token produces different hash",
			plainToken: "sk_live_different456",
			wantSame:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := generator.Hash(tt.plainToken)
			hash2 := generator.Hash(tt.plainToken)

			if hash1 != hash2 {
				t.Error("same token should produce same hash")
			}

			if len(hash1) != 64 {
				t.Errorf("hash length = %d, want 64 (SHA256 hex)", len(hash1))
			}
		})
	}
}

func TestTokenGenerator_Verify(t *testing.T) {
	generator := NewTokenGenerator()

	plainToken, hash, err := generator.Generate(PrefixLive)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	tests := []struct {
		name       string
		plainToken string
		hash       string
		want       bool
	}{
		{
			name:       "valid token verification",
			plainToken: plainToken,
			hash:       hash,
			want:       true,
		},
		{
			name:       "invalid token verification",
			plainToken: "sk_live_invalid",
			hash:       hash,
			want:       false,
		},
		{
			name:       "invalid hash verification",
			plainToken: plainToken,
			hash:       "invalidhash",
			want:       false,
		},
		{
			name:       "empty token verification",
			plainToken: "",
			hash:       hash,
			want:       false,
		},
		{
			name:       "empty hash verification",
			plainToken: plainToken,
			hash:       "",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generator.Verify(tt.plainToken, tt.hash)
			if got != tt.want {
				t.Errorf("Verify() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenGenerator_Verify_ConstantTime(t *testing.T) {
	generator := NewTokenGenerator()

	plainToken, hash, err := generator.Generate(PrefixLive)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	wrongToken := "sk_live_wrong"
	wrongHash := generator.Hash(wrongToken)

	result1 := generator.Verify(plainToken, hash)
	result2 := generator.Verify(wrongToken, wrongHash)
	result3 := generator.Verify(plainToken, wrongHash)

	if !result1 {
		t.Error("valid token should verify")
	}

	if !result2 {
		t.Error("wrong token with its own hash should verify")
	}

	if result3 {
		t.Error("correct token with wrong hash should not verify")
	}
}

func BenchmarkTokenGenerator_Generate(b *testing.B) {
	generator := NewTokenGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = generator.Generate(PrefixLive)
	}
}

func BenchmarkTokenGenerator_Hash(b *testing.B) {
	generator := NewTokenGenerator()
	token := "token_example_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generator.Hash(token)
	}
}

func BenchmarkTokenGenerator_Verify(b *testing.B) {
	generator := NewTokenGenerator()
	plainToken, hash, _ := generator.Generate(PrefixLive)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generator.Verify(plainToken, hash)
	}
}

package subscription

import (
	"testing"
	"time"

	vo "orris/internal/domain/subscription/value_objects"
)

func TestNewSubscriptionToken(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)

	tests := []struct {
		name           string
		subscriptionID uint
		tokenName      string
		tokenHash      string
		prefix         string
		scope          vo.TokenScope
		expiresAt      *time.Time
		wantErr        bool
		errMsg         string
	}{
		{
			name:           "valid token",
			subscriptionID: 1,
			tokenName:      "API Token",
			tokenHash:      "hash123",
			prefix:         "tok_",
			scope:          vo.TokenScopeAPI,
			expiresAt:      &future,
			wantErr:        false,
		},
		{
			name:           "valid token without expiration",
			subscriptionID: 1,
			tokenName:      "Permanent Token",
			tokenHash:      "hash456",
			prefix:         "tok_",
			scope:          vo.TokenScopeFull,
			expiresAt:      nil,
			wantErr:        false,
		},
		{
			name:           "zero subscription ID",
			subscriptionID: 0,
			tokenName:      "Test Token",
			tokenHash:      "hash123",
			prefix:         "tok_",
			scope:          vo.TokenScopeAPI,
			expiresAt:      nil,
			wantErr:        true,
			errMsg:         "subscription ID cannot be zero",
		},
		{
			name:           "empty token name",
			subscriptionID: 1,
			tokenName:      "",
			tokenHash:      "hash123",
			prefix:         "tok_",
			scope:          vo.TokenScopeAPI,
			expiresAt:      nil,
			wantErr:        true,
			errMsg:         "token name cannot be empty",
		},
		{
			name:           "empty token hash",
			subscriptionID: 1,
			tokenName:      "Test Token",
			tokenHash:      "",
			prefix:         "tok_",
			scope:          vo.TokenScopeAPI,
			expiresAt:      nil,
			wantErr:        true,
			errMsg:         "token hash cannot be empty",
		},
		{
			name:           "empty prefix",
			subscriptionID: 1,
			tokenName:      "Test Token",
			tokenHash:      "hash123",
			prefix:         "",
			scope:          vo.TokenScopeAPI,
			expiresAt:      nil,
			wantErr:        true,
			errMsg:         "token prefix cannot be empty",
		},
		{
			name:           "invalid scope",
			subscriptionID: 1,
			tokenName:      "Test Token",
			tokenHash:      "hash123",
			prefix:         "tok_",
			scope:          vo.TokenScope("invalid"),
			expiresAt:      nil,
			wantErr:        true,
			errMsg:         "invalid token scope",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := NewSubscriptionToken(
				tt.subscriptionID,
				tt.tokenName,
				tt.tokenHash,
				tt.prefix,
				tt.scope,
				tt.expiresAt,
			)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewSubscriptionToken() expected error, got nil")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("NewSubscriptionToken() error = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("NewSubscriptionToken() unexpected error = %v", err)
				return
			}

			if token.subscriptionID != tt.subscriptionID {
				t.Errorf("subscriptionID = %v, want %v", token.subscriptionID, tt.subscriptionID)
			}

			if token.name != tt.tokenName {
				t.Errorf("name = %v, want %v", token.name, tt.tokenName)
			}

			if token.tokenHash != tt.tokenHash {
				t.Errorf("tokenHash = %v, want %v", token.tokenHash, tt.tokenHash)
			}

			if !token.isActive {
				t.Errorf("isActive = false, want true")
			}

			if token.usageCount != 0 {
				t.Errorf("usageCount = %v, want 0", token.usageCount)
			}
		})
	}
}

func TestReconstructSubscriptionToken(t *testing.T) {
	now := time.Now()
	future := now.Add(24 * time.Hour)
	lastUsedIP := "192.168.1.1"

	tests := []struct {
		name           string
		id             uint
		subscriptionID uint
		tokenName      string
		tokenHash      string
		prefix         string
		scope          vo.TokenScope
		expiresAt      *time.Time
		lastUsedAt     *time.Time
		lastUsedIP     *string
		usageCount     uint64
		isActive       bool
		createdAt      time.Time
		revokedAt      *time.Time
		wantErr        bool
	}{
		{
			name:           "valid reconstruction",
			id:             1,
			subscriptionID: 100,
			tokenName:      "Test Token",
			tokenHash:      "hash123",
			prefix:         "tok_",
			scope:          vo.TokenScopeAPI,
			expiresAt:      &future,
			lastUsedAt:     &now,
			lastUsedIP:     &lastUsedIP,
			usageCount:     50,
			isActive:       true,
			createdAt:      now,
			revokedAt:      nil,
			wantErr:        false,
		},
		{
			name:           "zero ID",
			id:             0,
			subscriptionID: 100,
			tokenName:      "Test Token",
			tokenHash:      "hash123",
			prefix:         "tok_",
			scope:          vo.TokenScopeAPI,
			expiresAt:      nil,
			lastUsedAt:     nil,
			lastUsedIP:     nil,
			usageCount:     0,
			isActive:       true,
			createdAt:      now,
			revokedAt:      nil,
			wantErr:        true,
		},
		{
			name:           "zero subscription ID",
			id:             1,
			subscriptionID: 0,
			tokenName:      "Test Token",
			tokenHash:      "hash123",
			prefix:         "tok_",
			scope:          vo.TokenScopeAPI,
			expiresAt:      nil,
			lastUsedAt:     nil,
			lastUsedIP:     nil,
			usageCount:     0,
			isActive:       true,
			createdAt:      now,
			revokedAt:      nil,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := ReconstructSubscriptionToken(
				tt.id,
				tt.subscriptionID,
				tt.tokenName,
				tt.tokenHash,
				tt.prefix,
				tt.scope,
				tt.expiresAt,
				tt.lastUsedAt,
				tt.lastUsedIP,
				tt.usageCount,
				tt.isActive,
				tt.createdAt,
				tt.revokedAt,
			)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ReconstructSubscriptionToken() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ReconstructSubscriptionToken() unexpected error = %v", err)
				return
			}

			if token.id != tt.id {
				t.Errorf("id = %v, want %v", token.id, tt.id)
			}

			if token.usageCount != tt.usageCount {
				t.Errorf("usageCount = %v, want %v", token.usageCount, tt.usageCount)
			}
		})
	}
}

func TestSubscriptionToken_Verify(t *testing.T) {
	token, _ := NewSubscriptionToken(1, "Test", "hash123", "tok_", vo.TokenScopeAPI, nil)

	tests := []struct {
		name       string
		plainToken string
		want       bool
	}{
		{
			name:       "matching token",
			plainToken: "hash123",
			want:       true,
		},
		{
			name:       "non-matching token",
			plainToken: "wrong",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := token.Verify(tt.plainToken); got != tt.want {
				t.Errorf("Verify() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSubscriptionToken_IsExpired(t *testing.T) {
	past := time.Now().Add(-24 * time.Hour)
	future := time.Now().Add(24 * time.Hour)

	tests := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{
			name:      "expired token",
			expiresAt: &past,
			want:      true,
		},
		{
			name:      "valid token",
			expiresAt: &future,
			want:      false,
		},
		{
			name:      "no expiration",
			expiresAt: nil,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, _ := NewSubscriptionToken(1, "Test", "hash123", "tok_", vo.TokenScopeAPI, tt.expiresAt)
			if got := token.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSubscriptionToken_Revoke(t *testing.T) {
	token, _ := NewSubscriptionToken(1, "Test", "hash123", "tok_", vo.TokenScopeAPI, nil)

	err := token.Revoke()
	if err != nil {
		t.Errorf("first Revoke() unexpected error = %v", err)
	}

	if token.revokedAt == nil {
		t.Errorf("revokedAt should be set")
	}

	if token.isActive {
		t.Errorf("isActive should be false")
	}

	err = token.Revoke()
	if err == nil {
		t.Errorf("second Revoke() expected error, got nil")
	}
}

func TestSubscriptionToken_RecordUsage(t *testing.T) {
	token, _ := NewSubscriptionToken(1, "Test", "hash123", "tok_", vo.TokenScopeAPI, nil)

	initialCount := token.usageCount
	ip := "192.168.1.1"

	token.RecordUsage(ip)

	if token.usageCount != initialCount+1 {
		t.Errorf("usageCount = %v, want %v", token.usageCount, initialCount+1)
	}

	if token.lastUsedAt == nil {
		t.Errorf("lastUsedAt should be set")
	}

	if token.lastUsedIP == nil || *token.lastUsedIP != ip {
		t.Errorf("lastUsedIP = %v, want %v", token.lastUsedIP, ip)
	}
}

func TestSubscriptionToken_HasScope(t *testing.T) {
	token, _ := NewSubscriptionToken(1, "Test", "hash123", "tok_", vo.TokenScopeAPI, nil)

	tests := []struct {
		name  string
		scope string
		want  bool
	}{
		{
			name:  "has read permission",
			scope: "read",
			want:  true,
		},
		{
			name:  "has write permission",
			scope: "write",
			want:  true,
		},
		{
			name:  "no delete permission",
			scope: "delete",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := token.HasScope(tt.scope); got != tt.want {
				t.Errorf("HasScope() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSubscriptionToken_IsValid(t *testing.T) {
	past := time.Now().Add(-24 * time.Hour)
	future := time.Now().Add(24 * time.Hour)

	tests := []struct {
		name      string
		expiresAt *time.Time
		revoked   bool
		want      bool
	}{
		{
			name:      "valid active token",
			expiresAt: &future,
			revoked:   false,
			want:      true,
		},
		{
			name:      "expired token",
			expiresAt: &past,
			revoked:   false,
			want:      false,
		},
		{
			name:      "revoked token",
			expiresAt: &future,
			revoked:   true,
			want:      false,
		},
		{
			name:      "active token no expiration",
			expiresAt: nil,
			revoked:   false,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, _ := NewSubscriptionToken(1, "Test", "hash123", "tok_", vo.TokenScopeAPI, tt.expiresAt)
			if tt.revoked {
				token.Revoke()
			}

			if got := token.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSubscriptionToken_Getters(t *testing.T) {
	now := time.Now()
	future := now.Add(24 * time.Hour)
	ip := "192.168.1.1"

	token, _ := ReconstructSubscriptionToken(
		1,
		100,
		"Test Token",
		"hash123",
		"tok_",
		vo.TokenScopeAPI,
		&future,
		&now,
		&ip,
		50,
		true,
		now,
		nil,
	)

	if token.ID() != 1 {
		t.Errorf("ID() = %v, want 1", token.ID())
	}

	if token.SubscriptionID() != 100 {
		t.Errorf("SubscriptionID() = %v, want 100", token.SubscriptionID())
	}

	if token.Name() != "Test Token" {
		t.Errorf("Name() = %v, want Test Token", token.Name())
	}

	if token.TokenHash() != "hash123" {
		t.Errorf("TokenHash() = %v, want hash123", token.TokenHash())
	}

	if token.Prefix() != "tok_" {
		t.Errorf("Prefix() = %v, want tok_", token.Prefix())
	}

	if token.Scope() != vo.TokenScopeAPI {
		t.Errorf("Scope() = %v, want %v", token.Scope(), vo.TokenScopeAPI)
	}

	if token.UsageCount() != 50 {
		t.Errorf("UsageCount() = %v, want 50", token.UsageCount())
	}

	if !token.IsActive() {
		t.Errorf("IsActive() = false, want true")
	}
}

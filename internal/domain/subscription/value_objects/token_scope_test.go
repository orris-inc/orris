package value_objects

import (
	"encoding/json"
	"testing"
)

func TestNewTokenScope(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    TokenScope
		wantErr bool
	}{
		{
			name:    "valid full",
			value:   "full",
			want:    TokenScopeFull,
			wantErr: false,
		},
		{
			name:    "valid read_only",
			value:   "read_only",
			want:    TokenScopeReadOnly,
			wantErr: false,
		},
		{
			name:    "valid api",
			value:   "api",
			want:    TokenScopeAPI,
			wantErr: false,
		},
		{
			name:    "valid webhook",
			value:   "webhook",
			want:    TokenScopeWebhook,
			wantErr: false,
		},
		{
			name:    "valid admin",
			value:   "admin",
			want:    TokenScopeAdmin,
			wantErr: false,
		},
		{
			name:    "invalid scope",
			value:   "invalid",
			wantErr: true,
		},
		{
			name:    "empty scope",
			value:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTokenScope(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTokenScope() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && *got != tt.want {
				t.Errorf("NewTokenScope() = %v, want %v", *got, tt.want)
			}
		})
	}
}

func TestParseTokenScope(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    TokenScope
		wantErr bool
	}{
		{
			name:    "parse full with spaces",
			value:   "  full  ",
			want:    TokenScopeFull,
			wantErr: false,
		},
		{
			name:    "parse uppercase",
			value:   "FULL",
			want:    TokenScopeFull,
			wantErr: false,
		},
		{
			name:    "parse mixed case",
			value:   "FuLl",
			want:    TokenScopeFull,
			wantErr: false,
		},
		{
			name:    "parse empty",
			value:   "",
			wantErr: true,
		},
		{
			name:    "parse invalid",
			value:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTokenScope(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTokenScope() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseTokenScope() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenScope_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		scope TokenScope
		want  bool
	}{
		{
			name:  "valid full",
			scope: TokenScopeFull,
			want:  true,
		},
		{
			name:  "invalid scope",
			scope: TokenScope("invalid"),
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.scope.IsValid(); got != tt.want {
				t.Errorf("TokenScope.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenScope_CanPerform(t *testing.T) {
	tests := []struct {
		name   string
		scope  TokenScope
		action string
		want   bool
	}{
		{
			name:   "full can read",
			scope:  TokenScopeFull,
			action: "read",
			want:   true,
		},
		{
			name:   "full can write",
			scope:  TokenScopeFull,
			action: "write",
			want:   true,
		},
		{
			name:   "full can delete",
			scope:  TokenScopeFull,
			action: "delete",
			want:   true,
		},
		{
			name:   "read_only can read",
			scope:  TokenScopeReadOnly,
			action: "read",
			want:   true,
		},
		{
			name:   "read_only cannot write",
			scope:  TokenScopeReadOnly,
			action: "write",
			want:   false,
		},
		{
			name:   "api can api_call",
			scope:  TokenScopeAPI,
			action: "api_call",
			want:   true,
		},
		{
			name:   "webhook can webhook_trigger",
			scope:  TokenScopeWebhook,
			action: "webhook_trigger",
			want:   true,
		},
		{
			name:   "admin can manage_users",
			scope:  TokenScopeAdmin,
			action: "manage_users",
			want:   true,
		},
		{
			name:   "full cannot manage_users",
			scope:  TokenScopeFull,
			action: "manage_users",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.scope.CanPerform(tt.action); got != tt.want {
				t.Errorf("TokenScope.CanPerform() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenScope_GetPermissions(t *testing.T) {
	tests := []struct {
		name  string
		scope TokenScope
		want  int
	}{
		{
			name:  "full permissions",
			scope: TokenScopeFull,
			want:  4,
		},
		{
			name:  "read_only permissions",
			scope: TokenScopeReadOnly,
			want:  1,
		},
		{
			name:  "admin permissions",
			scope: TokenScopeAdmin,
			want:  7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.scope.GetPermissions()
			if len(got) != tt.want {
				t.Errorf("TokenScope.GetPermissions() length = %v, want %v", len(got), tt.want)
			}
		})
	}
}

func TestTokenScope_IsFull(t *testing.T) {
	tests := []struct {
		name  string
		scope TokenScope
		want  bool
	}{
		{
			name:  "full is full",
			scope: TokenScopeFull,
			want:  true,
		},
		{
			name:  "read_only is not full",
			scope: TokenScopeReadOnly,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.scope.IsFull(); got != tt.want {
				t.Errorf("TokenScope.IsFull() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenScope_IsReadOnly(t *testing.T) {
	tests := []struct {
		name  string
		scope TokenScope
		want  bool
	}{
		{
			name:  "read_only is read_only",
			scope: TokenScopeReadOnly,
			want:  true,
		},
		{
			name:  "full is not read_only",
			scope: TokenScopeFull,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.scope.IsReadOnly(); got != tt.want {
				t.Errorf("TokenScope.IsReadOnly() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenScope_IsAdmin(t *testing.T) {
	tests := []struct {
		name  string
		scope TokenScope
		want  bool
	}{
		{
			name:  "admin is admin",
			scope: TokenScopeAdmin,
			want:  true,
		},
		{
			name:  "full is not admin",
			scope: TokenScopeFull,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.scope.IsAdmin(); got != tt.want {
				t.Errorf("TokenScope.IsAdmin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenScope_HasWriteAccess(t *testing.T) {
	tests := []struct {
		name  string
		scope TokenScope
		want  bool
	}{
		{
			name:  "full has write access",
			scope: TokenScopeFull,
			want:  true,
		},
		{
			name:  "read_only has no write access",
			scope: TokenScopeReadOnly,
			want:  false,
		},
		{
			name:  "api has write access",
			scope: TokenScopeAPI,
			want:  true,
		},
		{
			name:  "admin has write access",
			scope: TokenScopeAdmin,
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.scope.HasWriteAccess(); got != tt.want {
				t.Errorf("TokenScope.HasWriteAccess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenScope_HasDeleteAccess(t *testing.T) {
	tests := []struct {
		name  string
		scope TokenScope
		want  bool
	}{
		{
			name:  "full has delete access",
			scope: TokenScopeFull,
			want:  true,
		},
		{
			name:  "read_only has no delete access",
			scope: TokenScopeReadOnly,
			want:  false,
		},
		{
			name:  "admin has delete access",
			scope: TokenScopeAdmin,
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.scope.HasDeleteAccess(); got != tt.want {
				t.Errorf("TokenScope.HasDeleteAccess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenScope_HasAdminAccess(t *testing.T) {
	tests := []struct {
		name  string
		scope TokenScope
		want  bool
	}{
		{
			name:  "admin has admin access",
			scope: TokenScopeAdmin,
			want:  true,
		},
		{
			name:  "full has no admin access",
			scope: TokenScopeFull,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.scope.HasAdminAccess(); got != tt.want {
				t.Errorf("TokenScope.HasAdminAccess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenScope_Equals(t *testing.T) {
	tests := []struct {
		name  string
		scope TokenScope
		other TokenScope
		want  bool
	}{
		{
			name:  "equal scopes",
			scope: TokenScopeFull,
			other: TokenScopeFull,
			want:  true,
		},
		{
			name:  "different scopes",
			scope: TokenScopeFull,
			other: TokenScopeReadOnly,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.scope.Equals(tt.other); got != tt.want {
				t.Errorf("TokenScope.Equals() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenScope_JSON(t *testing.T) {
	tests := []struct {
		name    string
		scope   TokenScope
		wantErr bool
	}{
		{
			name:    "marshal full",
			scope:   TokenScopeFull,
			wantErr: false,
		},
		{
			name:    "marshal admin",
			scope:   TokenScopeAdmin,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.scope)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				var unmarshaled TokenScope
				err = json.Unmarshal(data, &unmarshaled)
				if err != nil {
					t.Errorf("json.Unmarshal() error = %v", err)
					return
				}
				if unmarshaled != tt.scope {
					t.Errorf("json round trip failed: got %v, want %v", unmarshaled, tt.scope)
				}
			}
		})
	}
}

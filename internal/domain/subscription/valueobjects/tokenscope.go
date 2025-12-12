package valueobjects

import (
	"fmt"
	"strings"
)

type TokenScope string

const (
	TokenScopeFull     TokenScope = "full"
	TokenScopeReadOnly TokenScope = "read_only"
	TokenScopeAPI      TokenScope = "api"
	TokenScopeWebhook  TokenScope = "webhook"
	TokenScopeAdmin    TokenScope = "admin"
)

var ValidTokenScopes = map[TokenScope]bool{
	TokenScopeFull:     true,
	TokenScopeReadOnly: true,
	TokenScopeAPI:      true,
	TokenScopeWebhook:  true,
	TokenScopeAdmin:    true,
}

var TokenScopePermissions = map[TokenScope][]string{
	TokenScopeFull: {
		"read",
		"write",
		"delete",
		"update",
	},
	TokenScopeReadOnly: {
		"read",
	},
	TokenScopeAPI: {
		"read",
		"write",
		"api_call",
	},
	TokenScopeWebhook: {
		"read",
		"webhook_trigger",
	},
	TokenScopeAdmin: {
		"read",
		"write",
		"delete",
		"update",
		"manage_users",
		"manage_permissions",
		"manage_subscriptions",
	},
}

func NewTokenScope(value string) (*TokenScope, error) {
	scope := TokenScope(value)

	if value == "" {
		return nil, fmt.Errorf("token scope cannot be empty")
	}

	if !ValidTokenScopes[scope] {
		return nil, fmt.Errorf("invalid token scope: %s", value)
	}

	return &scope, nil
}

func ParseTokenScope(value string) (TokenScope, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	scope := TokenScope(normalized)

	if normalized == "" {
		return "", fmt.Errorf("token scope cannot be empty")
	}

	if !ValidTokenScopes[scope] {
		return "", fmt.Errorf("invalid token scope: %s", value)
	}

	return scope, nil
}

func (t TokenScope) String() string {
	return string(t)
}

func (t TokenScope) IsValid() bool {
	return ValidTokenScopes[t]
}

func (t TokenScope) CanPerform(action string) bool {
	permissions, exists := TokenScopePermissions[t]
	if !exists {
		return false
	}

	for _, permission := range permissions {
		if permission == action {
			return true
		}
	}

	return false
}

func (t TokenScope) GetPermissions() []string {
	permissions, exists := TokenScopePermissions[t]
	if !exists {
		return []string{}
	}
	return permissions
}

func (t TokenScope) IsFull() bool {
	return t == TokenScopeFull
}

func (t TokenScope) IsReadOnly() bool {
	return t == TokenScopeReadOnly
}

func (t TokenScope) IsAPI() bool {
	return t == TokenScopeAPI
}

func (t TokenScope) IsWebhook() bool {
	return t == TokenScopeWebhook
}

func (t TokenScope) IsAdmin() bool {
	return t == TokenScopeAdmin
}

func (t TokenScope) HasWriteAccess() bool {
	return t.CanPerform("write")
}

func (t TokenScope) HasDeleteAccess() bool {
	return t.CanPerform("delete")
}

func (t TokenScope) HasAdminAccess() bool {
	return t == TokenScopeAdmin
}

func (t TokenScope) Equals(other TokenScope) bool {
	return t == other
}

func (t TokenScope) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.String() + `"`), nil
}

func (t *TokenScope) UnmarshalJSON(data []byte) error {
	str := string(data)
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}

	scope, err := NewTokenScope(str)
	if err != nil {
		return err
	}

	*t = *scope
	return nil
}

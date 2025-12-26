// Package valueobjects provides value objects for the forward domain.
package valueobjects

// RuleScopeType represents the type of rule scope.
type RuleScopeType string

const (
	// RuleScopeTypeSystem indicates a system/admin-created rule.
	RuleScopeTypeSystem RuleScopeType = "system"
	// RuleScopeTypeUser indicates a user-owned rule.
	RuleScopeTypeUser RuleScopeType = "user"
)

// RuleScope represents the ownership scope of a forward rule.
// It encapsulates whether a rule is system-owned (admin-created) or user-owned.
type RuleScope struct {
	scopeType RuleScopeType
	userID    *uint
}

// SystemScope creates a RuleScope for system/admin-created rules.
func SystemScope() RuleScope {
	return RuleScope{scopeType: RuleScopeTypeSystem}
}

// UserScope creates a RuleScope for user-owned rules.
func UserScope(userID uint) RuleScope {
	return RuleScope{
		scopeType: RuleScopeTypeUser,
		userID:    &userID,
	}
}

// IsSystem returns true if this is a system/admin-created rule scope.
func (s RuleScope) IsSystem() bool {
	return s.scopeType == RuleScopeTypeSystem
}

// IsUser returns true if this is a user-owned rule scope.
func (s RuleScope) IsUser() bool {
	return s.scopeType == RuleScopeTypeUser
}

// UserID returns the user ID for user-owned scopes.
// Returns nil for system scopes.
func (s RuleScope) UserID() *uint {
	return s.userID
}

// BelongsTo checks if this scope belongs to the specified user.
// System scopes never belong to any user.
func (s RuleScope) BelongsTo(userID uint) bool {
	return s.userID != nil && *s.userID == userID
}

// Type returns the scope type.
func (s RuleScope) Type() RuleScopeType {
	return s.scopeType
}

// String returns a string representation of the scope.
func (s RuleScope) String() string {
	if s.IsSystem() {
		return "system"
	}
	return "user"
}

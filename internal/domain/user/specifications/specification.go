package specifications

import (
	"fmt"
	"strings"

	"orris/internal/domain/user"
	vo "orris/internal/domain/user/value_objects"
)

// Specification represents a business rule that can be checked
type Specification interface {
	// IsSatisfiedBy checks if the user satisfies the specification
	IsSatisfiedBy(user *user.User) bool
	
	// ToSQL converts the specification to SQL WHERE clause
	ToSQL() (string, []interface{})
	
	// And creates a composite specification with AND operator
	And(spec Specification) Specification
	
	// Or creates a composite specification with OR operator
	Or(spec Specification) Specification
	
	// Not creates a negated specification
	Not() Specification
}

// BaseSpecification provides common functionality for specifications
type BaseSpecification struct{}

// IsSatisfiedBy is a placeholder - should be overridden by concrete implementations
func (s BaseSpecification) IsSatisfiedBy(user *user.User) bool {
	return false
}

// ToSQL is a placeholder - should be overridden by concrete implementations
func (s BaseSpecification) ToSQL() (string, []interface{}) {
	return "1=1", []interface{}{}
}

// And creates an AND composite specification
func (s BaseSpecification) And(spec Specification) Specification {
	return &AndSpecification{
		left:  s,
		right: spec,
	}
}

// Or creates an OR composite specification
func (s BaseSpecification) Or(spec Specification) Specification {
	return &OrSpecification{
		left:  s,
		right: spec,
	}
}

// Not creates a NOT specification
func (s BaseSpecification) Not() Specification {
	return &NotSpecification{
		spec: s,
	}
}

// AndSpecification represents an AND composite specification
type AndSpecification struct {
	left  Specification
	right Specification
}

// IsSatisfiedBy checks if both specifications are satisfied
func (s *AndSpecification) IsSatisfiedBy(user *user.User) bool {
	return s.left.IsSatisfiedBy(user) && s.right.IsSatisfiedBy(user)
}

// ToSQL converts to SQL WHERE clause
func (s *AndSpecification) ToSQL() (string, []interface{}) {
	leftSQL, leftArgs := s.left.ToSQL()
	rightSQL, rightArgs := s.right.ToSQL()
	
	sql := fmt.Sprintf("(%s AND %s)", leftSQL, rightSQL)
	args := append(leftArgs, rightArgs...)
	
	return sql, args
}

// And creates an AND composite specification
func (s *AndSpecification) And(spec Specification) Specification {
	return &AndSpecification{
		left:  s,
		right: spec,
	}
}

// Or creates an OR composite specification
func (s *AndSpecification) Or(spec Specification) Specification {
	return &OrSpecification{
		left:  s,
		right: spec,
	}
}

// Not creates a NOT specification
func (s *AndSpecification) Not() Specification {
	return &NotSpecification{
		spec: s,
	}
}

// OrSpecification represents an OR composite specification
type OrSpecification struct {
	left  Specification
	right Specification
}

// IsSatisfiedBy checks if either specification is satisfied
func (s *OrSpecification) IsSatisfiedBy(user *user.User) bool {
	return s.left.IsSatisfiedBy(user) || s.right.IsSatisfiedBy(user)
}

// ToSQL converts to SQL WHERE clause
func (s *OrSpecification) ToSQL() (string, []interface{}) {
	leftSQL, leftArgs := s.left.ToSQL()
	rightSQL, rightArgs := s.right.ToSQL()
	
	sql := fmt.Sprintf("(%s OR %s)", leftSQL, rightSQL)
	args := append(leftArgs, rightArgs...)
	
	return sql, args
}

// And creates an AND composite specification
func (s *OrSpecification) And(spec Specification) Specification {
	return &AndSpecification{
		left:  s,
		right: spec,
	}
}

// Or creates an OR composite specification
func (s *OrSpecification) Or(spec Specification) Specification {
	return &OrSpecification{
		left:  s,
		right: spec,
	}
}

// Not creates a NOT specification
func (s *OrSpecification) Not() Specification {
	return &NotSpecification{
		spec: s,
	}
}

// NotSpecification represents a NOT specification
type NotSpecification struct {
	spec Specification
}

// IsSatisfiedBy checks if the specification is not satisfied
func (s *NotSpecification) IsSatisfiedBy(user *user.User) bool {
	return !s.spec.IsSatisfiedBy(user)
}

// ToSQL converts to SQL WHERE clause
func (s *NotSpecification) ToSQL() (string, []interface{}) {
	sql, args := s.spec.ToSQL()
	return fmt.Sprintf("NOT (%s)", sql), args
}

// And creates an AND composite specification
func (s *NotSpecification) And(spec Specification) Specification {
	return &AndSpecification{
		left:  s,
		right: spec,
	}
}

// Or creates an OR composite specification
func (s *NotSpecification) Or(spec Specification) Specification {
	return &OrSpecification{
		left:  s,
		right: spec,
	}
}

// Not creates a NOT specification
func (s *NotSpecification) Not() Specification {
	return &NotSpecification{
		spec: s,
	}
}

// ActiveUserSpecification checks if a user is active
type ActiveUserSpecification struct {
	BaseSpecification
}

// NewActiveUserSpecification creates a new active user specification
func NewActiveUserSpecification() *ActiveUserSpecification {
	return &ActiveUserSpecification{}
}

// IsSatisfiedBy checks if the user is active
func (s *ActiveUserSpecification) IsSatisfiedBy(user *user.User) bool {
	return user.Status().IsActive()
}

// ToSQL converts to SQL WHERE clause
func (s *ActiveUserSpecification) ToSQL() (string, []interface{}) {
	return "status = ?", []interface{}{vo.StatusActive.String()}
}

// EmailSpecification checks if a user has a specific email
type EmailSpecification struct {
	BaseSpecification
	email string
}

// NewEmailSpecification creates a new email specification
func NewEmailSpecification(email string) *EmailSpecification {
	return &EmailSpecification{
		email: strings.ToLower(strings.TrimSpace(email)),
	}
}

// IsSatisfiedBy checks if the user has the specified email
func (s *EmailSpecification) IsSatisfiedBy(user *user.User) bool {
	return user.Email().String() == s.email
}

// ToSQL converts to SQL WHERE clause
func (s *EmailSpecification) ToSQL() (string, []interface{}) {
	return "email = ?", []interface{}{s.email}
}

// EmailDomainSpecification checks if a user's email belongs to a specific domain
type EmailDomainSpecification struct {
	BaseSpecification
	domain string
}

// NewEmailDomainSpecification creates a new email domain specification
func NewEmailDomainSpecification(domain string) *EmailDomainSpecification {
	return &EmailDomainSpecification{
		domain: strings.ToLower(strings.TrimSpace(domain)),
	}
}

// IsSatisfiedBy checks if the user's email belongs to the specified domain
func (s *EmailDomainSpecification) IsSatisfiedBy(user *user.User) bool {
	return user.Email().Domain() == s.domain
}

// ToSQL converts to SQL WHERE clause
func (s *EmailDomainSpecification) ToSQL() (string, []interface{}) {
	return "email LIKE ?", []interface{}{"%@" + s.domain}
}

// BusinessEmailSpecification checks if a user has a business email
type BusinessEmailSpecification struct {
	BaseSpecification
}

// NewBusinessEmailSpecification creates a new business email specification
func NewBusinessEmailSpecification() *BusinessEmailSpecification {
	return &BusinessEmailSpecification{}
}

// IsSatisfiedBy checks if the user has a business email
func (s *BusinessEmailSpecification) IsSatisfiedBy(user *user.User) bool {
	return user.IsBusinessEmail()
}

// ToSQL converts to SQL WHERE clause
func (s *BusinessEmailSpecification) ToSQL() (string, []interface{}) {
	// List of free email domains to exclude
	freeEmailDomains := []string{
		"gmail.com", "yahoo.com", "hotmail.com", "outlook.com",
		"icloud.com", "me.com", "qq.com", "163.com",
	}
	
	conditions := make([]string, len(freeEmailDomains))
	args := make([]interface{}, len(freeEmailDomains))
	
	for i, domain := range freeEmailDomains {
		conditions[i] = "email NOT LIKE ?"
		args[i] = "%@" + domain
	}
	
	return strings.Join(conditions, " AND "), args
}

// StatusSpecification checks if a user has a specific status
type StatusSpecification struct {
	BaseSpecification
	status vo.Status
}

// NewStatusSpecification creates a new status specification
func NewStatusSpecification(status vo.Status) *StatusSpecification {
	return &StatusSpecification{
		status: status,
	}
}

// IsSatisfiedBy checks if the user has the specified status
func (s *StatusSpecification) IsSatisfiedBy(user *user.User) bool {
	return user.Status().Equals(s.status)
}

// ToSQL converts to SQL WHERE clause
func (s *StatusSpecification) ToSQL() (string, []interface{}) {
	return "status = ?", []interface{}{s.status.String()}
}

// CanPerformActionsSpecification checks if a user can perform actions
type CanPerformActionsSpecification struct {
	BaseSpecification
}

// NewCanPerformActionsSpecification creates a new specification for users who can perform actions
func NewCanPerformActionsSpecification() *CanPerformActionsSpecification {
	return &CanPerformActionsSpecification{}
}

// IsSatisfiedBy checks if the user can perform actions
func (s *CanPerformActionsSpecification) IsSatisfiedBy(user *user.User) bool {
	return user.CanPerformActions()
}

// ToSQL converts to SQL WHERE clause
func (s *CanPerformActionsSpecification) ToSQL() (string, []interface{}) {
	return "status = ?", []interface{}{vo.StatusActive.String()}
}

// RequiresVerificationSpecification checks if a user requires verification
type RequiresVerificationSpecification struct {
	BaseSpecification
}

// NewRequiresVerificationSpecification creates a new specification for users requiring verification
func NewRequiresVerificationSpecification() *RequiresVerificationSpecification {
	return &RequiresVerificationSpecification{}
}

// IsSatisfiedBy checks if the user requires verification
func (s *RequiresVerificationSpecification) IsSatisfiedBy(user *user.User) bool {
	return user.RequiresVerification()
}

// ToSQL converts to SQL WHERE clause
func (s *RequiresVerificationSpecification) ToSQL() (string, []interface{}) {
	return "status = ?", []interface{}{vo.StatusPending.String()}
}
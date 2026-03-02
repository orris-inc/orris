package valueobjects

import (
	"fmt"
	"strings"
	"unicode"
)

// commonPasswords is a small blacklist of the most commonly used passwords
// that satisfy complexity requirements but are still trivially guessable.
// All keys must be stored in lowercase for case-insensitive matching.
var commonPasswords = map[string]struct{}{
	"password123!":   {},
	"password1234!":  {},
	"passw0rd1234!":  {},
	"welcome12345!":  {},
	"qwerty12345!":   {},
	"admin12345!":    {},
	"changeme123!":   {},
	"letmein12345!":  {},
	"p@ssw0rd1234":   {},
	"abc123456789!":  {},
	"iloveyou1234!":  {},
	"monkey123456!":  {},
	"master123456!":  {},
	"dragon123456!":  {},
	"trustno1!1234":  {},
	"baseball1234!":  {},
	"shadow123456!":  {},
	"superman1234!":  {},
	"michael12345!":  {},
	"football1234!":  {},
}

// PasswordPolicy defines the password validation rules
type PasswordPolicy struct {
	MinLength        int
	RequireUppercase bool
	RequireLowercase bool
	RequireNumber    bool
	RequireSpecial   bool
}

// DefaultPasswordPolicy returns the default password policy with strong complexity requirements
func DefaultPasswordPolicy() *PasswordPolicy {
	return &PasswordPolicy{
		MinLength:        12,
		RequireUppercase: true,
		RequireLowercase: true,
		RequireNumber:    true,
		RequireSpecial:   true,
	}
}

// ValidatePassword validates password against the policy
func (p *PasswordPolicy) ValidatePassword(password string) error {
	if len(password) < p.MinLength {
		return fmt.Errorf("password must be at least %d characters long", p.MinLength)
	}

	if len(password) > 72 {
		return fmt.Errorf("password must not exceed 72 characters (bcrypt limitation)")
	}

	// Check against common password blacklist (case-insensitive)
	if _, found := commonPasswords[strings.ToLower(password)]; found {
		return fmt.Errorf("password is too common, please choose a more unique password")
	}

	var (
		hasUppercase bool
		hasLowercase bool
		hasNumber    bool
		hasSpecial   bool
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUppercase = true
		case unicode.IsLower(char):
			hasLowercase = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if p.RequireUppercase && !hasUppercase {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}

	if p.RequireLowercase && !hasLowercase {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}

	if p.RequireNumber && !hasNumber {
		return fmt.Errorf("password must contain at least one number")
	}

	if p.RequireSpecial && !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}

	return nil
}

// NewPasswordWithPolicy creates a new password value object with custom policy
func NewPasswordWithPolicy(plainPassword string, policy *PasswordPolicy) (*Password, error) {
	if policy == nil {
		policy = DefaultPasswordPolicy()
	}

	if err := policy.ValidatePassword(plainPassword); err != nil {
		return nil, err
	}

	return &Password{value: plainPassword}, nil
}

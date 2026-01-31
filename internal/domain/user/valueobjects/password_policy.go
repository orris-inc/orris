package valueobjects

import (
	"fmt"
	"unicode"
)

// PasswordPolicy defines the password validation rules
type PasswordPolicy struct {
	MinLength        int
	RequireUppercase bool
	RequireLowercase bool
	RequireNumber    bool
	RequireSpecial   bool
}

// DefaultPasswordPolicy returns the default password policy
func DefaultPasswordPolicy() *PasswordPolicy {
	return &PasswordPolicy{
		MinLength:        8,
		RequireUppercase: false,
		RequireLowercase: false,
		RequireNumber:    false,
		RequireSpecial:   false,
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

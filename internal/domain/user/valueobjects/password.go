package valueobjects

import (
	"fmt"
	"unicode"
)

type Password struct {
	value string
}

func NewPassword(plainPassword string) (*Password, error) {
	if err := validatePassword(plainPassword); err != nil {
		return nil, err
	}

	return &Password{value: plainPassword}, nil
}

func (p *Password) String() string {
	return p.value
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	if len(password) > 72 {
		return fmt.Errorf("password must not exceed 72 characters (bcrypt limitation)")
	}

	var (
		hasLetter bool
		hasNumber bool
	)

	for _, char := range password {
		switch {
		case unicode.IsLetter(char):
			hasLetter = true
		case unicode.IsNumber(char):
			hasNumber = true
		}
	}

	if !hasLetter {
		return fmt.Errorf("password must contain at least one letter")
	}

	if !hasNumber {
		return fmt.Errorf("password must contain at least one number")
	}

	return nil
}

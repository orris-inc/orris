package valueobjects

import (
	"fmt"
	"regexp"
	"strings"
)

// emailRegex is the regular expression for validating email addresses
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// Email represents an email address value object
type Email struct {
	value string
}

// NewEmail creates a new Email value object with validation
func NewEmail(value string) (*Email, error) {
	// Normalize the email address
	normalized := strings.TrimSpace(strings.ToLower(value))

	if normalized == "" {
		return nil, fmt.Errorf("email cannot be empty")
	}

	if len(normalized) > 255 {
		return nil, fmt.Errorf("email cannot exceed 255 characters")
	}

	if !emailRegex.MatchString(normalized) {
		return nil, fmt.Errorf("invalid email format: %s (only ASCII characters allowed: a-z, 0-9, . _ %% + -)", value)
	}

	return &Email{value: normalized}, nil
}

// String returns the string representation of the email
func (e *Email) String() string {
	return e.value
}

// Equals checks if two email objects are equal
func (e *Email) Equals(other *Email) bool {
	if e == nil || other == nil {
		return e == other
	}
	return e.value == other.value
}

// Domain returns the domain part of the email
func (e *Email) Domain() string {
	parts := strings.Split(e.value, "@")
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}

// LocalPart returns the local part of the email (before @)
func (e *Email) LocalPart() string {
	parts := strings.Split(e.value, "@")
	if len(parts) == 2 {
		return parts[0]
	}
	return ""
}

// IsBusinessEmail checks if the email is from a business domain (not free email providers)
func (e *Email) IsBusinessEmail() bool {
	freeEmailDomains := map[string]bool{
		"gmail.com":      true,
		"yahoo.com":      true,
		"hotmail.com":    true,
		"outlook.com":    true,
		"icloud.com":     true,
		"me.com":         true,
		"qq.com":         true,
		"163.com":        true,
		"126.com":        true,
		"sina.com":       true,
		"protonmail.com": true,
		"mail.com":       true,
	}

	domain := e.Domain()
	return !freeEmailDomains[domain]
}

// MarshalJSON implements json.Marshaler interface
func (e Email) MarshalJSON() ([]byte, error) {
	return []byte(`"` + e.value + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler interface
func (e *Email) UnmarshalJSON(data []byte) error {
	var value string
	if err := unmarshalJSON(data, &value); err != nil {
		return err
	}

	email, err := NewEmail(value)
	if err != nil {
		return err
	}

	*e = *email
	return nil
}

// Helper function for unmarshaling
func unmarshalJSON(data []byte, v interface{}) error {
	// Remove quotes if present
	str := string(data)
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}

	switch val := v.(type) {
	case *string:
		*val = str
		return nil
	default:
		return fmt.Errorf("unsupported type for unmarshal")
	}
}

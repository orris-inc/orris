package value_objects

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// nameRegex ensures the name contains only valid characters
var nameRegex = regexp.MustCompile(`^[a-zA-Z\s\-'\.]+$`)

// Name represents a person's name value object
type Name struct {
	value string
}

// NewName creates a new Name value object with validation
func NewName(value string) (*Name, error) {
	// Normalize the name
	normalized := strings.TrimSpace(value)
	
	if normalized == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}
	
	if len(normalized) < 2 {
		return nil, fmt.Errorf("name must be at least 2 characters long")
	}
	
	if len(normalized) > 100 {
		return nil, fmt.Errorf("name cannot exceed 100 characters")
	}
	
	// Check for valid characters
	if !nameRegex.MatchString(normalized) {
		return nil, fmt.Errorf("name contains invalid characters: %s", value)
	}
	
	// Check for consecutive spaces
	if strings.Contains(normalized, "  ") {
		return nil, fmt.Errorf("name cannot contain consecutive spaces")
	}
	
	return &Name{value: normalized}, nil
}

// String returns the string representation of the name
func (n *Name) String() string {
	return n.value
}

// Equals checks if two name objects are equal
func (n *Name) Equals(other *Name) bool {
	if n == nil || other == nil {
		return n == other
	}
	return strings.EqualFold(n.value, other.value)
}

// FirstName returns the first name part
func (n *Name) FirstName() string {
	parts := strings.Fields(n.value)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// LastName returns the last name part
func (n *Name) LastName() string {
	parts := strings.Fields(n.value)
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return ""
}

// MiddleNames returns the middle names if any
func (n *Name) MiddleNames() []string {
	parts := strings.Fields(n.value)
	if len(parts) <= 2 {
		return []string{}
	}
	return parts[1 : len(parts)-1]
}

// Initials returns the initials of the name
func (n *Name) Initials() string {
	parts := strings.Fields(n.value)
	var initials []string
	for _, part := range parts {
		if len(part) > 0 {
			initials = append(initials, string(part[0]))
		}
	}
	return strings.ToUpper(strings.Join(initials, ""))
}

// DisplayName returns a formatted display name
func (n *Name) DisplayName() string {
	parts := strings.Fields(n.value)
	var formatted []string
	caser := cases.Title(language.English)
	for _, part := range parts {
		if len(part) > 0 {
			// Capitalize first letter of each part
			formatted = append(formatted, caser.String(strings.ToLower(part)))
		}
	}
	return strings.Join(formatted, " ")
}

// IsMononym checks if the name is a single name (mononym)
func (n *Name) IsMononym() bool {
	return len(strings.Fields(n.value)) == 1
}

// MarshalJSON implements json.Marshaler interface
func (n Name) MarshalJSON() ([]byte, error) {
	return []byte(`"` + n.value + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler interface
func (n *Name) UnmarshalJSON(data []byte) error {
	// Remove quotes if present
	str := string(data)
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}
	
	name, err := NewName(str)
	if err != nil {
		return err
	}
	
	*n = *name
	return nil
}
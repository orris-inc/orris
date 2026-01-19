// Package valueobjects provides value objects for the external forward domain.
package valueobjects

// Status represents the status of an external forward rule.
type Status string

const (
	StatusEnabled  Status = "enabled"
	StatusDisabled Status = "disabled"
)

// IsValid checks if the status is valid.
func (s Status) IsValid() bool {
	switch s {
	case StatusEnabled, StatusDisabled:
		return true
	default:
		return false
	}
}

// IsEnabled returns true if the status is enabled.
func (s Status) IsEnabled() bool {
	return s == StatusEnabled
}

// String returns the string representation of the status.
func (s Status) String() string {
	return string(s)
}

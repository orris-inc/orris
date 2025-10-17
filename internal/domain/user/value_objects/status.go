package value_objects

import (
	"fmt"
	"strings"
)

// Status represents the user status value object
type Status string

// Status constants
const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusPending  Status = "pending"
	StatusSuspended Status = "suspended"
	StatusDeleted  Status = "deleted"
)

// ValidStatuses contains all valid status values
var ValidStatuses = map[Status]bool{
	StatusActive:    true,
	StatusInactive:  true,
	StatusPending:   true,
	StatusSuspended: true,
	StatusDeleted:   true,
}

// StatusTransitions defines allowed status transitions
var StatusTransitions = map[Status][]Status{
	StatusPending: {
		StatusActive,
		StatusInactive,
		StatusDeleted,
	},
	StatusActive: {
		StatusInactive,
		StatusSuspended,
		StatusDeleted,
	},
	StatusInactive: {
		StatusActive,
		StatusDeleted,
	},
	StatusSuspended: {
		StatusActive,
		StatusInactive,
		StatusDeleted,
	},
	StatusDeleted: {
		// No transitions from deleted status
	},
}

// NewStatus creates a new Status value object with validation
func NewStatus(value string) (*Status, error) {
	status := Status(value)

	if value == "" {
		// Default to pending for new users
		status = StatusPending
		return &status, nil
	}

	if !ValidStatuses[status] {
		return nil, fmt.Errorf("invalid status: %s", value)
	}

	return &status, nil
}

// ParseStatus parses a string to Status (case-insensitive)
func ParseStatus(value string) (Status, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	status := Status(normalized)

	if normalized == "" {
		return "", fmt.Errorf("status cannot be empty")
	}

	if !ValidStatuses[status] {
		return "", fmt.Errorf("invalid status: %s", value)
	}

	return status, nil
}

// String returns the string representation of the status
func (s Status) String() string {
	return string(s)
}

// Equals checks if two status objects are equal
func (s Status) Equals(other Status) bool {
	return s == other
}

// IsActive checks if the status is active
func (s Status) IsActive() bool {
	return s == StatusActive
}

// IsInactive checks if the status is inactive
func (s Status) IsInactive() bool {
	return s == StatusInactive
}

// IsPending checks if the status is pending
func (s Status) IsPending() bool {
	return s == StatusPending
}

// IsSuspended checks if the status is suspended
func (s Status) IsSuspended() bool {
	return s == StatusSuspended
}

// IsDeleted checks if the status is deleted
func (s Status) IsDeleted() bool {
	return s == StatusDeleted
}

// CanTransitionTo checks if the current status can transition to the target status
func (s Status) CanTransitionTo(target Status) bool {
	allowedTransitions, exists := StatusTransitions[s]
	if !exists {
		return false
	}
	
	for _, allowed := range allowedTransitions {
		if allowed == target {
			return true
		}
	}
	
	return false
}

// TransitionTo attempts to transition to a new status
func (s *Status) TransitionTo(target Status) error {
	if !s.CanTransitionTo(target) {
		return fmt.Errorf("cannot transition from %s to %s", s.String(), target.String())
	}
	
	*s = target
	return nil
}

// RequiresVerification checks if the status requires verification before activation
func (s Status) RequiresVerification() bool {
	return s == StatusPending
}

// CanPerformActions checks if a user with this status can perform actions
func (s Status) CanPerformActions() bool {
	return s == StatusActive
}

// IsTerminal checks if the status is terminal (no further transitions possible)
func (s Status) IsTerminal() bool {
	return s == StatusDeleted
}

// GetAllowedTransitions returns all allowed transitions from the current status
func (s Status) GetAllowedTransitions() []Status {
	transitions, exists := StatusTransitions[s]
	if !exists {
		return []Status{}
	}
	return transitions
}

// MarshalJSON implements json.Marshaler interface
func (s Status) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s.String() + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler interface
func (s *Status) UnmarshalJSON(data []byte) error {
	// Remove quotes if present
	str := string(data)
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}
	
	status, err := NewStatus(str)
	if err != nil {
		return err
	}
	
	*s = *status
	return nil
}
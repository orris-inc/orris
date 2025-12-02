package value_objects

// ForwardStatus represents the status of a forward rule.
type ForwardStatus string

const (
	ForwardStatusEnabled  ForwardStatus = "enabled"
	ForwardStatusDisabled ForwardStatus = "disabled"
)

var validForwardStatuses = map[ForwardStatus]bool{
	ForwardStatusEnabled:  true,
	ForwardStatusDisabled: true,
}

// String returns the string representation.
func (s ForwardStatus) String() string {
	return string(s)
}

// IsValid checks if the status is valid.
func (s ForwardStatus) IsValid() bool {
	return validForwardStatuses[s]
}

// IsEnabled checks if the status is enabled.
func (s ForwardStatus) IsEnabled() bool {
	return s == ForwardStatusEnabled
}

// IsDisabled checks if the status is disabled.
func (s ForwardStatus) IsDisabled() bool {
	return s == ForwardStatusDisabled
}

// CanEnable checks if the rule can be enabled.
func (s ForwardStatus) CanEnable() bool {
	return s == ForwardStatusDisabled
}

// CanDisable checks if the rule can be disabled.
func (s ForwardStatus) CanDisable() bool {
	return s == ForwardStatusEnabled
}

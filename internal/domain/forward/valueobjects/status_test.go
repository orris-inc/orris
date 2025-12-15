package valueobjects

import "testing"

// TestForwardStatus_IsValid tests the IsValid method for all statuses.
func TestForwardStatus_IsValid(t *testing.T) {
	testCases := []struct {
		name   string
		status ForwardStatus
		want   bool
	}{
		{"enabled is valid", ForwardStatusEnabled, true},
		{"disabled is valid", ForwardStatusDisabled, true},
		{"empty string is invalid", ForwardStatus(""), false},
		{"unknown status is invalid", ForwardStatus("unknown"), false},
		{"invalid status is invalid", ForwardStatus("pending"), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.status.IsValid()
			if got != tc.want {
				t.Errorf("IsValid() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestForwardStatus_IsEnabled tests the IsEnabled predicate.
func TestForwardStatus_IsEnabled(t *testing.T) {
	testCases := []struct {
		name   string
		status ForwardStatus
		want   bool
	}{
		{"enabled returns true", ForwardStatusEnabled, true},
		{"disabled returns false", ForwardStatusDisabled, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.status.IsEnabled()
			if got != tc.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestForwardStatus_IsDisabled tests the IsDisabled predicate.
func TestForwardStatus_IsDisabled(t *testing.T) {
	testCases := []struct {
		name   string
		status ForwardStatus
		want   bool
	}{
		{"disabled returns true", ForwardStatusDisabled, true},
		{"enabled returns false", ForwardStatusEnabled, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.status.IsDisabled()
			if got != tc.want {
				t.Errorf("IsDisabled() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestForwardStatus_CanEnable tests the CanEnable transition logic.
// Business rule: A rule can only be enabled if it's currently disabled.
func TestForwardStatus_CanEnable(t *testing.T) {
	testCases := []struct {
		name   string
		status ForwardStatus
		want   bool
	}{
		{"disabled can be enabled", ForwardStatusDisabled, true},
		{"enabled cannot be enabled", ForwardStatusEnabled, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.status.CanEnable()
			if got != tc.want {
				t.Errorf("CanEnable() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestForwardStatus_CanDisable tests the CanDisable transition logic.
// Business rule: A rule can only be disabled if it's currently enabled.
func TestForwardStatus_CanDisable(t *testing.T) {
	testCases := []struct {
		name   string
		status ForwardStatus
		want   bool
	}{
		{"enabled can be disabled", ForwardStatusEnabled, true},
		{"disabled cannot be disabled", ForwardStatusDisabled, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.status.CanDisable()
			if got != tc.want {
				t.Errorf("CanDisable() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestForwardStatus_String tests the String method.
func TestForwardStatus_String(t *testing.T) {
	testCases := []struct {
		name   string
		status ForwardStatus
		want   string
	}{
		{"enabled to string", ForwardStatusEnabled, "enabled"},
		{"disabled to string", ForwardStatusDisabled, "disabled"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.status.String()
			if got != tc.want {
				t.Errorf("String() = %v, want %v", got, tc.want)
			}
		})
	}
}

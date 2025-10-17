package value_objects

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatus_String(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected string
	}{
		{
			name:     "pending status",
			status:   StatusPending,
			expected: "pending",
		},
		{
			name:     "active status",
			status:   StatusActive,
			expected: "active",
		},
		{
			name:     "inactive status",
			status:   StatusInactive,
			expected: "inactive",
		},
		{
			name:     "suspended status",
			status:   StatusSuspended,
			expected: "suspended",
		},
		{
			name:     "deleted status",
			status:   StatusDeleted,
			expected: "deleted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

func TestStatus_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected bool
	}{
		{"active is active", StatusActive, true},
		{"pending is not active", StatusPending, false},
		{"inactive is not active", StatusInactive, false},
		{"suspended is not active", StatusSuspended, false},
		{"deleted is not active", StatusDeleted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.IsActive())
		})
	}
}

func TestStatus_IsInactive(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected bool
	}{
		{"inactive is inactive", StatusInactive, true},
		{"active is not inactive", StatusActive, false},
		{"pending is not inactive", StatusPending, false},
		{"suspended is not inactive", StatusSuspended, false},
		{"deleted is not inactive", StatusDeleted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.IsInactive())
		})
	}
}

func TestStatus_IsSuspended(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected bool
	}{
		{"suspended is suspended", StatusSuspended, true},
		{"active is not suspended", StatusActive, false},
		{"pending is not suspended", StatusPending, false},
		{"inactive is not suspended", StatusInactive, false},
		{"deleted is not suspended", StatusDeleted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.IsSuspended())
		})
	}
}

func TestStatus_IsDeleted(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected bool
	}{
		{"deleted is deleted", StatusDeleted, true},
		{"active is not deleted", StatusActive, false},
		{"pending is not deleted", StatusPending, false},
		{"inactive is not deleted", StatusInactive, false},
		{"suspended is not deleted", StatusSuspended, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.IsDeleted())
		})
	}
}

func TestStatus_CanPerformActions(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected bool
	}{
		{"active can perform actions", StatusActive, true},
		{"pending cannot perform actions", StatusPending, false},
		{"inactive cannot perform actions", StatusInactive, false},
		{"suspended cannot perform actions", StatusSuspended, false},
		{"deleted cannot perform actions", StatusDeleted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.CanPerformActions())
		})
	}
}

func TestStatus_RequiresVerification(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected bool
	}{
		{"pending requires verification", StatusPending, true},
		{"active does not require verification", StatusActive, false},
		{"inactive does not require verification", StatusInactive, false},
		{"suspended does not require verification", StatusSuspended, false},
		{"deleted does not require verification", StatusDeleted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.RequiresVerification())
		})
	}
}

func TestStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name       string
		from       Status
		to         Status
		canTransit bool
	}{
		// From Pending
		{"pending to active", StatusPending, StatusActive, true},
		{"pending to inactive", StatusPending, StatusInactive, true},
		{"pending to suspended", StatusPending, StatusSuspended, false}, // Not allowed from pending
		{"pending to deleted", StatusPending, StatusDeleted, true},
		{"pending to pending", StatusPending, StatusPending, false},

		// From Active
		{"active to inactive", StatusActive, StatusInactive, true},
		{"active to suspended", StatusActive, StatusSuspended, true},
		{"active to deleted", StatusActive, StatusDeleted, true},
		{"active to pending", StatusActive, StatusPending, false},
		{"active to active", StatusActive, StatusActive, false},

		// From Inactive
		{"inactive to active", StatusInactive, StatusActive, true},
		{"inactive to suspended", StatusInactive, StatusSuspended, false}, // Not allowed from inactive
		{"inactive to deleted", StatusInactive, StatusDeleted, true},
		{"inactive to pending", StatusInactive, StatusPending, false},
		{"inactive to inactive", StatusInactive, StatusInactive, false},

		// From Suspended
		{"suspended to active", StatusSuspended, StatusActive, true},
		{"suspended to inactive", StatusSuspended, StatusInactive, true},
		{"suspended to deleted", StatusSuspended, StatusDeleted, true},
		{"suspended to pending", StatusSuspended, StatusPending, false},
		{"suspended to suspended", StatusSuspended, StatusSuspended, false},

		// From Deleted
		{"deleted to active", StatusDeleted, StatusActive, false},
		{"deleted to pending", StatusDeleted, StatusPending, false},
		{"deleted to inactive", StatusDeleted, StatusInactive, false},
		{"deleted to suspended", StatusDeleted, StatusSuspended, false},
		{"deleted to deleted", StatusDeleted, StatusDeleted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.canTransit, tt.from.CanTransitionTo(tt.to))
		})
	}
}

func TestParseStatus(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  Status
		wantError bool
	}{
		{
			name:      "valid pending",
			input:     "pending",
			expected:  StatusPending,
			wantError: false,
		},
		{
			name:      "valid active",
			input:     "active",
			expected:  StatusActive,
			wantError: false,
		},
		{
			name:      "valid inactive",
			input:     "inactive",
			expected:  StatusInactive,
			wantError: false,
		},
		{
			name:      "valid suspended",
			input:     "suspended",
			expected:  StatusSuspended,
			wantError: false,
		},
		{
			name:      "valid deleted",
			input:     "deleted",
			expected:  StatusDeleted,
			wantError: false,
		},
		{
			name:      "uppercase active",
			input:     "ACTIVE",
			expected:  StatusActive,
			wantError: false,
		},
		{
			name:      "mixed case pending",
			input:     "Pending",
			expected:  StatusPending,
			wantError: false,
		},
		{
			name:      "invalid status",
			input:     "invalid",
			expected:  "",
			wantError: true,
		},
		{
			name:      "empty string",
			input:     "",
			expected:  "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := ParseStatus(tt.input)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, status)
			}
		})
	}
}

func TestValidStatuses(t *testing.T) {
	// Ensure all expected statuses are valid
	expectedStatuses := []Status{
		StatusPending,
		StatusActive,
		StatusInactive,
		StatusSuspended,
		StatusDeleted,
	}

	for _, status := range expectedStatuses {
		assert.True(t, ValidStatuses[status], "Status %s should be valid", status)
	}

	// Ensure invalid status is not in the map
	assert.False(t, ValidStatuses[Status("invalid")])
}

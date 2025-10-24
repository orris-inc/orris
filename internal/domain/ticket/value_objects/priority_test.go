package value_objects

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPriority(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Priority
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid low priority",
			input:   "low",
			want:    PriorityLow,
			wantErr: false,
		},
		{
			name:    "valid medium priority",
			input:   "medium",
			want:    PriorityMedium,
			wantErr: false,
		},
		{
			name:    "valid high priority",
			input:   "high",
			want:    PriorityHigh,
			wantErr: false,
		},
		{
			name:    "valid urgent priority",
			input:   "urgent",
			want:    PriorityUrgent,
			wantErr: false,
		},
		{
			name:    "invalid priority",
			input:   "invalid",
			wantErr: true,
			errMsg:  "invalid priority: invalid",
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
			errMsg:  "invalid priority",
		},
		{
			name:    "case sensitive - uppercase",
			input:   "LOW",
			wantErr: true,
			errMsg:  "invalid priority",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewPriority(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestPriority_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		priority Priority
		want     bool
	}{
		{"low is valid", PriorityLow, true},
		{"medium is valid", PriorityMedium, true},
		{"high is valid", PriorityHigh, true},
		{"urgent is valid", PriorityUrgent, true},
		{"invalid priority", Priority("invalid"), false},
		{"empty priority", Priority(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.priority.IsValid())
		})
	}
}

func TestPriority_String(t *testing.T) {
	tests := []struct {
		name     string
		priority Priority
		want     string
	}{
		{"low", PriorityLow, "low"},
		{"medium", PriorityMedium, "medium"},
		{"high", PriorityHigh, "high"},
		{"urgent", PriorityUrgent, "urgent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.priority.String())
		})
	}
}

func TestPriority_GetSLAHours(t *testing.T) {
	tests := []struct {
		name     string
		priority Priority
		want     int
	}{
		{"low priority 72 hours", PriorityLow, 72},
		{"medium priority 24 hours", PriorityMedium, 24},
		{"high priority 8 hours", PriorityHigh, 8},
		{"urgent priority 2 hours", PriorityUrgent, 2},
		{"invalid priority defaults to 72 hours", Priority("invalid"), 72},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.priority.GetSLAHours())
		})
	}
}

func TestPriority_StateCheckers(t *testing.T) {
	tests := []struct {
		name     string
		priority Priority
		checker  string
		expected bool
	}{
		{"low is low", PriorityLow, "IsLow", true},
		{"medium is not low", PriorityMedium, "IsLow", false},

		{"medium is medium", PriorityMedium, "IsMedium", true},
		{"low is not medium", PriorityLow, "IsMedium", false},

		{"high is high", PriorityHigh, "IsHigh", true},
		{"medium is not high", PriorityMedium, "IsHigh", false},

		{"urgent is urgent", PriorityUrgent, "IsUrgent", true},
		{"high is not urgent", PriorityHigh, "IsUrgent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result bool
			switch tt.checker {
			case "IsLow":
				result = tt.priority.IsLow()
			case "IsMedium":
				result = tt.priority.IsMedium()
			case "IsHigh":
				result = tt.priority.IsHigh()
			case "IsUrgent":
				result = tt.priority.IsUrgent()
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPriority_SLAOrdering(t *testing.T) {
	t.Run("SLA hours are in descending order from low to urgent", func(t *testing.T) {
		assert.Greater(t, PriorityLow.GetSLAHours(), PriorityMedium.GetSLAHours())
		assert.Greater(t, PriorityMedium.GetSLAHours(), PriorityHigh.GetSLAHours())
		assert.Greater(t, PriorityHigh.GetSLAHours(), PriorityUrgent.GetSLAHours())
	})

	t.Run("urgent has shortest SLA", func(t *testing.T) {
		priorities := []Priority{PriorityLow, PriorityMedium, PriorityHigh, PriorityUrgent}
		minSLA := PriorityUrgent.GetSLAHours()

		for _, p := range priorities {
			assert.GreaterOrEqual(t, p.GetSLAHours(), minSLA, "priority %s should have SLA >= urgent", p)
		}
	})

	t.Run("low has longest SLA", func(t *testing.T) {
		priorities := []Priority{PriorityLow, PriorityMedium, PriorityHigh, PriorityUrgent}
		maxSLA := PriorityLow.GetSLAHours()

		for _, p := range priorities {
			assert.LessOrEqual(t, p.GetSLAHours(), maxSLA, "priority %s should have SLA <= low", p)
		}
	})
}

func TestPriority_AllPrioritiesAreValid(t *testing.T) {
	priorities := []Priority{
		PriorityLow,
		PriorityMedium,
		PriorityHigh,
		PriorityUrgent,
	}

	for _, priority := range priorities {
		t.Run(priority.String(), func(t *testing.T) {
			assert.True(t, priority.IsValid(), "priority %s should be valid", priority)
		})
	}
}

func TestPriority_SLAValues(t *testing.T) {
	t.Run("verify exact SLA hours", func(t *testing.T) {
		expectations := map[Priority]int{
			PriorityLow:    72,
			PriorityMedium: 24,
			PriorityHigh:   8,
			PriorityUrgent: 2,
		}

		for priority, expectedHours := range expectations {
			assert.Equal(t, expectedHours, priority.GetSLAHours(),
				"priority %s should have %d hours SLA", priority, expectedHours)
		}
	})
}

func TestPriority_ComprehensiveValidation(t *testing.T) {
	t.Run("all priorities have consistent behavior", func(t *testing.T) {
		priorities := []Priority{PriorityLow, PriorityMedium, PriorityHigh, PriorityUrgent}

		for _, p := range priorities {
			assert.True(t, p.IsValid(), "priority %s should be valid", p)

			assert.NotEmpty(t, p.String(), "priority %s should have non-empty string", p)

			assert.Greater(t, p.GetSLAHours(), 0, "priority %s should have positive SLA hours", p)
		}
	})
}

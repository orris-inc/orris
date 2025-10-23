package node

import (
	"testing"
)

func TestNewNodeStatus_Valid(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected NodeStatus
	}{
		{"active status", "active", NodeStatusActive},
		{"inactive status", "inactive", NodeStatusInactive},
		{"maintenance status", "maintenance", NodeStatusMaintenance},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := NewNodeStatus(tt.status)
			if err != nil {
				t.Errorf("NewNodeStatus(%q) error = %v, want nil", tt.status, err)
				return
			}
			if status != tt.expected {
				t.Errorf("NewNodeStatus(%q) = %v, want %v", tt.status, status, tt.expected)
			}
		})
	}
}

func TestNewNodeStatus_Invalid(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{"invalid status", "invalid"},
		{"empty status", ""},
		{"random string", "random"},
		{"uppercase", "ACTIVE"},
		{"mixed case", "Active"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewNodeStatus(tt.status)
			if err == nil {
				t.Errorf("NewNodeStatus(%q) error = nil, want error", tt.status)
			}
		})
	}
}

func TestNodeStatus_String(t *testing.T) {
	tests := []struct {
		name     string
		status   NodeStatus
		expected string
	}{
		{"active", NodeStatusActive, "active"},
		{"inactive", NodeStatusInactive, "inactive"},
		{"maintenance", NodeStatusMaintenance, "maintenance"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.status.String()
			if result != tt.expected {
				t.Errorf("String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestNodeStatus_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		status   NodeStatus
		expected bool
	}{
		{"active is active", NodeStatusActive, true},
		{"inactive is not active", NodeStatusInactive, false},
		{"maintenance is not active", NodeStatusMaintenance, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.status.IsActive()
			if result != tt.expected {
				t.Errorf("IsActive() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNodeStatus_IsInactive(t *testing.T) {
	tests := []struct {
		name     string
		status   NodeStatus
		expected bool
	}{
		{"active is not inactive", NodeStatusActive, false},
		{"inactive is inactive", NodeStatusInactive, true},
		{"maintenance is not inactive", NodeStatusMaintenance, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.status.IsInactive()
			if result != tt.expected {
				t.Errorf("IsInactive() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNodeStatus_IsMaintenance(t *testing.T) {
	tests := []struct {
		name     string
		status   NodeStatus
		expected bool
	}{
		{"active is not maintenance", NodeStatusActive, false},
		{"inactive is not maintenance", NodeStatusInactive, false},
		{"maintenance is maintenance", NodeStatusMaintenance, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.status.IsMaintenance()
			if result != tt.expected {
				t.Errorf("IsMaintenance() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNodeStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name     string
		from     NodeStatus
		to       NodeStatus
		expected bool
	}{
		{"inactive to active", NodeStatusInactive, NodeStatusActive, true},
		{"inactive to maintenance", NodeStatusInactive, NodeStatusMaintenance, false},
		{"inactive to inactive", NodeStatusInactive, NodeStatusInactive, false},
		{"active to inactive", NodeStatusActive, NodeStatusInactive, true},
		{"active to maintenance", NodeStatusActive, NodeStatusMaintenance, true},
		{"active to active", NodeStatusActive, NodeStatusActive, false},
		{"maintenance to active", NodeStatusMaintenance, NodeStatusActive, true},
		{"maintenance to inactive", NodeStatusMaintenance, NodeStatusInactive, true},
		{"maintenance to maintenance", NodeStatusMaintenance, NodeStatusMaintenance, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.from.CanTransitionTo(tt.to)
			if result != tt.expected {
				t.Errorf("CanTransitionTo(%v -> %v) = %v, want %v", tt.from, tt.to, result, tt.expected)
			}
		})
	}
}

func TestNodeStatus_ValidateTransition(t *testing.T) {
	tests := []struct {
		name    string
		from    NodeStatus
		to      NodeStatus
		wantErr bool
	}{
		{"valid: inactive to active", NodeStatusInactive, NodeStatusActive, false},
		{"valid: active to inactive", NodeStatusActive, NodeStatusInactive, false},
		{"valid: active to maintenance", NodeStatusActive, NodeStatusMaintenance, false},
		{"valid: maintenance to active", NodeStatusMaintenance, NodeStatusActive, false},
		{"valid: maintenance to inactive", NodeStatusMaintenance, NodeStatusInactive, false},
		{"invalid: inactive to maintenance", NodeStatusInactive, NodeStatusMaintenance, true},
		{"invalid: active to active", NodeStatusActive, NodeStatusActive, true},
		{"invalid: inactive to inactive", NodeStatusInactive, NodeStatusInactive, true},
		{"invalid: maintenance to maintenance", NodeStatusMaintenance, NodeStatusMaintenance, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.from.ValidateTransition(tt.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTransition(%v -> %v) error = %v, wantErr %v", tt.from, tt.to, err, tt.wantErr)
			}
		})
	}
}

func TestNodeStatus_Equals(t *testing.T) {
	tests := []struct {
		name     string
		status1  NodeStatus
		status2  NodeStatus
		expected bool
	}{
		{"same active", NodeStatusActive, NodeStatusActive, true},
		{"same inactive", NodeStatusInactive, NodeStatusInactive, true},
		{"same maintenance", NodeStatusMaintenance, NodeStatusMaintenance, true},
		{"active vs inactive", NodeStatusActive, NodeStatusInactive, false},
		{"active vs maintenance", NodeStatusActive, NodeStatusMaintenance, false},
		{"inactive vs maintenance", NodeStatusInactive, NodeStatusMaintenance, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.status1.Equals(tt.status2)
			if result != tt.expected {
				t.Errorf("Equals(%v, %v) = %v, want %v", tt.status1, tt.status2, result, tt.expected)
			}
		})
	}
}

func TestGetAllNodeStatuses(t *testing.T) {
	statuses := GetAllNodeStatuses()

	if len(statuses) != 3 {
		t.Errorf("GetAllNodeStatuses() returned %d statuses, want 3", len(statuses))
	}

	expectedStatuses := map[NodeStatus]bool{
		NodeStatusActive:      true,
		NodeStatusInactive:    true,
		NodeStatusMaintenance: true,
	}

	for _, status := range statuses {
		if !expectedStatuses[status] {
			t.Errorf("GetAllNodeStatuses() contains unexpected status: %v", status)
		}
	}

	for status := range expectedStatuses {
		found := false
		for _, s := range statuses {
			if s == status {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GetAllNodeStatuses() missing expected status: %v", status)
		}
	}
}

func TestNodeStatus_TransitionRules(t *testing.T) {
	t.Run("inactive can only transition to active", func(t *testing.T) {
		status := NodeStatusInactive

		if !status.CanTransitionTo(NodeStatusActive) {
			t.Error("inactive should be able to transition to active")
		}

		if status.CanTransitionTo(NodeStatusMaintenance) {
			t.Error("inactive should not be able to transition to maintenance")
		}

		if status.CanTransitionTo(NodeStatusInactive) {
			t.Error("inactive should not be able to transition to itself")
		}
	})

	t.Run("active can transition to inactive or maintenance", func(t *testing.T) {
		status := NodeStatusActive

		if !status.CanTransitionTo(NodeStatusInactive) {
			t.Error("active should be able to transition to inactive")
		}

		if !status.CanTransitionTo(NodeStatusMaintenance) {
			t.Error("active should be able to transition to maintenance")
		}

		if status.CanTransitionTo(NodeStatusActive) {
			t.Error("active should not be able to transition to itself")
		}
	})

	t.Run("maintenance can transition to active or inactive", func(t *testing.T) {
		status := NodeStatusMaintenance

		if !status.CanTransitionTo(NodeStatusActive) {
			t.Error("maintenance should be able to transition to active")
		}

		if !status.CanTransitionTo(NodeStatusInactive) {
			t.Error("maintenance should be able to transition to inactive")
		}

		if status.CanTransitionTo(NodeStatusMaintenance) {
			t.Error("maintenance should not be able to transition to itself")
		}
	})
}

func TestNodeStatus_ComprehensiveTransitionMatrix(t *testing.T) {
	allStatuses := []NodeStatus{
		NodeStatusActive,
		NodeStatusInactive,
		NodeStatusMaintenance,
	}

	transitionMatrix := map[NodeStatus]map[NodeStatus]bool{
		NodeStatusInactive: {
			NodeStatusActive:      true,
			NodeStatusInactive:    false,
			NodeStatusMaintenance: false,
		},
		NodeStatusActive: {
			NodeStatusActive:      false,
			NodeStatusInactive:    true,
			NodeStatusMaintenance: true,
		},
		NodeStatusMaintenance: {
			NodeStatusActive:      true,
			NodeStatusInactive:    true,
			NodeStatusMaintenance: false,
		},
	}

	for from, transitions := range transitionMatrix {
		for to, expected := range transitions {
			t.Run(from.String()+" to "+to.String(), func(t *testing.T) {
				result := from.CanTransitionTo(to)
				if result != expected {
					t.Errorf("CanTransitionTo(%v -> %v) = %v, want %v", from, to, result, expected)
				}

				err := from.ValidateTransition(to)
				hasError := err != nil
				expectedError := !expected

				if hasError != expectedError {
					t.Errorf("ValidateTransition(%v -> %v) error = %v, want error: %v", from, to, err, expectedError)
				}
			})
		}
	}

	for _, from := range allStatuses {
		for _, to := range allStatuses {
			if _, ok := transitionMatrix[from][to]; !ok {
				t.Errorf("missing transition rule: %v -> %v", from, to)
			}
		}
	}
}

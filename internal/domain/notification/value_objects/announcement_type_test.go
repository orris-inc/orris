package value_objects

import (
	"testing"
)

func TestAnnouncementType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		typeVal  AnnouncementType
		expected bool
	}{
		{"valid system type", AnnouncementTypeSystem, true},
		{"valid maintenance type", AnnouncementTypeMaintenance, true},
		{"valid event type", AnnouncementTypeEvent, true},
		{"invalid type", AnnouncementType("invalid"), false},
		{"empty type", AnnouncementType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.typeVal.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAnnouncementType_String(t *testing.T) {
	tests := []struct {
		name     string
		typeVal  AnnouncementType
		expected string
	}{
		{"system type", AnnouncementTypeSystem, "system"},
		{"maintenance type", AnnouncementTypeMaintenance, "maintenance"},
		{"event type", AnnouncementTypeEvent, "event"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.typeVal.String(); got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAnnouncementType_CheckMethods(t *testing.T) {
	tests := []struct {
		name          string
		typeVal       AnnouncementType
		isSystem      bool
		isMaintenance bool
		isEvent       bool
	}{
		{"system type", AnnouncementTypeSystem, true, false, false},
		{"maintenance type", AnnouncementTypeMaintenance, false, true, false},
		{"event type", AnnouncementTypeEvent, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.typeVal.IsSystem(); got != tt.isSystem {
				t.Errorf("IsSystem() = %v, want %v", got, tt.isSystem)
			}
			if got := tt.typeVal.IsMaintenance(); got != tt.isMaintenance {
				t.Errorf("IsMaintenance() = %v, want %v", got, tt.isMaintenance)
			}
			if got := tt.typeVal.IsEvent(); got != tt.isEvent {
				t.Errorf("IsEvent() = %v, want %v", got, tt.isEvent)
			}
		})
	}
}

func TestNewAnnouncementType(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantType  AnnouncementType
		wantError bool
	}{
		{"valid system", "system", AnnouncementTypeSystem, false},
		{"valid maintenance", "maintenance", AnnouncementTypeMaintenance, false},
		{"valid event", "event", AnnouncementTypeEvent, false},
		{"invalid type", "invalid", "", true},
		{"empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewAnnouncementType(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("NewAnnouncementType() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if got != tt.wantType {
				t.Errorf("NewAnnouncementType() = %v, want %v", got, tt.wantType)
			}
		})
	}
}

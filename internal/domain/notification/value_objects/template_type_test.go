package value_objects

import (
	"testing"
)

func TestTemplateType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		typeVal  TemplateType
		expected bool
	}{
		{"valid subscription expiring", TemplateTypeSubscriptionExpiring, true},
		{"valid system maintenance", TemplateTypeSystemMaintenance, true},
		{"valid welcome", TemplateTypeWelcome, true},
		{"invalid type", TemplateType("invalid"), false},
		{"empty type", TemplateType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.typeVal.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTemplateType_String(t *testing.T) {
	tests := []struct {
		name     string
		typeVal  TemplateType
		expected string
	}{
		{"subscription expiring", TemplateTypeSubscriptionExpiring, "subscription_expiring"},
		{"system maintenance", TemplateTypeSystemMaintenance, "system_maintenance"},
		{"welcome", TemplateTypeWelcome, "welcome"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.typeVal.String(); got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTemplateType_CheckMethods(t *testing.T) {
	tests := []struct {
		name                    string
		typeVal                 TemplateType
		isSubscriptionExpiring  bool
		isSystemMaintenance     bool
		isWelcome               bool
	}{
		{"subscription expiring", TemplateTypeSubscriptionExpiring, true, false, false},
		{"system maintenance", TemplateTypeSystemMaintenance, false, true, false},
		{"welcome", TemplateTypeWelcome, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.typeVal.IsSubscriptionExpiring(); got != tt.isSubscriptionExpiring {
				t.Errorf("IsSubscriptionExpiring() = %v, want %v", got, tt.isSubscriptionExpiring)
			}
			if got := tt.typeVal.IsSystemMaintenance(); got != tt.isSystemMaintenance {
				t.Errorf("IsSystemMaintenance() = %v, want %v", got, tt.isSystemMaintenance)
			}
			if got := tt.typeVal.IsWelcome(); got != tt.isWelcome {
				t.Errorf("IsWelcome() = %v, want %v", got, tt.isWelcome)
			}
		})
	}
}

func TestNewTemplateType(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantType  TemplateType
		wantError bool
	}{
		{"valid subscription expiring", "subscription_expiring", TemplateTypeSubscriptionExpiring, false},
		{"valid system maintenance", "system_maintenance", TemplateTypeSystemMaintenance, false},
		{"valid welcome", "welcome", TemplateTypeWelcome, false},
		{"invalid type", "invalid", "", true},
		{"empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTemplateType(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("NewTemplateType() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if got != tt.wantType {
				t.Errorf("NewTemplateType() = %v, want %v", got, tt.wantType)
			}
		})
	}
}

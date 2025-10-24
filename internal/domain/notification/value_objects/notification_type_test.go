package value_objects

import (
	"testing"
)

func TestNotificationType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		typeVal  NotificationType
		expected bool
	}{
		{"valid system type", NotificationTypeSystem, true},
		{"valid activity type", NotificationTypeActivity, true},
		{"valid subscription type", NotificationTypeSubscription, true},
		{"valid template type", NotificationTypeTemplate, true},
		{"invalid type", NotificationType("invalid"), false},
		{"empty type", NotificationType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.typeVal.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNotificationType_String(t *testing.T) {
	tests := []struct {
		name     string
		typeVal  NotificationType
		expected string
	}{
		{"system type", NotificationTypeSystem, "system"},
		{"activity type", NotificationTypeActivity, "activity"},
		{"subscription type", NotificationTypeSubscription, "subscription"},
		{"template type", NotificationTypeTemplate, "template"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.typeVal.String(); got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNotificationType_CheckMethods(t *testing.T) {
	tests := []struct {
		name           string
		typeVal        NotificationType
		isSystem       bool
		isActivity     bool
		isSubscription bool
		isTemplate     bool
	}{
		{"system type", NotificationTypeSystem, true, false, false, false},
		{"activity type", NotificationTypeActivity, false, true, false, false},
		{"subscription type", NotificationTypeSubscription, false, false, true, false},
		{"template type", NotificationTypeTemplate, false, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.typeVal.IsSystem(); got != tt.isSystem {
				t.Errorf("IsSystem() = %v, want %v", got, tt.isSystem)
			}
			if got := tt.typeVal.IsActivity(); got != tt.isActivity {
				t.Errorf("IsActivity() = %v, want %v", got, tt.isActivity)
			}
			if got := tt.typeVal.IsSubscription(); got != tt.isSubscription {
				t.Errorf("IsSubscription() = %v, want %v", got, tt.isSubscription)
			}
			if got := tt.typeVal.IsTemplate(); got != tt.isTemplate {
				t.Errorf("IsTemplate() = %v, want %v", got, tt.isTemplate)
			}
		})
	}
}

func TestNewNotificationType(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantType  NotificationType
		wantError bool
	}{
		{"valid system", "system", NotificationTypeSystem, false},
		{"valid activity", "activity", NotificationTypeActivity, false},
		{"valid subscription", "subscription", NotificationTypeSubscription, false},
		{"valid template", "template", NotificationTypeTemplate, false},
		{"invalid type", "invalid", "", true},
		{"empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewNotificationType(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("NewNotificationType() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if got != tt.wantType {
				t.Errorf("NewNotificationType() = %v, want %v", got, tt.wantType)
			}
		})
	}
}

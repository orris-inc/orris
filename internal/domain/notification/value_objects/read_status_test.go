package value_objects

import (
	"testing"
)

func TestReadStatus_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		status   ReadStatus
		expected bool
	}{
		{"valid unread", ReadStatusUnread, true},
		{"valid read", ReadStatusRead, true},
		{"invalid status", ReadStatus("invalid"), false},
		{"empty status", ReadStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestReadStatus_String(t *testing.T) {
	tests := []struct {
		name     string
		status   ReadStatus
		expected string
	}{
		{"unread status", ReadStatusUnread, "unread"},
		{"read status", ReadStatusRead, "read"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestReadStatus_CheckMethods(t *testing.T) {
	tests := []struct {
		name     string
		status   ReadStatus
		isUnread bool
		isRead   bool
	}{
		{"unread status", ReadStatusUnread, true, false},
		{"read status", ReadStatusRead, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsUnread(); got != tt.isUnread {
				t.Errorf("IsUnread() = %v, want %v", got, tt.isUnread)
			}
			if got := tt.status.IsRead(); got != tt.isRead {
				t.Errorf("IsRead() = %v, want %v", got, tt.isRead)
			}
		})
	}
}

func TestNewReadStatus(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantStatus ReadStatus
		wantError  bool
	}{
		{"valid unread", "unread", ReadStatusUnread, false},
		{"valid read", "read", ReadStatusRead, false},
		{"invalid status", "invalid", "", true},
		{"empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewReadStatus(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("NewReadStatus() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if got != tt.wantStatus {
				t.Errorf("NewReadStatus() = %v, want %v", got, tt.wantStatus)
			}
		})
	}
}

package value_objects

import (
	"testing"
)

func TestAnnouncementStatus_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		status   AnnouncementStatus
		expected bool
	}{
		{"valid draft", AnnouncementStatusDraft, true},
		{"valid published", AnnouncementStatusPublished, true},
		{"valid expired", AnnouncementStatusExpired, true},
		{"valid deleted", AnnouncementStatusDeleted, true},
		{"invalid status", AnnouncementStatus("invalid"), false},
		{"empty status", AnnouncementStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAnnouncementStatus_String(t *testing.T) {
	tests := []struct {
		name     string
		status   AnnouncementStatus
		expected string
	}{
		{"draft status", AnnouncementStatusDraft, "draft"},
		{"published status", AnnouncementStatusPublished, "published"},
		{"expired status", AnnouncementStatusExpired, "expired"},
		{"deleted status", AnnouncementStatusDeleted, "deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAnnouncementStatus_CheckMethods(t *testing.T) {
	tests := []struct {
		name        string
		status      AnnouncementStatus
		isDraft     bool
		isPublished bool
		isExpired   bool
		isDeleted   bool
	}{
		{"draft status", AnnouncementStatusDraft, true, false, false, false},
		{"published status", AnnouncementStatusPublished, false, true, false, false},
		{"expired status", AnnouncementStatusExpired, false, false, true, false},
		{"deleted status", AnnouncementStatusDeleted, false, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsDraft(); got != tt.isDraft {
				t.Errorf("IsDraft() = %v, want %v", got, tt.isDraft)
			}
			if got := tt.status.IsPublished(); got != tt.isPublished {
				t.Errorf("IsPublished() = %v, want %v", got, tt.isPublished)
			}
			if got := tt.status.IsExpired(); got != tt.isExpired {
				t.Errorf("IsExpired() = %v, want %v", got, tt.isExpired)
			}
			if got := tt.status.IsDeleted(); got != tt.isDeleted {
				t.Errorf("IsDeleted() = %v, want %v", got, tt.isDeleted)
			}
		})
	}
}

func TestAnnouncementStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name     string
		from     AnnouncementStatus
		to       AnnouncementStatus
		expected bool
	}{
		{"draft to published", AnnouncementStatusDraft, AnnouncementStatusPublished, true},
		{"draft to deleted", AnnouncementStatusDraft, AnnouncementStatusDeleted, true},
		{"draft to expired", AnnouncementStatusDraft, AnnouncementStatusExpired, false},
		{"published to expired", AnnouncementStatusPublished, AnnouncementStatusExpired, true},
		{"published to deleted", AnnouncementStatusPublished, AnnouncementStatusDeleted, true},
		{"published to draft", AnnouncementStatusPublished, AnnouncementStatusDraft, false},
		{"expired to published", AnnouncementStatusExpired, AnnouncementStatusPublished, true},
		{"expired to deleted", AnnouncementStatusExpired, AnnouncementStatusDeleted, true},
		{"deleted to any", AnnouncementStatusDeleted, AnnouncementStatusPublished, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.from.CanTransitionTo(tt.to); got != tt.expected {
				t.Errorf("CanTransitionTo() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewAnnouncementStatus(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantStatus AnnouncementStatus
		wantError  bool
	}{
		{"valid draft", "draft", AnnouncementStatusDraft, false},
		{"valid published", "published", AnnouncementStatusPublished, false},
		{"valid expired", "expired", AnnouncementStatusExpired, false},
		{"valid deleted", "deleted", AnnouncementStatusDeleted, false},
		{"invalid status", "invalid", "", true},
		{"empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewAnnouncementStatus(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("NewAnnouncementStatus() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if got != tt.wantStatus {
				t.Errorf("NewAnnouncementStatus() = %v, want %v", got, tt.wantStatus)
			}
		})
	}
}

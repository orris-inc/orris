package trafficstatsutil

import (
	"testing"
	"time"
)

func TestValidateTimeRange(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	tomorrow := now.Add(24 * time.Hour)

	tests := []struct {
		name           string
		from           time.Time
		to             time.Time
		wantErr        bool
		errMsgContains string
	}{
		{
			name:    "valid time range",
			from:    yesterday,
			to:      now,
			wantErr: false,
		},
		{
			name:    "valid time range - same time",
			from:    now,
			to:      now,
			wantErr: false,
		},
		{
			name:    "valid time range - from before to",
			from:    yesterday,
			to:      tomorrow,
			wantErr: false,
		},
		{
			name:           "invalid - zero from time",
			from:           time.Time{},
			to:             now,
			wantErr:        true,
			errMsgContains: "from time is required",
		},
		{
			name:           "invalid - zero to time",
			from:           now,
			to:             time.Time{},
			wantErr:        true,
			errMsgContains: "to time is required",
		},
		{
			name:           "invalid - both zero times",
			from:           time.Time{},
			to:             time.Time{},
			wantErr:        true,
			errMsgContains: "from time is required",
		},
		{
			name:           "invalid - to before from",
			from:           now,
			to:             yesterday,
			wantErr:        true,
			errMsgContains: "to time must be after from time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTimeRange(tt.from, tt.to)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateTimeRange() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errMsgContains != "" && !contains(err.Error(), tt.errMsgContains) {
					t.Errorf("ValidateTimeRange() error = %v, want error containing %q", err.Error(), tt.errMsgContains)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateTimeRange() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidatePaginationInput(t *testing.T) {
	tests := []struct {
		name           string
		page           int
		pageSize       int
		wantErr        bool
		errMsgContains string
	}{
		{
			name:     "valid - positive values",
			page:     1,
			pageSize: 10,
			wantErr:  false,
		},
		{
			name:     "valid - zero values",
			page:     0,
			pageSize: 0,
			wantErr:  false,
		},
		{
			name:     "valid - large values",
			page:     100,
			pageSize: 1000,
			wantErr:  false,
		},
		{
			name:           "invalid - negative page",
			page:           -1,
			pageSize:       10,
			wantErr:        true,
			errMsgContains: "page must be non-negative",
		},
		{
			name:           "invalid - negative pageSize",
			page:           1,
			pageSize:       -1,
			wantErr:        true,
			errMsgContains: "page_size must be non-negative",
		},
		{
			name:           "invalid - both negative",
			page:           -1,
			pageSize:       -1,
			wantErr:        true,
			errMsgContains: "page must be non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePaginationInput(tt.page, tt.pageSize)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidatePaginationInput() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errMsgContains != "" && !contains(err.Error(), tt.errMsgContains) {
					t.Errorf("ValidatePaginationInput() error = %v, want error containing %q", err.Error(), tt.errMsgContains)
				}
			} else {
				if err != nil {
					t.Errorf("ValidatePaginationInput() unexpected error = %v", err)
				}
			}
		})
	}
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

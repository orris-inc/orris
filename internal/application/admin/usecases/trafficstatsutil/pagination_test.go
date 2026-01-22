package trafficstatsutil

import (
	"testing"

	"github.com/orris-inc/orris/internal/shared/constants"
)

func TestGetPaginationParams(t *testing.T) {
	tests := []struct {
		name         string
		page         int
		pageSize     int
		wantPage     int
		wantPageSize int
	}{
		{
			name:         "valid values - no adjustment needed",
			page:         2,
			pageSize:     20,
			wantPage:     2,
			wantPageSize: 20,
		},
		{
			name:         "page less than 1 - defaults to DefaultPage",
			page:         0,
			pageSize:     20,
			wantPage:     constants.DefaultPage,
			wantPageSize: 20,
		},
		{
			name:         "negative page - defaults to DefaultPage",
			page:         -1,
			pageSize:     20,
			wantPage:     constants.DefaultPage,
			wantPageSize: 20,
		},
		{
			name:         "pageSize less than 1 - defaults to DefaultPageSize",
			page:         1,
			pageSize:     0,
			wantPage:     1,
			wantPageSize: constants.DefaultPageSize,
		},
		{
			name:         "negative pageSize - defaults to DefaultPageSize",
			page:         1,
			pageSize:     -1,
			wantPage:     1,
			wantPageSize: constants.DefaultPageSize,
		},
		{
			name:         "both less than 1 - both default",
			page:         0,
			pageSize:     0,
			wantPage:     constants.DefaultPage,
			wantPageSize: constants.DefaultPageSize,
		},
		{
			name:         "pageSize exceeds MaxPageSize - capped",
			page:         1,
			pageSize:     200,
			wantPage:     1,
			wantPageSize: constants.MaxPageSize,
		},
		{
			name:         "pageSize equals MaxPageSize - no cap",
			page:         1,
			pageSize:     constants.MaxPageSize,
			wantPage:     1,
			wantPageSize: constants.MaxPageSize,
		},
		{
			name:         "pageSize just below MaxPageSize - no cap",
			page:         1,
			pageSize:     constants.MaxPageSize - 1,
			wantPage:     1,
			wantPageSize: constants.MaxPageSize - 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetPaginationParams(tt.page, tt.pageSize)
			if got.Page != tt.wantPage {
				t.Errorf("GetPaginationParams().Page = %v, want %v", got.Page, tt.wantPage)
			}
			if got.PageSize != tt.wantPageSize {
				t.Errorf("GetPaginationParams().PageSize = %v, want %v", got.PageSize, tt.wantPageSize)
			}
		})
	}
}

func TestApplyPagination(t *testing.T) {
	tests := []struct {
		name      string
		total     int
		page      int
		pageSize  int
		wantStart int
		wantEnd   int
	}{
		{
			name:      "first page of results",
			total:     50,
			page:      1,
			pageSize:  10,
			wantStart: 0,
			wantEnd:   10,
		},
		{
			name:      "second page of results",
			total:     50,
			page:      2,
			pageSize:  10,
			wantStart: 10,
			wantEnd:   20,
		},
		{
			name:      "last page - partial results",
			total:     25,
			page:      3,
			pageSize:  10,
			wantStart: 20,
			wantEnd:   25,
		},
		{
			name:      "page beyond total - start clamped",
			total:     20,
			page:      5,
			pageSize:  10,
			wantStart: 20,
			wantEnd:   20,
		},
		{
			name:      "empty result set",
			total:     0,
			page:      1,
			pageSize:  10,
			wantStart: 0,
			wantEnd:   0,
		},
		{
			name:      "exact page boundary",
			total:     30,
			page:      3,
			pageSize:  10,
			wantStart: 20,
			wantEnd:   30,
		},
		{
			name:      "single item total",
			total:     1,
			page:      1,
			pageSize:  10,
			wantStart: 0,
			wantEnd:   1,
		},
		{
			name:      "pageSize larger than total",
			total:     5,
			page:      1,
			pageSize:  20,
			wantStart: 0,
			wantEnd:   5,
		},
		{
			name:      "page 0 treated as page -1 result",
			total:     50,
			page:      0,
			pageSize:  10,
			wantStart: -10, // (0-1)*10 = -10, but we test behavior
			wantEnd:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Special handling for edge case where page=0 produces negative start
			// In real usage, GetPaginationParams would prevent this
			if tt.name == "page 0 treated as page -1 result" {
				start, end := ApplyPagination(tt.total, tt.page, tt.pageSize)
				// When start is negative (-10), the function doesn't clamp negative values
				// This is acceptable as GetPaginationParams ensures page >= 1
				if start != -10 || end != 0 {
					t.Errorf("ApplyPagination() = (%v, %v), want (%v, %v)", start, end, -10, 0)
				}
				return
			}

			start, end := ApplyPagination(tt.total, tt.page, tt.pageSize)
			if start != tt.wantStart {
				t.Errorf("ApplyPagination() start = %v, want %v", start, tt.wantStart)
			}
			if end != tt.wantEnd {
				t.Errorf("ApplyPagination() end = %v, want %v", end, tt.wantEnd)
			}
		})
	}
}

package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/shared/constants"
)

func TestValidatePagination(t *testing.T) {
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
			got := ValidatePagination(tt.page, tt.pageSize)
			if got.Page != tt.wantPage {
				t.Errorf("ValidatePagination().Page = %v, want %v", got.Page, tt.wantPage)
			}
			if got.PageSize != tt.wantPageSize {
				t.Errorf("ValidatePagination().PageSize = %v, want %v", got.PageSize, tt.wantPageSize)
			}
		})
	}
}

func TestParsePagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		queryParams  string
		wantPage     int
		wantPageSize int
	}{
		{
			name:         "no params - use defaults",
			queryParams:  "",
			wantPage:     constants.DefaultPage,
			wantPageSize: constants.DefaultPageSize,
		},
		{
			name:         "valid page and page_size",
			queryParams:  "page=3&page_size=25",
			wantPage:     3,
			wantPageSize: 25,
		},
		{
			name:         "invalid page - use default",
			queryParams:  "page=abc&page_size=20",
			wantPage:     constants.DefaultPage,
			wantPageSize: 20,
		},
		{
			name:         "page_size exceeds max - capped",
			queryParams:  "page=1&page_size=500",
			wantPage:     1,
			wantPageSize: constants.MaxPageSize,
		},
		{
			name:         "zero page - use default",
			queryParams:  "page=0&page_size=10",
			wantPage:     constants.DefaultPage,
			wantPageSize: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/?"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			got := ParsePagination(c)
			if got.Page != tt.wantPage {
				t.Errorf("ParsePagination().Page = %v, want %v", got.Page, tt.wantPage)
			}
			if got.PageSize != tt.wantPageSize {
				t.Errorf("ParsePagination().PageSize = %v, want %v", got.PageSize, tt.wantPageSize)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestTotalPages(t *testing.T) {
	tests := []struct {
		name     string
		total    int64
		pageSize int
		want     int
	}{
		{"empty", 0, 10, 1},
		{"exact division", 30, 10, 3},
		{"with remainder", 25, 10, 3},
		{"single page", 5, 10, 1},
		{"zero pageSize", 10, 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TotalPages(tt.total, tt.pageSize); got != tt.want {
				t.Errorf("TotalPages() = %v, want %v", got, tt.want)
			}
		})
	}
}

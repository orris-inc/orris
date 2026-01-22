package trafficstatsutil

import (
	"github.com/orris-inc/orris/internal/shared/constants"
)

// PaginationParams holds calculated pagination parameters.
type PaginationParams struct {
	Page     int
	PageSize int
}

// GetPaginationParams calculates pagination parameters with defaults.
// Page defaults to DefaultPage if less than 1.
// PageSize defaults to DefaultPageSize if less than 1, and is capped at MaxPageSize.
func GetPaginationParams(page, pageSize int) PaginationParams {
	if page < 1 {
		page = constants.DefaultPage
	}

	if pageSize < 1 {
		pageSize = constants.DefaultPageSize
	}
	if pageSize > constants.MaxPageSize {
		pageSize = constants.MaxPageSize
	}

	return PaginationParams{
		Page:     page,
		PageSize: pageSize,
	}
}

// ApplyPagination applies pagination to a slice and returns the start and end indices.
// Returns (start, end) indices for slicing: slice[start:end]
func ApplyPagination(total, page, pageSize int) (start, end int) {
	start = (page - 1) * pageSize
	end = start + pageSize

	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	return start, end
}

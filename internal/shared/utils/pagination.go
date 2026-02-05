package utils

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/shared/constants"
)

// Pagination holds parsed pagination parameters.
type Pagination struct {
	Page     int
	PageSize int
}

// ValidatePagination validates and normalizes pagination parameters.
// Page defaults to DefaultPage if less than 1.
// PageSize defaults to DefaultPageSize if less than 1, and is capped at MaxPageSize.
func ValidatePagination(page, pageSize int) Pagination {
	if page < 1 {
		page = constants.DefaultPage
	}

	if pageSize < 1 {
		pageSize = constants.DefaultPageSize
	}
	if pageSize > constants.MaxPageSize {
		pageSize = constants.MaxPageSize
	}

	return Pagination{
		Page:     page,
		PageSize: pageSize,
	}
}

// ParsePagination parses pagination parameters from Gin context query string.
// Returns validated pagination with defaults applied.
func ParsePagination(c *gin.Context) Pagination {
	return ParsePaginationWithLimits(c, constants.DefaultPageSize, constants.MaxPageSize)
}

// ParsePaginationWithLimits parses pagination parameters with custom default and max page size.
func ParsePaginationWithLimits(c *gin.Context, defaultPageSize, maxPageSize int) Pagination {
	page := parseQueryInt(c, "page", constants.DefaultPage)
	pageSize := parseQueryInt(c, "page_size", defaultPageSize)
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return Pagination{Page: page, PageSize: pageSize}
}

// parseQueryInt parses an integer query parameter with a default value.
func parseQueryInt(c *gin.Context, key string, defaultVal int) int {
	if val := c.Query(key); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n >= 1 {
			return n
		}
	}
	return defaultVal
}

// ApplyPagination calculates slice indices for pagination.
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

// TotalPages calculates total pages for a given total count.
func TotalPages(total int64, pageSize int) int {
	if total == 0 || pageSize == 0 {
		return 1
	}
	pages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if pages == 0 {
		return 1
	}
	return pages
}

// Package trafficstatsutil provides shared utilities for traffic statistics use cases.
package trafficstatsutil

import (
	"time"

	"github.com/orris-inc/orris/internal/shared/errors"
)

// ValidateTimeRange validates start and end time for traffic stats queries.
// Returns error if times are zero or if end time is before start time.
func ValidateTimeRange(from, to time.Time) error {
	if from.IsZero() {
		return errors.NewValidationError("from time is required")
	}

	if to.IsZero() {
		return errors.NewValidationError("to time is required")
	}

	if to.Before(from) {
		return errors.NewValidationError("to time must be after from time")
	}

	return nil
}

// ValidatePaginationInput validates page and pageSize values.
// Returns error if either value is negative.
func ValidatePaginationInput(page, pageSize int) error {
	if page < 0 {
		return errors.NewValidationError("page must be non-negative")
	}

	if pageSize < 0 {
		return errors.NewValidationError("page_size must be non-negative")
	}

	return nil
}

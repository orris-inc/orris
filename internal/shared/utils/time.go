package utils

import (
	"fmt"
	"sync"
	"time"
)

const (
	// DefaultTimezone is the default business timezone
	DefaultTimezone = "Asia/Shanghai"
)

var (
	bizLocation     *time.Location
	bizLocationOnce sync.Once
	initErr         error
)

// InitTimezone initializes the business timezone. Should be called once at startup.
// If tz is empty, defaults to Asia/Shanghai.
func InitTimezone(tz string) error {
	bizLocationOnce.Do(func() {
		if tz == "" {
			tz = DefaultTimezone
		}
		bizLocation, initErr = time.LoadLocation(tz)
	})
	return initErr
}

// MustInitTimezone initializes the business timezone and panics on error.
func MustInitTimezone(tz string) {
	if err := InitTimezone(tz); err != nil {
		panic(fmt.Sprintf("failed to initialize timezone %q: %v", tz, err))
	}
}

// Location returns the business timezone location.
// Returns time.Local if not initialized.
func Location() *time.Location {
	if bizLocation == nil {
		return time.Local
	}
	return bizLocation
}

// Now returns current time in business timezone.
func Now() time.Time {
	return time.Now().In(Location())
}

// TruncateToHour returns current time truncated to hour in business timezone.
func TruncateToHour() time.Time {
	return Now().Truncate(time.Hour)
}

// StartOfDay returns the start of day (00:00:00) for the given time in business timezone.
func StartOfDay(t time.Time) time.Time {
	t = t.In(Location())
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, Location())
}

// EndOfDay returns the end of day (23:59:59.999999999) for the given time in business timezone.
func EndOfDay(t time.Time) time.Time {
	t = t.In(Location())
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, Location())
}

// StartOfMonth returns the start of month for the given year and month in business timezone.
func StartOfMonth(year int, month time.Month) time.Time {
	return time.Date(year, month, 1, 0, 0, 0, 0, Location())
}

// StartOfYear returns the start of year in business timezone.
func StartOfYear(year int) time.Time {
	return time.Date(year, 1, 1, 0, 0, 0, 0, Location())
}

// InBizTimezone converts a time to business timezone.
func InBizTimezone(t time.Time) time.Time {
	return t.In(Location())
}

// AdjustToEndOfDay adjusts the given time to the end of that day (23:59:59.999999999).
// This is useful for date range queries where the 'to' date should include all
// records from that day, not just those at exactly 00:00:00.
//
// For example, if input is 2024-12-31 00:00:00, output will be 2024-12-31 23:59:59.999999999.
// This ensures that queries using "period <= to" will correctly include all records
// from the 'to' date.
func AdjustToEndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
}

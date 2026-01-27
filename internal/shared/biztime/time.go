// Package biztime provides utilities for business timezone calculations.
// All storage and transport use UTC. Business timezone is only used for
// calculating date boundaries (start/end of day, month, year).
//
// Design principles:
// - All time storage is in UTC
// - All business statistics must explicitly specify business timezone
// - Day/month statistics must calculate business timezone boundaries first, then convert to UTC for queries
// - Implicit Local timezone is prohibited
package biztime

import (
	"fmt"
	"sync"
	"time"
)

const (
	// DefaultTimezone is the default business timezone.
	DefaultTimezone = "Asia/Shanghai"
)

var (
	bizLocation     *time.Location
	bizLocationOnce sync.Once
	initErr         error
)

// Init initializes the business timezone. Should be called once at startup.
// If tz is empty, defaults to Asia/Shanghai.
func Init(tz string) error {
	bizLocationOnce.Do(func() {
		if tz == "" {
			tz = DefaultTimezone
		}
		bizLocation, initErr = time.LoadLocation(tz)
	})
	return initErr
}

// MustInit initializes the business timezone and panics on error.
func MustInit(tz string) {
	if err := Init(tz); err != nil {
		panic(fmt.Sprintf("failed to initialize business timezone %q: %v", tz, err))
	}
}

// Location returns the business timezone location.
// If not explicitly initialized, automatically initializes with the default timezone (Asia/Shanghai).
func Location() *time.Location {
	if bizLocation == nil {
		// Auto-initialize with default timezone if not explicitly initialized
		if err := Init(""); err != nil {
			panic(fmt.Sprintf("biztime: failed to auto-initialize with default timezone: %v", err))
		}
	}
	return bizLocation
}

// NowUTC returns current time in UTC.
func NowUTC() time.Time {
	return time.Now().UTC()
}

// TruncateToHourUTC returns current time truncated to hour in UTC.
func TruncateToHourUTC() time.Time {
	return NowUTC().Truncate(time.Hour)
}

// StartOfDayUTC returns the start of day (00:00:00) in business timezone, converted to UTC.
// This is for database queries where we need to find records from the start of a business day.
func StartOfDayUTC(t time.Time) time.Time {
	// Convert to business timezone first
	bizTime := t.In(Location())
	// Get start of day in business timezone
	startOfDay := time.Date(bizTime.Year(), bizTime.Month(), bizTime.Day(), 0, 0, 0, 0, Location())
	// Convert back to UTC for storage/query
	return startOfDay.UTC()
}

// EndOfDayUTC returns the end of day (23:59:59.999999999) in business timezone, converted to UTC.
// This is for database queries where we need to find records until the end of a business day.
func EndOfDayUTC(t time.Time) time.Time {
	// Convert to business timezone first
	bizTime := t.In(Location())
	// Get end of day in business timezone
	endOfDay := time.Date(bizTime.Year(), bizTime.Month(), bizTime.Day(), 23, 59, 59, 999999999, Location())
	// Convert back to UTC for storage/query
	return endOfDay.UTC()
}

// StartOfMonthUTC returns the start of month in business timezone, converted to UTC.
func StartOfMonthUTC(year int, month time.Month) time.Time {
	startOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, Location())
	return startOfMonth.UTC()
}

// EndOfMonthUTC returns the end of month in business timezone, converted to UTC.
func EndOfMonthUTC(year int, month time.Month) time.Time {
	// Get start of next month, then subtract 1 nanosecond
	nextMonth := time.Date(year, month+1, 1, 0, 0, 0, 0, Location())
	endOfMonth := nextMonth.Add(-time.Nanosecond)
	return endOfMonth.UTC()
}

// StartOfYearUTC returns the start of year in business timezone, converted to UTC.
func StartOfYearUTC(year int) time.Time {
	startOfYear := time.Date(year, 1, 1, 0, 0, 0, 0, Location())
	return startOfYear.UTC()
}

// EndOfYearUTC returns the end of year in business timezone, converted to UTC.
func EndOfYearUTC(year int) time.Time {
	endOfYear := time.Date(year, 12, 31, 23, 59, 59, 999999999, Location())
	return endOfYear.UTC()
}

// ToBizTimezone converts a UTC time to business timezone for display.
// Use this only when you need to display time to users.
func ToBizTimezone(t time.Time) time.Time {
	return t.In(Location())
}

// ToUTC converts a time (any timezone) to UTC.
func ToUTC(t time.Time) time.Time {
	return t.UTC()
}

// ParseDateInBizTimezone parses a date string (YYYY-MM-DD) as business timezone midnight,
// then returns the UTC equivalent.
func ParseDateInBizTimezone(dateStr string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02", dateStr, Location())
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format %q: %w", dateStr, err)
	}
	return t.UTC(), nil
}

// FormatInBizTimezone formats a UTC time as a string in business timezone.
func FormatInBizTimezone(t time.Time, layout string) string {
	return t.In(Location()).Format(layout)
}

// TruncateToHourInBiz truncates a time to hour boundary in business timezone,
// then returns the UTC equivalent. This is useful for hourly aggregation
// where "hour" means business timezone hour.
func TruncateToHourInBiz(t time.Time) time.Time {
	bizTime := t.In(Location())
	truncated := time.Date(bizTime.Year(), bizTime.Month(), bizTime.Day(), bizTime.Hour(), 0, 0, 0, Location())
	return truncated.UTC()
}

// MySQLTimezoneOffset returns the timezone offset in MySQL CONVERT_TZ format (e.g., "+08:00").
// This is useful for SQL queries that need to convert UTC to business timezone.
func MySQLTimezoneOffset() string {
	// Get offset at current time (handles DST if applicable)
	_, offset := time.Now().In(Location()).Zone()
	hours := offset / 3600
	minutes := (offset % 3600) / 60
	if minutes < 0 {
		minutes = -minutes
	}
	if hours >= 0 {
		return fmt.Sprintf("+%02d:%02d", hours, minutes)
	}
	return fmt.Sprintf("%03d:%02d", hours, minutes)
}

// FormatMetadataTime formats a UTC time for storage in metadata using RFC3339 format.
// This ensures consistent timestamp serialization across the application.
func FormatMetadataTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

// ParseMetadataTime parses a timestamp from metadata string (RFC3339 format).
// This is the counterpart to FormatMetadataTime for deserializing timestamps from metadata.
func ParseMetadataTime(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid metadata timestamp format %q: %w", s, err)
	}
	return t, nil
}

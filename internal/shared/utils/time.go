package utils

import (
	"time"
)

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

package trafficstatsutil

import (
	"time"

	"github.com/orris-inc/orris/internal/shared/biztime"
)

const (
	// RedisDataRetentionHours is the number of hours Redis retains traffic data.
	RedisDataRetentionHours = 48
)

// TimeWindow represents the calculated time boundaries for Redis and MySQL queries.
type TimeWindow struct {
	// AdjustedTo is the end time adjusted to end of day
	AdjustedTo time.Time
	// RedisDataStart is the earliest time Redis has data for
	RedisDataStart time.Time
	// IncludesRedisWindow indicates if the query overlaps with Redis data window
	IncludesRedisWindow bool
	// IncludesHistory indicates if the query includes historical data before Redis window
	IncludesHistory bool
}

// CalculateTimeWindow calculates the time boundaries for Redis and MySQL queries.
// Redis stores data for the last 48 hours, so queries may need to fetch from both sources.
func CalculateTimeWindow(from, to time.Time) TimeWindow {
	// Adjust 'to' time to end of day to include all records from that day
	adjustedTo := biztime.EndOfDayUTC(to)

	// Calculate time boundaries
	now := biztime.NowUTC()
	// Redis stores data for the last 48 hours
	redisDataStart := now.Add(-RedisDataRetentionHours * time.Hour)

	// Determine if query overlaps with Redis data window (last 48 hours)
	includesRedisWindow := !adjustedTo.Before(redisDataStart)
	includesHistory := from.Before(redisDataStart)

	return TimeWindow{
		AdjustedTo:          adjustedTo,
		RedisDataStart:      redisDataStart,
		IncludesRedisWindow: includesRedisWindow,
		IncludesHistory:     includesHistory,
	}
}

// GetRedisQueryRange returns the time range for Redis query.
// Returns the adjusted from time (clamped to RedisDataStart if needed) and the adjustedTo.
func (tw TimeWindow) GetRedisQueryRange(from time.Time) (redisFrom, redisTo time.Time) {
	redisFrom = from
	if redisFrom.Before(tw.RedisDataStart) {
		redisFrom = tw.RedisDataStart
	}
	return redisFrom, tw.AdjustedTo
}

// GetMySQLQueryRange returns the time range for MySQL query.
// Returns the from time and adjusted to time (excluding Redis window if applicable).
func (tw TimeWindow) GetMySQLQueryRange(from time.Time) (mysqlFrom, mysqlTo time.Time) {
	mysqlTo = tw.AdjustedTo
	if tw.IncludesRedisWindow {
		// Exclude Redis window from MySQL query
		mysqlTo = tw.RedisDataStart.Add(-time.Nanosecond)
	}
	return from, mysqlTo
}

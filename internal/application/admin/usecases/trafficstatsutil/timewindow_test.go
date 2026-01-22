package trafficstatsutil

import (
	"testing"
	"time"

	"github.com/orris-inc/orris/internal/shared/biztime"
)

func TestCalculateTimeWindow(t *testing.T) {
	// Note: CalculateTimeWindow uses biztime.NowUTC() internally, which means
	// there may be a small time drift between our test setup and the actual call.
	// We use offsets large enough to avoid edge case failures due to this drift.

	// Note: CalculateTimeWindow uses EndOfDayUTC(to) for adjustedTo,
	// which extends to the end of that calendar day. This affects
	// IncludesRedisWindow calculation since adjustedTo may extend beyond
	// the original 'to' time into the Redis data window.

	t.Run("time range entirely within Redis window", func(t *testing.T) {
		now := biztime.NowUTC()
		from := now.Add(-24 * time.Hour)
		to := now

		got := CalculateTimeWindow(from, to)

		if !got.IncludesRedisWindow {
			t.Errorf("IncludesRedisWindow = false, want true")
		}
		if got.IncludesHistory {
			t.Errorf("IncludesHistory = true, want false")
		}
	})

	t.Run("time range entirely before Redis window (history only)", func(t *testing.T) {
		now := biztime.NowUTC()
		// Use dates far enough in the past to ensure adjustedTo is also before Redis window
		from := now.Add(-200 * time.Hour)
		to := now.Add(-150 * time.Hour)

		got := CalculateTimeWindow(from, to)

		if got.IncludesRedisWindow {
			t.Errorf("IncludesRedisWindow = true, want false (adjustedTo=%v, redisDataStart=%v)",
				got.AdjustedTo, got.RedisDataStart)
		}
		if !got.IncludesHistory {
			t.Errorf("IncludesHistory = false, want true")
		}
	})

	t.Run("time range spans both Redis and history", func(t *testing.T) {
		now := biztime.NowUTC()
		from := now.Add(-100 * time.Hour)
		to := now

		got := CalculateTimeWindow(from, to)

		if !got.IncludesRedisWindow {
			t.Errorf("IncludesRedisWindow = false, want true")
		}
		if !got.IncludesHistory {
			t.Errorf("IncludesHistory = false, want true")
		}
	})

	t.Run("from well within Redis window", func(t *testing.T) {
		now := biztime.NowUTC()
		// Use a time clearly within Redis window (12 hours ago) to avoid timing edge cases
		from := now.Add(-12 * time.Hour)
		to := now

		got := CalculateTimeWindow(from, to)

		if !got.IncludesRedisWindow {
			t.Errorf("IncludesRedisWindow = false, want true")
		}
		if got.IncludesHistory {
			t.Errorf("IncludesHistory = true, want false (from=%v, redisDataStart=%v)",
				from, got.RedisDataStart)
		}
	})

	t.Run("from clearly before Redis data start boundary", func(t *testing.T) {
		now := biztime.NowUTC()
		// Use a time clearly before Redis window (60 hours ago, well past 48 hours)
		from := now.Add(-60 * time.Hour)
		to := now

		got := CalculateTimeWindow(from, to)

		if !got.IncludesRedisWindow {
			t.Errorf("IncludesRedisWindow = false, want true")
		}
		if !got.IncludesHistory {
			t.Errorf("IncludesHistory = false, want true (from=%v, redisDataStart=%v)",
				from, got.RedisDataStart)
		}
	})

	t.Run("very recent time range (last hour)", func(t *testing.T) {
		now := biztime.NowUTC()
		from := now.Add(-1 * time.Hour)
		to := now

		got := CalculateTimeWindow(from, to)

		if !got.IncludesRedisWindow {
			t.Errorf("IncludesRedisWindow = false, want true")
		}
		if got.IncludesHistory {
			t.Errorf("IncludesHistory = true, want false")
		}
	})

	t.Run("AdjustedTo is end of day", func(t *testing.T) {
		now := biztime.NowUTC()
		from := now.Add(-24 * time.Hour)
		to := now

		got := CalculateTimeWindow(from, to)

		expectedAdjustedTo := biztime.EndOfDayUTC(to)
		if !got.AdjustedTo.Equal(expectedAdjustedTo) {
			t.Errorf("AdjustedTo = %v, want %v", got.AdjustedTo, expectedAdjustedTo)
		}
	})
}

func TestTimeWindow_GetRedisQueryRange(t *testing.T) {
	now := biztime.NowUTC()
	redisDataStart := now.Add(-RedisDataRetentionHours * time.Hour)
	adjustedTo := biztime.EndOfDayUTC(now)

	tw := TimeWindow{
		AdjustedTo:          adjustedTo,
		RedisDataStart:      redisDataStart,
		IncludesRedisWindow: true,
		IncludesHistory:     true,
	}

	tests := []struct {
		name          string
		from          time.Time
		wantRedisFrom time.Time
		wantRedisTo   time.Time
	}{
		{
			name:          "from within Redis window - no adjustment",
			from:          now.Add(-24 * time.Hour),
			wantRedisFrom: now.Add(-24 * time.Hour),
			wantRedisTo:   adjustedTo,
		},
		{
			name:          "from before Redis window - clamped to RedisDataStart",
			from:          now.Add(-100 * time.Hour),
			wantRedisFrom: redisDataStart,
			wantRedisTo:   adjustedTo,
		},
		{
			name:          "from exactly at Redis data start",
			from:          redisDataStart,
			wantRedisFrom: redisDataStart,
			wantRedisTo:   adjustedTo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFrom, gotTo := tw.GetRedisQueryRange(tt.from)

			if !gotFrom.Equal(tt.wantRedisFrom) {
				t.Errorf("GetRedisQueryRange() redisFrom = %v, want %v", gotFrom, tt.wantRedisFrom)
			}

			if !gotTo.Equal(tt.wantRedisTo) {
				t.Errorf("GetRedisQueryRange() redisTo = %v, want %v", gotTo, tt.wantRedisTo)
			}
		})
	}
}

func TestTimeWindow_GetMySQLQueryRange(t *testing.T) {
	now := biztime.NowUTC()
	redisDataStart := now.Add(-RedisDataRetentionHours * time.Hour)
	adjustedTo := biztime.EndOfDayUTC(now)

	tests := []struct {
		name                string
		tw                  TimeWindow
		from                time.Time
		wantMySQLFrom       time.Time
		wantMySQLToExcluded bool // if true, mysqlTo should be redisDataStart - 1ns
	}{
		{
			name: "includes Redis window - MySQL range excludes Redis window",
			tw: TimeWindow{
				AdjustedTo:          adjustedTo,
				RedisDataStart:      redisDataStart,
				IncludesRedisWindow: true,
				IncludesHistory:     true,
			},
			from:                now.Add(-100 * time.Hour),
			wantMySQLFrom:       now.Add(-100 * time.Hour),
			wantMySQLToExcluded: true,
		},
		{
			name: "no Redis window - MySQL range uses full adjustedTo",
			tw: TimeWindow{
				AdjustedTo:          biztime.EndOfDayUTC(now.Add(-60 * time.Hour)),
				RedisDataStart:      redisDataStart,
				IncludesRedisWindow: false,
				IncludesHistory:     true,
			},
			from:                now.Add(-100 * time.Hour),
			wantMySQLFrom:       now.Add(-100 * time.Hour),
			wantMySQLToExcluded: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFrom, gotTo := tt.tw.GetMySQLQueryRange(tt.from)

			if !gotFrom.Equal(tt.wantMySQLFrom) {
				t.Errorf("GetMySQLQueryRange() mysqlFrom = %v, want %v", gotFrom, tt.wantMySQLFrom)
			}

			if tt.wantMySQLToExcluded {
				expectedTo := tt.tw.RedisDataStart.Add(-time.Nanosecond)
				if !gotTo.Equal(expectedTo) {
					t.Errorf("GetMySQLQueryRange() mysqlTo = %v, want %v (RedisDataStart - 1ns)", gotTo, expectedTo)
				}
			} else {
				if !gotTo.Equal(tt.tw.AdjustedTo) {
					t.Errorf("GetMySQLQueryRange() mysqlTo = %v, want %v", gotTo, tt.tw.AdjustedTo)
				}
			}
		})
	}
}

func TestRedisDataRetentionHours(t *testing.T) {
	// Verify the constant is set to expected value (48 hours)
	if RedisDataRetentionHours != 48 {
		t.Errorf("RedisDataRetentionHours = %v, want 48", RedisDataRetentionHours)
	}
}

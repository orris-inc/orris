package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// nopLogger is a no-op logger for testing.
type nopLogger struct{}

func newNopLogger() logger.Interface { return &nopLogger{} }

func (l *nopLogger) Debug(msg string, args ...any)                   {}
func (l *nopLogger) Info(msg string, args ...any)                    {}
func (l *nopLogger) Warn(msg string, args ...any)                    {}
func (l *nopLogger) Error(msg string, args ...any)                   {}
func (l *nopLogger) Fatal(msg string, args ...any)                   {}
func (l *nopLogger) With(args ...any) logger.Interface               { return l }
func (l *nopLogger) Named(name string) logger.Interface              { return l }
func (l *nopLogger) Debugw(msg string, keysAndValues ...interface{}) {}
func (l *nopLogger) Infow(msg string, keysAndValues ...interface{})  {}
func (l *nopLogger) Warnw(msg string, keysAndValues ...interface{})  {}
func (l *nopLogger) Errorw(msg string, keysAndValues ...interface{}) {}
func (l *nopLogger) Fatalw(msg string, keysAndValues ...interface{}) {}

func setupTestRedis(t *testing.T) (*redis.Client, func()) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return client, func() {
		client.Close()
		mr.Close()
	}
}

func TestFormatHourKey(t *testing.T) {
	// Initialize biztime
	biztime.MustInit("Asia/Shanghai")

	tests := []struct {
		name     string
		utcTime  time.Time
		expected string
	}{
		{
			name:     "UTC midnight converts to Shanghai 08:00",
			utcTime:  time.Date(2025, 1, 7, 0, 0, 0, 0, time.UTC),
			expected: "2025010708",
		},
		{
			name:     "UTC 16:00 converts to Shanghai 00:00 next day",
			utcTime:  time.Date(2025, 1, 7, 16, 0, 0, 0, time.UTC),
			expected: "2025010800",
		},
		{
			name:     "UTC 12:30 converts to Shanghai 20:30 same day",
			utcTime:  time.Date(2025, 1, 7, 12, 30, 0, 0, time.UTC),
			expected: "2025010720",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatHourKey(tt.utcTime)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseHourKey(t *testing.T) {
	// Initialize biztime
	biztime.MustInit("Asia/Shanghai")

	tests := []struct {
		name        string
		hourKey     string
		expectedUTC time.Time
		expectError bool
	}{
		{
			name:        "valid hour key",
			hourKey:     "2025010708",
			expectedUTC: time.Date(2025, 1, 7, 0, 0, 0, 0, time.UTC), // Shanghai 08:00 = UTC 00:00
		},
		{
			name:        "another valid hour key",
			hourKey:     "2025010800",
			expectedUTC: time.Date(2025, 1, 7, 16, 0, 0, 0, time.UTC), // Shanghai 00:00 = UTC 16:00 prev day
		},
		{
			name:        "invalid format",
			hourKey:     "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseHourKey(tt.hourKey)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedUTC, result)
			}
		})
	}
}

func TestHourlyTrafficKey(t *testing.T) {
	key := hourlyTrafficKey("2025010712", 123, "node", 456)
	assert.Equal(t, "sub_hourly:2025010712:123:node:456", key)
}

func TestHourlyActiveSetKey(t *testing.T) {
	key := hourlyActiveSetKey("2025010712")
	assert.Equal(t, "sub_hourly:active:2025010712", key)
}

func TestParseHourlyTrafficKey(t *testing.T) {
	tests := []struct {
		name            string
		key             string
		expectedHourKey string
		expectedSubID   uint
		expectedResType string
		expectedResID   uint
		expectError     bool
	}{
		{
			name:            "valid key",
			key:             "sub_hourly:2025010712:123:node:456",
			expectedHourKey: "2025010712",
			expectedSubID:   123,
			expectedResType: "node",
			expectedResID:   456,
		},
		{
			name:        "invalid prefix",
			key:         "invalid:2025010712:123:node:456",
			expectError: true,
		},
		{
			name:        "invalid format - missing parts",
			key:         "sub_hourly:2025010712:123",
			expectError: true,
		},
		{
			name:        "invalid subscription ID",
			key:         "sub_hourly:2025010712:abc:node:456",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hourKey, subID, resType, resID, err := parseHourlyTrafficKey(tt.key)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedHourKey, hourKey)
				assert.Equal(t, tt.expectedSubID, subID)
				assert.Equal(t, tt.expectedResType, resType)
				assert.Equal(t, tt.expectedResID, resID)
			}
		})
	}
}

func TestRedisHourlyTrafficCache_IncrementAndGet(t *testing.T) {
	// Initialize biztime
	biztime.MustInit("Asia/Shanghai")

	client, cleanup := setupTestRedis(t)
	defer cleanup()

	log := newNopLogger()
	cache := NewRedisHourlyTrafficCache(client, log)
	ctx := context.Background()

	// Test increment
	err := cache.IncrementHourlyTraffic(ctx, 1, "node", 100, 1000, 2000)
	require.NoError(t, err)

	// Test get - use current hour
	currentHour := biztime.TruncateToHourInBiz(biztime.NowUTC())
	upload, download, err := cache.GetHourlyTraffic(ctx, currentHour, 1, "node", 100)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), upload)
	assert.Equal(t, int64(2000), download)

	// Test increment again (should accumulate)
	err = cache.IncrementHourlyTraffic(ctx, 1, "node", 100, 500, 300)
	require.NoError(t, err)

	upload, download, err = cache.GetHourlyTraffic(ctx, currentHour, 1, "node", 100)
	require.NoError(t, err)
	assert.Equal(t, int64(1500), upload)
	assert.Equal(t, int64(2300), download)
}

func TestRedisHourlyTrafficCache_GetNonExistent(t *testing.T) {
	// Initialize biztime
	biztime.MustInit("Asia/Shanghai")

	client, cleanup := setupTestRedis(t)
	defer cleanup()

	log := newNopLogger()
	cache := NewRedisHourlyTrafficCache(client, log)
	ctx := context.Background()

	// Test get non-existent
	upload, download, err := cache.GetHourlyTraffic(ctx, time.Now(), 999, "node", 999)
	require.NoError(t, err)
	assert.Equal(t, int64(0), upload)
	assert.Equal(t, int64(0), download)
}

func TestRedisHourlyTrafficCache_ZeroIncrement(t *testing.T) {
	// Initialize biztime
	biztime.MustInit("Asia/Shanghai")

	client, cleanup := setupTestRedis(t)
	defer cleanup()

	log := newNopLogger()
	cache := NewRedisHourlyTrafficCache(client, log)
	ctx := context.Background()

	// Zero increment should be a no-op
	err := cache.IncrementHourlyTraffic(ctx, 1, "node", 100, 0, 0)
	require.NoError(t, err)

	// Verify nothing was written
	currentHour := biztime.TruncateToHourInBiz(biztime.NowUTC())
	upload, download, err := cache.GetHourlyTraffic(ctx, currentHour, 1, "node", 100)
	require.NoError(t, err)
	assert.Equal(t, int64(0), upload)
	assert.Equal(t, int64(0), download)
}

func TestRedisHourlyTrafficCache_GetAllHourlyTraffic(t *testing.T) {
	// Initialize biztime
	biztime.MustInit("Asia/Shanghai")

	client, cleanup := setupTestRedis(t)
	defer cleanup()

	log := newNopLogger()
	cache := NewRedisHourlyTrafficCache(client, log)
	ctx := context.Background()

	// Add traffic for multiple subscriptions/resources
	err := cache.IncrementHourlyTraffic(ctx, 1, "node", 100, 1000, 2000)
	require.NoError(t, err)

	err = cache.IncrementHourlyTraffic(ctx, 2, "node", 101, 500, 600)
	require.NoError(t, err)

	err = cache.IncrementHourlyTraffic(ctx, 1, "forward", 200, 300, 400)
	require.NoError(t, err)

	// Get all traffic for current hour
	currentHour := biztime.TruncateToHourInBiz(biztime.NowUTC())
	data, err := cache.GetAllHourlyTraffic(ctx, currentHour)
	require.NoError(t, err)
	assert.Len(t, data, 3)

	// Verify data (order may vary)
	dataMap := make(map[string]HourlyTrafficData)
	for _, d := range data {
		key := hourlyTrafficKey(formatHourKey(currentHour), d.SubscriptionID, d.ResourceType, d.ResourceID)
		dataMap[key] = d
	}

	// Check first entry
	key1 := hourlyTrafficKey(formatHourKey(currentHour), 1, "node", 100)
	assert.Equal(t, int64(1000), dataMap[key1].Upload)
	assert.Equal(t, int64(2000), dataMap[key1].Download)

	// Check second entry
	key2 := hourlyTrafficKey(formatHourKey(currentHour), 2, "node", 101)
	assert.Equal(t, int64(500), dataMap[key2].Upload)
	assert.Equal(t, int64(600), dataMap[key2].Download)

	// Check third entry
	key3 := hourlyTrafficKey(formatHourKey(currentHour), 1, "forward", 200)
	assert.Equal(t, int64(300), dataMap[key3].Upload)
	assert.Equal(t, int64(400), dataMap[key3].Download)
}

func TestRedisHourlyTrafficCache_CleanupHour(t *testing.T) {
	// Initialize biztime
	biztime.MustInit("Asia/Shanghai")

	client, cleanup := setupTestRedis(t)
	defer cleanup()

	log := newNopLogger()
	cache := NewRedisHourlyTrafficCache(client, log)
	ctx := context.Background()

	// Add traffic
	err := cache.IncrementHourlyTraffic(ctx, 1, "node", 100, 1000, 2000)
	require.NoError(t, err)

	err = cache.IncrementHourlyTraffic(ctx, 2, "node", 101, 500, 600)
	require.NoError(t, err)

	// Verify data exists
	currentHour := biztime.TruncateToHourInBiz(biztime.NowUTC())
	data, err := cache.GetAllHourlyTraffic(ctx, currentHour)
	require.NoError(t, err)
	assert.Len(t, data, 2)

	// Cleanup
	err = cache.CleanupHour(ctx, currentHour)
	require.NoError(t, err)

	// Verify data is gone
	data, err = cache.GetAllHourlyTraffic(ctx, currentHour)
	require.NoError(t, err)
	assert.Len(t, data, 0)

	// Verify individual keys are also gone
	upload, download, err := cache.GetHourlyTraffic(ctx, currentHour, 1, "node", 100)
	require.NoError(t, err)
	assert.Equal(t, int64(0), upload)
	assert.Equal(t, int64(0), download)
}

func TestRedisHourlyTrafficCache_GetHourlyTrafficRange(t *testing.T) {
	// Initialize biztime
	biztime.MustInit("Asia/Shanghai")

	client, cleanup := setupTestRedis(t)
	defer cleanup()

	log := newNopLogger()

	// Create a custom cache to directly manipulate Redis keys
	realCache := &RedisHourlyTrafficCache{
		client: client,
		logger: log,
	}
	ctx := context.Background()

	// Manually set up data for different hours
	baseTime := time.Date(2025, 1, 7, 0, 0, 0, 0, time.UTC)

	// Set data for 3 consecutive hours
	for i := 0; i < 3; i++ {
		hourTime := baseTime.Add(time.Duration(i) * time.Hour)
		hourKey := formatHourKey(hourTime)
		trafficKey := hourlyTrafficKey(hourKey, 1, "node", 100)
		activeKey := hourlyActiveSetKey(hourKey)

		client.HSet(ctx, trafficKey, hourlyFieldUpload, int64((i+1)*100))
		client.HSet(ctx, trafficKey, hourlyFieldDownload, int64((i+1)*200))
		client.SAdd(ctx, activeKey, trafficKey)
	}

	// Query range
	from := baseTime
	to := baseTime.Add(2 * time.Hour)

	points, err := realCache.GetHourlyTrafficRange(ctx, 1, "node", 100, from, to)
	require.NoError(t, err)
	assert.Len(t, points, 3)

	// Verify data
	assert.Equal(t, int64(100), points[0].Upload)
	assert.Equal(t, int64(200), points[0].Download)
	assert.Equal(t, int64(200), points[1].Upload)
	assert.Equal(t, int64(400), points[1].Download)
	assert.Equal(t, int64(300), points[2].Upload)
	assert.Equal(t, int64(600), points[2].Download)
}

func TestRedisHourlyTrafficCache_CleanupNonExistentHour(t *testing.T) {
	// Initialize biztime
	biztime.MustInit("Asia/Shanghai")

	client, cleanup := setupTestRedis(t)
	defer cleanup()

	log := newNopLogger()
	cache := NewRedisHourlyTrafficCache(client, log)
	ctx := context.Background()

	// Cleanup non-existent hour should not error
	pastHour := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	err := cache.CleanupHour(ctx, pastHour)
	require.NoError(t, err)
}

func TestRedisHourlyTrafficCache_GetAndCleanupHour(t *testing.T) {
	// Initialize biztime
	biztime.MustInit("Asia/Shanghai")

	client, cleanup := setupTestRedis(t)
	defer cleanup()

	log := newNopLogger()
	cache := NewRedisHourlyTrafficCache(client, log)
	ctx := context.Background()

	// Add traffic for multiple subscriptions/resources
	err := cache.IncrementHourlyTraffic(ctx, 1, "node", 100, 1000, 2000)
	require.NoError(t, err)

	err = cache.IncrementHourlyTraffic(ctx, 2, "node", 101, 500, 600)
	require.NoError(t, err)

	// Verify data exists
	currentHour := biztime.TruncateToHourInBiz(biztime.NowUTC())
	data, err := cache.GetAllHourlyTraffic(ctx, currentHour)
	require.NoError(t, err)
	assert.Len(t, data, 2)

	// Add more traffic
	err = cache.IncrementHourlyTraffic(ctx, 1, "forward", 200, 300, 400)
	require.NoError(t, err)

	// GetAndCleanupHour should return all data and clean up
	data, err = cache.GetAndCleanupHour(ctx, currentHour)
	require.NoError(t, err)
	assert.Len(t, data, 3)

	// Verify data (order may vary)
	dataMap := make(map[string]HourlyTrafficData)
	for _, d := range data {
		key := hourlyTrafficKey(formatHourKey(currentHour), d.SubscriptionID, d.ResourceType, d.ResourceID)
		dataMap[key] = d
	}

	// Check entries
	key1 := hourlyTrafficKey(formatHourKey(currentHour), 1, "node", 100)
	assert.Equal(t, int64(1000), dataMap[key1].Upload)
	assert.Equal(t, int64(2000), dataMap[key1].Download)

	key2 := hourlyTrafficKey(formatHourKey(currentHour), 2, "node", 101)
	assert.Equal(t, int64(500), dataMap[key2].Upload)
	assert.Equal(t, int64(600), dataMap[key2].Download)

	key3 := hourlyTrafficKey(formatHourKey(currentHour), 1, "forward", 200)
	assert.Equal(t, int64(300), dataMap[key3].Upload)
	assert.Equal(t, int64(400), dataMap[key3].Download)

	// Verify data is cleaned up
	data, err = cache.GetAllHourlyTraffic(ctx, currentHour)
	require.NoError(t, err)
	assert.Len(t, data, 0)

	// Verify individual keys are also gone
	upload, download, err := cache.GetHourlyTraffic(ctx, currentHour, 1, "node", 100)
	require.NoError(t, err)
	assert.Equal(t, int64(0), upload)
	assert.Equal(t, int64(0), download)
}

func TestRedisHourlyTrafficCache_GetAndCleanupHour_Empty(t *testing.T) {
	// Initialize biztime
	biztime.MustInit("Asia/Shanghai")

	client, cleanup := setupTestRedis(t)
	defer cleanup()

	log := newNopLogger()
	cache := NewRedisHourlyTrafficCache(client, log)
	ctx := context.Background()

	// GetAndCleanupHour on empty hour should return nil without error
	pastHour := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	data, err := cache.GetAndCleanupHour(ctx, pastHour)
	require.NoError(t, err)
	assert.Nil(t, data)
}

func TestRedisHourlyTrafficCache_InvalidResourceType(t *testing.T) {
	// Initialize biztime
	biztime.MustInit("Asia/Shanghai")

	client, cleanup := setupTestRedis(t)
	defer cleanup()

	log := newNopLogger()
	cache := NewRedisHourlyTrafficCache(client, log)
	ctx := context.Background()

	// Test IncrementHourlyTraffic with invalid resource type
	err := cache.IncrementHourlyTraffic(ctx, 1, "node:invalid", 100, 1000, 2000)
	assert.ErrorIs(t, err, ErrInvalidResourceType)

	// Test GetHourlyTraffic with invalid resource type
	_, _, err = cache.GetHourlyTraffic(ctx, time.Now(), 1, "type:with:colons", 100)
	assert.ErrorIs(t, err, ErrInvalidResourceType)

	// Test GetHourlyTrafficRange with invalid resource type
	_, err = cache.GetHourlyTrafficRange(ctx, 1, "bad:type", 100, time.Now(), time.Now())
	assert.ErrorIs(t, err, ErrInvalidResourceType)

	// Valid resource types should work
	err = cache.IncrementHourlyTraffic(ctx, 1, "node", 100, 1000, 2000)
	assert.NoError(t, err)

	err = cache.IncrementHourlyTraffic(ctx, 1, "forward_rule", 100, 1000, 2000)
	assert.NoError(t, err)
}

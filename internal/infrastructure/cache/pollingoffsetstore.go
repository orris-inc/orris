package cache

import (
	"context"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

const pollingOffsetKey = "telegram:polling:offset"

// PollingOffsetStore persists the Telegram polling offset across restarts.
type PollingOffsetStore struct {
	client *redis.Client
}

// NewPollingOffsetStore creates a new PollingOffsetStore instance.
func NewPollingOffsetStore(client *redis.Client) *PollingOffsetStore {
	return &PollingOffsetStore{client: client}
}

// GetOffset returns the last saved offset, or 0 if not found.
func (s *PollingOffsetStore) GetOffset(ctx context.Context) (int64, error) {
	val, err := s.client.Get(ctx, pollingOffsetKey).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get polling offset: %w", err)
	}

	offset, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse polling offset: %w", err)
	}

	return offset, nil
}

// SaveOffset persists the current offset.
func (s *PollingOffsetStore) SaveOffset(ctx context.Context, offset int64) error {
	val := strconv.FormatInt(offset, 10)
	if err := s.client.Set(ctx, pollingOffsetKey, val, 0).Err(); err != nil {
		return fmt.Errorf("failed to save polling offset: %w", err)
	}
	return nil
}

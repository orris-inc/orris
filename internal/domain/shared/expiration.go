package shared

import (
	"time"

	"github.com/orris-inc/orris/internal/shared/biztime"
)

// IsExpired checks if the given expiration time has passed.
// Returns false if expiresAt is nil (never expires).
func IsExpired(expiresAt *time.Time) bool {
	if expiresAt == nil {
		return false
	}
	return biztime.NowUTC().After(*expiresAt)
}

// IsExpiringSoon checks if the given expiration time is within the specified number of days.
// Returns false if expiresAt is nil (never expires).
func IsExpiringSoon(expiresAt *time.Time, days int) bool {
	if expiresAt == nil {
		return false
	}
	threshold := biztime.NowUTC().AddDate(0, 0, days)
	return expiresAt.Before(threshold)
}

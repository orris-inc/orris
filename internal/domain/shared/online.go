package shared

import "time"

// DefaultOnlineTimeout is the duration after which an entity is considered offline.
const DefaultOnlineTimeout = 5 * time.Minute

// IsOnline checks if the entity is online based on its last seen timestamp.
// Returns false if lastSeenAt is nil.
func IsOnline(lastSeenAt *time.Time) bool {
	if lastSeenAt == nil {
		return false
	}
	return time.Since(*lastSeenAt) < DefaultOnlineTimeout
}

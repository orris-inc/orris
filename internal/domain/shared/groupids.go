// Package shared provides reusable domain logic shared across aggregates.
package shared

// AddToGroupIDs adds a group ID to the slice if not already present.
// Returns the updated slice and true if the ID was added.
func AddToGroupIDs(ids []uint, id uint) ([]uint, bool) {
	for _, existing := range ids {
		if existing == id {
			return ids, false
		}
	}
	return append(ids, id), true
}

// RemoveFromGroupIDs removes a group ID from the slice.
// Returns the updated slice and true if the ID was removed.
func RemoveFromGroupIDs(ids []uint, id uint) ([]uint, bool) {
	for i, existing := range ids {
		if existing == id {
			return append(ids[:i], ids[i+1:]...), true
		}
	}
	return ids, false
}

// HasGroupID checks if a group ID exists in the slice.
func HasGroupID(ids []uint, id uint) bool {
	for _, existing := range ids {
		if existing == id {
			return true
		}
	}
	return false
}

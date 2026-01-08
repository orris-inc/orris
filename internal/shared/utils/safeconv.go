package utils

import "math"

// SafeUint64ToInt64 safely converts uint64 to int64.
// If the value exceeds math.MaxInt64, it returns math.MaxInt64.
func SafeUint64ToInt64(v uint64) int64 {
	if v > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(v)
}

// SafeInt64ToUint64 safely converts int64 to uint64.
// If the value is negative, it returns 0.
func SafeInt64ToUint64(v int64) uint64 {
	if v < 0 {
		return 0
	}
	return uint64(v)
}

// Package jsonutil provides JSON conversion utilities.
package jsonutil

import (
	"fmt"
	"strings"
)

// UintSliceToJSONArray converts a slice of uints to a JSON array string.
// Returns "[]" for empty or nil slices.
//
// Example:
//
//	[]uint{1, 2, 3} -> "[1,2,3]"
//	[]uint{}        -> "[]"
//	nil             -> "[]"
func UintSliceToJSONArray(ids []uint) string {
	if len(ids) == 0 {
		return "[]"
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprintf("%d", id)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

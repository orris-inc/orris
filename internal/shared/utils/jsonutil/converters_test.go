package jsonutil

import (
	"math"
	"testing"
)

// TestUintSliceToJSONArray tests UintSliceToJSONArray function with various inputs.
func TestUintSliceToJSONArray(t *testing.T) {
	tests := []struct {
		name     string
		input    []uint
		expected string
	}{
		{
			name:     "empty slice",
			input:    []uint{},
			expected: "[]",
		},
		{
			name:     "nil slice",
			input:    nil,
			expected: "[]",
		},
		{
			name:     "single element",
			input:    []uint{1},
			expected: "[1]",
		},
		{
			name:     "multiple elements",
			input:    []uint{1, 2, 3},
			expected: "[1,2,3]",
		},
		{
			name:     "zero value",
			input:    []uint{0},
			expected: "[0]",
		},
		{
			name:     "mixed values with zero",
			input:    []uint{0, 1, 2},
			expected: "[0,1,2]",
		},
		{
			name:     "large values",
			input:    []uint{math.MaxUint32, math.MaxUint32 - 1},
			expected: "[4294967295,4294967294]",
		},
		{
			name:     "max uint64 on 64-bit system",
			input:    []uint{math.MaxUint64},
			expected: "[18446744073709551615]",
		},
		{
			name:     "many elements",
			input:    []uint{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			expected: "[1,2,3,4,5,6,7,8,9,10]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UintSliceToJSONArray(tt.input)
			if result != tt.expected {
				t.Errorf("UintSliceToJSONArray(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

package setutil

import (
	"sort"
	"testing"
)

// TestNewUintSet verifies that NewUintSet creates an empty set.
func TestNewUintSet(t *testing.T) {
	s := NewUintSet()

	if s == nil {
		t.Fatal("NewUintSet() returned nil")
	}
	if s.Len() != 0 {
		t.Errorf("NewUintSet().Len() = %d, want 0", s.Len())
	}
}

// TestNewUintSetWithCap verifies that NewUintSetWithCap creates an empty set with capacity.
func TestNewUintSetWithCap(t *testing.T) {
	tests := []struct {
		name string
		cap  int
	}{
		{"zero capacity", 0},
		{"small capacity", 10},
		{"large capacity", 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewUintSetWithCap(tt.cap)

			if s == nil {
				t.Fatal("NewUintSetWithCap() returned nil")
			}
			if s.Len() != 0 {
				t.Errorf("NewUintSetWithCap(%d).Len() = %d, want 0", tt.cap, s.Len())
			}
		})
	}
}

// TestAdd verifies Add behavior for single elements.
func TestAdd(t *testing.T) {
	tests := []struct {
		name     string
		ids      []uint
		wantLen  int
		checkHas []uint
	}{
		{
			name:     "add single element",
			ids:      []uint{1},
			wantLen:  1,
			checkHas: []uint{1},
		},
		{
			name:     "add multiple distinct elements",
			ids:      []uint{1, 2, 3},
			wantLen:  3,
			checkHas: []uint{1, 2, 3},
		},
		{
			name:     "add duplicate elements",
			ids:      []uint{1, 1, 1},
			wantLen:  1,
			checkHas: []uint{1},
		},
		{
			name:     "add zero value",
			ids:      []uint{0},
			wantLen:  1,
			checkHas: []uint{0},
		},
		{
			name:     "add mixed with duplicates",
			ids:      []uint{5, 3, 5, 1, 3},
			wantLen:  3,
			checkHas: []uint{1, 3, 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewUintSet()

			for _, id := range tt.ids {
				s.Add(id)
			}

			if got := s.Len(); got != tt.wantLen {
				t.Errorf("Len() = %d, want %d", got, tt.wantLen)
			}

			for _, id := range tt.checkHas {
				if !s.Has(id) {
					t.Errorf("Has(%d) = false, want true", id)
				}
			}
		})
	}
}

// TestAddAll verifies AddAll behavior for batch operations.
func TestAddAll(t *testing.T) {
	tests := []struct {
		name     string
		ids      []uint
		wantLen  int
		checkHas []uint
	}{
		{
			name:     "add empty slice",
			ids:      []uint{},
			wantLen:  0,
			checkHas: []uint{},
		},
		{
			name:     "add nil slice",
			ids:      nil,
			wantLen:  0,
			checkHas: []uint{},
		},
		{
			name:     "add single element",
			ids:      []uint{42},
			wantLen:  1,
			checkHas: []uint{42},
		},
		{
			name:     "add multiple distinct elements",
			ids:      []uint{1, 2, 3, 4, 5},
			wantLen:  5,
			checkHas: []uint{1, 2, 3, 4, 5},
		},
		{
			name:     "add with duplicates",
			ids:      []uint{1, 2, 2, 3, 3, 3},
			wantLen:  3,
			checkHas: []uint{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewUintSet()
			s.AddAll(tt.ids)

			if got := s.Len(); got != tt.wantLen {
				t.Errorf("Len() = %d, want %d", got, tt.wantLen)
			}

			for _, id := range tt.checkHas {
				if !s.Has(id) {
					t.Errorf("Has(%d) = false, want true", id)
				}
			}
		})
	}
}

// TestAddAllMultipleCalls verifies AddAll with multiple calls.
func TestAddAllMultipleCalls(t *testing.T) {
	s := NewUintSet()

	s.AddAll([]uint{1, 2, 3})
	s.AddAll([]uint{3, 4, 5})
	s.AddAll([]uint{5, 6})

	wantLen := 6
	if got := s.Len(); got != wantLen {
		t.Errorf("Len() = %d, want %d", got, wantLen)
	}

	for i := uint(1); i <= 6; i++ {
		if !s.Has(i) {
			t.Errorf("Has(%d) = false, want true", i)
		}
	}
}

// TestHas verifies Has behavior.
func TestHas(t *testing.T) {
	s := NewUintSet()
	s.AddAll([]uint{1, 5, 10, 100})

	tests := []struct {
		name string
		id   uint
		want bool
	}{
		{"existing element 1", 1, true},
		{"existing element 5", 5, true},
		{"existing element 10", 10, true},
		{"existing element 100", 100, true},
		{"non-existing element 0", 0, false},
		{"non-existing element 2", 2, false},
		{"non-existing element 50", 50, false},
		{"non-existing element 999", 999, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.Has(tt.id); got != tt.want {
				t.Errorf("Has(%d) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

// TestHasOnEmptySet verifies Has behavior on empty set.
func TestHasOnEmptySet(t *testing.T) {
	s := NewUintSet()

	if s.Has(0) {
		t.Error("Has(0) on empty set = true, want false")
	}
	if s.Has(1) {
		t.Error("Has(1) on empty set = true, want false")
	}
	if s.Has(100) {
		t.Error("Has(100) on empty set = true, want false")
	}
}

// TestToSlice verifies ToSlice behavior.
func TestToSlice(t *testing.T) {
	tests := []struct {
		name string
		ids  []uint
		want []uint
	}{
		{
			name: "empty set",
			ids:  []uint{},
			want: []uint{},
		},
		{
			name: "single element",
			ids:  []uint{42},
			want: []uint{42},
		},
		{
			name: "multiple elements",
			ids:  []uint{3, 1, 4, 1, 5, 9, 2, 6},
			want: []uint{1, 2, 3, 4, 5, 6, 9},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewUintSet()
			s.AddAll(tt.ids)

			got := s.ToSlice()

			if len(got) != len(tt.want) {
				t.Errorf("ToSlice() length = %d, want %d", len(got), len(tt.want))
				return
			}

			// Sort both slices for comparison since order is not guaranteed
			sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })

			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("ToSlice()[%d] = %d, want %d", i, v, tt.want[i])
				}
			}
		})
	}
}

// TestLen verifies Len behavior.
func TestLen(t *testing.T) {
	tests := []struct {
		name string
		ids  []uint
		want int
	}{
		{"empty set", []uint{}, 0},
		{"single element", []uint{1}, 1},
		{"multiple distinct elements", []uint{1, 2, 3, 4, 5}, 5},
		{"with duplicates", []uint{1, 1, 2, 2, 3, 3}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewUintSet()
			s.AddAll(tt.ids)

			if got := s.Len(); got != tt.want {
				t.Errorf("Len() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestAddAndAddAllCombined verifies Add and AddAll work together correctly.
func TestAddAndAddAllCombined(t *testing.T) {
	s := NewUintSet()

	s.Add(1)
	s.AddAll([]uint{2, 3})
	s.Add(3) // duplicate
	s.AddAll([]uint{4, 1}) // 1 is duplicate
	s.Add(5)

	wantLen := 5
	if got := s.Len(); got != wantLen {
		t.Errorf("Len() = %d, want %d", got, wantLen)
	}

	for i := uint(1); i <= 5; i++ {
		if !s.Has(i) {
			t.Errorf("Has(%d) = false, want true", i)
		}
	}
}

// TestLargeSet verifies behavior with large number of elements.
func TestLargeSet(t *testing.T) {
	s := NewUintSetWithCap(10000)

	// Add 10000 elements
	for i := uint(0); i < 10000; i++ {
		s.Add(i)
	}

	if got := s.Len(); got != 10000 {
		t.Errorf("Len() = %d, want 10000", got)
	}

	// Verify some elements
	for i := uint(0); i < 10000; i += 1000 {
		if !s.Has(i) {
			t.Errorf("Has(%d) = false, want true", i)
		}
	}

	// Verify non-existing element
	if s.Has(10001) {
		t.Error("Has(10001) = true, want false")
	}
}

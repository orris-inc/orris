// Package setutil provides generic set utilities for common ID collection patterns.
package setutil

// UintSet is a set of uint values.
// It uses map[uint]struct{} internally for memory efficiency.
type UintSet struct {
	items map[uint]struct{}
}

// NewUintSet creates a new empty UintSet.
func NewUintSet() *UintSet {
	return &UintSet{
		items: make(map[uint]struct{}),
	}
}

// NewUintSetWithCap creates a new UintSet with initial capacity.
func NewUintSetWithCap(cap int) *UintSet {
	return &UintSet{
		items: make(map[uint]struct{}, cap),
	}
}

// Add adds an id to the set.
func (s *UintSet) Add(id uint) {
	s.items[id] = struct{}{}
}

// AddAll adds all ids to the set.
func (s *UintSet) AddAll(ids []uint) {
	for _, id := range ids {
		s.items[id] = struct{}{}
	}
}

// Has returns true if the id exists in the set.
func (s *UintSet) Has(id uint) bool {
	_, ok := s.items[id]
	return ok
}

// ToSlice returns all ids as a slice.
// The order is not guaranteed.
func (s *UintSet) ToSlice() []uint {
	result := make([]uint, 0, len(s.items))
	for id := range s.items {
		result = append(result, id)
	}
	return result
}

// Len returns the number of elements in the set.
func (s *UintSet) Len() int {
	return len(s.items)
}

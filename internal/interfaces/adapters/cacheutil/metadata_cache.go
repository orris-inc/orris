// Package cacheutil provides common cache utilities for infrastructure adapters.
package cacheutil

import (
	"sync"
	"time"

	"github.com/orris-inc/orris/internal/shared/biztime"
)

// MetadataCache provides thread-safe caching for metadata with SID lookup.
// It uses a double-checked locking pattern for efficient cache refresh.
type MetadataCache[T any] struct {
	items       map[uint]*T
	sidToID     map[string]uint
	mu          sync.RWMutex
	lastUpdated time.Time
	ttl         time.Duration
}

// NewMetadataCache creates a new MetadataCache with the specified TTL.
func NewMetadataCache[T any](ttl time.Duration) *MetadataCache[T] {
	return &MetadataCache[T]{
		items:   make(map[uint]*T),
		sidToID: make(map[string]uint),
		ttl:     ttl,
	}
}

// TryRefresh attempts to refresh the cache using double-checked locking.
// Returns true if the caller should perform the refresh (lock is held).
// Caller must call FinishRefresh or AbortRefresh after this returns true.
func (c *MetadataCache[T]) TryRefresh() bool {
	// First check with read lock
	c.mu.RLock()
	needsRefresh := biztime.NowUTC().Sub(c.lastUpdated) > c.ttl
	c.mu.RUnlock()

	if !needsRefresh {
		return false
	}

	// Acquire write lock
	c.mu.Lock()

	// Double-check after acquiring write lock
	if biztime.NowUTC().Sub(c.lastUpdated) <= c.ttl {
		c.mu.Unlock()
		return false
	}

	// Caller should perform refresh while holding the lock
	return true
}

// FinishRefresh completes a refresh operation by updating the cache with new items.
// Must only be called after TryRefresh returns true.
// The getID and getSID functions extract the ID and SID from each item.
func (c *MetadataCache[T]) FinishRefresh(items []*T, getID func(*T) uint, getSID func(*T) string) {
	defer c.mu.Unlock()

	c.items = make(map[uint]*T, len(items))
	c.sidToID = make(map[string]uint, len(items))
	for _, item := range items {
		id := getID(item)
		sid := getSID(item)
		c.items[id] = item
		c.sidToID[sid] = id
	}
	c.lastUpdated = biztime.NowUTC()
}

// AbortRefresh releases the write lock without updating the cache.
// Must only be called after TryRefresh returns true.
func (c *MetadataCache[T]) AbortRefresh() {
	c.mu.Unlock()
}

// CacheResult holds the result of GetBySIDs operation.
type CacheResult[T any] struct {
	IDs   []uint
	Items []*T
}

// GetBySIDs returns items by SIDs. If sidList is nil, returns all items.
// Returns the internal IDs and corresponding items.
func (c *MetadataCache[T]) GetBySIDs(sidList []string) CacheResult[T] {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if sidList == nil {
		// Return all items
		ids := make([]uint, 0, len(c.items))
		items := make([]*T, 0, len(c.items))
		for id, item := range c.items {
			ids = append(ids, id)
			items = append(items, item)
		}
		return CacheResult[T]{IDs: ids, Items: items}
	}

	// Return specific items
	ids := make([]uint, 0, len(sidList))
	items := make([]*T, 0, len(sidList))
	for _, sid := range sidList {
		if id, ok := c.sidToID[sid]; ok {
			ids = append(ids, id)
			if item, ok := c.items[id]; ok {
				items = append(items, item)
			}
		}
	}
	return CacheResult[T]{IDs: ids, Items: items}
}

// BuildIDMap returns a map from ID to item for quick lookup.
// This creates a new map from the provided items slice.
func BuildIDMap[T any](items []*T, getID func(*T) uint) map[uint]*T {
	result := make(map[uint]*T, len(items))
	for _, item := range items {
		result[getID(item)] = item
	}
	return result
}


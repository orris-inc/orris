package cacheutil

import (
	"sync"
	"testing"
	"time"
)

// testMetadata is a simple struct for testing purposes.
type testMetadata struct {
	ID   uint
	SID  string
	Name string
}

func getTestID(t *testMetadata) uint     { return t.ID }
func getTestSID(t *testMetadata) string  { return t.SID }

func TestNewMetadataCache(t *testing.T) {
	ttl := 5 * time.Minute
	cache := NewMetadataCache[testMetadata](ttl)

	if cache == nil {
		t.Fatal("NewMetadataCache returned nil")
	}
	if cache.ttl != ttl {
		t.Errorf("expected ttl %v, got %v", ttl, cache.ttl)
	}
	if cache.items == nil {
		t.Error("items map should be initialized")
	}
	if cache.sidToID == nil {
		t.Error("sidToID map should be initialized")
	}
}

func TestTryRefresh_NewCacheNeedsRefresh(t *testing.T) {
	// A new cache should need refresh immediately since lastUpdated is zero.
	cache := NewMetadataCache[testMetadata](5 * time.Minute)

	needsRefresh := cache.TryRefresh()
	if !needsRefresh {
		t.Error("new cache should need refresh")
	}

	// Must release the lock after TryRefresh returns true.
	cache.AbortRefresh()
}

func TestTryRefresh_NotExpired(t *testing.T) {
	cache := NewMetadataCache[testMetadata](5 * time.Minute)

	// Perform initial refresh.
	if !cache.TryRefresh() {
		t.Fatal("new cache should need refresh")
	}
	cache.FinishRefresh([]*testMetadata{}, getTestID, getTestSID)

	// Immediately try again - should not need refresh.
	needsRefresh := cache.TryRefresh()
	if needsRefresh {
		t.Error("cache should not need refresh before TTL expires")
		cache.AbortRefresh()
	}
}

func TestTryRefresh_Expired(t *testing.T) {
	// Use a very short TTL for testing.
	cache := NewMetadataCache[testMetadata](1 * time.Millisecond)

	// Perform initial refresh.
	if !cache.TryRefresh() {
		t.Fatal("new cache should need refresh")
	}
	cache.FinishRefresh([]*testMetadata{}, getTestID, getTestSID)

	// Wait for TTL to expire.
	time.Sleep(5 * time.Millisecond)

	// Should need refresh now.
	needsRefresh := cache.TryRefresh()
	if !needsRefresh {
		t.Error("cache should need refresh after TTL expires")
	} else {
		cache.AbortRefresh()
	}
}

func TestFinishRefresh(t *testing.T) {
	cache := NewMetadataCache[testMetadata](5 * time.Minute)

	items := []*testMetadata{
		{ID: 1, SID: "test_1", Name: "Item 1"},
		{ID: 2, SID: "test_2", Name: "Item 2"},
		{ID: 3, SID: "test_3", Name: "Item 3"},
	}

	if !cache.TryRefresh() {
		t.Fatal("new cache should need refresh")
	}
	cache.FinishRefresh(items, getTestID, getTestSID)

	// Verify items were stored.
	result := cache.GetBySIDs(nil)
	if len(result.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(result.Items))
	}
}

func TestAbortRefresh(t *testing.T) {
	cache := NewMetadataCache[testMetadata](5 * time.Minute)

	if !cache.TryRefresh() {
		t.Fatal("new cache should need refresh")
	}
	cache.AbortRefresh()

	// Cache should still need refresh since we aborted.
	needsRefresh := cache.TryRefresh()
	if !needsRefresh {
		t.Error("cache should still need refresh after abort")
	} else {
		cache.AbortRefresh()
	}
}

func TestGetBySIDs_NilReturnsAll(t *testing.T) {
	cache := NewMetadataCache[testMetadata](5 * time.Minute)

	items := []*testMetadata{
		{ID: 1, SID: "test_1", Name: "Item 1"},
		{ID: 2, SID: "test_2", Name: "Item 2"},
		{ID: 3, SID: "test_3", Name: "Item 3"},
	}

	if !cache.TryRefresh() {
		t.Fatal("new cache should need refresh")
	}
	cache.FinishRefresh(items, getTestID, getTestSID)

	result := cache.GetBySIDs(nil)
	if len(result.IDs) != 3 {
		t.Errorf("expected 3 IDs, got %d", len(result.IDs))
	}
	if len(result.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(result.Items))
	}
}

func TestGetBySIDs_SpecificSIDs(t *testing.T) {
	cache := NewMetadataCache[testMetadata](5 * time.Minute)

	items := []*testMetadata{
		{ID: 1, SID: "test_1", Name: "Item 1"},
		{ID: 2, SID: "test_2", Name: "Item 2"},
		{ID: 3, SID: "test_3", Name: "Item 3"},
	}

	if !cache.TryRefresh() {
		t.Fatal("new cache should need refresh")
	}
	cache.FinishRefresh(items, getTestID, getTestSID)

	tests := []struct {
		name        string
		sids        []string
		wantIDs     []uint
		wantCount   int
	}{
		{
			name:      "single existing SID",
			sids:      []string{"test_1"},
			wantIDs:   []uint{1},
			wantCount: 1,
		},
		{
			name:      "multiple existing SIDs",
			sids:      []string{"test_1", "test_3"},
			wantIDs:   []uint{1, 3},
			wantCount: 2,
		},
		{
			name:      "empty slice",
			sids:      []string{},
			wantIDs:   []uint{},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cache.GetBySIDs(tt.sids)
			if len(result.Items) != tt.wantCount {
				t.Errorf("expected %d items, got %d", tt.wantCount, len(result.Items))
			}
			if len(result.IDs) != tt.wantCount {
				t.Errorf("expected %d IDs, got %d", tt.wantCount, len(result.IDs))
			}
		})
	}
}

func TestGetBySIDs_NonExistentSIDs(t *testing.T) {
	cache := NewMetadataCache[testMetadata](5 * time.Minute)

	items := []*testMetadata{
		{ID: 1, SID: "test_1", Name: "Item 1"},
		{ID: 2, SID: "test_2", Name: "Item 2"},
	}

	if !cache.TryRefresh() {
		t.Fatal("new cache should need refresh")
	}
	cache.FinishRefresh(items, getTestID, getTestSID)

	tests := []struct {
		name      string
		sids      []string
		wantCount int
	}{
		{
			name:      "single non-existent SID",
			sids:      []string{"non_existent"},
			wantCount: 0,
		},
		{
			name:      "mixed existing and non-existent SIDs",
			sids:      []string{"test_1", "non_existent", "test_2"},
			wantCount: 2,
		},
		{
			name:      "all non-existent SIDs",
			sids:      []string{"foo", "bar", "baz"},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cache.GetBySIDs(tt.sids)
			if len(result.Items) != tt.wantCount {
				t.Errorf("expected %d items, got %d", tt.wantCount, len(result.Items))
			}
		})
	}
}

func TestBuildIDMap(t *testing.T) {
	items := []*testMetadata{
		{ID: 1, SID: "test_1", Name: "Item 1"},
		{ID: 2, SID: "test_2", Name: "Item 2"},
		{ID: 3, SID: "test_3", Name: "Item 3"},
	}

	idMap := BuildIDMap(items, getTestID)

	if len(idMap) != 3 {
		t.Errorf("expected map size 3, got %d", len(idMap))
	}

	for _, item := range items {
		if mapped, ok := idMap[item.ID]; !ok {
			t.Errorf("ID %d not found in map", item.ID)
		} else if mapped != item {
			t.Errorf("mapped item mismatch for ID %d", item.ID)
		}
	}
}

func TestBuildIDMap_Empty(t *testing.T) {
	items := []*testMetadata{}
	idMap := BuildIDMap(items, getTestID)

	if len(idMap) != 0 {
		t.Errorf("expected empty map, got %d items", len(idMap))
	}
}

func TestBuildIDMap_Nil(t *testing.T) {
	var items []*testMetadata
	idMap := BuildIDMap(items, getTestID)

	if idMap == nil {
		t.Error("expected non-nil map")
	}
	if len(idMap) != 0 {
		t.Errorf("expected empty map, got %d items", len(idMap))
	}
}

func TestConcurrentAccess(t *testing.T) {
	cache := NewMetadataCache[testMetadata](1 * time.Millisecond)

	items := []*testMetadata{
		{ID: 1, SID: "test_1", Name: "Item 1"},
		{ID: 2, SID: "test_2", Name: "Item 2"},
		{ID: 3, SID: "test_3", Name: "Item 3"},
	}

	// Initial refresh.
	if cache.TryRefresh() {
		cache.FinishRefresh(items, getTestID, getTestSID)
	}

	var wg sync.WaitGroup
	goroutines := 10
	iterations := 100

	// Start multiple goroutines that read and potentially refresh.
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// Read operation.
				result := cache.GetBySIDs(nil)
				_ = result.Items

				// Potential refresh operation.
				if cache.TryRefresh() {
					cache.FinishRefresh(items, getTestID, getTestSID)
				}
			}
		}()
	}

	wg.Wait()
}

func TestDoubleCheckedLocking(t *testing.T) {
	cache := NewMetadataCache[testMetadata](1 * time.Millisecond)

	items := []*testMetadata{
		{ID: 1, SID: "test_1", Name: "Item 1"},
	}

	// Initial refresh.
	if cache.TryRefresh() {
		cache.FinishRefresh(items, getTestID, getTestSID)
	}

	// Wait for TTL to expire.
	time.Sleep(5 * time.Millisecond)

	refreshCount := 0
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Start multiple goroutines that try to refresh.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if cache.TryRefresh() {
				mu.Lock()
				refreshCount++
				mu.Unlock()
				cache.FinishRefresh(items, getTestID, getTestSID)
			}
		}()
	}

	wg.Wait()

	// Due to double-checked locking, only one goroutine should have performed the refresh.
	if refreshCount != 1 {
		t.Errorf("expected exactly 1 refresh, got %d", refreshCount)
	}
}

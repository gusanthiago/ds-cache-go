package lru

import (
	"strconv"
	"sync"
	"testing"
)

// TestPutCreatesAndUpdates tests the Put function of Cache.
func TestPutCreatesAndUpdates(t *testing.T) {
	c := New("test", 2)

	// Test adding a new key
	if created := c.Put("key1", "value1"); !created {
		t.Errorf("Put(new key) created = false; want true")
	}

	// Test updating an existing key
	if created := c.Put("key1", "value2"); created {
		t.Errorf("Put(existing key) created = true; want false")
	}

	value, ok := c.Get("key1")
	if !ok || value != "value2" {
		t.Errorf("Get(key1) = (%q, %v); want (\"value2\", true)", value, ok)
	}
	if size := c.Stats().Size; size != 1 {
		t.Errorf("Size after update = %d; want 1", size)
	}
}

// TestGetMissingKey tests the Get function of Cache with a non-existing key.
func TestGetMissingKey(t *testing.T) {
	c := New("test", 2)

	if value, ok := c.Get("absent"); ok || value != "" {
		t.Errorf("Get(absent) = (%q, %v); want (\"\", false)", value, ok)
	}
}

// TestEvictsLeastRecentlyUsed tests that a full cache evicts the coldest key.
func TestEvictsLeastRecentlyUsed(t *testing.T) {
	c := New("test", 2)
	c.Put("key1", "value1")
	c.Put("key2", "value2")

	// Reading key1 makes key2 the coldest key, so key2 is evicted next
	if _, ok := c.Get("key1"); !ok {
		t.Fatal("Get(key1) missed before eviction")
	}
	c.Put("key3", "value3")

	if _, ok := c.Get("key2"); ok {
		t.Error("key2 survived; want it evicted as least recently used")
	}
	if _, ok := c.Get("key1"); !ok {
		t.Error("key1 was evicted; want it retained after being read")
	}
	if evictions := c.Stats().Evictions; evictions != 1 {
		t.Errorf("Evictions = %d; want 1", evictions)
	}
}

// TestDelete tests the Delete function of Cache.
func TestDelete(t *testing.T) {
	c := New("test", 2)
	c.Put("key1", "value1")

	if removed := c.Delete("key1"); !removed {
		t.Error("Delete(existing) = false; want true")
	}
	if removed := c.Delete("key1"); removed {
		t.Error("Delete(absent) = true; want false")
	}
	if size := c.Stats().Size; size != 0 {
		t.Errorf("Size after delete = %d; want 0", size)
	}

	// Test that a deleted key frees its slot
	c.Put("key2", "value2")
	c.Put("key3", "value3")
	if _, ok := c.Get("key2"); !ok {
		t.Error("key2 evicted early; delete did not free its slot")
	}
}

// TestCapacityFloor tests that a capacity below 1 is raised to 1.
func TestCapacityFloor(t *testing.T) {
	c := New("test", 0)

	if capacity := c.Stats().Capacity; capacity != 1 {
		t.Errorf("Capacity for New(0) = %d; want 1", capacity)
	}

	c.Put("key1", "value1")
	if _, ok := c.Get("key1"); !ok {
		t.Error("cache built with capacity 0 cannot hold a single entry")
	}
}

// TestHitAndMissCounters tests the Stats function of Cache.
func TestHitAndMissCounters(t *testing.T) {
	c := New("test", 4)
	c.Put("key1", "value1")

	c.Get("key1")
	c.Get("key1")
	c.Get("absent")

	stats := c.Stats()
	if stats.Hits != 2 {
		t.Errorf("Hits = %d; want 2", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Misses = %d; want 1", stats.Misses)
	}
}

// TestConcurrentAccess tests reads racing writes. Run with -race, it fails if
// Get stops locking while it reorders the list.
func TestConcurrentAccess(t *testing.T) {
	c := New("test", 64)

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				key := "key" + strconv.Itoa(j%128)
				c.Put(key, strconv.Itoa(worker))
				c.Get(key)
				c.Delete(key)
			}
		}(i)
	}
	wg.Wait()

	if size := c.Stats().Size; size > 64 {
		t.Errorf("Size = %d; want it bounded by capacity 64", size)
	}
}

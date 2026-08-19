package tiered

import (
	"testing"

	"ds-cache/internal/cache"
	"ds-cache/internal/cache/lru"
)

// newStack creates an L1/L2 stack and returns it with its tiers, so a test can
// check which tier holds a key.
func newStack(l1Cap, l2Cap int) (*Cache, *lru.Cache, *lru.Cache) {
	l1 := lru.New("l1", l1Cap)
	l2 := lru.New("l2", l2Cap)
	return New("stack", l1, l2), l1, l2
}

// TestPutWritesThroughEveryTier tests the Put function of Cache.
func TestPutWritesThroughEveryTier(t *testing.T) {
	stack, l1, l2 := newStack(2, 8)

	// Test adding a new key
	if created := stack.Put("key1", "value1"); !created {
		t.Error("Put(new key) created = false; want true")
	}
	if _, ok := l1.Get("key1"); !ok {
		t.Error("key1 missing from L1")
	}
	if _, ok := l2.Get("key1"); !ok {
		t.Error("key1 missing from L2")
	}

	// Test updating an existing key
	if created := stack.Put("key1", "value2"); created {
		t.Error("Put(existing key) created = true; want false")
	}
	if value, _ := stack.Get("key1"); value != "value2" {
		t.Errorf("Get(key1) = %q; want \"value2\"", value)
	}
}

// TestGetPromotesFromDeeperTier tests that an L2 hit is copied into L1.
func TestGetPromotesFromDeeperTier(t *testing.T) {
	stack, l1, l2 := newStack(1, 8)

	// Seed L2 only, so the first read has to reach past a cold L1
	l2.Put("key1", "value1")

	value, ok := stack.Get("key1")
	if !ok || value != "value1" {
		t.Fatalf("Get(key1) = (%q, %v); want (\"value1\", true)", value, ok)
	}

	if _, ok := l1.Get("key1"); !ok {
		t.Error("key1 was not promoted into L1 after an L2 hit")
	}
}

// TestGetMissAcrossAllTiers tests the Get function of Cache with a key held by
// no tier.
func TestGetMissAcrossAllTiers(t *testing.T) {
	stack, _, _ := newStack(2, 8)

	if value, ok := stack.Get("absent"); ok || value != "" {
		t.Errorf("Get(absent) = (%q, %v); want (\"\", false)", value, ok)
	}
	if misses := stack.Stats().Misses; misses != 1 {
		t.Errorf("Misses = %d; want 1", misses)
	}
}

// TestDeleteClearsEveryTier tests the Delete function of Cache.
func TestDeleteClearsEveryTier(t *testing.T) {
	stack, l1, l2 := newStack(2, 8)
	stack.Put("key1", "value1")

	if removed := stack.Delete("key1"); !removed {
		t.Error("Delete(existing) = false; want true")
	}
	if _, ok := l1.Get("key1"); ok {
		t.Error("key1 still in L1 after delete")
	}
	if _, ok := l2.Get("key1"); ok {
		t.Error("key1 still in L2 after delete; a stale copy would be promoted back")
	}
	if removed := stack.Delete("key1"); removed {
		t.Error("Delete(absent) = true; want false")
	}
}

// TestL1EvictionKeepsValueInL2 tests that an L1 eviction does not lose the key.
func TestL1EvictionKeepsValueInL2(t *testing.T) {
	stack, l1, _ := newStack(1, 8)

	stack.Put("key1", "value1")
	stack.Put("key2", "value2")

	if _, ok := l1.Get("key1"); ok {
		t.Error("key1 still in L1; want it evicted by the one entry capacity")
	}
	if value, ok := stack.Get("key1"); !ok || value != "value1" {
		t.Errorf("Get(key1) = (%q, %v); want (\"value1\", true) from L2", value, ok)
	}
}

// TestStatsReportsDeepestTierCapacity tests the Stats function of Cache.
func TestStatsReportsDeepestTierCapacity(t *testing.T) {
	stack, _, _ := newStack(2, 8)
	stack.Put("key1", "value1")
	stack.Get("key1")

	stats := stack.Stats()
	if stats.Capacity != 8 {
		t.Errorf("Capacity = %d; want 8", stats.Capacity)
	}
	if stats.Size != 1 {
		t.Errorf("Size = %d; want 1", stats.Size)
	}
	if stats.Hits != 1 {
		t.Errorf("Hits = %d; want 1", stats.Hits)
	}
}

// TestFlattenReportsEveryLayer tests the Flatten function over a stack.
func TestFlattenReportsEveryLayer(t *testing.T) {
	stack, _, _ := newStack(2, 8)

	stats := cache.Flatten(stack)
	if len(stats) != 3 {
		t.Fatalf("Flatten returned %d layers; want 3 (stack, l1, l2)", len(stats))
	}

	want := []string{"stack", "l1", "l2"}
	for i, name := range want {
		if stats[i].Name != name {
			t.Errorf("layer %d name = %q; want %q", i, stats[i].Name, name)
		}
	}
}

// TestNewPanicsWithoutTiers tests the New function with no tier.
func TestNewPanicsWithoutTiers(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("New() with no tiers did not panic")
		}
	}()
	New("empty")
}

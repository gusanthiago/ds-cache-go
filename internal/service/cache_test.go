package service

import (
	"errors"
	"strconv"
	"testing"

	"ds-cache/internal/cache"
	"ds-cache/internal/cache/lru"
	"ds-cache/internal/cache/tiered"
)

// newTestService creates a cluster of tiered nodes matching the wiring of main.
func newTestService(t *testing.T, nodeCount int) *CacheService {
	t.Helper()

	nodes := make([]cache.Store, 0, nodeCount)
	for i := 1; i <= nodeCount; i++ {
		name := "node-" + strconv.Itoa(i)
		nodes = append(nodes, tiered.New(name,
			lru.New(name+"-l1", 2),
			lru.New(name+"-l2", 16),
		))
	}

	svc, err := NewCacheService(nodes)
	if err != nil {
		t.Fatalf("NewCacheService() error = %v", err)
	}
	return svc
}

// TestNewCacheServiceRejectsEmptyCluster tests the NewCacheService function
// with no node.
func TestNewCacheServiceRejectsEmptyCluster(t *testing.T) {
	if _, err := NewCacheService(nil); !errors.Is(err, ErrNoNodes) {
		t.Errorf("NewCacheService(nil) error = %v; want ErrNoNodes", err)
	}
}

// TestSetThenGetIsImmediatelyVisible tests that a write is readable as soon as
// Set returns.
func TestSetThenGetIsImmediatelyVisible(t *testing.T) {
	svc := newTestService(t, 3)

	created, err := svc.Set("key1", "value1")
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if !created {
		t.Error("Set(new key) created = false; want true")
	}

	value, found, err := svc.Get("key1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !found || value != "value1" {
		t.Errorf("Get(key1) = (%q, %v); want (\"value1\", true)", value, found)
	}
}

// TestSetExistingKeyReportsNotCreated tests the Set function with an existing key.
func TestSetExistingKeyReportsNotCreated(t *testing.T) {
	svc := newTestService(t, 3)
	svc.Set("key1", "value1")

	created, err := svc.Set("key1", "value2")
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if created {
		t.Error("Set(existing key) created = true; want false")
	}

	if value, _, _ := svc.Get("key1"); value != "value2" {
		t.Errorf("Get(key1) = %q; want \"value2\"", value)
	}
}

// TestGetMissing tests the Get function with a non-existing key.
func TestGetMissing(t *testing.T) {
	svc := newTestService(t, 3)

	value, found, err := svc.Get("absent")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if found || value != "" {
		t.Errorf("Get(absent) = (%q, %v); want (\"\", false)", value, found)
	}
}

// TestDelete tests the Delete function of CacheService.
func TestDelete(t *testing.T) {
	svc := newTestService(t, 3)
	svc.Set("key1", "value1")

	removed, err := svc.Delete("key1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !removed {
		t.Error("Delete(existing) = false; want true")
	}

	if _, found, _ := svc.Get("key1"); found {
		t.Error("key1 still readable after delete")
	}
}

// TestEmptyKeyRejected tests that every operation rejects an empty key.
func TestEmptyKeyRejected(t *testing.T) {
	svc := newTestService(t, 3)

	if _, _, err := svc.Get(""); !errors.Is(err, ErrEmptyKey) {
		t.Errorf("Get(\"\") error = %v; want ErrEmptyKey", err)
	}
	if _, err := svc.Set("", "value"); !errors.Is(err, ErrEmptyKey) {
		t.Errorf("Set(\"\") error = %v; want ErrEmptyKey", err)
	}
	if _, err := svc.Delete(""); !errors.Is(err, ErrEmptyKey) {
		t.Errorf("Delete(\"\") error = %v; want ErrEmptyKey", err)
	}
}

// TestStatsCoversEveryNodeAndLayer tests the Stats function of CacheService.
func TestStatsCoversEveryNodeAndLayer(t *testing.T) {
	svc := newTestService(t, 3)
	svc.Set("key1", "value1")
	svc.Get("key1")

	stats := svc.Stats()
	if len(stats) != 3 {
		t.Fatalf("Stats() returned %d nodes; want 3", len(stats))
	}

	for _, s := range stats {
		if len(s.Layers) != 3 {
			t.Errorf("node %s reported %d layers; want 3 (stack, l1, l2)", s.Node, len(s.Layers))
		}
	}
}

// TestHashKey tests the hashKey function.
func TestHashKey(t *testing.T) {
	tests := []struct {
		key      string
		size     int
		expected int
	}{
		{"test", 2, 1},
		{"test1", 2, 0},
		{"test0", 3, 0},
		{"test1", 3, 1},
		{"test3", 3, 2},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := hashKey(tt.key, tt.size); got != tt.expected {
				t.Errorf("hashKey(%q, %d) = %d; want %d", tt.key, tt.size, got, tt.expected)
			}
		})
	}
}

// TestHashKeyIsStable tests that a key always maps to the same node, otherwise
// a written key becomes unreadable from the node the next lookup picks.
func TestHashKeyIsStable(t *testing.T) {
	const size = 7

	for i := 0; i < 1000; i++ {
		key := "key" + strconv.Itoa(i)
		got := hashKey(key, size)
		if got < 0 || got >= size {
			t.Fatalf("hashKey(%q) = %d; want within [0, %d)", key, got, size)
		}
		if again := hashKey(key, size); again != got {
			t.Fatalf("hashKey(%q) is unstable: %d then %d", key, got, again)
		}
	}
}

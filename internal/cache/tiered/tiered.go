package tiered

import (
	"sync/atomic"

	"ds-cache/internal/cache"
)

// Cache stacks stores from fastest to slowest (L1, L2, ...) and behaves as a
// single Store. A read probes each tier in order and copies a hit into the
// faster tiers. A write goes to every tier. A delete removes the key from
// every tier, so no stale copy is left in a slower one.
type Cache struct {
	name  string
	tiers []cache.Store

	hits   atomic.Uint64
	misses atomic.Uint64
}

var (
	_ cache.Store   = (*Cache)(nil)
	_ cache.Layered = (*Cache)(nil)
)

// New creates a new Cache from the given tiers, fastest first.
// It panics if no tier is given.
// Returns a pointer to the new Cache.
func New(name string, tiers ...cache.Store) *Cache {
	if len(tiers) == 0 {
		panic("tiered: at least one tier is required")
	}
	return &Cache{name: name, tiers: tiers}
}

// Get returns the value for the given key from the fastest tier that holds it,
// and copies it into the tiers that missed.
// Returns false if no tier holds the key.
func (c *Cache) Get(key string) (string, bool) {
	for i, tier := range c.tiers {
		value, ok := tier.Get(key)
		if !ok {
			continue
		}

		for _, faster := range c.tiers[:i] {
			faster.Put(key, value)
		}

		c.hits.Add(1)
		return value, true
	}

	c.misses.Add(1)
	return "", false
}

// Put writes the key-value pair to every tier.
// Returns the result of the deepest tier, which is the last one written and
// the one holding the full key set.
func (c *Cache) Put(key, value string) bool {
	created := false
	for _, tier := range c.tiers {
		created = tier.Put(key, value)
	}
	return created
}

// Delete removes the given key from every tier.
// Returns true if any tier held the key.
func (c *Cache) Delete(key string) bool {
	removed := false
	for _, tier := range c.tiers {
		if tier.Delete(key) {
			removed = true
		}
	}
	return removed
}

// Tiers returns a copy of the tiers, fastest first.
func (c *Cache) Tiers() []cache.Store {
	return append([]cache.Store(nil), c.tiers...)
}

// Stats returns a snapshot of the counters of the whole stack. Hits and misses
// count a lookup once, whatever tier answered it. Size and capacity come from
// the deepest tier, which bounds what the stack can hold.
func (c *Cache) Stats() cache.Stats {
	var evictions uint64
	var deepest cache.Stats
	for _, tier := range c.tiers {
		deepest = tier.Stats()
		evictions += deepest.Evictions
	}

	return cache.Stats{
		Name:      c.name,
		Hits:      c.hits.Load(),
		Misses:    c.misses.Load(),
		Evictions: evictions,
		Size:      deepest.Size,
		Capacity:  deepest.Capacity,
	}
}

// Name returns the name of the cache.
func (c *Cache) Name() string { return c.name }

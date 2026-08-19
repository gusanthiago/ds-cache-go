package lru

import (
	"container/list"
	"sync"
	"sync/atomic"

	"ds-cache/internal/cache"
)

type entry struct {
	key   string
	value string
}

// Cache is a Store bounded by a fixed capacity, evicting the least recently
// used entry when it is full. A read counts as use, so Get takes the same
// exclusive lock as Put.
type Cache struct {
	name     string
	capacity int

	mu    sync.Mutex
	order *list.List
	items map[string]*list.Element

	hits      atomic.Uint64
	misses    atomic.Uint64
	evictions atomic.Uint64
}

var _ cache.Store = (*Cache)(nil)

// New creates a new Cache with the given name and capacity.
// A capacity below 1 is raised to 1.
// Returns a pointer to the new Cache.
func New(name string, capacity int) *Cache {
	capacity = max(capacity, 1)
	return &Cache{
		name:     name,
		capacity: capacity,
		order:    list.New(),
		items:    make(map[string]*list.Element, capacity),
	}
}

// Get returns the value for the given key and moves the key to the front of
// the list.
// Returns false if the key is not found.
func (c *Cache) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	el, ok := c.items[key]
	if !ok {
		c.misses.Add(1)
		return "", false
	}

	c.order.MoveToFront(el)
	c.hits.Add(1)
	return el.Value.(*entry).value, true
}

// Put adds a new key-value pair to the cache. If the key already exists, it
// updates the value and moves the key to the front of the list.
// If the cache is full, it removes the least recently used key-value pair.
// Returns true if the key is new, false if the key already exists.
func (c *Cache) Put(key, value string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.items[key]; ok {
		el.Value.(*entry).value = value
		c.order.MoveToFront(el)
		return false
	}

	if len(c.items) >= c.capacity {
		c.evictOldest()
	}

	c.items[key] = c.order.PushFront(&entry{key: key, value: value})
	return true
}

// Delete removes the given key from the cache.
// Returns true if the key was present.
func (c *Cache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	el, ok := c.items[key]
	if !ok {
		return false
	}

	c.order.Remove(el)
	delete(c.items, key)
	return true
}

// Stats returns a snapshot of the counters of this layer.
func (c *Cache) Stats() cache.Stats {
	c.mu.Lock()
	size := len(c.items)
	c.mu.Unlock()

	return cache.Stats{
		Name:      c.name,
		Hits:      c.hits.Load(),
		Misses:    c.misses.Load(),
		Evictions: c.evictions.Load(),
		Size:      size,
		Capacity:  c.capacity,
	}
}

// Name returns the name of the cache.
func (c *Cache) Name() string { return c.name }

// evictOldest removes the entry at the back of the list.
// Callers must hold the lock.
func (c *Cache) evictOldest() {
	back := c.order.Back()
	if back == nil {
		return
	}

	c.order.Remove(back)
	delete(c.items, back.Value.(*entry).key)
	c.evictions.Add(1)
}

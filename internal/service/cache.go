package service

import (
	"errors"
	"hash/fnv"
	"log/slog"

	"ds-cache/internal/cache"
)

var (
	ErrEmptyKey = errors.New("cache key must not be empty")
	ErrNoNodes  = errors.New("cache service requires at least one node")
)

// CacheService routes keys across the nodes of the cluster. A node is a named
// Store, so it can be a single cache or a tiered stack.
type CacheService struct {
	nodes []cache.Store
}

// NodeStats holds the counters of one node and of its cache layers.
type NodeStats struct {
	Node   string        `json:"node"`
	Layers []cache.Stats `json:"layers"`
}

// NewCacheService creates a new CacheService over the given nodes.
// Returns ErrNoNodes if the list is empty.
func NewCacheService(nodes []cache.Store) (*CacheService, error) {
	if len(nodes) == 0 {
		return nil, ErrNoNodes
	}
	return &CacheService{nodes: nodes}, nil
}

// Get returns the value for the given key from the node that owns it.
// Returns false if the key is not found, and ErrEmptyKey if the key is empty.
func (s *CacheService) Get(key string) (string, bool, error) {
	if key == "" {
		return "", false, ErrEmptyKey
	}

	owner := s.ownerOf(key)
	value, ok := owner.Get(key)
	slog.Debug("cache get", "key", key, "node", owner.Name(), "hit", ok)

	return value, ok, nil
}

// Set writes the key-value pair to the node that owns it. The write is
// synchronous, so the value is readable as soon as Set returns.
// Returns true if the key is new, and ErrEmptyKey if the key is empty.
func (s *CacheService) Set(key, value string) (bool, error) {
	if key == "" {
		return false, ErrEmptyKey
	}

	owner := s.ownerOf(key)
	created := owner.Put(key, value)
	slog.Debug("cache set", "key", key, "node", owner.Name(), "created", created)

	return created, nil
}

// Delete removes the given key from the node that owns it.
// Returns true if the key was present, and ErrEmptyKey if the key is empty.
func (s *CacheService) Delete(key string) (bool, error) {
	if key == "" {
		return false, ErrEmptyKey
	}

	owner := s.ownerOf(key)
	removed := owner.Delete(key)
	slog.Debug("cache delete", "key", key, "node", owner.Name(), "removed", removed)

	return removed, nil
}

// Stats returns the counters of every node and cache layer of the cluster.
func (s *CacheService) Stats() []NodeStats {
	stats := make([]NodeStats, 0, len(s.nodes))
	for _, n := range s.nodes {
		stats = append(stats, NodeStats{Node: n.Name(), Layers: cache.Flatten(n)})
	}
	return stats
}

// ownerOf returns the node responsible for the given key.
func (s *CacheService) ownerOf(key string) cache.Store {
	return s.nodes[hashKey(key, len(s.nodes))]
}

// hashKey maps the given key onto one of size nodes using FNV-1a. The same key
// always maps to the same node while the cluster size does not change.
// Callers must pass a size of at least 1.
func hashKey(key string, size int) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() % uint32(size))
}

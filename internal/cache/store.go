package cache

// Store is a cache layer that holds keys under its own eviction policy.
// Implementations must be safe for concurrent use.
type Store interface {
	Name() string
	Get(key string) (value string, ok bool)
	Put(key, value string) (created bool)
	Delete(key string) (removed bool)
	Stats() Stats
}

// Layered is a Store built from other Stores.
// Tiers returns those stores, fastest first.
type Layered interface {
	Store
	Tiers() []Store
}

type Stats struct {
	Name      string `json:"name"`
	Hits      uint64 `json:"hits"`
	Misses    uint64 `json:"misses"`
	Evictions uint64 `json:"evictions"`
	Size      int    `json:"size"`
	Capacity  int    `json:"capacity"`
}

// Flatten returns the stats of the given store followed by the stats of every
// tier below it.
// Returns a single entry when the store is not layered.
func Flatten(s Store) []Stats {
	stats := []Stats{s.Stats()}
	if layered, ok := s.(Layered); ok {
		for _, tier := range layered.Tiers() {
			stats = append(stats, Flatten(tier)...)
		}
	}
	return stats
}

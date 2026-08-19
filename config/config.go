package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

const (
	defaultPort         = ":8882"
	defaultNodes        = 3
	defaultNodeCapacity = 1000
	defaultLogLevel     = "info"

	l1Divisor = 10
)

type Config struct {
	Port         string
	Nodes        int
	NodeCapacity int
	L1Capacity   int
	LogLevel     slog.Level
}

// Load reads the configuration from the environment, applying a default for
// every value that is unset or empty.
// Returns an error if a value cannot produce a working cache.
func Load() (*Config, error) {
	nodes, err := positiveIntEnv("CACHE_MANAGER_PSEUDO_NODES", defaultNodes)
	if err != nil {
		return nil, err
	}

	capacity, err := positiveIntEnv("CACHE_MANAGER_NODES_CAPACITY", defaultNodeCapacity)
	if err != nil {
		return nil, err
	}

	// L1 defaults to a fraction of the node capacity, never below 1.
	l1, err := positiveIntEnv("CACHE_MANAGER_L1_CAPACITY", max(capacity/l1Divisor, 1))
	if err != nil {
		return nil, err
	}
	if l1 > capacity {
		return nil, fmt.Errorf("CACHE_MANAGER_L1_CAPACITY (%d) must not exceed CACHE_MANAGER_NODES_CAPACITY (%d)", l1, capacity)
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(stringEnv("CACHE_MANAGER_LOG_LEVEL", defaultLogLevel))); err != nil {
		return nil, fmt.Errorf("CACHE_MANAGER_LOG_LEVEL must be one of debug, info, warn, error: %w", err)
	}

	return &Config{
		Port:         normalizePort(stringEnv("CACHE_MANAGER_PORT", defaultPort)),
		Nodes:        nodes,
		NodeCapacity: capacity,
		L1Capacity:   l1,
		LogLevel:     level,
	}, nil
}

// stringEnv returns the trimmed value of the given key.
// Returns the fallback if the key is unset or empty.
func stringEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

// positiveIntEnv returns the value of the given key parsed as an integer.
// Returns the fallback if the key is unset or empty, and an error if the value
// is not an integer of at least 1.
func positiveIntEnv(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}

	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	if v < 1 {
		return 0, fmt.Errorf("%s must be at least 1, got %d", key, v)
	}
	return v, nil
}

// normalizePort accepts "8882" or ":8882" and returns the ":8882" form that
// net/http expects.
func normalizePort(port string) string {
	if strings.HasPrefix(port, ":") {
		return port
	}
	return ":" + port
}

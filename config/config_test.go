package config

import (
	"log/slog"
	"testing"
)

// TestLoadDefaults tests the Load function with no value set.
func TestLoadDefaults(t *testing.T) {
	// Blanking the variables covers the empty case and keeps the test
	// independent of the shell it runs in
	for _, key := range []string{
		"CACHE_MANAGER_PORT",
		"CACHE_MANAGER_PSEUDO_NODES",
		"CACHE_MANAGER_NODES_CAPACITY",
		"CACHE_MANAGER_L1_CAPACITY",
		"CACHE_MANAGER_LOG_LEVEL",
	} {
		t.Setenv(key, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != defaultPort {
		t.Errorf("Port = %q; want %q", cfg.Port, defaultPort)
	}
	if cfg.Nodes != defaultNodes {
		t.Errorf("Nodes = %d; want %d", cfg.Nodes, defaultNodes)
	}
	if cfg.NodeCapacity != defaultNodeCapacity {
		t.Errorf("NodeCapacity = %d; want %d", cfg.NodeCapacity, defaultNodeCapacity)
	}
	if cfg.L1Capacity != defaultNodeCapacity/l1Divisor {
		t.Errorf("L1Capacity = %d; want %d", cfg.L1Capacity, defaultNodeCapacity/l1Divisor)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel = %v; want %v", cfg.LogLevel, slog.LevelInfo)
	}
}

// TestLoadReadsEnvironment tests that Load reads every value from the
// environment instead of using a hardcoded one.
func TestLoadReadsEnvironment(t *testing.T) {
	t.Setenv("CACHE_MANAGER_PORT", "9000")
	t.Setenv("CACHE_MANAGER_PSEUDO_NODES", "5")
	t.Setenv("CACHE_MANAGER_NODES_CAPACITY", "50")
	t.Setenv("CACHE_MANAGER_L1_CAPACITY", "7")
	t.Setenv("CACHE_MANAGER_LOG_LEVEL", "debug")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != ":9000" {
		t.Errorf("Port = %q; want \":9000\"", cfg.Port)
	}
	if cfg.Nodes != 5 {
		t.Errorf("Nodes = %d; want 5", cfg.Nodes)
	}
	if cfg.NodeCapacity != 50 {
		t.Errorf("NodeCapacity = %d; want 50", cfg.NodeCapacity)
	}
	if cfg.L1Capacity != 7 {
		t.Errorf("L1Capacity = %d; want 7", cfg.L1Capacity)
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("LogLevel = %v; want %v", cfg.LogLevel, slog.LevelDebug)
	}
}

// TestLoadRejectsInvalidValues tests the Load function with values that cannot
// produce a working cache.
func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := map[string]map[string]string{
		"non-numeric nodes": {"CACHE_MANAGER_PSEUDO_NODES": "many"},
		"zero nodes":        {"CACHE_MANAGER_PSEUDO_NODES": "0"},
		"negative capacity": {"CACHE_MANAGER_NODES_CAPACITY": "-1"},
		"zero l1":           {"CACHE_MANAGER_L1_CAPACITY": "0"},
		"l1 exceeds l2": {
			"CACHE_MANAGER_NODES_CAPACITY": "10",
			"CACHE_MANAGER_L1_CAPACITY":    "11",
		},
		"unknown log level": {"CACHE_MANAGER_LOG_LEVEL": "verbose"},
	}

	for name, env := range tests {
		t.Run(name, func(t *testing.T) {
			for key, value := range env {
				t.Setenv(key, value)
			}

			if _, err := Load(); err == nil {
				t.Errorf("Load() error = nil; want a rejection for %s", name)
			}
		})
	}
}

// TestL1DefaultNeverZero tests that a small capacity still leaves room for one
// L1 entry.
func TestL1DefaultNeverZero(t *testing.T) {
	t.Setenv("CACHE_MANAGER_NODES_CAPACITY", "3")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.L1Capacity != 1 {
		t.Errorf("L1Capacity = %d; want 1", cfg.L1Capacity)
	}
}

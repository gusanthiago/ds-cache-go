package utils

import (
	"testing"
)

// TestHashNode tests the HashNode function.
func TestHashNode(t *testing.T) {
	tests := []struct {
		input    string
		lenNodes int
		expected uint32
	}{
		{"test", 2, 1},
		{"test1", 2, 0},
		{"test0", 3, 0},
		{"test1", 3, 1},
		{"test3", 3, 2},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := HashNode(tt.input, tt.lenNodes)
			if result != tt.expected {
				t.Errorf("HashNode(%v, %v) = %v; want %v", tt.input, tt.lenNodes, result, tt.expected)
			}
		})
	}
}

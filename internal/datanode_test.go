package internal

import (
	"testing"
)

// TestNewDataNode tests the creation of a new DataNode.
func TestNewDataNode(t *testing.T) {
	name := "testNode"
	capacity := 2
	dn := NewDataNode(name, capacity)

	if dn.name != name {
		t.Errorf("NewDataNode() name = %v; want %v", dn.name, name)
	}
	if dn.capacity != capacity {
		t.Errorf("NewDataNode() capacity = %v; want %v", dn.capacity, capacity)
	}
	if dn.cache.Capacity != capacity {
		t.Errorf("NewDataNode() cache capacity = %v; want %v", dn.cache.Capacity, capacity)
	}
}

// TestPut tests the Put function of DataNode.
func TestPut(t *testing.T) {
	dn := NewDataNode("testNode", 2)

	// Test adding a new key
	isNew := dn.Put("key1", "value1")
	if !isNew {
		t.Errorf("Put() isNew = %v; want %v", isNew, true)
	}
	if dn.cache.Items["key1"].Data != "value1" {
		t.Errorf("Put() value = %v; want %v", dn.cache.Items["key1"].Data, "value1")
	}

	// Test updating an existing key
	isNew = dn.Put("key1", "value2")
	if isNew {
		t.Errorf("Put() isNew = %v; want %v", isNew, false)
	}
	if dn.cache.Items["key1"].Data != "value2" {
		t.Errorf("Put() value = %v; want %v", dn.cache.Items["key1"].Data, "value2")
	}

	// Test adding a new key when capacity is full
	dn.Put("key2", "value2")
	isNew = dn.Put("key3", "value3")
	if !isNew {
		t.Errorf("Put isNew = %v; want %v", isNew, true)
	}
	if _, ok := dn.cache.Items["key1"]; ok {
		t.Errorf("Put() key1 should have been evicted")
	}
}

// TestGet tests the Get function of DataNode.
func TestGet(t *testing.T) {
	dn := NewDataNode("testNode", 2)
	dn.Put("key1", "value1")

	// Test getting an existing key
	value := dn.Get("key1")
	if value != "value1" {
		t.Errorf("Get() value = %v; want %v", value, "value1")
	}

	// Test getting a non-existing key
	value = dn.Get("key2")
	if value != "NOT_FOUND" {
		t.Errorf("Get() value = %v; want %v", value, "NOT_FOUND")
	}
}

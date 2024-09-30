package internal

import (
	"container/list"
	"fmt"
	"sync"
)

type Node struct {
	DLink *list.Element
	Data  string
}

type LRUCache struct {
	dLinkedList *list.List
	Items       map[string]*Node
	Capacity    int
	lock        sync.Mutex
}

type DataNode struct {
	name     string
	capacity int
	cache    LRUCache
}

// NewDataNode creates a new DataNode with the given name and capacity.
// It initializes the cache with the given capacity.
// Returns a pointer to the new DataNode.
func NewDataNode(name string, capacity int) *DataNode {
	fmt.Printf("Intializing cache node (%s) with capacity (%d)\n", name, capacity)
	return &DataNode{
		name:     name,
		capacity: capacity,
		cache:    LRUCache{dLinkedList: list.New(), Items: make(map[string]*Node), Capacity: capacity},
	}
}

// Put adds a new key-value pair to the cache. If the key already exists, it updates the value and moves the key to the front of the list.
// If the cache is full, it removes the least recently used key-value pair.
// Returns true if the key is new, false if the key already exists.
func (d *DataNode) Put(key string, value string) bool {
	d.cache.lock.Lock()
	defer d.cache.lock.Unlock()

	fmt.Printf("Adding key (%s) with value (%s) to cache\n", key, value)

	cacheIt, ok := d.cache.Items[key]
	if !ok {
		if d.cache.Capacity == len(d.cache.Items) {
			back := d.cache.dLinkedList.Back()
			d.cache.dLinkedList.Remove(back)
			delete(d.cache.Items, back.Value.(string))
		}
		d.cache.Items[key] = &Node{Data: value, DLink: d.cache.dLinkedList.PushFront(key)}
		return true
	} else {
		cacheIt.Data = value
		d.cache.Items[key] = cacheIt
		d.cache.dLinkedList.MoveToFront(cacheIt.DLink)
		return false
	}
}

// Get returns the value for the given key from the cache.
// If the key is not found, it panics.
func (t *DataNode) Get(key string) string {
	fmt.Printf("Getting value for key (%s) from cache\n", key)
	if cacheIt, ok := t.cache.Items[key]; ok {
		t.cache.dLinkedList.MoveToFront(cacheIt.DLink)
		return cacheIt.Data
	}
	return "NOT_FOUND"
}

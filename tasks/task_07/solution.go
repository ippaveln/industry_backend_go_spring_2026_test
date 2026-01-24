package main

import (
	"container/list"
	"sync"
)

type entry[K comparable, V any] struct {
	key   K
	value V
}

type LRUCache[K comparable, V any] struct {
	capacity int

	mu    sync.Mutex
	ll    list.List
	items map[K]*list.Element
}

func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	return &LRUCache[K, V]{
		capacity: capacity,
		items:    make(map[K]*list.Element),
	}
}

func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	var zero V
	if c == nil || c.capacity <= 0 {
		return zero, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.items == nil {
		return zero, false
	}

	el, found := c.items[key]
	if !found {
		return zero, false
	}

	c.ll.MoveToFront(el)
	ent := el.Value.(*entry[K, V])
	return ent.value, true
}

func (c *LRUCache[K, V]) Set(key K, value V) {
	if c == nil || c.capacity <= 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.items == nil {
		c.items = make(map[K]*list.Element)
	}

	// update existing (refresh -> MRU)
	if el, found := c.items[key]; found {
		ent := el.Value.(*entry[K, V])
		ent.value = value
		c.ll.MoveToFront(el)
		return
	}

	// evict if full
	if len(c.items) >= c.capacity {
		back := c.ll.Back()
		if back != nil {
			ent := back.Value.(*entry[K, V])
			delete(c.items, ent.key)
			c.ll.Remove(back)
		}
	}

	// insert new as MRU
	el := c.ll.PushFront(&entry[K, V]{key: key, value: value})
	c.items[key] = el
}

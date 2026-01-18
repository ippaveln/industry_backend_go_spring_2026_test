package main

type LRUCache[K comparable, V any] struct {
	capacity int
	cache    map[K]V
	lru      map[K]int64
}

func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	if capacity <= 0 {
		return nil
	}
	return &LRUCache[K, V]{
		capacity: capacity,
		cache:    make(map[K]V, capacity),
		lru:      make(map[K]int64, capacity),
	}
}

func (l *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	if l == nil {
		var zero V
		return zero, false
	}
	value, ok = l.cache[key]
	l.lru[key] += 1
	return
}

func (l *LRUCache[K, V]) Set(key K, value V) {
	if l == nil {
		return
	}
	if len(l.cache) < l.capacity {
		l.cache[key] = value
		l.lru[key] += 1
		return
	}
	// Find the least recently used key
	var lruKey K
	var minUsage int64 = -1
	for k, usage := range l.lru {
		if minUsage == -1 || usage < minUsage {
			minUsage = usage
			lruKey = k
		}
	}
	// Evict the least recently used key
	delete(l.cache, lruKey)
	delete(l.lru, lruKey)

	// Add the new key-value pair
	l.cache[key] = value
	l.lru[key] = 1
}

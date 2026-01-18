package main

type Cache[K comparable, V any] struct {
	cache map[K]V
}

func NewCache[K comparable, V any](capacity int) *Cache[K, V] {
	if capacity <= 0 {
		return nil
	}
	return &Cache[K, V]{
		cache: make(map[K]V, capacity),
	}
}

func (l *Cache[K, V]) Get(key K) (value V, ok bool) {
	if l == nil {
		var zero V
		return zero, false
	}
	value, ok = l.cache[key]
	return
}

func (l *Cache[K, V]) Set(key K, value V) {
	if l == nil {
		return
	}
	if len(l.cache) == 0 {
		l.cache = make(map[K]V)
	}
	l.cache[key] = value
}

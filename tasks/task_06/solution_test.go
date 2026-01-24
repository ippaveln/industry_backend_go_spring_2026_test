package main

import (
	"testing"
)

func Test_LRU_check_interface(t *testing.T) {
	t.Parallel()

	var _ LRU[string, string] = (*LRUCache[string, string])(nil)
}

func Test_LRU_basic_operations(t *testing.T) {
	t.Parallel()

	cache := NewLRUCache[string, string](2)

	// Test Set and Get
	cache.Set("a", "1")
	cache.Set("b", "2")

	if val, ok := cache.Get("a"); !ok || val != "1" {
		t.Fatalf("Get(a) = %q, %v; want %q, true", val, ok, "1")
	}

	if val, ok := cache.Get("b"); !ok || val != "2" {
		t.Fatalf("Get(b) = %q, %v; want %q, true", val, ok, "2")
	}

	// Test LRU eviction
	cache.Set("c", "3") // This should evict "a"

	if _, ok := cache.Get("a"); ok {
		t.Fatalf("Get(a) = _, %v; want _, false", ok)
	}

	if val, ok := cache.Get("c"); !ok || val != "3" {
		t.Fatalf("Get(c) = %q, %v; want %q, true", val, ok, "3")
	}

	// Test updating existing key
	cache.Set("b", "20")
	if val, ok := cache.Get("b"); !ok || val != "20" {
		t.Fatalf("Get(b) = %q, %v; want %q, true", val, ok, "20")
	}
}

func Test_LRU_capacity_one(t *testing.T) {
	t.Parallel()

	cache := NewLRUCache[string, string](1)

	cache.Set("a", "1")
	if val, ok := cache.Get("a"); !ok || val != "1" {
		t.Fatalf("Get(a) = %q, %v; want %q, true", val, ok, "1")
	}

	cache.Set("b", "2") // This should evict "a"
	if _, ok := cache.Get("a"); ok {
		t.Fatalf("Get(a) = _, %v; want _, false", ok)
	}

	if val, ok := cache.Get("b"); !ok || val != "2" {
		t.Fatalf("Get(b) = %q, %v; want %q, true", val, ok, "2")
	}
}

func Test_LRU_zero_capacity(t *testing.T) {
	t.Parallel()

	cache := NewLRUCache[string, string](0)

	cache.Set("a", "1")
	if _, ok := cache.Get("a"); ok {
		t.Fatalf("Get(a) = _, %v; want _, false", ok)
	}

	cache.Set("b", "2")
	if _, ok := cache.Get("b"); ok {
		t.Fatalf("Get(b) = _, %v; want _, false", ok)
	}
}
func Test_LRU_overwrite(t *testing.T) {
	t.Parallel()

	cache := NewLRUCache[string, string](2)

	cache.Set("a", "1")
	cache.Set("a", "10") // Overwrite

	if val, ok := cache.Get("a"); !ok || val != "10" {
		t.Fatalf("Get(a) = %q, %v; want %q, true", val, ok, "10")
	}

	cache.Set("b", "2")
	cache.Set("c", "3") // This should evict "a" if overwrite didn't work

	if val, ok := cache.Get("a"); ok || val == "10" {
		t.Fatalf("Get(a) = %q, %v; want %q, true", val, ok, "10")
	}
}

func Test_LRU_eviction_order(t *testing.T) {
	t.Parallel()

	cache := NewLRUCache[string, string](3)

	cache.Set("a", "1")
	cache.Set("b", "2")
	cache.Set("c", "3")

	// Access "a" and "b" to make "c" the least recently used
	if _, ok := cache.Get("a"); !ok {
		t.Fatalf("Get(a) failed; want true")
	}
	if _, ok := cache.Get("b"); !ok {
		t.Fatalf("Get(b) failed; want true")
	}

	cache.Set("d", "4") // This should evict "c"

	if _, ok := cache.Get("c"); ok {
		t.Fatalf("Get(c) = _, %v; want _, false", ok)
	}

	if val, ok := cache.Get("d"); !ok || val != "4" {
		t.Fatalf("Get(d) = %q, %v; want %q, true", val, ok, "4")
	}
}

func Test_LRU_overwrite_evicts_correct_key(t *testing.T) {
	t.Parallel()

	cache := NewLRUCache[string, string](2)

	cache.Set("a", "1")
	cache.Set("a", "10") // update makes "a" MRU
	cache.Set("b", "2")  // now "b" MRU, "a" LRU
	cache.Set("c", "3")  // should evict "a"

	if _, ok := cache.Get("a"); ok {
		t.Fatalf("Get(a) ok=%v; want false (a should be evicted)", ok)
	}
	if val, ok := cache.Get("b"); !ok || val != "2" {
		t.Fatalf("Get(b) = %q, %v; want %q, true", val, ok, "2")
	}
	if val, ok := cache.Get("c"); !ok || val != "3" {
		t.Fatalf("Get(c) = %q, %v; want %q, true", val, ok, "3")
	}
}

// Обновление существующего ключа НЕ должно вызывать eviction,
// но должно менять порядок (делать ключ MRU).
func Test_LRU_update_does_not_evict_but_refreshes(t *testing.T) {
	t.Parallel()

	cache := NewLRUCache[string, string](2)

	cache.Set("a", "1")
	cache.Set("b", "2")

	cache.Set("a", "10") // refresh a, so b becomes LRU
	cache.Set("c", "3")  // should evict b

	if val, ok := cache.Get("a"); !ok || val != "10" {
		t.Fatalf("Get(a) = %q, %v; want %q, true", val, ok, "10")
	}
	if _, ok := cache.Get("b"); ok {
		t.Fatalf("Get(b) ok=%v; want false (b should be evicted)", ok)
	}
	if val, ok := cache.Get("c"); !ok || val != "3" {
		t.Fatalf("Get(c) = %q, %v; want %q, true", val, ok, "3")
	}
}

// Проверяем, что Get() тоже освежает (делает MRU).
func Test_LRU_get_refreshes_recency(t *testing.T) {
	t.Parallel()

	cache := NewLRUCache[string, string](2)

	cache.Set("a", "1")
	cache.Set("b", "2")

	// refresh a -> b becomes LRU
	if val, ok := cache.Get("a"); !ok || val != "1" {
		t.Fatalf("Get(a) = %q, %v; want %q, true", val, ok, "1")
	}

	cache.Set("c", "3") // should evict b

	if _, ok := cache.Get("b"); ok {
		t.Fatalf("Get(b) ok=%v; want false (b should be evicted)", ok)
	}
	if val, ok := cache.Get("a"); !ok || val != "1" {
		t.Fatalf("Get(a) = %q, %v; want %q, true", val, ok, "1")
	}
	if val, ok := cache.Get("c"); !ok || val != "3" {
		t.Fatalf("Get(c) = %q, %v; want %q, true", val, ok, "3")
	}
}

// Важно: отсутствие ключа должно отличаться от "ключ есть, но значение = zero value".
func Test_LRU_missing_key_returns_zero_value_and_ok_false(t *testing.T) {
	t.Parallel()

	cache := NewLRUCache[string, int](2)

	val, ok := cache.Get("missing")
	if ok {
		t.Fatalf("Get(missing) ok=%v; want false", ok)
	}
	if val != 0 {
		t.Fatalf("Get(missing) val=%v; want 0 (zero value)", val)
	}
}

func Test_LRU_can_store_zero_value_distinct_from_missing(t *testing.T) {
	t.Parallel()

	cache := NewLRUCache[string, string](2)

	cache.Set("empty", "")
	val, ok := cache.Get("empty")
	if !ok {
		t.Fatalf("Get(empty) ok=%v; want true", ok)
	}
	if val != "" {
		t.Fatalf("Get(empty) val=%q; want empty string", val)
	}

	val2, ok2 := cache.Get("missing")
	if ok2 {
		t.Fatalf("Get(missing) ok=%v; want false", ok2)
	}
	// val2 тут тоже будет "", но ok2=false — это и есть требуемое отличие
	_ = val2
}

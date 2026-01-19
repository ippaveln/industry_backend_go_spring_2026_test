package main

import (
	"testing"
)

func Test_LRU_check_interface(t *testing.T) {
	t.Parallel()

	var c LRU[string, string]
	c = &LRUCache[string, string]{}
	_ = c
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

	if val, ok := cache.Get("a"); !ok || val != "10" {
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

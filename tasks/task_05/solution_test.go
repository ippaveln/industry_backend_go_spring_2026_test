package main

import (
	"testing"
)

func Test_cache_basic_operations(t *testing.T) {
	t.Parallel()

	cache := NewCache[string, string](2)

	// Test Set and Get
	cache.Set("a", "1")
	cache.Set("b", "2")

	if val, ok := cache.Get("a"); !ok || val != "1" {
		t.Fatalf("Get(a) = %q, %v; want %q, true", val, ok, "1")
	}

	if val, ok := cache.Get("b"); !ok || val != "2" {
		t.Fatalf("Get(b) = %q, %v; want %q, true", val, ok, "2")
	}

	cache.Set("c", "3")

	if val, ok := cache.Get("a"); !ok || val != "1" {
		t.Fatalf("%v, %v = Get(a); want %v, true", val, ok, "1")
	}

	if val, ok := cache.Get("c"); !ok || val != "3" {
		t.Fatalf("Get(c) = %q, %v; want %q, true", val, ok, "3")
	}

	// Test updating existing key
	cache.Set("b", "20")
	if val, ok := cache.Get("b"); !ok || val != "20" {
		t.Fatalf("Get(b) = %q, %v; want %q, true", val, ok, "20")
	}

	if val, ok := cache.Get("d"); ok {
		t.Fatalf("Get(d) = %q, %v; want _, false", val, ok)
	}
}

func Test_LRU_capacity_one(t *testing.T) {
	t.Parallel()

	cache := NewCache[string, string](1)

	cache.Set("a", "1")
	if val, ok := cache.Get("a"); !ok || val != "1" {
		t.Fatalf("Get(a) = %q, %v; want %q, true", val, ok, "1")
	}

	cache.Set("b", "2")
	if val, ok := cache.Get("a"); !ok || val != "1" {
		t.Fatalf("Get(a) = %v, %v; want %v, true", val, ok, "1")
	}

	if val, ok := cache.Get("b"); !ok || val != "2" {
		t.Fatalf("Get(b) = %q, %v; want %q, true", val, ok, "2")
	}
}

func Test_cache_zero_capacity(t *testing.T) {
	t.Parallel()

	cache := NewCache[string, string](0)

	cache.Set("a", "1")
	if val, ok := cache.Get("a"); ok || val == "1" {
		t.Fatalf("Get(a) = %v, %v; want %v, false", val, ok, "1")
	}

	cache.Set("b", "2")
	if val, ok := cache.Get("b"); ok || val == "2" {
		t.Fatalf("Get(b) = %v, %v; want %v, true", val, ok, "2")
	}
}

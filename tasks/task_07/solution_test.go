package main

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
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

type failOnce struct {
	once  sync.Once
	done  chan struct{}
	errCh chan error
}

func waitTimeout(wg *sync.WaitGroup, d time.Duration) bool {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(d):
		return false
	}
}

func newFailOnce() *failOnce {
	return &failOnce{
		done:  make(chan struct{}),
		errCh: make(chan error, 1),
	}
}

func (f *failOnce) Failf(format string, args ...any) {
	f.once.Do(func() {
		f.errCh <- fmt.Errorf(format, args...)
		close(f.done)
	})
}

func (f *failOnce) Done() <-chan struct{} { return f.done }

func (f *failOnce) Err() error {
	select {
	case err := <-f.errCh:
		return err
	default:
		return nil
	}
}

func Test_LRU_concurrent_get_set_no_eviction_expected(t *testing.T) {
	t.Parallel()

	const keysCount = 64
	const writers = 8
	const readers = 8
	const itersPerG = 50_000

	cache := NewLRUCache[string, int](keysCount)
	for i := 0; i < keysCount; i++ {
		cache.Set(fmt.Sprintf("k%d", i), 0)
	}

	f := newFailOnce()
	start := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(writers + readers)

	for w := 0; w < writers; w++ {
		go func(seed int64) {
			defer wg.Done()
			r := rand.New(rand.NewSource(seed))
			<-start
			for i := 0; i < itersPerG; i++ {
				select {
				case <-f.Done():
					return
				default:
				}
				k := fmt.Sprintf("k%d", r.Intn(keysCount))
				cache.Set(k, i)
			}
		}(time.Now().UnixNano() + int64(w))
	}

	for rID := 0; rID < readers; rID++ {
		go func(seed int64) {
			defer wg.Done()
			r := rand.New(rand.NewSource(seed))
			<-start
			for i := 0; i < itersPerG; i++ {
				select {
				case <-f.Done():
					return
				default:
				}
				k := fmt.Sprintf("k%d", r.Intn(keysCount))
				v, ok := cache.Get(k)
				if !ok {
					f.Failf("Get(%s) ok=false; want true (no eviction expected)", k)
					return
				}
				if v < 0 {
					f.Failf("Get(%s) value=%d; want >=0", k, v)
					return
				}
			}
		}(time.Now().UnixNano() + int64(1000+rID))
	}

	close(start)
	if !waitTimeout(&wg, 5*time.Second) {
		t.Fatal("timeout waiting for goroutines (possible deadlock)")
	}

	if err := f.Err(); err != nil {
		t.Fatal(err)
	}
}

func Test_LRU_concurrent_eviction_stress_no_panic(t *testing.T) {
	t.Parallel()

	const cap = 8
	const keySpace = 128
	const goroutines = 16
	const itersPerG = 30_000

	cache := NewLRUCache[string, int](cap)

	f := newFailOnce()
	start := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(seed int64) {
			defer wg.Done()
			r := rand.New(rand.NewSource(seed))
			<-start
			for i := 0; i < itersPerG; i++ {
				select {
				case <-f.Done():
					return
				default:
				}
				k := fmt.Sprintf("k%d", r.Intn(keySpace))
				if r.Intn(2) == 0 {
					cache.Set(k, i)
				} else {
					_, _ = cache.Get(k)
				}
			}
		}(time.Now().UnixNano() + int64(g))
	}

	close(start)
	if !waitTimeout(&wg, 5*time.Second) {
		t.Fatal("timeout waiting for goroutines (possible deadlock)")
	}

	if err := f.Err(); err != nil {
		t.Fatal(err)
	}
}

func Test_LRU_concurrent_value_consistency_same_key(t *testing.T) {
	t.Parallel()

	type pair struct {
		A int
		B int
	}

	const writers = 4
	const readers = 4
	const itersPerG = 200_000

	cache := NewLRUCache[string, pair](1)
	cache.Set("x", pair{A: 0, B: 0})

	f := newFailOnce()
	start := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(writers + readers)

	for w := 0; w < writers; w++ {
		go func(base int) {
			defer wg.Done()
			<-start
			for i := 0; i < itersPerG; i++ {
				select {
				case <-f.Done():
					return
				default:
				}
				v := base + i
				cache.Set("x", pair{A: v, B: v})
			}
		}(w * 1_000_000)
	}

	for rID := 0; rID < readers; rID++ {
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < itersPerG; i++ {
				select {
				case <-f.Done():
					return
				default:
				}
				v, ok := cache.Get("x")
				if !ok {
					f.Failf("Get(x) ok=false; want true")
					return
				}
				if v.A != v.B {
					f.Failf("torn read detected: A=%d B=%d", v.A, v.B)
					return
				}
			}
		}()
	}

	close(start)
	if !waitTimeout(&wg, 5*time.Second) {
		t.Fatal("timeout waiting for goroutines (possible deadlock)")
	}

	if err := f.Err(); err != nil {
		t.Fatal(err)
	}
}

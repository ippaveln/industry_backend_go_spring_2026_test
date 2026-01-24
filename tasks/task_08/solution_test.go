package main

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fakeClock struct {
	now time.Time
}

func newFakeClock(start time.Time) *fakeClock { return &fakeClock{now: start} }

func (c *fakeClock) Now() time.Time { return c.now }

func (c *fakeClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

func TestLimiter_burst_initial(t *testing.T) {
	t.Parallel()

	clk := newFakeClock(time.Unix(0, 0))
	burst := 3
	l := NewLimiter(clk, 1.0, burst)

	for i := 0; i < burst; i++ {
		if ok := l.Allow(); !ok {
			t.Fatalf("Allow()=%v at i=%d; want true (initial burst)", ok, i)
		}
	}
	if ok := l.Allow(); ok {
		t.Fatalf("Allow()=%v; want false (burst exhausted)", ok)
	}
}

func TestLimiter_spend_tokens_then_refill_over_time(t *testing.T) {
	t.Parallel()

	clk := newFakeClock(time.Unix(0, 0))
	l := NewLimiter(clk, 1.0, 2) // 1 token/sec, burst 2

	// drain initial burst
	if !l.Allow() || !l.Allow() {
		t.Fatalf("expected to spend initial burst tokens")
	}
	if ok := l.Allow(); ok {
		t.Fatalf("Allow()=%v; want false (empty)", ok)
	}

	// 0.5 sec -> still not enough for 1 token
	clk.Advance(500 * time.Millisecond)
	if ok := l.Allow(); ok {
		t.Fatalf("Allow()=%v; want false (only 0.5 token accrued)", ok)
	}

	// +0.5 sec (total 1 sec) -> 1 token
	clk.Advance(500 * time.Millisecond)
	if ok := l.Allow(); !ok {
		t.Fatalf("Allow()=%v; want true (1 token accrued)", ok)
	}
	if ok := l.Allow(); ok {
		t.Fatalf("Allow()=%v; want false (spent the only token)", ok)
	}
}

func TestLimiter_refill_is_capped_by_burst(t *testing.T) {
	t.Parallel()

	clk := newFakeClock(time.Unix(0, 0))
	burst := 5
	l := NewLimiter(clk, 1000.0, burst) // высокая скорость, но ёмкость ограничена burst

	// drain
	for i := 0; i < burst; i++ {
		if !l.Allow() {
			t.Fatalf("expected to drain initial burst, i=%d", i)
		}
	}
	if l.Allow() {
		t.Fatalf("expected empty bucket")
	}

	// далеко во времени: токенов должно стать максимум burst
	clk.Advance(10 * time.Second)

	for i := 0; i < burst; i++ {
		if !l.Allow() {
			t.Fatalf("Allow()=false at i=%d; want true (refilled up to burst)", i)
		}
	}
	if l.Allow() {
		t.Fatalf("Allow()=true; want false (should not exceed burst capacity)")
	}
}

func TestLimiter_rate_zero_no_refill(t *testing.T) {
	t.Parallel()

	clk := newFakeClock(time.Unix(0, 0))
	l := NewLimiter(clk, 0.0, 2)

	if !l.Allow() || !l.Allow() {
		t.Fatalf("expected initial burst to be available even with rate=0")
	}
	if l.Allow() {
		t.Fatalf("expected empty after spending burst")
	}

	clk.Advance(100 * time.Second)
	if l.Allow() {
		t.Fatalf("Allow()=true; want false (rate=0 must not refill)")
	}
}

func TestLimiter_burst_zero_always_reject(t *testing.T) {
	t.Parallel()

	clk := newFakeClock(time.Unix(0, 0))
	l := NewLimiter(clk, 10.0, 0)

	if l.Allow() {
		t.Fatalf("Allow()=true; want false (burst=0)")
	}
	clk.Advance(10 * time.Second)
	if l.Allow() {
		t.Fatalf("Allow()=true; want false (burst=0 caps capacity at 0)")
	}
}

func TestLimiter_fractional_rate_accumulates_fractionally(t *testing.T) {
	t.Parallel()

	clk := newFakeClock(time.Unix(0, 0))
	l := NewLimiter(clk, 2.5, 10) // 2.5 token/sec

	// drain initial burst (10)
	for i := 0; i < 10; i++ {
		if !l.Allow() {
			t.Fatalf("expected initial burst, i=%d", i)
		}
	}
	if l.Allow() {
		t.Fatalf("expected empty after draining burst")
	}

	// +1s => +2.5 токена, значит 2 Allow должны пройти, 3-й должен отказать
	clk.Advance(1 * time.Second)

	if !l.Allow() {
		t.Fatalf("Allow()=false; want true (token #1 after 1s)")
	}
	if !l.Allow() {
		t.Fatalf("Allow()=false; want true (token #2 after 1s)")
	}
	if l.Allow() {
		t.Fatalf("Allow()=true; want false (only 0.5 token left)")
	}
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

// 1) Потокобезопасность: много goroutine одновременно вызывают Allow() в один момент времени.
// Ожидание: true должно быть ровно burst раз (корзина стартует полной), остальное false.
func TestLimiter_concurrent_allow_respects_initial_burst(t *testing.T) {
	t.Parallel()

	clk := newFakeClock(time.Unix(0, 0))

	burst := 100
	l := NewLimiter(clk, 0.0, burst) // rate=0, чтобы точно не было refill

	const goroutines = 1000

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(goroutines)

	var allowed int64

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			<-start
			if l.Allow() {
				atomic.AddInt64(&allowed, 1)
			}
		}()
	}

	close(start)

	if !waitTimeout(&wg, 3*time.Second) {
		t.Fatal("timeout waiting for goroutines (possible deadlock)")
	}

	if got := int(atomic.LoadInt64(&allowed)); got != burst {
		t.Fatalf("concurrent Allow() allowed=%d; want %d", got, burst)
	}
}

// 2) Потокобезопасность + корректность: после refill разрешений не больше, чем накопилось токенов.
// Сценарий: сливаем весь burst, двигаем время, затем одновременно дергаем Allow().
func TestLimiter_concurrent_allow_after_refill_is_capped(t *testing.T) {
	t.Parallel()

	clk := newFakeClock(time.Unix(0, 0))

	burst := 50
	rate := 10.0 // 10 токенов/сек
	l := NewLimiter(clk, rate, burst)

	// drain burst
	for i := 0; i < burst; i++ {
		if !l.Allow() {
			t.Fatalf("failed to drain initial burst at i=%d", i)
		}
	}
	if l.Allow() {
		t.Fatalf("expected empty after draining burst")
	}

	// +2s => +20 tokens
	clk.Advance(2 * time.Second)
	want := 20

	const goroutines = 500
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(goroutines)

	var allowed int64
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			<-start
			if l.Allow() {
				atomic.AddInt64(&allowed, 1)
			}
		}()
	}
	close(start)

	if !waitTimeout(&wg, 3*time.Second) {
		t.Fatal("timeout waiting for goroutines (possible deadlock)")
	}

	if got := int(atomic.LoadInt64(&allowed)); got != want {
		t.Fatalf("after refill concurrent Allow() allowed=%d; want %d", got, want)
	}
}

// 3) Стресс на конкурентные вызовы: много попыток, время не движется.
// Инвариант: разрешений не больше burst (потому что refill не происходит).
func TestLimiter_concurrent_stress_no_refill_never_exceeds_burst(t *testing.T) {
	t.Parallel()

	clk := newFakeClock(time.Unix(0, 0))

	burst := 30
	l := NewLimiter(clk, 0.0, burst)

	const goroutines = 32
	const callsPerG = 2000

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(goroutines)

	var allowed int64
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < callsPerG; i++ {
				if l.Allow() {
					atomic.AddInt64(&allowed, 1)
				}
			}
		}()
	}

	close(start)

	if !waitTimeout(&wg, 5*time.Second) {
		t.Fatal("timeout waiting for goroutines (possible deadlock)")
	}

	if got := int(atomic.LoadInt64(&allowed)); got != burst {
		t.Fatalf("allowed=%d; want %d (rate=0, no refill)", got, burst)
	}
}

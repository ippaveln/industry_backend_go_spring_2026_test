package main

import (
	"sync"
	"time"
)

type Clock interface {
	Now() time.Time
}

type Limiter struct {
	clock Clock

	ratePerSec float64
	burst      int

	mu     sync.Mutex
	tokens float64
	last   time.Time
}

func NewLimiter(clock Clock, ratePerSec float64, burst int) *Limiter {
	l := &Limiter{
		clock:      clock,
		ratePerSec: ratePerSec,
		burst:      burst,
	}
	now := time.Time{}
	if clock != nil {
		now = clock.Now()
	}
	l.last = now

	if burst > 0 {
		l.tokens = float64(burst) // старт полный => "burst в начале"
	} else {
		l.tokens = 0
	}
	return l
}

func (l *Limiter) Allow() bool {
	if l == nil || l.clock == nil {
		return false
	}
	if l.burst <= 0 {
		return false
	}
	if l.ratePerSec < 0 {
		// если вдруг передали отрицательный rate, трактуем как 0 (не пополняется)
		// можно и panic, но тесты ожидают просто "не пополняется"
		l.ratePerSec = 0
	}

	now := l.clock.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	// время назад — токены не "откатываем"
	if now.After(l.last) && l.ratePerSec > 0 {
		elapsed := now.Sub(l.last).Seconds()
		l.tokens += elapsed * l.ratePerSec

		capacity := float64(l.burst)
		if l.tokens > capacity {
			l.tokens = capacity
		}
	}
	l.last = now

	if l.tokens >= 1.0 {
		l.tokens -= 1.0
		return true
	}
	return false
}

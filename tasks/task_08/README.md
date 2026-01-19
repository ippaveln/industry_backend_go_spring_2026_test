#  Rate limiter  (token bucket)

```go
type Limiter struct{ ... }
func NewLimiter(clock Clock, ratePerSec float64, burst int) *Limiter
func (l *Limiter) Allow() bool
```
Детерминированно с фейковыми часами.
Юнит-тесты: burst в начале, расход токенов, восстановление со временем, пограничные случаи.

![task 08](badges/tasks/task_08.svg)

// TODO!!!
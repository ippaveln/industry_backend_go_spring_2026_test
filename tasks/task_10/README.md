# Retry с backoff (без сна в тестах)
```go
type Sleeper interface{ Sleep(time.Duration) }
func Retry(ctx context.Context, sleeper Sleeper, attempts int, base time.Duration, fn func() error) error
```

Экспоненциальная задержка, прекращение по ctx.Done(), ошибки оборачивать.
Юнит-тесты: число попыток, корректные задержки (через фейковый sleeper), остановка по контексту.Worker pool + context

![task 10](badges/tasks/task_10.svg)

// TODO!!!
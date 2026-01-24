#  Worker pool + context

```go
func ParallelMap[T any, R any](ctx context.Context, workers int, in []T, fn func(context.Context, T) (R, error)) ([]R, error)
```

Сохраняет порядок результатов как во входе. При первой ошибке отменяет контекст и останавливает лишнюю работу.
Юнит-тесты: порядок, отмена, workers=1/много, ошибка в середине, отсутствие goroutine leak (косвенно: завершение).


![task 09](../../badges/tasks/task_09.svg)

// TODO!!!
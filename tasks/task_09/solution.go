package main

import (
	"context"
	"errors"
	"sync"
)

var errInvalidWorkers = errors.New("workers must be > 0")

func ParallelMap[T any, R any](
	ctx context.Context,
	workers int,
	in []T,
	fn func(context.Context, T) (R, error),
) ([]R, error) {
	if workers <= 0 {
		return nil, errInvalidWorkers
	}
	if len(in) == 0 {
		return []R{}, nil
	}
	if err := ctx.Err(); err != nil {
		// Контракт: если ctx уже отменён — ничего не запускаем.
		return nil, err
	}

	ctx2, cancel := context.WithCancel(ctx)
	defer cancel()

	type job struct {
		idx int
		val T
	}

	w := workers
	if w > len(in) {
		w = len(in)
	}

	out := make([]R, len(in))

	// Буфер = w, чтобы успело стартовать несколько задач до первой ошибки (для теста sawCancel>0),
	// но при этом не “раскрутить” хвост работы.
	jobs := make(chan job, w)

	var (
		wg       sync.WaitGroup
		errOnce  sync.Once
		firstErr error
	)

	setErr := func(err error) {
		if err == nil {
			return
		}
		errOnce.Do(func() {
			firstErr = err
			cancel()
		})
	}

	workerFn := func() {
		defer wg.Done()

		for {
			// Если задача уже есть в очереди, берём её даже при отменённом ctx2:
			// fn увидит ctx2.Done() и быстро вернёт ctx.Err(), что нужно для теста “observe cancel”.
			var (
				j  job
				ok bool
			)

			select {
			case j, ok = <-jobs:
				if !ok {
					return
				}
			default:
				// Нет готовой задачи — тогда уже уважаем отмену.
				select {
				case <-ctx2.Done():
					return
				case j, ok = <-jobs:
					if !ok {
						return
					}
				}
			}

			r, err := fn(ctx2, j.val)
			if err != nil {
				setErr(err)
				return
			}
			out[j.idx] = r
		}
	}

	wg.Add(w)
	for i := 0; i < w; i++ {
		go workerFn()
	}

	// Producer: прекращаем выдачу новых задач при отмене.
	for i, v := range in {
		select {
		case <-ctx2.Done():
			close(jobs)
			wg.Wait()
			if firstErr != nil {
				return nil, firstErr
			}
			return nil, ctx.Err()
		case jobs <- job{idx: i, val: v}:
		}
	}

	close(jobs)
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

package main

import (
	"context"
	"errors"
	"runtime"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func TestParallelMap_WorkersNonPositive_ReturnsErrorAndDoesNotCallFn(t *testing.T) {
	t.Parallel()

	cases := []int{0, -1, -10}

	for _, workers := range cases {
		t.Run("workers="+strconv.Itoa(workers), func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			in := []int{1, 2, 3}

			var called int32
			fn := func(ctx context.Context, v int) (int, error) {
				atomic.AddInt32(&called, 1)
				return v, nil
			}

			out, err := ParallelMap[int, int](ctx, workers, in, fn)
			if err == nil {
				t.Fatalf("expected error for workers=%d, got nil (out=%v)", workers, out)
			}
			if got := atomic.LoadInt32(&called); got != 0 {
				t.Fatalf("fn called %d times, want 0 for workers=%d", got, workers)
			}
		})
	}
}

func TestParallelMap_EmptyInput_ReturnsEmptyAndNilError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	var called int32
	fn := func(ctx context.Context, v int) (int, error) {
		atomic.AddInt32(&called, 1)
		return v, nil
	}

	{
		out, err := ParallelMap[int, int](ctx, 4, nil, fn)
		if err != nil {
			t.Fatalf("nil input: unexpected error: %v", err)
		}
		if len(out) != 0 {
			t.Fatalf("nil input: len(out)=%d, want 0", len(out))
		}
	}
	{
		out, err := ParallelMap[int, int](ctx, 4, []int{}, fn)
		if err != nil {
			t.Fatalf("empty input: unexpected error: %v", err)
		}
		if len(out) != 0 {
			t.Fatalf("empty input: len(out)=%d, want 0", len(out))
		}
	}

	if got := atomic.LoadInt32(&called); got != 0 {
		t.Fatalf("fn called %d times for empty input, want 0", got)
	}
}

func TestParallelMap_OrderPreserved(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	in := make([]int, 40)
	for i := range in {
		in[i] = i
	}

	fn := func(ctx context.Context, v int) (int, error) {
		// Большие v быстрее, завершение будет "вразнобой".
		delay := time.Duration(len(in)-v) * 2 * time.Millisecond
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(delay):
		}
		return v * 10, nil
	}

	out, err := ParallelMap[int, int](ctx, 6, in, fn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != len(in) {
		t.Fatalf("len(out)=%d, want %d", len(out), len(in))
	}
	for i := range in {
		want := in[i] * 10
		if out[i] != want {
			t.Fatalf("out[%d]=%d, want %d (order must match input)", i, out[i], want)
		}
	}
}

func TestParallelMap_MaxParallelismRespected_HardBarrier(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	const workers = 7
	in := make([]int, 3000)
	for i := range in {
		in[i] = i
	}

	var started int32
	block := make(chan struct{})

	fn := func(ctx context.Context, v int) (int, error) {
		atomic.AddInt32(&started, 1)
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-block:
		}
		return v, nil
	}

	type res struct {
		out []int
		err error
	}
	done := make(chan res, 1)
	go func() {
		out, err := ParallelMap[int, int](ctx, workers, in, fn)
		done <- res{out: out, err: err}
	}()

	waitAtLeast(t, &started, workers, 300*time.Millisecond)

	// Если стартует больше чем workers одновременно (например, горутина на элемент), started начнёт расти.
	time.Sleep(80 * time.Millisecond)
	if got := atomic.LoadInt32(&started); got > workers {
		close(block)
		t.Fatalf("started=%d, want <= %d (must not start more than workers concurrently)", got, workers)
	}

	close(block)

	r := <-done
	if r.err != nil {
		t.Fatalf("unexpected error: %v", r.err)
	}
	if len(r.out) != len(in) {
		t.Fatalf("len(out)=%d, want %d", len(r.out), len(in))
	}
}

func TestParallelMap_CancelOnError_StopsStartingNewWork(t *testing.T) {
	t.Parallel()

	// Важное: если cancel на ошибке НЕ сделан, задачи зависнут на <-ctx.Done(),
	// и ParallelMap не завершится "быстро". Чтобы тест не зависал навечно,
	// мы запускаем ParallelMap в отдельной горутине и ставим таймаут ожидания.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const workers = 8
	const n = 2000
	const errAt = 5

	in := make([]int, n)
	for i := range in {
		in[i] = i
	}

	sentinel := errors.New("boom")

	var started int32
	var startedAfter int32

	fn := func(ctx context.Context, v int) (int, error) {
		atomic.AddInt32(&started, 1)

		if v == errAt {
			// Дать шанс стартовать воркерам, чтобы отмена была проверяемой.
			waitAtLeast(t, &started, workers, 200*time.Millisecond)
			return 0, sentinel
		}

		if v > errAt {
			atomic.AddInt32(&startedAfter, 1)
		}

		<-ctx.Done()
		return 0, ctx.Err()
	}

	errCh := make(chan error, 1)
	go func() {
		_, err := ParallelMap[int, int](ctx, workers, in, fn)
		errCh <- err
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, sentinel) {
			t.Fatalf("err=%v, want sentinel", err)
		}
	case <-time.After(600 * time.Millisecond):
		// Даже если реализация сломана, мы её “добьём” внешним cancel, чтобы тест не завис.
		cancel()
		t.Fatalf("did not return in time (cancel/stop likely broken)")
	}

	// После ошибки не должно “прокручиваться” много задач после errAt.
	if got := atomic.LoadInt32(&startedAfter); got > int32(workers+5) {
		t.Fatalf("startedAfter=%d, want <= %d (should not keep processing after cancel)", got, workers+5)
	}
}

func TestParallelMap_ContextCanceledBeforeStart(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	in := []int{1, 2, 3, 4}

	var called int32
	fn := func(ctx context.Context, v int) (int, error) {
		atomic.AddInt32(&called, 1)
		return v, nil
	}

	_, err := ParallelMap[int, int](ctx, 4, in, fn)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v, want context.Canceled", err)
	}
	if got := atomic.LoadInt32(&called); got != 0 {
		t.Fatalf("fn called %d times, want 0 when ctx already canceled", got)
	}
}

func TestParallelMap_ContextCanceledDuringWork(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	in := make([]int, 200)
	for i := range in {
		in[i] = i
	}

	fn := func(ctx context.Context, v int) (int, error) {
		<-ctx.Done()
		return 0, ctx.Err()
	}

	time.AfterFunc(30*time.Millisecond, cancel)

	_, err := ParallelMap[int, int](ctx, 10, in, fn)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v, want context.Canceled", err)
	}
}

func TestParallelMap_NoGoroutineLeak_Indirect(t *testing.T) {
	t.Parallel()

	before := runtime.NumGoroutine()

	const iters = 60
	for i := 0; i < iters; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)

		in := make([]int, 120)
		for j := range in {
			in[j] = j
		}

		sentinel := errors.New("err")

		fn := func(ctx context.Context, v int) (int, error) {
			if v == 0 {
				return 0, sentinel
			}
			<-ctx.Done()
			return 0, ctx.Err()
		}

		_, _ = ParallelMap[int, int](ctx, 10, in, fn)
		cancel()
	}

	runtime.GC()
	time.Sleep(80 * time.Millisecond)

	after := runtime.NumGoroutine()
	if after > before+100 {
		t.Fatalf("goroutines grew too much: before=%d after=%d (possible leak)", before, after)
	}
}

func waitAtLeast(t *testing.T, v *int32, want int32, d time.Duration) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(v) >= want {
			return
		}
		time.Sleep(1 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for >=%d, got %d", want, atomic.LoadInt32(v))
}

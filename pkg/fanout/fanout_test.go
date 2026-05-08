// pkg/fanout/fanout_test.go
package fanout

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

func TestRun_EmptyTasks(t *testing.T) {
	t.Run("nil slice", func(t *testing.T) {
		before := runtime.NumGoroutine()
		results, err := Run[int](context.Background(), 4, nil)
		after := runtime.NumGoroutine()
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if results != nil {
			t.Fatalf("results = %v, want nil", results)
		}
		if after > before {
			t.Errorf("goroutines leaked: before=%d after=%d", before, after)
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		results, err := Run[int](context.Background(), 4, []Task[int]{})
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if results != nil {
			t.Fatalf("results = %v, want nil", results)
		}
	})
}

func TestRun_AllSuccess(t *testing.T) {
	const n = 10
	tasks := make([]Task[int], n)
	for i := 0; i < n; i++ {
		i := i
		tasks[i] = func(ctx context.Context) (int, error) {
			return i * 10, nil
		}
	}

	results, err := Run(context.Background(), 4, tasks)
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if len(results) != n {
		t.Fatalf("len(results) = %d, want %d", len(results), n)
	}
	for i, r := range results {
		if r.Index != i {
			t.Errorf("results[%d].Index = %d, want %d", i, r.Index, i)
		}
		if r.Err != nil {
			t.Errorf("results[%d].Err = %v, want nil", i, r.Err)
		}
		if r.Value != i*10 {
			t.Errorf("results[%d].Value = %d, want %d", i, r.Value, i*10)
		}
	}
}

func TestRun_PerTaskErrorIsolated(t *testing.T) {
	wantErr := errors.New("idx-2 boom")
	tasks := []Task[int]{
		func(ctx context.Context) (int, error) { return 0, nil },
		func(ctx context.Context) (int, error) { return 10, nil },
		func(ctx context.Context) (int, error) { return 0, wantErr },
		func(ctx context.Context) (int, error) { return 30, nil },
		func(ctx context.Context) (int, error) { return 40, nil },
	}

	results, err := Run(context.Background(), 2, tasks)
	if err != nil {
		t.Fatalf("top err = %v, want nil (collect-all)", err)
	}
	for i, r := range results {
		switch i {
		case 2:
			if !errors.Is(r.Err, wantErr) {
				t.Errorf("results[2].Err = %v, want %v", r.Err, wantErr)
			}
		default:
			if r.Err != nil {
				t.Errorf("results[%d].Err = %v, want nil (sibling unaffected)", i, r.Err)
			}
		}
	}
}

func TestRun_ResultsOrderedByInputIndex(t *testing.T) {
	const n = 5
	tasks := make([]Task[int], n)
	for i := 0; i < n; i++ {
		i := i
		tasks[i] = func(ctx context.Context) (int, error) {
			time.Sleep(time.Duration(n-i) * 5 * time.Millisecond)
			return i, nil
		}
	}
	results, err := Run(context.Background(), n, tasks)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	for i, r := range results {
		if r.Index != i || r.Value != i {
			t.Errorf("results[%d] = {Index:%d Value:%d}, want {Index:%d Value:%d}",
				i, r.Index, r.Value, i, i)
		}
	}
}

func TestRun_RespectsMaxConcurrency(t *testing.T) {
	const max = 3
	const n = 10
	var inFlight, peak atomic.Int32

	tasks := make([]Task[int], n)
	for i := 0; i < n; i++ {
		tasks[i] = func(ctx context.Context) (int, error) {
			cur := inFlight.Add(1)
			defer inFlight.Add(-1)
			for {
				p := peak.Load()
				if cur <= p || peak.CompareAndSwap(p, cur) {
					break
				}
			}
			time.Sleep(20 * time.Millisecond)
			return 0, nil
		}
	}

	if _, err := Run(context.Background(), max, tasks); err != nil {
		t.Fatalf("err = %v", err)
	}
	if got := peak.Load(); got > max {
		t.Errorf("peak in-flight = %d, want <= %d", got, max)
	}
}

func TestRun_UnlimitedConcurrencyWhenZero(t *testing.T) {
	const n = 8
	var inFlight, peak atomic.Int32
	start := make(chan struct{})

	tasks := make([]Task[int], n)
	for i := 0; i < n; i++ {
		tasks[i] = func(ctx context.Context) (int, error) {
			<-start
			cur := inFlight.Add(1)
			defer inFlight.Add(-1)
			for {
				p := peak.Load()
				if cur <= p || peak.CompareAndSwap(p, cur) {
					break
				}
			}
			time.Sleep(20 * time.Millisecond)
			return 0, nil
		}
	}

	done := make(chan struct{})
	go func() {
		_, err := Run(context.Background(), 0, tasks)
		if err != nil {
			t.Errorf("err = %v", err)
		}
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	close(start)
	<-done

	if got := peak.Load(); got != n {
		t.Errorf("peak in-flight = %d, want %d (unlimited)", got, n)
	}
}

func TestRun_CtxCancelledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	const n = 5
	var ran atomic.Int32
	tasks := make([]Task[int], n)
	for i := 0; i < n; i++ {
		tasks[i] = func(ctx context.Context) (int, error) {
			ran.Add(1)
			return 99, nil
		}
	}

	results, err := Run(ctx, 0, tasks)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("top err = %v, want context.Canceled", err)
	}
	if len(results) != n {
		t.Fatalf("len(results) = %d, want %d", len(results), n)
	}
	for i, r := range results {
		if !errors.Is(r.Err, context.Canceled) {
			t.Errorf("results[%d].Err = %v, want Canceled", i, r.Err)
		}
		if r.Value != 0 {
			t.Errorf("results[%d].Value = %v, want zero", i, r.Value)
		}
	}
	if ran.Load() != 0 {
		t.Errorf("tasks ran = %d, want 0 (none should execute)", ran.Load())
	}
}

func TestRun_CtxCancelledMidFlight(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	const n = 6
	tasks := make([]Task[int], n)
	for i := 0; i < n; i++ {
		i := i
		tasks[i] = func(ctx context.Context) (int, error) {
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-time.After(200 * time.Millisecond):
				return i, nil
			}
		}
	}

	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	results, err := Run(ctx, 2, tasks)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("top err = %v, want Canceled", err)
	}
	if len(results) != n {
		t.Fatalf("len(results) = %d, want %d", len(results), n)
	}
	for i, r := range results {
		if !errors.Is(r.Err, context.Canceled) {
			t.Errorf("results[%d].Err = %v, want Canceled", i, r.Err)
		}
	}
}

func TestRun_PanicCapturedAsTaskPanicError(t *testing.T) {
	tasks := []Task[int]{
		func(ctx context.Context) (int, error) { return 1, nil },
		func(ctx context.Context) (int, error) { panic("boom") },
		func(ctx context.Context) (int, error) { return 3, nil },
	}

	results, err := Run(context.Background(), 2, tasks)
	if err != nil {
		t.Fatalf("top err = %v, want nil", err)
	}

	if results[0].Err != nil || results[0].Value != 1 {
		t.Errorf("results[0] = %+v, want {Value:1, Err:nil}", results[0])
	}
	if results[2].Err != nil || results[2].Value != 3 {
		t.Errorf("results[2] = %+v, want {Value:3, Err:nil}", results[2])
	}

	var panicErr *ErrTaskPanic
	if !errors.As(results[1].Err, &panicErr) {
		t.Fatalf("results[1].Err = %v, want *ErrTaskPanic", results[1].Err)
	}
	if panicErr.Recovered != "boom" {
		t.Errorf("Recovered = %v, want %q", panicErr.Recovered, "boom")
	}
	if len(panicErr.Stack) == 0 {
		t.Errorf("Stack is empty")
	}
	if results[1].Value != 0 {
		t.Errorf("results[1].Value = %v, want zero", results[1].Value)
	}
}

func TestRun_FailFast_CancelsRemainingTasks(t *testing.T) {
	wantErr := errors.New("idx-0 boom")
	const n = 6
	var siblingsCompleted atomic.Int32
	var siblingsReady atomic.Int32

	started := make(chan struct{})
	tasks := make([]Task[int], n)

	// task 0: wait for all siblings to be in their select, then return wantErr → triggers fail-fast cancel
	tasks[0] = func(ctx context.Context) (int, error) {
		<-started
		return 0, wantErr
	}

	// siblings 1..n-1: count themselves in, then select on ctx.Done vs 2s timer
	for i := 1; i < n; i++ {
		i := i
		tasks[i] = func(ctx context.Context) (int, error) {
			siblingsReady.Add(1)
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-time.After(2 * time.Second):
				siblingsCompleted.Add(1)
				return i, nil
			}
		}
	}

	// Coordinator: wait until all n-1 siblings are in their select, then release task 0
	go func() {
		for siblingsReady.Load() < int32(n-1) {
			time.Sleep(time.Millisecond)
		}
		// small grace period to ensure they're past `siblingsReady.Add(1)` and into select
		time.Sleep(10 * time.Millisecond)
		close(started)
	}()

	// maxConcurrency=0 (unlimited) so all 6 goroutines spawn and run their task bodies concurrently
	results, err := Run(context.Background(), 0, tasks, WithFailFast())
	if err != nil {
		t.Fatalf("top err = %v, want nil (outer ctx not cancelled)", err)
	}
	if !errors.Is(results[0].Err, wantErr) {
		t.Errorf("results[0].Err = %v, want %v", results[0].Err, wantErr)
	}
	for i := 1; i < n; i++ {
		if !errors.Is(results[i].Err, context.Canceled) {
			t.Errorf("results[%d].Err = %v, want context.Canceled", i, results[i].Err)
		}
	}
	if siblingsCompleted.Load() != 0 {
		t.Errorf("siblings completed = %d, want 0 (all cancelled before timer fired)", siblingsCompleted.Load())
	}
}

func TestRun_FailFast_NoErrorIsNoOp(t *testing.T) {
	const n = 5
	tasks := make([]Task[int], n)
	for i := 0; i < n; i++ {
		i := i
		tasks[i] = func(ctx context.Context) (int, error) { return i, nil }
	}

	results, err := Run(context.Background(), 2, tasks, WithFailFast())
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	for i, r := range results {
		if r.Err != nil || r.Value != i {
			t.Errorf("results[%d] = %+v, want {Value:%d, Err:nil}", i, r, i)
		}
	}
}

func TestRun_NoGoroutineLeak(t *testing.T) {
	const n = 20

	warm := make([]Task[int], 5)
	for i := range warm {
		warm[i] = func(ctx context.Context) (int, error) { return 0, nil }
	}
	_, _ = Run(context.Background(), 3, warm)

	time.Sleep(50 * time.Millisecond)
	before := runtime.NumGoroutine()

	tasks := make([]Task[int], n)
	for i := 0; i < n; i++ {
		i := i
		tasks[i] = func(ctx context.Context) (int, error) { return i, nil }
	}
	if _, err := Run(context.Background(), 4, tasks); err != nil {
		t.Fatalf("err = %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	after := runtime.NumGoroutine()

	if after > before+2 {
		t.Errorf("goroutine leak: before=%d after=%d", before, after)
	}
}

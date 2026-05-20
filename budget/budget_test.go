package budget

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBudget_ZeroCap_NoEnforcement(t *testing.T) {
	t.Parallel()
	tr := NewStrict(Budget{}) // all zeros = no caps
	for i := 0; i < 1000; i++ {
		if err := tr.Charge(Usage{Tokens: 1_000_000, Calls: 1, Wall: time.Second, Cost: 999.0}); err != nil {
			t.Fatalf("iter %d: unexpected error on zero-cap charge: %v", i, err)
		}
	}
	snap := tr.Snapshot()
	if snap.Calls != 1000 {
		t.Errorf("Snapshot().Calls = %d, want 1000", snap.Calls)
	}
	if snap.Tokens != 1000*1_000_000 {
		t.Errorf("Snapshot().Tokens = %d, want %d", snap.Tokens, 1000*1_000_000)
	}
}

func TestCharge_Calls(t *testing.T) {
	t.Parallel()
	tr := NewStrict(Budget{MaxCalls: 3})
	for i := 0; i < 3; i++ {
		if err := tr.Charge(Usage{Calls: 1}); err != nil {
			t.Fatalf("charge %d: unexpected error: %v", i+1, err)
		}
	}
	err := tr.Charge(Usage{Calls: 1})
	if err == nil {
		t.Fatalf("4th charge: expected ErrCallsExceeded, got nil")
	}
	if !errors.Is(err, ErrCallsExceeded) {
		t.Errorf("expected errors.Is(err, ErrCallsExceeded), err=%v", err)
	}
	if !errors.Is(err, ErrBudgetExceeded) {
		t.Errorf("expected errors.Is(err, ErrBudgetExceeded), err=%v", err)
	}
	if snap := tr.Snapshot(); snap.Calls != 3 {
		t.Errorf("denied charge mutated state: Snapshot().Calls = %d, want 3", snap.Calls)
	}
}

func TestCharge_Tokens(t *testing.T) {
	t.Parallel()
	tr := NewStrict(Budget{MaxTokens: 100})
	if err := tr.Charge(Usage{Tokens: 40}); err != nil {
		t.Fatalf("charge 1: %v", err)
	}
	if err := tr.Charge(Usage{Tokens: 40}); err != nil {
		t.Fatalf("charge 2: %v", err)
	}
	err := tr.Charge(Usage{Tokens: 40})
	if !errors.Is(err, ErrTokensExceeded) {
		t.Fatalf("3rd charge: expected ErrTokensExceeded, got %v", err)
	}
	if snap := tr.Snapshot(); snap.Tokens != 80 {
		t.Errorf("denied charge mutated tokens: Snapshot().Tokens = %d, want 80", snap.Tokens)
	}
}

func TestCharge_Cost(t *testing.T) {
	t.Parallel()
	tr := NewStrict(Budget{MaxCost: 1.0})
	if err := tr.Charge(Usage{Cost: 0.4}); err != nil {
		t.Fatalf("charge 1: %v", err)
	}
	if err := tr.Charge(Usage{Cost: 0.4}); err != nil {
		t.Fatalf("charge 2: %v", err)
	}
	err := tr.Charge(Usage{Cost: 0.4})
	if !errors.Is(err, ErrCostExceeded) {
		t.Fatalf("3rd charge: expected ErrCostExceeded, got %v", err)
	}
}

func TestCharge_Wall_Sentinel(t *testing.T) {
	t.Parallel()
	tr := NewStrict(Budget{MaxWall: 100 * time.Millisecond})
	if err := tr.Charge(Usage{Wall: 40 * time.Millisecond}); err != nil {
		t.Fatalf("charge 1: %v", err)
	}
	if err := tr.Charge(Usage{Wall: 40 * time.Millisecond}); err != nil {
		t.Fatalf("charge 2: %v", err)
	}
	err := tr.Charge(Usage{Wall: 40 * time.Millisecond})
	if !errors.Is(err, ErrWallExceeded) {
		t.Fatalf("3rd charge: expected ErrWallExceeded, got %v", err)
	}
}

func TestCharge_OrderingMatters(t *testing.T) {
	t.Parallel()
	tr := NewStrict(Budget{MaxCalls: 1, MaxTokens: 1})
	if err := tr.Charge(Usage{Calls: 1, Tokens: 1}); err != nil {
		t.Fatalf("first charge: %v", err)
	}
	err := tr.Charge(Usage{Calls: 1, Tokens: 100})
	if !errors.Is(err, ErrCallsExceeded) {
		t.Fatalf("expected ErrCallsExceeded (Calls checked before Tokens), got %v", err)
	}
	if errors.Is(err, ErrTokensExceeded) {
		t.Errorf("should NOT be ErrTokensExceeded — order is Calls→Tokens→Cost→Wall")
	}
}

func TestRemaining(t *testing.T) {
	t.Parallel()
	type charge struct{ u Usage }
	cases := []struct {
		name    string
		budget  Budget
		charges []charge
		want    Usage
	}{
		{
			name:   "zero-budget-zero-remaining",
			budget: Budget{},
			want:   Usage{},
		},
		{
			name:   "partial-consumption",
			budget: Budget{MaxTokens: 100, MaxCalls: 10, MaxCost: 5.0, MaxWall: time.Second},
			charges: []charge{
				{u: Usage{Tokens: 30, Calls: 2, Cost: 1.0, Wall: 200 * time.Millisecond}},
			},
			want: Usage{Tokens: 70, Calls: 8, Cost: 4.0, Wall: 800 * time.Millisecond},
		},
		{
			name:   "over-consumption-clamps-zero",
			budget: Budget{MaxTokens: 10, MaxCalls: 1},
			// Charge a single Usage that does NOT exceed any cap — to
			// land exactly at cap and observe zero remaining.
			charges: []charge{
				{u: Usage{Tokens: 10, Calls: 1}},
			},
			want: Usage{},
		},
		{
			name:   "zero-cap-on-one-dim-yields-zero-on-that-dim",
			budget: Budget{MaxTokens: 100, MaxCalls: 0},
			charges: []charge{
				{u: Usage{Tokens: 30, Calls: 5}},
			},
			// Calls cap is 0 → Remaining.Calls = 0 (zero cap → zero remaining).
			want: Usage{Tokens: 70, Calls: 0},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tr := NewStrict(tc.budget)
			for _, c := range tc.charges {
				_ = tr.Charge(c.u) // ignore err — we only care about Remaining()
			}
			got := tr.Remaining(tc.budget)
			if got != tc.want {
				t.Errorf("Remaining = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestSnapshot_AfterDeniedCharge(t *testing.T) {
	t.Parallel()
	tr := NewStrict(Budget{MaxCalls: 2})
	_ = tr.Charge(Usage{Calls: 1})
	_ = tr.Charge(Usage{Calls: 1})
	before := tr.Snapshot()
	err := tr.Charge(Usage{Calls: 1, Tokens: 50, Cost: 1.5, Wall: time.Second})
	if !errors.Is(err, ErrCallsExceeded) {
		t.Fatalf("expected ErrCallsExceeded, got %v", err)
	}
	after := tr.Snapshot()
	if before != after {
		t.Errorf("denied charge mutated Snapshot: before=%+v after=%+v", before, after)
	}
}

func TestWithBudget_ContextDeadline(t *testing.T) {
	t.Parallel()
	parent := context.Background()
	ctx, _ := WithBudget(parent, Budget{MaxWall: 10 * time.Millisecond})
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatalf("expected ctx to have a deadline")
	}
	if time.Until(deadline) > 50*time.Millisecond {
		t.Errorf("deadline too far in the future: %s", time.Until(deadline))
	}
	time.Sleep(25 * time.Millisecond)
	if err := ctx.Err(); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("ctx.Err() = %v, want DeadlineExceeded", err)
	}
}

func TestWithBudget_NoWall_NoDeadline(t *testing.T) {
	t.Parallel()
	ctx, _ := WithBudget(context.Background(), Budget{MaxCalls: 5})
	if _, has := ctx.Deadline(); has {
		t.Errorf("expected NO deadline when MaxWall == 0")
	}
}

func TestFrom_RoundTrip(t *testing.T) {
	t.Parallel()
	ctx, t1 := WithBudget(context.Background(), Budget{MaxCalls: 5})
	t2, ok := From(ctx)
	if !ok {
		t.Fatalf("From: expected ok=true")
	}
	if t1 != t2 {
		t.Errorf("From returned a different Tracker instance: t1=%p t2=%p", t1, t2)
	}
}

func TestFrom_Absent(t *testing.T) {
	t.Parallel()
	tr, ok := From(context.Background())
	if ok {
		t.Errorf("From on bare ctx: expected ok=false, got ok=true tr=%v", tr)
	}
	if tr != nil {
		t.Errorf("From on bare ctx: expected nil tracker, got %v", tr)
	}
	// nil ctx defensive case
	tr2, ok2 := From(nil)
	if ok2 || tr2 != nil {
		t.Errorf("From(nil): expected (nil, false), got (%v, %v)", tr2, ok2)
	}
}

// TestCharge_Concurrent_Race is the load-bearing concurrency test —
// 100 goroutines × 100 charges of Usage{Calls:1, Tokens:1} against a
// Budget with MaxCalls/MaxTokens of 5000. We assert (1) the cap fires
// (Snapshot().Calls != 10000, i.e., NOT every charge succeeded) and
// (2) successes + ErrCallsExceeded denials sum to exactly 10000. Must
// pass under `go test -race`.
func TestCharge_Concurrent_Race(t *testing.T) {
	t.Parallel()
	const (
		goroutines    = 100
		perGoroutine  = 100
		totalAttempts = goroutines * perGoroutine // 10000
		cap           = 5000
	)
	tr := NewStrict(Budget{MaxCalls: cap, MaxTokens: cap})

	var successes int64
	var denials int64
	var otherErrs int64

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				err := tr.Charge(Usage{Calls: 1, Tokens: 1})
				switch {
				case err == nil:
					atomic.AddInt64(&successes, 1)
				case errors.Is(err, ErrCallsExceeded) || errors.Is(err, ErrTokensExceeded):
					atomic.AddInt64(&denials, 1)
				default:
					atomic.AddInt64(&otherErrs, 1)
				}
			}
		}()
	}
	wg.Wait()

	if otherErrs != 0 {
		t.Fatalf("got %d unexpected errors", otherErrs)
	}
	if successes+denials != int64(totalAttempts) {
		t.Errorf("successes(%d) + denials(%d) = %d, want %d", successes, denials, successes+denials, totalAttempts)
	}
	snap := tr.Snapshot()
	if snap.Calls == totalAttempts {
		t.Errorf("Snapshot().Calls = %d (no cap fired) — expected cap to fire", snap.Calls)
	}
	// Stronger invariant: under check-before-commit, the committed
	// Calls count must equal the success count and must be ≤ cap.
	if int64(snap.Calls) != successes {
		t.Errorf("Snapshot().Calls = %d, want successes = %d", snap.Calls, successes)
	}
	if snap.Calls > cap {
		t.Errorf("Snapshot().Calls = %d exceeded cap %d (check-before-commit violated)", snap.Calls, cap)
	}
}

func TestErrSentinelsWrapUmbrella(t *testing.T) {
	t.Parallel()
	dims := []error{ErrTokensExceeded, ErrCallsExceeded, ErrWallExceeded, ErrCostExceeded}
	for _, e := range dims {
		if !errors.Is(e, ErrBudgetExceeded) {
			t.Errorf("errors.Is(%v, ErrBudgetExceeded) = false; want true", e)
		}
	}
	// One-way wrap: the umbrella is not "an" instance of any specific
	// dimension.
	if errors.Is(ErrBudgetExceeded, ErrTokensExceeded) {
		t.Errorf("errors.Is(ErrBudgetExceeded, ErrTokensExceeded) = true; want false (one-way wrap)")
	}
}

func TestNewSoft_NeverErrors_StillAccumulates(t *testing.T) {
	t.Parallel()
	soft := NewSoft(Budget{MaxCalls: 1, MaxTokens: 1, MaxCost: 0.01, MaxWall: time.Nanosecond})
	for i := 0; i < 10; i++ {
		if err := soft.Charge(Usage{Calls: 1, Tokens: 100, Cost: 1.0, Wall: time.Second}); err != nil {
			t.Fatalf("soft.Charge %d: unexpected error %v", i, err)
		}
	}
	snap := soft.Snapshot()
	if snap.Calls != 10 || snap.Tokens != 1000 {
		t.Errorf("soft.Snapshot = %+v; expected Calls=10 Tokens=1000", snap)
	}
}

func TestWithTracker_NoDeadline(t *testing.T) {
	t.Parallel()
	soft := NewSoft(Budget{MaxWall: 10 * time.Millisecond})
	ctx := WithTracker(context.Background(), soft)
	if _, has := ctx.Deadline(); has {
		t.Errorf("WithTracker derived a deadline; it must not")
	}
	got, ok := From(ctx)
	if !ok || got != soft {
		t.Errorf("From after WithTracker: ok=%v got=%v want=%v", ok, got, soft)
	}
}

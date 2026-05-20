package budget

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ErrBudgetExceeded is the umbrella sentinel for a budget cap being
// exceeded. Each dimension-specific error (ErrTokensExceeded,
// ErrCallsExceeded, ErrWallExceeded, ErrCostExceeded) wraps this value,
// so callers MAY do an umbrella check via
// errors.Is(err, ErrBudgetExceeded) without caring which dimension
// tripped, or a dim-specific check via errors.Is(err, ErrCallsExceeded)
// to react to a particular cap.
var ErrBudgetExceeded = errors.New("budget: exceeded")

// ErrTokensExceeded is returned by Tracker.Charge when accumulated
// Usage.Tokens would exceed Budget.MaxTokens. Wraps ErrBudgetExceeded.
var ErrTokensExceeded = fmt.Errorf("%w: tokens", ErrBudgetExceeded)

// ErrCallsExceeded is returned by Tracker.Charge when accumulated
// Usage.Calls would exceed Budget.MaxCalls. Wraps ErrBudgetExceeded.
// Per Q2 (operator-confirmed 2026-05-20), Calls counts attempts:
// a denied charge still consumed one call against the cap on the
// next attempt's pre-charge path — see Tracker.Charge.
var ErrCallsExceeded = fmt.Errorf("%w: calls", ErrBudgetExceeded)

// ErrWallExceeded is returned by Tracker.Charge when accumulated
// Usage.Wall would exceed Budget.MaxWall. Note: in practice the
// chokepoint (35-02) enforces wall-clock primarily via the
// context.WithDeadline derived in WithBudget; this sentinel exists
// for callers that track wall via Charge explicitly. Wraps
// ErrBudgetExceeded.
var ErrWallExceeded = fmt.Errorf("%w: wall", ErrBudgetExceeded)

// ErrCostExceeded is returned by Tracker.Charge when accumulated
// Usage.Cost would exceed Budget.MaxCost. Wraps ErrBudgetExceeded.
var ErrCostExceeded = fmt.Errorf("%w: cost", ErrBudgetExceeded)

// Budget is the per-dimension cap policy. Zero on a field means "no cap
// for this dimension" — a Budget{} (zero value) imposes no caps at all.
//
// Negative values are undefined behavior in this slice (35-01); a
// Budget.Validate() method may be added in v1.3 if a real foot-gun
// surfaces (see 35-RESEARCH.md §"Open questions" Q5). Callers MUST pass
// non-negative values.
type Budget struct {
	// MaxTokens is the cap on accumulated Usage.Tokens. 0 = no cap.
	MaxTokens int
	// MaxCalls is the cap on Tracker.Charge attempts. 0 = no cap.
	// Q2: counts attempts (charge fires pre-call), not successes.
	MaxCalls int
	// MaxWall is the cap on accumulated Usage.Wall. 0 = no cap.
	// WithBudget additionally derives a context.WithDeadline of
	// time.Now().Add(MaxWall) when this field is positive.
	MaxWall time.Duration
	// MaxCost is the cap on accumulated Usage.Cost. 0 = no cap.
	MaxCost float64
}

// Usage is the unit of consumption charged against a Tracker. Distinct
// from llm.Usage and agents.Usage by design (Q1, operator-confirmed
// 2026-05-20) — the three packages model different concerns and the
// package selector disambiguates at the call site.
type Usage struct {
	// Tokens consumed by a single LLM call (typically prompt+completion).
	Tokens int
	// Calls is the number of LLM call attempts to charge. Callers in
	// the chokepoint pass Calls: 1 per pre-call charge.
	Calls int
	// Wall is the wall-clock duration consumed by a single LLM call.
	Wall time.Duration
	// Cost is the dollar (or other unit) cost of a single LLM call.
	// Pricing is computed upstream of this package (KC-4).
	Cost float64
}

// Tracker is the concurrency-safe budget bookkeeping interface. The
// default implementation (returned by NewStrict and NewSoft) uses
// atomic counters for Tokens/Calls and a mutex for Wall/Cost. All
// methods are safe for concurrent use from arbitrary goroutines.
type Tracker interface {
	// Charge attempts to record one call's worth of Usage against the
	// tracker. The strict implementation checks each dimension against
	// the bound Budget in this order — Calls, Tokens, Cost, Wall —
	// and returns the corresponding dim-specific sentinel on the first
	// cap exceeded WITHOUT mutating any counter (check-before-commit).
	// When all dimensions pass, the deltas are committed atomically and
	// nil is returned.
	//
	// Q2 (operator-confirmed 2026-05-20): Calls is incremented on the
	// pre-call path. A denied charge still costs one call against the
	// cap when the caller retries — i.e., the cap counts attempts, not
	// successes. The soft implementation (NewSoft) always commits and
	// returns nil.
	Charge(u Usage) error

	// Snapshot returns a copy of the currently accumulated Usage.
	// Atomic loads on the hot-path counters; the wall/cost fields are
	// read under the tracker's mutex.
	Snapshot() Usage

	// Remaining returns the per-dimension headroom relative to b,
	// floor-clamped to zero on each dimension (never negative). A zero
	// cap on b for any dimension yields zero remaining for that
	// dimension — "zero cap → zero remaining". Pass the same Budget
	// the tracker was constructed with for the canonical answer.
	Remaining(b Budget) Usage
}

// ctxKey is the unexported context-key type used to attach a Tracker.
// Package-private to prevent accidental collision across packages.
type ctxKey struct{}

// trackerKey is the single instance of ctxKey used for context.Value
// lookups. Stored as an unexported var so the key cannot leak.
var trackerKey = ctxKey{}

// strictTracker is the default Tracker implementation: Charge enforces
// the bound Budget and returns the dim-specific sentinel on cap exceeded.
// Counters for Tokens/Calls are int64 fields accessed via sync/atomic;
// Wall and Cost are guarded by mu.
type strictTracker struct {
	budget Budget
	tokens int64 // atomic
	calls  int64 // atomic
	mu     sync.Mutex
	wall   time.Duration
	cost   float64
}

// NewStrict returns a Tracker that returns a dim-specific sentinel
// error (wrapping ErrBudgetExceeded) when any cap on b would be
// exceeded. Tracker state is NOT mutated on a denied charge. This is
// the default semantics used by WithBudget.
func NewStrict(b Budget) Tracker {
	return &strictTracker{budget: b}
}

// NewSoft returns a Tracker whose Charge always returns nil but still
// accumulates Usage into Snapshot. Useful for observability-only
// deployments (collect spend without enforcing it) and for tests.
// Compose with WithTracker to attach to a context.
func NewSoft(b Budget) Tracker {
	return &softTracker{strictTracker{budget: b}}
}

// Charge implements the strict-mode check-before-commit semantics
// described on the Tracker interface. Order of checks: Calls → Tokens
// → Cost → Wall. On any cap exceeded the function returns the
// dim-specific sentinel without touching any counter.
//
// The full check-and-commit transaction is serialized under t.mu so
// concurrent callers cannot race past a cap (two goroutines each
// observing curCalls=4999 and both committing to 5001 would violate
// the cap). Snapshot reads remain lock-free via atomic loads — writes
// happen under mu using atomic stores so the happens-before is
// preserved.
func (t *strictTracker) Charge(u Usage) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	curCalls := atomic.LoadInt64(&t.calls)
	curTokens := atomic.LoadInt64(&t.tokens)

	wantCalls := curCalls + int64(u.Calls)
	wantTokens := curTokens + int64(u.Tokens)
	wantWall := t.wall + u.Wall
	wantCost := t.cost + u.Cost

	if t.budget.MaxCalls > 0 && wantCalls > int64(t.budget.MaxCalls) {
		return ErrCallsExceeded
	}
	if t.budget.MaxTokens > 0 && wantTokens > int64(t.budget.MaxTokens) {
		return ErrTokensExceeded
	}
	if t.budget.MaxCost > 0 && wantCost > t.budget.MaxCost {
		return ErrCostExceeded
	}
	if t.budget.MaxWall > 0 && wantWall > t.budget.MaxWall {
		return ErrWallExceeded
	}

	// All checks passed; commit deltas.
	atomic.StoreInt64(&t.calls, wantCalls)
	atomic.StoreInt64(&t.tokens, wantTokens)
	t.wall = wantWall
	t.cost = wantCost
	return nil
}

// Snapshot returns a value-copy of the current accumulated Usage.
func (t *strictTracker) Snapshot() Usage {
	tokens := atomic.LoadInt64(&t.tokens)
	calls := atomic.LoadInt64(&t.calls)
	t.mu.Lock()
	wall := t.wall
	cost := t.cost
	t.mu.Unlock()
	return Usage{
		Tokens: int(tokens),
		Calls:  int(calls),
		Wall:   wall,
		Cost:   cost,
	}
}

// Remaining computes b - snapshot(), floor-clamped to zero per
// dimension. Zero cap on any dimension yields zero remaining for that
// dimension.
func (t *strictTracker) Remaining(b Budget) Usage {
	snap := t.Snapshot()
	var rem Usage
	if b.MaxTokens > 0 {
		if r := b.MaxTokens - snap.Tokens; r > 0 {
			rem.Tokens = r
		}
	}
	if b.MaxCalls > 0 {
		if r := b.MaxCalls - snap.Calls; r > 0 {
			rem.Calls = r
		}
	}
	if b.MaxWall > 0 {
		if r := b.MaxWall - snap.Wall; r > 0 {
			rem.Wall = r
		}
	}
	if b.MaxCost > 0 {
		if r := b.MaxCost - snap.Cost; r > 0 {
			rem.Cost = r
		}
	}
	return rem
}

// softTracker accumulates Usage like strictTracker but Charge always
// returns nil regardless of caps.
type softTracker struct {
	strictTracker
}

// Charge accumulates u into the tracker and always returns nil. Cap
// checks are skipped; the bound Budget on the embedded strictTracker is
// still consulted by Remaining for headroom reporting. Serialized
// under the embedded mutex so concurrent soft charges don't tear
// the wall/cost fields.
func (t *softTracker) Charge(u Usage) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	atomic.AddInt64(&t.calls, int64(u.Calls))
	atomic.AddInt64(&t.tokens, int64(u.Tokens))
	t.wall += u.Wall
	t.cost += u.Cost
	return nil
}

// WithBudget is the one-liner common-case helper. It constructs a
// strict Tracker via NewStrict(b), attaches it to parent via
// WithTracker, and — when b.MaxWall > 0 — derives a
// context.WithDeadline(parent, time.Now().Add(b.MaxWall)) so wall-clock
// is enforced by ctx cancellation in addition to the per-Charge
// sentinel. The cancel func from WithDeadline is intentionally
// discarded: the deadline fires on its own and the tracker is the
// canonical source of "budget exhausted" for the non-wall dimensions.
// Callers who need a soft tracker should instead compose
// NewSoft + WithTracker themselves.
func WithBudget(parent context.Context, b Budget) (context.Context, Tracker) {
	t := NewStrict(b)
	ctx := parent
	if b.MaxWall > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(parent, time.Now().Add(b.MaxWall))
		_ = cancel // intentional: deadline fires on its own; tracker is the
		// canonical "budget exhausted" signal for non-wall dimensions.
	}
	return WithTracker(ctx, t), t
}

// WithTracker attaches an arbitrary Tracker to parent under an
// unexported context-key. Unlike WithBudget, it does NOT derive a
// deadline — soft-tracker callers who want wall-clock cancellation
// must wrap with context.WithDeadline themselves or use WithBudget.
func WithTracker(parent context.Context, t Tracker) context.Context {
	return context.WithValue(parent, trackerKey, t)
}

// From extracts the Tracker attached to ctx. It returns (nil, false)
// when no tracker is present — the chokepoint in 35-02 uses this
// no-op-when-absent guarantee to preserve "zero behavior change when
// no budget is set".
func From(ctx context.Context) (Tracker, bool) {
	if ctx == nil {
		return nil, false
	}
	v := ctx.Value(trackerKey)
	if v == nil {
		return nil, false
	}
	t, ok := v.(Tracker)
	if !ok {
		return nil, false
	}
	return t, true
}

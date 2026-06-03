// Package agents implements Agent paradigms (Simple/ReAct/Reflection/PlanAndSolve)
// and a Tool subsystem on top of pkg/llm. Subpackage of pkg/llm; inherits the
// same portability contract: no internal/* imports, no project-specific pkg/*,
// no business vocabulary.
package agents

import (
	"context"
	"errors"
)

// Agent, Tool, the trace data types (Result/Step/StepEvent/Usage), the StepKind
// enum, and the ErrToolNotFound/ErrEmptyInput sentinels now live in the leaf
// contract github.com/costa92/llm-agent-contract/agents and are re-exported here
// via aliases.go (same type identity, so all impls below compile unchanged).

// normalizedOnStep wraps a nil OnStep into a no-op so callers of runInternal
// can call onStep(s) without checking nil at every emission point. Each Agent's
// Run/RunStream entry calls this once at the boundary; runInternal stays clean.
func normalizedOnStep(cb func(Step)) func(Step) {
	if cb == nil {
		return func(Step) {}
	}
	return cb
}

// runStreamFromBlocking is the shared helper for each Agent's RunStream.
// It spawns a goroutine that runs runFn (which calls onStep at each step),
// pipes step events into a buffered channel, and emits a Done event on exit.
//
// Buffer size note (eng review 2026-04-27): 16 balances test consumers (slice
// append, slow) against typical SSE consumers (http.Flusher, fast). Slow
// consumers wired here will see producer back-pressure inside the agent loop —
// the next LLM call will wait for consumer drain. Not a bug; a trade-off.
// Slow consumers should run a draining goroutine themselves, or use the
// synchronous OnStep callback instead.
func runStreamFromBlocking(
	ctx context.Context,
	runFn func(ctx context.Context, onStep func(Step)) (Result, error),
) (<-chan StepEvent, error) {
	ch := make(chan StepEvent, 16)
	go func() {
		defer close(ch)
		cb := func(s Step) {
			select {
			case ch <- StepEvent{Step: s}:
			case <-ctx.Done():
			}
		}
		res, err := runFn(ctx, cb)
		// Build the single terminal event. Priority: err > ctx.Err > Final.
		// This avoids Trap 1 — emitting two Done events on the same run
		// when runFn races with ctx cancel (e.g. returns (res, nil) the
		// same instant ctx is canceled). Consumers using
		// `for ev := range ch` see exactly one Done event with the
		// most-informative error.
		final := StepEvent{Done: true}
		switch {
		case err != nil:
			final.Err = err
		case ctx.Err() != nil:
			final.Err = ctx.Err()
		default:
			final.Final = &res
		}
		// Best-effort send: buffer=16 means a sane consumer will accept
		// this. If the consumer dropped the channel we fall through
		// default rather than block. We intentionally do NOT select on
		// ctx.Done() here — when ctx is canceled mid-run we still WANT
		// the consumer to learn that, so a blocking case is wrong.
		select {
		case ch <- final:
		default:
		}
	}()
	return ch, nil
}

// Sentinel errors. Subpackage stays portable — does not import pkg/errors.
// Callers in internal/* translate via errors.Is at the boundary.
// ErrToolNotFound and ErrEmptyInput moved to the contract and are re-exported
// in aliases.go (identity-preserving, so errors.Is still matches across modules).
var (
	ErrMaxStepsExceeded      = errors.New("agents: max steps exceeded")
	ErrToolAlreadyRegistered = errors.New("agents: tool already registered")
	ErrPlanningFailed        = errors.New("agents: planning failed")
	ErrParseToolCall         = errors.New("agents: failed to parse tool call")
)

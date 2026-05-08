// Package agents implements Agent paradigms (Simple/ReAct/Reflection/PlanAndSolve)
// and a Tool subsystem on top of pkg/llm. Subpackage of pkg/llm; inherits the
// same portability contract: no internal/* imports, no project-specific pkg/*,
// no business vocabulary.
package agents

import (
	"context"
	"errors"
)

// Agent is the minimal contract every Agent implementation satisfies.
type Agent interface {
	Name() string
	Run(ctx context.Context, input string) (Result, error)
	// RunStream emits trace step events through a channel. The channel is closed
	// when the Agent finishes; the final event always has Done=true with either
	// Final or Err set. Phase 8 SSE handlers are the natural consumer; service
	// layers don't need to write Step→event conversion themselves.
	RunStream(ctx context.Context, input string) (<-chan StepEvent, error)
}

// StepEvent is the transport unit emitted by RunStream.
//
//   - Done = false: Step is an intermediate event, Final/Err are nil.
//   - Done = true: terminal event, exactly one of Final or Err is non-nil.
//   - Channel close after the terminal event signals no more events.
type StepEvent struct {
	Step  Step
	Done  bool
	Final *Result
	Err   error
}

// Result carries the final answer plus full trace and accumulated usage.
//
// Trace memory contract (eng review 2026-04-27): Result.Trace is a debug
// snapshot for synchronous Run() callers and has no size limit. Streaming
// consumers (RunStream / SSE / gRPC stream) should consume StepEvents only
// and ignore Result.Trace at the end — they're the same information twice.
// Phase 8 SSE handlers should discard res.Trace once the channel closes
// (events already flushed to client). High-concurrency services that ignore
// this rule end up holding 50–100 Steps (~4KB each) per in-flight handler
// — 100 concurrent handlers ≈ 40MB wasted.
type Result struct {
	Answer string
	Trace  []Step
	Usage  Usage
}

// Usage tracks LLM cost across a single Run.
type Usage struct {
	LLMCalls int
	Tokens   int
}

// Step is one entry in the trace. Kind decides which fields are meaningful.
type Step struct {
	Kind    StepKind
	Content string // Thought / Reflection / Plan body
	Tool    string // Action only
	Args    string // Action only — raw JSON string
	Result  string // Observation only
}

// StepKind enumerates trace step types.
type StepKind string

const (
	StepThought     StepKind = "thought"
	StepAction      StepKind = "action"
	StepObservation StepKind = "observation"
	StepReflection  StepKind = "reflection"
	StepPlan        StepKind = "plan"
	StepFinal       StepKind = "final"
)

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
		if err != nil {
			select {
			case ch <- StepEvent{Done: true, Err: err}:
			case <-ctx.Done():
			}
			return
		}
		select {
		case ch <- StepEvent{Done: true, Final: &res}:
		case <-ctx.Done():
		}
	}()
	return ch, nil
}

// Sentinel errors. Subpackage stays portable — does not import pkg/errors.
// Callers in internal/* translate via errors.Is at the boundary.
var (
	ErrMaxStepsExceeded      = errors.New("agents: max steps exceeded")
	ErrToolNotFound          = errors.New("agents: tool not found")
	ErrToolAlreadyRegistered = errors.New("agents: tool already registered")
	ErrPlanningFailed        = errors.New("agents: planning failed")
	ErrParseToolCall         = errors.New("agents: failed to parse tool call")
	ErrEmptyInput            = errors.New("agents: empty input")
)

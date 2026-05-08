// pkg/fanout/doc.go

// Package fanout is a stateless generic primitive for "N tasks -> N results
// with bounded concurrency" — the most common shape of ad-hoc fan-out scattered
// across this codebase before consolidation.
//
// # When to use
//
// Use Run when you have a finite slice of independent units of work, want
// bounded concurrency, and need each unit's outcome (value or error) back in
// input order. Examples: parallel tool calls, parallel LLM invocations,
// parallel scoring of evaluation items.
//
// # When NOT to use
//
//   - Long-lived background submission pool (use a goroutine pool like
//     panjf2000/ants instead).
//   - Streaming jobs of unknown size.
//   - When you only need ctx-bound concurrency and don't care about
//     per-task results — golang.org/x/sync/errgroup.Group with SetLimit
//     is simpler.
//
// # Compared to errgroup
//
//   - fanout always returns one Result per input Task, even on cancel/panic.
//     errgroup loses individual errors after the first one.
//   - fanout's top-level error is only ctx.Err(); per-task errors live in
//     Result.Err. errgroup conflates the two.
//   - fanout recovers panics into *ErrTaskPanic. errgroup propagates them
//     up the goroutine stack and crashes the process.
//   - fanout has WithFailFast() opt-in; errgroup is fail-fast by default.
//
// # Invariants
//
//   - len(results) == len(tasks), always (even when ctx is already cancelled).
//   - results[i].Index == i (no sort needed by callers).
//   - Run never panics due to a Task's panic.
//   - Top-level error is ctx.Err() or nil — never a Task's error.
//
// # Concurrency primitive
//
// Pure stdlib: sync.WaitGroup + chan struct{} semaphore. No external deps.
package fanout

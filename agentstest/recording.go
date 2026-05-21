package agentstest

import (
	"context"
	"encoding/json"
	"sync"

	agents "github.com/costa92/llm-agent"
)

// RecordedCall captures a single Execute invocation on a RecordingTool.
type RecordedCall struct {
	Args json.RawMessage
	Out  string
	Err  error
}

// RecordingTool wraps another agents.Tool and records every Execute
// call. Safe for concurrent use — callers can run the wrapped tool
// from multiple goroutines and read [Calls] afterwards.
//
// Typical use:
//
//	rec := agentstest.NewRecordingTool(agentstest.NewStubTool("upper", "OK"))
//	registry := agents.NewRegistry(rec)
//	_, _ = agent.Run(ctx, "...")
//	if got := rec.Calls(); len(got) != 1 {
//	    t.Fatalf("expected 1 call, got %d", len(got))
//	}
type RecordingTool struct {
	inner agents.Tool

	mu    sync.Mutex
	calls []RecordedCall
}

// NewRecordingTool wraps inner with call recording.
func NewRecordingTool(inner agents.Tool) *RecordingTool {
	return &RecordingTool{inner: inner}
}

// Name implements agents.Tool by delegating to the wrapped tool.
func (r *RecordingTool) Name() string { return r.inner.Name() }

// Description implements agents.Tool by delegating to the wrapped tool.
func (r *RecordingTool) Description() string { return r.inner.Description() }

// Schema implements agents.Tool by delegating to the wrapped tool.
func (r *RecordingTool) Schema() json.RawMessage { return r.inner.Schema() }

// Execute delegates to the wrapped tool and records the args/output/error.
func (r *RecordingTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	out, err := r.inner.Execute(ctx, args)
	r.mu.Lock()
	// Defensive copy of args — callers may reuse the backing slice.
	argsCopy := append(json.RawMessage(nil), args...)
	r.calls = append(r.calls, RecordedCall{Args: argsCopy, Out: out, Err: err})
	r.mu.Unlock()
	return out, err
}

// Calls returns a snapshot of every recorded call in invocation order.
// The returned slice is freshly allocated; mutating it does not affect
// future recordings.
func (r *RecordingTool) Calls() []RecordedCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]RecordedCall, len(r.calls))
	copy(out, r.calls)
	return out
}

// Reset drops all recorded calls.
func (r *RecordingTool) Reset() {
	r.mu.Lock()
	r.calls = nil
	r.mu.Unlock()
}

// Compile-time assertion: *RecordingTool satisfies agents.Tool.
var _ agents.Tool = (*RecordingTool)(nil)

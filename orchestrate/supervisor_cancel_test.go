package orchestrate

import (
	"context"
	"errors"
	"testing"

	agents "github.com/costa92/llm-agent"
)

// TestSupervisor_RunStream_EmitsDoneOnCancel (T3) mirrors the agent-level
// T1 against Supervisor.RunStream. Supervisor has a LOCAL reimplementation
// of runStreamFromBlocking (see supervisor.go Q2 comment) — without
// keeping both in lock-step, Supervisor cancel behavior would diverge from
// agent.RunStream and nested Supervisors would lose Done events.
func TestSupervisor_RunStream_EmitsDoneOnCancel(t *testing.T) {
	opts := validOpts()
	sup := NewSupervisor("sup", opts)

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := sup.RunStream(ctx, "seed")
	if err != nil {
		t.Fatalf("RunStream: %v", err)
	}
	// Cancel immediately. The state-graph plan-dispatch loop must observe
	// ctx.Done() at the first ctx-aware checkpoint and return ctx.Err().
	cancel()

	var events []agents.StepEvent
	for ev := range ch {
		events = append(events, ev)
	}

	if len(events) < 1 {
		t.Fatalf("got 0 events, want at least 1 terminal Done event")
	}
	last := events[len(events)-1]
	if !last.Done {
		t.Fatalf("last event Done = false, want true; events=%+v", events)
	}
	if !errors.Is(last.Err, context.Canceled) {
		t.Fatalf("last event Err = %v, want context.Canceled", last.Err)
	}
	if last.Final != nil {
		t.Fatalf("last event Final = %+v, want nil on cancel", last.Final)
	}
}

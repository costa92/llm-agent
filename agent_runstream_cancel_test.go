package agents

import (
	"context"
	"errors"
	"testing"
)

// TestRunStreamFromBlocking_EmitsDoneOnCancel (T1) verifies that when ctx is
// canceled mid-run, the channel receives a terminal StepEvent{Done: true,
// Err: ctx.Err()} BEFORE close. Previously the helper silently closed the
// channel on ctx.Done() — consumers using `for ev := range ch` had no way to
// distinguish a clean finish from a mid-stream cancel.
func TestRunStreamFromBlocking_EmitsDoneOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// runFn blocks on ctx until canceled, then returns (zero, ctx.Err()).
	runFn := func(ctx context.Context, onStep func(Step)) (Result, error) {
		<-ctx.Done()
		return Result{}, ctx.Err()
	}

	ch, err := runStreamFromBlocking(ctx, runFn)
	if err != nil {
		t.Fatalf("runStreamFromBlocking: %v", err)
	}

	// Trigger cancel.
	cancel()

	var events []StepEvent
	for ev := range ch {
		events = append(events, ev)
	}

	if len(events) < 1 {
		t.Fatalf("got %d events, want at least 1 terminal Done event", len(events))
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

// TestRunStreamFromBlocking_EmitsDoneOnSuccess (T2) is the happy-path
// regression guard for Trap 1: when runFn returns successfully and ctx is
// NOT canceled, the consumer must see exactly one terminal event with
// Final set and Err == nil. Without the err > ctx.Err > Final priority
// switch this test still passes because there is no concurrent cancel.
func TestRunStreamFromBlocking_EmitsDoneOnSuccess(t *testing.T) {
	ctx := context.Background()
	expected := Result{Answer: "ok"}
	runFn := func(ctx context.Context, onStep func(Step)) (Result, error) {
		return expected, nil
	}

	ch, err := runStreamFromBlocking(ctx, runFn)
	if err != nil {
		t.Fatalf("runStreamFromBlocking: %v", err)
	}

	var events []StepEvent
	for ev := range ch {
		events = append(events, ev)
	}

	if len(events) != 1 {
		t.Fatalf("got %d events, want exactly 1", len(events))
	}
	last := events[0]
	if !last.Done {
		t.Fatal("last event Done = false, want true")
	}
	if last.Err != nil {
		t.Fatalf("last event Err = %v, want nil", last.Err)
	}
	if last.Final == nil || last.Final.Answer != "ok" {
		t.Fatalf("last event Final = %+v, want Answer=ok", last.Final)
	}
}

// TestRunStreamFromBlocking_ErrTakesPriorityOverCtxErr guards Trap 1: when
// runFn returns a non-ctx error AND ctx is already canceled, the consumer
// should see the runFn err (not ctx.Err()). Priority: err > ctx.Err > Final.
func TestRunStreamFromBlocking_ErrTakesPriorityOverCtxErr(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before runFn even starts

	sentinel := errors.New("runFn failure")
	runFn := func(ctx context.Context, onStep func(Step)) (Result, error) {
		return Result{}, sentinel
	}

	ch, err := runStreamFromBlocking(ctx, runFn)
	if err != nil {
		t.Fatalf("runStreamFromBlocking: %v", err)
	}

	var events []StepEvent
	for ev := range ch {
		events = append(events, ev)
	}

	if len(events) < 1 {
		t.Fatalf("got %d events, want at least 1", len(events))
	}
	last := events[len(events)-1]
	if !last.Done {
		t.Fatalf("last event Done = false, want true")
	}
	if !errors.Is(last.Err, sentinel) {
		t.Fatalf("last event Err = %v, want sentinel runFn err", last.Err)
	}
}

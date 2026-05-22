package agents

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/costa92/llm-agent/budget"
)

func TestSimpleAgent_Run_TransparentlyForwards(t *testing.T) {
	llmMock := newScriptedLLM(textResp("hello world"))
	a := NewSimpleAgent(llmMock, SimpleOptions{SystemPrompt: "you are helpful"})

	res, err := a.Run(context.Background(), "hi")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Answer != "hello world" {
		t.Errorf("Answer = %q", res.Answer)
	}
	if res.Usage.LLMCalls != 1 {
		t.Errorf("LLMCalls = %d, want 1", res.Usage.LLMCalls)
	}
	if len(res.Trace) != 1 || res.Trace[0].Kind != StepFinal {
		t.Errorf("Trace = %+v", res.Trace)
	}
}

func TestSimpleAgent_Run_EmptyInput(t *testing.T) {
	a := NewSimpleAgent(newScriptedLLM(), SimpleOptions{})
	_, err := a.Run(context.Background(), "")
	if !errors.Is(err, ErrEmptyInput) {
		t.Errorf("err = %v, want ErrEmptyInput", err)
	}
}

func TestSimpleAgent_RunStream_DeliversFinal(t *testing.T) {
	llmMock := newScriptedLLM(textResp("answer"))
	a := NewSimpleAgent(llmMock, SimpleOptions{})
	ch, err := a.RunStream(context.Background(), "x")
	if err != nil {
		t.Fatal(err)
	}
	var events []StepEvent
	for ev := range ch {
		events = append(events, ev)
	}
	if len(events) < 2 {
		t.Fatalf("events = %d", len(events))
	}
	last := events[len(events)-1]
	if !last.Done || last.Final == nil || last.Final.Answer != "answer" {
		t.Errorf("last event = %+v", last)
	}
}

func TestSimpleAgent_RunStream_CtxCancelClosesChannel(t *testing.T) {
	llmMock := newScriptedLLM(textResp("ok"))
	a := NewSimpleAgent(llmMock, SimpleOptions{})
	ctx, cancel := context.WithCancel(context.Background())
	ch, _ := a.RunStream(ctx, "x")
	cancel()
	for range ch { // drain — channel must close, not deadlock
	}
}

// TestSimpleAgent_RunStream_CtxCancelEmitsDoneEvent (T4) is the end-to-end
// regression guard for P1-4. After cancel, the channel must yield a terminal
// StepEvent{Done: true, Err: ctx.Err()} so SSE handlers (and any
// `for ev := range ch` consumer) can distinguish a mid-stream cancel from a
// clean finish.
func TestSimpleAgent_RunStream_CtxCancelEmitsDoneEvent(t *testing.T) {
	llmMock := newScriptedLLM(textResp("ok"))
	a := NewSimpleAgent(llmMock, SimpleOptions{})
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := a.RunStream(ctx, "x")
	if err != nil {
		t.Fatalf("RunStream: %v", err)
	}
	cancel()

	var events []StepEvent
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
	// Either the scripted LLM returned ok before cancel landed (Final set,
	// Err nil), OR cancel fired first (Err = context.Canceled, Final nil).
	// Both are valid end-states; what's NOT valid is a silent close with
	// zero events.
	if last.Err == nil && last.Final == nil {
		t.Fatalf("last event has neither Err nor Final set: %+v", last)
	}
	if last.Err != nil && !errors.Is(last.Err, context.Canceled) {
		t.Fatalf("last event Err = %v, want context.Canceled or nil-with-Final", last.Err)
	}
}

func TestSimpleAgent_OnStep_Invoked(t *testing.T) {
	llmMock := newScriptedLLM(textResp("hi"))
	var got []Step
	a := NewSimpleAgent(llmMock, SimpleOptions{
		OnStep: func(s Step) { got = append(got, s) },
	})
	_, err := a.Run(context.Background(), "x")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Kind != StepFinal {
		t.Errorf("got = %v", got)
	}
}

func TestSimpleAgent_Name(t *testing.T) {
	a := NewSimpleAgent(newScriptedLLM(), SimpleOptions{Name: "custom"})
	if a.Name() != "custom" {
		t.Errorf("Name = %q", a.Name())
	}
	def := NewSimpleAgent(newScriptedLLM(), SimpleOptions{})
	if def.Name() != "simple" {
		t.Errorf("default Name = %q", def.Name())
	}
}

// TestSimple_BudgetExhaustion proves cross-Run budget enforcement (35-04 / CC-1).
//
// SimpleAgent makes exactly 1 LLM call per Run(), so we cannot exhaust a budget
// inside a single Run. Instead we run the agent twice in the same ctx with
// MaxCalls=1. The first Run charges 1 call and succeeds; the second Run's
// pre-call charge denies because wantCalls=2 > cap=1. This is the load-bearing
// "tracker survives across Run boundaries within a single ctx" property the
// Phase 37 Supervisor will lean on.
func TestSimple_BudgetExhaustion(t *testing.T) {
	ctx, tracker := budget.WithBudget(context.Background(), budget.Budget{MaxCalls: 1})
	llmMock := newScriptedLLM(
		textResp("first"),
		textResp("second"), // never reached — denied at pre-call charge
		textResp("third"),
	)
	a := NewSimpleAgent(llmMock, SimpleOptions{})

	// First Run: succeeds. Charges 1 against MaxCalls=1.
	res1, err1 := a.Run(ctx, "input-1")
	if err1 != nil {
		t.Fatalf("first Run: %v", err1)
	}
	if res1.Answer != "first" {
		t.Errorf("first Answer = %q, want %q", res1.Answer, "first")
	}

	// Second Run: pre-call charge denies (wantCalls=2 > 1).
	res2, err2 := a.Run(ctx, "input-2")
	if !errors.Is(err2, budget.ErrCallsExceeded) {
		t.Fatalf("second Run: err = %v, want ErrCallsExceeded", err2)
	}
	if !errors.Is(err2, budget.ErrBudgetExceeded) {
		t.Fatalf("second Run: err = %v, want ErrBudgetExceeded (umbrella)", err2)
	}
	if !reflect.DeepEqual(res2, Result{}) {
		t.Fatalf("second Run: expected zero Result on chokepoint error, got %+v", res2)
	}

	// Tracker reflects only the successful first Run; the denied attempt did
	// not mutate state (check-before-commit).
	snap := tracker.Snapshot()
	if snap.Calls != 1 {
		t.Errorf("tracker Snapshot().Calls = %d, want 1", snap.Calls)
	}
	// Confirm we can also retrieve the tracker via budget.From(ctx).
	tr, ok := budget.From(ctx)
	if !ok {
		t.Fatalf("budget.From(ctx) returned ok=false")
	}
	if got := tr.Snapshot().Calls; got != 1 {
		t.Errorf("budget.From(ctx).Snapshot().Calls = %d, want 1", got)
	}
}

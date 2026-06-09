package agents

import (
	"context"
	"errors"
	"testing"
)

type callbackStubAgent struct {
	name string
	err  error
}

func (a callbackStubAgent) Name() string {
	if a.name == "" {
		return "stub"
	}
	return a.name
}

func (a callbackStubAgent) Run(ctx context.Context, input string) (Result, error) {
	if a.err != nil {
		return Result{}, a.err
	}
	return Result{
		Answer: "answer:" + input,
		Trace:  []Step{{Kind: StepThought, Content: "think"}, {Kind: StepFinal, Content: "answer:" + input}},
		Usage:  Usage{LLMCalls: 1, Tokens: 7},
	}, nil
}

func (a callbackStubAgent) RunStream(ctx context.Context, input string) (<-chan StepEvent, error) {
	if a.err != nil {
		return nil, a.err
	}
	ch := make(chan StepEvent, 3)
	ch <- StepEvent{Step: Step{Kind: StepThought, Content: "think"}}
	ch <- StepEvent{Step: Step{Kind: StepFinal, Content: "answer:" + input}}
	res := Result{Answer: "answer:" + input, Usage: Usage{LLMCalls: 1, Tokens: 7}}
	ch <- StepEvent{Done: true, Final: &res}
	close(ch)
	return ch, nil
}

func TestWrapAgentRunStreamObservesAndPreservesEvents(t *testing.T) {
	var observed []RunEvent
	a := WrapAgent(callbackStubAgent{name: "wrapped"}, CallbackFunc(func(ctx context.Context, ev RunEvent) {
		observed = append(observed, ev)
	}))
	ch, err := a.RunStream(context.Background(), "x")
	if err != nil {
		t.Fatalf("RunStream: %v", err)
	}
	var got []StepEvent
	for ev := range ch {
		got = append(got, ev)
	}
	if len(got) != 3 {
		t.Fatalf("got %d step events, want 3", len(got))
	}
	if len(observed) != 3 {
		t.Fatalf("observed %d events, want 3", len(observed))
	}
	if observed[0].Kind != RunEventAgentStep || observed[0].Step.Kind != StepThought {
		t.Fatalf("observed[0] = %#v, want thought step", observed[0])
	}
	if observed[2].Kind != RunEventAgentDone || observed[2].Final.Answer != "answer:x" {
		t.Fatalf("observed[2] = %#v, want done answer:x", observed[2])
	}
}

func TestWrapAgentRunObservesDone(t *testing.T) {
	var kinds []RunEventKind
	a := WrapAgent(callbackStubAgent{name: "wrapped"}, CallbackFunc(func(ctx context.Context, ev RunEvent) {
		kinds = append(kinds, ev.Kind)
	}))
	res, err := a.Run(context.Background(), "x")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Answer != "answer:x" {
		t.Fatalf("answer = %q, want answer:x", res.Answer)
	}
	if len(kinds) != 3 || kinds[0] != RunEventAgentStep || kinds[2] != RunEventAgentDone {
		t.Fatalf("kinds = %v, want step/step/done", kinds)
	}
}

func TestWrapAgentCallbackPanicDoesNotAffectRunStream(t *testing.T) {
	a := WrapAgent(callbackStubAgent{}, CallbackFunc(func(ctx context.Context, ev RunEvent) {
		panic("callback failed")
	}))
	ch, err := a.RunStream(context.Background(), "x")
	if err != nil {
		t.Fatalf("RunStream: %v", err)
	}
	count := 0
	for range ch {
		count++
	}
	if count != 3 {
		t.Fatalf("got %d step events, want 3", count)
	}
}

func TestWrapAgentRunStreamStartErrorObserved(t *testing.T) {
	want := errors.New("start failed")
	var observed []RunEvent
	a := WrapAgent(callbackStubAgent{err: want}, CallbackFunc(func(ctx context.Context, ev RunEvent) {
		observed = append(observed, ev)
	}))
	_, err := a.RunStream(context.Background(), "x")
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
	if len(observed) != 1 || observed[0].Kind != RunEventAgentError || !errors.Is(observed[0].Err, want) {
		t.Fatalf("observed = %#v, want one agent_error", observed)
	}
}

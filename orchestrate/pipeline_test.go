package orchestrate

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/costa92/llm-agent"
)

// stubAgent is a minimal agents.Agent test double: returns a fixed
// answer prefixed with the input it received (so tests can assert
// the input was threaded through correctly).
type stubAgent struct {
	name      string
	transform func(input string) string
	err       error
	llmCalls  int
}

func (a *stubAgent) Name() string { return a.name }

func (a *stubAgent) Run(_ context.Context, input string) (agents.Result, error) {
	if a.err != nil {
		return agents.Result{}, a.err
	}
	out := input
	if a.transform != nil {
		out = a.transform(input)
	}
	calls := a.llmCalls
	if calls == 0 {
		calls = 1
	}
	return agents.Result{
		Answer: out,
		Usage:  agents.Usage{LLMCalls: calls, Tokens: len(out)},
	}, nil
}

func (a *stubAgent) RunStream(_ context.Context, _ string) (<-chan agents.StepEvent, error) {
	return nil, errors.New("stubAgent: stream not implemented")
}

func TestPipeline_RunsAllStepsInOrder(t *testing.T) {
	a := &stubAgent{name: "A", transform: func(s string) string { return s + "→A" }}
	b := &stubAgent{name: "B", transform: func(s string) string { return s + "→B" }}
	c := &stubAgent{name: "C", transform: func(s string) string { return s + "→C" }}

	p := NewPipeline("test", Step{Name: "first", Agent: a}, Step{Name: "second", Agent: b}, Step{Name: "third", Agent: c})
	res, err := p.Run(context.Background(), "input")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.FinalAnswer != "input→A→B→C" {
		t.Errorf("FinalAnswer = %q, want input→A→B→C", res.FinalAnswer)
	}
	if len(res.StepResults) != 3 {
		t.Fatalf("got %d step results, want 3", len(res.StepResults))
	}
	if res.StepResults[0].Step != "first" || res.StepResults[2].Step != "third" {
		t.Errorf("step order wrong: %v", res.StepResults)
	}
}

func TestPipeline_AdaptOverridesDefaultThreading(t *testing.T) {
	// Without Adapt, B sees A's answer directly. With Adapt, B sees a
	// transformed version.
	a := &stubAgent{name: "A", transform: func(s string) string { return "A" + s }}
	b := &stubAgent{name: "B", transform: func(s string) string { return "got:" + s }}

	p := NewPipeline("adapt-test",
		Step{Name: "a", Agent: a},
		Step{Name: "b", Agent: b, Adapt: func(prev agents.Result) string { return "wrapped(" + prev.Answer + ")" }},
	)
	res, _ := p.Run(context.Background(), "x")
	if res.FinalAnswer != "got:wrapped(Ax)" {
		t.Errorf("FinalAnswer = %q, want got:wrapped(Ax)", res.FinalAnswer)
	}
}

func TestPipeline_AccumulatesUsage(t *testing.T) {
	a := &stubAgent{name: "A", transform: func(s string) string { return s }, llmCalls: 2}
	b := &stubAgent{name: "B", transform: func(s string) string { return s }, llmCalls: 3}

	p := NewPipeline("usage", Step{Name: "a", Agent: a}, Step{Name: "b", Agent: b})
	res, _ := p.Run(context.Background(), "hi")
	if res.TotalUsage.LLMCalls != 5 {
		t.Errorf("LLMCalls = %d, want 5", res.TotalUsage.LLMCalls)
	}
	if res.TotalUsage.Tokens == 0 {
		t.Error("Tokens not accumulated")
	}
}

func TestPipeline_StepErrorAborts(t *testing.T) {
	a := &stubAgent{name: "A", transform: func(s string) string { return s }}
	bad := &stubAgent{name: "bad", err: errors.New("boom")}
	c := &stubAgent{name: "C", transform: func(s string) string { return s + "→C" }}

	p := NewPipeline("err", Step{Name: "a", Agent: a}, Step{Name: "bad", Agent: bad}, Step{Name: "c", Agent: c})
	_, err := p.Run(context.Background(), "x")
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected wrapped boom error, got %v", err)
	}
	// C must NOT have run (counter would have been > 0)
}

func TestPipeline_EmptyStepsErrors(t *testing.T) {
	p := NewPipeline("empty")
	_, err := p.Run(context.Background(), "x")
	if !errors.Is(err, ErrEmptyPipeline) {
		t.Errorf("expected ErrEmptyPipeline, got %v", err)
	}
}

func TestPipeline_NilAgentErrors(t *testing.T) {
	p := NewPipeline("nil-agent", Step{Name: "broken", Agent: nil})
	_, err := p.Run(context.Background(), "x")
	if err == nil || !strings.Contains(err.Error(), "nil Agent") {
		t.Fatalf("expected nil-agent error, got %v", err)
	}
}

func TestPipeline_NameDefault(t *testing.T) {
	if NewPipeline("").Name() != "pipeline" {
		t.Errorf("empty Name should default to %q", "pipeline")
	}
	if NewPipeline("custom").Name() != "custom" {
		t.Errorf("custom Name should pass through")
	}
}

// Ensure stubAgent satisfies agents.Agent at compile time.
var _ agents.Agent = (*stubAgent)(nil)

// Demo helper kept here so other tests in this package can build chained
// scenarios without each redefining stubAgent.
func _exampleStubAgent() agents.Agent {
	return &stubAgent{name: "demo", transform: func(s string) string { return fmt.Sprintf("demo(%s)", s) }}
}

package agents

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/costa92/llm-agent/budget"
)

func TestReflectionAgent_RevisesAcrossRounds(t *testing.T) {
	llmMock := newScriptedLLM(
		// Round 0: initial gen
		textResp("first draft"),
		// Round 1: critique
		textResp("CRITIQUE: needs more detail"),
		// Round 1: revise
		textResp("second draft with detail"),
		// Round 2: critique
		textResp("APPROVED"),
	)
	a := NewReflectionAgent(llmMock, ReflectionOptions{MaxRounds: 2})
	res, err := a.Run(context.Background(), "write a tagline")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Answer != "second draft with detail" {
		t.Errorf("Answer = %q", res.Answer)
	}
	// Stop early on APPROVED → 4 LLM calls (gen + critique + revise + critique)
	if res.Usage.LLMCalls != 4 {
		t.Errorf("LLMCalls = %d, want 4", res.Usage.LLMCalls)
	}
	hasReflection := false
	for _, s := range res.Trace {
		if s.Kind == StepReflection {
			hasReflection = true
		}
	}
	if !hasReflection {
		t.Error("Trace should contain a StepReflection")
	}
}

func TestReflectionAgent_EmptyInput(t *testing.T) {
	a := NewReflectionAgent(newScriptedLLM(), ReflectionOptions{})
	_, err := a.Run(context.Background(), "")
	if !errors.Is(err, ErrEmptyInput) {
		t.Errorf("err = %v", err)
	}
}

// TestReflection_BudgetExhaustion proves Reflection's gen→critique→revise
// cycle honors a MaxCalls budget at the chokepoint (35-04 / CC-1).
//
// Reflection makes 3 calls in a full round (initial draft, critique, revise).
// With Budget{MaxCalls: 2} the third pre-call charge — the revise — is denied,
// so the agent returns zero Result + ErrCallsExceeded. We deliberately script
// a non-APPROVED critique so the loop does not break before revise.
func TestReflection_BudgetExhaustion(t *testing.T) {
	ctx, _ := budget.WithBudget(context.Background(), budget.Budget{MaxCalls: 2})

	llmMock := newScriptedLLM(
		textResp("initial draft"),                  // call 1 — gen
		textResp("CRITIQUE: needs more punch"),     // call 2 — critique (not APPROVED)
		textResp("revised draft (would be third)"), // call 3 — DENIED at pre-call charge
	)
	a := NewReflectionAgent(llmMock, ReflectionOptions{MaxRounds: 1})

	result, err := a.Run(ctx, "task")
	if !errors.Is(err, budget.ErrCallsExceeded) {
		t.Fatalf("err = %v, want ErrCallsExceeded", err)
	}
	if !errors.Is(err, budget.ErrBudgetExceeded) {
		t.Fatalf("err = %v, want ErrBudgetExceeded (umbrella)", err)
	}
	if !reflect.DeepEqual(result, Result{}) {
		t.Fatalf("expected zero Result on chokepoint error (reflection.go:81/94/110), got %+v", result)
	}

	tr, ok := budget.From(ctx)
	if !ok {
		t.Fatalf("budget.From(ctx) returned ok=false")
	}
	if got := tr.Snapshot().Calls; got != 2 {
		t.Errorf("tracker Snapshot().Calls = %d, want 2 (cap; denied revise did not mutate)", got)
	}
}

func TestReflectionAgent_OnStep_Invoked(t *testing.T) {
	llmMock := newScriptedLLM(
		textResp("draft"),
		textResp("APPROVED"),
	)
	var count int
	a := NewReflectionAgent(llmMock, ReflectionOptions{
		MaxRounds: 1,
		OnStep:    func(Step) { count++ },
	})
	_, err := a.Run(context.Background(), "task")
	if err != nil {
		t.Fatal(err)
	}
	// initial draft + critique (APPROVED, no revise) + final = 3 callbacks
	if count != 3 {
		t.Errorf("OnStep called %d times, want 3", count)
	}
}

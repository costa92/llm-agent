package agents

import (
	"context"
	"errors"
	"testing"
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

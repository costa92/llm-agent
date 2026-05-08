package agents

import (
	"context"
	"errors"
	"testing"
)

func TestPlanAndSolveAgent_PlanThenExec(t *testing.T) {
	llmMock := newScriptedLLM(
		// 1: plan
		textResp("PLAN:\n1. greet\n2. answer\n3. wrap up"),
		// 2-4: per-step exec
		textResp("Hello"),
		textResp("42"),
		textResp("Bye"),
		// 5: synthesize final
		textResp("Hello — 42 — Bye"),
	)
	a := NewPlanAndSolveAgent(llmMock, PlanAndSolveOptions{MaxSteps: 5})
	res, err := a.Run(context.Background(), "answer the question")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Answer != "Hello — 42 — Bye" {
		t.Errorf("Answer = %q", res.Answer)
	}
	if res.Usage.LLMCalls != 5 {
		t.Errorf("LLMCalls = %d, want 5 (1 plan + 3 steps + 1 synth)", res.Usage.LLMCalls)
	}
	planCount, thoughtCount, finalCount := 0, 0, 0
	for _, s := range res.Trace {
		switch s.Kind {
		case StepPlan:
			planCount++
		case StepThought:
			thoughtCount++
		case StepFinal:
			finalCount++
		}
	}
	if planCount != 1 || thoughtCount != 3 || finalCount != 1 {
		t.Errorf("kinds = plan:%d thought:%d final:%d", planCount, thoughtCount, finalCount)
	}
}

func TestPlanAndSolveAgent_PlanParseFail(t *testing.T) {
	llmMock := newScriptedLLM(textResp("not a plan"))
	a := NewPlanAndSolveAgent(llmMock, PlanAndSolveOptions{MaxSteps: 5})
	_, err := a.Run(context.Background(), "go")
	if !errors.Is(err, ErrPlanningFailed) {
		t.Errorf("err = %v, want ErrPlanningFailed", err)
	}
}

func TestPlanAndSolveAgent_EmptyInput(t *testing.T) {
	a := NewPlanAndSolveAgent(newScriptedLLM(), PlanAndSolveOptions{})
	_, err := a.Run(context.Background(), "")
	if !errors.Is(err, ErrEmptyInput) {
		t.Errorf("err = %v", err)
	}
}

// TestPlanAndSolveAgent_TruncatesAtMaxSteps verifies the steps[:MaxSteps]
// truncation actually fires (feature exists in code but was untested).
func TestPlanAndSolveAgent_TruncatesAtMaxSteps(t *testing.T) {
	// Plan with 5 steps, MaxSteps=3 → only 3 steps run + 1 plan + 1 synth = 5 LLM calls
	llmMock := newScriptedLLM(
		textResp("PLAN:\n1. a\n2. b\n3. c\n4. d\n5. e"),
		textResp("ra"), textResp("rb"), textResp("rc"),
		textResp("synthesized"),
	)
	a := NewPlanAndSolveAgent(llmMock, PlanAndSolveOptions{MaxSteps: 3})
	res, err := a.Run(context.Background(), "task")
	if err != nil {
		t.Fatal(err)
	}
	if res.Usage.LLMCalls != 5 {
		t.Errorf("LLMCalls = %d, want 5 (truncated to MaxSteps=3)", res.Usage.LLMCalls)
	}
}

func TestPlanAndSolveAgent_OnStep_Invoked(t *testing.T) {
	llmMock := newScriptedLLM(
		textResp("PLAN:\n1. a\n2. b"),
		textResp("ra"),
		textResp("rb"),
		textResp("synth"),
	)
	var kinds []StepKind
	a := NewPlanAndSolveAgent(llmMock, PlanAndSolveOptions{
		MaxSteps: 5,
		OnStep:   func(s Step) { kinds = append(kinds, s.Kind) },
	})
	_, err := a.Run(context.Background(), "task")
	if err != nil {
		t.Fatal(err)
	}
	// plan + 2 thoughts + final
	want := []StepKind{StepPlan, StepThought, StepThought, StepFinal}
	if !equalSlice(kinds, want) {
		t.Errorf("OnStep kinds = %v, want %v", kinds, want)
	}
}

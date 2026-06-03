package agents

// budget_integration_test.go — cross-paradigm uniformity test for the budget
// chokepoint (35-04 / CC-1). Asserts every agent paradigm (Simple, ReAct,
// Reflection, Plan-and-Solve, FunctionCall) propagates an identical sentinel
// shape when MaxCalls exhausts at the chokepoint: errors.Is matches the
// dim-specific ErrCallsExceeded AND the umbrella ErrBudgetExceeded, the
// returned agents.Result is the zero value, and the tracker's Snapshot().Calls
// equals the cap (denied attempt did NOT mutate state — check-before-commit).
//
// SimpleAgent and FunctionCallAgent are single-call-per-Run paradigms — the
// only way to exercise the cap against them is the cross-Run pattern (the
// tracker survives across `agent.Run` calls within a single ctx). That
// cross-Run property is load-bearing for Phase 37 Supervisor, so both rows
// are kept in the table; they take a `crossRun: true` branch.

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/costa92/llm-agent/budget"
	"github.com/costa92/llm-agent-contract/llm"
)

// paradigmCase couples a paradigm's name with a factory that builds the agent
// + a deterministic ScriptedLLM (and any required Registry) and the number of
// LLM calls the paradigm naturally consumes before MaxCalls cuts the run.
type paradigmCase struct {
	name     string
	build    func() Agent
	needs    int  // calls used by the paradigm before deny — also cap = needs-1
	crossRun bool // when true, run twice in the same ctx (single-call paradigms)
}

// buildSimple constructs a SimpleAgent scripted for 3 turns (cross-Run pattern).
func buildSimple() Agent {
	mock := newScriptedLLM(textResp("one"), textResp("two"), textResp("three"))
	return NewSimpleAgent(mock, SimpleOptions{})
}

// buildReAct scripts 4 action turns so the loop attempts >2 LLM calls.
func buildReAct() Agent {
	tool := &recordingTool{name: "echo", out: "ok"}
	reg := NewRegistry(tool)
	mock := newScriptedLLM(
		textResp("Thought: t1\nAction: echo\nArgs: {}"),
		textResp("Thought: t2\nAction: echo\nArgs: {}"),
		textResp("Thought: t3\nAction: echo\nArgs: {}"),
		textResp("Thought: t4\nFinal: done"),
	)
	return NewReActAgent(mock, ReActOptions{Registry: reg, MaxSteps: 5})
}

// buildReflection scripts 3 turns (draft / non-APPROVED critique / would-be-revise).
func buildReflection() Agent {
	mock := newScriptedLLM(
		textResp("draft"),
		textResp("CRITIQUE: thin"),
		textResp("revised"),
	)
	return NewReflectionAgent(mock, ReflectionOptions{MaxRounds: 1})
}

// buildPlanSolve scripts plan (2 steps) + step-1 + step-2-or-synth.
func buildPlanSolve() Agent {
	mock := newScriptedLLM(
		textResp("PLAN:\n1. a\n2. b"),
		textResp("ra"),
		textResp("rb"),
	)
	return NewPlanAndSolveAgent(mock, PlanAndSolveOptions{MaxSteps: 5})
}

// buildFunctionCall scripts two responses: a tool-call (consumed on the first
// Run) + an unreached text response (the second Run is denied at pre-charge).
func buildFunctionCall() Agent {
	tool := NewFuncTool("noop", "no-op", json.RawMessage(`{}`),
		func(_ context.Context, _ json.RawMessage) (string, error) { return "ok", nil })
	reg := NewRegistry(tool)
	mock := llm.NewScriptedLLM(
		llm.WithProvider("scripted"),
		llm.WithCapabilities(llm.Capabilities{Tools: true}),
		llm.WithResponses(
			llm.Response{
				Provider:  "scripted",
				ToolCalls: []llm.ToolCall{{Name: "noop", Arguments: json.RawMessage(`{}`)}},
			},
			llm.TextResponse("unreached"),
		),
	)
	a, err := NewFunctionCallAgent(mock, FunctionCallOptions{Registry: reg})
	if err != nil {
		// build*-helpers run inside the test; panic surfaces as a t.Fatal at the call site.
		panic("buildFunctionCall: " + err.Error())
	}
	return a
}

// TestAllParadigms_BudgetUniformity is the cross-paradigm uniformity test
// (35-04 / CC-1). It does NOT assert Result content shape — each paradigm has
// its own Answer/Trace conventions, and on this chokepoint-error path every
// paradigm returns agents.Result{}. It DOES assert:
//
//   - errors.Is(err, budget.ErrCallsExceeded)
//   - errors.Is(err, budget.ErrBudgetExceeded) — umbrella
//   - result == agents.Result{} (zero on chokepoint error)
//   - tracker.Snapshot().Calls == row.needs - 1 (cap; denied attempt did not mutate)
//
// SimpleAgent + FunctionCallAgent take the crossRun branch (single-call
// paradigms — proves cross-Run enforcement, the Phase 37 Supervisor property).
func TestAllParadigms_BudgetUniformity(t *testing.T) {
	paradigms := []paradigmCase{
		{name: "simple", build: buildSimple, needs: 2, crossRun: true},
		{name: "react", build: buildReAct, needs: 3, crossRun: false},
		{name: "reflection", build: buildReflection, needs: 3, crossRun: false},
		{name: "plansolve", build: buildPlanSolve, needs: 3, crossRun: false},
		{name: "functioncall", build: buildFunctionCall, needs: 2, crossRun: true},
	}

	for _, p := range paradigms {
		t.Run(p.name, func(t *testing.T) {
			cap := p.needs - 1
			ctx, tracker := budget.WithBudget(context.Background(), budget.Budget{MaxCalls: cap})
			agent := p.build()

			var result Result
			var err error
			if p.crossRun {
				// Run twice — first succeeds, second is denied at pre-call charge.
				if _, err1 := agent.Run(ctx, "first"); err1 != nil {
					t.Fatalf("[%s] first Run failed unexpectedly: %v", p.name, err1)
				}
				result, err = agent.Run(ctx, "second")
			} else {
				// Single Run — exhausts internally during the paradigm's loop.
				result, err = agent.Run(ctx, "go")
			}

			if !errors.Is(err, budget.ErrCallsExceeded) {
				t.Fatalf("[%s] err = %v, want ErrCallsExceeded", p.name, err)
			}
			if !errors.Is(err, budget.ErrBudgetExceeded) {
				t.Fatalf("[%s] err = %v, want ErrBudgetExceeded (umbrella)", p.name, err)
			}
			if !reflect.DeepEqual(result, Result{}) {
				t.Fatalf("[%s] expected zero Result on chokepoint error, got %+v", p.name, result)
			}
			if got := tracker.Snapshot().Calls; got != cap {
				t.Errorf("[%s] tracker Snapshot().Calls = %d, want %d (cap; denied attempt must not mutate)", p.name, got, cap)
			}
		})
	}
}

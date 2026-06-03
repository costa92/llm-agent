package agents

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/costa92/llm-agent/budget"
	"github.com/costa92/llm-agent-contract/llm"
)

func TestFunctionCallAgent_NoToolCalls_ReturnsText(t *testing.T) {
	llmMock := llm.NewScriptedLLM(
		llm.WithProvider("scripted"),
		llm.WithCapabilities(llm.Capabilities{Tools: true}),
		llm.WithResponses(llm.TextResponse("direct answer")),
	)
	a, err := NewFunctionCallAgent(llmMock, FunctionCallOptions{Registry: NewRegistry()})
	if err != nil {
		t.Fatal(err)
	}
	res, err := a.Run(context.Background(), "hi")
	if err != nil {
		t.Fatal(err)
	}
	if res.Answer != "direct answer" {
		t.Errorf("Answer = %q", res.Answer)
	}
}

func TestFunctionCallAgent_RunsToolCallsInParallel(t *testing.T) {
	tool1 := &recordingTool{name: "calc", out: "42"}
	tool2 := &recordingTool{name: "search", out: "blue sky"}
	reg := NewRegistry(tool1, tool2)

	llmMock := llm.NewScriptedLLM(
		llm.WithProvider("scripted"),
		llm.WithCapabilities(llm.Capabilities{Tools: true}),
		llm.WithResponses(llm.Response{
			Provider: "scripted",
			ToolCalls: []llm.ToolCall{
				{Name: "calc", Arguments: json.RawMessage(`{"expr":"6*7"}`)},
				{Name: "search", Arguments: json.RawMessage(`{"q":"sky"}`)},
			},
		}),
	)
	a, err := NewFunctionCallAgent(llmMock, FunctionCallOptions{Registry: reg})
	if err != nil {
		t.Fatal(err)
	}

	res, err := a.Run(context.Background(), "ask both")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Answer, "42") || !strings.Contains(res.Answer, "blue sky") {
		t.Errorf("Answer should contain both tool outputs: %q", res.Answer)
	}
	actionCount := 0
	for _, s := range res.Trace {
		if s.Kind == StepAction {
			actionCount++
		}
	}
	if actionCount != 2 {
		t.Errorf("StepAction count = %d, want 2", actionCount)
	}
}

func TestFunctionCallAgent_UnknownTool(t *testing.T) {
	llmMock := llm.NewScriptedLLM(
		llm.WithProvider("scripted"),
		llm.WithCapabilities(llm.Capabilities{Tools: true}),
		llm.WithResponses(llm.Response{
			Provider:  "scripted",
			ToolCalls: []llm.ToolCall{{Name: "nope", Arguments: json.RawMessage(`{}`)}},
		}),
	)
	a, err := NewFunctionCallAgent(llmMock, FunctionCallOptions{Registry: NewRegistry()})
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.Run(context.Background(), "x")
	if !errors.Is(err, ErrToolNotFound) {
		t.Errorf("err = %v, want ErrToolNotFound", err)
	}
}

func TestFunctionCallAgent_EmptyInput(t *testing.T) {
	model := llm.NewScriptedLLM(
		llm.WithProvider("scripted"),
		llm.WithCapabilities(llm.Capabilities{Tools: true}),
	)
	a, err := NewFunctionCallAgent(model, FunctionCallOptions{Registry: NewRegistry()})
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.Run(context.Background(), "  ")
	if !errors.Is(err, ErrEmptyInput) {
		t.Errorf("err = %v", err)
	}
}

// TestFunctionCallAgent_PartialToolFailure_AbortsButOthersAlreadyRan documents
// the fail-fast semantic after AsyncRunner refactor: any tool error aborts the
// integration, but tools already ran in parallel before the failure was detected.
// Side effects from non-failing tools are NOT undone.
func TestFunctionCallAgent_PartialToolFailure_AbortsButOthersAlreadyRan(t *testing.T) {
	calledA := false
	toolA := NewFuncTool("a-ok", "ok tool", json.RawMessage(`{}`),
		func(_ context.Context, _ json.RawMessage) (string, error) {
			calledA = true
			return "ok", nil
		})
	toolB := NewFuncTool("b-fail", "fail tool", json.RawMessage(`{}`),
		func(_ context.Context, _ json.RawMessage) (string, error) {
			return "", errors.New("intentional fail")
		})
	reg := NewRegistry(toolA, toolB)

	llmMock := llm.NewScriptedLLM(
		llm.WithProvider("scripted"),
		llm.WithCapabilities(llm.Capabilities{Tools: true}),
		llm.WithResponses(llm.Response{
			Provider: "scripted",
			ToolCalls: []llm.ToolCall{
				{Name: "a-ok", Arguments: json.RawMessage(`{}`)},
				{Name: "b-fail", Arguments: json.RawMessage(`{}`)},
			},
		}),
	)
	a, err := NewFunctionCallAgent(llmMock, FunctionCallOptions{Registry: reg})
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.Run(context.Background(), "x")

	if err == nil || !strings.Contains(err.Error(), "intentional fail") {
		t.Errorf("err = %v, want 'intentional fail'", err)
	}
	if !calledA {
		t.Error("toolA should have been called (parallel execution before failure check)")
	}
}

// TestFunctionCall_BudgetExhaustion proves FunctionCallAgent honors a MaxCalls
// budget at the chokepoint (35-04 / CC-1).
//
// FunctionCallAgent is single-turn — exactly 1 LLM call per Run, with tools
// dispatched in parallel after the response. With Budget{MaxCalls: 1} the
// first Run succeeds (its single LLM call charges 1 against cap=1 and the
// scripted tool fires); a second Run in the same ctx is denied at its
// pre-call charge (wantCalls=2 > 1). This proves both the chokepoint
// enforces the cap AND the tracker survives across Run boundaries within a
// ctx — the cross-Run property Phase 37 Supervisor relies on.
//
// The closure-captured atomic counter proves the tool dispatched during the
// first Run's response handling, BEFORE the second Run's chokepoint deny.
func TestFunctionCall_BudgetExhaustion(t *testing.T) {
	ctx, _ := budget.WithBudget(context.Background(), budget.Budget{MaxCalls: 1})

	var toolHits atomic.Int32
	counter := NewFuncTool("count", "increments a counter", json.RawMessage(`{}`),
		func(_ context.Context, _ json.RawMessage) (string, error) {
			toolHits.Add(1)
			return "tick", nil
		})
	reg := NewRegistry(counter)

	llmMock := llm.NewScriptedLLM(
		llm.WithProvider("scripted"),
		llm.WithCapabilities(llm.Capabilities{Tools: true}),
		llm.WithResponses(
			// first Run: tool-binding call → schedules `count` once
			llm.Response{
				Provider:  "scripted",
				ToolCalls: []llm.ToolCall{{Name: "count", Arguments: json.RawMessage(`{}`)}},
			},
			// second Run: never reached — denied at pre-call charge
			llm.TextResponse("unreached"),
		),
	)
	a, err := NewFunctionCallAgent(llmMock, FunctionCallOptions{Registry: reg})
	if err != nil {
		t.Fatal(err)
	}

	// First Run: succeeds. 1 LLM call charged, tool runs.
	res1, err1 := a.Run(ctx, "go-1")
	if err1 != nil {
		t.Fatalf("first Run: %v", err1)
	}
	if !strings.Contains(res1.Answer, "tick") {
		t.Errorf("first Answer = %q, want contains %q", res1.Answer, "tick")
	}
	if got := toolHits.Load(); got != 1 {
		t.Fatalf("toolHits after first Run = %d, want 1 (tool ran during first Run)", got)
	}

	// Second Run: pre-call charge denies before any LLM call or tool run.
	res2, err2 := a.Run(ctx, "go-2")
	if !errors.Is(err2, budget.ErrCallsExceeded) {
		t.Fatalf("second Run: err = %v, want ErrCallsExceeded", err2)
	}
	if !errors.Is(err2, budget.ErrBudgetExceeded) {
		t.Fatalf("second Run: err = %v, want ErrBudgetExceeded (umbrella)", err2)
	}
	if !reflect.DeepEqual(res2, Result{}) {
		t.Fatalf("second Run: expected zero Result on chokepoint error, got %+v", res2)
	}
	if got := toolHits.Load(); got != 1 {
		t.Errorf("toolHits after second Run = %d, want 1 (tool must NOT have fired again)", got)
	}

	tr, ok := budget.From(ctx)
	if !ok {
		t.Fatalf("budget.From(ctx) returned ok=false")
	}
	if got := tr.Snapshot().Calls; got != 1 {
		t.Errorf("tracker Snapshot().Calls = %d, want 1 (cap; denied 2nd attempt did not mutate)", got)
	}
}

func TestFunctionCallAgent_FailsFastWithoutToolCapability(t *testing.T) {
	model := &llm.ChatOnlyMock{
		Provider: "test",
		Model:    "chat-only",
		Resp:     llm.Response{Text: "unused"},
	}
	_, err := NewFunctionCallAgent(model, FunctionCallOptions{Registry: NewRegistry()})
	if !errors.Is(err, llm.ErrCapabilityNotSupported) {
		t.Fatalf("err = %v, want ErrCapabilityNotSupported", err)
	}
	if !strings.Contains(err.Error(), "test/chat-only") {
		t.Fatalf("err = %v, want provider/model context", err)
	}
}

package agents

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/costa92/llm-agent/llm"
)

func TestFunctionCallAgent_NoToolCalls_ReturnsText(t *testing.T) {
	llmMock := newScriptedLLM(llm.GenerateResponse{
		Text:         "direct answer",
		FinishReason: llm.FinishReasonStop,
		Provider:     "scripted",
	})
	a := NewFunctionCallAgent(llmMock, FunctionCallOptions{Registry: NewRegistry()})
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

	llmMock := newScriptedLLM(llm.GenerateResponse{
		Provider: "scripted",
		ToolCalls: []llm.ToolCall{
			{Name: "calc", Arguments: json.RawMessage(`{"expr":"6*7"}`)},
			{Name: "search", Arguments: json.RawMessage(`{"q":"sky"}`)},
		},
	})
	a := NewFunctionCallAgent(llmMock, FunctionCallOptions{Registry: reg})

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
	llmMock := newScriptedLLM(llm.GenerateResponse{
		Provider:  "scripted",
		ToolCalls: []llm.ToolCall{{Name: "nope", Arguments: json.RawMessage(`{}`)}},
	})
	a := NewFunctionCallAgent(llmMock, FunctionCallOptions{Registry: NewRegistry()})
	_, err := a.Run(context.Background(), "x")
	if !errors.Is(err, ErrToolNotFound) {
		t.Errorf("err = %v, want ErrToolNotFound", err)
	}
}

func TestFunctionCallAgent_EmptyInput(t *testing.T) {
	a := NewFunctionCallAgent(newScriptedLLM(), FunctionCallOptions{Registry: NewRegistry()})
	_, err := a.Run(context.Background(), "  ")
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

	llmMock := newScriptedLLM(llm.GenerateResponse{
		Provider: "scripted",
		ToolCalls: []llm.ToolCall{
			{Name: "a-ok", Arguments: json.RawMessage(`{}`)},
			{Name: "b-fail", Arguments: json.RawMessage(`{}`)},
		},
	})
	a := NewFunctionCallAgent(llmMock, FunctionCallOptions{Registry: reg})
	_, err := a.Run(context.Background(), "x")

	if err == nil || !strings.Contains(err.Error(), "intentional fail") {
		t.Errorf("err = %v, want 'intentional fail'", err)
	}
	if !calledA {
		t.Error("toolA should have been called (parallel execution before failure check)")
	}
}

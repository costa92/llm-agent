package patterns

import (
	"context"
	"encoding/json"
	"testing"

	agents "github.com/costa92/llm-agent"
	"github.com/costa92/llm-agent-contract/llm"
)

func TestCatalogStableOrder(t *testing.T) {
	got := Catalog()
	want := []ID{Simple, ReAct, FunctionCall, PlanAndSolve, Reflection, Workspace}
	if len(got) != len(want) {
		t.Fatalf("catalog len = %d, want %d", len(got), len(want))
	}
	for i, id := range want {
		if got[i].ID != id {
			t.Fatalf("catalog[%d].ID = %q, want %q", i, got[i].ID, id)
		}
	}
}

func TestBuildSimple(t *testing.T) {
	a, err := Build(Simple, llm.NewScriptedLLM(llm.WithResponses(llm.TextResponse("ok"))), Options{Name: "assistant"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if a.Name() != "assistant" {
		t.Fatalf("name = %q, want assistant", a.Name())
	}
	res, err := a.Run(context.Background(), "x")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Answer != "ok" {
		t.Fatalf("answer = %q, want ok", res.Answer)
	}
}

func TestBuildWrapsCallback(t *testing.T) {
	var kinds []agents.RunEventKind
	a, err := Build(Simple, llm.NewScriptedLLM(llm.WithResponses(llm.TextResponse("ok"))), Options{
		Callback: agents.CallbackFunc(func(ctx context.Context, ev agents.RunEvent) {
			kinds = append(kinds, ev.Kind)
		}),
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	_, err = a.Run(context.Background(), "x")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(kinds) != 2 || kinds[0] != agents.RunEventAgentStep || kinds[1] != agents.RunEventAgentDone {
		t.Fatalf("kinds = %v, want step/done", kinds)
	}
}

func TestBuildUnknownPattern(t *testing.T) {
	_, err := Build(ID("unknown"), llm.NewScriptedLLM(), Options{})
	if err == nil {
		t.Fatal("Build unknown pattern returned nil error")
	}
}

func TestBuildFunctionCallRequiresToolCaller(t *testing.T) {
	_, err := Build(FunctionCall, &llm.ChatOnlyMock{Provider: "test", Model: "chat-only"}, Options{Registry: agents.NewRegistry()})
	if err == nil {
		t.Fatal("Build FunctionCall with chat-only model returned nil error")
	}
}

func TestBuildFunctionCall(t *testing.T) {
	model := llm.NewScriptedLLM(llm.WithResponses(llm.TextResponse("direct")))
	a, err := Build(FunctionCall, model, Options{Registry: agents.NewRegistry()})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	res, err := a.Run(context.Background(), "x")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Answer != "direct" {
		t.Fatalf("answer = %q, want direct", res.Answer)
	}
}

func TestBuildPlanAndSolve(t *testing.T) {
	model := llm.NewScriptedLLM(llm.WithResponses(
		llm.TextResponse("PLAN:\n1. inspect"),
		llm.TextResponse("inspection done"),
		llm.TextResponse("final"),
	))
	a, err := Build(PlanAndSolve, model, Options{MaxSteps: 2})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	res, err := a.Run(context.Background(), "x")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Answer != "final" {
		t.Fatalf("answer = %q, want final", res.Answer)
	}
}

func TestBuildReflection(t *testing.T) {
	model := llm.NewScriptedLLM(llm.WithResponses(
		llm.TextResponse("draft"),
		llm.TextResponse("APPROVED"),
	))
	a, err := Build(Reflection, model, Options{MaxRounds: 2})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	res, err := a.Run(context.Background(), "x")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Answer != "draft" {
		t.Fatalf("answer = %q, want draft", res.Answer)
	}
}

func TestBuildReActWithRegistry(t *testing.T) {
	reg := agents.NewRegistry(agents.NewFuncTool(
		"echo",
		"echo input",
		json.RawMessage(`{"type":"object"}`),
		func(ctx context.Context, args json.RawMessage) (string, error) { return "tool-output", nil },
	))
	model := &llm.ChatOnlyMock{Provider: "test", Model: "chat-only", Resp: llm.TextResponse("Thought: done\nFinal: answer")}
	a, err := Build(ReAct, model, Options{Registry: reg, MaxSteps: 2})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	res, err := a.Run(context.Background(), "x")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Answer != "answer" {
		t.Fatalf("answer = %q, want answer", res.Answer)
	}
}

func TestBuildWorkspaceRequiresRegistry(t *testing.T) {
	_, err := Build(Workspace, llm.NewScriptedLLM(), Options{})
	if err == nil {
		t.Fatal("Build Workspace without registry returned nil error")
	}
}

func TestBuildWorkspaceUsesCallerRegistry(t *testing.T) {
	reg := agents.NewRegistry(agents.NewFuncTool(
		"note",
		"record a note",
		json.RawMessage(`{"type":"object"}`),
		func(ctx context.Context, args json.RawMessage) (string, error) { return "noted", nil },
	))
	model := &llm.ChatOnlyMock{Provider: "test", Model: "chat-only", Resp: llm.TextResponse("Thought: done\nFinal: answer")}
	a, err := Build(Workspace, model, Options{Registry: reg, MaxSteps: 2})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if a.Name() != "workspace" {
		t.Fatalf("name = %q, want workspace", a.Name())
	}
	res, err := a.Run(context.Background(), "x")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Answer != "answer" {
		t.Fatalf("answer = %q, want answer", res.Answer)
	}
}

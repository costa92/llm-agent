package agents

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/costa92/llm-agent/budget"
	"github.com/costa92/llm-agent/llm"
)

// recordingTool records its invocations so tests can assert.
type recordingTool struct {
	name   string
	called []string // raw args strings, in order
	out    string
}

func (r *recordingTool) Name() string            { return r.name }
func (r *recordingTool) Description() string     { return "test tool " + r.name }
func (r *recordingTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (r *recordingTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	r.called = append(r.called, string(args))
	return r.out, nil
}

func TestReActAgent_HappyPath_FinalAfterOneAction(t *testing.T) {
	tool := &recordingTool{name: "echo", out: "tool said hello"}
	reg := NewRegistry(tool)

	llmMock := newScriptedLLM(
		// Round 1: think + act
		textResp("Thought: I should call echo.\nAction: echo\nArgs: {\"x\":1}"),
		// Round 2: observation feedback → final
		textResp("Thought: Done.\nFinal: all set"),
	)

	a := NewReActAgent(llmMock, ReActOptions{Registry: reg, MaxSteps: 5})
	res, err := a.Run(context.Background(), "go")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Answer != "all set" {
		t.Errorf("Answer = %q", res.Answer)
	}
	if len(tool.called) != 1 || tool.called[0] != `{"x":1}` {
		t.Errorf("tool.called = %v", tool.called)
	}
	if res.Usage.LLMCalls != 2 {
		t.Errorf("LLMCalls = %d", res.Usage.LLMCalls)
	}
	// Trace should contain: thought, action, observation, thought, final
	kinds := traceKinds(res.Trace)
	want := []StepKind{StepThought, StepAction, StepObservation, StepThought, StepFinal}
	if !equalSlice(kinds, want) {
		t.Errorf("Trace kinds = %v, want %v", kinds, want)
	}
}

func TestReActAgent_NativeToolPath_UsesStructuredToolCalls(t *testing.T) {
	tool := &recordingTool{name: "echo", out: "tool said hello"}
	reg := NewRegistry(tool)
	model := llm.NewScriptedLLM(
		llm.WithProvider("scripted"),
		llm.WithModel("tools"),
		llm.WithCapabilities(llm.Capabilities{Tools: true}),
		llm.WithResponses(llm.ToolCallResponse("echo", `{"x":1}`)),
	)

	a := NewReActAgent(model, ReActOptions{Registry: reg, MaxSteps: 5})
	res, err := a.Run(context.Background(), "go")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Answer != "tool said hello" {
		t.Fatalf("Answer = %q", res.Answer)
	}
	if len(tool.called) != 1 || tool.called[0] != `{"x":1}` {
		t.Fatalf("tool.called = %v", tool.called)
	}
	want := []StepKind{StepAction, StepObservation, StepFinal}
	if got := traceKinds(res.Trace); !equalSlice(got, want) {
		t.Fatalf("Trace kinds = %v, want %v", got, want)
	}
	if res.Usage.LLMCalls != 1 {
		t.Fatalf("LLMCalls = %d, want 1", res.Usage.LLMCalls)
	}
}

func TestReActAgent_FallsBackWhenToolCapabilityUnavailable(t *testing.T) {
	tool := &recordingTool{name: "echo", out: "tool said hello"}
	reg := NewRegistry(tool)
	model := &llm.ChatOnlyMock{
		Provider: "test",
		Model:    "chat-only",
		Resp: llm.Response{
			Text:         "Thought: Done.\nFinal: fallback path",
			FinishReason: llm.FinishReasonStop,
			Provider:     "test",
		},
	}

	a := NewReActAgent(model, ReActOptions{Registry: reg, MaxSteps: 5})
	res, err := a.Run(context.Background(), "go")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Answer != "fallback path" {
		t.Fatalf("Answer = %q", res.Answer)
	}
	if len(tool.called) != 0 {
		t.Fatalf("tool.called = %v, want no native tool execution", tool.called)
	}
	want := []StepKind{StepThought, StepFinal}
	if got := traceKinds(res.Trace); !equalSlice(got, want) {
		t.Fatalf("Trace kinds = %v, want %v", got, want)
	}
}

func TestReActAgent_MaxStepsExceeded(t *testing.T) {
	tool := &recordingTool{name: "loop"}
	reg := NewRegistry(tool)
	// Always return Action — never Final → expect ErrMaxStepsExceeded
	resps := make([]llm.Response, 10)
	for i := range resps {
		resps[i] = textResp("Thought: keep going\nAction: loop\nArgs: {}")
	}
	llmMock := newScriptedLLM(resps...)

	a := NewReActAgent(llmMock, ReActOptions{Registry: reg, MaxSteps: 3})
	_, err := a.Run(context.Background(), "go")
	if !errors.Is(err, ErrMaxStepsExceeded) {
		t.Errorf("err = %v, want ErrMaxStepsExceeded", err)
	}
}

func TestReActAgent_UnknownTool(t *testing.T) {
	llmMock := newScriptedLLM(textResp("Thought: x\nAction: nope\nArgs: {}"))
	a := NewReActAgent(llmMock, ReActOptions{Registry: NewRegistry(), MaxSteps: 3})
	_, err := a.Run(context.Background(), "go")
	if !errors.Is(err, ErrToolNotFound) {
		t.Errorf("err = %v, want ErrToolNotFound", err)
	}
}

func TestReActAgent_EmptyInput(t *testing.T) {
	a := NewReActAgent(newScriptedLLM(), ReActOptions{})
	_, err := a.Run(context.Background(), "  ")
	if !errors.Is(err, ErrEmptyInput) {
		t.Errorf("err = %v", err)
	}
}

// TestParseReAct_RobustToVariations documents the parser's tolerance for LLM
// format variations. Real LLMs occasionally emit "**Thought:**" (bold) or
// "1. Thought:" (numbered) — current parser does NOT support these.
// If parser is improved later, flip wantOK to true.
func TestParseReAct_RobustToVariations(t *testing.T) {
	cases := []struct {
		name   string
		in     string
		wantOK bool
	}{
		{"plain", "Thought: a\nAction: t\nArgs: {}", true},
		{"final-only", "Thought: x\nFinal: done", true},
		{"bold-markdown", "**Thought:** a\n**Action:** t\n**Args:** {}", false},
		{"numbered", "1. Thought: a\n2. Action: t\n3. Args: {}", false},
		{"trailing-cr", "Thought: a\r\nAction: t\r\nArgs: {}\r\n", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			thought, action, args, final, err := parseReAct(tc.in)
			gotOK := err == nil && (final != "" || (action != "" && args != ""))
			_ = thought
			if gotOK != tc.wantOK {
				t.Errorf("parseReAct(%q): gotOK=%v wantOK=%v (action=%q args=%q final=%q err=%v)",
					tc.in, gotOK, tc.wantOK, action, args, final, err)
			}
		})
	}
}

func TestReActAgent_OnStep_Invoked(t *testing.T) {
	tool := &recordingTool{name: "echo", out: "ok"}
	reg := NewRegistry(tool)
	llmMock := newScriptedLLM(
		textResp("Thought: t1\nAction: echo\nArgs: {}"),
		textResp("Thought: t2\nFinal: done"),
	)
	var got []StepKind
	a := NewReActAgent(llmMock, ReActOptions{
		Registry: reg,
		MaxSteps: 5,
		OnStep:   func(s Step) { got = append(got, s.Kind) },
	})
	_, err := a.Run(context.Background(), "go")
	if err != nil {
		t.Fatal(err)
	}
	want := []StepKind{StepThought, StepAction, StepObservation, StepThought, StepFinal}
	if !equalSlice(got, want) {
		t.Errorf("OnStep kinds = %v, want %v", got, want)
	}
}

// TestReAct_BudgetExhaustion proves the ReAct scratchpad loop honors a MaxCalls
// budget at the chokepoint (35-04 / CC-1).
//
// ReAct typically needs ≥3 LLM calls in a scratchpad cycle (action → observe
// → action → observe → final). With Budget{MaxCalls: 2} we expect the third
// pre-call charge to be denied, so the loop returns zero Result + the
// dim-specific sentinel. The first two LLM calls successfully run a tool
// each (recordingTool); the denied 3rd attempt does NOT mutate the tracker
// (check-before-commit), so Snapshot().Calls == 2.
func TestReAct_BudgetExhaustion(t *testing.T) {
	ctx, _ := budget.WithBudget(context.Background(), budget.Budget{MaxCalls: 2})

	tool := &recordingTool{name: "echo", out: "ok"}
	reg := NewRegistry(tool)

	// Script 4 turns — all Action+Args, never Final — so the loop keeps
	// driving past MaxSteps. With MaxCalls=2 the 3rd pre-call charge denies
	// before MaxSteps can fire (MaxSteps=8 default).
	llmMock := newScriptedLLM(
		textResp("Thought: t1\nAction: echo\nArgs: {\"i\":1}"),
		textResp("Thought: t2\nAction: echo\nArgs: {\"i\":2}"),
		textResp("Thought: t3\nAction: echo\nArgs: {\"i\":3}"),
		textResp("Thought: t4\nAction: echo\nArgs: {\"i\":4}"),
	)
	a := NewReActAgent(llmMock, ReActOptions{Registry: reg, MaxSteps: 5})

	result, err := a.Run(ctx, "go")
	if !errors.Is(err, budget.ErrCallsExceeded) {
		t.Fatalf("err = %v, want ErrCallsExceeded", err)
	}
	if !errors.Is(err, budget.ErrBudgetExceeded) {
		t.Fatalf("err = %v, want ErrBudgetExceeded (umbrella)", err)
	}
	if !reflect.DeepEqual(result, Result{}) {
		t.Fatalf("expected zero Result on chokepoint error (react.go:106), got %+v", result)
	}

	tr, ok := budget.From(ctx)
	if !ok {
		t.Fatalf("budget.From(ctx) returned ok=false")
	}
	if got := tr.Snapshot().Calls; got != 2 {
		t.Errorf("tracker Snapshot().Calls = %d, want 2 (cap; denied 3rd attempt did not mutate)", got)
	}
}

// --- helpers (used by other agent tests too) ---

func traceKinds(trace []Step) []StepKind {
	out := make([]StepKind, len(trace))
	for i, s := range trace {
		out[i] = s.Kind
	}
	return out
}

func equalSlice[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

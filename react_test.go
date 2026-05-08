package agents

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

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

func TestReActAgent_MaxStepsExceeded(t *testing.T) {
	tool := &recordingTool{name: "loop"}
	reg := NewRegistry(tool)
	// Always return Action — never Final → expect ErrMaxStepsExceeded
	resps := make([]llm.GenerateResponse, 10)
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

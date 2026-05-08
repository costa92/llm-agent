package orchestrate

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/costa92/llm-agent"
)

// scriptedAgent returns a sequence of pre-set replies, one per call.
type scriptedAgent struct {
	name    string
	mu      sync.Mutex
	calls   int
	replies []string
}

func (s *scriptedAgent) Name() string { return s.name }
func (s *scriptedAgent) Run(_ context.Context, _ string) (agents.Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.calls >= len(s.replies) {
		s.calls++
		return agents.Result{}, errors.New("scripted: exhausted")
	}
	r := s.replies[s.calls]
	s.calls++
	return agents.Result{Answer: r, Usage: agents.Usage{LLMCalls: 1, Tokens: len(r)}}, nil
}
func (s *scriptedAgent) RunStream(_ context.Context, _ string) (<-chan agents.StepEvent, error) {
	return nil, errors.New("scripted: stream unsupported")
}

func TestRolePlay_RunsUntilDoneMarker(t *testing.T) {
	user := &scriptedAgent{name: "user", replies: []string{
		"Step 1: write a hello",
		"Step 2: now sign off",
		"Great work — <TASK_DONE>",
	}}
	assistant := &scriptedAgent{name: "assistant", replies: []string{
		"Hello!",
		"Goodbye!",
		// Not reached — user said TASK_DONE first
	}}

	rp := NewRolePlay(user, assistant, "task", RolePlayOptions{MaxTurns: 10})
	res, err := rp.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.Concluded {
		t.Errorf("expected Concluded=true")
	}
	if len(res.Turns) != 3 {
		t.Fatalf("got %d turns, want 3 (last has user-only TASK_DONE)", len(res.Turns))
	}
}

func TestRolePlay_AssistantTriggersDone(t *testing.T) {
	user := &scriptedAgent{name: "user", replies: []string{"do task X"}}
	assistant := &scriptedAgent{name: "assistant", replies: []string{"Result computed. <TASK_DONE>"}}

	rp := NewRolePlay(user, assistant, "task X", RolePlayOptions{MaxTurns: 10})
	res, _ := rp.Run(context.Background())
	if !res.Concluded {
		t.Errorf("expected Concluded=true when assistant emits marker")
	}
	if !strings.Contains(res.FinalOutput, "<TASK_DONE>") {
		t.Errorf("FinalOutput should be assistant's last message: %q", res.FinalOutput)
	}
}

func TestRolePlay_HitsMaxTurns(t *testing.T) {
	// Neither side ever says done.
	user := &scriptedAgent{name: "user", replies: []string{"keep going", "more", "and more"}}
	assistant := &scriptedAgent{name: "assistant", replies: []string{"ok", "ok", "ok"}}

	rp := NewRolePlay(user, assistant, "endless", RolePlayOptions{MaxTurns: 3})
	res, _ := rp.Run(context.Background())
	if res.Concluded {
		t.Errorf("expected Concluded=false on MaxTurns hit")
	}
	if len(res.Turns) != 3 {
		t.Fatalf("got %d turns, want 3", len(res.Turns))
	}
}

func TestRolePlay_NilAgentErrors(t *testing.T) {
	_, err := NewRolePlay(nil, &scriptedAgent{}, "t", RolePlayOptions{}).Run(context.Background())
	if !errors.Is(err, ErrNilAgent) {
		t.Errorf("expected ErrNilAgent for nil user, got %v", err)
	}
	_, err = NewRolePlay(&scriptedAgent{}, nil, "t", RolePlayOptions{}).Run(context.Background())
	if !errors.Is(err, ErrNilAgent) {
		t.Errorf("expected ErrNilAgent for nil assistant, got %v", err)
	}
}

func TestRolePlay_AccumulatesUsage(t *testing.T) {
	user := &scriptedAgent{name: "u", replies: []string{"a", "b", "<TASK_DONE>"}}
	assistant := &scriptedAgent{name: "a", replies: []string{"x", "y"}}
	rp := NewRolePlay(user, assistant, "t", RolePlayOptions{MaxTurns: 10})
	res, _ := rp.Run(context.Background())
	// 2 user calls before TASK_DONE → 2 user replies + 2 assistant replies + 1 final user TASK_DONE = 5 LLM calls
	if res.Usage.LLMCalls != 5 {
		t.Errorf("LLMCalls = %d, want 5", res.Usage.LLMCalls)
	}
}

func TestRolePlay_DefaultDoneMarker(t *testing.T) {
	user := &scriptedAgent{name: "u", replies: []string{"<TASK_DONE>"}}
	rp := NewRolePlay(user, &scriptedAgent{name: "a"}, "t", RolePlayOptions{})
	res, _ := rp.Run(context.Background())
	if !res.Concluded {
		t.Error("default DoneMarker <TASK_DONE> should trigger conclusion")
	}
}

func TestRolePlay_CustomInitPrompt(t *testing.T) {
	// Track that the user agent sees the custom InitPrompt verbatim.
	user := &capturingAgent{name: "u", reply: "<TASK_DONE>"}
	rp := NewRolePlay(user, &scriptedAgent{name: "a"}, "ignored task",
		RolePlayOptions{InitPrompt: "CUSTOM PROMPT XYZ"})
	_, _ = rp.Run(context.Background())
	if user.lastInput != "CUSTOM PROMPT XYZ" {
		t.Errorf("user got %q, want CUSTOM PROMPT XYZ", user.lastInput)
	}
}

type capturingAgent struct {
	name      string
	reply     string
	lastInput string
}

func (c *capturingAgent) Name() string { return c.name }
func (c *capturingAgent) Run(_ context.Context, input string) (agents.Result, error) {
	c.lastInput = input
	return agents.Result{Answer: c.reply, Usage: agents.Usage{LLMCalls: 1}}, nil
}
func (c *capturingAgent) RunStream(_ context.Context, _ string) (<-chan agents.StepEvent, error) {
	return nil, errors.New("nope")
}

package orchestrate

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/costa92/llm-agent"
)

// countingAgent records every input it received and returns a fixed reply.
type countingAgent struct {
	name        string
	reply       string
	callCount   atomic.Int32
	lastInput   atomic.Value // string
}

func (a *countingAgent) Name() string { return a.name }
func (a *countingAgent) Run(_ context.Context, input string) (agents.Result, error) {
	a.callCount.Add(1)
	a.lastInput.Store(input)
	return agents.Result{Answer: a.reply, Usage: agents.Usage{LLMCalls: 1, Tokens: 1}}, nil
}
func (a *countingAgent) RunStream(_ context.Context, _ string) (<-chan agents.StepEvent, error) {
	return nil, errors.New("countingAgent: stream unsupported")
}

func TestRoundRobin_RotatesSpeakers(t *testing.T) {
	a := &countingAgent{name: "A", reply: "from A"}
	b := &countingAgent{name: "B", reply: "from B"}
	c := &countingAgent{name: "C", reply: "from C"}

	chat := NewRoundRobinChat("test", []agents.Agent{a, b, c}, RoundRobinOptions{
		MaxTurns: 6,
	})
	res, err := chat.Run(context.Background(), "discuss")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.History) != 6 {
		t.Fatalf("got %d turns, want 6", len(res.History))
	}
	want := []string{"A", "B", "C", "A", "B", "C"}
	for i, w := range want {
		if res.History[i].Speaker != w {
			t.Errorf("turn %d speaker = %q, want %q", i, res.History[i].Speaker, w)
		}
	}
	if res.Stopped != "max_turns" {
		t.Errorf("Stopped = %q, want max_turns", res.Stopped)
	}
}

func TestRoundRobin_TerminationStopsEarly(t *testing.T) {
	a := &countingAgent{name: "A", reply: "first"}
	b := &countingAgent{name: "B", reply: "<TASK_DONE>"}
	c := &countingAgent{name: "C", reply: "should not run"}

	chat := NewRoundRobinChat("test", []agents.Agent{a, b, c}, RoundRobinOptions{
		MaxTurns:    10,
		Termination: TextMatch("<TASK_DONE>"),
	})
	res, _ := chat.Run(context.Background(), "task")
	if len(res.History) != 2 {
		t.Fatalf("got %d turns, want 2 (terminated by B's marker)", len(res.History))
	}
	if res.Stopped != "termination" {
		t.Errorf("Stopped = %q, want termination", res.Stopped)
	}
	if c.callCount.Load() != 0 {
		t.Errorf("C should not have run; callCount = %d", c.callCount.Load())
	}
}

func TestRoundRobin_AccumulatesUsage(t *testing.T) {
	a := &countingAgent{name: "A", reply: "x"}
	chat := NewRoundRobinChat("test", []agents.Agent{a}, RoundRobinOptions{MaxTurns: 5})
	res, _ := chat.Run(context.Background(), "task")
	if res.Usage.LLMCalls != 5 {
		t.Errorf("LLMCalls = %d, want 5", res.Usage.LLMCalls)
	}
}

func TestRoundRobin_PromptIncludesPriorHistory(t *testing.T) {
	a := &countingAgent{name: "A", reply: "alpha"}
	b := &countingAgent{name: "B", reply: "beta"}
	chat := NewRoundRobinChat("test", []agents.Agent{a, b}, RoundRobinOptions{MaxTurns: 3})
	_, _ = chat.Run(context.Background(), "discuss X")

	// Third turn = A again. Its lastInput should contain B's prior reply
	// AND A's own first reply.
	thirdInput, _ := a.lastInput.Load().(string)
	if !strings.Contains(thirdInput, "alpha") {
		t.Errorf("turn 3 input missing prior 'alpha': %q", thirdInput)
	}
	if !strings.Contains(thirdInput, "beta") {
		t.Errorf("turn 3 input missing prior 'beta': %q", thirdInput)
	}
	if !strings.Contains(thirdInput, "Task: discuss X") {
		t.Errorf("turn 3 input missing task header: %q", thirdInput)
	}
}

func TestRoundRobin_NoAgentsErrors(t *testing.T) {
	chat := NewRoundRobinChat("empty", nil, RoundRobinOptions{})
	_, err := chat.Run(context.Background(), "task")
	if !errors.Is(err, ErrNoAgents) {
		t.Errorf("expected ErrNoAgents, got %v", err)
	}
}

func TestRoundRobin_ContextCancelStops(t *testing.T) {
	a := &countingAgent{name: "A", reply: "x"}
	chat := NewRoundRobinChat("test", []agents.Agent{a}, RoundRobinOptions{MaxTurns: 100})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel
	res, _ := chat.Run(ctx, "task")
	if res.Stopped != "ctx_cancel" {
		t.Errorf("Stopped = %q, want ctx_cancel", res.Stopped)
	}
	if len(res.History) != 0 {
		t.Errorf("history non-empty after pre-cancel: %v", res.History)
	}
}

func TestRoundRobin_DefaultMaxTurns(t *testing.T) {
	a := &countingAgent{name: "A", reply: "x"}
	chat := NewRoundRobinChat("test", []agents.Agent{a}, RoundRobinOptions{}) // no MaxTurns
	res, _ := chat.Run(context.Background(), "task")
	if len(res.History) != 20 {
		t.Errorf("got %d turns, want default 20", len(res.History))
	}
}

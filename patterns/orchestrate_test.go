package patterns

import (
	"context"
	"errors"
	"strings"
	"testing"

	agents "github.com/costa92/llm-agent"
	"github.com/costa92/llm-agent/orchestrate"
)

type stubAgent struct {
	name      string
	transform func(string) string
	err       error
}

func (a *stubAgent) Name() string {
	if a.name == "" {
		return "stub"
	}
	return a.name
}

func (a *stubAgent) Run(_ context.Context, input string) (agents.Result, error) {
	if a.err != nil {
		return agents.Result{}, a.err
	}
	out := input
	if a.transform != nil {
		out = a.transform(input)
	}
	return agents.Result{
		Answer: out,
		Trace:  []agents.Step{{Kind: agents.StepFinal, Content: out}},
		Usage:  agents.Usage{LLMCalls: 1, Tokens: len(out)},
	}, nil
}

func (a *stubAgent) RunStream(_ context.Context, _ string) (<-chan agents.StepEvent, error) {
	return nil, errors.New("stubAgent: stream not implemented")
}

func TestParseDispatchLine(t *testing.T) {
	d, err := ParseDispatchLine("dispatch to worker: input")
	if err != nil {
		t.Fatalf("ParseDispatchLine: %v", err)
	}
	if d.WorkerName != "worker" || d.Input != "input" {
		t.Fatalf("dispatch = %#v, want worker/input", d)
	}
	d, err = ParseDispatchLine("FINISH")
	if err != nil || d != nil {
		t.Fatalf("finish = (%#v,%v), want nil,nil", d, err)
	}
	if _, err := ParseDispatchLine("bad"); err == nil {
		t.Fatal("bad dispatch returned nil error")
	}
}

func TestBuildSupervisorDefaultParserAndAggregator(t *testing.T) {
	planner := &stubAgent{name: "planner", transform: func(input string) string {
		if strings.Contains(input, "Round 1") {
			return "dispatch to w: one"
		}
		return "FINISH"
	}}
	worker := &stubAgent{name: "w", transform: func(input string) string { return "worker:" + input }}
	sup := BuildSupervisor(SupervisorOptions{
		Name:      "sup",
		Planner:   planner,
		Workers:   map[string]agents.Agent{"w": worker},
		MaxRounds: 2,
	})
	res, err := sup.Run(context.Background(), "seed")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Answer != "w(one)=worker:one" {
		t.Fatalf("answer = %q, want joined worker result", res.Answer)
	}
}

func TestBuildSupervisorUnknownWorker(t *testing.T) {
	planner := &stubAgent{name: "planner", transform: func(string) string {
		return "dispatch to missing: input"
	}}
	sup := BuildSupervisor(SupervisorOptions{
		Planner: planner,
		Workers: map[string]agents.Agent{"w": &stubAgent{name: "w"}},
	})
	_, err := sup.Run(context.Background(), "seed")
	if !errors.Is(err, orchestrate.ErrSupervisorUnknownWorker) {
		t.Fatalf("err = %v, want ErrSupervisorUnknownWorker", err)
	}
}

func TestBuildFanOutFanInDefaultPlanAndAggregate(t *testing.T) {
	planner := &stubAgent{name: "planner", transform: func(string) string { return "alpha\nbeta" }}
	worker := &stubAgent{name: "worker", transform: func(input string) string { return "done:" + input }}
	aggregator := &stubAgent{name: "agg", transform: func(input string) string {
		if !strings.Contains(input, "done:alpha") || !strings.Contains(input, "done:beta") {
			t.Fatalf("aggregate input missing worker results: %q", input)
		}
		return "summary"
	}}
	f := BuildFanOutFanIn(FanOutOptions{
		Planner:    planner,
		Workers:    map[string]agents.Agent{"worker": worker},
		Aggregator: aggregator,
	})
	res, err := f.Run(context.Background(), "seed")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.FinalAnswer != "summary" {
		t.Fatalf("final = %q, want summary", res.FinalAnswer)
	}
}

func TestBuildRoundRobinPreservesChatResult(t *testing.T) {
	a := &stubAgent{name: "a", transform: func(string) string { return "first" }}
	b := &stubAgent{name: "b", transform: func(string) string { return "done" }}
	chat := BuildRoundRobin(RoundRobinOptions{
		Agents:      []agents.Agent{a, b},
		MaxTurns:    4,
		Termination: orchestrate.TextMatch("done"),
	})
	res, err := chat.Run(context.Background(), "task")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Stopped != "termination" || len(res.History) != 2 {
		t.Fatalf("result = %#v, want termination with two turns", res)
	}
}

func TestBuildRolePlayPreservesRolePlayResult(t *testing.T) {
	user := &stubAgent{name: "user", transform: func(string) string { return "please execute" }}
	assistant := &stubAgent{name: "assistant", transform: func(string) string { return "done <TASK_DONE>" }}
	rp := BuildRolePlay(RolePlayOptions{User: user, Assistant: assistant, TaskPrompt: "task", MaxTurns: 2})
	res, err := rp.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.Concluded || res.FinalOutput != "done <TASK_DONE>" {
		t.Fatalf("result = %#v, want concluded final output", res)
	}
}

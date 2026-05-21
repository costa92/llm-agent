package orchestrate

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	agents "github.com/costa92/llm-agent"
)

type supervisorCountingAgent struct {
	name      string
	transform func(input string) string
	llmCalls  int
}

func (a *supervisorCountingAgent) Name() string { return a.name }

func (a *supervisorCountingAgent) Run(_ context.Context, input string) (agents.Result, error) {
	out := input
	if a.transform != nil {
		out = a.transform(input)
	}
	calls := a.llmCalls
	if calls == 0 {
		calls = 1
	}
	return agents.Result{Answer: out, Usage: agents.Usage{LLMCalls: calls, Tokens: len(out)}}, nil
}

func (a *supervisorCountingAgent) RunStream(_ context.Context, _ string) (<-chan agents.StepEvent, error) {
	return nil, errors.New("countingAgent: stream not implemented")
}

func parseDemoDispatch(plannerAnswer string) (*Dispatch, error) {
	trimmed := strings.TrimSpace(plannerAnswer)
	switch {
	case trimmed == "":
		return nil, nil
	case strings.EqualFold(trimmed, "FINISH"):
		return nil, nil
	case strings.HasPrefix(strings.ToLower(trimmed), "dispatch to "):
		rest := strings.TrimSpace(trimmed[len("dispatch to "):])
		parts := strings.SplitN(rest, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid dispatch %q", plannerAnswer)
		}
		worker := strings.TrimSpace(parts[0])
		input := strings.TrimSpace(parts[1])
		if worker == "" || input == "" {
			return nil, fmt.Errorf("invalid dispatch %q", plannerAnswer)
		}
		return &Dispatch{WorkerName: worker, Input: input}, nil
	default:
		return nil, fmt.Errorf("invalid dispatch %q", plannerAnswer)
	}
}

func joinWorkerResults(results []WorkerResult) (string, error) {
	if len(results) == 0 {
		return "", nil
	}
	parts := make([]string, 0, len(results))
	for _, wr := range results {
		parts = append(parts, fmt.Sprintf("%s(%s)=%s", wr.Dispatch.WorkerName, wr.Dispatch.Input, wr.Result.Answer))
	}
	return strings.Join(parts, " | "), nil
}

func validOpts() SupervisorOptions {
	planner := &supervisorCountingAgent{
		name: "planner",
		transform: func(input string) string {
			switch {
			case strings.Contains(input, "Round 1"):
				return "dispatch to w: one"
			case strings.Contains(input, "Round 2"):
				return "dispatch to w: two"
			default:
				return "FINISH"
			}
		},
	}
	worker := &supervisorCountingAgent{
		name: "w",
		transform: func(input string) string {
			return "worker:" + input
		},
	}
	return SupervisorOptions{
		Planner: planner,
		Workers: map[string]agents.Agent{
			"w": worker,
		},
		MaxRounds:       3,
		ParseDispatch:    parseDemoDispatch,
		BuildAggregate: func(results []WorkerResult) (string, error) {
			joined, err := joinWorkerResults(results)
			if err != nil {
				return "", err
			}
			if joined == "" {
				return "final:", nil
			}
			return "final: " + joined, nil
		},
	}
}

func TestSupervisor_HappyPath(t *testing.T) {
	sup := NewSupervisor("sup", validOpts())
	res, err := sup.Run(context.Background(), "seed")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := "final: w(one)=worker:one | w(two)=worker:two"
	if res.Answer != want {
		t.Fatalf("Answer = %q, want %q", res.Answer, want)
	}
}

func TestSupervisor_Validation(t *testing.T) {
	tests := []struct {
		name string
		mut  func(*SupervisorOptions)
		err  error
	}{
		{name: "nil planner", mut: func(o *SupervisorOptions) { o.Planner = nil }, err: ErrSupervisorNilPlanner},
		{name: "no workers", mut: func(o *SupervisorOptions) { o.Workers = nil }, err: ErrSupervisorNoWorkers},
		{name: "nil parse", mut: func(o *SupervisorOptions) { o.ParseDispatch = nil }, err: ErrSupervisorNilParseDispatch},
		{name: "nil aggregate", mut: func(o *SupervisorOptions) { o.BuildAggregate = nil }, err: ErrSupervisorNilBuildAggregate},
		{name: "zero max rounds", mut: func(o *SupervisorOptions) { o.MaxRounds = 0 }, err: nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opts := validOpts()
			tc.mut(&opts)
			_, err := NewSupervisor("sup", opts).Run(context.Background(), "seed")
			if tc.err != nil {
				if !errors.Is(err, tc.err) {
					t.Fatalf("errors.Is(%v, %v) = false", err, tc.err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "MaxRounds must be positive") {
				t.Fatalf("expected max-rounds validation error, got %v", err)
			}
		})
	}
}

func TestSupervisor_MaxRoundsExceeded(t *testing.T) {
	opts := validOpts()
	opts.MaxRounds = 1
	opts.Planner = &supervisorCountingAgent{
		name: "planner",
		transform: func(string) string {
			return "FINISH"
		},
	}
	sup := NewSupervisor("sup", opts)
	res, err := sup.Run(context.Background(), "seed")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Answer != "final:" {
		t.Fatalf("Answer = %q", res.Answer)
	}
}

func TestSupervisor_UnknownWorker(t *testing.T) {
	opts := validOpts()
	opts.Planner = &supervisorCountingAgent{
		name: "planner",
		transform: func(string) string { return "dispatch to missing: input" },
	}
	sup := NewSupervisor("sup", opts)
	_, err := sup.Run(context.Background(), "seed")
	if !errors.Is(err, ErrSupervisorUnknownWorker) {
		t.Fatalf("errors.Is(err, ErrSupervisorUnknownWorker) = false: %v", err)
	}
}

func TestSupervisor_ParseDispatchError(t *testing.T) {
	opts := validOpts()
	opts.Planner = &supervisorCountingAgent{
		name: "planner",
		transform: func(string) string { return "no dispatch here" },
	}
	sup := NewSupervisor("sup", opts)
	_, err := sup.Run(context.Background(), "seed")
	if !errors.Is(err, ErrSupervisorParseDispatch) {
		t.Fatalf("errors.Is(err, ErrSupervisorParseDispatch) = false: %v", err)
	}
	if !strings.Contains(err.Error(), "no dispatch here") {
		t.Fatalf("expected parser text in error, got %v", err)
	}
}

func TestSupervisor_ParseDispatchFinish(t *testing.T) {
	opts := validOpts()
	opts.Planner = &supervisorCountingAgent{
		name: "planner",
		transform: func(string) string { return "FINISH" },
	}
	opts.BuildAggregate = func(results []WorkerResult) (string, error) {
		if len(results) != 0 {
			t.Fatalf("BuildAggregate got %d results, want 0", len(results))
		}
		return "finished", nil
	}
	sup := NewSupervisor("sup", opts)
	res, err := sup.Run(context.Background(), "seed")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Answer != "finished" {
		t.Fatalf("Answer = %q, want finished", res.Answer)
	}
	if res.Usage.LLMCalls != 1 {
		t.Fatalf("LLMCalls = %d, want 1", res.Usage.LLMCalls)
	}
}

func TestSupervisor_CtxCancel(t *testing.T) {
	opts := validOpts()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewSupervisor("sup", opts).Run(ctx, "seed")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("errors.Is(err, context.Canceled) = false: %v", err)
	}
}

func TestSupervisor_RunStreamEmitsRoundEvents(t *testing.T) {
	opts := validOpts()
	opts.MaxRounds = 2
	sup := NewSupervisor("sup", opts)
	ch, err := sup.RunStream(context.Background(), "seed")
	if err != nil {
		t.Fatalf("RunStream: %v", err)
	}
	var kinds []agents.StepKind
	var done *agents.Result
	for ev := range ch {
		if ev.Done {
			done = ev.Final
			if ev.Err != nil {
				t.Fatalf("terminal err: %v", ev.Err)
			}
			continue
		}
		kinds = append(kinds, ev.Step.Kind)
	}
	want := []agents.StepKind{
		agents.StepAction,
		agents.StepObservation,
		agents.StepAction,
		agents.StepObservation,
		agents.StepFinal,
	}
	if len(kinds) != len(want) {
		t.Fatalf("got %d kinds, want %d: %v", len(kinds), len(want), kinds)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("kind[%d] = %q, want %q", i, kinds[i], want[i])
		}
	}
	if done == nil || done.Answer != "final: w(one)=worker:one | w(two)=worker:two" {
		t.Fatalf("terminal result = %#v", done)
	}
}

func TestSupervisor_UsageRollup(t *testing.T) {
	opts := validOpts()
	opts.MaxRounds = 2
	opts.Planner = &supervisorCountingAgent{
		name:      "planner",
		transform: func(string) string { return "dispatch to w: one" },
		llmCalls:  2,
	}
	opts.Workers["w"] = &supervisorCountingAgent{
		name:      "w",
		transform: func(input string) string { return "worker:" + input },
		llmCalls:  3,
	}
	sup := NewSupervisor("sup", opts)
	res, err := sup.Run(context.Background(), "seed")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Usage.LLMCalls != 12 {
		t.Fatalf("LLMCalls = %d, want 12", res.Usage.LLMCalls)
	}
	if res.Usage.Tokens == 0 {
		t.Fatal("Tokens not accumulated")
	}
}

func TestSupervisor_NameDefault(t *testing.T) {
	if got := NewSupervisor("", validOpts()).Name(); got != "supervisor" {
		t.Fatalf("Name() = %q, want supervisor", got)
	}
}

func TestSupervisor_SatisfiesAgentInterface(t *testing.T) {
	var got agents.Agent = NewSupervisor("sup", validOpts())
	if got == nil || got.Name() != "sup" {
		t.Fatalf("agent assertion failed: %#v", got)
	}
}

func TestSupervisor_ConcurrentRunIsRaceClean(t *testing.T) {
	opts := validOpts()
	sup := NewSupervisor("sup", opts)
	var okRuns int64
	var wg sync.WaitGroup
	for g := 0; g < 2; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				res, err := sup.Run(context.Background(), "seed")
				if err != nil {
					t.Errorf("Run: %v", err)
					return
				}
				if res.Answer == "" {
					t.Error("empty answer")
					return
				}
				atomic.AddInt64(&okRuns, 1)
			}
		}()
	}
	wg.Wait()
	if okRuns != 20 {
		t.Fatalf("okRuns = %d, want 20", okRuns)
	}
}

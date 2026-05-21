package orchestrate

import (
	"context"
	"errors"
	"strings"
	"testing"

	agents "github.com/costa92/llm-agent"
)

type composeWState struct {
	in  string
	out string
}

type graphAsAgent struct {
	name string
	cg   *CompiledGraph[composeWState]
}

var _ agents.Agent = (*graphAsAgent)(nil)

func (g *graphAsAgent) Name() string { return g.name }

func (g *graphAsAgent) Run(ctx context.Context, input string) (agents.Result, error) {
	final, err := g.cg.Run(ctx, composeWState{in: input})
	if err != nil {
		return agents.Result{}, err
	}
	return agents.Result{Answer: final.out, Usage: agents.Usage{LLMCalls: 1, Tokens: len(final.out)}}, nil
}

func (g *graphAsAgent) RunStream(_ context.Context, _ string) (<-chan agents.StepEvent, error) {
	return nil, errors.New("graphAsAgent: stream not implemented")
}

func TestSupervisor_InsideStateGraph(t *testing.T) {
	sup := NewSupervisor("inner", validOpts())
	type outerState struct {
		input string
		supRes string
		final  string
	}
	g := NewStateGraph[outerState]()
	g.AddNode("preprocess", func(_ context.Context, st outerState) (outerState, error) {
		st.input = strings.TrimSpace(st.input)
		return st, nil
	})
	g.AddNode("supervise", func(ctx context.Context, st outerState) (outerState, error) {
		res, err := sup.Run(ctx, st.input)
		if err != nil {
			return st, err
		}
		st.supRes = res.Answer
		return st, nil
	})
	g.AddNode("postprocess", func(_ context.Context, st outerState) (outerState, error) {
		st.final = "[" + st.supRes + "]"
		return st, nil
	})
	g.SetEntry("preprocess").
		AddEdge("preprocess", "supervise").
		AddEdge("supervise", "postprocess").
		AddEdge("postprocess", NodeEnd)
	cg, err := g.Compile()
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	out, err := cg.Run(context.Background(), outerState{input: "seed"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := "[final: w(one)=worker:one | w(two)=worker:two]"
	if out.final != want {
		t.Fatalf("final = %q, want %q", out.final, want)
	}
}

func TestStateGraph_InsideSupervisor(t *testing.T) {
	g := NewStateGraph[composeWState]()
	g.AddNode("transform", func(_ context.Context, st composeWState) (composeWState, error) {
		st.out = "transformed: " + st.in
		return st, nil
	})
	g.SetEntry("transform").AddEdge("transform", NodeEnd)
	cg, err := g.Compile()
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	worker := &graphAsAgent{name: "graph-worker", cg: cg}
	opts := validOpts()
	opts.Planner = &supervisorCountingAgent{
		name:      "planner",
		transform: func(string) string { return "dispatch to graph-worker: hi" },
	}
	opts.Workers = map[string]agents.Agent{"graph-worker": worker}
	opts.BuildAggregate = func(results []WorkerResult) (string, error) {
		joined, err := joinWorkerResults(results)
		if err != nil {
			return "", err
		}
		return "agg: " + joined, nil
	}
	sup := NewSupervisor("outer", opts)
	res, err := sup.Run(context.Background(), "seed")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(res.Answer, "transformed: hi") {
		t.Fatalf("Answer = %q", res.Answer)
	}
}

func TestSupervisor_OfSupervisor(t *testing.T) {
	innerOpts := validOpts()
	innerOpts.Planner = &supervisorCountingAgent{
		name:      "inner-planner",
		transform: func(string) string { return "dispatch to w: nested" },
	}
	inner := NewSupervisor("inner", innerOpts)

	outerOpts := validOpts()
	outerOpts.Planner = &supervisorCountingAgent{
		name:      "outer-planner",
		transform: func(string) string { return "dispatch to sub: nested" },
	}
	outerOpts.Workers = map[string]agents.Agent{"sub": inner}
	outerOpts.BuildAggregate = func(results []WorkerResult) (string, error) {
		joined, err := joinWorkerResults(results)
		if err != nil {
			return "", err
		}
		return "outer<" + joined + ">", nil
	}
	outer := NewSupervisor("outer", outerOpts)
	res, err := outer.Run(context.Background(), "seed")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(res.Answer, "worker:nested") {
		t.Fatalf("Answer = %q", res.Answer)
	}
	if !strings.HasPrefix(res.Answer, "outer<") {
		t.Fatalf("Answer = %q", res.Answer)
	}
}

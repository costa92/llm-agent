// Demo 08: Supervisor / multi-agent coordination.
//
// Demonstrates the Phase 37 `orchestrate.Supervisor` surface in three
// deterministic shapes:
//
//   - Basic   : planner + 2 workers, 2 rounds, aggregate final answer.
//   - Budget  : the same kind of supervisor under budget.WithBudget,
//               showing the budget cap surfaces as budget.ErrCallsExceeded.
//   - Compose : Supervisor inside a StateGraph node.
//
// Run:
//
//	cd examples && go run ./08-supervisor
package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	agents "github.com/costa92/llm-agent"
	"github.com/costa92/llm-agent/budget"
	"github.com/costa92/llm-agent-contract/llm"
	"github.com/costa92/llm-agent/orchestrate"
)

func main() {
	demoBasic()
	fmt.Println()
	demoBudget()
	fmt.Println()
	demoComposeWithStateGraph()
	fmt.Println("OK")
}

func demoBasic() {
	fmt.Println("--- Basic: planner + workers ---")
	sup := demoSupervisor(
		[]string{"dispatch to alpha: one", "dispatch to beta: two", "FINISH"},
		map[string]agents.Agent{
			"alpha": simpleWorker("alpha"),
			"beta":  simpleWorker("beta"),
		},
	)
	res, err := sup.Run(context.Background(), "assemble report")
	if err != nil {
		fmt.Printf("unexpected error: %v\n", err)
		return
	}
	fmt.Println(res.Answer)
}

func demoBudget() {
	fmt.Println("--- Budget: MaxCalls ---")
	sup := demoSupervisor(
		[]string{"dispatch to alpha: one", "dispatch to beta: two", "FINISH"},
		map[string]agents.Agent{
			"alpha": simpleWorker("alpha"),
			"beta":  simpleWorker("beta"),
		},
	)
	ctx, _ := budget.WithBudget(context.Background(), budget.Budget{MaxCalls: 3})
	_, err := sup.Run(ctx, "assemble report")
	fmt.Printf("errors.Is(err, budget.ErrCallsExceeded) = %v\n", errors.Is(err, budget.ErrCallsExceeded))
	fmt.Printf("errors.Is(err, budget.ErrBudgetExceeded) = %v\n", errors.Is(err, budget.ErrBudgetExceeded))
}

func demoComposeWithStateGraph() {
	fmt.Println("--- Compose: Supervisor in StateGraph ---")
	sup := demoSupervisor(
		[]string{"dispatch to alpha: one", "FINISH"},
		map[string]agents.Agent{"alpha": simpleWorker("alpha")},
	)
	type outerState struct {
		input string
		out   string
	}
	g := orchestrate.NewStateGraph[outerState]()
	g.AddNode("supervise", func(ctx context.Context, st outerState) (outerState, error) {
		res, err := sup.Run(ctx, st.input)
		if err != nil {
			return st, err
		}
		st.out = "compose:" + res.Answer
		return st, nil
	})
	g.SetEntry("supervise").AddEdge("supervise", orchestrate.NodeEnd)
	cg, err := g.Compile()
	if err != nil {
		fmt.Printf("unexpected error: %v\n", err)
		return
	}
	out, err := cg.Run(context.Background(), outerState{input: "compose"})
	if err != nil {
		fmt.Printf("unexpected error: %v\n", err)
		return
	}
	fmt.Println(out.out)
}

func demoSupervisor(plannerResponses []string, workers map[string]agents.Agent) *orchestrate.Supervisor {
	return orchestrate.NewSupervisor("demo", orchestrate.SupervisorOptions{
		Planner:        agents.NewSimpleAgent(scriptedModel(plannerResponses...), agents.SimpleOptions{Name: "planner"}),
		Workers:        workers,
		MaxRounds:      3,
		ParseDispatch:  parseDispatch,
		BuildAggregate: aggregateResults,
	})
}

func parseDispatch(answer string) (*orchestrate.Dispatch, error) {
	trimmed := strings.TrimSpace(answer)
	if strings.EqualFold(trimmed, "FINISH") {
		return nil, nil
	}
	if !strings.HasPrefix(strings.ToLower(trimmed), "dispatch to ") {
		return nil, fmt.Errorf("invalid dispatch %q", answer)
	}
	rest := strings.TrimSpace(trimmed[len("dispatch to "):])
	parts := strings.SplitN(rest, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid dispatch %q", answer)
	}
	return &orchestrate.Dispatch{
		WorkerName: strings.TrimSpace(parts[0]),
		Input:      strings.TrimSpace(parts[1]),
	}, nil
}

func aggregateResults(results []orchestrate.WorkerResult) (string, error) {
	parts := make([]string, 0, len(results))
	for _, wr := range results {
		parts = append(parts, fmt.Sprintf("%s(%s)=%s", wr.Dispatch.WorkerName, wr.Dispatch.Input, wr.Result.Answer))
	}
	if len(parts) == 0 {
		return "final: <none>", nil
	}
	return "final: " + strings.Join(parts, " | "), nil
}

func simpleWorker(name string) agents.Agent {
	model := llm.NewScriptedLLM(
		llm.WithResponses(
			llm.Response{Text: "worker:" + name + ":1", Provider: "scripted"},
			llm.Response{Text: "worker:" + name + ":2", Provider: "scripted"},
		),
	)
	return agents.NewSimpleAgent(model, agents.SimpleOptions{Name: name})
}

func scriptedModel(responses ...string) llm.ChatModel {
	out := make([]llm.Response, 0, len(responses))
	for _, r := range responses {
		out = append(out, llm.Response{Text: r, Provider: "scripted", Usage: llm.Usage{Source: llm.UsageReported}})
	}
	return llm.NewScriptedLLM(llm.WithResponses(out...))
}

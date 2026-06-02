package orchestrate

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	agents "github.com/costa92/llm-agent"
	"github.com/costa92/llm-agent/budget"
	"github.com/costa92/llm-agent-contract/llm"
	"github.com/costa92/llm-agent/policy"
)

type ctxKey int

const supervisorCtxKey ctxKey = 1

type slowLLM struct {
	inner llm.ChatModel
	delay time.Duration
}

func (s *slowLLM) Generate(ctx context.Context, req llm.Request) (llm.Response, error) {
	select {
	case <-time.After(s.delay):
		return s.inner.Generate(ctx, req)
	case <-ctx.Done():
		return llm.Response{}, ctx.Err()
	}
}

func (s *slowLLM) Stream(ctx context.Context, req llm.Request) (llm.StreamReader, error) {
	return s.inner.Stream(ctx, req)
}

func (s *slowLLM) Info() llm.ProviderInfo { return s.inner.Info() }

type ctxCheckingModel struct {
	inner       llm.ChatModel
	hits        *int64
	expectedVal any
}

func (m *ctxCheckingModel) Generate(ctx context.Context, req llm.Request) (llm.Response, error) {
	if got := ctx.Value(supervisorCtxKey); got != m.expectedVal {
		return llm.Response{}, errors.New("ctx value mismatch")
	}
	atomic.AddInt64(m.hits, 1)
	return m.inner.Generate(ctx, req)
}

func (m *ctxCheckingModel) Stream(ctx context.Context, req llm.Request) (llm.StreamReader, error) {
	return m.inner.Stream(ctx, req)
}

func (m *ctxCheckingModel) Info() llm.ProviderInfo { return m.inner.Info() }

type blockOnNthGate struct {
	name    string
	blockOn int64
	hits    *int64
}

func (g *blockOnNthGate) Name() string { return g.name }

func (g *blockOnNthGate) Inspect(_ context.Context, ev policy.Event) policy.Decision {
	if ev.Kind != policy.PreGenerate {
		return policy.Decision{Action: policy.Allow}
	}
	n := atomic.AddInt64(g.hits, 1)
	if n == g.blockOn {
		return policy.Decision{Action: policy.Block, Reason: "blocked"}
	}
	return policy.Decision{Action: policy.Allow}
}

func budgetSupervisorOpts(planner llm.ChatModel, worker llm.ChatModel) SupervisorOptions {
	return SupervisorOptions{
		Planner: agents.NewSimpleAgent(planner, agents.SimpleOptions{Name: "planner"}),
		Workers: map[string]agents.Agent{
			"w": agents.NewSimpleAgent(worker, agents.SimpleOptions{Name: "w"}),
		},
		MaxRounds:    3,
		ParseDispatch: parseDemoDispatch,
		BuildAggregate: func(results []WorkerResult) (string, error) {
			joined, err := joinWorkerResults(results)
			if err != nil {
				return "", err
			}
			return joined, nil
		},
	}
}

func TestSupervisor_BudgetPropagatesToWorker(t *testing.T) {
	planner := scriptedPlanner(
		"dispatch to w: a",
		"dispatch to w: b",
		"FINISH",
	)
	worker := scriptedWorker("worker-a", "worker-b")
	ctx, tracker := budget.WithBudget(context.Background(), budget.Budget{MaxCalls: 3})
	sup := NewSupervisor("budget", budgetSupervisorOpts(planner, worker))
	_, err := sup.Run(ctx, "seed")
	if !errors.Is(err, budget.ErrCallsExceeded) {
		t.Fatalf("errors.Is(err, budget.ErrCallsExceeded) = false: %v", err)
	}
	if !errors.Is(err, budget.ErrBudgetExceeded) {
		t.Fatalf("errors.Is(err, budget.ErrBudgetExceeded) = false: %v", err)
	}
	if tracker.Snapshot().Calls != 3 {
		t.Fatalf("tracker calls = %d, want 3", tracker.Snapshot().Calls)
	}
}

func TestSupervisor_BudgetMaxWall(t *testing.T) {
	planner := scriptedPlanner("dispatch to w: a")
	worker := &slowLLM{inner: scriptedWorker("slow"), delay: 200 * time.Millisecond}
	ctx, _ := budget.WithBudget(context.Background(), budget.Budget{MaxWall: 50 * time.Millisecond})
	_, err := NewSupervisor("wall", budgetSupervisorOpts(planner, worker)).Run(ctx, "seed")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("errors.Is(err, context.DeadlineExceeded) = false: %v", err)
	}
}

func TestSupervisor_PolicyPerWorker(t *testing.T) {
	var hits int64
	gate := &blockOnNthGate{name: "worker-gate", blockOn: 2, hits: &hits}
	worker := policy.Wrap(scriptedWorker("worker-1", "worker-2"), gate)
	planner := scriptedPlanner("dispatch to w: a", "dispatch to w: b", "FINISH")
	sup := NewSupervisor("policy-worker", budgetSupervisorOpts(planner, worker))
	_, err := sup.Run(context.Background(), "seed")
	if !errors.Is(err, policy.ErrBlocked) {
		t.Fatalf("errors.Is(err, policy.ErrBlocked) = false: %v", err)
	}
	if !strings.Contains(err.Error(), `policy-worker`) || !strings.Contains(err.Error(), `worker "w" round 2`) {
		t.Fatalf("wrapped worker context missing: %v", err)
	}
}

func TestSupervisor_PolicyPerPlanner(t *testing.T) {
	var hits int64
	gate := &blockOnNthGate{name: "planner-gate", blockOn: 1, hits: &hits}
	planner := policy.Wrap(scriptedPlanner("dispatch to w: a"), gate)
	sup := NewSupervisor("policy-planner", budgetSupervisorOpts(planner, scriptedWorker("worker")))
	_, err := sup.Run(context.Background(), "seed")
	if !errors.Is(err, policy.ErrBlocked) {
		t.Fatalf("errors.Is(err, policy.ErrBlocked) = false: %v", err)
	}
	if !strings.Contains(err.Error(), "planner round 1") {
		t.Fatalf("planner round missing: %v", err)
	}
}

func TestSupervisor_BudgetBeatsPolicy(t *testing.T) {
	var hits int64
	gate := &blockOnNthGate{name: "worker-gate", blockOn: 1, hits: &hits}
	worker := policy.Wrap(scriptedWorker("worker"), gate)
	planner := scriptedPlanner("dispatch to w: a", "FINISH")
	ctx, _ := budget.WithBudget(context.Background(), budget.Budget{MaxCalls: 1})
	_, err := NewSupervisor("budget-beats-policy", budgetSupervisorOpts(planner, worker)).Run(ctx, "seed")
	if !errors.Is(err, budget.ErrCallsExceeded) {
		t.Fatalf("errors.Is(err, budget.ErrCallsExceeded) = false: %v", err)
	}
	if errors.Is(err, policy.ErrBlocked) {
		t.Fatalf("policy block leaked through: %v", err)
	}
	if atomic.LoadInt64(&hits) != 0 {
		t.Fatalf("policy gate hit %d times, want 0", atomic.LoadInt64(&hits))
	}
}

func TestSupervisor_NoDetachedCtx(t *testing.T) {
	var hits int64
	planner := scriptedPlanner("dispatch to w: a", "FINISH")
	workerModel := &ctxCheckingModel{
		inner:       scriptedWorker("worker"),
		hits:        &hits,
		expectedVal: "sentinel",
	}
	ctx := context.WithValue(context.Background(), supervisorCtxKey, "sentinel")
	_, err := NewSupervisor("ctx", budgetSupervisorOpts(planner, workerModel)).Run(ctx, "seed")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if atomic.LoadInt64(&hits) == 0 {
		t.Fatal("worker did not observe ctx value")
	}
}

func scriptedPlanner(responses ...string) llm.ChatModel {
	parts := make([]llm.Response, 0, len(responses))
	for _, r := range responses {
		parts = append(parts, llm.Response{Text: r, Provider: "scripted", Usage: llm.Usage{Source: llm.UsageReported}})
	}
	return llm.NewScriptedLLM(llm.WithResponses(parts...))
}

func scriptedWorker(responses ...string) llm.ChatModel {
	parts := make([]llm.Response, 0, len(responses))
	for _, r := range responses {
		parts = append(parts, llm.Response{Text: r, Provider: "scripted", Usage: llm.Usage{Source: llm.UsageReported}})
	}
	return llm.NewScriptedLLM(llm.WithResponses(parts...))
}

package orchestrate

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/costa92/llm-agent/pkg/fanout"
	"github.com/costa92/llm-agent"
)

// PlannedTask is one unit of work emitted by the planner and executed by
// a worker agent.
type PlannedTask struct {
	Name   string
	Input  string
	Worker string // optional when exactly one worker is registered
}

// TaskResult pairs a planned task with the worker result that completed it.
type TaskResult struct {
	Task   PlannedTask
	Result agents.Result
}

// PlanParser converts the planner agent's result into concrete tasks.
// Nil uses the default newline parser: one non-empty line = one task.
type PlanParser func(plan agents.Result) ([]PlannedTask, error)

// AggregateInputBuilder formats the fan-out execution into the aggregator's
// input prompt. Nil uses a deterministic default formatter.
type AggregateInputBuilder func(originalInput string, plan agents.Result, results []TaskResult) string

// FanOutFanInOptions configures planner/worker/aggregator orchestration.
type FanOutFanInOptions struct {
	Planner        agents.Agent
	Workers        map[string]agents.Agent
	Aggregator     agents.Agent
	ParsePlan      PlanParser
	BuildAggregate AggregateInputBuilder
	MaxConcurrency int // default = len(tasks)
}

// FanOutFanIn runs a planner -> parallel workers -> aggregator workflow.
//
// Use this when an input naturally decomposes into mostly independent
// sub-tasks that can be executed concurrently and then summarized.
type FanOutFanIn struct {
	name string
	opts FanOutFanInOptions
}

// FanOutFanInResult carries the full plan, each worker result in task order,
// and the final aggregated answer.
type FanOutFanInResult struct {
	Plan             agents.Result
	Tasks            []PlannedTask
	WorkerResults    []TaskResult
	AggregatorResult agents.Result
	FinalAnswer      string
	TotalUsage       agents.Usage
}

// NewFanOutFanIn constructs a FanOutFanIn orchestrator.
func NewFanOutFanIn(name string, opts FanOutFanInOptions) *FanOutFanIn {
	return &FanOutFanIn{name: name, opts: opts}
}

// Name returns the orchestrator name used in errors/logs.
func (f *FanOutFanIn) Name() string {
	if f.name == "" {
		return "fanout-fanin"
	}
	return f.name
}

// Run executes planner -> workers -> aggregator.
func (f *FanOutFanIn) Run(ctx context.Context, input string) (FanOutFanInResult, error) {
	if f.opts.Planner == nil {
		return FanOutFanInResult{}, ErrFanOutNilPlanner
	}
	if f.opts.Aggregator == nil {
		return FanOutFanInResult{}, ErrFanOutNilAggregator
	}
	if len(f.opts.Workers) == 0 {
		return FanOutFanInResult{}, ErrFanOutNoWorkers
	}

	out := FanOutFanInResult{}

	planRes, err := f.opts.Planner.Run(ctx, input)
	if err != nil {
		return out, fmt.Errorf("fanout %q: planner: %w", f.Name(), err)
	}
	out.Plan = planRes
	out.TotalUsage = addUsage(out.TotalUsage, planRes.Usage)

	parser := f.opts.ParsePlan
	if parser == nil {
		parser = defaultPlanParser
	}
	tasks, err := parser(planRes)
	if err != nil {
		return out, fmt.Errorf("fanout %q: parse plan: %w", f.Name(), err)
	}
	if len(tasks) == 0 {
		return out, ErrFanOutNoTasks
	}
	out.Tasks = tasks

	workerResults, workerUsage, err := f.runWorkers(ctx, tasks)
	if err != nil {
		return out, err
	}
	out.WorkerResults = workerResults
	out.TotalUsage = addUsage(out.TotalUsage, workerUsage)

	aggInput := buildAggregateInput(input, planRes, workerResults, f.opts.BuildAggregate)
	aggRes, err := f.opts.Aggregator.Run(ctx, aggInput)
	if err != nil {
		return out, fmt.Errorf("fanout %q: aggregator: %w", f.Name(), err)
	}
	out.AggregatorResult = aggRes
	out.FinalAnswer = aggRes.Answer
	out.TotalUsage = addUsage(out.TotalUsage, aggRes.Usage)
	return out, nil
}

func (f *FanOutFanIn) runWorkers(ctx context.Context, tasks []PlannedTask) ([]TaskResult, agents.Usage, error) {
	// pre-spawn: 串行解析所有 worker;失败立即 abort
	workers := make([]agents.Agent, len(tasks))
	for i, task := range tasks {
		w, err := f.resolveWorker(i, task)
		if err != nil {
			return nil, agents.Usage{}, fmt.Errorf("fanout %q: task %q: %w", f.Name(), task.Name, err)
		}
		workers[i] = w
	}

	// 把 (task, worker) wrap 成 fanout.Task[TaskResult]
	fanoutTasks := make([]fanout.Task[TaskResult], len(tasks))
	for i, task := range tasks {
		task, worker := task, workers[i]
		fanoutTasks[i] = func(ctx context.Context) (TaskResult, error) {
			res, err := worker.Run(ctx, task.Input)
			if err != nil {
				return TaskResult{}, fmt.Errorf("fanout %q: worker %q for task %q: %w",
					f.Name(), worker.Name(), task.Name, err)
			}
			return TaskResult{Task: task, Result: res}, nil
		}
	}

	// 委托 fanout.Run + WithFailFast
	maxConcurrency := f.opts.MaxConcurrency
	if maxConcurrency <= 0 || maxConcurrency > len(tasks) {
		maxConcurrency = len(tasks)
	}
	results, ctxErr := fanout.Run(ctx, maxConcurrency, fanoutTasks, fanout.WithFailFast())
	if ctxErr != nil {
		// outer ctx 取消 —— 直接透传(顺手修原版边界 bug)
		return nil, agents.Usage{}, ctxErr
	}

	// 扫第一个非 Canceled 的 err(就是触发 fail-fast 的真 worker err)
	for _, r := range results {
		if r.Err != nil && !errors.Is(r.Err, context.Canceled) {
			return nil, agents.Usage{}, r.Err
		}
	}

	// 全部成功 → 装配 + 累加 usage
	out := make([]TaskResult, len(results))
	var usage agents.Usage
	for i, r := range results {
		out[i] = r.Value
		usage = addUsage(usage, r.Value.Result.Usage)
	}
	return out, usage, nil
}

func (f *FanOutFanIn) resolveWorker(index int, task PlannedTask) (agents.Agent, error) {
	if task.Worker != "" {
		worker, ok := f.opts.Workers[task.Worker]
		if !ok || worker == nil {
			return nil, fmt.Errorf("%w: %q", ErrFanOutUnknownWorker, task.Worker)
		}
		return worker, nil
	}
	if len(f.opts.Workers) == 1 {
		for _, worker := range f.opts.Workers {
			if worker == nil {
				return nil, ErrFanOutNilWorker
			}
			return worker, nil
		}
	}
	names := make([]string, 0, len(f.opts.Workers))
	for name := range f.opts.Workers {
		names = append(names, name)
	}
	sort.Strings(names)
	name := names[index%len(names)]
	worker := f.opts.Workers[name]
	if worker == nil {
		return nil, fmt.Errorf("%w: %q", ErrFanOutNilWorker, name)
	}
	return worker, nil
}

func buildAggregateInput(original string, plan agents.Result, results []TaskResult, custom AggregateInputBuilder) string {
	if custom != nil {
		return custom(original, plan, results)
	}
	var b strings.Builder
	b.WriteString("Original input:\n")
	b.WriteString(original)
	b.WriteString("\n\nPlan:\n")
	b.WriteString(plan.Answer)
	b.WriteString("\n\nWorker results:\n")
	for i, r := range results {
		fmt.Fprintf(&b, "%d. task=%s worker=%s\ninput=%s\nresult=%s\n\n",
			i+1, r.Task.Name, r.Task.Worker, r.Task.Input, r.Result.Answer)
	}
	return strings.TrimSpace(b.String())
}

func defaultPlanParser(plan agents.Result) ([]PlannedTask, error) {
	lines := strings.Split(strings.TrimSpace(plan.Answer), "\n")
	out := make([]PlannedTask, 0, len(lines))
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, PlannedTask{
			Name:  fmt.Sprintf("task-%d", i+1),
			Input: line,
		})
	}
	return out, nil
}

func addUsage(a, b agents.Usage) agents.Usage {
	return agents.Usage{
		LLMCalls: a.LLMCalls + b.LLMCalls,
		Tokens:   a.Tokens + b.Tokens,
	}
}

var (
	ErrFanOutNilPlanner    = errors.New("orchestrate: fanout requires non-nil planner")
	ErrFanOutNilAggregator = errors.New("orchestrate: fanout requires non-nil aggregator")
	ErrFanOutNoWorkers     = errors.New("orchestrate: fanout requires at least one worker")
	ErrFanOutNoTasks       = errors.New("orchestrate: fanout planner returned no tasks")
	ErrFanOutUnknownWorker = errors.New("orchestrate: fanout task references unknown worker")
	ErrFanOutNilWorker     = errors.New("orchestrate: fanout worker is nil")
)

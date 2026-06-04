[English](./SUPERVISOR.md) | [简体中文](./SUPERVISOR.zh-CN.md)

# Supervisor

`Supervisor` is the iterative multi-agent orchestration primitive in `orchestrate`.
It runs a planner/worker loop: the planner decides the next worker dispatch,
workers execute, the planner observes the result, and the cycle repeats until
the planner finishes or `MaxRounds` is reached.

## When to use it

Use `Supervisor` when one agent must coordinate specialists across multiple
rounds and the next step depends on prior worker output.

Typical fits:

- support triage with follow-up checks
- reviewer/revise loops
- research workflows that need incremental dispatch
- nested orchestration, where a supervisor itself becomes a worker

Use `FanOutFanIn` instead when the planner can split work once, workers run in
parallel, and a single aggregator merges the result.

## Core model

`SupervisorOptions` has five required parts:

- `Planner` decides what to do next.
- `Workers` is the dispatch table keyed by worker name.
- `MaxRounds` caps the planner/worker loop.
- `ParseDispatch` turns the planner's text into a `Dispatch`.
- `BuildAggregate` turns collected `WorkerResult` values into the final answer.

Each round follows this flow:

1. The planner receives the task plus prior worker results.
2. `ParseDispatch` converts the planner answer into a `Dispatch` or `nil`.
3. The selected worker runs on `Dispatch.Input`.
4. The result is stored and returned to the planner on the next round.
5. `BuildAggregate` produces the final answer.

`MaxRounds` is a graceful stop, not an error. When the cap is reached, the
supervisor returns the results collected so far.

## Minimal example

```go
sup := orchestrate.NewSupervisor("demo", orchestrate.SupervisorOptions{
    Planner: plannerAgent,
    Workers: map[string]agents.Agent{
        "alpha": alphaAgent,
        "beta":  betaAgent,
    },
    MaxRounds: 3,
    ParseDispatch: func(answer string) (*orchestrate.Dispatch, error) {
        // parse planner text into worker name + input
    },
    BuildAggregate: func(results []orchestrate.WorkerResult) (string, error) {
        // merge all worker outputs into the final answer
    },
})
```

## Example project

See [`examples/08-supervisor`](../examples/08-supervisor) for three deterministic
demos:

- basic planner + workers loop
- budget propagation with `budget.WithBudget`
- nested use inside `StateGraph`

Run it with:

```bash
cd examples && go run ./08-supervisor
```

## Related primitives

- `Pipeline` for fixed handoffs
- `FanOutFanIn` for one-shot parallel decomposition
- `StateGraph` for lower-level typed control flow


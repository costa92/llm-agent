[English](./README.md) | [简体中文](./README.zh-CN.md)

# 08 - Supervisor

A deterministic demo of `orchestrate.Supervisor`.

`Supervisor` is the iterative orchestration primitive for planner/worker
workflows. It is best when the planner needs to inspect previous worker output
before choosing the next dispatch.

## Run

```sh
cd examples && go run ./08-supervisor
```

## Demos

- `Basic` shows a planner coordinating two workers across two rounds.
- `Budget` shows budget propagation through planner and worker calls.
- `Compose` shows a `Supervisor` used as a `StateGraph` node.

## API shape

```go
sup := orchestrate.NewSupervisor("demo", orchestrate.SupervisorOptions{
    Planner:        plannerAgent,
    Workers:        map[string]agents.Agent{"alpha": alphaAgent},
    MaxRounds:      3,
    ParseDispatch:  parseDispatch,
    BuildAggregate: aggregateResults,
})
```

## Notes

- `MaxRounds` is the supervisor loop cap.
- `Budget.MaxCalls` is the total LLM-call cap across planner + workers.
- The canonical worker composition stack is `policy.Wrap(otelmodel.Wrap(provider))`; this example does not import `otelmodel`.
- Hitting `MaxRounds` is graceful: the supervisor aggregates the results collected so far.

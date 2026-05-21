# 08 - Supervisor

A deterministic demo of `orchestrate.Supervisor`.

## Run

```sh
cd examples && go run ./08-supervisor
```

## Demos

- `Basic` shows a planner coordinating two workers across two rounds.
- `Budget` shows the same surface under `budget.WithBudget`; the cap is `Budget.MaxCalls`, not rounds.
- `Compose` shows a `Supervisor` used as a `StateGraph` node.

## Notes

- `MaxRounds` is the supervisor loop cap.
- `Budget.MaxCalls` is the total LLM-call cap across planner + workers.
- The canonical worker composition stack is `policy.Wrap(otelmodel.Wrap(provider))`; this example does not import `otelmodel`.
- Hitting `MaxRounds` is graceful: the supervisor aggregates the results collected so far.

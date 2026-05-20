# 06 — Budget / cancellation context

A deterministic, network-free demo of `budget.WithBudget` wired through a
`SimpleAgent`. Exercises the three budget dimensions (`MaxCalls`,
`MaxTokens`, `MaxWall`) enforced by the chokepoint
`agent_chatmodel.go::generateFromPrompt`. Covers requirement CC-1
(Keystone K7's local cousin).

Run:

```bash
cd examples && go run ./06-budget
```

`examples/` is its own Go module (`replace … => ../`), so the demo picks
up local edits to the framework instantly.

## The Budget struct

```go
type Budget struct {
    MaxTokens int           // cap on cumulative resp.Usage.TotalTokens. 0 = no cap.
    MaxCalls  int           // cap on attempts (Q2: counts attempts, not successes). 0 = no cap.
    MaxWall   time.Duration // cap on wall-clock; WithBudget derives context.WithDeadline.
    MaxCost   float64       // cap on accumulated cost. v1.2: NOT charged (CostMapper deferred to v1.3).
}
```

## How enforcement happens

Every agent paradigm (Simple / ReAct / Reflection / PlanSolve /
FunctionCall) funnels through `agent_chatmodel.go::generateFromPrompt`.
That helper does two `Tracker.Charge` calls around the underlying
`model.Generate`:

1. **Pre-call** `Charge(Usage{Calls: 1})` — fires BEFORE the network.
   Returns `ErrCallsExceeded` and **short-circuits the LLM call** when
   `MaxCalls` would be exceeded. Per Q2, the chokepoint counts attempts:
   a denied charge consumes one call against the cap.
2. **Post-call** `Charge(Usage{Tokens: resp.Usage.TotalTokens})` — fires
   AFTER a successful Generate. When `MaxTokens` would be exceeded the
   chokepoint returns BOTH the valid response AND the sentinel
   (`ErrTokensExceeded`). The demo proves this by counting how many
   times the underlying model.Generate is invoked.

When no `Tracker` is attached (`budget.From(ctx)` returns `(nil, false)`),
both branches are no-ops — the load-bearing backwards-compat guarantee.

## Wall-clock + cancellation

`WithBudget` automatically composes a `context.WithDeadline` when
`MaxWall > 0`. Callers do **not** need to wrap their own deadline:

```go
ctx, _ := budget.WithBudget(context.Background(), budget.Budget{MaxWall: 50*time.Millisecond})
// ctx carries both the Tracker AND a deadline at now+50ms.
```

The demo wraps the scripted model with a `slowLLM` shim that sleeps 200 ms
while honoring `ctx.Done()`. The deadline fires before the response is
returned and the chokepoint surfaces the raw `context.DeadlineExceeded`
(zero new error surface for wall-clock — Decision 4).

## What's deferred to v1.3

- `budget.Wrap(inner) llm.ChatModel` decorator so non-agent callers
  (rag, context-compress, eval harnesses) can opt in without going
  through `generateFromPrompt`.
- `Estimator` interface — pre-call token denial (`MaxTokens` becomes a
  pre-call check using `Estimator.EstimateTokens(req)`).
- `CostMapper` / cost-table — pricing × tokens → `Usage.Cost` so
  `MaxCost` can actually be charged.

## Stdlib-only

The `budget` package depends only on `context`, `errors`, `fmt`, `sync`,
`sync/atomic`, `time` — the core repo's stdlib-only invariant is
preserved.

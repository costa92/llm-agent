# Phase 17 Research: Cost and latency observability

**Researched:** 2026-05-16
**Phase:** 17 — cost and latency observability
**Requirements:** RAG-OBS-01, RAG-OBS-02
**Repos:** `llm-agent-rag` (instrumentation), `llm-agent-otel` (`otelrag` emission)

## Current state (codebase scan)

`llm-agent-rag` records *structural* trace data but **no cost or latency**:

- `rag.Diagnostics` / `rag.Trace` (in `rag/system.go`) — chunk IDs, route
  policy, rerank scores. No durations, no token counts.
- `retrieve.Trace` — query variants, routing, fusion attribution. No timing.
- `rag.ImportTrace` (`rag/observer.go`) — already counts `EmbedCount`; no
  durations. Carried to the `OnImport` observer.
- `ingest.ImportResult` — `{Documents, Chunks int; ChunkIDs []string}`. The
  public return of `Import`; no metrics.
- `generate.Response` is `{Text string}` — **no token usage field**.
- `embed.Embedder` is `Embed(ctx, text) (Vector, error)` — no usage.
- `pack` already has a `TokenCounter` seam (`SimpleCounter` counts words +
  CJK runes) used for the context-window budget — reusable for estimation.
- No `time.Time` / `time.Duration` field exists anywhere in the SDK.

`llm-agent-otel`:

- `otelrag` wraps `*rag.System` via `Wrapper` (a span per `Import`/`Retrieve`/
  `Ask`) and also offers `Observer()` returning a `rag.Observer` of span
  events. **It emits spans only — zero metrics** (no counters, no histograms).
- `otelrag.Config` selects a `TracerProvider` only.
- `otelmetrics.Recorder` is the established metrics pattern in the repo:
  `Int64Counter` + `Int64Histogram` built from a `MeterProvider`, noop
  fallback. `otelrag` will mirror its shape but keep RAG-local metric names.
- `llm-agent-otel/go.mod` requires `llm-agent-rag v0.2.0` (no `replace`). The
  new instrumentation fields land in the `llm-agent-rag` working tree,
  **untagged** — see Decision 5 for how `otelrag` consumes them.

## What RAG-OBS-01 / RAG-OBS-02 ask for

- **RAG-OBS-01** — token counts, per-stage durations, and embedding/
  generation call counts recorded in `Trace`/`Diagnostics` for every import,
  retrieve, and ask flow.
- **RAG-OBS-02** — `otelrag` emits rate/error/duration (RED) plus cost
  metrics derived from those fields.

## Decision 1 — a new leaf package `obs` holds the metric types

A new stdlib-only package `llm-agent-rag/obs` owns the shared types so both
`rag` and `retrieve` can embed them without an import cycle (`rag` →
`retrieve` already; `obs` imports only `context`, `sync/atomic`, `time`):

```go
package obs

type StageTiming struct { Stage string; Duration time.Duration }
type CallCounts  struct { Embed, Generate int }
type TokenUsage  struct { PromptTokens, CompletionTokens, TotalTokens int; Estimated bool }
type Metrics struct {
    TotalDuration time.Duration
    Stages        []StageTiming   // per-stage wall-clock, in execution order
    Calls         CallCounts
    Tokens        TokenUsage
}
```

`Metrics` is attached at three sites: `rag.Diagnostics.Metrics` (ask flow),
`retrieve.Trace.Metrics` (retrieve flow), and **both** `ingest.ImportResult`
and `rag.ImportTrace` (import flow — see Decision 4).

## Decision 2 — per-stage durations measured at the flow layer

The flow layer already brackets every stage, so timing is non-invasive
`time.Now()` / `time.Since` around the existing calls — no stage-internal
changes:

- **Ask** (`rag/ask.go`): stages `retrieve`, `rerank`, `pack`, `generate`,
  plus `Metrics.TotalDuration` end-to-end.
- **Retrieve** (`rag/retrieve.go`): stages `preprocess`, `retrieve`.
- **Import** (`rag/import.go`): stages `embed` (the split+embed loop) and
  `upsert`.

## Decision 3 — call counts via a context-scoped counter

Embedding/generation calls in the retrieve and ask flows are *nested* (the
query embedding lives in `DenseRetriever`; MQE/HyDE generation lives in
`LLMExpansionPreprocessor`). Counting them needs a context-scoped counter:

- `obs.Counter` (atomic `Embed`/`Generate` tallies) + `obs.WithCounter(ctx,
  c)` / `obs.CounterFrom(ctx)` (nil-safe).
- `rag.New` wraps the embedder and model in thin `countingEmbedder` /
  `countingModel` decorators that increment whichever `Counter` is on the
  call's context. The wrapped instances are threaded into the **default**
  retriever/preprocessor wiring so nested calls are caught.
- Each flow snapshots the counter before/after and diffs — robust whether
  the flow is top-level or nested (ask → retrieve share one counter; the diff
  gives each its own count).
- The import loop calls `embedder.Embed` directly, so it counts inline (the
  existing `embedCount`) — no counter needed there.
- **Limitation:** counting covers the *default* wiring. A caller who supplies
  their own `opts.Retriever`/`opts.Preprocessor` holding an unwrapped
  embedder/model bypasses the decorators. Documented, not solved in v0.6.

## Decision 4 — token accounting: `generate.Usage`, estimate as fallback

`generate.Response` gains an optional `Usage` field:

```go
type Usage struct { PromptTokens, CompletionTokens, TotalTokens int }
type Response struct { Text string; Usage Usage }
```

Model adapters that know real token counts populate it; the bundled scripted
models leave it zero. The ask flow maps `resp.Usage` into
`obs.Metrics.Tokens`; when `resp.Usage` is zero it **estimates** via the
`pack.TokenCounter` (prompt text + answer text) and sets
`TokenUsage.Estimated = true`. No new dependency — `Usage` is a plain struct,
estimation reuses the existing `pack.SimpleCounter`.

`ImportResult` carries `Metrics` (not only `ImportTrace`) so the import
flow's metrics are available to any caller — including the `otelrag` wrapper,
which sees only the return value, not the observer payload.

## Decision 5 — `otelrag` emits RED + cost metrics; cross-repo coupling

`otelrag.Config` gains an optional `MeterProvider`; `Wrapper` gains four
RAG-local instruments built in `Wrap` (noop fallback, so `Wrap`'s signature
is unchanged and instrument-creation errors degrade gracefully):

- `rag.requests` — `Int64Counter` (Rate), tagged `rag.operation`.
- `rag.errors` — `Int64Counter` (Errors).
- `rag.operation.duration` — `Float64Histogram` (Duration), the wrapper's own
  wall-clock; also recorded per stage from `Metrics.Stages` tagged `rag.stage`.
- `rag.tokens` — `Int64Counter` (Cost), from `Diagnostics.Metrics.Tokens`,
  tagged `rag.token.kind` (`prompt`/`completion`).

RED (rate/error/duration) needs only the wrapper's own wall-clock — no
dependency on the new SDK fields. **Cost** metrics (`rag.tokens`, embed
counts) need `Diagnostics.Metrics` / `ImportResult.Metrics`, which exist only
in the untagged `llm-agent-rag` working tree.

**Cross-repo plan:** `otelrag` is developed and verified locally against the
working tree via a temporary `go.work` linking both modules (the sanctioned,
`.gitignore`d local-dev tool — no committed `replace`, per hard rule 3).
A CI-green committed `llm-agent-otel` build against the new fields **waits on
an `llm-agent-rag` re-tag at v0.6 close** — exactly the v0.5 pattern
(develop-with-worklink → tag `llm-agent-rag` → bump `otel`'s `require`). This
is recorded as explicit carry-forward debt.

## Slice breakdown

- **17-01** — `obs` package (`Metrics`/`StageTiming`/`CallCounts`/
  `TokenUsage` types + `Counter` + context helpers); `countingEmbedder`/
  `countingModel` decorators wired in `rag.New`; per-stage durations and call
  counts recorded into `rag.Diagnostics`, `retrieve.Trace`,
  `ingest.ImportResult`, and `rag.ImportTrace`. (RAG-OBS-01 — measurement)
- **17-02** — `generate.Usage` on `generate.Response`; ask flow maps real
  usage or estimates via `pack.TokenCounter`, populating
  `obs.Metrics.Tokens` with the `Estimated` flag. (RAG-OBS-01 — tokens)
- **17-03** — `otelrag` RED + cost metrics: `MeterProvider` in `Config`, four
  instruments on `Wrapper`, recorded in `Import`/`Retrieve`/`Ask`. Verified
  locally via a temporary `go.work`. (RAG-OBS-02)

## Risks / notes

- 17-02 changes the public `generate.Response` struct (additive — a new
  field). Existing callers and the scripted test models keep compiling;
  `Usage` is simply zero for them.
- 17-03 cannot reach a committed CI-green state until `llm-agent-rag` is
  re-tagged at v0.6 close. The slice is verified locally via `go.work`; the
  tag + `require` bump is carry-forward debt to the milestone close. This
  mirrors the v0.5 `llm-agent-rag v0.2.0` cut.
- `go.work`-based local verification needs the `llm-agent-rag` transitive
  module graph (pgx/pgvector) resolvable offline. Those are already in the
  local module cache from Phases 14-16 builds; if a `go mod`-graph step is
  network-blocked in the sandbox, 17-03 reports it honestly (the Phase 14
  live-Postgres precedent) and the build is confirmed at tag time.
- Call-count coverage is the default wiring only (Decision 3 limitation).
- 17-02 depends on 17-01 (`obs.Metrics.Tokens`); 17-03 depends on 17-01+02.

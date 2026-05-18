---
phase: 17-cost-and-latency-observability
plan: 03
type: execute
status: complete
completed: 2026-05-16
repo: llm-agent-otel
requirements: [RAG-OBS-02]
---

# Summary: 17-03 otelrag RED + cost metrics

## Objective

Deliver RAG-OBS-02: the `otelrag.Wrapper` emits RED metrics (request
counter, error counter, duration histogram) and cost metrics (token
counter) for every `Import`/`Retrieve`/`Ask`, derived from the `obs.Metrics`
the RAG SDK records (Phases 17-01/02).

## Delivered

- `otelrag.Config` gains `MeterProvider apimetric.MeterProvider`;
  `meterProvider()` falls back to `metric/noop` — metric emission degrades
  exactly as span emission does.
- `otelrag/metrics.go` (new):
  - RAG-local metric names: `rag.requests`, `rag.errors`,
    `rag.operation.duration`, `rag.tokens`.
  - Metric attribute keys: `rag.operation`, `rag.stage`, `rag.token.kind`,
    `rag.error`.
  - `instruments` (two `Int64Counter`, one `Float64Histogram`, one
    `Int64Counter`); `newInstruments` builds them with a per-instrument
    no-op fallback, so `Wrap` keeps its non-erroring `*Wrapper` signature.
  - `recordOp` — request count, error count on failure, op-level wall-clock
    duration (ms), and a per-stage duration for each `obs.StageTiming`.
  - `recordTokens` — prompt/completion token counts, tagged `rag.token.kind`.
- `Wrapper` gains an `instr` field built in `Wrap`. `Import`, `Retrieve`,
  and `Ask` each measure their own wall-clock and call `recordOp`; `Ask`
  also calls `recordTokens` from `Diagnostics.Metrics.Tokens` on success.
  Span behaviour is unchanged — metrics are emitted alongside.

## Files

- `otelrag/otelrag.go` — `time`/`apimetric` imports; `MeterProvider` on
  `Config`; `instr` on `Wrapper`; `Wrap` builds instruments;
  `Import`/`Retrieve`/`Ask` record metrics.
- `otelrag/metrics.go` — new: instruments + recording helpers.
- `otelrag/otelrag_test.go` — new metric tests: Ask emits requests +
  duration + tokens; an error emits `rag.errors`; a no-`MeterProvider`
  Wrap is no-op-safe.

## Verification

Verified locally via a temporary `go.work` linking `llm-agent-otel` and the
`llm-agent-rag` working tree (the sanctioned `.gitignore`d local-dev tool —
no committed `replace`):

- `go build ./otelrag/...` (GOWORK=workfile) — BUILD OK
- `go vet ./otelrag/...` (GOWORK=workfile) — VET OK
- `go test ./otelrag/... -count=1` (GOWORK=workfile) — `ok`
- `git diff --stat go.mod go.sum` — empty (`otelrag/go.mod` and `go.sum`
  untouched; no committed `replace`)
- workfile removed after verification

## Notes / carry-forward

- `otelrag` now references RAG-SDK fields (`Diagnostics.Metrics`,
  `ImportResult.Metrics`, `obs`) that exist only in the untagged
  `llm-agent-rag` working tree. A plain `GOWORK=off go build ./...` of
  `llm-agent-otel` is therefore **expected to fail against the pinned
  `llm-agent-rag v0.2.0`** — this is the documented mid-milestone state.
- **Carry-forward to v0.6 close:** tag `llm-agent-rag`, bump
  `otelrag/go.mod`'s `require github.com/costa92/llm-agent-rag` to that
  tag, and confirm `GOWORK=off go build ./...` is green. This mirrors the
  v0.5 `llm-agent-rag v0.2.0` cut (develop-with-worklink → tag → bump).
- `Retrieve` emits op-level RED only — the public `Retrieve` return carries
  no `obs.Metrics`; `Import` and `Ask` emit per-stage durations too.
- The module-graph step resolved offline — `llm-agent-rag`'s transitive
  deps were already in the local module cache from Phases 14-16 builds.

## Phase 17 status

All three slices complete. RAG-OBS-01 (17-01 durations + call counts; 17-02
token accounting) and RAG-OBS-02 (17-03 `otelrag` RED + cost metrics) are
delivered. `llm-agent-rag` gained no new dependency; `llm-agent-otel`'s
`go.mod`/`go.sum` are unchanged (the `require` bump is a tag-time step).

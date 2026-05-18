---
phase: 17-cost-and-latency-observability
plan: 01
type: execute
status: complete
completed: 2026-05-16
repo: llm-agent-rag
requirements: [RAG-OBS-01]
---

# Summary: 17-01 stage durations + call counts

## Objective

Add the measurement half of RAG-OBS-01 — per-stage wall-clock durations and
embedding/generation call counts recorded into the public surface of every
import, retrieve, and ask flow.

## Delivered

- `obs` package (new, leaf — imports only `context`/`sync/atomic`/`time`):
  - `Metrics{TotalDuration, Stages []StageTiming, Calls CallCounts, Tokens
    TokenUsage}`, `StageTiming{Stage, Duration}`, `CallCounts{Embed,
    Generate}`, `TokenUsage{Prompt/Completion/Total Tokens, Estimated}`.
  - `Counter` — concurrency-safe `atomic.Int64` embed/generate tallies;
    `NewCounter`, `AddEmbed`/`AddGenerate` (nil-safe), `Counts`.
  - `WithCounter`/`CounterFrom` context helpers (`CounterFrom` returns nil
    when absent; the result is safe to pass straight to `Add*`).
- `rag/instrument.go` — `countingEmbedder` / `countingModel` decorators that
  increment the `obs.Counter` on the call context per `Embed`/`Generate`.
- `rag.New` wraps the system embedder and model in the decorators (a nil
  model stays nil, so `Ask` still returns `ErrModelRequired`); the wrapped
  instances are threaded into the default `DenseRetriever` and
  `LLMExpansionPreprocessor` so nested calls are counted.
- `obs.Metrics` field added to `rag.Diagnostics`, `retrieve.Trace`,
  `rag.ImportTrace`, and `ingest.ImportResult`.
- `Ask` records stages `retrieve`/`rerank`/`pack`/`generate` (rerank and
  pack only when they run), `TotalDuration`, and `Calls` from a fresh
  context counter → `Diagnostics.Metrics`.
- `retrieve()` records stages `preprocess`/`retrieve`, `TotalDuration`, and
  `Calls` via a before/after counter diff (correct standalone or nested) →
  `retrieve.Trace.Metrics`.
- `Import` records stages `embed`/`upsert`, `TotalDuration`, and
  `Calls.Embed` (the existing inline count) → `ImportResult.Metrics` and
  `ImportTrace.Metrics`.

## Files

- `obs/obs.go`, `obs/obs_test.go` — new package + tests.
- `rag/instrument.go` — new: counting decorators.
- `rag/instrument_test.go` — new: ask/import/retrieve metric tests.
- `rag/system.go` — `obs` import; `Diagnostics.Metrics`; decorator wiring.
- `rag/ask.go`, `rag/retrieve.go`, `rag/import.go` — stage timing + counts.
- `rag/observer.go` — `ImportTrace.Metrics`.
- `retrieve/retrieve.go` — `Trace.Metrics`.
- `ingest/types.go` — `ImportResult.Metrics`.

## Verification

All `<verify>` commands run, all green:

- `GOWORK=off go build ./...` — BUILD OK
- `GOWORK=off go vet ./...` — VET OK
- `GOWORK=off go test ./obs ./rag ./retrieve ./ingest -count=1` — ok
- `GOWORK=off go test ./... -count=1` — all 17 packages ok
- `git diff --stat go.mod go.sum` — empty (no new dependency)
- core facade (from the core repo `llm-agent`): `GOWORK=off go vet ./rag/...
  && go test ./rag/...` — ok

## Notes

- `obs.Metrics.Tokens` is present but zero — populated by 17-02.
- Call counting covers the default wiring only: a caller-supplied
  `opts.Retriever`/`opts.Preprocessor` holds an unwrapped embedder/model and
  bypasses the decorators (documented limitation, RESEARCH Decision 3).
- The import `embed` stage brackets the whole split+embed loop (including
  `RemoveByFilter` for `ReplaceSource`) — the dominant cost is embedding.
- No new module dependency — entirely stdlib + existing seams.

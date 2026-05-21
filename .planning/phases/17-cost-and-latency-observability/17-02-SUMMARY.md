---
phase: 17-cost-and-latency-observability
plan: 02
type: execute
status: complete
completed: 2026-05-16
repo: llm-agent-rag
requirements: [RAG-OBS-01]
---

> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent-rag` current docs.


# Summary: 17-02 token accounting

## Objective

Complete RAG-OBS-01's token half — `generate.Response` carries optional
token usage, and the ask flow records token cost into `obs.Metrics.Tokens`,
estimating when the model does not report usage.

## Delivered

- `generate.Usage{PromptTokens, CompletionTokens, TotalTokens int}` and a
  `Usage` field on `generate.Response` — an additive change: existing
  callers and the bundled scripted models leave it zero.
- `rag/ask.go` `deriveTokenUsage(req, resp)`:
  - model-reported usage (any count > 0) → copied verbatim,
    `Estimated = false`; a zero `TotalTokens` is filled from the two parts.
  - no reported usage → estimated via `pack.SimpleCounter`: prompt tokens
    from the flattened request (`promptText` — system prompt + each message
    content), completion tokens from `resp.Text`, `TotalTokens` their sum,
    `Estimated = true`.
- `Ask` assigns the result to `Diagnostics.Metrics.Tokens`.

## Files

- `generate/types.go` — `Usage` struct + `Response.Usage` field.
- `rag/ask.go` — `generate`/`strings` imports; `deriveTokenUsage` +
  `promptText` helpers; `metrics.Tokens` populated before assembly.
- `rag/tokens_test.go` — new: `usageModel` stub; reported-usage test
  (exact, `Estimated=false`, total filled from parts) and absent-usage test
  (positive estimate, `Estimated=true`, total = prompt + completion).

## Verification

All `<verify>` commands run, all green:

- `GOWORK=off go build ./...` — BUILD OK
- `GOWORK=off go vet ./...` — VET OK
- `GOWORK=off go test ./rag ./generate -count=1` — ok
- `GOWORK=off go test ./... -count=1` — all 17 packages ok
- `git diff --stat go.mod go.sum` — empty (no new dependency)
- core facade (from the core repo `llm-agent`): `GOWORK=off go vet ./rag/...
  && go test ./rag/...` — ok

## Notes

- The `Response.Usage` change is additive — the scripted `fakeModel` and
  every existing caller compile and pass unchanged; `Usage` is simply zero
  for them and exercises the estimation path.
- Estimation reuses the existing `pack.SimpleCounter` (words×1.3 + CJK
  runes) — no new dependency.
- Embedding-token accounting is out of scope: the import flow's cost signal
  is its `Calls.Embed` count (17-01).
- `obs.Metrics.Tokens` is now fully populated; 17-03 (`otelrag`) consumes it
  for the `rag.tokens` cost metric.

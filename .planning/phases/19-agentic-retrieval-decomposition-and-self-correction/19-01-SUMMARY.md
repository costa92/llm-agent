---
phase: 19-agentic-retrieval-decomposition-and-self-correction
plan: 01
type: execute
status: complete
completed: 2026-05-17
repo: llm-agent-rag
requirements: [RAG-AGENT-01]
---

# Summary: 19-01 MultiHopRetriever + query decomposers

## Objective

Deliver RAG-AGENT-01 — a `MultiHopRetriever` decorator that decomposes a
compound query into sub-queries, retrieves for each through a wrapped
`Retriever`, and merges the sub-retrievals with per-hop attribution.

## Delivered

- `retrieve/multihop.go` (new):
  - `QueryDecomposer` interface (`Decompose(ctx, query) ([]string, error)`).
  - `HopAttribution{SubQuery, HitCount}`.
  - `HeuristicDecomposer` — splits on the conjunction "and"
    (case-insensitive regex), drops fragments < 3 runes, dedups; a query
    with no usable split decomposes to itself. No model, deterministic.
  - `LLMDecomposer{Model, MaxSubQueries}` — prompts a `generate.Model` for
    one sub-question per line; lenient parsing (strip list markers, drop
    empties, dedup); caps at `MaxSubQueries` (`<= 0` → 5); nil model or
    empty parse → `[query]`.
  - `MultiHopRetriever{Base, Decomposer}` implementing `retrieve.Retriever`:
    decomposes `req.Query`, runs `Base.Retrieve` per sub-query (sub-request
    copy, `QueryVariants` cleared), merges hits (dedup by `Chunk.ID`, max
    score, first-seen order), sorts by score and truncates to `req.TopK`.
    Nil `Base` → `ErrBaseRetrieverRequired`; nil `Decomposer` →
    `HeuristicDecomposer`.
- `retrieve.Trace.Hops []HopAttribution` — per-hop sub-query + hit count.

## Files

- `retrieve/multihop.go` — new.
- `retrieve/retrieve.go` — `Trace.Hops` field.
- `retrieve/multihop_test.go` — new: decomposer tests, merge/dedup/TopK,
  single-hop pass-through, nil-base error.

## Verification

All `<verify>` commands run, all green:

- `GOWORK=off go build ./...` — BUILD OK
- `GOWORK=off go vet ./...` — VET OK
- `GOWORK=off go test ./retrieve -count=1` — ok
- `GOWORK=off go test ./... -count=1` — all 17 packages ok
- `git diff --stat go.mod go.sum` — empty (no new dependency)
- core facade (from the core repo `llm-agent`): `GOWORK=off go vet ./rag/...
  && go test ./rag/...` — ok

## Notes

- `MultiHopRetriever` stays an opt-in `rag.Options.Retriever` decorator —
  the default retriever wiring is unchanged.
- A non-compound query decomposes to itself → exactly one `Base.Retrieve`
  call (verified by `TestMultiHopRetrieverSingleHopPassThrough`).
- `HeuristicDecomposer` is best-effort (may over-split "black and white");
  the `QueryDecomposer` seam lets callers supply `LLMDecomposer` or their
  own — tested with the existing `scriptedModel` stub.
- v0.6 multi-hop is parallel decomposition + merge, not sequential
  cross-hop reasoning (out of scope, noted in the plan).
- No new module dependency — `generate.Model` seam + stdlib only.

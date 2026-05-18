# Phase 15 Research: Model-based reranking and rerank explainability

**Researched:** 2026-05-15
**Phase:** 15 — model-based reranking and rerank explainability
**Requirements:** RAG-RERANK-01, RAG-RERANK-02

## Current state (codebase scan)

- `rerank` package: `Reranker` interface
  `Rerank(ctx, Request) ([]store.Hit, Trace, error)`. `Request{Query, Hits}`.
  `Trace{InputChunkIDs, OutputChunkIDs}` — only ID lists, no scores.
- Two implementations: `NoopReranker` (pass-through) and `HeuristicReranker`
  (adds `lexicalBoost` = 0.05 per matched query token over Title+Content+
  Heading+SectionPath, re-sorts).
- `rag.System` holds a `reranker rerank.Reranker`, defaulting to
  `HeuristicReranker{}`. `rag/ask.go` calls it when `Search.EnableRerank` and
  records only `Trace.RerankedChunkIDs` (the output ID order).
- `rag.Diagnostics` carries hit/section/route trace data but nothing about
  *why* the rerank reordered things.
- `contract/contract_test.go` pins `rag.Diagnostics` by name only — adding
  fields is additive and safe.

## Decision 1 — no new dependency

The gap analysis floated "an optional HTTP-client dep" for a model reranker.
That is unnecessary: a rerank API client is plain `net/http` + `encoding/json`.
Phase 15 stays **stdlib-only** for `llm-agent-rag` — no new `go.mod` entry, no
build tag. (The `postgres` subpackage remains the module's only non-stdlib
surface.)

## Decision 2 — `ScoringModel` seam, mirroring `generate.Model`

A model-based reranker depends on an abstract scoring seam, not a concrete
vendor:

```go
// rerank package
type ScoringModel interface {
    Score(ctx context.Context, query string, documents []string) ([]float64, error)
}
```

`ModelReranker{Model ScoringModel, TopN int}` implements `rerank.Reranker`:
build a document string per hit, call `Score`, set each hit's score to the
model score, sort descending, optionally truncate to `TopN`. `Model == nil`
→ `ErrScoringModelRequired`. This keeps `ModelReranker` itself deterministic
and unit-testable with a stub `ScoringModel` (the project's `ScriptedLLM`
discipline).

## Decision 3 — `HTTPScoringModel` concrete implementation

A concrete `ScoringModel` over `net/http` makes RAG-RERANK-01 real ("calls an
external cross-encoder/rerank model"). It POSTs JSON and reads a
Cohere/Jina/TEI-style response:

- request: `{"model":<name>,"query":<q>,"documents":[<doc>...]}`
- response: `{"results":[{"index":<i>,"relevance_score":<f>}...]}`

`index` maps each score back to document order (the API may return results
sorted by relevance). Config: endpoint URL, optional `Authorization` bearer
token, model name, `*http.Client` (nil → `http.DefaultClient`). Tested
deterministically against an `httptest.Server` — no live network.

## Decision 4 — rerank explainability via `Trace.Scores`

Extend `rerank.Trace` with per-hit detail:

```go
type RerankScore struct {
    ChunkID     string
    InputScore  float64
    OutputScore float64
    InputRank   int   // 1-based rank before rerank
    OutputRank  int   // 1-based rank after rerank
    RankDelta   int   // InputRank - OutputRank; positive = promoted
}
// Trace gains: Scores []RerankScore
```

All three rerankers populate `Scores`. `rag.Diagnostics` gains
`RerankScores []rerank.RerankScore`, populated in `rag/ask.go` from the
rerank trace, so a caller can audit every promotion/demotion.

## Slice breakdown

- **15-01** — rerank explainability: `rerank.RerankScore` + `Trace.Scores`,
  populated by `Noop`/`Heuristic` rerankers; surfaced through
  `rag.Diagnostics.RerankScores`. (RAG-RERANK-02)
- **15-02** — `rerank.ScoringModel` seam, `ModelReranker`, and
  `HTTPScoringModel` net/http client. (RAG-RERANK-01)

## Risks / notes

- `ModelReranker` must populate the 15-01 `Trace.Scores` too — hence 15-02
  depends on 15-01.
- Adding `RerankScores` to `Diagnostics` is additive; the `contract` gate is
  re-run in both slices.
- Live rerank-API verification is out of scope (no network in CI); the
  `httptest.Server` test covers the HTTP path deterministically.

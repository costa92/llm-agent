---
phase: 15-model-based-reranking-and-rerank-explainability
plan: 02
type: execute
status: complete
completed: 2026-05-15
repo: llm-agent-rag
requirements: [RAG-RERANK-01]
---

> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent-rag` current docs.

# Summary: 15-02 ModelReranker + HTTPScoringModel

## Objective

Add a model-based reranker behind the existing `rerank.Reranker` interface:
an abstract `ScoringModel` seam, a `ModelReranker` that reranks by model
score, and a concrete `HTTPScoringModel` that calls an external rerank API
over `net/http`.

## Delivered

- `rerank.ScoringModel` — abstract seam:
  `Score(ctx, query string, documents []string) ([]float64, error)`. Mirrors
  the `generate.Model` seam idiom; keeps `ModelReranker` testable without a
  network.
- `rerank.ErrScoringModelRequired` — returned by `ModelReranker` with a nil
  model.
- `rerank.ModelReranker{Model, TopN}` — implements `rerank.Reranker`. Builds
  a structured document per hit (`hitDocument`: title + content + heading +
  section path), calls `Score`, sets each hit's score to the model score,
  sorts descending (stable, tie-broken by original order), truncates to
  `TopN` when set, and populates `Trace.Scores` via the 15-01 `buildScores`
  helper. Errors on a nil model or a score-count mismatch.
- `rerank.HTTPScoringModel{Endpoint, Model, Token, Client}` — a `ScoringModel`
  over `net/http`. POSTs `{model, query, documents}` JSON, sends a bearer
  token when set, errors on empty endpoint or non-2xx, decodes a
  Cohere/Jina/TEI-style `{results:[{index, relevance_score}]}` response, and
  maps each result back to document order by `index`.
- `HeuristicReranker` refactored to share `hitDocument` (was an inline
  `structuredText` expression) — single definition of a chunk's rerank text.

## Files

- `rerank/rerank.go` — `ScoringModel`, `ErrScoringModelRequired`,
  `ModelReranker`, `hitDocument`; `errors`/`fmt` imports.
- `rerank/httpmodel.go` — new: `HTTPScoringModel` + JSON request/response
  types, `var _ ScoringModel = HTTPScoringModel{}` compile-time check.
- `rerank/rerank_test.go` — `reverseScorer` stub; `ModelReranker` reorder /
  `TopN` / nil-model tests.
- `rerank/httpmodel_test.go` — new: `HTTPScoringModel` against an
  `httptest.Server` (out-of-order results, empty endpoint, 500 response).

## Verification

All `<verify>` commands run, all green:

- `go build ./...` — BUILD OK
- `go vet ./...` — VET OK
- `go test ./rerank ./rag ./contract -count=1` — ok (contract gate passes)
- `go test ./... -count=1` — all 14 packages ok
- **no new dependency:** `git diff --stat go.mod go.sum` — empty; `go.mod`
  still lists only the `postgres` deps (`pgx/v5`, `pgvector-go`). The whole
  rerank model path is stdlib (`bytes`, `encoding/json`, `net/http`, ...).
- core: `GOWORK=off go vet ./rag/... && go test ./rag/...` — ok

## Notes

- The original `<verify>` block proposed `test ! -f go.sum` — wrong for this
  repo: `llm-agent-rag` *does* have a `go.sum` (the `postgres` subpackage's
  deps). The correct check, used here, is `git diff --stat go.mod go.sum`
  being empty — Phase 15 added no dependency.
- `ModelReranker` is **not** wired as the `rag.System` default; the default
  stays `HeuristicReranker` (no network). `ModelReranker` is opt-in through
  the pre-existing `rag.Options.Reranker`.
- `HTTPScoringModel` targets the common Cohere/Jina/TEI response shape;
  vendor-specific auth flows beyond a bearer token are out of scope.

## Phase 15 status

Both slices (15-01, 15-02) complete. RAG-RERANK-01 (model-based reranker +
HTTP rerank-API client) and RAG-RERANK-02 (per-hit rerank explainability in
the trace and `Ask` diagnostics) are delivered. Phase 15 is complete, with
no new module dependency.

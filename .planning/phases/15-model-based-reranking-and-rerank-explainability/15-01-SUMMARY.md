---
phase: 15-model-based-reranking-and-rerank-explainability
plan: 01
type: execute
status: complete
completed: 2026-05-15
repo: llm-agent-rag
requirements: [RAG-RERANK-02]
---

# Summary: 15-01 Rerank explainability

## Objective

Make rerank decisions auditable: expose each chunk's score and rank before
vs. after rerank, plus how far it moved, on `rerank.Trace` and on the `Ask`
answer's `Diagnostics`.

## Delivered

- `rerank.RerankScore{ChunkID, InputScore, OutputScore, InputRank,
  OutputRank, RankDelta}` — per-hit rerank detail. `RankDelta = InputRank -
  OutputRank` (positive = promoted). `InputRank 0` means the chunk was not in
  the rerank input.
- `rerank.Trace.Scores []RerankScore` — additive trace field.
- `rerank.buildScores(input, output)` — unexported helper that pairs each
  output hit with its pre-rerank score and 1-based rank.
- `NoopReranker` and `HeuristicReranker` both populate `Trace.Scores` via
  `buildScores`. For Noop every delta is 0 and in/out scores are equal.
- `rag.Diagnostics.RerankScores []rerank.RerankScore` — surfaces the detail
  on the `Ask` answer. `rag/ask.go` copies `rerankTrace.Scores` when rerank
  runs; leaves it nil when `EnableRerank` is false or no reranker is set.

## Files

- `rerank/rerank.go` — `RerankScore`, `Trace.Scores`, `buildScores`;
  `Noop`/`Heuristic` populate `Scores`.
- `rag/system.go` — `Diagnostics.RerankScores` field.
- `rag/ask.go` — `rerankScores` local, populated from the rerank trace, set
  on `Diagnostics`.
- `rerank/rerank_test.go` — `findScore` helper; tests for Noop zero-deltas
  and Heuristic promotion (rank delta + score change).
- `rag/system_test.go` — `TestAskPopulatesRerankScores` (populated when
  rerank on, nil when off).

## Verification

All `<verify>` commands run, all green:

- `go build ./...` — BUILD OK
- `go vet ./...` — VET OK
- `go test ./rerank ./rag ./contract -count=1` — ok (contract gate passes)
- `go test ./... -count=1` — all 14 packages ok
- core: `GOWORK=off go vet ./rag/... && go test ./rag/...` — ok

## Notes

- Adding `Diagnostics.RerankScores` is additive; the `contract` gate passes
  in both repos. The core `llm-agent/rag` facade re-exposes `Diagnostics`
  wholesale (`rag/tool.go` serializes it), so the new field flows through for
  free.
- `buildScores` is reused by `ModelReranker` in 15-02 — hence the dependency.

## Next slice

15-02 — `rerank.ScoringModel` seam, `ModelReranker`, and `HTTPScoringModel`
(`net/http` rerank-API client; no new dependency).

---
phase: 14-lexical-retrieval-and-principled-hybrid-fusion
plan: 01
type: execute
status: complete
completed: 2026-05-15
repo: llm-agent-rag
requirements: [RAG-RETR2-01]
---

# Summary: 14-01 Okapi BM25 in-memory lexical retriever

## Objective

Replace `LexicalRetriever`'s binary token-overlap scoring with Okapi BM25 for
the in-memory path, and add an optional `store.LexicalSearcher` capability so
backends that rank natively are used instead of an in-process scan.

## Delivered

- `store.LexicalSearcher` — optional capability interface
  (`LexicalSearch(ctx, Query) ([]Hit, error)`). Stores may implement it;
  retrieval type-asserts and falls back when absent.
- `store.Query.Text` — new field carrying the raw query string for lexical
  search. Vector search ignores it; this is the only change needed to thread
  query text through the existing `store.Query` shape (the interface itself
  is unchanged).
- `retrieve.BM25Params{K1, B}` — BM25 tuning constants; zero value resolves
  to the standard defaults (k1 1.2, b 0.75) via `orDefault`.
- `retrieve.bm25Scores` — ranks a corpus with Okapi BM25 using the
  non-negative IDF variant `ln(1 + (N-df+0.5)/(df+0.5))`. Corpus stats
  (N, df, avgdl) are computed over the supplied chunks.
- `LexicalRetriever` gained a `Params BM25Params` field. `Retrieve` now
  delegates to `store.LexicalSearcher` when the store implements it and no
  route path is set; otherwise it lists the namespace, applies route
  filtering, and ranks with BM25. Sort is deterministic (score desc,
  tie-break by chunk ID).

## Files

- `store/store.go` — added `Query.Text`, `LexicalSearcher` interface.
- `retrieve/retrieve.go` — added `math` import, `BM25Params`, `bm25Scores`;
  rewrote `LexicalRetriever`. `lexicalScore` is retained — `StructureRetriever`
  still uses it for leaf-chunk boosting.
- `retrieve/retrieve_test.go` — renamed `TestLexicalRetrieverUsesContentOverlap`
  → `TestLexicalRetrieverRanksByBM25`; added term-frequency, IDF,
  length-normalization, empty-query, and `LexicalSearcher`-delegation tests.

## Verification

All `<verify>` commands run, all green:

- `go build ./...` — BUILD OK
- `go vet ./...` — VET OK
- `go test ./retrieve ./store ./rag -count=1` — ok
- `go test ./... -count=1` — all 14 packages ok
- core: `GOWORK=off go vet ./rag/... && go test ./rag/...` — ok

## Notes

- **Deviation from plan:** the plan said "build a `store.Query` from `req`" but
  `store.Query` had no text field. Added `Query.Text` (additive, safe) rather
  than inventing a clunkier `LexicalSearch` signature. Noted here per the
  slice workflow.
- `LexicalSearcher` is delegated to only when `len(req.RoutePath) == 0`.
  Route-constrained lexical queries fall back to the in-process BM25 scan
  (which applies `chunkInRoute`). Route-path lexical on Postgres is an edge
  case; native route filtering can be added later if a consumer needs it.
- The in-memory BM25 recomputes corpus stats per query (O(corpus)). Acceptable
  for the in-memory store's intended scale; caching is deferred (out of v0.6).

## Next slice

14-02 — implement `store.LexicalSearcher` on `postgres.Store` via a
`tsvector` generated column + GIN index + `ts_rank_cd`.

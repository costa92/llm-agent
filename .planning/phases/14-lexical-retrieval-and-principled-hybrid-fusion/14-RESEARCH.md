# Phase 14 Research: Lexical retrieval and principled hybrid fusion

**Researched:** 2026-05-15
**Phase:** 14 — lexical retrieval and principled hybrid fusion
**Requirements:** RAG-RETR2-01, RAG-RETR2-02

## Current state (codebase scan)

- `retrieve.LexicalRetriever` (`retrieve/retrieve.go`) scores via
  `lexicalScore()` — binary token overlap: `+1` per distinct query token
  present in the chunk. No term frequency, no IDF, no length normalization.
  This is the gap RAG-RETR2-01 closes.
- `retrieve.HybridRetriever` **already** fuses dense + lexical + structure
  with Reciprocal Rank Fusion: `rrfScores[id] += 1/(rank + 1 + 60)`. The RRF
  constant `60` is hardcoded and the per-signal contributions are not
  surfaced anywhere — the merged `Trace` only carries legacy section fields.
  So RAG-RETR2-02's real gap is **attribution + configurability**, not
  "add RRF" (RRF is present).
- `store.Store.Search` is vector-only. `InMemoryStore` and `postgres.Store`
  both keep raw `content TEXT`. `postgres` schema has no `tsvector` column.
- `retrieve.Retriever` interface: `Retrieve(ctx, Request) ([]store.Hit, Trace, error)`.
  `store.Hit{Chunk, Score}`. Adding fields to `Trace` is additive/safe.

## Decision 1 — BM25 (Okapi) for in-memory lexical scoring

Score for query `Q` against document `D`:

```
score(D,Q) = Σ_{t∈Q} IDF(t) · ( tf(t,D)·(k1+1) ) / ( tf(t,D) + k1·(1 - b + b·|D|/avgdl) )
IDF(t)     = ln( 1 + (N - df(t) + 0.5) / (df(t) + 0.5) )
```

- `k1 = 1.2`, `b = 0.75` — standard defaults; expose as `BM25Params{K1,B}`,
  zero value resolves to defaults.
- The `ln(1 + …)` IDF variant is non-negative (avoids the classic BM25
  negative-IDF artifact on very common terms).
- Corpus stats (`N`, `df`, `avgdl`) are computed over the namespace's chunks
  obtained via `store.List`. For the in-memory store that is the full
  namespace — correct BM25, not an approximation. Per-query recompute is
  O(corpus); acceptable for the in-memory path (production scale uses the
  Postgres path below).
- Reuse the existing `tokenize()` helper so query and document tokenization
  stay consistent.

## Decision 2 — Postgres full-text search for the persistent path

Recomputing BM25 in Go over every Postgres row defeats the point of a
persistent backend. Instead, Postgres does the lexical ranking natively:

- Schema: a `STORED GENERATED` column
  `content_tsv tsvector GENERATED ALWAYS AS (to_tsvector(<cfg>, content)) STORED`
  plus a GIN index on it. `ALTER TABLE … ADD COLUMN IF NOT EXISTS` keeps
  `Migrate()` idempotent for existing tables.
- Query: `websearch_to_tsquery(<cfg>, $1)` (forgiving user-facing syntax)
  with `content_tsv @@ query`, ranked by `ts_rank_cd(content_tsv, query)`.
- Text-search config (`'english'`) is a `postgres.Config` field so non-English
  corpora can override it.

Postgres `ts_rank_cd` is not BM25 — it is a coverage-density rank. That is an
acceptable per-backend difference: the conformance contract is "lexical
retrieval returns relevance-ordered hits," not "identical scores across
backends" (mirrors how vector backends already differ in distance metrics).

## Decision 3 — optional `LexicalSearcher` capability interface

```go
// store package
type LexicalSearcher interface {
    LexicalSearch(ctx context.Context, q Query) ([]Hit, error)
}
```

`LexicalRetriever.Retrieve` type-asserts its `Store` to `LexicalSearcher`:
delegate when supported (Postgres), else fall back to in-memory BM25 over
`List`. This mirrors the established optional-capability pattern (e.g.
`RemoveByFilter`) and keeps `store.Store` itself unchanged.

## Decision 4 — fusion attribution in the trace

Add a `FusionAttribution{ChunkID, DenseRank, LexicalRank, StructureRank,
RRFScore}` and a `Trace.Fusion []FusionAttribution`. `HybridRetriever`
records each signal's rank as it applies RRF and emits the slice sorted by
final RRF score. The RRF constant becomes `HybridRetriever.RRFConstant`
(zero → 60). Rank `0` means "signal did not return this chunk."

## Slice breakdown

- **14-01** — Okapi BM25 in-memory lexical retriever + `LexicalSearcher`
  interface + fallback wiring. (RAG-RETR2-01)
- **14-02** — Postgres `tsvector`/`ts_rank_cd` lexical path implementing
  `LexicalSearcher`; env-gated test. (RAG-RETR2-01)
- **14-03** — configurable RRF constant + per-signal fusion attribution in
  `Trace`. (RAG-RETR2-02)

## Risks / notes

- Adding `Trace.Fusion` is additive; the Phase 13 `contract` gate pins the
  consumed surface — adding exported fields should not break it, but the
  contract test gets a verification check in 14-03.
- The in-memory BM25 per-query corpus scan is O(N); fine for the in-memory
  store's intended scale. No caching layer in v0.6 (caching is deferred).

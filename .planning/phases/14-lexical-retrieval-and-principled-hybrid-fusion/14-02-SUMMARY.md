---
phase: 14-lexical-retrieval-and-principled-hybrid-fusion
plan: 02
type: execute
status: complete
completed: 2026-05-15
repo: llm-agent-rag
requirements: [RAG-RETR2-01]
---

# Summary: 14-02 Postgres tsvector lexical path

## Objective

Implement `store.LexicalSearcher` on `postgres.Store` so the persistent
backend ranks keyword queries with PostgreSQL's native full-text engine
instead of an in-process scan.

## Delivered

- `postgres.Config.TextSearchConfig` — selects the PostgreSQL text-search
  configuration; empty resolves to `"english"` via the `textSearchConfig()`
  accessor. Validated against `isSafeIdent` in `New` since it is interpolated
  into DDL.
- `Migrate()` now adds, idempotently, a generated full-text column and its
  index: `content_tsv tsvector GENERATED ALWAYS AS (to_tsvector(<cfg>,
  content)) STORED` plus `CREATE INDEX IF NOT EXISTS … USING GIN (content_tsv)`.
  The explicit text-search config keeps `to_tsvector` immutable, which a
  generated column requires.
- `postgres.Store.LexicalSearch` — implements `store.LexicalSearcher`. Uses
  `websearch_to_tsquery(<cfg>, $text)` to parse the query, `content_tsv @@
  query` to match, and `ts_rank_cd` to rank; applies the same namespace +
  caller-filter + security-filter `WHERE` clause as `Search` via `buildWhere`.
  Empty/whitespace query → empty result, no error.
- Compile-time assertion `var _ store.LexicalSearcher = (*Store)(nil)`.
- `storetest.RunLexicalConformance` — opt-in lexical conformance suite that
  self-skips when the store does not implement `store.LexicalSearcher`. Five
  subtests: keyword match, term-frequency ranking, namespace isolation,
  security-filter trimming, empty query.
- `postgres_conformance_test.go` refactor: extracted `openTestPool` and a
  `newTableStore` factory helper; added `TestPostgresLexicalConformance`
  (env-gated on `LLM_AGENT_RAG_PG_URL`, same as the store conformance test).

## Files

- `postgres/postgres.go` — `Config.TextSearchConfig`, `textSearchConfig()`,
  `New` validation, `Migrate` DDL, `LexicalSearch`, compile-time check.
- `store/storetest/storetest.go` — `RunLexicalConformance`, `mustLexical`
  helper, five `testLexical*` subtests.
- `postgres/postgres_conformance_test.go` — `openTestPool` + `newTableStore`
  helpers, `TestPostgresLexicalConformance`.

## Verification

- `go build ./...` — BUILD OK
- `go vet ./...` — VET OK
- `go test ./... -count=1` — all 14 packages ok
- `go test ./postgres -v -run 'Lexical|Conformance'` — both env-gated tests
  **SKIP** cleanly (no `LLM_AGENT_RAG_PG_URL` set).

**Live-Postgres status:** no live Postgres is reachable in this environment,
so the `tsvector`/`ts_rank_cd` SQL path and `RunLexicalConformance` subtests
were **not executed against a real database**. They compile, vet clean, and
skip correctly. Live verification depends on the still-pending Live-Postgres
CI wiring (carried-forward v0.5 debt) — flagged honestly here rather than
claimed as passing.

## Notes

- `LexicalSearch` reuses `scanChunkWithDistance` — its trailing float64 column
  is the `ts_rank_cd` score here rather than a cosine distance. Functionally
  correct (same 11 columns + one trailing float); the method comment notes it.
- `TextSearchConfig` is restricted to a single safe identifier (no
  schema-qualified `pg_catalog.english` form). Sufficient for v0.6.
- Route-path-constrained lexical queries still fall back to in-memory BM25
  (decided in 14-01); `LexicalSearch` itself is namespace + filter scoped only.

## Next slice

14-03 — configurable RRF constant + per-signal fusion attribution in the
retrieval `Trace`.

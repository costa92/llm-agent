---
phase: 21-graph-storage
plan: 02
type: execute
status: complete
completed: 2026-05-18
repo: llm-agent-rag
requirements: [RAG-GRAPH-03]
---

> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent-rag` current docs.


# Summary: 21-02 postgres GraphStore

## Objective

Complete RAG-GRAPH-03's storage half: `postgres.Store` implements
`store.GraphStore` over `entities`/`relations` tables with recursive-CTE
traversal — no graph database — passing the same `RunGraphConformance`
suite as the in-memory store.

## Delivered

- `postgres.Store.Migrate` extended with two idempotent tables —
  `<table>_entities` and `<table>_relations` (namespaced, provenance as
  `text[]`, `(namespace, id)` primary keys) — plus traversal indexes on
  `(namespace, source)` / `(namespace, target)` and a
  `(namespace, lower(name))` index for `FindEntities`.
- `postgres/graph.go` — `*Store` implements `store.GraphStore`:
  - `UpsertGraph` — `INSERT ... ON CONFLICT (namespace, id) DO UPDATE`
    union-merging `source_chunk_ids` (array `UNION`) and summing relation
    `weight`; entities and relations in one transaction.
  - `RemoveGraphBySource` — removes chunk IDs from `source_chunk_ids`
    (`EXCEPT`), then `DELETE`s rows whose provenance became empty (GC).
  - `Neighborhood` — a `WITH RECURSIVE` CTE walking `relations` both
    directions from the seeds, bounded by an explicit depth predicate
    (`depth` clamped to `[0, 2]` — KG-7) and a row `LIMIT`
    (`maxNeighborhoodRows`); returns the reached `graph.Subgraph`.
  - `FindEntities` — `WHERE lower(name) = ANY($2)`.
  - compile-time `var _ store.GraphStore = (*Store)(nil)`.
- `TestPostgresGraphConformance` — env-gated, runs the shared
  `RunGraphConformance` suite; `newTableStore` cleanup extended to drop
  the `_entities`/`_relations` tables too.

## Files

- `postgres/postgres.go` — `Migrate` graph DDL.
- `postgres/graph.go` — new: the `GraphStore` implementation.
- `postgres/postgres_conformance_test.go` — `TestPostgresGraphConformance`;
  graph-table cleanup.

## Verification

All `<verify>` commands run, all green:

- `GOWORK=off go build ./...` — BUILD OK
- `GOWORK=off go vet ./...` — VET OK
- `GOWORK=off go test ./postgres ./store/... -count=1` — ok;
  `TestPostgresGraphConformance` SKIPs cleanly without `LLM_AGENT_RAG_PG_URL`
- `GOWORK=off go test ./... -count=1` — all packages ok (no FAIL)
- `git diff --stat go.mod go.sum` — empty (`pgx`/`pgvector` already present)
- core facade (from the core repo `llm-agent`): `GOWORK=off go vet
  ./rag/... && go test ./rag/...` — ok

## Notes — live-Postgres limitation (carried debt)

The `postgres` `GraphStore` (the new tables, the `ON CONFLICT` upserts, the
recursive-CTE `Neighborhood`) **compiles and vets clean and the conformance
test skips cleanly, but it was NOT run against a live database** — no
Postgres is available in this environment. The SQL is syntactically
reviewed but not runtime-verified. This is the same carried-forward v0.5
debt as the Phase 14 `tsvector` path: live-Postgres CI wiring
(testcontainers-go or GH Actions services) remains pending. Reported
honestly — `TestPostgresGraphConformance` will exercise it once
`LLM_AGENT_RAG_PG_URL` points at a real database.

- Traversal is bounded in SQL (KG-7): the depth predicate (`w.depth < $3`,
  `$3 ≤ 2`) plus the final-select row `LIMIT`. The strict per-hop fan-out
  cap is the in-memory store's; the SQL path approximates it via the depth
  bound + total `LIMIT`.
- No new module dependency.

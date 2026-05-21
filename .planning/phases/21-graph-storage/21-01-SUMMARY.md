---
phase: 21-graph-storage
plan: 01
type: execute
status: complete
completed: 2026-05-18
repo: llm-agent-rag
requirements: [RAG-GRAPH-03]
---

> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent-rag` current docs.


# Summary: 21-01 GraphStore interface + in-memory implementation

## Objective

Open Phase 21 with the graph-storage seam — `store.GraphStore` optional
capability, `graph.Subgraph`, a pure-stdlib in-memory implementation, and a
shared `RunGraphConformance` suite. First half of RAG-GRAPH-03.

## Delivered

- `graph.Subgraph{Entities, Relations, Depth map[string]int}` — the
  neighborhood-traversal result; `graph.NormalizeName` exported (the
  shared canonical-name normalization, now reused by `store`).
- `store.GraphStore` interface (next to `LexicalSearcher`) — `UpsertGraph`,
  `RemoveGraphBySource`, `Neighborhood`, `FindEntities`; `store` imports
  `graph` (no cycle — `graph` imports neither `store` nor `ingest`).
- `store/graph.go` — the in-memory `GraphStore` on `*InMemoryStore`:
  per-namespace adjacency state (`nsGraph`), guarded by the store's
  existing `RWMutex`.
  - `UpsertGraph` — union-merge: entities/relations merge by ID, unioning
    provenance and deduping descriptions.
  - `RemoveGraphBySource` — drops chunk IDs from provenance, GCs any
    entity/relation left unreferenced.
  - `Neighborhood` — BFS from seeds, `depth` clamped to `[0, 2]` (KG-7),
    each hop fan-out-capped at `maxGraphFanout` (64); deterministically
    ordered output; returns a `graph.Subgraph`.
  - `FindEntities` — case-folded exact name match.
  - compile-time `var _ store.GraphStore = (*InMemoryStore)(nil)`.
- `store/storetest.RunGraphConformance` — 5-subtest shared suite
  (upsert+neighborhood, depth hard-bound, union-merge, remove+GC,
  find-by-name); skips cleanly when a store does not implement
  `GraphStore` (the `RunLexicalConformance` contract).

## Files

- `graph/graph.go` — `Subgraph` type.
- `graph/canonicalize.go` — `normalizeName` → exported `NormalizeName`.
- `store/store.go` — `graph` import; `GraphStore` interface.
- `store/inmemory.go` — `graphs` field on `InMemoryStore` + `New` init.
- `store/graph.go` — new: in-memory `GraphStore` + merge/GC helpers.
- `store/storetest/storetest.go` — `graph` import; `RunGraphConformance`
  + subtests + `chainGraph`/`graphStore` helpers.
- `store/graph_test.go` — new: `RunGraphConformance` for the in-memory store.

## Verification

All `<verify>` commands run, all green:

- `GOWORK=off go build ./...` — BUILD OK
- `GOWORK=off go vet ./...` — VET OK
- `GOWORK=off go test ./store/... ./graph -count=1` — ok
- `GOWORK=off go test ./... -count=1` — all packages ok (no FAIL)
- `git diff --stat go.mod go.sum` — empty (no new dependency)
- core facade (from the core repo `llm-agent`): `GOWORK=off go vet
  ./rag/... && go test ./rag/...` — ok

## Notes

- Traversal bounds (KG-7) are enforced in the in-memory BFS: `depth`
  clamped to ≤ 2, per-hop fan-out capped at 64. 21-02's recursive CTE
  enforces the same in SQL.
- `store` now imports `graph` — confirmed acyclic.
- No new module dependency — the in-memory `GraphStore` and `graph.Subgraph`
  are pure stdlib.

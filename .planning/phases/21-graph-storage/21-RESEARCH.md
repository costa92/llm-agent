# Phase 21 Research: Graph storage

**Researched:** 2026-05-18
**Phase:** 21 — graph storage
**Requirements:** RAG-GRAPH-03, RAG-GRAPH-04
**Repos:** `llm-agent-rag`
**Upstream:** `.planning/research/v0.7-graphrag-SUMMARY.md` §5 (the storage
keystone) and §6 (seams); `20-RESEARCH.md` (the `graph` package).

## Current state (codebase scan)

- `store.LexicalSearcher` (`store/store.go:34`) is the **optional-capability
  precedent**: an interface a `Store` *may* implement; retrieval
  type-asserts for it and falls back gracefully. `GraphStore` copies this
  pattern bit-for-bit.
- `store.Store` itself is unchanged by GraphRAG — graph data lives behind a
  new capability interface, not bolted onto `Store`.
- `store/storetest` — the shared conformance suite: a `Factory func(t)
  store.Store`, `RunConformance` / `RunLexicalConformance` run `t.Run`
  subtests against it. Phase 21 adds `RunGraphConformance`.
- `postgres.Store` — `Migrate(ctx)` runs idempotent DDL; `isSafeIdent`
  guards configurable identifiers; `LexicalSearch` + a compile-time
  `var _ store.LexicalSearcher = (*Store)(nil)` assertion. The graph tables
  go in the same `Migrate()`.
- `graph` package (Phase 20) — `Entity`, `Relation`, `Graph`,
  `Canonicalize`. `graph` imports only stdlib + `generate`; it does **not**
  import `store`, so `store` can import `graph` with no cycle.
- `rag/import.go` `Import` (Phase 20) produces the canonicalized graph on
  `ImportResult.Graph` but does **not** persist it — Phase 21 gives it a
  persistence target and wires the call.

## Decision 1 — `store.GraphStore` optional capability (RAG-GRAPH-03)

A new interface in the `store` package, next to `LexicalSearcher`
(`store` imports `graph` — no cycle):

```go
type GraphStore interface {
    UpsertGraph(ctx context.Context, namespace string, g graph.Graph) error
    RemoveGraphBySource(ctx context.Context, namespace string, chunkIDs []string) error
    Neighborhood(ctx context.Context, namespace string, seedIDs []string, depth int) (graph.Subgraph, error)
    FindEntities(ctx context.Context, namespace string, names []string) ([]graph.Entity, error)
}
```

`UpsertGraph` takes a `graph.Graph` (Phase 20's canonicalized output).
The graph is **per-namespace** (matches `store` namespacing). Consumers
(the Phase 22 `GraphRetriever`) type-assert a `store.Store` for
`GraphStore` and degrade gracefully when absent — the `LexicalSearcher`
contract.

## Decision 2 — `graph.Subgraph`, the `Neighborhood` result

`Neighborhood` must return enough for Phase 22's proximity-decay scoring —
the reached entities, the relations among them, and each entity's hop
distance from the seeds. Add to the `graph` package:

```go
type Subgraph struct {
    Entities  []Entity
    Relations []Relation
    Depth     map[string]int // entity ID -> hops from nearest seed (seeds = 0)
}
```

## Decision 3 — in-memory `GraphStore` (21-01)

The default `store.InMemoryStore` gains a `GraphStore` implementation: a
pure-stdlib adjacency structure (`map[namespace]` → `map[entityID]Entity`
+ adjacency `map[entityID][]Relation`). `Neighborhood` is a breadth-first
walk. **Traversal is hard-bounded (KG-7):** `depth` is clamped to `[0, 2]`,
and each hop expands at most a fixed `maxFanout` neighbors per entity
(a documented constant). Zero dependencies; trivially deterministic.

## Decision 4 — `postgres` `GraphStore` (21-02)

`postgres.Store` gains a `GraphStore` implementation: two tables —
`entities` and `relations` — created in the existing idempotent
`Migrate()` (provenance stored as a text[]/jsonb column). `Neighborhood`
uses a `WITH RECURSIVE` CTE bounded by an explicit depth predicate and a
row `LIMIT` (the fan-out/explosion guard — KG-7). Compile-time
`var _ store.GraphStore = (*Store)(nil)`. A shared `RunGraphConformance`
suite (in `store/storetest`) runs against **both** the in-memory store and
postgres — mirroring `RunLexicalConformance`.

## Decision 5 — merge + GC semantics (RAG-GRAPH-04)

- `UpsertGraph` is **union-merge**, not replace: entities merge by `ID`
  (descriptions concatenated-deduped, `SourceChunkIDs` unioned); relations
  merge by `ID`. Importing new documents grows the namespace graph.
- `RemoveGraphBySource(chunkIDs)` removes those chunk IDs from every
  entity/relation's provenance; an entity or relation left with **empty
  provenance** is garbage-collected; one still referenced by other chunks
  survives. This is the LightRAG union/GC model (KG-5).

## Decision 6 — `Import` wiring (21-03)

`rag/import.go` `Import`, after `Canonicalize` (Phase 20), type-asserts
`s.store` for `store.GraphStore`. When the store implements it and an
`EntityExtractor` is configured:
- on a `ReplaceSource` re-ingest, call `RemoveGraphBySource` for the
  replaced document's chunk IDs **before** `UpsertGraph` — so the graph
  reconciles (old contributions removed, new subgraph unioned) rather than
  just appending;
- call `UpsertGraph` with the canonicalized graph.
A store that does not implement `GraphStore` → graph is produced on
`ImportResult.Graph` but not persisted (graceful degradation). End-to-end
re-ingest reconciliation is tested on the in-memory store; the postgres
path is env-gated (live DB).

## Slice breakdown

- **21-01** — `store.GraphStore` interface + `graph.Subgraph`; the
  in-memory `GraphStore` implementation (adjacency, bounded `Neighborhood`,
  merge + GC); `store/storetest.RunGraphConformance` run against the
  in-memory store. (RAG-GRAPH-03)
- **21-02** — `postgres` `GraphStore`: `entities`/`relations` tables in
  `Migrate()`, recursive-CTE `Neighborhood`; `RunGraphConformance` wired
  for postgres (env-gated). (RAG-GRAPH-03)
- **21-03** — `Import` persists the graph via `UpsertGraph`; `ReplaceSource`
  re-ingest calls `RemoveGraphBySource` first; end-to-end reconciliation
  verified (in-memory; postgres env-gated). (RAG-GRAPH-04)

## Risks / notes

- `store` importing `graph` is safe — `graph` imports neither `store` nor
  `ingest` (confirmed: Phase 20's `graph` imports only stdlib + `generate`).
- Live-Postgres limitation persists from v0.5/Phase 14: the `postgres`
  graph path (tables, recursive CTE, conformance) compiles and is
  env-gated but is not run against a real database in this environment —
  reported honestly, carried debt.
- Traversal bounds (KG-7) are enforced in *both* impls — the in-memory BFS
  and the recursive CTE — not just one.
- 21-02 and 21-03 depend on 21-01 (the interface + `Subgraph`); 21-02 and
  21-03 are otherwise independent.
- No new module dependency — `pgx`/`pgvector` already present for
  `postgres`; the in-memory impl and `graph.Subgraph` are stdlib.

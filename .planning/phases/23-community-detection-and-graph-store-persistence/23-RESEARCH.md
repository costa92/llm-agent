# Phase 23 Research: Community detection and graph-store persistence

**Researched:** 2026-05-19
**Phase:** 23 — community detection & graph-store persistence (first v0.8 phase)
**Requirements:** RAG-GRAPH3-01, RAG-GRAPH3-02
**Repos:** `llm-agent-rag`
**Upstream:** `.planning/research/v0.8-graphrag-tier3-SUMMARY.md` §3 (community
detection) and §1 (codebase scan); v0.7 `graph` package at tag `v0.4.0`.

## Current state (codebase scan, `/tmp/llm-agent-rag` @ v0.4.0)

- `graph.Graph{Entities []Entity, Relations []Relation}` — the canonicalized
  graph. `Entity{ID, Name, Type, Description, SourceChunkIDs, Metadata}`,
  `Relation{ID, Source, Target, Relation, Description, SourceChunkIDs,
  Weight}`. After `Canonicalize`, `Relation.Source`/`Target` hold canonical
  entity IDs. `graph` is a leaf package — stdlib + `generate` seam only.
- `store.GraphStore` — `UpsertGraph` / `RemoveGraphBySource` / `Neighborhood`
  / `FindEntities`. Per-namespace. **No whole-graph read method** — bounded
  `Neighborhood` only. Community detection needs the *whole* namespace
  graph, so a snapshot read is required (Decision 3).
- in-memory `GraphStore` — `store/graph.go`, `nsGraph` adjacency, deterministic
  ordering throughout. postgres `GraphStore` — `postgres/graph.go`,
  `_entities`/`_relations` tables, `Migrate()` idempotent DDL.
- `storetest` — `RunConformance` / `RunLexicalConformance` /
  `RunGraphConformance`, `Factory func(t) store.Store`.
- `rag.Import` (`rag/import.go`) — extracts per-chunk, `graph.Canonicalize`,
  persists via `GraphStore.UpsertGraph`; on `ReplaceSource` re-ingest it
  `RemoveGraphBySource` first. `Options.EntityExtractor`,
  `ImportResult.Graph *graph.Graph`.

## Decision 1 — `Community` type + `CommunityDetector` seam (in `graph`)

The `graph` package is the right home — detection is a pure computation over
a `graph.Graph`, no store, no LLM, no embedder, stdlib only.

```go
package graph

// Community is one cluster in the hierarchy. Level 0 is the finest;
// higher levels group lower-level communities. IDs are deterministic.
type Community struct {
    ID          string
    Level       int
    ParentID    string   // "" at the top level
    EntityIDs   []string // member entity IDs, sorted
    RelationIDs []string // member relation IDs, sorted (level 0)
}

// CommunityDetector partitions a Graph into a community hierarchy.
// Implementations are deterministic — same graph, same hierarchy.
type CommunityDetector interface {
    Detect(ctx context.Context, g Graph) ([]Community, error)
}
```

`graph.Graph` gains an optional `Communities []Community` field so a detected
structure can travel with the graph (additive — v0.7 zero-value behavior
unchanged).

## Decision 2 — stdlib Louvain, deterministic; LabelPropagation alternative

Per v0.8 keystone KG3-3 / KG3-6. **Louvain** is the recommended default
(`LouvainDetector{Resolution float64}`): modularity-gain greedy moves +
graph coarsening for the hierarchy (each coarsening pass = one level).
`LabelPropagationDetector{}` is the faster, simpler alternative behind the
same seam — mirroring v0.7's `LLMEntityExtractor` / `DictionaryEntityExtractor`
dual-mode pair.

**Determinism is mandatory and the whole testability story:**
- iterate entities in sorted `ID` order;
- break modularity-gain ties by lowest community ID;
- no randomness, no random restarts;
- community IDs are a deterministic function of `level` + sorted members
  (e.g. `L{level}-{firstMemberID}`), so a given graph always yields the same
  hierarchy — golden-output unit tests, exactly like `Canonicalize`.

Edge weights: `Relation.Weight` (default to 1.0 when zero). Undirected for
modularity (sum weights both directions). Leiden is **not** in scope — a
documented future swap behind the seam.

## Decision 3 — capability: a sibling `CommunityStore`, not a `GraphStore` change

v0.8 keystone KG3-5 left "extend `GraphStore`" vs "sibling capability" as a
Phase-23 call. **Decision: a sibling `store.CommunityStore`.** Rationale:
- the v0.7 `GraphStore` interface stays **byte-identical** — no risk to the
  shipped contract;
- detection needs a whole-graph snapshot, which `GraphStore` does not
  expose; bundling the snapshot read with community persistence in one new
  capability keeps each slice independently compilable (23-02 adds the
  interface + in-memory impl with no postgres edit; 23-03 adds postgres);
- a store could implement graph traversal without communities — the sibling
  keeps that possible.

```go
package store

// CommunityStore is an optional capability for community detection and
// persistence. Type-asserted and degraded gracefully, like GraphStore.
type CommunityStore interface {
    // GraphSnapshot returns the full stored graph for a namespace — the
    // input to community detection.
    GraphSnapshot(ctx context.Context, namespace string) (graph.Graph, error)
    // UpsertCommunities replaces the namespace's community set.
    UpsertCommunities(ctx context.Context, namespace string, communities []graph.Community) error
    // Communities returns the namespace's stored community set.
    Communities(ctx context.Context, namespace string) ([]graph.Community, error)
}
```

`UpsertCommunities` is replace-all (detection always produces the full set
for a namespace) — that is also how re-detection on re-ingest reconciles.

## Decision 4 — `Import` wiring + re-detection on re-ingest (KG3-7)

After the existing graph-persist block in `rag.Import`: if the store is a
`store.CommunityStore` **and** `Options.CommunityDetector` is set —
`GraphSnapshot(ns)` → `Detect` → `UpsertCommunities(ns, …)`. Because this
reads the snapshot *after* `UpsertGraph` (and after `RemoveGraphBySource` on
a `ReplaceSource` re-ingest), re-detection is automatic — every import that
changes the namespace graph re-detects the whole namespace. Full
re-detection is acceptable for v0.8 (cheap stdlib); incremental maintenance
is deferred. The detected set is attached to `res.Graph.Communities` and its
count surfaces on `ImportResult`.

## Slice breakdown

- **23-01** — `graph` package: `Community` type, `Graph.Communities` field,
  `CommunityDetector` seam; deterministic stdlib `LouvainDetector` +
  `LabelPropagationDetector`; golden-output unit tests. Pure `graph`, no
  store. (RAG-GRAPH3-01)
- **23-02** — `store.CommunityStore` capability; pure-stdlib in-memory
  implementation on `InMemoryStore` (`GraphSnapshot`, `UpsertCommunities`,
  `Communities`); `storetest.RunCommunityConformance` shared suite.
  (RAG-GRAPH3-02)
- **23-03** — postgres `CommunityStore` (`_communities` table in `Migrate()`,
  `GraphSnapshot` over `_entities`/`_relations`); `Import` wiring
  (`Options.CommunityDetector`, post-persist detect, `res.Graph.Communities`),
  re-detection on `ReplaceSource` re-ingest. (RAG-GRAPH3-02)

## Risks / notes

- Louvain determinism is the main risk — any unordered map iteration leaks
  non-determinism into the hierarchy. Every node/community loop sorts first;
  golden tests pin the output.
- `GraphSnapshot` over postgres is a full `SELECT` of the namespace's
  entities+relations — fine at this SDK's scale; the same env-gated,
  not-live-verified caveat as the rest of the postgres path.
- 23-02 depends on 23-01; 23-03 depends on 23-01+02.
- No new module dependency — detection is stdlib, the capability reuses
  `graph` types.

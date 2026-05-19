# Phase 26 Research: Path-ranking and subgraph-as-evidence

**Researched:** 2026-05-20
**Phase:** 26 — path-ranking & subgraph-as-evidence (first v0.9 phase)
**Requirements:** RAG-GRAPH4-01, RAG-GRAPH4-02
**Repos:** `llm-agent-rag`
**Upstream:** `.planning/research/v0.9-graphrag-refinements-SUMMARY.md` §5;
v0.7 `retrieve.GraphRetriever`; the `graph` package at tag `v0.5.0`.

## Current state (codebase scan, `/tmp/llm-agent-rag` @ v0.5.0)

- `graph.Subgraph{Entities []Entity, Relations []Relation, Depth map[string]int}`
  — the neighborhood-traversal result; `Depth` is each entity's hop from the
  nearest seed. **This is the subgraph-as-evidence type — it already exists.**
- `graph.Relation{ID, Source, Target, Relation, Description, SourceChunkIDs,
  Weight}` — directed; after `Canonicalize`, `Source`/`Target` are canonical
  entity IDs. `graph/community.go` has an unexported `edgeWeight(r Relation)
  float64` helper (`Weight` if > 0, else 1.0) — reusable in-package.
- `retrieve.GraphRetriever{Linker, Store, MaxDepth, HopDecay}` —
  `Retrieve` links the query → seed entities, `gs.Neighborhood(ns, seedIDs,
  depth)` → `graph.Subgraph` `sub`, maps provenance chunks to best hop,
  emits proximity-decay `store.Hit`s + a `GraphTrace`.
- `retrieve.GraphTrace{SeedEntityIDs, ReachedEntityIDs, MaxHop, CommunityIDs}`
  — `CommunityIDs` (v0.8) is the precedent for adding fields additively.
  `rag.Diagnostics.GraphTrace` is the `retrieve.GraphTrace` value — new
  `GraphTrace` fields surface through `Diagnostics` automatically.
- `store/graph.go` — traversal is hard-bounded: `maxGraphDepth = 2`,
  `maxGraphFanout = 64`. `graph` cannot import `store` (cycle) — the path
  ranker carries its own bound.

## Decision 1 — `RankedPath` + `PathRanker` seam (in `graph`)

The `graph` package is the right home — path ranking is a pure computation
over a `graph.Subgraph`, no store, no LLM, stdlib only.

```go
package graph

// RankedPath is one scored simple path between two entities.
type RankedPath struct {
    EntityIDs   []string // ordered: path[0]..path[n], the traversal
    RelationIDs []string // ordered edges between consecutive entities
    Score       float64  // deterministic composite; higher = stronger
}

// PathRanker ranks simple paths within a Subgraph between seed entity
// pairs. Deterministic: same Subgraph + same seedPairs => same []RankedPath.
type PathRanker interface {
    RankPaths(sub Subgraph, seedPairs [][2]string) []RankedPath
}
// WeightedPathRanker{LengthDecay float64} — the default stdlib impl
```

## Decision 2 — `WeightedPathRanker`: deterministic, bounded, stdlib

- **Enumeration.** For each `[2]string` seed pair, a bounded DFS over the
  `Subgraph`'s relations enumerates *simple* paths (no repeated entity).
  Path length is capped at `maxPathLen` edges (a `graph`-package const = 2,
  matching `store`'s `maxGraphDepth`; `graph` cannot import `store`). The
  `Subgraph` is already fan-out/depth bounded, so the path count is bounded
  by construction. Relations are treated undirected for connectivity.
- **Scoring** — a composite of signals already present, no randomness:
  - **length**: `LengthDecay^(edges-1)` (`LengthDecay <= 0` → 0.5) — shorter
    paths score higher;
  - **edge weight**: the product of `edgeWeight(r)` over the path's
    relations;
  - **provenance overlap**: a small bonus when consecutive relations share
    `SourceChunkIDs` (co-attested evidence).
  Combine into one scalar.
- **Total order.** Sort `[]RankedPath` by `Score` desc, tie-broken by the
  path's joined `EntityIDs` sequence — a total, reproducible order, the same
  discipline as `Canonicalize` / `LouvainDetector`. Golden-testable.

## Decision 3 — opt-in path mode on `GraphRetriever` (additive)

`GraphRetriever` gains a `PathRanker graph.PathRanker` field. When non-nil,
after `Retrieve` builds `sub`, it computes the seed pairs (every unordered
pair of linked seed entity IDs), calls `RankPaths(sub, pairs)`, and records
the result on the trace. When nil, `Retrieve` is **byte-identical** to v0.7
— chunk hits, scoring, RRF behavior all unchanged.

`GraphTrace` gains two additive fields (mirroring how v0.8 added
`CommunityIDs`):

```go
type GraphTrace struct {
    SeedEntityIDs    []string
    ReachedEntityIDs []string
    MaxHop           int
    CommunityIDs     []string           // v0.8
    Paths            []RankedPath       // v0.9 — nil unless PathRanker set
    EvidenceSubgraph *graph.Subgraph    // v0.9 — the structured evidence
}
```

`EvidenceSubgraph` is the `sub` the retriever already built — surfaced as
the structured evidence object. It rides through `rag.Diagnostics.GraphTrace`
for free. `VariantRetriever` already propagates the whole `Trace.Graph`
block (the v0.7 22-03 fix), so the new fields flow without a `retrieve.go`
change.

## Slice breakdown

- **26-01** — `graph` package: `RankedPath` type + `PathRanker` seam +
  `WeightedPathRanker` (bounded-DFS enumeration, composite deterministic
  score, total tie-break); golden-output unit tests. (RAG-GRAPH4-01)
- **26-02** — `retrieve.GraphRetriever` opt-in `PathRanker` field;
  `GraphTrace` gains `Paths` + `EvidenceSubgraph`; `Retrieve` byte-identical
  when nil; tests cover both modes + the `Diagnostics` surfacing.
  (RAG-GRAPH4-02)
- **26-03** — deterministic path-ranking worked example;
  `docs/graphrag.md` path-evidence section. (RAG-GRAPH4-02)

## Risks / notes

- Path enumeration on a hub entity could be wide — bounded by the
  `Subgraph` already being depth-2/fan-out-64 limited, plus `maxPathLen`.
  No unbounded recursion.
- Determinism: every map drained to a sorted slice; DFS visits neighbors in
  sorted entity-ID order; the score tie-break is total.
- 26-02 depends on 26-01; 26-03 depends on 26-01+02.
- No new module dependency — path ranking is stdlib over the existing
  `Subgraph`.

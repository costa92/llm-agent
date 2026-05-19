# Phase 22 Research: Graph-traversal retrieval and fusion

**Researched:** 2026-05-18
**Phase:** 22 — graph-traversal retrieval and fusion (final v0.7 phase)
**Requirements:** RAG-GRAPH-05, RAG-GRAPH-06
**Repos:** `llm-agent-rag`
**Upstream:** `.planning/research/v0.7-graphrag-SUMMARY.md` §4 (traversal
retrieval) and §6 (seams); `20-RESEARCH.md`, `21-RESEARCH.md`.

## Current state (codebase scan)

- `retrieve.HybridRetriever{Dense, Lexical, Structure Retriever;
  RRFConstant float64}` fuses signals via reciprocal rank fusion:
  `apply(hits, ranks)` adds `1/(k+rank)` per signal into `rrfScores`;
  `FusionAttribution{ChunkID, DenseRank, LexicalRank, StructureRank,
  RRFScore}` records per-signal ranks. RRF is signal-count-agnostic — a
  fourth signal is a clean extension.
- `Structure` is gated per-query by `req.EnableStructure` — the precedent
  for an `EnableGraph` toggle.
- `retrieve.Retriever` — `Retrieve(ctx, Request) ([]store.Hit, Trace,
  error)`. `retrieve.Trace` carries `Fusion`, `Hops`, `Metrics`.
- `store.GraphStore` (Phase 21) — `Neighborhood`, `FindEntities`;
  `graph.Subgraph{Entities, Relations, Depth map[string]int}`.
- `graph.Entity.SourceChunkIDs` is the provenance linking an entity back to
  the chunks it was extracted from — the bridge from graph nodes to
  `store.Hit`s.
- `store.Store.Get(ctx, id)` fetches one `StoredChunk` by ID.
- `eval` package — `Dataset`/`Example`, `Evaluator`, `Metrics`
  (precision/recall/MRR/grounding), the RAG-Triad harness.

## Decision 1 — `EntityLinker` lives in `retrieve`, not `graph`

`EntityLinker.Link` needs a `store.GraphStore` to resolve the query to
seed entities. `store` imports `graph` (Phase 21) — so `graph` must **not**
import `store`. Therefore `EntityLinker` cannot live in `graph`. It lives
in `retrieve`, which already imports `store`, `graph`, `embed`, `generate`
— no cycle:

```go
package retrieve

type EntityLinker interface {
    Link(ctx context.Context, query string, gs store.GraphStore, namespace string) ([]graph.Entity, error)
}
// LexicalEntityLinker — tokenize the query, gs.FindEntities by token.
```

`LexicalEntityLinker` is the v0.7 deliverable: no embedder needed, keeps
the default path zero-LLM. An `EmbeddingEntityLinker` (embed query vs.
entity-name embeddings) is enabled by the seam but **deferred** — entity
embeddings are not stored in v0.7; building that is a v0.8 item.

## Decision 2 — `GraphRetriever` (a `retrieve.Retriever`)

```go
type GraphRetriever struct {
    Linker   EntityLinker
    Store    store.Store // type-asserted for store.GraphStore
    MaxDepth int         // default 1, hard cap 2
    HopDecay float64     // proximity score decay per hop, default 0.5
}
```

`Retrieve`: type-assert `Store` for `store.GraphStore` (degrade to empty
result if absent); `Link` the query to seed entities; `Neighborhood` from
the seed IDs at `MaxDepth`; for each reached entity, fetch its
`SourceChunkIDs` via `Store.Get` and score the resulting hits by
proximity decay — `score = HopDecay^hop` — deduping a chunk reached via
multiple entities to its best (lowest-hop) score; sort, truncate to
`req.TopK`. The returned `Trace` carries graph attribution
(`GraphTrace{SeedEntityIDs, ReachedEntityIDs, MaxHop}`).

## Decision 3 — fuse as a fourth RRF signal

`HybridRetriever` gains a `Graph Retriever` field; `FusionAttribution`
gains `GraphRank int`. In `HybridRetriever.Retrieve`, when
`req.EnableGraph && r.Graph != nil`, run the graph retriever and
`apply(graphHits, graphRank)` — the same `1/(k+rank)` RRF as the other
three signals. `req.EnableGraph` (and `rag.SearchOptions.EnableGraph`) is
the per-query toggle, mirroring `EnableStructure` — and the on/off switch
the eval A/B needs. Default `rag.System` behavior is unchanged unless graph
retrieval is explicitly wired and enabled (KG-4).

## Decision 4 — graph attribution in Trace + Diagnostics

`retrieve.Trace` gains a `Graph GraphTrace` block (seed entities,
reached entities, max hop). `HybridRetriever` merges the graph sub-trace
in. `rag.Diagnostics` surfaces it (a `GraphTrace`-shaped field), mirroring
how `Fusion`/`Hops` are already surfaced.

## Decision 5 — eval coverage (RAG-GRAPH-06)

- A graph-recall path in `eval`: run a labelled dataset through retrieval
  with `EnableGraph` off and on, and report the recall (and MRR) of each —
  the A/B that answers "does the graph signal help?".
- A deterministic worked example: the `DictionaryEntityExtractor` + the
  in-memory `GraphStore` + `GraphRetriever`, scripted model — fully
  reproducible, per repo convention.
- Docs: the GraphRAG usage guide, the v0.8 deferral list (community
  detection, global search, fuzzy resolution), and the "Neo4j is a future
  `GraphStore` impl" note.

## Slice breakdown

- **22-01** — `EntityLinker` seam + `LexicalEntityLinker`; `GraphRetriever`
  (link → bounded `Neighborhood` → proximity-decay-scored hits + graph
  `Trace`). (RAG-GRAPH-05)
- **22-02** — fuse graph as a fourth RRF signal in `HybridRetriever`
  (`Graph` field, `FusionAttribution.GraphRank`, `EnableGraph` toggle on
  `Request`/`SearchOptions`); graph attribution in `retrieve.Trace` +
  `rag.Diagnostics`. (RAG-GRAPH-05)
- **22-03** — `eval` graph-recall A/B (graph on vs off); a deterministic
  worked example; the GraphRAG docs + the v0.8 deferral note.
  (RAG-GRAPH-06)

## Risks / notes

- `GraphRetriever` fetches provenance chunks via per-chunk `Store.Get` —
  fine at this SDK's scale; a batched get is a possible later optimization.
- `EmbeddingEntityLinker` is deferred (no stored entity embeddings in
  v0.7) — the `EntityLinker` seam keeps it a clean future addition.
- The graph signal degrades to nothing when the store is not a
  `GraphStore`, when no entity links, or when `EnableGraph` is false —
  hybrid retrieval is unaffected (KG-4).
- 22-02 depends on 22-01; 22-03 depends on 22-01+02.
- No new module dependency — `retrieve` already imports everything needed.

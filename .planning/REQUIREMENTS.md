# Requirements: v0.8 GraphRAG Tier-3 — communities, global search, fuzzy resolution

**Defined:** 2026-05-19
**Shipped:** 2026-05-20 — `llm-agent-rag` tagged `v0.5.0`; milestone audit
PASS, 6/6 requirements (`.planning/v0.8-MILESTONE-AUDIT.md`).
**Core Value:** the core `llm-agent` module stays stdlib-only and zero-dep;
`llm-agent-rag` extends its v0.7 GraphRAG with hierarchical community
detection, lazy community summaries, a map-reduce global-search answer
path, and embedding-similarity fuzzy entity resolution — all additive,
with no graph database and no new module dependency.

## Milestone Scope

v0.8 delivers the **Tier-3 GraphRAG** capabilities v0.7 explicitly deferred
(v0.7 keystone calls KG-1 and KG-6). Building on the shipped v0.7 stack —
the `graph` package, the `store.GraphStore` optional capability, and
`retrieve.GraphRetriever` — v0.8 adds:

- **hierarchical community detection** — deterministic stdlib clustering of
  the canonicalized entity graph into a nested community hierarchy;
- **lazy community summaries** — LLM-written "community reports" generated
  at query time and cached (LazyGraphRAG model), with eager pre-warming as
  an opt-in;
- **map-reduce global search** — a `rag.System.AskGlobal` answer path for
  whole-corpus "sense-making" questions, distinct from chunk retrieval;
- **fuzzy entity resolution** — an opt-in embedding-similarity pre-pass that
  merges near-duplicate entities before `Canonicalize`.

The milestone is **additive**: no existing seam changes shape. New `graph`
types (`Community`, `CommunityReport`), new seams (`CommunityDetector`,
`CommunitySummarizer`, `EntityResolver`), community persistence behind the
graph-store capability, and a new `rag.System.AskGlobal` method. v0.7's
`GraphRetriever` and `HybridRetriever` are untouched.

Reference: `.planning/research/v0.8-graphrag-tier3-SUMMARY.md` — the
GraphRAG Tier-3 domain research and the KG3-1..KG3-8 keystone decisions
this milestone ratifies.

## v0.8 Requirements

### Community detection and persistence

- [x] **RAG-GRAPH3-01**: the `graph` package gains a `Community` type and a
      `CommunityDetector` seam; a pure-stdlib **Louvain** detector produces a
      deterministic community hierarchy (sorted iteration, deterministic
      tie-breaks, no randomness, native hierarchy via coarsening passes),
      unit-tested with golden output. A `LabelPropagationDetector`
      alternative ships behind the same seam (dual-mode, mirroring v0.7's
      extractor pair). No LLM, no embedder, no new dependency.
- [x] **RAG-GRAPH3-02**: community persistence sits behind the graph-store
      capability (additive `GraphStore` methods or a sibling `CommunityStore`
      capability — a plan-time call) — a pure-stdlib in-memory implementation
      and a `postgres` implementation (`_communities` table in the existing
      `Migrate()`); a shared conformance suite covers both. Community
      detection runs in Go regardless of store and is wired into `Import` as
      a post-`Canonicalize` stage; counts surface on `ImportResult`, and a
      graph-changing `ReplaceSource` re-ingest re-detects communities for
      that namespace.

### Community summaries and global search

- [x] **RAG-GRAPH3-03**: the `graph` package gains a `CommunityReport` type
      and a `CommunitySummarizer` seam; an `LLMCommunitySummarizer` over
      `generate.Model` writes a community report with lenient parsing,
      unit-tested against a scripted model (including malformed output).
      Reports are generated **lazily** and held in a cache keyed by a
      deterministic content hash of the community's membership; the cache is
      persisted via the `postgres` `_community_reports` table when a
      community-capable store is present, in-memory otherwise.
- [x] **RAG-GRAPH3-04**: `rag.System.AskGlobal` answers whole-corpus
      questions by map-reduce over community reports — community selection →
      per-community map (partial answer + helpfulness score) → reduce
      (synthesis). It does **not** implement `retrieve.Retriever` and does
      **not** pass through `HybridRetriever`/rerank/pack; `Ask` is unchanged.
      A global-search block surfaces in `Diagnostics` (communities consulted,
      map scores, map/reduce token counts). An opt-in
      `PrewarmCommunityReports` fills the cache eagerly. Scripted-model
      tested on a fixed graph.

### Fuzzy resolution and evaluation

- [x] **RAG-GRAPH3-05**: the `graph` package gains an `EntityResolver` seam;
      an `EmbeddingEntityResolver` over `embed.Embedder` clusters
      near-duplicate entities by cosine similarity (same-`Type` only,
      conservative threshold, deterministic canonical-form pick) and rewrites
      their names to a shared surface form **before** `graph.Canonicalize`
      runs — `Canonicalize` and its v0.7 tests untouched. The `rag.System`
      default is `NoopEntityResolver` (v0.7 behavior byte-identical);
      unit-tested against a scripted embedder.
- [x] **RAG-GRAPH3-06**: the `eval` package gains a global-search evaluation
      harness over the RAG-Triad / `LLMJudge` path (comprehensiveness /
      groundedness on whole-corpus questions — **not** `RunGraphAB`
      recall@k); a deterministic scripted-model worked example ships, and
      `docs/graphrag.md` is updated with Tier-3 usage, the lazy-vs-eager
      tradeoff, the fuzzy-resolution false-positive caveat, and the v0.9
      deferral list (DRIFT search, incremental community maintenance,
      path-ranking).

## Out of Scope

| Feature | Reason |
|---------|--------|
| DRIFT search (global primer → local follow-up loop) | A refinement on top of working global + local search; deferred to v0.9 once both are shipped and evaluated |
| Incremental community *maintenance* (updating only changed communities) | v0.8 does full per-namespace re-detection on re-ingest — cheap stdlib; incremental maintenance is a documented deferred optimization |
| Leiden community detection | Marginal quality gain over Louvain for this SDK's graph sizes does not justify the implementation/test surface; a documented future swap behind the `CommunityDetector` seam |
| Path-ranking / structured subgraph-as-evidence output | Tier-2 path retrieval — orthogonal to Tier-3, a v0.9+ item |
| Claim / covariate extraction | MS-GraphRAG's third primitive — out of scope |
| A dedicated graph database (Neo4j etc.) | Community detection runs in Go regardless of store; `GraphStore` stays an interface so a graph-DB impl can be added later in isolation |
| Embedding or graph-store deps in core `llm-agent` | Violates the zero-dependency core value |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| RAG-GRAPH3-01 | Phase 23 | Done |
| RAG-GRAPH3-02 | Phase 23 | Done |
| RAG-GRAPH3-03 | Phase 24 | Done |
| RAG-GRAPH3-04 | Phase 24 | Done |
| RAG-GRAPH3-05 | Phase 25 | Done |
| RAG-GRAPH3-06 | Phase 25 | Done |

**Coverage:**
- v0.8 requirements: 6 total
- Mapped to phases: 6
- Unmapped: 0
- Delivered: 6/6 (audit PASS)

---
*Requirements defined: 2026-05-19 — v0.8 GraphRAG Tier-3 milestone, scoped
from `.planning/research/v0.8-graphrag-tier3-SUMMARY.md`. Shipped 2026-05-20
(`llm-agent-rag v0.5.0`); frozen copy archived to
`.planning/milestones/v0.8-REQUIREMENTS.md`. Previous milestone v0.7
archived to `.planning/milestones/v0.7-REQUIREMENTS.md`.*

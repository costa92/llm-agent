# Requirements: v0.9 GraphRAG refinements — DRIFT search and path-ranking

**Defined:** 2026-05-20
**Shipped:** 2026-05-20 — `llm-agent-rag` tagged `v0.6.0`; milestone audit
PASS, 4/4 requirements (`.planning/v0.9-MILESTONE-AUDIT.md`).
**Core Value:** the core `llm-agent` module stays stdlib-only and zero-dep;
`llm-agent-rag` refines its GraphRAG stack with DRIFT hybrid search and
path-ranked subgraph evidence — both additive, with no graph database and
no new module dependency.

## Milestone Scope

v0.9 delivers two of the three GraphRAG refinements v0.8 named on its
deferral list (v0.8 keystone KG3-1). Building on the shipped v0.7 Tier-1
graph and v0.8 Tier-3 communities/global search:

- **path-ranking / subgraph-as-evidence** — beyond v0.7's proximity-decay
  node scoring, rank multi-hop *paths* between linked entities
  deterministically and surface a structured subgraph as evidence;
- **DRIFT search** — a third answer path that runs a global "primer" pass
  for broad orientation, then a bounded local follow-up loop that drills
  into the entities the primer surfaced, then synthesizes.

**Incremental community maintenance** — the third v0.8 deferral — is
**deferred again** (KG4-5): v0.8's full re-detection on re-ingest is
correct and fast at this SDK's scale; incremental Louvain is a large,
subtle second algorithm solving a performance problem the SDK does not
have. It is documented with an explicit profiling trigger for a later
milestone.

The milestone is **additive**: no existing seam changes shape. A new
`rag.System.AskDrift` answer method, new path-ranking types in `graph`, an
opt-in path mode on `retrieve.GraphRetriever`, and a DRIFT eval harness.
`AskGlobal`, `GraphRetriever`, `GraphStore`, `CommunityStore`, and
`LouvainDetector` are untouched.

Reference: `.planning/research/v0.9-graphrag-refinements-SUMMARY.md` — the
domain research and the KG4-1..KG4-7 keystone decisions this milestone
ratifies.

## v0.9 Requirements

### Path-ranking and subgraph-as-evidence

- [x] **RAG-GRAPH4-01**: the `graph` package gains a `RankedPath` type and a
      `PathRanker` seam; a pure-stdlib deterministic path ranker enumerates
      simple multi-hop paths within a `Subgraph` (bounded DFS, depth cap 2)
      and scores them by a composite of path length, `Relation.Weight`, and
      provenance overlap, with a total tie-break on the path's entity-ID
      sequence — golden-output unit-tested. No LLM, no new dependency.
- [x] **RAG-GRAPH4-02**: `retrieve.GraphRetriever` gains an opt-in path
      mode (an injectable `PathRanker`); `retrieve.GraphTrace` gains
      additive `Paths` and `EvidenceSubgraph` fields surfaced through
      `rag.Diagnostics`. `GraphRetriever.Retrieve`'s signature, chunk-hit
      output, and RRF-fusion behavior — and all v0.7/v0.8 tests — are
      byte-identical when path mode is off. A deterministic worked example
      and docs ship.

### DRIFT search

- [x] **RAG-GRAPH4-03**: `rag.System.AskDrift` answers by a global primer
      pass (the v0.8 global path) followed by a hard-bounded local
      follow-up loop (round cap — default 2, hard cap 3 — terminating on
      no-new-entities) and a synthesis step. It is a separate answer path —
      it does not implement `retrieve.Retriever` and is not a mode flag on
      `Ask`/`AskGlobal`. `DriftOptions` and a `Diagnostics.Drift` block
      ship; the orchestration is scripted-model golden-tested; an empty
      primer degrades gracefully to a local-loop answer.
- [x] **RAG-GRAPH4-04**: the `eval` package gains a `DriftEvaluator` — a
      RAG-Triad / `LLMJudge` harness for DRIFT answers (groundedness,
      answer-relevance — not chunk recall@k), mirroring `GlobalEvaluator`;
      a deterministic scripted-model worked example ships, and
      `docs/graphrag.md` is finalized with DRIFT usage, the primer/local
      budget, and the deferral list (incremental community maintenance with
      its trigger condition, claim extraction, graph DB).

## Out of Scope

| Feature | Reason |
|---------|--------|
| Incremental community maintenance | v0.8's full re-detection is correct and fast at SDK scale; incremental Louvain is a large, subtle second algorithm — deferred with a documented profiling trigger (KG4-5) |
| DRIFT loops beyond a small fixed round cap | Unbounded LLM recursion; v0.9 hard-caps rounds at 3 (KG4-3) |
| Claim / covariate extraction | MS-GraphRAG's third primitive — out of scope |
| A dedicated graph database (Neo4j etc.) | Path ranking runs in Go over the existing `Subgraph`; DRIFT orchestrates existing seams |
| Embedding or graph-store deps in core `llm-agent` | Violates the zero-dependency core value |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| RAG-GRAPH4-01 | Phase 26 | Done |
| RAG-GRAPH4-02 | Phase 26 | Done |
| RAG-GRAPH4-03 | Phase 27 | Done |
| RAG-GRAPH4-04 | Phase 27 | Done |

**Coverage:**
- v0.9 requirements: 4 total
- Mapped to phases: 4
- Unmapped: 0
- Delivered: 4/4 (audit PASS)

---
*Requirements defined: 2026-05-20 — v0.9 GraphRAG refinements milestone,
scoped from `.planning/research/v0.9-graphrag-refinements-SUMMARY.md`.
Shipped 2026-05-20 (`llm-agent-rag v0.6.0`); frozen copy archived to
`.planning/milestones/v0.9-REQUIREMENTS.md`. Previous milestone v0.8
archived to `.planning/milestones/v0.8-REQUIREMENTS.md`.*

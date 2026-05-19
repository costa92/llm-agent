# Roadmap: llm-agent

**Last updated:** 2026-05-20
**Current state:** between milestones — `v0.9` GraphRAG refinements shipped
(`llm-agent-rag v0.6.0`, audit PASS); next milestone not yet scoped
**Active scope:** none — awaiting next-milestone definition

## Archived Milestones

- [x] **v0.3: Deployable multi-repo release** — shipped 2026-05-12.
  Delivered `llm/v2`, three real provider adapters, OTel wrappers, and the
  customer-support demo stack across 4 repos.
  - Archive: `.planning/milestones/v0.3-ROADMAP.md`
  - Requirements archive: `.planning/milestones/v0.3-REQUIREMENTS.md`
  - Audit: `.planning/v0.3-MILESTONE-AUDIT.md`
- [x] **v0.5: RAG productionization and standalone SDK evolution** — shipped
  2026-05-15. Structure-aware retrieval, a PostgreSQL + pgvector backend
  with a shared conformance suite, tracing hooks, an evaluation framework, a
  feedback loop, and cross-repo contract gates. `llm-agent-rag` tagged
  `v0.2.0`.
  - Archive: `.planning/milestones/v0.5-ROADMAP.md`
  - Requirements archive: `.planning/milestones/v0.5-REQUIREMENTS.md`
- [x] **v0.6: Production-grade retrieval quality and safety** — shipped
  2026-05-18. Six phases (14-19): BM25 lexical retrieval + RRF fusion,
  model-based reranking with explainability, the RAG Triad, cost/latency
  observability (`obs` + `otelrag` RED metrics), content safety (`guard`),
  and agentic retrieval (`MultiHopRetriever` + `CorrectiveAsker`).
  `llm-agent-rag` tagged `v0.3.0`; no new dependency.
  - Archive: `.planning/milestones/v0.6-ROADMAP.md`
  - Requirements archive: `.planning/milestones/v0.6-REQUIREMENTS.md`
  - Audit: `.planning/v0.6-MILESTONE-AUDIT.md`
- [x] **v0.7: GraphRAG — relationship-traversal retrieval** — shipped
  2026-05-19. Three phases (20-22): Tier-1 lightweight GraphRAG — the
  `graph` package + dual-mode entity/relation extraction, a
  `store.GraphStore` optional capability (stdlib in-memory + `postgres`
  recursive-CTE) with re-ingest reconciliation, and `retrieve.GraphRetriever`
  fused as a fourth RRF signal. `llm-agent-rag` tagged `v0.4.0`; no new
  dependency, no graph database.
  - Archive: `.planning/milestones/v0.7-ROADMAP.md`
  - Requirements archive: `.planning/milestones/v0.7-REQUIREMENTS.md`
  - Audit: `.planning/v0.7-MILESTONE-AUDIT.md`
- [x] **v0.8: GraphRAG Tier-3 — communities, global search, fuzzy
  resolution** — shipped 2026-05-20. Three phases (23-25): hierarchical
  community detection (deterministic stdlib Louvain), a `store.CommunityStore`
  capability (in-memory + `postgres`), lazy community summaries, the
  `rag.System.AskGlobal` map-reduce global-search answer path, and an opt-in
  `EmbeddingEntityResolver` fuzzy entity-resolution pre-pass. `llm-agent-rag`
  tagged `v0.5.0`; no new dependency, no graph database.
  - Archive: `.planning/milestones/v0.8-ROADMAP.md`
  - Requirements archive: `.planning/milestones/v0.8-REQUIREMENTS.md`
  - Audit: `.planning/v0.8-MILESTONE-AUDIT.md`
- [x] **v0.9: GraphRAG refinements — DRIFT search and path-ranking** —
  shipped 2026-05-20. Two phases (26-27): path-ranked subgraph evidence (a
  deterministic stdlib `graph.PathRanker` + an opt-in `GraphRetriever`
  mode) and DRIFT hybrid search (`rag.System.AskDrift` — a global primer +
  bounded local follow-up loop + synthesis — plus `eval.DriftEvaluator`).
  `llm-agent-rag` tagged `v0.6.0`; no new dependency, no graph database.
  - Archive: `.planning/milestones/v0.9-ROADMAP.md`
  - Requirements archive: `.planning/milestones/v0.9-REQUIREMENTS.md`
  - Audit: `.planning/v0.9-MILESTONE-AUDIT.md`

## Active Forward Work

None — `v0.9` is closed and archived. The next milestone has not been
scoped. Candidate directions carried forward:

- **Incremental community maintenance** — update only the communities a
  re-ingest perturbs, instead of v0.8's full per-namespace re-detection.
  Deferred from v0.9 (keystone KG4-5); revisit if profiling shows
  `Detect` dominating re-ingest on a real corpus.
- **The `llm-agent-rag` deployment layer** — the HTTP service, CLI, and
  caching surface deferred since v0.6.
- **Live-Postgres CI wiring** — carried-forward infra debt (see below).
- A **v1.0** stability pass — the SDK now spans the full practical GraphRAG
  spectrum; a v1.0 could lock the public API.

## Known Carry-forward Debt

- Live-Postgres CI wiring (testcontainers-go or GH Actions services) is
  still pending from v0.5; the Phase 14 Postgres `tsvector` path, the
  Phase 21 `postgres` graph path, and the Phase 23-24 `postgres`
  `_communities`/`_community_reports` paths all need verification against a
  live database.
- Incremental community maintenance is deferred (KG4-5) — v0.8's full
  re-detection on re-ingest stays until profiling shows it a bottleneck.
- Regex-based content safety (`guard`, v0.6) is best-effort — known
  patterns, not novel/obfuscated ones.
- `EmbeddingEntityResolver` (v0.8) has documented false-positive risk; it
  ships conservative (high threshold, same-type-only) and opt-in.
- Formal verification artifacts are still uneven after Phase 0.
- The refsvc demo remains intentionally demo-grade in observability fidelity
  and packaging.

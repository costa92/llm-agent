# Roadmap: llm-agent

**Last updated:** 2026-05-19
**Current state:** between milestones — `v0.7` GraphRAG shipped
(`llm-agent-rag v0.4.0`, audit PASS); next milestone not yet scoped
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

## Active Forward Work

None — `v0.7` is closed and archived. The next milestone has not been
scoped. Candidate directions carried forward:

- **v0.8 GraphRAG Tier-3** — Microsoft-GraphRAG-style hierarchical
  community detection, LLM community summaries, and map-reduce global /
  DRIFT search; fuzzy / embedding-similarity entity resolution. Explicitly
  deferred from v0.7 (keystone KG-1, KG-6).
- **`llm-agent-rag` deployment layer** — the HTTP service, CLI, and caching
  surface deferred since v0.6.
- **Live-Postgres CI wiring** — carried-forward infra debt (see below).

## Known Carry-forward Debt

- Live-Postgres CI wiring (testcontainers-go or GH Actions services) is
  still pending from v0.5; the Phase 14 Postgres `tsvector` path — and the
  Phase 21 `postgres` graph path (`entities`/`relations` tables, recursive-CTE
  `Neighborhood`) — need verification against a live database.
- Regex-based content safety (`guard`, v0.6) is best-effort — known
  patterns, not novel/obfuscated ones.
- Entity canonicalization (v0.7) is deterministic exact-match only; fuzzy /
  embedding entity resolution is deferred to v0.8.
- Formal verification artifacts are still uneven after Phase 0.
- The refsvc demo remains intentionally demo-grade in observability fidelity
  and packaging.

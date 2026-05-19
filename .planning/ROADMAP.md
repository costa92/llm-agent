# Roadmap: llm-agent

**Last updated:** 2026-05-21
**Current state:** between milestones — `v1.0` API stabilization shipped
(`llm-agent-rag v1.0.0`, audit PASS 6/6); next milestone not yet scoped
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
  2026-05-19. Three phases (20-22): Tier-1 lightweight GraphRAG.
  `llm-agent-rag` tagged `v0.4.0`; no new dependency, no graph database.
  - Archive: `.planning/milestones/v0.7-ROADMAP.md`
  - Requirements archive: `.planning/milestones/v0.7-REQUIREMENTS.md`
  - Audit: `.planning/v0.7-MILESTONE-AUDIT.md`
- [x] **v0.8: GraphRAG Tier-3 — communities, global search, fuzzy
  resolution** — shipped 2026-05-20. Three phases (23-25): community
  detection, lazy summaries, `AskGlobal` map-reduce global search, fuzzy
  entity resolution. `llm-agent-rag` tagged `v0.5.0`; no new dependency.
  - Archive: `.planning/milestones/v0.8-ROADMAP.md`
  - Requirements archive: `.planning/milestones/v0.8-REQUIREMENTS.md`
  - Audit: `.planning/v0.8-MILESTONE-AUDIT.md`
- [x] **v0.9: GraphRAG refinements — DRIFT search and path-ranking** —
  shipped 2026-05-20. Two phases (26-27): path-ranked subgraph evidence and
  DRIFT hybrid search. `llm-agent-rag` tagged `v0.6.0`; no new dependency.
  - Archive: `.planning/milestones/v0.9-ROADMAP.md`
  - Requirements archive: `.planning/milestones/v0.9-REQUIREMENTS.md`
  - Audit: `.planning/v0.9-MILESTONE-AUDIT.md`
- [x] **v1.0: API stabilization and the compatibility promise** — shipped
  2026-05-21. Three phases (28-30): a written exported-surface audit and
  the pre-freeze breaking renames (`eval.Evaluator`→`RetrievalEvaluator`,
  `eval.Result`→`RetrievalResult`; the `ragkit` root repurposed as a
  documented doc-anchor); full package + exported-symbol doc-comment
  coverage and a written `docs/compatibility.md` Go-module compatibility
  promise; a pure-stdlib `internal/apisnapshot` exported-surface gate
  (`api/v1.snapshot.txt` + a `go test` regeneration-diff) plus a
  `-tags llmagent` CI step. `llm-agent-rag` tagged `v1.0.0`; no new
  dependency, no behavior change.
  - Archive: `.planning/milestones/v1.0-ROADMAP.md`
  - Requirements archive: `.planning/milestones/v1.0-REQUIREMENTS.md`
  - Audit: `.planning/v1.0-MILESTONE-AUDIT.md`

## Active Forward Work

None — `v1.0` is closed and archived. `llm-agent-rag` has reached a
stable, frozen, documented `v1.0.0` public API. The next milestone has
not been scoped. Candidate directions carried forward:

- **The `llm-agent-rag` deployment layer** — the HTTP service, CLI, and
  caching surface deferred since v0.6. The first obvious `v1.x` (additive)
  candidate.
- **Incremental community maintenance** — update only the communities a
  re-ingest perturbs, instead of v0.8's full per-namespace re-detection.
  Deferred from v0.9 (KG4-5); additive, so `v1.x`-eligible.
- **Live-Postgres CI wiring** — carried-forward infra debt (see below).
- **Core `llm-agent` / sister-repo milestones** — the core module and the
  `-otel`/`-providers`/`-customer-support` repos are on their own version
  tracks (v1.0 scope was `llm-agent-rag` only, KS-1) and may each warrant
  their own next milestone.

## Known Carry-forward Debt

- Live-Postgres CI wiring (testcontainers-go or GH Actions services) is
  still pending from v0.5; the Phase 14/21/23-24 `postgres` paths need
  verification against a live database.
- Incremental community maintenance is deferred (v0.9 KG4-5) — v0.8's full
  re-detection stays.
- The `llm-agent-rag` deployment layer (HTTP service, CLI, caching) remains
  a deliberate non-goal, deferred since v0.6.
- Regex-based content safety (`guard`, v0.6) is best-effort.
- `EmbeddingEntityResolver` (v0.8) has documented false-positive risk.
- The refsvc demo remains intentionally demo-grade.

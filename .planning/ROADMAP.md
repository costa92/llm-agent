# Roadmap: llm-agent

**Last updated:** 2026-05-18
**Current state:** between milestones — `v0.6` shipped (`llm-agent-rag`
tagged `v0.3.0`); no milestone is currently active.
**Next:** run `/gsd-new-milestone` to scope the next milestone.

## Archived Milestones

- [x] **v0.3: Deployable multi-repo release** — shipped 2026-05-12.
  Delivered `llm/v2`, three real provider adapters, OTel wrappers, and the
  customer-support demo stack across 4 repos.
  - Archive: `.planning/milestones/v0.3-ROADMAP.md`
  - Requirements archive: `.planning/milestones/v0.3-REQUIREMENTS.md`
  - Audit: `.planning/v0.3-MILESTONE-AUDIT.md`
- [x] **v0.5: RAG productionization and standalone SDK evolution** — shipped
  2026-05-15. Delivered structure-aware retrieval, a PostgreSQL + pgvector
  backend with a shared conformance suite, tracing hooks, an evaluation
  framework, a feedback loop, and cross-repo contract gates. `llm-agent-rag`
  tagged `v0.2.0`.
  - Archive: `.planning/milestones/v0.5-ROADMAP.md`
  - Requirements archive: `.planning/milestones/v0.5-REQUIREMENTS.md`
- [x] **v0.6: Production-grade retrieval quality and safety** — shipped
  2026-05-18. Six phases (14-19), one per seam v0.5 left thin: BM25 lexical
  retrieval + principled RRF fusion, model-based reranking with
  explainability, the generation-side RAG Triad, cost/latency observability
  (`obs` + `otelrag` RED metrics), content safety (`guard` PII redaction +
  injection defense), and agentic retrieval (`MultiHopRetriever` +
  `CorrectiveAsker`). 12/12 requirements delivered; `llm-agent-rag` gained
  no new dependency. Tagged `v0.3.0`.
  - Archive: `.planning/milestones/v0.6-ROADMAP.md`
  - Requirements archive: `.planning/milestones/v0.6-REQUIREMENTS.md`
  - Audit: `.planning/v0.6-MILESTONE-AUDIT.md`

## Next Milestone

No milestone is currently active. Scope the next one with
`/gsd-new-milestone`. Candidate areas surfaced but deliberately deferred
during v0.6 (see Known Carry-forward Debt and the v0.6 gap analysis): the
`llm-agent-rag` deployment layer (HTTP service, CLI, caching), GraphRAG /
relationship traversal, and PDF/OCR ingestion.

## Known Carry-forward Debt

- `llm-agent-otel`'s `require github.com/costa92/llm-agent-rag` is still
  pinned to `v0.2.0`; bumping it to `v0.3.0` (so `otelrag` builds against
  the v0.6 RAG SDK without a `go.work`) is a pending step — see
  `.planning/STATE.md` Blockers.
- Live-Postgres CI wiring (testcontainers-go or GH Actions services) is
  still pending from v0.5; the Phase 14 Postgres `tsvector` lexical path
  remains unverified against a live database.
- Regex-based content safety (`guard`) is best-effort — it catches known
  PII and injection patterns, not novel/obfuscated ones; a model-based
  classifier is a future-milestone item.
- Formal verification artifacts are still uneven after Phase 0.
- The refsvc demo remains intentionally demo-grade in observability
  fidelity and packaging.

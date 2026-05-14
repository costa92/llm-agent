# Roadmap: llm-agent

**Last updated:** 2026-05-14
**Current state:** `v0.4` deprecation-removal cycle complete
**Active scope:** `v0.5` RAG productionization milestone

## Archived Milestones

- [x] **v0.3: Deployable multi-repo release** — shipped 2026-05-12.
  Delivered `llm/v2`, three real provider adapters, OTel wrappers, and the
  customer-support demo stack across 4 repos.
  - Archive: `.planning/milestones/v0.3-ROADMAP.md`
  - Requirements archive: `.planning/milestones/v0.3-REQUIREMENTS.md`
  - Audit: `.planning/v0.3-MILESTONE-AUDIT.md`

## Milestone v0.5: RAG productionization and standalone SDK evolution

**Goal**: turn the extracted `llm-agent-rag` module from a reusable baseline
into a production-oriented retrieval system while preserving a thin,
zero-dependency compatibility facade in the core `llm-agent` repo.

**Repos**: `llm-agent`, `llm-agent-rag`

**Requirements in scope**:

- `RAG-CORE-01..04`
- `RAG-INGEST-01..03`
- `RAG-RETRIEVE-01..04`
- `RAG-STRUCT-01..02`
- `RAG-OPS-01..03`
- `RAG-ECO-01..02`

## Active Forward Work

### Phase 8: RAG core contract hardening

**Status**: complete 2026-05-14

**Goal**: harden the standalone/core RAG contract so metadata filters,
security-trim semantics, citations, diagnostics, and provenance become
first-class before larger retrieval changes land.

**Depends on**:

- standalone `llm-agent-rag v0.1.x` baseline exists
- core repo already consumes the standalone module
- `v0.4` release line is stable

**Repos**: `llm-agent-rag`, `llm-agent`

**Requirements covered**:

- `RAG-CORE-01`
- `RAG-CORE-02`
- `RAG-CORE-03`
- `RAG-CORE-04`

**Planned work**:

- `08-01` implement real metadata filtering in the default store
- `08-02` add mandatory security filters and non-bypassable retrieval plumbing
- `08-03` add citations, provenance, and retrieval trace structures to `Ask`
- `08-04` align the core `rag/` compatibility facade and tool surface with the
  richer standalone contract

### Phase 9: Source-aware ingestion and index lifecycle

**Status**: in progress

**Goal**: move from flat chunk ingestion to source-aware, version-aware,
section-aware indexing with safe update semantics.

**Depends on**:

- Phase 8

**Repos**: `llm-agent-rag`

**Requirements covered**:

- `RAG-INGEST-01`
- `RAG-INGEST-02`
- `RAG-INGEST-03`

**Planned work**:

- `09-01` add source identity, version, checksum, and embedding-version fields
- `09-02` add heading-aware markdown ingestion and section metadata
- `09-03` define re-import, delete-by-source, and tombstone semantics

### Phase 10: Retrieval policies, hybrid recall, and context packing

**Status**: planned

**Goal**: turn retrieval into a configurable policy pipeline with dense,
lexical, hybrid, rerank, query-governance, and budget-aware prompt packing.

**Depends on**:

- Phase 9

**Repos**: `llm-agent-rag`, `llm-agent`

**Requirements covered**:

- `RAG-RETRIEVE-01`
- `RAG-RETRIEVE-02`
- `RAG-RETRIEVE-03`
- `RAG-RETRIEVE-04`

**Planned work**:

- `10-01` add retrieval policy and query-preprocessor seams
- `10-02` add lexical retrieval and hybrid fusion
- `10-03` move MQE/HyDE into standalone reusable policy hooks
- `10-04` add rerank plus token-budget-aware context packing

### Phase 11: Structure-aware retrieval and explainability

**Status**: planned

**Goal**: add PageIndex-style section/path-aware retrieval and search
trajectory output for long, hierarchical documents.

**Depends on**:

- Phase 10

**Repos**: `llm-agent-rag`

**Requirements covered**:

- `RAG-STRUCT-01`
- `RAG-STRUCT-02`

**Planned work**:

- `11-01` add document tree or section hierarchy primitives
- `11-02` add structured retrieval path and lineage-rich results

### Phase 12: Persistence, tracing, and backend conformance

**Status**: planned

**Goal**: make the standalone SDK deployable beyond in-memory demos with at
least one persistent backend and first-class tracing hooks.

**Depends on**:

- Phase 10

**Repos**: `llm-agent-rag`

**Requirements covered**:

- `RAG-OPS-01`
- `RAG-OPS-02`

**Planned work**:

- `12-01` implement one persistent vector backend
- `12-02` add backend conformance coverage and tracing hooks for import,
  retrieve, pack, and ask

### Phase 13: Evaluation, feedback loop, and ecosystem contract

**Status**: planned

**Goal**: close the milestone with regression-ready evaluation assets,
feedback-loop tooling, documentation, and CI contract gates across both repos.

**Depends on**:

- Phases 8-12

**Repos**: `llm-agent-rag`, `llm-agent`

**Requirements covered**:

- `RAG-OPS-03`
- `RAG-ECO-01`
- `RAG-ECO-02`

**Planned work**:

- `13-01` add retrieval and grounding regression datasets plus CI gates
- `13-02` document production deployment, backend selection, and compatibility
  guidance
- `13-03` add online-to-offline feedback workflow for production misses

## Known Carry-forward Debt

- Formal verification artifacts are still uneven after Phase 0.
- The refsvc demo remains intentionally demo-grade in observability fidelity
  and packaging.
- The standalone RAG module is still at an early baseline with only in-memory
  storage and thin retrieval diagnostics.

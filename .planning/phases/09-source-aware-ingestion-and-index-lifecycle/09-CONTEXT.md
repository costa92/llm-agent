# Phase 9: Source-Aware Ingestion and Index Lifecycle - Context

**Gathered:** 2026-05-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 9 is the ingestion and index-lifecycle foundation phase for the `v0.5`
RAG productionization milestone.

This phase moves the standalone SDK from flat text chunking toward
source-aware, version-aware indexing semantics that later structure-aware
retrieval and persistent backends can rely on.

Phase 9 produces:

- source identity and version metadata in ingestion contracts
- checksum and embedding-version lineage on indexed chunks
- heading-aware markdown ingestion groundwork
- explicit update semantics for re-import and delete-by-source paths

Phase 9 explicitly covers:

- `llm-agent-rag/ingest`
- `llm-agent-rag/rag` import path
- metadata propagation into `store.StoredChunk`
- tests that lock update and lineage semantics

Phase 9 does NOT cover:

- lexical/hybrid retrieval policy
- PageIndex-style structure-aware retrieval
- persistent vector backends
- full production connector fleet

</domain>

<decisions>
## Implementation Decisions

### D-01: Metadata lineage starts in `ingest.Document`

- Source identity, version, checksum, and related lineage fields belong at the
  document ingestion boundary.
- Chunk and stored-chunk metadata should inherit these values automatically.

### D-02: Add structure without breaking simple text import

- Existing raw-text and simple import flows must keep working.
- New metadata fields should be additive and safe for zero-value use.

### D-03: Stable metadata names matter

- The first wave should standardize metadata keys now so later retrieval,
  tracing, and persistent backends do not need a second migration.

### D-04: Update lifecycle starts with explicit semantics, not backend magic

- Re-import and delete-by-source behavior should be represented explicitly in
  contracts before persistent backends are introduced.

</decisions>

<canonical_refs>
## Canonical References

### Roadmap and requirements

- `.planning/ROADMAP.md` — Phase 9 scope and ordering
- `.planning/REQUIREMENTS.md` — `RAG-INGEST-01..03`
- `.planning/STATE.md` — active milestone position

### Prior phase and design docs

- `.planning/phases/08-rag-core-contract-hardening/08-01-SUMMARY.md`
- `docs/2026-05-14-rag-production-enhancement-plan.md`
- `docs/2026-05-13-standalone-rag-sdk-design.md`

### Current implementation

- `/tmp/llm-agent-rag/ingest/types.go`
- `/tmp/llm-agent-rag/ingest/splitter.go`
- `/tmp/llm-agent-rag/ingest/import.go`
- `/tmp/llm-agent-rag/rag/import.go`
- `/tmp/llm-agent-rag/store/types.go`

</canonical_refs>

<specifics>
## Success Markers to Preserve

- existing `Import` and `ImportFrom` flows remain simple for callers that only
  provide raw text
- new metadata lineage is automatically propagated into chunks and stored
  chunks
- later phases can build retrieval and backend logic from stable metadata keys
- standalone tests stay green with `GOWORK=off`

</specifics>

<deferred>
## Deferred Ideas

- markdown hierarchy-aware chunking beyond baseline metadata is still part of
  later Phase 9 slices
- delete-by-source execution helpers can begin as metadata semantics and be
  expanded further in later plans

</deferred>

---

*Phase: 09-source-aware-ingestion-and-index-lifecycle*
*Context gathered: 2026-05-14 via GSD continuation after Phase 8*

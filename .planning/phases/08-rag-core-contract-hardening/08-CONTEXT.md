# Phase 8: RAG Core Contract Hardening - Context

**Gathered:** 2026-05-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 8 is the first active phase of the `v0.5` RAG productionization
milestone.

This phase does not try to solve all of production RAG. Its job is to harden
the baseline contract between the standalone `llm-agent-rag` repo and the
historical `llm-agent/rag` facade so later retrieval and indexing work has a
stable foundation.

Phase 8 produces:

- real metadata filtering in the standalone default store
- a contract distinction between optional caller filters and mandatory security
  filters
- machine-readable citations and retrieval diagnostics in the standalone ask
  path
- provenance-rich retrieval results
- compatibility alignment in the core `rag/` wrapper and tool surface

Phase 8 explicitly covers:

- `llm-agent-rag/store`
- `llm-agent-rag/rag`
- core compatibility wiring in `llm-agent/rag`
- tests and CI guards needed to preserve this contract

Phase 8 does NOT cover:

- persistent vector stores
- lexical or hybrid retrieval
- markdown hierarchy-aware ingestion
- PageIndex-style structure-aware retrieval
- production service packaging

</domain>

<decisions>
## Implementation Decisions

### D-01: Standalone RAG stays the source of truth

- New contract richness lands in `llm-agent-rag` first.
- The core repo mirrors that behavior through adapters rather than inventing a
  separate RAG implementation path.

### D-02: Security filters are distinct from normal caller filters

- The API must support application-visible metadata filters.
- The API must also support non-bypassable security trimming that the caller
  cannot silently omit.
- Tests must prove security filters are always applied.

### D-03: Diagnostics are part of the public contract

- Retrieval traces, provenance, and citations are not debug-only extras.
- They are required output so later eval, feedback, and observability work has
  stable data to build on.

### D-04: Keep the core repo zero-dependency

- No provider or backend dependencies move into `llm-agent`.
- The compatibility facade can adapt richer response types, but the core value
  of the main repo cannot be violated by this phase.

### D-05: Prefer incremental API expansion over rewrites

- Add new fields and options in ways that preserve existing use where
  reasonable.
- Break historical `rag` behavior only when necessary to fix correctness gaps.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Roadmap and requirements

- `.planning/ROADMAP.md` — active milestone and Phase 8 scope
- `.planning/REQUIREMENTS.md` — `RAG-CORE-01..04`
- `.planning/STATE.md` — current project and milestone position
- `.planning/PROJECT.md` — core value and multi-repo constraints

### RAG design and milestone research

- `docs/2026-05-14-rag-production-enhancement-plan.md` — primary design and
  phased milestone plan
- `docs/2026-05-13-standalone-rag-sdk-design.md` — original standalone SDK
  architecture
- `docs/2026-05-13-rag-sdk-migration-status.md` — current migration and
  compatibility status

### Current implementation

- `/tmp/llm-agent-rag/rag/system.go`
- `/tmp/llm-agent-rag/rag/options.go`
- `/tmp/llm-agent-rag/rag/retrieve.go`
- `/tmp/llm-agent-rag/rag/ask.go`
- `/tmp/llm-agent-rag/store/store.go`
- `/tmp/llm-agent-rag/store/inmemory.go`
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/rag.go`
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/tool.go`

</canonical_refs>

<specifics>
## Success Markers to Preserve

- `llm-agent-rag` remains publishable as a standalone module.
- Core `llm-agent` continues to expose the historical `rag` API for existing
  callers.
- Metadata filtering is real and tested in the standalone store, not simulated
  only in the compatibility adapter.
- The answer path returns enough structure for future rerank, eval, and
  feedback-loop work without forcing another contract redesign.
- `go test ./...` remains green in both repos with `GOWORK=off`.

</specifics>

<deferred>
## Deferred Ideas

- lexical/hybrid retrieval policy is Phase 10
- section-aware retrieval is Phase 11
- persistent backends are Phase 12
- feedback-loop CI and production regression promotion are Phase 13

</deferred>

---

*Phase: 08-rag-core-contract-hardening*
*Context gathered: 2026-05-14 via GSD milestone setup*

# Phase 10: Retrieval Policies, Hybrid Recall, and Context Packing - Context

**Gathered:** 2026-05-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 10 turns retrieval from a single hardcoded dense path into an explicit
policy pipeline with extension seams for query preprocessing, multiple
retrievers, fusion, and prompt-context packing.

This phase begins with the minimum orchestration seam:

- `QueryPreprocessor`
- `Retriever`
- default dense retrieval policy

Later Phase 10 slices can then add lexical retrieval, hybrid fusion, MQE/HyDE
integration, and context packing without another contract redesign.

Phase 10 explicitly covers:

- `llm-agent-rag/rag`
- new retrieval-policy packages in the standalone repo
- integration points for advanced query expansion hooks

Phase 10 does NOT cover:

- persistent backend implementation
- PageIndex-style structure-aware retrieval
- core `llm-agent` compatibility updates unless needed for consumption later

</domain>

<decisions>
## Implementation Decisions

### D-01: Add seams before algorithms

- The first step is policy structure, not complex retrieval logic.
- Default dense retrieval should continue to work unchanged for existing
  callers.

### D-02: Query governance starts additive

- Query rewriting, classification, and decomposition should enter behind a
  preprocessor seam.
- No caller should be forced to adopt them to keep current behavior.

### D-03: Dense retrieval remains the default baseline

- Existing `Retrieve(...)` behavior should continue to map to dense retrieval
  unless a different policy is configured.

</decisions>

<canonical_refs>
## Canonical References

- `.planning/ROADMAP.md`
- `.planning/REQUIREMENTS.md` — `RAG-RETRIEVE-01..04`
- `.planning/phases/09-source-aware-ingestion-and-index-lifecycle/09-01-SUMMARY.md`
- `docs/2026-05-14-rag-production-enhancement-plan.md`
- `/tmp/llm-agent-rag/rag/retrieve.go`
- `/tmp/llm-agent-rag/rag/options.go`
- `/tmp/llm-agent-rag/advanced/llm.go`

</canonical_refs>

<specifics>
## Success Markers to Preserve

- existing `Retrieve(...)` callers continue to work
- the new seam is explicit enough to host lexical/hybrid retrieval next
- MQE/HyDE can move under the policy layer later without another rewrite

</specifics>

---

*Phase: 10-retrieval-policies-hybrid-recall-and-context-packing*
*Context gathered: 2026-05-14 via GSD continuation after Phase 9*

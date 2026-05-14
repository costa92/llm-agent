# Phase 8: RAG Core Contract Hardening - Research

**Researched:** 2026-05-14
**Status:** Seeded from the v0.5 roadmap, standalone RAG design docs, and the
current implementation in `llm-agent-rag` plus the core compatibility facade

## Locked Inputs

- `llm-agent-rag` is the source of truth for new RAG behavior.
- `llm-agent/rag` remains a compatibility facade and tool adapter, not a second
  implementation path.
- The core `llm-agent` repo must stay stdlib-only and zero-dependency.
- The default standalone store currently exposes `Filters` in its contract but
  does not actually enforce them.
- The standalone ask path currently returns only text, hits, and prompt; it
  does not yet expose first-class citations or retrieval diagnostics.

## Known High-Risk Areas

1. Overloading one `Filters` field for both user-visible filtering and
   mandatory security trimming
2. Adding richer retrieval and answer structures in the standalone SDK without
   breaking the compatibility facade's historical behavior
3. Duplicating filter logic in both repos instead of moving correctness into
   `llm-agent-rag`
4. Introducing contract churn that makes later Phase 9/10 work harder rather
   than easier
5. Returning diagnostics that are not stable or useful enough for later eval
   and feedback-loop phases

## Research Conclusions

### Store Contract

- `store.Query` is the right place to separate caller-controlled filters from
  mandatory security filters.
- The default in-memory store should become the reference implementation for
  filtering behavior instead of leaving correctness to adapter-side
  post-processing.

### Ask / Retrieval Contract

- `rag.Answer` should evolve into a richer envelope with citations and trace
  fields while preserving the existing `Text`, `Hits`, and `Prompt` values.
- Retrieval traces should begin as stable, low-complexity structures:
  query, namespace, effective filters, selected hit IDs, and counts.

### Core Compatibility Layer

- The core wrapper can continue to return plain `string` from `Ask(...)` to
  preserve the old surface.
- Any new detailed standalone behavior should be consumed internally first:
  `Search`, `Ask`, and tool responses can include richer provenance without
  forcing a breaking change in the core package API.

## Research Tasks Deferred Into Execution

- choose the narrowest filter type expansion that supports both optional and
  mandatory filters
- decide whether delete-by-document lands in this phase or stays deferred to
  Phase 9 index lifecycle work
- define the minimum viable citation and trace schema for `Answer`
- determine how the core tool JSON shape should expose citations and
  diagnostics without breaking existing callers

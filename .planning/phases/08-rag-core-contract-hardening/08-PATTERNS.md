# Phase 8: RAG Core Contract Hardening - Pattern Map

**Mapped:** 2026-05-14

## Reuse From Prior Work

- Standalone-first implementation pattern:
  - new behavior lands in `llm-agent-rag`
  - core repo mirrors it through adapters and compatibility wrappers
- Truthful capability pattern:
  - contracts should describe real behavior, not optimistic future intent
- Thin facade pattern:
  - preserve old public API shape where reasonable
  - keep richer internal behavior behind adapters until a deliberate public API
    expansion is planned

## New Patterns to Add

### Dual-filter retrieval pattern

- `store.Query` separates:
  - caller-visible metadata filters
  - mandatory security filters
- stores apply both
- tests prove mandatory filters cannot be bypassed by omission or override

### Provenance-first retrieval pattern

- search hits carry enough stable metadata to explain why a result appeared
- answer generation carries forward hit IDs and citation-ready references
- retrieval traces record effective query inputs and selected outputs

### Rich-answer envelope pattern

- standalone `rag.Answer` remains easy to consume
- new fields are additive:
  - citations
  - diagnostics
  - trace
- prompt rendering remains pluggable through `prompt.Template`

### Core-compat mirror pattern

- `llm-agent/rag` keeps historical `Ask(...) string` behavior
- new standalone diagnostics are consumed internally by:
  - core search wrappers
  - tool responses
  - future compatibility helpers

## Contract Guard Patterns

### Store filter conformance

- assert namespace isolation
- assert caller metadata filter matching
- assert mandatory security filter enforcement
- assert combined-filter behavior remains deterministic

### Answer diagnostics conformance

- assert cited hit IDs are stable
- assert answer diagnostics include enough context for future evaluation
- assert the compatibility facade still passes existing tests

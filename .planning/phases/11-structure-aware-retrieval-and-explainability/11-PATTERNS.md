# Phase 11 Pattern Map

Mapped: 2026-05-14

## Reuse

- additive metadata propagation pattern from Phase 9
- retriever seam pattern from Phase 10
- answer trace / citation extension pattern from Phase 8 and Phase 10

## New Patterns

### Structured chunk pattern

- store normalized structure fields on `StoredChunk`
- keep the original metadata map, but stop relying on it as the only structured
  source of truth

### Structure-aware retriever pattern

- implement section/path matching as another retriever under the existing
  retrieval seam
- allow hybrid retrieval to fuse structure-aware hits with dense/lexical hits

### Explainable prompt pattern

- prompts render section lineage when available
- traces record search path and matched section identifiers
- citations surface section lineage for downstream UIs and audits

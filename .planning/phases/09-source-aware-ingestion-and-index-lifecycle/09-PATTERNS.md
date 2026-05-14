# Phase 9: Source-Aware Ingestion and Index Lifecycle - Pattern Map

**Mapped:** 2026-05-14

## Reuse From Prior Work

- Additive-contract pattern:
  - new fields should preserve simple existing callers
- Splitter metadata propagation pattern:
  - copy document metadata once per chunk
  - enrich with chunk-local metadata like index and total
- Standalone-first implementation pattern:
  - ingestion semantics live in `llm-agent-rag` first

## New Patterns to Add

### Source lineage metadata pattern

- `ingest.Document` owns source identity fields
- chunk metadata receives normalized keys automatically
- stored chunks preserve those keys without additional adapter logic

### Versioned-ingest pattern

- document lineage should express:
  - source identity
  - version identity
  - checksum identity
  - embedding-version identity
- later delete/update flows rely on these keys being present and stable

### Metadata-first lifecycle pattern

- define update and delete semantics in metadata before implementing persistent
  backend mutation helpers
- avoid backend-specific coupling in the core ingest contract

# Phase 9: Source-Aware Ingestion and Index Lifecycle - Research

**Researched:** 2026-05-14
**Status:** Seeded from the Phase 9 roadmap, the standalone RAG design docs,
and the current ingest/import implementation in `llm-agent-rag`

## Locked Inputs

- `ingest.Document` is the narrowest stable place to add source lineage fields.
- Existing chunk metadata is already copied through the splitter and import
  pipeline, which makes additive lineage propagation low-risk.
- Current chunk IDs are deterministic per document and chunk index; this should
  remain true as metadata grows.
- Persistent stores are not in scope yet, so lifecycle work here should focus
  on contracts and metadata rather than backend-specific mutation APIs.

## Known High-Risk Areas

1. Introducing lineage fields in too many places instead of deriving them from
   one source-of-truth document contract
2. Hard-coding metadata keys inconsistently between chunking and store import
3. Adding update semantics that imply backend guarantees the current SDK does
   not yet have
4. Over-designing markdown or source connectors before the metadata contract is
   stable

## Research Conclusions

### Document Contract

- Additive fields on `ingest.Document` should cover:
  - `SourceID`
  - `Version`
  - `Checksum`
  - `EmbeddingVersion`
- Chunk metadata should automatically inherit these into standard metadata keys
  so retrieval and future backends can inspect them without special casing.

### Import Semantics

- `ingest.ImportOptions` is the right place for import-policy flags later, but
  the first slice can focus on metadata propagation and tests.
- The first lifecycle milestone should define metadata for future delete/update
  flows even if explicit delete-by-source helpers land in a later slice.

## Research Tasks Deferred Into Execution

- choose exact metadata key names and keep them consistent
- ensure splitter tests cover metadata inheritance
- decide whether checksum should be caller-supplied only in this slice or gain
  automatic defaulting later

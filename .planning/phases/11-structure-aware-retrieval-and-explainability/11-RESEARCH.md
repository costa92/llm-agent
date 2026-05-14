# Phase 11 Research

Researched: 2026-05-14
Status: execution-seeded

## Locked Inputs

- PageIndex-style retrieval depends on document hierarchy, not only chunk text
- this repo already has markdown-derived path metadata from Phase 9
- the fastest useful step is to turn existing heading/path metadata into
  retrieval primitives before designing a richer tree model

## Conclusions

- `store.StoredChunk` should carry section lineage directly:
  - `SectionID`
  - `SectionPath`
  - `Heading`
  - `HeadingLevel`
- structure-aware retrieval can start as a retriever implementation over the
  existing store contract rather than a new index backend
- explainability should land through retrieval trace, prompt rendering, and
  citations before any advanced planner/routing work

## Deferred Follow-ups

- richer document tree objects and parent/child expansion
- page/range semantics for PDFs or paginated sources
- explicit retrieval planner that chooses dense vs lexical vs structure-aware
  based on query class

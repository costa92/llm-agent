# Phase 11 Context

Phase: 11 — structure-aware retrieval and explainability
Date: 2026-05-14
Primary repo: `llm-agent-rag`

## Why This Phase Exists

Phase 10 completed retrieval policy seams, hybrid recall, MQE/HyDE migration,
reranking, and budget-aware context packing. The next gap is that retrieval is
still mostly flat: chunk IDs and scores exist, but section hierarchy and search
trajectory are not first-class retrieval outputs.

## Inputs

- Phase 9 already added markdown heading-aware splitting and section metadata:
  - `heading`
  - `heading_level`
  - `section_path`
- Phase 10 already introduced:
  - retrieval policy seams
  - hybrid retrieval
  - ask-time trace/citation diagnostics
- production design notes call for:
  - section/path-aware retrieval
  - search trajectory
  - prompts that can cite structure, not only chunk IDs

## Constraints

- standalone-first: implement in `llm-agent-rag` first
- keep additive contracts where possible
- preserve default retrieval behavior for non-structured corpora
- do not introduce a heavy persistent index yet; Phase 12 is the backend phase

## Desired Outcomes

- hierarchical document/section metadata is preserved as first-class stored
  structure, not only loose metadata maps
- retrieval can score section/path cues in addition to dense/lexical content
- ask-time traces and citations expose section lineage and search trajectory

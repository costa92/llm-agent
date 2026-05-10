---
phase: 01-walking-skeleton-generate
plan: 07
subsystem: provider-authoring
tags:
  - docs
  - provider-contract
  - phase-boundary
dependency_graph:
  requires:
    - 01-05
  provides:
    - PROVIDER_AUTHORING.md v0.1
    - canonical Generate-only adapter contract for third parties
  affects:
    - future provider adapters
    - Phase 2 guide expansion
tech_stack:
  added: []
  patterns:
    - top-level contract guide
    - explicit phase-boundary documentation
    - cross-repo example references
key_files:
  created:
    - /home/hellotalk/code/go/src/github.com/costa92/llm-agent/PROVIDER_AUTHORING.md
decisions:
  - "Phase 1 provider guidance is intentionally narrow: document the Generate-only contract now and defer streaming/tools/embeddings until the milestone phases that define them."
  - "The author guide points adapter writers at the sister-repo OpenAI, Anthropic, and Ollama implementations as canonical examples instead of duplicating per-provider wire details in the core repo."
metrics:
  completed: 2026-05-10
  doc_lines: 248
  sections: 8
---

# Phase 1 Plan 07: Provider Author Guide Summary

**One-liner:** Added `PROVIDER_AUTHORING.md` to the core repo, locking the v0.1 Generate-only provider contract, constructor pattern, typed-error taxonomy, conformance pattern, and explicit Phase 1 non-goals.

## Tasks Completed

| # | Name | Key Files |
|---|------|-----------|
| 1 | Write the top-level Provider Author Guide | `PROVIDER_AUTHORING.md` |
| 2 | Document the canonical constructor and typed-error patterns | `PROVIDER_AUTHORING.md` |
| 3 | Bound Phase 1 scope and cross-reference the sister-repo examples | `PROVIDER_AUTHORING.md` |

## Verification Results

- `rg -n '^## ' PROVIDER_AUTHORING.md` — PASS (8 required sections present)
- `wc -l PROVIDER_AUTHORING.md` — PASS (`248`, within the 150-250 target)
- `rg -n '## 5\\. Error Taxonomy|401, 403|WithModel\\(string\\)|01\\. Phase 1 Boundary|Phase 1 Boundary' PROVIDER_AUTHORING.md` — PASS

Document contents include:

- audience and scope
- `llm.ChatModel` contract
- Generate-only response/usage/error expectations
- canonical `New(opts ...Option)` pattern with required `WithModel`
- the Phase 1 HTTP-status to typed-error mapping table
- conformance-harness guidance
- explicit non-goals for streaming, tools, embeddings, retry, and OTel

## What Comes Next

- Phase 1 is ready to close
- Next logical work is Phase 2 planning: streaming on all three providers plus `StreamEvent` validation

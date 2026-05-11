---
phase: 04-embeddings-rag-regression
plan: 03
subsystem: anthropic-documented-gap
tags: [anthropic, embeddings, capability-gap]
metrics:
  completed: 2026-05-11
---

# Phase 4 Plan 03: Anthropic Embedding Gap Summary

Kept Anthropic explicitly non-`Embedder`, verified `Capabilities.Embeddings=false`, added regression coverage proving the adapter does not claim embedding support, and updated provider-facing docs to make the documented gap explicit.

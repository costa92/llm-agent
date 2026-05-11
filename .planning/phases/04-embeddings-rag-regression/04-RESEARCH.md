# Phase 4: Embeddings + RAG Regression - Research

**Researched:** 2026-05-11  
**Status:** Seeded from roadmap + Phase 3 outcomes; provider-specific wire
checks happen during execution

## Locked Inputs

- OpenAI embeddings land in the existing provider adapter rather than a new
  embeddings-only package.
- Ollama embeddings use the provider's native embed endpoint and keep nightly
  live verification separate from PR CI.
- Anthropic explicitly does not implement `llm.Embedder`; the gap is part of
  the contract.
- Shared embedding assertions belong in `internal/contract`.
- `rag/` integration should consume `llm.Embedder` without breaking the v0.2
  API shape.

## Known High-Risk Areas

1. OpenAI model/dimension defaults drifting from the recorded fixture
2. Ollama embedding model variance and dimension discovery
3. Anthropic gap accidentally leaking as a nil panic or undocumented runtime
   failure
4. RAG code depending too tightly on the legacy embedding abstraction
5. Provider guide drifting from actual gap-handling behavior

## Research Tasks Deferred Into Execution

- Confirm current OpenAI embeddings request/response shape in the pinned SDK
- Confirm the current Ollama embed endpoint shape and any model metadata needed
  for dimension assertions
- Inspect `rag/` constructors and tests to choose the narrowest regression seam
- Confirm whether Anthropic adapter should expose an explicit `Embed` method
  returning `ErrCapabilityNotSupported` or rely purely on non-implementation +
  helper docs

# Phase 4: Embeddings + RAG Regression - Pattern Map

**Mapped:** 2026-05-11

## Reuse From Prior Phases

- Provider package layout stays unchanged: `openai/`, `anthropic/`, `ollama/`
- Shared conformance remains in `internal/contract/`
- `ProviderInfo.Capabilities` remains the truthful runtime capability surface
- Nightly Ollama live validation stays isolated from fast fixture CI

## New Patterns to Add

### Embedder capability pattern

- Real embedding providers implement `llm.Embedder`
- Batch input order is preserved exactly in output vectors
- Usage returns through the shared `llm.Usage` type

### Documented-gap pattern

- Unsupported capability returns `llm.ErrCapabilityNotSupported`
- Error text names the provider/model when possible
- Provider docs and conformance both treat the gap as expected behavior

### Embedding conformance pattern

- Assert vector count equals input count
- Assert dimensions are stable per provider/model fixture
- Assert Anthropic gap via `errors.Is(..., llm.ErrCapabilityNotSupported)`

### RAG regression pattern

- Reuse existing fixture corpus and retrieval assertions
- Swap in a real `llm.Embedder` via the narrowest constructor seam
- Preserve public API shape even if internals gain v0.3 capability plumbing

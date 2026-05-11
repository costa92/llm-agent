# Phase 4: Embeddings + RAG Regression - Context

**Gathered:** 2026-05-11  
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 4 closes the provider walking skeleton by landing embeddings on OpenAI
and Ollama, documenting Anthropic's deliberate gap, extending shared
conformance, and proving the existing `rag/` surface can consume a real
`llm.Embedder`.

This phase produces:

- `llm-agent-providers/openai/` `Embedder` support
- `llm-agent-providers/ollama/` `Embedder` support
- `llm-agent-providers/anthropic/` explicit non-support path with
  `llm.ErrCapabilityNotSupported`
- `llm-agent-providers/internal/contract/` embedding conformance
- `llm-agent/rag/` regression coverage against the new capability surface
- `llm-agent/PROVIDER_AUTHORING.md` guidance for the documented-gap pattern

Phase 4 explicitly covers:

- batch embeddings in input order
- dimension reporting and assertion
- truthful `ProviderInfo.Capabilities.Embeddings`
- Anthropic's documented gap as a first-class contract outcome
- `rag.RAGSystem` compatibility with `llm.Embedder`

Phase 4 does NOT cover:

- structured outputs
- OTel wrappers
- reference service integration
- vector store backends beyond the existing in-memory implementation

</domain>

<decisions>
## Implementation Decisions

### D-01: Anthropic gap stays explicit

- Anthropic does not grow a fake or degraded embedding implementation.
- Missing capability is expressed through `ProviderInfo.Capabilities.Embeddings`
  plus `llm.ErrCapabilityNotSupported`.

### D-02: Conformance remains shared

- Embeddings extend the existing `internal/contract` harness.
- There is no provider-specific embedding test framework.

### D-03: RAG surface stays back-compat

- Existing `rag/` API shape in this repo does not break.
- New tests prove that a real `llm.Embedder` can back indexing and retrieval
  without reworking the public RAG abstraction.

### D-04: Dimensions are contract data

- OpenAI uses the documented embedding model dimension as an assertion target.
- Ollama observed dimensions are asserted against the bound model metadata or
  captured fixture.
- Conformance treats dimension drift as a first-class failure.

</decisions>

<canonical_refs>
## Canonical References

- `.planning/ROADMAP.md` - Phase 4 scope, success criteria, and plan ordering
- `.planning/REQUIREMENTS.md` - `OAI-04`, `ANT-04`, `OLL-04`, `CONF-06`
- `.planning/STATE.md` - project position after Phase 3 completion
- `.planning/phases/03-native-tool-calling-agent-refactor/03-05-SUMMARY.md`
- `/tmp/llm-agent-providers/openai/`
- `/tmp/llm-agent-providers/anthropic/`
- `/tmp/llm-agent-providers/ollama/`
- `/tmp/llm-agent-providers/internal/contract/`
- `rag/`
- `PROVIDER_AUTHORING.md`

</canonical_refs>

<specifics>
## Success Markers to Preserve

- Capability honesty remains per bound model, not per provider brand.
- OpenAI and Ollama embeddings slot into the same `llm.Embedder` contract.
- Anthropic's non-support is observable and documented, not hidden.
- `rag/` remains stdlib-only and source-compatible from the user's perspective.

</specifics>

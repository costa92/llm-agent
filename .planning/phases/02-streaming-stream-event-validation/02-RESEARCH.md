# Phase 2: Streaming on All 3 Providers + StreamEvent Validation - Research

**Researched:** 2026-05-10  
**Status:** Seeded from roadmap + Phase 1 outcomes; provider-specific deep dives happen during execution

## Locked Inputs

- OpenAI: `openai-go/v3` remains the SDK baseline from Phase 1
- Anthropic: `anthropic-sdk-go` remains the SDK baseline from Phase 1
- Ollama: `ollama/api` remains the SDK baseline from Phase 1
- Shared harness: extend the existing `internal/contract` fixture system rather
  than introducing a second streaming-specific framework

## Known High-Risk Areas

1. OpenAI per-tool-call `index` keying under interleaved deltas
2. Anthropic `partial_json` buffering discipline
3. Ollama callback-to-iterator cancellation and leak behavior
4. Pre-first-byte retry vs post-first-byte no-retry boundary
5. Usage accounting on incomplete streams

## Research Tasks Deferred Into Execution

- Confirm current OpenAI stream request shape for `stream_options.include_usage`
- Confirm current Anthropic accumulate helper behavior for mixed text/tool-use blocks
- Confirm exact Ollama callback error propagation semantics
- Record per-provider notes in implementation comments and tests as they are verified

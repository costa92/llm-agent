# Phase 2: Streaming on All 3 Providers + StreamEvent Validation - Pattern Map

**Mapped:** 2026-05-10

## Reuse From Phase 1

- Provider package layout stays unchanged: `openai/`, `anthropic/`, `ollama/`
- Shared conformance remains in `internal/contract/`
- Request/response mapping helpers stay provider-local
- Error taxonomy remains the same typed-error surface from `llm/errors.go`

## New Patterns to Add

### StreamReader adapter pattern

- Provider-local stream implementation owns transport-specific buffering
- Shared outward contract is `StreamReader.Next() (StreamEvent, error)`
- Cancellation must be observed by both the underlying SDK and the outward iterator

### Event-assembly pattern

- OpenAI: accumulate by streamed delta and per-call `index`
- Anthropic: accumulate by content-block `index`
- Ollama: adapt callback chunks to `TextDelta` and terminal `Done`

### Retry-state-machine pattern

- Explicit internal state transitions
- Retry only in `Connecting`
- No retry after transition to `FirstByte` or `Streaming`

### Streaming conformance pattern

- Fixture-driven replay for provider wire formats
- Cancel-mid-stream tests
- Partial-usage-on-error tests
- `goleak.VerifyTestMain` remains active at suite level

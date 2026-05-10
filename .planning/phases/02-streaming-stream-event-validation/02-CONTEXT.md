# Phase 2: Streaming on All 3 Providers + StreamEvent Validation - Context

**Gathered:** 2026-05-10  
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 2 adds real `ChatModel.Stream` implementations to all three provider
adapters and extends the shared conformance suite to validate `StreamEvent`
behavior, cancel semantics, retry boundaries, and partial-usage handling.

This phase produces:

- `llm-agent-providers/openai/` streaming over `openai-go/v3`
- `llm-agent-providers/anthropic/` streaming over `anthropic-sdk-go`
- `llm-agent-providers/ollama/` streaming over `ollama/api`
- `llm-agent-providers/internal/contract/` streaming conformance harness
- `PROVIDER_AUTHORING.md` v0.2 updates deferred until the execution wave that
  actually lands streaming behavior in the core repo

Phase 2 explicitly covers:

- `StreamReader.Next()` semantics
- `StreamEvent` ordering and assembly
- cost-record `Usage.Source` behavior on clean completion vs aborted stream
- retry state machine `Connecting -> FirstByte -> Streaming -> Done`
- no goroutine leaks on cancel or early error

Phase 2 does NOT cover:

- native tool calling behavior beyond preserving infrastructure needed for
  future indexed tool-call deltas
- agent refactors
- embeddings
- OTel emission

</domain>

<decisions>
## Implementation Decisions

### D-01: OpenAI streaming must force usage emission

- All OpenAI streaming requests include `stream_options.include_usage = true`.
- This is required to satisfy `OAI-06` and is asserted in conformance tests by
  inspecting the outbound request body.

### D-02: Anthropic partial JSON is finalized on content-block stop

- `partial_json` deltas are buffered per content-block `index`.
- JSON decode happens only when the matching `content_block_stop` arrives.
- `message_stop` is not a flush boundary.

### D-03: Ollama callback stream is bridged into the shared iterator contract

- The Ollama SDK's callback-driven stream is adapted into the existing
  `StreamReader` contract with explicit `ctx.Done()` propagation.
- Cancelled streams must unwind promptly and leak no goroutines.

### D-04: Retry is pre-first-byte only

- A stream that fails before delivering the first event may retry once under
  default policy.
- A stream that fails after the first byte never retries.
- This state machine must be unit-tested per adapter.

### D-05: Partial usage is represented as unknown, never zero-reported

- Clean stream completion returns `Usage.Source = reported` when the provider
  yields counts.
- Aborted streams return `Usage.Source = unknown`.
- Never emit `reported` with zero tokens when the truth is unavailable.

</decisions>

<canonical_refs>
## Canonical References

- `.planning/ROADMAP.md` - Phase 2 scope, pitfalls, and success criteria
- `.planning/REQUIREMENTS.md` - `CONF-03`, `OAI-02/06/07`, `ANT-02/06`, `OLL-02/06`
- `.planning/STATE.md` - current milestone position
- `.planning/phases/01-walking-skeleton-generate/01-05-SUMMARY.md` - existing shared conformance harness
- `/tmp/llm-agent-providers/openai/`
- `/tmp/llm-agent-providers/anthropic/`
- `/tmp/llm-agent-providers/ollama/`
- `/tmp/llm-agent-providers/internal/contract/`
- `llm/stream.go`
- `llm/types.go`

</canonical_refs>

<specifics>
## Success Markers to Preserve

- OpenAI tool-call index keying infrastructure must be stream-safe even before
  full tool-calling lands in Phase 3.
- Anthropic multi-block streaming must remain index-stable across text and
  future tool-use blocks.
- Ollama streaming must remain local-first and Docker-testable in nightly CI.
- `goleak` remains mandatory in the shared conformance suite.

</specifics>

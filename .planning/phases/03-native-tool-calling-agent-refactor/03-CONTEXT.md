# Phase 3: Native Tool Calling + Agent Refactor - Context

**Gathered:** 2026-05-11  
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 3 adds native tool-calling support to all three provider adapters and
refactors the core agent constructors so they consume the new `ChatModel`
capability surface directly.

This phase produces:

- `llm-agent-providers/openai/` native tool-calling over `openai-go/v3`
- `llm-agent-providers/anthropic/` native tool-calling over
  `anthropic-sdk-go`
- `llm-agent-providers/ollama/` native tool-calling with a per-model strategy
  table
- `llm-agent-providers/internal/contract/` tool-calling conformance harness
- `llm-agent/agent/` constructors that branch on `ToolCaller` capability rather
  than provider-specific assumptions

Phase 3 explicitly covers:

- immutable `WithTools(...)` binding semantics
- provider-local tool-call parsing and tool result re-entry
- agent-layer dedupe on `(message_id, tool_use_id)`
- capability-degrade behavior when a bound model does not support native tools
- constructor-time validation vs runtime fallback across agent paradigms

Phase 3 does NOT cover:

- embeddings
- OTel wrappers
- reference customer-support service integration
- broader prompt-engineering or planner behavior changes unrelated to tools

</domain>

<decisions>
## Implementation Decisions

### D-01: `WithTools` is immutable on every provider

- Binding tools always returns a NEW adapter value.
- The base model instance is never mutated.
- Tests must verify concurrent use of the same base model with different tool
  sets.

### D-02: Agent capability checks are explicit and honest

- Agents that can degrade (`ReAct`, `Reflection`, `PlanSolve`) inspect
  `ToolCaller` and choose native tools only when present.
- Agents that require native tools (`FunctionCall`) fail at construction time
  with a clear error naming the bound model.

### D-03: Dedupe is keyed on `(message_id, tool_use_id)`

- Provider adapters surface enough identity for the agent layer to reject
  duplicate tool dispatch.
- This is the second line of defense after Phase 2's no-retry-after-first-byte
  stream state machine.

### D-04: Anthropic block parsing is index-stable and block-local

- Multiple `tool_use` content blocks are parsed independently.
- Parsing one block must never overwrite or flush another block's partial
  state.

### D-05: Ollama tool support is model-strategy driven

- Tool calling on Ollama is controlled by a per-model strategy table.
- Unsupported models fail with a capability-degrade error referencing
  `ProviderInfo.Capabilities.ToolCaller=false`.

</decisions>

<canonical_refs>
## Canonical References

- `.planning/ROADMAP.md` - Phase 3 scope, pitfalls, and success criteria
- `.planning/REQUIREMENTS.md` - `CONF-04`, `OAI-03`, `ANT-03`, `OLL-03`,
  `CORE-05`
- `.planning/STATE.md` - current milestone position
- `.planning/phases/02-streaming-stream-event-validation/02-04-SUMMARY.md` -
  streaming contract gate now complete
- `/tmp/llm-agent-providers/openai/`
- `/tmp/llm-agent-providers/anthropic/`
- `/tmp/llm-agent-providers/ollama/`
- `/tmp/llm-agent-providers/internal/contract/`
- `llm/model.go`
- `llm/types.go`
- `agent/`

</canonical_refs>

<specifics>
## Success Markers to Preserve

- Phase 2 streaming invariants remain intact while tool calling is layered on
  top.
- `ProviderInfo.Capabilities` remains truthful per bound model, not per
  provider brand.
- The same tool-calling calculator scenario is reusable across all three
  adapters.
- Core agents stay stdlib-only and provider-agnostic.

</specifics>

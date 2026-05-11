# Phase 3: Native Tool Calling + Agent Refactor - Pattern Map

**Mapped:** 2026-05-11

## Reuse From Phase 2

- Provider package layout stays unchanged: `openai/`, `anthropic/`, `ollama/`
- Shared conformance remains in `internal/contract/`
- Streaming assembly and retry guards remain provider-local
- Capability truth still hangs off `ProviderInfo` on the bound model instance

## New Patterns to Add

### Immutable tool-binding pattern

- `WithTools(...)` returns a shallow-cloned adapter carrying the tool schema
- Base adapters stay reusable across goroutines and tool sets
- Tests assert no mutation of the receiver

### Provider-local tool-call assembly pattern

- OpenAI: accumulate tool calls by streamed/provider `index`
- Anthropic: accumulate by content-block `index`
- Ollama: parse by model-family strategy selected at construction time

### Capability-degrade pattern

- Unsupported native tool calling returns a typed error
- Error text names the bound model and points at
  `ProviderInfo.Capabilities.ToolCaller=false`
- Agents that can degrade fall back to scratchpad prompting explicitly

### Agent constructor pattern

- Constructors accept `llm.ChatModel`
- Native-tool agents use type assertions for `ToolCaller`
- Fallback-capable agents choose the native or scratchpad path once at
  construction time, not on every loop iteration

### Tool-calling conformance pattern

- Shared calculator-tool scenario across providers
- Parallel tool-call fixtures for OpenAI and Anthropic
- Capability-degrade coverage for unsupported Ollama models
- Dedupe coverage for repeated `(message_id, tool_use_id)`

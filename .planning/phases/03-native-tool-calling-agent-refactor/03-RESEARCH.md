# Phase 3: Native Tool Calling + Agent Refactor - Research

**Researched:** 2026-05-11  
**Status:** Seeded from roadmap + Phase 2 outcomes; provider-specific deep
dives happen during execution

## Locked Inputs

- OpenAI: extend the existing `openai-go/v3` adapter rather than introducing a
  Responses-vs-Chat split in v0.3
- Anthropic: extend the existing `anthropic-sdk-go` adapter and explicitly
  survey `BetaToolRunner` before choosing the lower-level event surface
- Ollama: extend the existing `ollama/api` adapter with model-specific tool
  strategies rather than assuming one universal wire format
- Shared harness: extend `internal/contract` again rather than adding a second
  tool-only test framework
- Core: agent constructors consume `llm.ChatModel` plus optional capability
  interfaces via type assertions

## Known High-Risk Areas

1. OpenAI parallel tool-call index keying under interleaved deltas
2. Anthropic multi-block `tool_use` parse independence
3. Ollama per-model strategy drift across model families
4. Duplicate tool dispatch after retry or replay
5. Agent fallback behavior drifting between constructors

## Research Tasks Deferred Into Execution

- Confirm current OpenAI tool-call request and streaming delta shapes used by
  the bound SDK version
- Confirm whether Anthropic `BetaToolRunner` helps or obscures the required
  adapter contract
- Survey current Ollama model families already exercised in upstream issues and
  docs to seed the first strategy table
- Inspect current `agent/` constructors to identify where native-tool fast
  paths can land without widening public APIs

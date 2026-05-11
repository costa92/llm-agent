---
phase: 03-native-tool-calling-agent-refactor
plan: 04
subsystem: tool-calling-conformance
tags:
  - conformance
  - tool-calling
  - capability-degrade
  - dedupe
dependency_graph:
  requires:
    - 03-01
    - 03-02
    - 03-03
  provides:
    - shared tool-calling conformance matrix
    - cross-provider calculator scenario
    - capability-degrade and dedupe coverage
  affects:
    - 03-05 core agent refactor
    - Phase 3 close criteria
tech_stack:
  added: []
  patterns:
    - fixture-driven tool-call replay
    - shared WithTools helper at contract layer
    - provider-agnostic tool-call JSON assertions
key_files:
  modified:
    - /tmp/llm-agent-providers/internal/contract/contract.go
    - /tmp/llm-agent-providers/internal/contract/generate_test.go
  created:
    - /tmp/llm-agent-providers/internal/contract/testdata/openai/tool_happy_gpt-4o-mini.json
    - /tmp/llm-agent-providers/internal/contract/testdata/openai/tool_parallel_gpt-4o-mini.json
    - /tmp/llm-agent-providers/internal/contract/testdata/anthropic/tool_happy_claude-3-5-haiku.json
    - /tmp/llm-agent-providers/internal/contract/testdata/anthropic/tool_multiblock_claude-3-5-haiku.json
    - /tmp/llm-agent-providers/internal/contract/testdata/ollama/tool_happy_llama3.1-8b.json
decisions:
  - "Shared tool-calling conformance extends the existing Fixture schema instead of creating a second tool-only harness."
  - "Capability-degrade is tested at the `WithTools(...)` boundary for unsupported Ollama models and must wrap `llm.ErrCapabilityNotSupported`."
  - "Dedupe on `(message_id, tool_use_id)` is enforced in contract tests as a second-line invariant independent of provider transport behavior."
metrics:
  completed: 2026-05-11
  tool_fixtures: 5
---

# Phase 3 Plan 04: Tool-Calling Conformance Summary

**One-liner:** Extended the shared `internal/contract` harness to cover native tool calling across OpenAI, Anthropic, and Ollama, including the shared calculator scenario, parallel/multi-block tool use, unsupported-model capability-degrade, and dedupe-key behavior.

## Tasks Completed

| # | Name | Key Files |
|---|------|-----------|
| 1 | Extend Fixture schema and add `AssertToolCalling` helper | `internal/contract/contract.go` |
| 2 | Add shared tool-calling matrix and dedupe/capability tests | `internal/contract/generate_test.go` |
| 3 | Add provider fixtures for happy-path, parallel, and multi-block tool calls | `internal/contract/testdata/*/tool_*.json` |

## Verification Results

- `GOCACHE=/tmp/go-build go test ./internal/contract/... -count=1` — PASS
- `GOCACHE=/tmp/go-build go vet ./internal/contract/...` — PASS
- `git diff --check` in `/tmp/llm-agent-providers` — pending final commit check

Shared tool-calling coverage now includes:

- shared calculator tool scenario across all three providers
- OpenAI parallel tool-call fixture
- Anthropic multi-block `tool_use` fixture
- Ollama unsupported-model capability-degrade via `llm.ErrCapabilityNotSupported`
- dedupe behavior keyed on `(message_id, tool_use_id)`

## What Comes Next

- `03-05`: core agent capability refactor
- Phase 3 closeout after agent constructors consume `ChatModel` + `ToolCaller`

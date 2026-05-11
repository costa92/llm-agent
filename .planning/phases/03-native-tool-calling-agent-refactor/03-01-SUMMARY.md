---
phase: 03-native-tool-calling-agent-refactor
plan: 01
subsystem: openai-native-tools
tags:
  - openai
  - tool-calling
  - immutable-binding
  - streaming
dependency_graph:
  requires: []
  provides:
    - OpenAI ToolCaller implementation
    - immutable WithTools binding
    - Generate and Stream tool-call mapping
  affects:
    - 03-04 shared tool-calling conformance
    - 03-05 core agent capability refactor
tech_stack:
  added: []
  patterns:
    - immutable adapter cloning
    - per-index streamed tool-call assembly
    - provider response to llm.ToolCall mapping
key_files:
  modified:
    - /tmp/llm-agent-providers/openai/openai.go
    - /tmp/llm-agent-providers/openai/map.go
    - /tmp/llm-agent-providers/openai/options.go
    - /tmp/llm-agent-providers/openai/openai_test.go
decisions:
  - "OpenAI advertises `Capabilities.Tools=true` at construction time because tool support is a property of the bound model, not of whether tools are currently attached."
  - "WithTools clones the adapter and copies the bound tool slice, preserving concurrent-safe reuse of the base model."
  - "Streaming tool calls emit `EventToolCallEnd` when finish reason becomes `tool_calls`, ordered by stable provider index."
metrics:
  completed: 2026-05-11
  files_modified: 4
---

# Phase 3 Plan 01: OpenAI Native Tool Calling Summary

**One-liner:** Implemented OpenAI native tool calling with immutable `WithTools(...)`, response `ToolCalls` mapping for both sync and streaming paths, and concurrent-binding tests that prove the base adapter is not mutated.

## Tasks Completed

| # | Name | Key Files |
|---|------|-----------|
| 1 | Add `ToolCaller` capability and immutable `WithTools(...)` | `openai/openai.go`, `openai/options.go` |
| 2 | Map OpenAI function tools into chat-completions request payloads | `openai/map.go` |
| 3 | Map sync and streaming tool-call responses into `llm.ToolCall` / `StreamEvent` | `openai/map.go`, `openai/openai.go` |
| 4 | Add tests for capability truth, immutability, sync tool calls, and streamed indexed tool calls | `openai/openai_test.go` |

## Verification Results

- `GOCACHE=/tmp/go-build go test ./openai/... -count=1` — PASS
- `GOCACHE=/tmp/go-build go build ./openai/...` — PASS
- `git diff --check` in `/tmp/llm-agent-providers` — pending final commit check

Covered tool-calling scenarios:

- `Info().Capabilities.Tools == true` on the bound OpenAI model
- `WithTools(...)` returns distinct bound values without cross-request tool leakage
- sync `Generate()` maps `finish_reason=tool_calls` plus multiple `tool_calls`
- streaming emits per-index start / args-delta / end events and preserves final usage

## What Comes Next

- `03-02`: Anthropic native tool calling
- `03-03`: Ollama native tool calling + strategy table
- `03-04`: shared tool-calling conformance after all three providers land

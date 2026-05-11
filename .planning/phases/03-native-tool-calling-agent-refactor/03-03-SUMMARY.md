---
phase: 03-native-tool-calling-agent-refactor
plan: 03
subsystem: ollama-native-tools
tags:
  - ollama
  - tool-calling
  - strategy-table
  - capability-degrade
dependency_graph:
  requires: []
  provides:
    - Ollama ToolCaller implementation
    - per-model strategy table
    - native plus fallback tool-call parsing
  affects:
    - 03-04 shared tool-calling conformance
    - 03-05 core agent capability refactor
tech_stack:
  added: []
  patterns:
    - capability truth from model-family strategy
    - immutable adapter cloning
    - parser fallback by model family
key_files:
  modified:
    - /tmp/llm-agent-providers/ollama/ollama.go
    - /tmp/llm-agent-providers/ollama/map.go
    - /tmp/llm-agent-providers/ollama/options.go
    - /tmp/llm-agent-providers/ollama/ollama_test.go
  created:
    - /tmp/llm-agent-providers/ollama/tool_strategy.go
decisions:
  - "Ollama tool support is selected by a local per-model strategy table instead of assuming every Ollama model supports native tools."
  - "Unsupported models fail from `WithTools(...)` with `llm.ErrCapabilityNotSupported` and an explicit `ProviderInfo.Capabilities.Tools=false` hint."
  - "The adapter prefers structured `message.tool_calls` from the Ollama API, but falls back to model-family parsers for `llama3.1` (`<|python_tag|>`) and `qwen3-coder` (`<tool_call>...</tool_call>`)."
metrics:
  completed: 2026-05-11
  files_modified: 5
---

# Phase 3 Plan 03: Ollama Native Tool Calling Summary

**One-liner:** Implemented Ollama native tool calling with a per-model strategy table, honest capability-degrade behavior for unsupported models, and fallback parsing paths for `llama3.1` and `qwen3-coder`.

## Tasks Completed

| # | Name | Key Files |
|---|------|-----------|
| 1 | Add `ToolCaller` capability and immutable `WithTools(...)` | `ollama/ollama.go`, `ollama/options.go` |
| 2 | Add per-model strategy table and capability-degrade error path | `ollama/tool_strategy.go`, `ollama/options.go` |
| 3 | Inject Ollama tool schema into chat requests | `ollama/map.go` |
| 4 | Map native `tool_calls` plus model-family fallback parsers into `llm.ToolCall` | `ollama/map.go`, `ollama/tool_strategy.go` |
| 5 | Add tests for capability truth, unsupported-model failure, native tool calls, and qwen XML fallback | `ollama/ollama_test.go` |

## Verification Results

- `GOCACHE=/tmp/go-build go test ./ollama/... -count=1` — PASS
- `GOCACHE=/tmp/go-build go build ./ollama/...` — PASS
- `git diff --check` in `/tmp/llm-agent-providers` — pending final commit check

Covered tool-calling scenarios:

- `llama3.1:8b` reports `Capabilities.Tools == true`
- `llama2` reports `Capabilities.Tools == false` and `WithTools(...)` fails with `ErrCapabilityNotSupported`
- outbound request body includes Ollama `tools` definitions
- native `message.tool_calls` map to `llm.ToolCall`
- `qwen3-coder` fallback parser extracts `<tool_call>{...}</tool_call>` payloads when structured tool calls are absent

## What Comes Next

- `03-04`: shared tool-calling conformance
- `03-05`: core agent capability refactor

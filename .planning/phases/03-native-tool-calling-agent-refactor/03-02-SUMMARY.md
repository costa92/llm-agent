---
phase: 03-native-tool-calling-agent-refactor
plan: 02
subsystem: anthropic-native-tools
tags:
  - anthropic
  - tool-calling
  - immutable-binding
  - multi-block
dependency_graph:
  requires: []
  provides:
    - Anthropic ToolCaller implementation
    - tool_use response mapping
    - immutable WithTools binding
  affects:
    - 03-04 shared tool-calling conformance
    - 03-05 core agent capability refactor
tech_stack:
  added: []
  patterns:
    - immutable adapter cloning
    - block-local tool_use parsing
    - direct Messages API over BetaToolRunner
key_files:
  modified:
    - /tmp/llm-agent-providers/anthropic/anthropic.go
    - /tmp/llm-agent-providers/anthropic/map.go
    - /tmp/llm-agent-providers/anthropic/options.go
    - /tmp/llm-agent-providers/anthropic/anthropic_test.go
decisions:
  - "Anthropic native tools stay on the low-level Messages API; SDK BetaToolRunner is intentionally not used because it owns the conversation loop and is not concurrent-safe."
  - "WithTools clones the adapter and copies the tool slice so separate bindings cannot leak into each other."
  - "Sync responses map every `tool_use` block to an `llm.ToolCall`, preserving independent multi-block parsing already established in streaming."
metrics:
  completed: 2026-05-11
  files_modified: 4
---

# Phase 3 Plan 02: Anthropic Native Tool Calling Summary

**One-liner:** Implemented Anthropic native tool calling with immutable `WithTools(...)`, request-side tool/schema injection, sync `tool_use` mapping into `llm.ToolCall`, and tests covering independent multi-block parsing.

## Tasks Completed

| # | Name | Key Files |
|---|------|-----------|
| 1 | Add `ToolCaller` capability and immutable `WithTools(...)` | `anthropic/anthropic.go`, `anthropic/options.go` |
| 2 | Inject Anthropic tool schema and `tool_choice:auto` into message requests | `anthropic/map.go` |
| 3 | Map sync `tool_use` blocks into `llm.ToolCall` values | `anthropic/map.go` |
| 4 | Add tests for capability truth, immutable bindings, request shape, and multi-block tool use | `anthropic/anthropic_test.go` |

## Verification Results

- `GOCACHE=/tmp/go-build go test ./anthropic/... -count=1` — PASS
- `GOCACHE=/tmp/go-build go build ./anthropic/...` — PASS
- `git diff --check` in `/tmp/llm-agent-providers` — pending final commit check

Covered tool-calling scenarios:

- `Info().Capabilities.Tools == true` on the bound Anthropic model
- `WithTools(...)` returns isolated values with no cross-request tool leakage
- outbound request body includes `tools` plus `tool_choice:auto`
- sync `Generate()` maps multiple `tool_use` blocks to independent `llm.ToolCall` values
- streaming multi-block parsing from Phase 2 remains intact

## What Comes Next

- `03-03`: Ollama native tool calling + strategy table
- `03-04`: shared tool-calling conformance after all three providers land

---
phase: 03-native-tool-calling-agent-refactor
plan: 05
subsystem: core-agent-capability-refactor
tags:
  - agents
  - chatmodel
  - capability-negotiation
  - tool-calling
dependency_graph:
  requires:
    - 03-01
    - 03-02
    - 03-03
    - 03-04
  provides:
    - ChatModel-based core agent constructors
    - ReAct native-tools fast path with scratchpad fallback
    - FunctionCall fast-fail capability gate
  affects:
    - Phase 3 close criteria
    - Phase 4 planning entry point
tech_stack:
  added: []
  patterns:
    - ChatModel prompt shim helper
    - type assertion plus Capabilities.Tools negotiation
    - constructor-time native-vs-fallback binding
key_files:
  modified:
    - /home/hellotalk/code/go/src/github.com/costa92/llm-agent/simple.go
    - /home/hellotalk/code/go/src/github.com/costa92/llm-agent/reflection.go
    - /home/hellotalk/code/go/src/github.com/costa92/llm-agent/plan_solve.go
    - /home/hellotalk/code/go/src/github.com/costa92/llm-agent/react.go
    - /home/hellotalk/code/go/src/github.com/costa92/llm-agent/function_call.go
    - /home/hellotalk/code/go/src/github.com/costa92/llm-agent/scriptedllm_test.go
    - /home/hellotalk/code/go/src/github.com/costa92/llm-agent/function_call_test.go
    - /home/hellotalk/code/go/src/github.com/costa92/llm-agent/react_test.go
    - /home/hellotalk/code/go/src/github.com/costa92/llm-agent/example_tool_use_test.go
    - /home/hellotalk/code/go/src/github.com/costa92/llm-agent/example_multi_agent_test.go
    - /home/hellotalk/code/go/src/github.com/costa92/llm-agent/examples/scriptedllm/scriptedllm.go
    - /home/hellotalk/code/go/src/github.com/costa92/llm-agent/examples/02-tool-use/main.go
  created:
    - /home/hellotalk/code/go/src/github.com/costa92/llm-agent/agent_chatmodel.go
decisions:
  - "Core agent paradigms now depend on llm.ChatModel; only tool-aware agents negotiate llm.ToolCaller."
  - "ReAct chooses native tools once at construction time when both type assertion and Capabilities.Tools are true; otherwise it preserves the scratchpad loop."
  - "FunctionCallAgent is native-only and now fails at construction with provider/model context plus llm.ErrCapabilityNotSupported when tools are unavailable."
metrics:
  completed: 2026-05-11
  tests_added: 3
---

# Phase 3 Plan 05: Core Agent Capability Refactor Summary

**One-liner:** Migrated the core agent constructors onto `llm.ChatModel`, added constructor-time capability negotiation for native tool use, preserved scratchpad fallback for `ReAct`, and made `FunctionCallAgent` fail fast when native tools are unavailable.

## Tasks Completed

| # | Name | Key Files |
|---|------|-----------|
| 1 | Add shared ChatModel prompt/capability helpers | `agent_chatmodel.go` |
| 2 | Refactor `Simple`, `Reflection`, and `PlanAndSolve` onto `llm.ChatModel` | `simple.go`, `reflection.go`, `plan_solve.go` |
| 3 | Add `ReAct` native-tool fast path with scratchpad fallback | `react.go`, `react_test.go` |
| 4 | Make `FunctionCallAgent` native-only with constructor-time fast-fail | `function_call.go`, `function_call_test.go` |
| 5 | Update local scripted examples/tests to compile on the new surface | `scriptedllm_test.go`, `example_*`, `examples/scriptedllm` |

## Verification Results

- `GOCACHE=/tmp/go-build go test ./...` — PASS
- `GOCACHE=/tmp/go-build go build ./...` — PASS
- `git diff --check` — pending final commit check

Key regressions now covered:

- `FunctionCallAgent` returns `llm.ErrCapabilityNotSupported` with provider/model context when bound to a chat-only model.
- `ReAct` uses native tool calls when the model satisfies `ToolCaller` and `Capabilities.Tools`.
- `ReAct` falls back cleanly to scratchpad parsing when native tool capability is absent.

## What Comes Next

- Phase 3 is complete.
- Next logical step is Phase 4 planning and execution.

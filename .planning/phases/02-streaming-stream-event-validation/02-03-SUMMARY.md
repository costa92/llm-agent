---
phase: 02-streaming-stream-event-validation
plan: 03
subsystem: ollama-streaming
tags:
  - ollama
  - streaming
  - cancellation
  - callback-bridge
dependency_graph:
  requires: []
  provides:
    - Ollama Stream implementation
    - callback-to-iterator bridge
    - ctx-cancel propagation for local streaming
  affects:
    - 02-04 shared streaming conformance
    - 01-06 nightly live CI usefulness in later phases
tech_stack:
  added: []
  patterns:
    - callback stream bridged via channels
    - ctx-first cancellation semantics
    - final done-response to EventDone mapping
key_files:
  modified:
    - /tmp/llm-agent-providers/ollama/map.go
    - /tmp/llm-agent-providers/ollama/ollama.go
    - /tmp/llm-agent-providers/ollama/ollama_test.go
decisions:
  - "Ollama streaming uses the existing callback API and bridges it into `StreamReader` with channels instead of introducing a transport-specific iterator."
  - "After `ctx` cancellation, `Next()` checks `ctx.Err()` before draining any buffered response so cancellation wins over late-arriving chunks."
  - "Normal stream completion emits `EventDone` from the final `done=true` callback with reported prompt/eval counts."
metrics:
  completed: 2026-05-10
  files_modified: 3
---

# Phase 2 Plan 03: Ollama Streaming Summary

**One-liner:** Implemented Ollama `Stream()` by bridging the SDK callback stream into the shared iterator contract, with proper `stream=true` requests, `EventDone` mapping, and prompt cancellation semantics.

## Tasks Completed

| # | Name | Key Files |
|---|------|-----------|
| 1 | Add streaming request shape for Ollama | `ollama/map.go` |
| 2 | Add callback-to-iterator stream reader | `ollama/ollama.go` |
| 3 | Add streaming tests for happy path and cancel-mid-stream | `ollama/ollama_test.go` |

## Verification Results

- `GOCACHE=/tmp/go-build go test ./ollama/... -count=1` — PASS
- `GOCACHE=/tmp/go-build go build ./ollama/...` — PASS
- `git diff --check` in `/tmp/llm-agent-providers` — PASS

Covered stream scenarios:

- request body flips to `stream=true`
- text deltas accumulate to `hello`
- final `done=true` callback maps to `EventDone` with reported usage
- cancel-mid-stream returns `context.Canceled` quickly instead of leaking buffered chunks

## What Comes Next

- `02-04`: shared streaming conformance across OpenAI, Anthropic, and Ollama

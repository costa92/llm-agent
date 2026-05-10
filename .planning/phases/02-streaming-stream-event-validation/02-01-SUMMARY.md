---
phase: 02-streaming-stream-event-validation
plan: 01
subsystem: openai-streaming
tags:
  - openai
  - streaming
  - retry-state-machine
  - usage
dependency_graph:
  requires: []
  provides:
    - OpenAI Stream implementation
    - include_usage request enforcement
    - pre-first-byte retry boundary
  affects:
    - 02-04 shared streaming conformance
tech_stack:
  added: []
  patterns:
    - iterator-style stream reader
    - queue-based event emission
    - pre-first-byte-only retry
key_files:
  modified:
    - /tmp/llm-agent-providers/openai/openai.go
    - /tmp/llm-agent-providers/openai/map.go
    - /tmp/llm-agent-providers/openai/openai_test.go
decisions:
  - "OpenAI streaming emits `EventDone` from the final usage chunk, preserving the last seen finish reason."
  - "The adapter retries exactly once only when the stream fails before any event is delivered to the caller."
  - "Tool-call delta infrastructure is seeded now via per-index event emission, but full tool semantics remain a Phase 3 concern."
metrics:
  completed: 2026-05-10
  files_modified: 3
---

# Phase 2 Plan 01: OpenAI Streaming Summary

**One-liner:** Implemented OpenAI `Stream()` with real SSE handling, final usage capture, request-body `include_usage` enforcement, and the Phase 2 retry boundary of "retry once before first byte, never after".

## Tasks Completed

| # | Name | Key Files |
|---|------|-----------|
| 1 | Add OpenAI stream request shape with `stream_options.include_usage=true` | `openai/map.go` |
| 2 | Add iterator-style `StreamReader` over the SDK SSE stream | `openai/openai.go` |
| 3 | Add streaming tests for happy path and retry boundary | `openai/openai_test.go` |

## Verification Results

- `GOCACHE=/tmp/go-build go test ./openai/... -count=1` — PASS
- `GOCACHE=/tmp/go-build go build ./openai/...` — PASS
- `git diff --check` in `/tmp/llm-agent-providers` — PASS

Covered stream scenarios:

- happy-path text delta accumulation to `hello`
- final usage chunk mapped to `UsageReported`
- outbound request body contains `stream_options.include_usage=true`
- stream error before first byte retries once and succeeds
- stream error after first byte does not retry

## What Comes Next

- `02-02`: Anthropic streaming
- `02-03`: Ollama streaming
- `02-04`: shared streaming conformance after all three provider implementations land

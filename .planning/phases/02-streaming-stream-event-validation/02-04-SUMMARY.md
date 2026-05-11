---
phase: 02-streaming-stream-event-validation
plan: 04
subsystem: streaming-conformance
tags:
  - conformance
  - streaming
  - cancellation
  - goleak
dependency_graph:
  requires:
    - 02-01
    - 02-02
    - 02-03
  provides:
    - shared streaming conformance matrix
    - cross-provider cancel-mid-stream checks
    - cross-provider partial-error checks
  affects:
    - Phase 2 close
    - Phase 3 planning entry
tech_stack:
  added: []
  patterns:
    - fixture-driven stream happy-path replay
    - provider-specific live handlers under one shared contract
    - goleak-covered cancel and error checks
key_files:
  modified:
    - /tmp/llm-agent-providers/internal/contract/contract.go
    - /tmp/llm-agent-providers/internal/contract/generate_test.go
  created:
    - /tmp/llm-agent-providers/internal/contract/testdata/openai/stream_happy_gpt-4o-mini.json
    - /tmp/llm-agent-providers/internal/contract/testdata/anthropic/stream_happy_claude-3-5-haiku.json
    - /tmp/llm-agent-providers/internal/contract/testdata/ollama/stream_happy_llama3.1-8b.json
decisions:
  - "Shared stream conformance reuses the existing Fixture schema for happy-path replay and supplements it with provider-specific live handlers for cancel and partial-error checks."
  - "Stream accumulation in the harness restores provider/model identity from `model.Info()` because `llm.AccumulateStream` intentionally returns only the flattened response body and terminal metadata."
  - "Cross-provider partial-usage-on-error is currently represented as 'stream errors before EventDone' rather than a synthetic usage record."
metrics:
  completed: 2026-05-11
  stream_fixtures: 3
---

# Phase 2 Plan 04: Streaming Conformance Summary

**One-liner:** Extended the shared `internal/contract` harness from Generate-only to streaming, covering happy-path replay, cancel-mid-stream cleanup, and partial-error behavior across OpenAI, Anthropic, and Ollama.

## Tasks Completed

| # | Name | Key Files |
|---|------|-----------|
| 1 | Add shared `AssertStream` helper and response normalization | `internal/contract/contract.go` |
| 2 | Add shared stream conformance matrix and cross-provider live handler tests | `internal/contract/generate_test.go` |
| 3 | Add three happy-path streaming fixtures | `internal/contract/testdata/*/stream_happy_*.json` |

## Verification Results

- `GOCACHE=/tmp/go-build go test ./internal/contract/... -count=1` — PASS
- `GOCACHE=/tmp/go-build go vet ./internal/contract/...` — PASS
- `git diff --check` in `/tmp/llm-agent-providers` — PASS

Shared streaming coverage now includes:

- fixture-driven happy-path stream accumulation on all three providers
- OpenAI request-body assertion for `stream_options.include_usage=true`
- cancel-mid-stream returns `context.Canceled` within the contract window
- first-byte-delivered stream errors do not silently emit `EventDone`
- `goleak.VerifyTestMain` remains active across the suite

## What Comes Next

- Phase 2 is ready to close
- Next logical work is Phase 3 planning: native tool calling on all three providers plus agent refactor

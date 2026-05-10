---
phase: 01-walking-skeleton-generate
plan: 02
subsystem: openai
tags:
  - openai
  - provider-adapter
  - generate-only
  - typed-errors
dependency_graph:
  requires:
    - 01-01
  provides:
    - github.com/costa92/llm-agent-providers/openai.New
    - openai.OpenAI implements llm.ChatModel
  affects:
    - 01-05 conformance harness
    - 01-07 provider authoring guide examples
tech_stack:
  added:
    - github.com/openai/openai-go/v3 v3.35.0
  patterns:
    - functional options constructor
    - provider-local HTTP status to llm typed-error mapping
    - model-bound ProviderInfo
key_files:
  created:
    - /tmp/llm-agent-providers/openai/openai.go
    - /tmp/llm-agent-providers/openai/options.go
    - /tmp/llm-agent-providers/openai/map.go
    - /tmp/llm-agent-providers/openai/errors.go
    - /tmp/llm-agent-providers/openai/doc.go
    - /tmp/llm-agent-providers/openai/openai_test.go
    - /tmp/llm-agent-providers/openai/README.md
  modified:
    - /tmp/llm-agent-providers/go.mod
    - /tmp/llm-agent-providers/go.sum
decisions:
  - "Resolved SDK drift with current module resolution: openai-go/v3 pinned to v3.35.0 in implementation, despite earlier stale GitHub crawl showing v3.32.0."
  - "Disabled openai-go automatic retries with option.WithMaxRetries(0) because Phase 1 must not smuggle Phase 2 retry semantics into adapter behavior."
metrics:
  completed: 2026-05-10
  files_created: 7
  files_modified: 2
---

# Phase 1 Plan 02: OpenAI Generate Adapter Summary

**One-liner:** Implemented the `openai` sister-repo adapter for Phase 1 as a Generate-only `llm.ChatModel`, with model-bound `Info()`, a clear Stream stub, and deterministic typed-error mapping over `openai-go/v3`.

## Tasks Completed

| # | Name | Key Files |
|---|------|-----------|
| 1 | Add OpenAI adapter constructor, request/response mapping, and error mapping | `openai/openai.go`, `options.go`, `map.go`, `errors.go` |
| 2 | Add package docs and minimal usage note | `openai/doc.go`, `openai/README.md` |
| 3 | Add Generate happy-path and HTTP-status taxonomy tests | `openai/openai_test.go` |
| 4 | Pin sister-repo dependencies to core pre-release tag and OpenAI SDK | `/tmp/llm-agent-providers/go.mod`, `go.sum` |

## Verification Results

- `GOCACHE=/tmp/go-build go test ./openai/...` — PASS
- `GOCACHE=/tmp/go-build go build ./openai/...` — PASS

Covered test scenarios:

- `New` rejects missing `WithModel`
- `Info()` returns bound model and all-false optional capabilities
- `Stream()` returns the Phase 1 stub error
- Generate happy path normalizes `stop` to `llm.FinishReasonStop`
- `401`, `403` -> `*llm.AuthError`
- `429 insufficient_quota` -> `*llm.RateLimitError{Reason:"quota_exhausted", RetryAfter:...}`
- `429 generic` -> `*llm.RateLimitError`
- `500` -> `*llm.TransientError`
- `404` -> `*llm.InvalidRequestError`

## Notable Deviations

- The plan text pinned `github.com/openai/openai-go/v3@v3.35.0`, while an earlier GitHub releases crawl only showed `v3.32.0`. Actual module resolution on 2026-05-10 succeeded at `v3.35.0`, and implementation follows that resolved version.
- `option.WithMaxRetries(0)` was added in the constructor so the SDK does not silently retry `429/5xx` in Phase 1. This keeps retry behavior aligned with the roadmap, where the explicit K4 retry state machine lands in Phase 2.

## What Comes Next

- `01-03`: Anthropic Generate-only adapter
- `01-04`: Ollama Generate-only adapter
- `01-05`: shared `internal/contract` conformance harness over all three adapters

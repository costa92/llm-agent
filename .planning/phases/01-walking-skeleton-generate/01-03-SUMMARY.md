---
phase: 01-walking-skeleton-generate
plan: 03
subsystem: anthropic
tags:
  - anthropic
  - provider-adapter
  - generate-only
  - system-prompt-lift
dependency_graph:
  requires:
    - 01-01
  provides:
    - github.com/costa92/llm-agent-providers/anthropic.New
    - anthropic.Anthropic implements llm.ChatModel
  affects:
    - 01-05 conformance harness
    - 01-07 provider authoring guide examples
tech_stack:
  added:
    - github.com/anthropics/anthropic-sdk-go v1.41.0
  patterns:
    - functional options constructor
    - provider-local HTTP status to llm typed-error mapping
    - top-level system prompt lift
key_files:
  created:
    - /tmp/llm-agent-providers/anthropic/anthropic.go
    - /tmp/llm-agent-providers/anthropic/options.go
    - /tmp/llm-agent-providers/anthropic/map.go
    - /tmp/llm-agent-providers/anthropic/errors.go
    - /tmp/llm-agent-providers/anthropic/doc.go
    - /tmp/llm-agent-providers/anthropic/anthropic_test.go
    - /tmp/llm-agent-providers/anthropic/README.md
  modified:
    - /tmp/llm-agent-providers/go.mod
    - /tmp/llm-agent-providers/go.sum
decisions:
  - "Q2 resolved with Path A: anthropic-sdk-go publicly exports `type Error = apierror.Error`, so errors.As(err, *anthropic.Error) is used directly."
  - "Disabled SDK automatic retries with option.WithMaxRetries(0) to keep Phase 1 behavior aligned with the roadmap's Phase 2 retry state machine."
  - "System prompts are always lifted to top-level `system`; `role=system` messages are never emitted in request `messages`."
metrics:
  completed: 2026-05-10
  files_created: 7
  files_modified: 2
---

# Phase 1 Plan 03: Anthropic Generate Adapter Summary

**One-liner:** Implemented the `anthropic` sister-repo adapter for Phase 1 as a Generate-only `llm.ChatModel`, including Anthropic-specific top-level `system` lifting and `529 overloaded_error -> RateLimitError` mapping.

## Tasks Completed

| # | Name | Key Files |
|---|------|-----------|
| 1 | Add Anthropic adapter constructor, request/response mapping, and error mapping | `anthropic/anthropic.go`, `options.go`, `map.go`, `errors.go` |
| 2 | Verify Q2 error path and use public `anthropic.Error` alias | `/tmp/anthropic-pubapi.txt`, `anthropic/errors.go` |
| 3 | Add package docs and minimal usage note | `anthropic/doc.go`, `anthropic/README.md` |
| 4 | Add Generate happy-path, system-lift, and HTTP-status taxonomy tests | `anthropic/anthropic_test.go` |

## Verification Results

- `GOCACHE=/tmp/go-build go test ./anthropic/...` — PASS
- `GOCACHE=/tmp/go-build go build ./anthropic/...` — PASS

Covered test scenarios:

- `New` rejects missing `WithModel`
- `Info()` returns bound model and all-false optional capabilities
- `Stream()` returns the Phase 1 stub error
- Generate happy path normalizes `end_turn` to `llm.FinishReasonStop`
- `Request.SystemPrompt` goes to top-level `system`
- `role=system` messages are lifted into `system` and omitted from `messages`
- `401` -> `*llm.AuthError`
- `400 invalid_request_error` -> `*llm.InvalidRequestError`
- `429 rate_limit_error` -> `*llm.RateLimitError`
- `529 overloaded_error` -> `*llm.RateLimitError`
- `500 api_error` -> `*llm.TransientError`

## What Comes Next

- `01-04`: Ollama Generate-only adapter
- `01-05`: shared `internal/contract` conformance harness over OpenAI, Anthropic, and Ollama

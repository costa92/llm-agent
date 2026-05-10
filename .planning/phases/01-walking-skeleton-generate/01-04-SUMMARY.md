---
phase: 01-walking-skeleton-generate
plan: 04
subsystem: ollama
tags:
  - ollama
  - provider-adapter
  - generate-only
  - status-capture
dependency_graph:
  requires:
    - 01-01
  provides:
    - github.com/costa92/llm-agent-providers/ollama.New
    - ollama.Ollama implements llm.ChatModel
  affects:
    - 01-05 conformance harness
tech_stack:
  added:
    - github.com/ollama/ollama v0.23.2
  patterns:
    - functional options constructor
    - custom RoundTripper status capture
    - sync chat via stream=false
key_files:
  created:
    - /tmp/llm-agent-providers/ollama/ollama.go
    - /tmp/llm-agent-providers/ollama/options.go
    - /tmp/llm-agent-providers/ollama/map.go
    - /tmp/llm-agent-providers/ollama/errors.go
    - /tmp/llm-agent-providers/ollama/doc.go
    - /tmp/llm-agent-providers/ollama/ollama_test.go
    - /tmp/llm-agent-providers/ollama/README.md
  modified:
    - /tmp/llm-agent-providers/go.mod
    - /tmp/llm-agent-providers/go.sum
decisions:
  - "Phase 1 keeps the transport-based status capture pattern even though `ollama/api` also exposes public status error structs; this preserves the plan's cross-SDK fallback pattern and keeps classification stable if plain errors surface."
  - "All requests force `stream=false`, because Ollama's Chat API always uses the streaming code path internally even for a single final response."
metrics:
  completed: 2026-05-10
  files_created: 7
  files_modified: 2
---

# Phase 1 Plan 04: Ollama Generate Adapter Summary

**One-liner:** Implemented the `ollama` sister-repo adapter for Phase 1 as a Generate-only `llm.ChatModel`, with explicit `stream=false`, model-bound `Info()`, and transport-backed typed-error mapping for local `/api/chat`.

## Tasks Completed

| # | Name | Key Files |
|---|------|-----------|
| 1 | Add Ollama adapter constructor, request/response mapping, and error mapping | `ollama/ollama.go`, `options.go`, `map.go`, `errors.go` |
| 2 | Add package docs and minimal usage note | `ollama/doc.go`, `ollama/README.md` |
| 3 | Add Generate happy-path and error-taxonomy tests | `ollama/ollama_test.go` |

## Verification Results

- `GOCACHE=/tmp/go-build go test ./ollama/...` — PASS
- `GOCACHE=/tmp/go-build go build ./ollama/...` — PASS

Covered test scenarios:

- `New` rejects missing `WithModel`
- `Info()` returns bound model and all-false optional capabilities
- `Stream()` returns the Phase 1 stub error
- Generate happy path returns one final response from NDJSON scanner path
- request body enforces `"stream": false`
- `404 model not found / not pulled` -> `*llm.InvalidRequestError`
- `401` -> `*llm.AuthError`
- `500` -> `*llm.TransientError`
- no daemon / connection refused -> `*llm.TransientError`
- `400` -> `*llm.InvalidRequestError`

## What Comes Next

- `01-05`: shared `internal/contract` conformance harness across OpenAI, Anthropic, and Ollama

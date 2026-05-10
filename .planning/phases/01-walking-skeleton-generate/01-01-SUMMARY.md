---
phase: 01-walking-skeleton-generate
plan: 01
subsystem: llm
tags:
  - llm
  - typed-errors
  - provider-contract
  - go-stdlib-only
dependency_graph:
  requires:
    - 00-01a
    - 00-01b
  provides:
    - llm.AuthError
    - llm.RateLimitError
    - llm.InvalidRequestError
    - llm.TransientError
  affects:
    - plan 01-02 openai adapter error mapping
    - plan 01-03 anthropic adapter error mapping
    - plan 01-04 ollama adapter error mapping
tech_stack:
  added: []
  patterns:
    - stdlib-only imports
    - typed errors with errors.As
    - Unwrap chain preservation
key_files:
  created:
    - llm/errors_test.go
  modified:
    - llm/errors.go
decisions:
  - "Provider adapter error normalization lives in core llm package, not per provider package."
  - "Typed errors expose Wrapped error and use pointer receivers so downstream errors.As checks are stable."
  - "RateLimitError carries RetryAfter and Reason as raw provider-facing fields; parsing stays with callers."
metrics:
  completed: 2026-05-10
  files_created: 1
  files_modified: 1
---

# Phase 1 Plan 01: Core Typed Errors Summary

**One-liner:** Added four core typed provider errors in `llm/errors.go` and regression tests in `llm/errors_test.go`, giving Phase 1 adapters a shared `errors.As` contract without breaking the stdlib-only invariant.

## Tasks Completed

| # | Name | Key Files |
|---|------|-----------|
| 1 | Add four typed provider errors with `Error()` + `Unwrap()` | `llm/errors.go` |
| 2 | Add regression coverage for `errors.As`, unwrap chain, nil wrapped, and rate-limit formatting | `llm/errors_test.go` |

## Verification Results

- `go test ./llm/...` — PASS
- `GOCACHE=/tmp/go-build go vet ./llm/...` — PASS
- `GOCACHE=/tmp/go-build go build ./llm/...` — PASS
- `grep -c 'type \(AuthError\|RateLimitError\|InvalidRequestError\|TransientError\) struct' llm/errors.go` — `4`
- `grep -c 'func (e \*\(AuthError\|RateLimitError\|InvalidRequestError\|TransientError\)) Unwrap() error' llm/errors.go` — `4`

## File Inventory

| File | Change | Provides |
|------|--------|----------|
| `llm/errors.go` | modified | `AuthError`, `RateLimitError`, `InvalidRequestError`, `TransientError` with preserved unwrap chains |
| `llm/errors_test.go` | new | table-style regression coverage for the new typed errors |

## Constraints Check

- `llm` remains stdlib-only.
- New typed errors use `Wrapped error`, not `any`.
- All new methods are pointer receivers, matching the intended `errors.As` usage.

## Deviation From Plan

The code and tests for Plan 01 are complete, but the release artifact is not:

- `v0.3.0-pre.2` has **not** been created/pushed yet in this session.

This means downstream sister-repo plans can use the local source tree contract immediately, but `llm-agent-providers` cannot consume it via a tagged version until the commit/tag step is done.

## What Comes Next

- Create a commit for this plan.
- Create and push tag `v0.3.0-pre.2`.
- Then continue with Phase 1 Wave 1 adapter plans (`01-02`, `01-03`, `01-04`).

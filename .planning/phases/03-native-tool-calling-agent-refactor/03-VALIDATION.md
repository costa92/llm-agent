---
phase: 3
phase_slug: native-tool-calling-agent-refactor
date: 2026-05-12
---

# Phase 3 Validation Strategy

> Reconstructed after milestone close from Phase 3 PLAN/SUMMARY artifacts to
> backfill the Nyquist validation record. This document points at already
> executed tests and recorded behaviors.

## Test Framework

| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` across `llm-agent-providers` and core `llm-agent` |
| Config file | None |
| Quick run | `GOCACHE=/tmp/go-build go test ./internal/contract/... -count=1` or `GOCACHE=/tmp/go-build go test ./... -count=1` in the touched repo |
| Full suite | `GOCACHE=/tmp/go-build go test ./... -count=1` |
| Supporting checks | `GOCACHE=/tmp/go-build go vet ./internal/contract/...`, `GOCACHE=/tmp/go-build go build ./...` |

## Phase Requirements -> Test Map

| Req ID | Behavior | Test type | Command |
|--------|----------|-----------|---------|
| OAI-03 | OpenAI native tool-calling surface works and preserves immutable `WithTools(...)` semantics | unit | `go test ./openai/... -count=1` |
| ANT-03 | Anthropic native tool use works across multi-block tool payloads | unit | `go test ./anthropic/... -count=1` |
| OLL-03 | Ollama per-model tool strategies distinguish supported and unsupported models | unit | `go test ./ollama/... -count=1` |
| CONF-04 | Shared conformance covers calculator scenario, parallel calls, multi-block calls, and capability-degrade | conformance | `go test ./internal/contract/... -count=1` |
| CONF-05 | Dedupe behavior keyed on `(message_id, tool_use_id)` is enforced | conformance | `go test ./internal/contract/... -count=1` |
| CORE-10 | Core agents consume `llm.ChatModel`, negotiate tools, and fail/fallback correctly | unit/integration | `go test ./... -count=1` in core repo |

## Sampling Rate

- After each provider adapter task: run the touched adapter package tests.
- After shared conformance changes: run `go test ./internal/contract/...`.
- After core agent refactor: run `go test ./...` and `go build ./...` in the
  core repo.
- Before phase close: both provider-repo and core-repo full suites must be
  green.

## Plan -> Validation Map

| Plan | Validation deliverable |
|------|------------------------|
| 03-01 | OpenAI adapter tests cover tool invocation semantics |
| 03-02 | Anthropic adapter tests cover native tool use |
| 03-03 | Ollama tests cover strategy-table behavior and unsupported-model failure |
| 03-04 | Shared `internal/contract` tool-calling matrix and dedupe checks |
| 03-05 | Core `react_test.go`, `function_call_test.go`, example tests, and full repo build/test |

## Manual-Only Verifications

All Phase 3 requirements have automated coverage recorded in shipped tests.
No remaining manual-only Nyquist gap is required for this phase.

## Evidence Carried Forward

- `03-04-SUMMARY.md`: `go test ./internal/contract/...` and `go vet
  ./internal/contract/...` pass
- `03-05-SUMMARY.md`: `go test ./...` and `go build ./...` pass in the core
  repo

## Phase-Level Sign-Off

- Shared conformance and core regression tests cover the phase boundary.
- The core agent refactor has explicit regression evidence for native-tool
  fast-path and fallback behavior.
- This phase now has a validation artifact on disk for future audits.

---

*Validation strategy backfilled on 2026-05-12 from existing Phase 3 artifacts.*

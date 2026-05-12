---
phase: 2
phase_slug: streaming-stream-event-validation
date: 2026-05-12
---

# Phase 2 Validation Strategy

> Reconstructed after milestone close from Phase 2 PLAN/SUMMARY artifacts to
> backfill the Nyquist validation record. Existing tests and commands are the
> source of truth; this document does not claim new execution beyond what the
> summaries already recorded.

## Test Framework

| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (Go 1.26) + `go.uber.org/goleak` in `llm-agent-providers/internal/contract` |
| Config file | None |
| Quick run | `cd /tmp/llm-agent-providers && GOCACHE=/tmp/go-build go test ./openai/... ./anthropic/... ./ollama/... ./internal/contract/... -count=1` |
| Full suite | `cd /tmp/llm-agent-providers && GOCACHE=/tmp/go-build go test ./... -count=1` |
| Integration/liveness | `cd /tmp/llm-agent-providers && GOCACHE=/tmp/go-build go vet ./internal/contract/...` |

## Phase Requirements -> Test Map

| Req ID | Behavior | Test type | Command |
|--------|----------|-----------|---------|
| OAI-02 | OpenAI stream happy path accumulates text deltas | unit | `go test ./openai/... -count=1` |
| OAI-06 | OpenAI stream requests include `stream_options.include_usage=true` and preserve usage source | unit | `go test ./openai/... -count=1` |
| OAI-07 | OpenAI retries once before first byte and never after first byte | unit | `go test ./openai/... -count=1` |
| ANT-02 | Anthropic stream accumulation preserves ordered event semantics | unit | `go test ./anthropic/... -count=1` |
| ANT-06 | Anthropic stream error paths preserve the three-state usage contract | unit | `go test ./anthropic/... -count=1` |
| OLL-02 | Ollama streaming honors cancel and avoids goroutine leaks | unit | `go test ./ollama/... -count=1` |
| OLL-06 | Ollama stream usage/retry semantics match the shared contract | unit | `go test ./ollama/... -count=1` |
| CONF-03 | Shared conformance covers happy-path replay, cancel-mid-stream, and partial-error behavior | conformance | `go test ./internal/contract/... -count=1` |

## Sampling Rate

- After every provider task commit: run the touched provider package tests.
- After the conformance task: run `go test ./internal/contract/... -count=1`.
- Before phase close: run `go test ./... -count=1` in `llm-agent-providers`.
- Goleak continuity is enforced through the shared conformance package rather
  than per-provider bespoke harnesses.

## Plan -> Validation Map

| Plan | Validation deliverable |
|------|------------------------|
| 02-01 | `openai/openai_test.go` verifies SSE handling, include-usage enforcement, and retry boundary |
| 02-02 | `anthropic/anthropic_test.go` verifies stream accumulation and error-path behavior |
| 02-03 | `ollama/ollama_test.go` verifies callback-to-stream bridging and cancel behavior |
| 02-04 | `internal/contract/generate_test.go` + stream fixtures verify cross-provider conformance; `goleak.VerifyTestMain` remains active |

## Manual-Only Verifications

All Phase 2 requirements have automated coverage recorded in the shipped
provider test suites and shared conformance harness. No additional manual-only
Nyquist gap remains after this backfill.

## Evidence Carried Forward

- `02-01-SUMMARY.md`: `go test ./openai/...` and `go build ./openai/...` pass
- `02-02-SUMMARY.md`: `go test ./anthropic/...` and `go build ./anthropic/...`
  pass
- `02-03-SUMMARY.md`: `go test ./ollama/...` and `go build ./ollama/...` pass
- `02-04-SUMMARY.md`: `go test ./internal/contract/...`, `go vet
  ./internal/contract/...`, and `git diff --check` pass

## Phase-Level Sign-Off

- Shared streaming coverage exists for all Phase 2 requirements.
- The cross-provider conformance harness is the Nyquist sampling backbone for
  this phase.
- `goleak` is part of the recorded verification surface.
- This phase should now be treated as having a validation artifact on disk.

---

*Validation strategy backfilled on 2026-05-12 from existing Phase 2 artifacts.*

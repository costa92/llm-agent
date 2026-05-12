---
phase: 6
phase_slug: reference-customer-support
date: 2026-05-12
---

# Phase 6 Validation Strategy

> Reconstructed after milestone close from Phase 6 PLAN/SUMMARY artifacts plus
> the shipped `06-VERIFICATION.md`. This backfill records the Nyquist sampling
> surface using already executed tests and runtime checks.

## Test Framework

| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` in `llm-agent-customer-support` with sibling-aware `go.work`; selected Docker runtime checks |
| Config file | None |
| Quick run | `GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go test ./internal/... -count=1` or focused package commands |
| Full suite | `GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go test ./... -count=1` |
| Supporting checks | `GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go build ./...`, `docker compose -f compose/compose.yaml config`, targeted live probes |

## Phase Requirements -> Test Map

| Req ID | Behavior | Test type | Command |
|--------|----------|-----------|---------|
| REFSVC-01 | server bootstrap/config/tracer wiring works | unit | `go test ./internal/config ./internal/app -count=1` |
| REFSVC-02 | `/chat`, `/chat/stream`, `/healthz`, `/readyz` behave correctly | unit | `go test ./internal/httpapi ./internal/app -count=1` |
| REFSVC-03 | `X-Trace-Id` is surfaced on transport responses | unit | `go test ./internal/httpapi ./internal/app -count=1` |
| REFSVC-04 | chat and embedding providers are independently configurable | unit | `go test ./... -count=1` |
| REFSVC-05 | support flow with RAG + `StateGraph` + tools works | unit | `go test ./internal/supportflow -count=1` |
| REFSVC-06 | SQLite/Postgres session storage works | unit | `go test ./... -count=1` |
| REFSVC-07 | hard caps fail closed | unit | `go test ./... -count=1` |
| REFSVC-08 | `DISABLE_LLM=1` panic switch works | unit | `go test ./... -count=1` |
| REFSVC-09 | prompt-injection guardrails and trace marking work | unit/live | `go test ./internal/guardrails ./internal/supportflow -count=1`; `go test ./internal/supportflow -run TestFlow_FlaggedInputMarksTraceAttribute -count=1` |
| REFSVC-10 | compose stack/runtime path reaches a working service surface | runtime | `docker compose -f compose/compose.yaml config`; live `/readyz` and `/chat` probe evidence recorded in `06-VERIFICATION.md` |
| REFSVC-11 | Grafana dashboard asset is provisioned | unit/runtime | `go test ./compose -run TestDemoAssetsExistAndDocumentObservability -count=1`; Grafana API search evidence in `06-VERIFICATION.md` |
| REFSVC-12 | tail-sampling collector policy is implemented and observable | unit/runtime | `go test ./compose -count=1`; live OTLP probe evidence in `06-VERIFICATION.md` |
| REFSVC-13 | README demo-only boundary is explicit | unit/doc | `go test ./compose -run TestDemoAssetsExistAndDocumentObservability -count=1` |

## Sampling Rate

- After each plan: run focused package tests for the touched transport/flow
  slice.
- After major integration points: run `go test ./... -count=1`.
- Before phase close: require full suite green, compose config validation, and
  recorded runtime proof for service and observability claims.

## Plan -> Validation Map

| Plan | Validation deliverable |
|------|------------------------|
| 06-01 | bootstrap/config/app tests and full build |
| 06-02 | HTTP transport tests and full suite |
| 06-03 | provider-selection regressions in service test surface |
| 06-04 | support-flow tests |
| 06-05 | session-store regressions in full suite |
| 06-06 | cap/panic-switch regressions in full suite |
| 06-07 | guardrails tests plus trace-attribute follow-up |
| 06-08 | compose asset tests, compose config, and demo README checks |

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Full cold-machine compose-native app-container proof | REFSVC-10 / REFSVC-12 | The archived milestone proof used the compose dependency stack plus a locally built app binary after cold image/model pulls exceeded the closeout window | Re-run `docker compose -f compose/compose.yaml up --build` on a favorable environment and capture `readyz`, `/chat`, trace correlation, and sampling proof |

## Evidence Carried Forward

- `06-01-SUMMARY.md`: bootstrap tests, full suite, and build pass
- `06-02-SUMMARY.md`: transport tests, full suite, and build pass
- `06-07-SUMMARY.md`: guardrails/supportflow tests, full suite, and build pass
- `06-08-SUMMARY.md`: compose asset tests, full suite, and compose config pass
- `06-VERIFICATION.md`: runtime `/readyz`, `/chat`, Grafana dashboard, and
  tail-sampling retention proof

## Phase-Level Sign-Off

- Phase 6 now has both a verification report and a validation strategy on disk.
- The strongest remaining manual follow-up is explicitly scoped to a
  compose-native cold-machine rerun, not a missing phase artifact.
- This phase should no longer count as missing Nyquist validation artifacts in
  future milestone audits.

---

*Validation strategy backfilled on 2026-05-12 from existing Phase 6 artifacts.*

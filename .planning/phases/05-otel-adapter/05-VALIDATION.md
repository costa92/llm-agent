---
phase: 5
phase_slug: otel-adapter
date: 2026-05-12
---

# Phase 5 Validation Strategy

> Reconstructed after milestone close from Phase 5 PLAN/SUMMARY artifacts to
> backfill the Nyquist validation record.

## Test Framework

| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` in `llm-agent-otel` with a temporary sibling-aware `go.work` |
| Config file | None |
| Quick run | `GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go test ./otelmodel/... ./otelagent/... ./otelmetrics/... ./otelslog/... -count=1` |
| Full suite | `GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go test ./... -count=1` |
| Supporting checks | `GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go build ./...` |

## Phase Requirements -> Test Map

| Req ID | Behavior | Test type | Command |
|--------|----------|-----------|---------|
| OTEL-01 | `otelmodel.Wrap(...)` preserves optional capability interfaces | unit | `go test ./otelmodel/... -count=1` |
| OTEL-02 | `otelagent.Wrap(...)` emits the agent span tree without changing contract | unit | `go test ./otelagent/... -count=1` |
| OTEL-03 | `gen_ai.*` semconv constants and opt-in gates are centralized | unit | `go test ./... -run 'Test(Metrics|ContentCapture|Cardinality)' -count=1` |
| OTEL-04 | Metrics emit token/latency/TTFT series | unit | `go test ./... -run 'Test(Metrics|ContentCapture|Cardinality)' -count=1` |
| OTEL-05 | Metric attribute allowlist excludes high-cardinality fields | unit | `go test ./... -run 'Test(Metrics|ContentCapture|Cardinality)' -count=1` |
| OTEL-06 | Content capture defaults off and redacts when enabled | unit | `go test ./... -run 'Test(Metrics|ContentCapture|Cardinality)' -count=1` |
| OTEL-07 | Streaming instrumentation avoids span explosion | unit | `go test ./otelmodel/... -count=1` |
| OTEL-08 | `slog` bridge preserves context correlation | unit | `go test ./otelslog/... -count=1` |
| OTEL-09 | OTLP HTTP default and gRPC opt-in exporter wiring build and test | unit | `go test ./... -run 'Test(ExporterConfig|ComposeAssets|README_Documents|NewTracerProvider)' -count=1` |
| OTEL-10 | Documentation and compose demo assets exist and match the shipped surface | unit/doc | `go test ./... -run 'Test(ExporterConfig|ComposeAssets|README_Documents|NewTracerProvider)' -count=1` |

## Sampling Rate

- After each wrapper package task: run the touched package tests.
- After metrics/content-capture work: run the targeted guardrail test subset.
- After exporter/demo task: run targeted exporter/doc tests, then full suite.
- Before phase close: full `go test ./...` and `go build ./...` must be green.

## Plan -> Validation Map

| Plan | Validation deliverable |
|------|------------------------|
| 05-01 | `otelmodel` capability-preservation and span-shape tests |
| 05-02 | `otelagent` trace-tree tests |
| 05-03 | metrics/content-capture/cardinality tests |
| 05-04 | `otelslog` correlation tests |
| 05-05 | exporter/demo/doc regression tests plus full build/test |

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Visual confirmation that the compose demo trace is easy to inspect in Grafana UI | OTEL-10 | The shipped evidence is asset/test-level plus docs; the milestone did not archive a UI walkthrough artifact | Run the compose demo in `llm-agent-otel`, issue a sample wrapped call, and inspect Tempo/Grafana manually if stronger operator proof is needed |

## Evidence Carried Forward

- `05-01-SUMMARY.md`: `go test ./otelmodel/...` and `go build ./otelmodel/...`
  pass
- `05-02-SUMMARY.md`: `go test ./otelagent/...` and `go build ./otelagent/...`
  pass
- `05-03-SUMMARY.md`: targeted guardrail tests, full `go build`, and full
  `go test ./...` pass
- `05-04-SUMMARY.md`: `go test ./otelslog/...` and `go build ./otelslog/...`
  pass
- `05-05-SUMMARY.md`: exporter/demo/doc tests plus full suite pass

## Phase-Level Sign-Off

- All Phase 5 requirements have automated verification coverage on disk.
- One optional manual UI-proof path remains, but it is no longer a missing
  Nyquist artifact.
- This phase now has a validation artifact on disk for future audits.

---

*Validation strategy backfilled on 2026-05-12 from existing Phase 5 artifacts.*

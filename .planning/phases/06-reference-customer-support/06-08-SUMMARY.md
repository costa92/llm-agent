# Phase 06-08 Summary

Date: 2026-05-11
Repo: `llm-agent-customer-support`
Plan: [06-08-PLAN.md](/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/phases/06-reference-customer-support/06-08-PLAN.md)

## Objective

Package the reference service into a documented local demo stack with
observability, tail-sampling, and an explicit demo-only boundary.

## Delivered

- Added a local demo stack under `compose/`:
  - `compose/compose.yaml` wiring app + Ollama + OTel Collector + Grafana
  - `compose/otel-collector.yaml` with tail-sampling and spanmetrics export
  - `compose/Dockerfile` for containerized app builds
  - Grafana provisioning for datasources + dashboard auto-load
- Added a pre-provisioned Grafana dashboard at
  `dashboards/customer-support-observability.json` with panels named:
  - `Request Latency`
  - `Token Usage`
  - `Estimated Cost`
  - `Error Rate`
  - `Tool Success Ratio`
- Added an asset-level regression test in `compose/assets_test.go` so the repo
  now verifies the demo stack files, collector policies, dashboard panel names,
  and README operator instructions.
- Followed up after the initial package commit to align two roadmap promises
  exactly:
  - `compose/otel-collector.yaml` now uses `decision_wait: 30s`, matching
    `REFSVC-12` and the Phase 6 tail-sampling contract.
  - blocked prompt-injection requests now mark the active trace with
    `prompt_injection_attempt=true`, making the `REFSVC-09` observability claim
    true in code rather than only in prose.
- Updated `README.md` with:
  - the exact `docker compose -f compose/compose.yaml up --build` startup path
  - `readyz` and `/chat` verification commands
  - Grafana dashboard expectations
  - tail-sampling policy notes
  - explicit demo-only caveats around observability and production hardening
- Corrected the `github.com/lib/pq v1.12.3/go.mod` checksum entry in `go.sum`,
  which would otherwise block fresh module verification during isolated builds.
- Tightened the Ollama bootstrap path so `ollama-init` pre-pulls both
  `llama3.1:8b` and `nomic-embed-text`; this avoids a false-ready demo stack
  that only becomes usable after the first chat request triggers a lazy pull.

## Files

- `/tmp/llm-agent-customer-support/compose/assets_test.go`
- `/tmp/llm-agent-customer-support/compose/compose.yaml`
- `/tmp/llm-agent-customer-support/compose/Dockerfile`
- `/tmp/llm-agent-customer-support/compose/otel-collector.yaml`
- `/tmp/llm-agent-customer-support/compose/grafana/provisioning/datasources/datasources.yaml`
- `/tmp/llm-agent-customer-support/compose/grafana/provisioning/dashboards/dashboards.yaml`
- `/tmp/llm-agent-customer-support/dashboards/customer-support-observability.json`
- `/tmp/llm-agent-customer-support/README.md`
- `/tmp/llm-agent-customer-support/go.sum`

## Verification

Executed against the local 4-repo workspace:

```bash
GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go test ./compose -count=1
GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go test ./... -count=1
docker compose -f compose/compose.yaml config
docker compose -f compose/compose.yaml up --build
```

Result:

- `go test ./compose -count=1`: pass
- `go test ./... -count=1`: pass
- `docker compose -f compose/compose.yaml config`: pass
- targeted follow-up tests for the `decision_wait` collector asset and the
  prompt-injection trace attribute: pass
- `docker compose -f compose/compose.yaml up --build`: started successfully but
  did not reach final `readyz` / `/chat` assertions within this session because
  the first cold boot spent the full verification window pulling large Docker
  images and Ollama model layers

## Notes

- The shipped dashboard is intentionally demo-oriented. `Token Usage`,
  `Estimated Cost`, and `Tool Success Ratio` are span-derived observability
  views, not production billing truth.
- This closes the planned Phase 6 implementation surface in code and docs. The
  remaining manual follow-up is a full cold-machine compose smoke test after the
  heavy Docker/Ollama layers are cached locally.

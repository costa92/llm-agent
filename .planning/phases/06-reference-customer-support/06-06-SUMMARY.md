# Phase 06-06 Summary

Date: 2026-05-11
Repo: `llm-agent-customer-support`
Plan: [06-06-PLAN.md](/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/phases/06-reference-customer-support/06-06-PLAN.md)

## Objective

Wire K7 hard caps and the panic switch into the running service before the
security-polish and demo-packaging plans.

## Delivered

- Added `internal/limits` with config-driven runtime guardrails for:
  - per-request token caps
  - per-IP per-minute request caps
  - per-run tool-loop caps
  - retry-attempt caps
  - cross-request daily token budget
  - live `DISABLE_LLM` panic switch
- Added consistent HTTP error mapping so guard failures surface as:
  - `429 Too Many Requests` for cap violations
  - `503 Service Unavailable` for the panic switch
- Extended `internal/httpapi` to run guard preflight checks before agent
  execution on both JSON chat and SSE chat routes.
- Extended `internal/app` to build one shared guard instance and wrap the
  support-flow agent with postflight enforcement.
- Extended `internal/config` with defaults for all K7 cap knobs.
- Added tests covering:
  - preflight token and rate-limit failures
  - tool-loop and retry-overflow postflight failures
  - cross-request daily budget exhaustion
  - `DISABLE_LLM` flipping live without service restart
  - HTTP `429` and `503` behavior through the transport layer

## Files

- `/tmp/llm-agent-customer-support/internal/limits/limits.go`
- `/tmp/llm-agent-customer-support/internal/limits/limits_test.go`
- `/tmp/llm-agent-customer-support/internal/httpapi/httpapi.go`
- `/tmp/llm-agent-customer-support/internal/httpapi/httpapi_test.go`
- `/tmp/llm-agent-customer-support/internal/config/config.go`
- `/tmp/llm-agent-customer-support/internal/config/config_test.go`
- `/tmp/llm-agent-customer-support/internal/app/app.go`
- `/tmp/llm-agent-customer-support/internal/app/app_test.go`
- `/tmp/llm-agent-customer-support/README.md`

## Verification

Executed against the local 4-repo workspace:

```bash
GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go test ./internal/limits ./internal/httpapi -count=1
GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go test ./... -count=1
GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go build ./...
```

Result:

- `go test ./internal/limits ./internal/httpapi -count=1`: pass
- `go test ./... -count=1`: pass
- `go build ./...`: pass

## Notes

- The panic switch is deliberately polled on each request through env lookup so
  `DISABLE_LLM=1` takes effect immediately.
- Token accounting remains intentionally coarse for this reference service:
  request-side estimation uses word count, while cross-request budget
  accumulation consumes the agent-reported token count.

# Phase 06-01 Summary

Date: 2026-05-11
Repo: `llm-agent-customer-support`
Plan: [06-01-PLAN.md](/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/phases/06-reference-customer-support/06-01-PLAN.md)

## Objective

Land the first runnable bootstrap for the reference customer-support service:
env-var config, provider-aware model selection, OTel tracer-provider setup,
wrapped agent construction, and graceful shutdown.

## Delivered

- Added `internal/config` with env parsing, provider constants, OTLP defaults,
  provider-specific default models, and shutdown timeout handling.
- Added `internal/app` with:
  - provider-aware `DefaultModelFactory(...)`
  - tracer-provider factory using `llm-agent-otel`
  - wrapped `otelmodel.Wrap(...)` + `otelagent.Wrap(...)`
  - `http.Server` bootstrap and context-driven graceful shutdown
- Added `cmd/server/main.go` with SIGINT/SIGTERM handling and app startup.
- Added bootstrap tests covering config defaults, provider selection, runnable
  wrapped agent construction, and tracer-provider shutdown on cancel.
- Updated the service README to reflect the actual Phase 6 bootstrap state.
- Added temporary local `replace` directives in `go.mod` so the service can
  build against sibling checkouts before coordinated sister-repo tags exist.

## Files

- `/tmp/llm-agent-customer-support/cmd/server/main.go`
- `/tmp/llm-agent-customer-support/cmd/server/main_test.go`
- `/tmp/llm-agent-customer-support/internal/app/app.go`
- `/tmp/llm-agent-customer-support/internal/app/app_test.go`
- `/tmp/llm-agent-customer-support/internal/config/config.go`
- `/tmp/llm-agent-customer-support/internal/config/config_test.go`
- `/tmp/llm-agent-customer-support/README.md`
- `/tmp/llm-agent-customer-support/go.mod`

## Verification

Executed against a local `go.work` binding the 4 sibling repos together:

```bash
GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go test ./internal/config ./internal/app -count=1
GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go test ./... -count=1
GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go build ./...
```

Result:

- `go test ./internal/config ./internal/app`: pass
- `go test ./...`: pass
- `go build ./...`: pass

## Notes

- The temporary `replace` directives are intentional for local Phase 6
  execution and align with the documented INFRA-06 escape hatch. They should be
  removed or replaced with coordinated tagged versions before release-branch
  work.

# Phase 05-03 Summary

Date: 2026-05-11
Repo: `llm-agent-otel`
Plan: [05-03-PLAN.md](/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/phases/05-otel-adapter/05-03-PLAN.md)

## Objective

Centralize `gen_ai.*` semantic-convention definitions and land the metrics,
cardinality, and content-capture guardrails required before exporter wiring.

## Delivered

- Added root-level `semconv_gen_ai.go` with:
  - centralized `gen_ai.*` attribute and metric-name constants
  - `OTEL_SEMCONV_STABILITY_OPT_IN=gen_ai_latest_experimental` gate
  - `OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT` gate
  - simple redaction helper for captured content
- Added `otelmetrics/` package with recorder support for:
  - `gen_ai.client.token.usage`
  - `gen_ai.client.operation.duration`
  - `gen_ai.client.operation.time_to_first_chunk`
  - `agent.iterations`
  - `agent.tool.invocations`
- Enforced a strict metric attribute allowlist:
  - `gen_ai.system`
  - `gen_ai.request.model`
  - `gen_ai.operation.name`
  - `error.type`
  - `gen_ai.response.finish_reason`
  - `server.address`
- Explicitly excluded high-cardinality fields such as `user.id` and
  `session.id` from metrics.
- Added content-capture helpers that emit no message-body attributes by
  default and redact captured text when the env flag is enabled.
- Switched `otelmodel` semconv emission to the shared root constants and gate.

## Files

- `/tmp/llm-agent-otel/semconv_gen_ai.go`
- `/tmp/llm-agent-otel/semconv_gen_ai_test.go`
- `/tmp/llm-agent-otel/otelmetrics/otelmetrics.go`
- `/tmp/llm-agent-otel/otelmetrics/otelmetrics_test.go`
- `/tmp/llm-agent-otel/otelmodel/otelmodel.go`
- `/tmp/llm-agent-otel/otelmodel/otelmodel_test.go`
- `/tmp/llm-agent-otel/otelmodel/semconv_gen_ai.go`

## Verification

Executed against a temporary local `go.work` binding `llm-agent-otel` to the
current core repo checkout:

```bash
GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go test ./... -run 'Test(Metrics|ContentCapture|Cardinality)' -count=1
GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go build ./...
GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go test ./... -count=1
```

Result:

- targeted guardrail tests: pass
- `go build ./...`: pass
- full `go test ./...`: pass

## Notes

- Metric allowlisting is implemented at record time rather than exporter time,
  so high-cardinality fields never enter metric timeseries in the first place.
- Semconv emission is now centrally gated; wrappers can continue to emit spans
  safely while experimental `gen_ai.*` attributes stay opt-in.

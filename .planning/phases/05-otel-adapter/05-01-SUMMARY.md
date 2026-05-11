# Phase 05-01 Summary

Date: 2026-05-11
Repo: `llm-agent-otel`
Plan: [05-01-PLAN.md](/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/phases/05-otel-adapter/05-01-PLAN.md)

## Objective

Build `otelmodel.Wrap(llm.ChatModel)` so OpenTelemetry instrumentation can wrap
chat models without erasing optional capability interfaces.

## Delivered

- Added `otelmodel.Wrap(model, opts...) llm.ChatModel`.
- Preserved `llm.ToolCaller`, `llm.Embedder`, and `llm.StructuredOutputs`
  through capability-specific wrapper structs.
- Instrumented `Generate` with one span per call and response usage
  attributes.
- Instrumented `Stream` with one span per stream, a bounded
  `gen_ai.first_token` event, and end-of-stream usage/final-reason
  attributes.
- Added embedding instrumentation for wrapped models that implement
  `llm.Embedder`.
- Added compile-time interface assertions covering every wrapper shape.
- Added unit tests for capability preservation, single-span generate/stream,
  first-token event emission, and `WithTools(...)` rewrap behavior.

## Files

- `/tmp/llm-agent-otel/otelmodel/config.go`
- `/tmp/llm-agent-otel/otelmodel/otelmodel.go`
- `/tmp/llm-agent-otel/otelmodel/otelmodel_test.go`
- `/tmp/llm-agent-otel/otelmodel/semconv_gen_ai.go`
- `/tmp/llm-agent-otel/go.mod`
- `/tmp/llm-agent-otel/go.sum`

## Verification

Executed against a temporary local `go.work` binding `llm-agent-otel` to the
current core repo checkout:

```bash
GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go test ./otelmodel/... -count=1
GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go build ./otelmodel/...
```

Result:

- `go test`: pass
- `go build`: pass

## Notes

- `TracerProvider` is now stored on the base wrapper and reused when
  `WithTools(...)` / `WithSchema(...)` return rebound models, avoiding any
  fragile reverse lookup from `trace.Tracer`.
- Semconv keys are kept local in `otelmodel/semconv_gen_ai.go` for now; later
  Phase 5 plans can centralize or expand them if needed.

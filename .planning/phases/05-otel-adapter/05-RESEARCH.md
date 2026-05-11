# Phase 5: OTel Adapter - Research

**Researched:** 2026-05-11  
**Status:** Seeded from roadmap; semconv and exporter details will be checked
during execution

## Locked Inputs

- Instrumentation lives entirely in `llm-agent-otel`, not the core repo.
- Wrappers are decorator-style and must preserve capability interfaces.
- `gen_ai.*` semconv emission is opt-in through
  `OTEL_SEMCONV_STABILITY_OPT_IN=gen_ai_latest_experimental`.
- `grafana/otel-lgtm` remains the e2e demo target for the compose example.

## Known High-Risk Areas

1. Accidentally stripping `ToolCaller` / `Embedder` / `StructuredOutputs`
   during wrapping
2. Metric cardinality drift from user/session attributes
3. Content capture leaking message bodies by default
4. Streaming instrumentation producing span explosion
5. Semconv constant duplication across packages

## Research Tasks Deferred Into Execution

- Confirm current `gen_ai.*` experimental semconv naming expected by the chosen
  OTel Go release and Grafana stack
- Inspect available OTel log bridge options for `slog.Handler`
- Choose the smallest public config surface that still covers exporter,
  semconv, and content-capture toggles
- Confirm compose example wiring for OTLP HTTP on port `4318`

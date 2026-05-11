# Phase 5: OTel Adapter - Pattern Map

**Mapped:** 2026-05-11

## Reuse From Prior Phases

- Capability negotiation still hangs off `llm.ChatModel` plus optional
  interfaces
- Shared provider metadata comes from `ProviderInfo`
- Shared `llm.Usage` and `FinishReason` already normalize provider outputs

## New Patterns to Add

### Capability-preserving wrapper pattern

- Base wrapper stores the inner model
- Wrapper re-exposes `ToolCaller`, `Embedder`, and `StructuredOutputs` when
  the inner implements them
- Capability-specific methods wrap returned values again

### Single-span operation pattern

- `Generate`, `Stream`, and agent `Run` each create one top-level span
- Streaming records first-token timing and bounded events on that span

### Semconv constants pattern

- Attribute keys and metric names live in one `semconv_gen_ai.go`
- Emission checks the stability opt-in flag centrally

### Metric allowlist pattern

- Provider, model, operation, error type, finish reason, and server address are
  safe metric attributes
- User/session/request identifiers never land on metrics

### Default-safe content capture pattern

- No message content by default
- Redaction helper runs before any optional content capture

### Demo compose pattern

- Keep the example small: wrapped model/agent + OTLP exporter +
  `grafana/otel-lgtm`
- Use it as an integration proof, not as product infrastructure

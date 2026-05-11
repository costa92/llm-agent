# Phase 5: OTel Adapter - Context

**Gathered:** 2026-05-11  
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 5 wraps the now-stable `llm.ChatModel` and `agents.Agent` surfaces with
OpenTelemetry instrumentation in the sister repo `llm-agent-otel`.

This phase produces:

- `otelmodel.Wrap(ChatModel) ChatModel`
- `otelagent.Wrap(Agent) Agent`
- centralized `gen_ai.*` semantic-convention constants and config gating
- metric cardinality guardrails
- content-capture default-OFF behavior with redaction
- single-span streaming instrumentation with first-token timing
- slog-to-OTel bridge
- exporter + `compose/` example proving traces in `grafana/otel-lgtm`

Phase 5 explicitly covers:

- capability-preserving wrappers for `ToolCaller`, `Embedder`, and
  `StructuredOutputs`
- one-span-per-operation tracing for generate, stream, and agent runs
- low-cardinality metrics and high-cardinality span-only attributes
- semconv opt-in via `OTEL_SEMCONV_STABILITY_OPT_IN`
- default-safe content handling

Phase 5 does NOT cover:

- customer-support service integration
- Kubernetes deployment
- prompt-injection product policy
- removing deprecated core APIs

</domain>

<decisions>
## Implementation Decisions

### D-01: Wrappers preserve capability surfaces

- Wrapping a provider must never erase capabilities that the inner model
  implements.
- Capability preservation is done by re-exposing interfaces on wrapper values,
  not by widening the base `ChatModel` interface.

### D-02: Metrics stay intentionally low-cardinality

- Only a small allowlist of attributes lands on metrics.
- High-cardinality fields such as `user.id` and `session.id` stay on spans and
  logs only.

### D-03: Content capture is opt-in and redacted

- Prompt/response content is absent by default.
- When enabled, content is routed through a redaction helper before it reaches
  spans or logs.

### D-04: Streaming instrumentation is span-stable

- A streaming operation emits exactly one span.
- Chunk events become at most a small event set on that span, not one span per
  chunk.

### D-05: Semconv churn is isolated

- `gen_ai.*` attribute keys live in one constants file.
- Environment-variable opt-in is checked centrally, not ad hoc in each wrapper.

</decisions>

<canonical_refs>
## Canonical References

- `.planning/ROADMAP.md` - Phase 5 scope, pitfalls, and success criteria
- `.planning/REQUIREMENTS.md` - `OTEL-01` through `OTEL-10`
- `.planning/STATE.md` - milestone position after Phase 4 completion
- `.planning/phases/04-embeddings-rag-regression/04-05-SUMMARY.md`
- `/tmp/llm-agent-otel/`
- `/tmp/llm-agent-providers/`
- `llm/`
- root-agent package in this repo

</canonical_refs>

<specifics>
## Success Markers to Preserve

- Wrapping stays additive: existing model and agent behavior must still work.
- Core `llm-agent` remains stdlib-only; OTel deps stay isolated in the sister
  repo.
- Provider capability truth from earlier phases remains observable after
  wrapping.

</specifics>

# Phase 6: Reference Customer-Support Service - Pattern Map

**Mapped:** 2026-05-11

## Reuse From Prior Phases

- `llm.ChatModel` + capability interfaces stay the model integration seam
- Provider truth still comes from `ProviderInfo`
- `otelmodel.Wrap(...)` and `otelagent.Wrap(...)` are the observability seams
- `agents.Agent` remains the service-facing runtime contract

## New Patterns to Add

### Bootstrap seam pattern

- `config` package owns env parsing + defaults
- `app` package owns tracer provider, agent factory, server lifecycle
- `cmd/server` stays tiny and only wires signal handling to `app.Run`

### Provider-switch pattern

- One config value selects the active chat provider
- Provider-specific API key / base URL knobs stay local to the factory
- Default model selection is provider-aware, not duplicated in handlers

### Graceful shutdown pattern

- Service start is context-driven
- Shutdown path closes `http.Server` and tracer provider exactly once
- Tests verify cancellation-driven exit, not only constructor success

### Future middleware pattern

- HTTP handler tree starts from `ServeMux`
- Later plans add trace-id headers, caps, auth placeholders, and guardrails as
  middleware layers rather than rewriting server startup

### Demo-docs drift guard

- README must always describe the repo's actual current state
- Once bootstrap code lands, "skeleton only / no Go source" language must be
  removed immediately

# Phase 6: Reference Customer-Support Service - Research

**Researched:** 2026-05-11  
**Status:** Seeded from roadmap; security and compose details will deepen during execution

## Locked Inputs

- Service repo is the integration point for core + providers + OTel.
- `06-01` is a bootstrap plan, not the full product surface.
- Runtime provider selection must align with `LLM_PROVIDER=openai|anthropic|ollama`.
- OTel bootstrap should reuse `llm-agent-otel` exporter defaults rather than
  inventing a parallel config surface.
- K8s remains explicitly out of scope for v0.3.

## Known High-Risk Areas

1. Letting bootstrap sprawl into endpoint/business logic before transport seams
   are stable
2. Hiding provider selection in ad hoc env reads spread across handlers
3. Treating telemetry shutdown as optional and leaking exporters on exit
4. Baking prompt-injection or cap logic directly into bootstrap instead of
   reserving middleware/service seams
5. Leaving README/runtime docs in a Phase-0-only state after Go source lands

## Research Tasks Deferred Into Execution

- Re-check prompt-injection guardrail patterns before `06-07`
- Choose the narrowest persistence abstraction that supports SQLite now and
  Postgres later without long-lived agent actors
- Confirm compose stack timings and dashboard provisioning path for `06-08`
- Decide exact SSE event shape and trace-header propagation pattern for `06-02`

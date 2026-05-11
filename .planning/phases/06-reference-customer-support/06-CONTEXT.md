# Phase 6: Reference Customer-Support Service - Context

**Gathered:** 2026-05-11  
**Status:** Ready for execution

<domain>
## Phase Boundary

Phase 6 builds the deployable reference service in the sister repo
`llm-agent-customer-support`.

This phase composes the already-finished provider adapters, core agent/runtime
surface, and OTel wrappers into one HTTP service that can be started locally
with `docker compose up`.

Phase 6 produces:

- service bootstrap in `cmd/server/main.go`
- env-driven provider and telemetry configuration
- HTTP API for sync + streaming chat
- provider-switching and embedding-provider independence
- multi-agent customer-support flow using RAG + `StateGraph` + tools
- durable session storage
- hard caps and panic switch wiring
- prompt-injection guardrails
- compose/demo assets, dashboards, and operator documentation

Phase 6 explicitly covers:

- real provider selection (`openai`, `anthropic`, `ollama`)
- OTel tracer lifecycle and trace correlation at the service boundary
- deployable local-demo defaults with explicit "demo only" caveats
- service-level guardrails that fail closed

Phase 6 does NOT cover:

- Kubernetes manifests or Helm charts
- production authentication/authorization
- multi-region or multi-tenant deployment
- deprecation removal in the core repo

</domain>

<decisions>
## Implementation Decisions

### D-01: Phase 6 starts from a thin bootstrap

- `06-01` only lands config parsing, provider/agent factory seams, telemetry
  setup, and graceful shutdown.
- HTTP business endpoints, storage, guardrails, and compose assets remain in
  later plans.

### D-02: Real providers are selected by config, not hidden behind build tags

- The service binary keeps one code path with runtime provider selection.
- Per-provider defaults live in config, but model construction happens in one
  factory package.

### D-03: OTel lives at the boundary, not inside the core repo

- Tracer provider creation uses `llm-agent-otel`.
- Models and agents are wrapped before entering the service layer so later HTTP
  handlers inherit traces automatically.

### D-04: Guardrails are layered, not mixed into one giant handler

- Config, provider/agent construction, transport handlers, caps, storage, and
- injection defense each get their own package seams.
- Later plans can add middleware and storage without rewriting bootstrap code.

### D-05: Demo-first defaults remain explicit

- Local defaults target `:8080`, OTLP HTTP on `:4318`, and simple readiness for
  a single-node compose demo.
- Production hardening stays documented as out-of-scope work, not implied by
  the bootstrap.

</decisions>

<canonical_refs>
## Canonical References

- `.planning/ROADMAP.md` - Phase 6 scope, ordering, and success criteria
- `.planning/REQUIREMENTS.md` - `REFSVC-01` through `REFSVC-13`
- `.planning/STATE.md` - umbrella project execution state
- `/tmp/llm-agent-customer-support/`
- `/tmp/llm-agent-providers/`
- `/tmp/llm-agent-otel/`
- `README.md` in each sibling repo for cross-repo iteration expectations

</canonical_refs>

<specifics>
## Success Markers to Preserve

- The service repo may depend on external modules; the core repo must not.
- Bootstrap code should already fit later HTTP/API work without renaming major
  package boundaries.
- Graceful shutdown must close both the HTTP server and telemetry provider.
- Provider selection should already match the later `LLM_PROVIDER=...` contract.

</specifics>

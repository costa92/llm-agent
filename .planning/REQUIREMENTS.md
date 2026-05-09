# Requirements: llm-agent v0.3

**Defined:** 2026-05-10
**Core Value:** The core `llm-agent` module stays stdlib-only and zero-dep — anyone can `go get` it and read every line. Providers, telemetry, and reference services live in sister repos so users opt into dependencies one package at a time.

This document captures the v0.3 milestone scope across 4 sibling Go modules. Categories are scoped by repo where possible (CORE = this repo; OAI/ANT/OLL/CONF = `llm-agent-providers`; OTEL = `llm-agent-otel`; REFSVC = `llm-agent-customer-support`).

## v1 Requirements

### Multi-repo Infrastructure (`llm-agent` umbrella)

- [ ] **INFRA-01**: 4 sibling Go modules exist with their own `go.mod`: `llm-agent` (this repo), `llm-agent-providers`, `llm-agent-otel`, `llm-agent-customer-support`
- [ ] **INFRA-02**: `go.work` is `.gitignore`d in every repo; CI runs with `GOWORK=off`
- [ ] **INFRA-03**: A `Makefile` or shell script in each sister repo writes a sibling-aware `go.work` for local cross-repo dev
- [ ] **INFRA-04**: CI gate rejects `replace` directives on tagged-release branches (release-time check, not PR-time)
- [ ] **INFRA-05**: Umbrella CI in `llm-agent` builds all 4 repos against `llm-agent` HEAD on every PR (catches cross-repo break)
- [ ] **INFRA-06**: README in each sister repo documents the cross-repo iteration pattern (`go.work` recommended; `replace` only as a documented temporary escape hatch)
- [ ] **INFRA-07**: Versioning policy documented across all 4 repos: `llm-agent` v0.3.x; sister repos start at v0.1.x; CHANGELOG `### Breaking` section per repo

### Core Capability Interfaces (`llm-agent`, package `llm/v2`)

- [ ] **CORE-01**: New `llm/v2` package defines `ChatModel` base interface (Generate + Stream) — every provider implements this
- [ ] **CORE-02**: `ToolCaller` capability interface defines `WithTools(tools) ToolCaller` (immutable; returns new value, does NOT mutate receiver)
- [ ] **CORE-03**: `Embedder` capability interface defines `Embed(ctx, []string) ([]Vector, Usage, error)` — separate from `ChatModel` because Anthropic does not support embeddings
- [ ] **CORE-04**: `StructuredOutputs` capability interface defines `WithSchema(schema) ChatModel`
- [ ] **CORE-05**: Typed `StreamEvent` union with `Kind` enum (TextDelta / ToolCallStart / ToolCallArgsDelta / ToolCallEnd / Done) and stable per-tool-call `Index` field — adapters emit native granularity, never lowest-common-denominator chunks (K1)
- [ ] **CORE-06**: `ProviderInfo` struct returned by `ChatModel.Info()` reflects the bound model's capabilities (per-(provider × model), not per-provider) — provider instances bind a model at construction time (K2)
- [ ] **CORE-07**: Mock implementations (`ScriptedLLM`-style) for `ChatModel` + each capability, suitable for use in agent tests
- [ ] **CORE-08**: Existing `llm.Client` (v0.2 surface) remains callable, marked Deprecated with godoc + removal target version
- [ ] **CORE-09**: Migration guide in `docs/migration-v0.2-to-v0.3.md` (or similar) — concrete diff examples for each agent paradigm
- [ ] **CORE-10**: Agent paradigms (Simple/ReAct/Reflection/PlanSolve/FunctionCall) refactored to consume `ChatModel` + type-assert for capabilities + fall back gracefully (e.g., scratchpad templating when `ToolCaller` unavailable)
- [ ] **CORE-11**: Provider Author Guide (`PROVIDER_AUTHORING.md`) — what an adapter must do to claim conformance, including streaming + tool-call wire-format expectations and capability-degradation rules

### Cross-provider Conformance (`llm-agent-providers/internal/contract`)

- [ ] **CONF-01**: Shared httptest-based fixture harness — same fixtures run against every adapter
- [ ] **CONF-02**: Generate (sync) conformance: request shape, response shape, error taxonomy, finish-reason normalization
- [ ] **CONF-03**: Streaming conformance: TTFT, TextDelta ordering, partial-usage-on-error, cancel-mid-stream cleanup
- [ ] **CONF-04**: Tool-call conformance: parallel tool calls (OpenAI), multi-block tool-use (Anthropic), capability-degraded path (Ollama llama2)
- [ ] **CONF-05**: Tool-call dedupe: `(message_id, tool_use_id)` keying tested across all adapters
- [ ] **CONF-06**: Embedding conformance: dimension assertion, batch-embed shape, `ErrNotSupported` on Anthropic
- [ ] **CONF-07**: Recorded fixture script per provider (capture from real API for replay)
- [ ] **CONF-08**: `goleak` integration in conformance suite — no goroutines leak after a test exits

### OpenAI Provider (`llm-agent-providers/openai`)

- [ ] **OAI-01**: Implements `ChatModel.Generate` against `github.com/openai/openai-go/v3` (Responses API for new code; Chat Completions fallback for older models)
- [ ] **OAI-02**: Implements streaming with native delta accumulation (keyed by `tool_calls[].index`, not by name) (Pitfall 1)
- [ ] **OAI-03**: Implements `ToolCaller` with native function-calling (parallel tool calls supported)
- [ ] **OAI-04**: Implements `Embedder` against OpenAI Embeddings API
- [ ] **OAI-05**: Typed error taxonomy: `RateLimitError`, `AuthError`, `InvalidRequestError`, `TransientError` (mapped from openai-go errors)
- [ ] **OAI-06**: Three-state cost record: `Reported` / `Estimated` / `Unknown` — `stream_options.include_usage=true` enforced for streams (K4)
- [ ] **OAI-07**: Retry state machine `Connecting → FirstByte → Streaming → Done` — never retries after first byte delivered (Pitfall 4)
- [ ] **OAI-08**: Passes the cross-provider conformance suite

### Anthropic Provider (`llm-agent-providers/anthropic`)

- [ ] **ANT-01**: Implements `ChatModel.Generate` against `github.com/anthropics/anthropic-sdk-go`
- [ ] **ANT-02**: Implements streaming using `Message.Accumulate(event)` — `partial_json` buffered until `content_block_stop` (NOT `message_stop`); second tool-use block at higher index does NOT overwrite first (Pitfall 2)
- [ ] **ANT-03**: Implements `ToolCaller` with native tool use (Beta tool runner where applicable)
- [ ] **ANT-04**: Does NOT implement `Embedder` — `ProviderInfo.Capabilities` returns `Embedder=false`; calls return `ErrNotSupported` cleanly (gap is documented, not papered over)
- [ ] **ANT-05**: Typed error taxonomy mapped from anthropic-sdk-go errors
- [ ] **ANT-06**: Three-state cost record + retry state machine consistent with K4
- [ ] **ANT-07**: Passes the cross-provider conformance suite

### Ollama Provider (`llm-agent-providers/ollama`)

- [ ] **OLL-01**: Implements `ChatModel.Generate` against `github.com/ollama/ollama/api`
- [ ] **OLL-02**: Implements streaming — bridges callback-based stream into channel/iterator with proper `ctx.Done()` propagation; no goroutine leaks on cancel
- [ ] **OLL-03**: Implements `ToolCaller` with per-model strategy table (Llama 3 vs. Qwen3 vs. Mistral parsing differences) — `ProviderInfo.Capabilities.ToolCaller` is per-bound-model (Pitfall 19)
- [ ] **OLL-04**: Implements `Embedder` against Ollama's `/api/embed`
- [ ] **OLL-05**: Typed error taxonomy
- [ ] **OLL-06**: Three-state cost record (note: Ollama is local; `Reported` cost is $0 by definition; usage tokens still tracked) + retry state machine consistent with K4
- [ ] **OLL-07**: Passes the cross-provider conformance suite
- [ ] **OLL-08**: Nightly CI job runs the conformance suite against a real Ollama container (testcontainers-go); PR CI runs mocks only

### OpenTelemetry Adapter (`llm-agent-otel`)

- [ ] **OTEL-01**: `otelmodel.Wrap(ChatModel) ChatModel` decorator — wrapping preserves capability interfaces (re-implements `ToolCaller`, `Embedder`, `StructuredOutputs` on the wrapped value when the inner has them) (K3)
- [ ] **OTEL-02**: `otelagent.Wrap(Agent) Agent` decorator — emits `invoke_agent` / `chat` / `execute_tool` span tree per OTel `gen_ai.*` semconv
- [ ] **OTEL-03**: `gen_ai.*` semconv attribute names centralized in one constants file; emission gated behind `OTEL_SEMCONV_STABILITY_OPT_IN=gen_ai_latest_experimental`; bumping the major version of `llm-agent-otel` is the migration mechanism when upstream stabilizes (K5, Pitfall 10)
- [ ] **OTEL-04**: Metrics emitted: `gen_ai.client.token.usage`, `gen_ai.client.operation.duration`, `gen_ai.client.operation.time_to_first_chunk` + framework-level `agent.iterations`, `agent.tool.invocations`
- [ ] **OTEL-05**: Metric attribute allowlist (~6 attrs: provider, model, operation, error.type, finish_reason, server.address) — high-cardinality attrs (user.id, session.id) on spans only; CI test asserts that 1000 distinct user IDs produce ≤50 attribute combinations (Pitfall 7)
- [ ] **OTEL-06**: Content capture (prompts/responses) DEFAULT OFF; respects `OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT`; redactor utility available (Pitfall 8)
- [ ] **OTEL-07**: Span-explosion test — a 500-chunk stream produces exactly 1 span (chunks become span events, not separate spans) (Pitfall 9)
- [ ] **OTEL-08**: `slog.Handler` bridge to OTel logs; emits structured fields (trace_id, span_id, gen_ai.* fields)
- [ ] **OTEL-09**: OTLP HTTP exporter on port 4318 (default); OTLP gRPC opt-in
- [ ] **OTEL-10**: Documentation + example: how to wrap an agent + provider with OTel, including end-to-end traces visible in `grafana/otel-lgtm`

### Reference Customer-Support Service (`llm-agent-customer-support`)

- [ ] **REFSVC-01**: `cmd/server/main.go` — env-var config, OTel init, agent factory, `http.Server` with graceful shutdown (SIGINT/SIGTERM)
- [ ] **REFSVC-02**: HTTP API: `POST /chat` (one-shot JSON), `POST /chat/stream` (SSE), `GET /healthz`, `GET /readyz`
- [ ] **REFSVC-03**: `X-Trace-Id` response header on every request; client can correlate to OTel traces
- [ ] **REFSVC-04**: Provider switch via env var `LLM_PROVIDER=openai|anthropic|ollama` — same service binary serves all three; `EMBEDDING_PROVIDER=openai|ollama` independent (Anthropic chat + OpenAI/Ollama embeddings is a supported combo)
- [ ] **REFSVC-05**: Multi-agent customer-support flow: RAG knowledge lookup + `StateGraph` triage routing + native tool calling — extends the existing `support_triage` example
- [ ] **REFSVC-06**: Session storage: SQLite (dev), Postgres (prod) — agents constructed per-request, NOT long-running session actors; session state lives in DB
- [ ] **REFSVC-07**: Hard caps wired in from Day 1: `MAX_TOKENS_PER_REQUEST`, `MAX_TOOL_CALLS_PER_AGENT_LOOP`, `MAX_REQUESTS_PER_IP_PER_MINUTE`, `RETRY_MAX_ATTEMPTS`, `DAILY_TOKEN_BUDGET` (K7, Pitfall 17)
- [ ] **REFSVC-08**: `DISABLE_LLM=1` panic switch — flips the service to "all providers fail closed" without restart (K7)
- [ ] **REFSVC-09**: Prompt-injection guardrails Day 1: input filter, tool allowlist with server-side `user_id` enforcement (NEVER trust LLM-supplied IDs), retrieved RAG content marked as untrusted in system prompt (Pitfall 18)
- [ ] **REFSVC-10**: `compose.yaml`: app + Ollama + `grafana/otel-lgtm` (single-container observability stack); `docker compose up` reaches "service ready, traces visible in Grafana" in <60s
- [ ] **REFSVC-11**: Pre-provisioned Grafana dashboard JSON committed to repo: latency p50/p99, tokens/min, cost/min, error rate, tool-call success ratio
- [ ] **REFSVC-12**: Tail-sampling collector config: 100% errors, 100% latency >5s, 10% otherwise; `decision_wait=30s` (Pitfall 11)
- [ ] **REFSVC-13**: README clearly marks "demo only — production deployment requires X, Y, Z hardening" (single-container otel-lgtm, no auth on /chat, dev keys, etc.)

### Deprecation Removal (`llm-agent` v0.4 cut)

- [ ] **DEPRC-01**: Audit complete — zero internal users of `llm.Client` remain in `llm-agent` repo (examples + tests use `llm/v2`)
- [ ] **DEPRC-02**: `llm.Client` and v0.2-era types removed from `llm-agent` v0.4.0
- [ ] **DEPRC-03**: CHANGELOG `### Breaking` section documents the removal with migration link
- [ ] **DEPRC-04**: Sister repos bump `require github.com/costa92/llm-agent v0.4.x`; coordinated tag across all 4

## v2 Requirements

Deferred to v0.4 / future. Tracked but not in v0.3 roadmap.

### Differentiator Features (P2)

- **DIFF-01**: Structured-output / JSON-schema mode — `GenerateRequest.ResponseFormat` honored by adapters that support it
- **DIFF-02**: First-class prompt caching — `Message.CacheControl` (Anthropic explicit; OpenAI implicit auto); OTel `gen_ai.usage.cache_read.input_tokens`
- **DIFF-03**: Per-request cost guardrail — `agent.WithMaxCostUSD(...)` halts the agent loop when projected cost crosses threshold
- **DIFF-04**: Versioned `cost.Table` — provider-rate table feeding the cost guardrail
- **DIFF-05**: Replay testing — `RecordingLLM` captures real responses to fixture files for deterministic re-runs
- **DIFF-06**: Production-split observability compose — separate Loki / Tempo / Prometheus / Grafana / Collector services (vs the v0.3 single-container demo)
- **DIFF-07**: Kubernetes manifests / Helm chart for refsvc (with kind/k3d CI from the start — never half-shipped)

## Out of Scope

Explicit v0.3 exclusions. Reasoning included so they don't get re-added.

| Feature | Reason |
|---------|--------|
| Vision / multimodal LLM support | Wire format is its own surface; dilutes v0.3 wire-format work; deserves its own milestone |
| Vector store backends (Pinecone/Weaviate/Pgvector) | Existing `InMemoryStore` is enough for refsvc demo; sister-repo `llm-agent-vectorstores` is later |
| Real RL training (in-process trainer) | `rl/` keeps Python TRL bridge stub; training is a wholly different scope |
| Cross-framework bridges (LangChain/LlamaIndex/CrewAI) | Keeps surface area pure-Go; users compose at agent level if needed |
| Production-grade distributed a2a/anp (service discovery, rate limiting, retry/circuit-breaking) | Separate milestone needing security review; v0.3 keeps `comm/` at toy/demo level |
| GUI / Studio / playground UI | Out of band for a Go library; not a framework concern |
| v1.0 stability commitment | Real-world feedback gating v1.0 hasn't accumulated; ship v0.3, learn, then promote |
| Single-repo monolith with build tags or hard provider deps in `llm-agent` core | Violates Core Value (stdlib-only, zero-dep, readable) |
| Anthropic Embeddings adapter (Voyage AI, etc.) | Sister-repo "kitchen-sink avoidance" — refsvc users compose `LLM_PROVIDER=anthropic` + `EMBEDDING_PROVIDER=openai\|ollama` |

## Traceability

Final phase mapping ratified by `gsd-roadmapper` on 2026-05-10. Every v1 requirement maps to exactly one phase. See `.planning/ROADMAP.md` for phase details and success criteria.

| Requirement | Phase | Status |
|-------------|-------|--------|
| INFRA-01 | Phase 0 | Pending |
| INFRA-02 | Phase 0 | Pending |
| INFRA-03 | Phase 0 | Pending |
| INFRA-04 | Phase 0 | Pending |
| INFRA-05 | Phase 0 | Pending |
| INFRA-06 | Phase 0 | Pending |
| INFRA-07 | Phase 0 | Pending |
| CORE-01 | Phase 0 | Pending |
| CORE-02 | Phase 0 | Pending |
| CORE-03 | Phase 0 | Pending |
| CORE-04 | Phase 0 | Pending |
| CORE-05 | Phase 0 | Pending |
| CORE-06 | Phase 0 | Pending |
| CORE-07 | Phase 0 | Pending |
| CORE-08 | Phase 0 | Pending |
| CORE-09 | Phase 0 | Pending |
| CORE-10 | Phase 3 | Pending |
| CORE-11 | Phase 1 | Pending |
| CONF-01 | Phase 1 | Pending |
| CONF-02 | Phase 1 | Pending |
| CONF-03 | Phase 2 | Pending |
| CONF-04 | Phase 3 | Pending |
| CONF-05 | Phase 3 | Pending |
| CONF-06 | Phase 4 | Pending |
| CONF-07 | Phase 1 | Pending |
| CONF-08 | Phase 1 | Pending |
| OAI-01 | Phase 1 | Pending |
| OAI-02 | Phase 2 | Pending |
| OAI-03 | Phase 3 | Pending |
| OAI-04 | Phase 4 | Pending |
| OAI-05 | Phase 1 | Pending |
| OAI-06 | Phase 2 | Pending |
| OAI-07 | Phase 2 | Pending |
| OAI-08 | Phase 4 | Pending |
| ANT-01 | Phase 1 | Pending |
| ANT-02 | Phase 2 | Pending |
| ANT-03 | Phase 3 | Pending |
| ANT-04 | Phase 4 | Pending |
| ANT-05 | Phase 1 | Pending |
| ANT-06 | Phase 2 | Pending |
| ANT-07 | Phase 4 | Pending |
| OLL-01 | Phase 1 | Pending |
| OLL-02 | Phase 2 | Pending |
| OLL-03 | Phase 3 | Pending |
| OLL-04 | Phase 4 | Pending |
| OLL-05 | Phase 1 | Pending |
| OLL-06 | Phase 2 | Pending |
| OLL-07 | Phase 4 | Pending |
| OLL-08 | Phase 1 | Pending |
| OTEL-01 | Phase 5 | Pending |
| OTEL-02 | Phase 5 | Pending |
| OTEL-03 | Phase 5 | Pending |
| OTEL-04 | Phase 5 | Pending |
| OTEL-05 | Phase 5 | Pending |
| OTEL-06 | Phase 5 | Pending |
| OTEL-07 | Phase 5 | Pending |
| OTEL-08 | Phase 5 | Pending |
| OTEL-09 | Phase 5 | Pending |
| OTEL-10 | Phase 5 | Pending |
| REFSVC-01 | Phase 6 | Pending |
| REFSVC-02 | Phase 6 | Pending |
| REFSVC-03 | Phase 6 | Pending |
| REFSVC-04 | Phase 6 | Pending |
| REFSVC-05 | Phase 6 | Pending |
| REFSVC-06 | Phase 6 | Pending |
| REFSVC-07 | Phase 6 | Pending |
| REFSVC-08 | Phase 6 | Pending |
| REFSVC-09 | Phase 6 | Pending |
| REFSVC-10 | Phase 6 | Pending |
| REFSVC-11 | Phase 6 | Pending |
| REFSVC-12 | Phase 6 | Pending |
| REFSVC-13 | Phase 6 | Pending |
| DEPRC-01 | Phase 7 | Pending |
| DEPRC-02 | Phase 7 | Pending |
| DEPRC-03 | Phase 7 | Pending |
| DEPRC-04 | Phase 7 | Pending |

**Coverage:**
- v1 requirements: 65 total
- Mapped: 65/65 ✓
- Orphans: 0
- Duplicates: 0

**Cross-cuts (book-keeping, not duplicates):** Several requirements have implementation surface across multiple phases — they're listed against the phase that delivers their primary success criterion. K4 (cost record + retry SM, OAI-06/07, ANT-06, OLL-06) is designed in Phase 1's typed errors, enforced in Phase 2's streaming, consumed in Phase 3's tool dedupe, surfaced in Phase 5's OTel attrs. CORE-11 (Provider Author Guide) is incremental: v0.1 in Phase 1, v0.2 in Phase 2, v0.3 in Phase 4. The `OAI-08 / ANT-07 / OLL-07` "passes complete suite" markers are validated incrementally at every gate but the final tick lands in Phase 4 when the suite is complete.

---
*Requirements defined: 2026-05-10*
*Last updated: 2026-05-10 — traceability ratified by gsd-roadmapper; phase mapping locked*

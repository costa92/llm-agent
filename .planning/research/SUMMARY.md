# Project Research Summary

**Project:** llm-agent v0.3 — provider adapters (OpenAI/Anthropic/Ollama) + OpenTelemetry + deployable customer-support reference service
**Domain:** Go LLM agent framework, multi-repo umbrella (4 repos)
**Researched:** 2026-05-10
**Confidence:** HIGH overall

This summary is the cross-cut. The 4 research files (`STACK.md`, `FEATURES.md`, `ARCHITECTURE.md`, `PITFALLS.md`) carry the detail; this file surfaces the decisions, conflicts, and load-bearing risks the roadmapper must resolve before phase planning.

---

## Executive Summary

The four research streams agree on the shape of the milestone: 4 sibling Go modules (`llm-agent` core stays stdlib-only; `llm-agent-providers`, `llm-agent-otel`, `llm-agent-customer-support` carry deps), three official-SDK provider adapters that converge on a small `ChatModel + ToolCaller + Embedder + StructuredOutputs` interface set in a new `llm/v2` package, OTel attached as a decorator (`Wrap(ChatModel) ChatModel`) emitting `gen_ai.*` semconv attributes behind a stability opt-in, and a `docker compose` reference service using `grafana/otel-lgtm` as the all-in-one backend. The keystone abstraction is the streaming event union (`StreamEvent.Kind` with stable per-tool-call indexing) — it has to express OpenAI's per-index delta model AND Anthropic's per-content-block model AND Ollama's whole-call-at-once model without leaking provider semantics into the agent layer.

The dominant risk profile is *not* technical novelty — it's evolution risk. `gen_ai.*` semconv is still **Development** as of 2026-05-10 (will rename); Ollama tool-call wire format is per-model (not per-provider), so the capability-negotiation contract has to take a model parameter; provider streaming has 3+ ways to lose money silently (retry-after-first-byte double-billing, mid-stream goroutine leaks, PII captured by default in span attributes). All three streams of risk are mitigated by the *same* design move: small interfaces + decorator wrappers + opt-in flags + allowlists, never bitmasks or fat interfaces. PITFALLS.md is the binding document on this front; STACK.md and ARCHITECTURE.md must defer to it where they conflict.

The single most important sequencing question for the roadmapper: **build OpenAI fully then validate via Anthropic** (Architecture's recommendation, Phase 3 → 5) versus **walking-skeleton-first across all three** (Pitfalls' Pitfall 20, "Phase 1 perfectionism"). Both researchers are right about their concern. Resolution is below in *Conflicts to Resolve*.

---

## Key Findings

### Recommended Stack

(Detail: `STACK.md`)

The stack is opinionated and well-understood — no novel decisions. Each adapter uses the official Go SDK because (a) wire-format work tracks daily by the upstream vendor, (b) we get Responses-API / Messages-API / Embed updates for free, (c) hand-roll only where no SDK exists (Ollama is the edge case — official `ollama/api` exists but is pre-1.0, so pin minor and accept some churn).

**Core technologies:**
- `github.com/openai/openai-go/v3` v3.35.0 — OpenAI adapter; full Responses/Chat/Embeddings/streaming/tools coverage
- `github.com/anthropics/anthropic-sdk-go` v1.41.0 — Anthropic adapter; `Message.Accumulate(event)` for streaming; **no embeddings endpoint** (capability gap, not a bug)
- `github.com/ollama/ollama/api` v0.23.2 — Ollama adapter; callback-based streaming (must bridge to channel/iterator)
- `go.opentelemetry.io/otel` v1.43.0 + OTLP HTTP exporter (port 4318) by default; gRPC opt-in
- `grafana/otel-lgtm` (single container: Loki+Tempo+Prometheus+Grafana+Collector) as the **demo** observability stack — production deployments split this into 5 services
- `log/slog` (stdlib) bridged to OTel logs; no zap/zerolog
- `testcontainers-go/modules/ollama` for nightly live tests; `httptest.Server` + recorded fixtures for PR CI

**Deployment shape (refsvc):**
- `compose.yaml` ~50 lines: app + ollama + lgtm. One-line provider switch via `LLM_PROVIDER=openai|anthropic|ollama`.
- Helm/K8s explicitly **deferred** (out-of-scope for v0.3 per PROJECT.md and Pitfall 16 — half-shipping K8s is worse than not shipping it).

### Expected Features

(Detail: `FEATURES.md`)

Feature competition is against `langchaingo` (breadth leader, no production posture), `eino` (production-shaped but heavy deps), and `genkit Go` (best deploy story but Google-Cloud-shaped). The cross-product position v0.3 occupies — **stdlib core + 3-provider conformance + OTel-semconv-conformant + deployable reference** — is unique.

**Must have (table stakes — P1):**
- Generate + Stream + native tool calling, all 3 providers
- Embeddings on OpenAI + Ollama; Anthropic returns `ErrNotSupported` (document the gap, point to Voyage AI)
- `llm/v2` capability interfaces (`ChatModel` always; `ToolCaller`, `Embedder`, `StructuredOutputs` optional via type assertion)
- OTel `gen_ai.*` semconv attributes (request/response/usage/finish-reason); spans for `chat`, `execute_tool`, `invoke_agent`; metrics `gen_ai.client.token.usage`, `gen_ai.client.operation.duration`, TTFT
- Cross-provider conformance test suite (httptest + recorded fixtures) — IS the Provider Author Guide's contract
- Reference service: `POST /chat`, `/chat/stream` (SSE), `/healthz`, `/readyz`, graceful shutdown, `X-Trace-Id` response header, session state via existing `memory.Manager`, env-var config
- Pre-provisioned Grafana dashboard JSON (latency p50/p99, tokens/min, cost/min, error rate, tool-call success ratio)
- Typed error taxonomy (`RateLimitError` / `AuthError` / `InvalidRequestError` / `TransientError`)

**Should have (differentiators, P2 — slot into v0.3.x or v0.4):**
- Structured-output / JSON-schema mode (`GenerateRequest.ResponseFormat`)
- First-class prompt caching (`Message.CacheControl`) — Anthropic explicit, OpenAI auto, OTel `gen_ai.usage.cache_read.input_tokens`
- Per-request cost-limit guardrail (`agent.WithMaxCostUSD`)
- Provider rate-table (versioned `cost.Table`) — feeds the cost guardrail
- Prompt-injection guardrail recipe in refsvc (input filter + output filter + tool allowlist)
- Replay-testing (`RecordingLLM` capturing real responses to fixture files)

**Defer (v0.4+, explicitly out of scope per PROJECT.md):**
- Vision / multimodal (separate milestone — wire-format is its own surface)
- Vector store backends (Pinecone/Weaviate/Pgvector) — InMemoryStore is enough for refsvc; sister-repo `llm-agent-vectorstores` is later
- Distributed a2a/anp production-shape
- GUI / Studio / playground UI
- Real RL training (Python TRL bridge stub remains)
- Cross-framework bridges (LangChain/LlamaIndex)
- v1.0 stability commitment

### Architecture Approach

(Detail: `ARCHITECTURE.md`)

Multi-repo siblings: providers and otel both depend on `llm-agent` (specifically `llm/v2`); they NEVER depend on each other. Composition happens in the leaf consumer (refsvc). This is the same dependency shape `go.opentelemetry.io/contrib/instrumentation/...` uses — proven pattern.

**Three load-bearing pattern choices:**

1. **Small interfaces + type assertion + `ProviderInfo` hint struct** for capability negotiation (NOT bitmask, NOT fat interface). Mirrors Eino's `BaseChatModel + ToolCallingChatModel`; rejects langchaingo's fat-interface-with-CallOption escape hatch. **Caveat below in Conflicts.**
2. **Decorator wrapper** for OTel (`otelmodel.Wrap(inner) ChatModel`) — composes with retry/cache wrappers; opt-in; zero coupling between core and otel. Hooks/callbacks rejected because they leak observability into core's API surface.
3. **Typed `StreamEvent` union** with stable per-tool-call indexing — solves the OpenAI-delta vs. Anthropic-content-block divergence at the abstraction layer, NOT lowest-common-denominator. Adapters emit native granularity; consumers that don't care use a helper to accumulate. This is the single most important new type in `llm/v2`.

**Major components:**
1. `llm/v2/` (core repo) — `ChatModel` (base), `ToolCaller`/`Embedder`/`StructuredOutputs` (optional capabilities), `ProviderInfo` (capability hint), `StreamReader`/`StreamEvent` (typed event union)
2. `agent/` (core repo, refactored) — paradigms (Simple/ReAct/Reflection/PlanSolve/FunctionCall) consume `ChatModel`, type-assert for `ToolCaller`, fall back to scratchpad templating when missing
3. `llm-agent-providers/{openai,anthropic,ollama}/` — adapters; each implements as many capabilities as its SDK supports
4. `llm-agent-providers/internal/contract/` — shared httptest conformance suite, run against all three adapters
5. `llm-agent-otel/` — `otelmodel.Wrap` + `otelagent.Wrap` + slog handler + metrics
6. `llm-agent-customer-support/` — HTTP handlers, session store (sqlite dev / postgres prod), per-request agent construction (NOT long-running session actors), docker-compose, Grafana dashboard JSON

### Critical Pitfalls

(Detail: `PITFALLS.md` — 22 pitfalls indexed, plus Performance/Security/Integration tables and a 22-item "looks done but isn't" checklist)

The five that, if unaddressed, sink the milestone:

1. **OpenAI streaming `tool_calls` must key by `index` (not `name`)** — `parallel_tool_calls=true` interleaves chunks; keying wrong silently corrupts arguments. Conformance test with interleaved indices is the gate. (Pitfall 1)
2. **Anthropic `partial_json` must be buffered until `content_block_stop` (NOT `message_stop`)** — different parse trigger from OpenAI; second tool-use block at higher `index` mustn't overwrite first. (Pitfall 2)
3. **Goroutine leaks on context-cancel during streaming are inevitable without `goleak` in CI** — adapter `Close()` must propagate cancel + close `resp.Body` on every exit path. Stdlib-only constraint applies to core only; sister repos can take `go.uber.org/goleak` as a test dep. (Pitfall 3)
4. **PII in OTel span attributes is the highest-blast-radius default** — capture must be DEFAULT OFF; respect `OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT`; refsvc compose ships content-capture OFF. GDPR exposure is the failure mode. (Pitfall 8)
5. **Capabilities are per-(provider × model), NOT per-provider** — Ollama's `qwen3-coder` emits XML where `llama3` emits `<|python_tag|>`. `ProviderInfo` as currently designed in ARCHITECTURE.md takes only the provider type, not the model. **This is the design hole that conflicts most directly with Pitfall 6.** Resolution required during Phase 0/1.

Other top-tier pitfalls that must be designed-against (not retrofitted):
- **Retry double-billing** (Pitfall 4) — never retry after first byte delivered; encode as state machine `Connecting → FirstByte → Streaming → Done`; tool calls dedupe by `(message_id, tool_use_id)`
- **Three-state cost record** (Pitfall 5) — `Reported` / `Estimated` / `Unknown`; never log `tokens=0` when truth is "we don't know"; surface `gen_ai.usage.source` as span attr + low-cardinality metric label
- **OTel cardinality bomb** (Pitfall 7) — metric attrs are an *allowlist* (~6 attrs), not a denylist; high-cardinality attrs (`user.id`, `session.id`) go on spans only
- **`gen_ai.*` semconv is still Development** (Pitfall 10) — emit behind `OTEL_SEMCONV_STABILITY_OPT_IN`; centralize attr names in one constants file; bump `llm-agent-otel` major when upstream stabilizes
- **Cost-runaway demo** (Pitfall 17) — refsvc has hard caps DAY ONE: `MAX_TOKENS_PER_REQUEST`, `MAX_REQUESTS_PER_IP_PER_MINUTE`, `DAILY_TOKEN_BUDGET`, `RETRY_MAX_ATTEMPTS`, plus a `DISABLE_LLM=1` panic switch
- **Prompt injection in customer-support refsvc** (Pitfall 18) — least-privilege tools, server-side `user_id` enforcement (NEVER trust LLM-supplied IDs), retrieved-content marked as untrusted in system prompt
- **Multi-repo coordination failures** (Pitfalls 12-15) — `replace` directives banned in tagged releases (CI gate), `go.work` `.gitignore`'d (CI runs `GOWORK=off`), umbrella CI builds all 4 repos against `llm-agent` HEAD, every Deprecated symbol has a target removal version

---

## The 5–7 Keystone Decisions

These are the decisions that, if right, make the rest fall into place. If wrong, every downstream phase pays.

| # | Decision | Locked-in by | Cascades into |
|---|----------|-------------|---------------|
| K1 | **`StreamEvent` typed union with per-index keying** (NOT lowest-common-denominator chunks) | Phase 0/1 in `llm/v2` | All 3 adapters; tool-call dedupe; OTel TTFT span event; agent loop's tool-call accumulator |
| K2 | **`Capabilities(model)` takes a model parameter** (per-(provider × model), not per-provider) | Phase 0 in `llm/v2` | Ollama adapter (varies per model); agent fallback hierarchy; `ProviderInfo` struct shape |
| K3 | **OTel attached as decorator wrappers** (`Wrap(ChatModel) ChatModel`, `Wrap(Agent) Agent`), never as hooks/callbacks | Phase 0 (`llm/v2` interfaces must compose under wrapping) | `llm-agent-otel` repo's whole API; user opt-in story; ability to compose retry + obs + cache wrappers |
| K4 | **Three-state cost record** (`Reported`/`Estimated`/`Unknown`) + retry state machine (`Connecting → FirstByte → Streaming → Done`) | Phase 1 (every adapter) | Cost dashboards; per-request guardrail (P2); double-bill prevention; tool-call dedupe |
| K5 | **`gen_ai.*` semconv constants centralized + behind `OTEL_SEMCONV_STABILITY_OPT_IN`** with content-capture DEFAULT OFF | Phase 3 (`llm-agent-otel`) | Forward-compat with semconv promotions; PII compliance; refsvc compose defaults |
| K6 | **Multi-repo discipline encoded as CI gates** — `go.work` gitignored, `GOWORK=off` in CI, `replace` ban on tagged releases, umbrella 4-repo build on every `llm-agent` PR | Phase 0 (multi-repo infra) | Every release across 4 repos; cross-repo BC contract; `go get` safety for downstream |
| K7 | **Reference service has hard caps + panic switch from Day 1** (per-IP RPM, daily token budget, max-tokens, max-tool-calls, retry max, `DISABLE_LLM=1`) | Phase 4 (refsvc MVP) | Cost-runaway prevention; demo deployability; documents the production-hardening boundary |

If a phase plan does not name how it implements one of K1–K7 (or explicitly defers it with rationale), the phase is under-specified.

---

## Conflicts to Resolve (cross-cuts the roadmapper must arbitrate)

### Conflict A — Build order: sequential validation vs. walking skeleton

- **ARCHITECTURE.md (Phase 3 → 5 → 6):** OpenAI fully then Anthropic. Rationale: validate the abstraction with one provider; stress-test with second; if Anthropic forces a v2 change, only OpenAI needs re-validation.
- **PITFALLS.md (Pitfall 20):** Walking-skeleton-first across all 3 — Generate-only on all → +Stream on all → +Tools on all → +Embed on all. Rationale: prevents Phase 1 perfectionism; surfaces abstraction holes EARLY rather than after one provider is over-specified.

**Recommendation for the roadmapper:** Hybrid. Drive the breadth at every depth gate, but lead with OpenAI within each gate.
- Gate 1: **Generate (sync)** on all 3 providers — minimum viable adapter shape, validates `ChatModel` and `ProviderInfo(model)`
- Gate 2: **Streaming** on all 3 — validates `StreamEvent` union (the K1 keystone) against all 3 wire formats simultaneously
- Gate 3: **Native tool calls** on all 3 — validates `ToolCaller` + index/dedupe semantics
- Gate 4: **Embeddings** on OpenAI + Ollama; Anthropic returns `ErrNotSupported`

This keeps OpenAI the "first to depth" within each gate (so ambiguity is resolved against the most-documented wire format first) but never lets a single provider get >1 gate ahead of the others. The cross-provider conformance suite is built incrementally, gate by gate.

### Conflict B — `ProviderInfo` granularity (per-provider vs. per-model)

- **ARCHITECTURE.md:** `ProviderInfo` is a single struct returned by `Info() ProviderInfo` — one shape per provider instance. Includes `Model string` field but capability booleans are per-instance.
- **PITFALLS.md (Pitfall 6):** Capabilities are per-`(provider × model)`, not per-provider. Ollama is the canonical case (Llama 3 vs. Qwen3 vs. Mistral all differ on tool format). Bitmask AND type assertion both lie when capabilities vary by model.

**Recommendation:** PITFALLS wins. `ProviderInfo` as designed is incomplete. Either:
- (a) `ChatModel.Info() ProviderInfo` is per-instance, where each provider instance is constructed with a specific model (`openai.New(openai.WithModel("gpt-4o"))`) — then `Info()` correctly reflects the configured model. **This is the simplest fix.** Adapters validate at construction time.
- (b) Add `Capabilities(model string) ProviderInfo` as a separate method for callers that want to query without binding.

Pick (a) for the v0.3 surface. The agent layer holds a `ChatModel` value already configured for a model; `Info()` reflects that model's capabilities. The roadmap should call this out in the Phase 0/1 design doc.

### Conflict C — `replace` directives policy

- **STACK.md:** `replace` is a "documented escape hatch" for iterating on unreleased core changes; documented in sister-repo READMEs.
- **PITFALLS.md (Pitfall 12):** `replace` left in a tagged release breaks downstream `go get`. CI gate must reject any `replace` in a tagged-release branch.

**No real conflict** — STACK is talking about local-dev hatch, PITFALLS about release discipline. But the roadmap must include both: the README guidance AND the CI gate. The right phrasing is **"`go.work` is the recommended pattern for cross-repo iteration; `replace` only as a documented temporary escape hatch, never tagged."**

### Conflict D — K8s scope

- **STACK.md** describes a Helm/K8s shape with `otwld/ollama-helm` reference chart.
- **FEATURES.md** lists "Optional Kubernetes manifests / Helm chart variant" as P3 (deferred).
- **PROJECT.md** says "Optional Kubernetes manifests / Helm chart variant" under Active.
- **PITFALLS.md (Pitfall 16)** says: don't half-ship K8s — either do it with its own CI (kind/k3d) and `gpu-test` Job, or explicitly say "K8s is NOT part of v0.3."

**Recommendation:** Defer K8s. Document in refsvc README as "out of scope for v0.3 — see issue #N." If a phase wants to attempt it, it needs its own kind/k3d CI from the start. This contradicts PROJECT.md's Active list, which the roadmapper should flag as a candidate for movement to Out of Scope at the next `/gsd-transition`.

---

## Implications for Roadmap

Suggested phase structure. Phases are sized for solo-side-project pace; quality gates dominate (per PROJECT.md "quality > speed").

### Phase 0: Multi-repo infra + `llm/v2` keystone interfaces
**Rationale:** Everything downstream depends on K1, K2, K3, K6. Setting up CI policy now (Pitfalls 12–14) prevents costly retrofits.
**Delivers:**
- `llm/v2/` package in `llm-agent` with `ChatModel`, `ToolCaller`, `Embedder`, `StructuredOutputs`, `StreamReader`/`StreamEvent` (typed union), `ProviderInfo` (per-instance, model already-bound)
- `ScriptedLLM`/mock implementations of all interfaces; 100% type-level test coverage
- Multi-repo discipline: 4 empty sister repos initialized with `go.mod`; `.gitignore` includes `go.work`; CI runs `GOWORK=off go build ./...`; CI gate rejects `replace` directives on release branches
- Umbrella CI: `go.work`-based 4-repo build that runs on every `llm-agent` PR
- `RESEARCH_LOG.md` template per repo

**Addresses:** capability interfaces (FEATURES Cross-Cutting D); typed stream union (ARCHITECTURE pattern 1)
**Avoids:** Pitfalls 6 (capability shape), 12 (`replace`), 13 (`go.work`), 14 (cross-repo break), 22 (drift)
**Research flag:** **None** — design well-understood; covered by ARCHITECTURE + PITFALLS

### Phase 1: Three-provider walking skeleton — Generate (sync) only
**Rationale:** Lock the `ChatModel` + `ProviderInfo(model)` contract against all 3 wire formats before touching streaming. Per Conflict A resolution.
**Delivers:**
- `openai`, `anthropic`, `ollama` packages in `llm-agent-providers`; each implements `ChatModel.Generate` only
- Typed error taxonomy (`RateLimitError` etc.)
- `internal/contract/generate_test.go` — runs same fixtures against all 3 adapters
- Conformance fixture capture script per provider
- Provider Author Guide v0.1 (markdown) — what `ChatModel.Generate` must do

**Addresses:** P1 features 1.1–1.3 partial; FEATURES table-stakes A.1, A.6, A.7, A.8, A.9, A.10; cross-cutting D
**Avoids:** Pitfall 20 (perfectionism — by forcing breadth)
**Research flag:** **NONE** — provider SDK Generate APIs are well-documented; STACK.md covers it

### Phase 2: Streaming on all 3 providers + `StreamEvent` validation
**Rationale:** This gate validates K1. If `StreamEvent` shape needs revision, only Generate-only adapters need re-touch (cheap), not full-feature adapters (expensive).
**Delivers:**
- `Stream` method on all 3 adapters
- Goroutine-leak-safe streaming (`goleak` in CI for sister repos; cancel-mid-stream test per adapter)
- Three-state cost record (`Reported`/`Estimated`/`Unknown`) wired into `Usage`
- TTFT measurement infra (will be consumed by OTel later)
- Conformance suite extended: cancel-mid-stream, partial-usage-on-error, OpenAI `stream_options.include_usage=true` enforcement

**Addresses:** FEATURES table-stakes A.2; K4
**Avoids:** Pitfalls 3 (goroutine leak), 5 (partial usage), 4 (retry double-bill design baked in)
**Research flag:** **YES** — Anthropic SSE `content_block_delta` semantics are subtle; ARCHITECTURE has the design but real fixtures from a live Anthropic streaming session will catch surprises. Plan a `/gsd-research-phase` 0.5-day budget.

### Phase 3: Native tool calling on all 3 providers
**Rationale:** Tool calling is the agent-paradigm-unblocker (FunctionCall + ReAct become real). Tests K1's per-tool-call indexing under interleaved load.
**Delivers:**
- `ToolCaller.WithTools(tools) ToolCaller` (immutable per ARCHITECTURE pattern 2)
- Per-provider tool-call delta accumulation (OpenAI by index, Anthropic by content-block index, Ollama whole-call)
- Tool-call dedupe at agent layer keyed by `(message_id, tool_use_id)`
- Ollama per-model strategy table (Llama 3 vs. Qwen3 vs. Mistral parsing)
- Refactor `react.go` and `function_call.go` to consume `ChatModel` + type-assert `ToolCaller` + scratchpad fallback
- Conformance suite extended: parallel tool calls (OpenAI), multi-block (Anthropic), capability-degrade (Ollama llama2)

**Addresses:** FEATURES A.3, differentiator "Capability-negotiated agents"; K1
**Avoids:** Pitfalls 1 (OpenAI index keying), 2 (Anthropic content-block parsing), 4 (tool dedupe), 6 (capability-per-model), 19 (Ollama divergence)
**Research flag:** **YES** — Anthropic `BetaToolRunner` ergonomics + OpenAI Responses API tool semantics evolve; Ollama tool format per-model is unstable upstream. Budget 1 day reading current SDK docs + issue trackers per Pitfall 21.

### Phase 4: Embeddings on OpenAI + Ollama; Anthropic gap documented
**Rationale:** Closes the provider walking skeleton. Unlocks RAG with non-Hash backing in refsvc.
**Delivers:**
- `Embedder` capability on OpenAI + Ollama adapters; `ErrNotSupported` on Anthropic
- `rag.RAGSystem` validates against new embedder (regression check on v0.2 abstraction)
- Conformance suite extended: dimension assertion, batch-embed shape

**Addresses:** FEATURES A.4 (table stake)
**Avoids:** breaking RAG abstraction
**Research flag:** **NONE**

### Phase 5: OTel adapter (`llm-agent-otel`)
**Rationale:** Now that the 3 providers + `llm/v2` are stable (Phases 1–4), OTel can wrap a known-good `ChatModel`. K3 + K5 both land here.
**Delivers:**
- `otelmodel.Wrap(ChatModel)` + `otelagent.Wrap(Agent)` decorator wrappers
- `gen_ai.*` semconv attributes centralized in one constants file; `OTEL_SEMCONV_STABILITY_OPT_IN` honored
- Span tree per ARCHITECTURE: `invoke_agent → chat → execute_tool` for ReAct; `invoke_workflow → invoke_agent → ...` for multi-agent
- Metrics: `gen_ai.client.token.usage`, `gen_ai.client.operation.duration`, `gen_ai.client.operation.time_to_first_chunk` + custom `agent.iterations`, `agent.tool.invocations`
- Metric attribute allowlist (~6 attrs); cardinality CI test (1000 user IDs → ≤50 attribute combinations)
- Content-capture DEFAULT OFF; `OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT` respected; redactor available
- slog handler bridge to OTel logs
- OTLP HTTP (default, port 4318) + OTLP gRPC (opt-in) exporters
- Span-explosion test: 500-chunk stream produces exactly 1 span

**Addresses:** FEATURES B (all OTel features); K3, K5
**Avoids:** Pitfalls 7 (cardinality), 8 (PII), 9 (span explosion), 10 (semconv churn)
**Research flag:** **YES** — `gen_ai.*` semconv status is Development; check upstream `semantic-conventions` repo at phase open. Budget 1 day per Pitfall 21.

### Phase 6: Reference customer-support service (`llm-agent-customer-support`)
**Rationale:** The integration test for everything. Composes providers + otel + core. K7 lands here.
**Delivers:**
- `cmd/server/main.go`: env-var config, OTel init, factory, http.Server with graceful shutdown
- `POST /chat` (one-shot) + `POST /chat/stream` (SSE); `/healthz`, `/readyz`; `X-Trace-Id` response header
- Session storage (sqlite for dev; postgres optional)
- Customer-support multi-agent: RAG knowledge lookup + StateGraph triage + tools
- **Hard caps Day 1:** `MAX_TOKENS_PER_REQUEST=1000`, `MAX_TOOL_CALLS_PER_AGENT_LOOP=5`, `MAX_REQUESTS_PER_IP_PER_MINUTE=10`, `RETRY_MAX_ATTEMPTS=2`, `DAILY_TOKEN_BUDGET=100000`, `DISABLE_LLM=1` panic switch
- **Prompt-injection guardrails Day 1:** input filter, tool allowlist with server-side `user_id` enforcement, RAG content marked untrusted in system prompt
- `compose.yaml`: app + ollama + `grafana/otel-lgtm`; provider switch via `LLM_PROVIDER` env
- Pre-provisioned Grafana dashboard JSON (latency p50/p99, tokens/min, cost/min, error rate, tool-call success ratio)
- Tail-sampling collector config: 100% errors, 100% latency >5s, 10% otherwise; `decision_wait=30s`
- README: "demo only; production deployment requires X, Y, Z hardening" banner

**Addresses:** FEATURES C (all reference service features); K7
**Avoids:** Pitfalls 11 (sampling), 16 (K8s scope creep — explicitly out), 17 (cost runaway), 18 (prompt injection)
**Research flag:** **YES** — guardrail patterns for prompt injection are still evolving in 2026; budget 1 day surveying current OWASP LLM Top 10 + recent advisories per Pitfall 21.

### Phase 7: Deprecation removal & v0.4 cut
**Rationale:** Honor the dual-track BC promise — `llm.Client` removed one minor cycle after Deprecated marker.
**Delivers:**
- Internal callers (examples, refsvc) audited — zero internal users of deprecated API
- `llm/Client` removed in `llm-agent` v0.4.0
- CHANGELOG `### Breaking` section
- Sister repos bump `require llm-agent v0.4.0`; coordinated tag

**Addresses:** PROJECT.md "Dual-track BC"
**Avoids:** Pitfall 15 (deprecation never removed)
**Research flag:** **NONE**

### Phase Ordering Rationale

- **Walking-skeleton breadth (Phases 1→4) before depth (Phase 5 OTel, Phase 6 refsvc):** resolves Conflict A; lets Phase 5's OTel wrap a stable `ChatModel`; lets Phase 6 compose three known-good components.
- **Phase 0 must finish before any Phase 1 work:** the multi-repo CI policy is the gate. Phase 1 PRs that pre-date the umbrella CI will hide cross-repo breakage.
- **Phase 5 (OTel) is independent of Phase 4 (embeddings)** — could parallelize if there were two devs. Solo, sequential is safer; embeddings has the lighter design surface so it goes first.
- **Phase 6 is the longest phase** because it's the only one that exercises the full vertical. Budget accordingly.
- **Phase 7 is calendar-gated** (one minor cycle after deprecation marker), not effort-gated.

### Research Flags

Phases that warrant a `/gsd-research-phase` budget at phase opening:
- **Phase 2 (streaming):** Anthropic SSE `content_block_delta` semantics; OpenAI `stream_options` evolution. ~0.5 day.
- **Phase 3 (tool calls):** SDK ergonomics still moving (`BetaToolRunner`, Responses API tool semantics); Ollama per-model wire-format issue tracker. ~1 day.
- **Phase 5 (OTel):** `gen_ai.*` semconv promotion status; vendor backend support (Datadog, Grafana). ~1 day.
- **Phase 6 (refsvc):** Prompt injection mitigation patterns; OWASP LLM Top 10 current state. ~1 day.

Phases with standard patterns (skip research-phase):
- **Phase 0:** Pure design; ARCHITECTURE.md + PITFALLS.md cover it.
- **Phase 1, 4:** Provider SDK Generate/Embed APIs are stable and well-documented in STACK.md.
- **Phase 7:** Mechanical cleanup; no research needed.

---

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | **HIGH** | All SDK versions cross-checked against GitHub releases on 2026-05-10; OTLP/HTTP-vs-gRPC defaults verified against spec; `grafana/otel-lgtm` is a known reference image. Only MEDIUM caveat: `gen_ai.*` semconv status. |
| Features | **HIGH on table-stakes & differentiators**, MEDIUM on competitor framing | OTel GenAI semconv attributes verified at official spec; provider SDK feature parity verified; competitor (langchaingo/eino/genkit) feature claims are README-level (MEDIUM) |
| Architecture | **HIGH on core patterns**, MEDIUM on refsvc shape | Eino interfaces, Anthropic streaming spec, openai-go API verified against current docs/source. Reference-service shape is synthesized from go-llm/ADK/LangGraph patterns; no single canonical pattern, hence MEDIUM. |
| Pitfalls | **HIGH** | All 22 pitfalls grounded in named, dated sources (issue trackers, official docs, recent blog posts on documented incidents). Recovery strategies are heuristic-level. |

**Overall confidence:** **HIGH**. The roadmapper has enough to plan against without further research. The four research flags above are phase-opening reads, not gaps blocking the roadmap.

### Gaps to Address

These were not fully resolved during research and need attention during planning or implementation:

- **Anthropic Go SDK ergonomics for tool-loop helpers (`BetaToolRunner`, `NewToolRunnerStreaming`):** Context7 has the API surface but real usage patterns at scale are sparse. Mitigation: Phase 3 research budget (1 day) reading SDK source + recent issue tracker.
- **`ProviderInfo` per-(provider × model) shape:** ARCHITECTURE proposes per-instance `Info()`; PITFALLS demands per-model granularity. Concrete struct shape and validation behavior must be settled in Phase 0 design doc. Recommendation noted under Conflict B above.
- **`grafana/otel-lgtm` for production:** Single-container is for demo only. Refsvc compose ships demo shape; the production path (5-service split) is mentioned in STACK but not designed. **Roadmap action:** explicitly defer the production-split compose to v0.4; document in refsvc README.
- **Anthropic embeddings story:** Anthropic does not ship embeddings; STACK + FEATURES say `ErrNotSupported`. The customer-support refsvc, however, needs RAG. Decision: refsvc uses OpenAI or Ollama for embeddings even when chat provider is Anthropic — i.e., chat-provider and embedding-provider are independent env vars. **Roadmap action:** make this explicit in Phase 6 design (`LLM_PROVIDER=anthropic` implies `EMBEDDING_PROVIDER=openai|ollama`).
- **K8s scope:** PROJECT.md Active list says "Optional K8s manifests"; PITFALLS Pitfall 16 says don't half-ship. **Roadmap action:** flag PROJECT.md for amendment at next `/gsd-transition`; explicitly move K8s to v0.4 / Out of Scope.
- **Walking-skeleton vs. validation-first build order:** Conflict A above. The hybrid recommendation here needs to be ratified in Phase 0's design doc, not assumed.
- **Replay testing (`RecordingLLM`) priority:** Listed P2 in FEATURES; arguably P1 for the conformance suite to be self-maintaining. Roadmapper to decide whether it lands in Phase 1 (with conformance suite) or P2 follow-up.

---

## Sources

Research was synthesized from 4 detailed files in this directory; each carries its own complete sources list.

### Primary (HIGH confidence) — covered by all 4 files
- [OpenTelemetry GenAI semantic conventions](https://opentelemetry.io/docs/specs/semconv/gen-ai/) — spec status (Development as of 2026-05-10)
- [openai/openai-go GitHub](https://github.com/openai/openai-go) — v3.35.0 release; streaming, tools, embeddings
- [anthropics/anthropic-sdk-go GitHub](https://github.com/anthropics/anthropic-sdk-go) — v1.41.0 release; `Message.Accumulate`, `BetaToolRunner`
- [ollama/ollama Go API](https://pkg.go.dev/github.com/ollama/ollama/api) — v0.23.2; callback streaming
- Context7 IDs: `/openai/openai-go`, `/anthropics/anthropic-sdk-go`, `/open-telemetry/opentelemetry-go`, `/cloudwego/eino`, `/tmc/langchaingo`

### Secondary (MEDIUM confidence)
- Competitor framework comparisons (langchaingo / eino / genkit Go) — README-level
- Reference-service shape synthesis from go-llm / ADK Go / LangGraph patterns
- Cost-tracking and retry-policy patterns (Tail-Tolerant Retry, FlexPrice metering)
- Prompt-injection mitigation patterns (OWASP LLM Top 10, 2026 advisories)

### Local context
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/PROJECT.md` — milestone scope, Active/Validated/Out-of-scope, Key Decisions
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/CHANGELOG.md` — BC policy, 0.x line discipline
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/llm/client.go` — current `Client`, `Tool`, `ToolCall`, `StreamChunk` shapes (the seam to evolve in `llm/v2`)
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.github/workflows/test.yml` — current CI shape; informs umbrella CI design

### Detailed research files (in this directory)
- `STACK.md` — versions, multi-repo layout, deployment, testing
- `FEATURES.md` — table-stakes / differentiators / anti-features; competitor matrix; P1/P2/P3 prioritization
- `ARCHITECTURE.md` — capability negotiation, streaming union, decorator pattern, build order, repo layout
- `PITFALLS.md` — 22 pitfalls with prevention/verification/recovery; performance/security/integration tables; "looks done but isn't" checklist; pitfall-to-phase mapping

---
*Research completed: 2026-05-10*
*Ready for roadmap: yes*

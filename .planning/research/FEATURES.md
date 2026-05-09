# Feature Research

**Domain:** Production-grade Go LLM agent framework — provider adapters + OTel observability + reference customer-support service
**Researched:** 2026-05-10
**Confidence:** HIGH (provider SDKs / OTel GenAI conventions are HIGH-confidence — verified at official docs and SDK repos; competitor feature parity claims are MEDIUM — based on framework READMEs and docs; user-expectation framing is MEDIUM — extrapolated from `langchaingo` / `eino` / `genkit` parity bars)

## Feature Landscape

### Table Stakes (Users Expect These)

Features users assume exist in 2026. Missing any of these and a langchaingo / eino / genkit user will say "this is a toy."

#### A. Provider Adapter — Wire-Format Features

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Generate (sync chat completion)** | Bare minimum LLM call; every framework has it | LOW | Already in `llm.Client.Generate`; provider impl is wire-format work only |
| **Streaming (SSE/chunked)** | Real chat UX needs token-by-token display; supported by all 3 SDKs and every Go framework (langchaingo, eino, genkit) | MEDIUM | Already in `llm.GenerateStream`; provider impl must reassemble cross-chunk tool calls — see PITFALLS.md |
| **Native tool calling (parallel where supported)** | All 3 SDKs expose it; ReAct/FunctionCall agents become useful only when tools are native (not regex-parsed). Anthropic + OpenAI default to parallel | HIGH | The single biggest abstraction risk: OpenAI emits incremental `tool_calls` deltas, Anthropic emits whole `tool_use` blocks, Ollama mimics OpenAI shape. v2 abstraction must be lossless across all three |
| **Embeddings** | RAG is broken without real embeddings (HashEmbedder is a learning toy). `langchaingo/embeddings` ships 6+ providers; `eino` ships an `Embedding` component | LOW | Per-provider: OpenAI `text-embedding-3-*`, Ollama `/api/embed`, Anthropic via Voyage AI (3rd party — don't ship in v0.3, document the gap) |
| **Token usage exposure (input + output)** | Required for OTel `gen_ai.usage.input_tokens` / `output_tokens` and for cost tracking. All 3 SDKs return it; `StreamUsage` already in `llm` types | LOW | Stream finalization must surface usage on the terminal chunk |
| **System-prompt vs message conventions** | OpenAI uses `system` role; Anthropic uses top-level `system` parameter (NOT a message); Ollama follows OpenAI shape. Frameworks must hide this | LOW | Already abstracted in `llm.Message` — verify Anthropic adapter lifts system messages out of the array |
| **Retry / backoff on 429 / 5xx** | Both official Go SDKs (`openai-go`, `anthropic-sdk-go`) ship default exponential backoff with 2 retries; users expect the framework not to throw on transient 429 | LOW | Delegate to SDK-default behavior in v0.3; expose retry-policy hook for v0.4 |
| **Rate-limit aware error type** | Users need to distinguish "you exceeded TPM, back off" from "you sent malformed input" to make retry decisions in agent loops | LOW | Wrap SDK error into a typed `llm.RateLimitError` / `llm.AuthError` / `llm.InvalidRequestError` taxonomy |
| **Finish-reason normalization** | OpenAI: `stop` / `length` / `tool_calls`. Anthropic: `end_turn` / `max_tokens` / `tool_use`. Already abstracted by `llm.FinishReason` — must be set correctly per provider | LOW | Add `FinishReasonToolCalls` if not present |
| **Context-cancellation honored** | Idiomatic Go; all 3 SDKs accept `context.Context`. Streaming chans must close on `ctx.Done()` | LOW | Already in `llm.Client` signature; verify provider goroutines drain |

#### B. OpenTelemetry Observability — Spans, Metrics, Attributes

OTel GenAI semantic conventions ([opentelemetry.io/docs/specs/semconv/gen-ai/](https://opentelemetry.io/docs/specs/semconv/gen-ai/)) are the de-facto standard in 2026 (Datadog v1.37 native, Grafana Loki ingestion). v0.3 must emit conformant attributes or vendors won't auto-parse the traces.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Span per LLM call** with operation name `chat` / `text_completion` / `embeddings` | Spec-mandated; required for any vendor backend to identify the span as a GenAI op | LOW | One span around `llm.Client.Generate` |
| **Required attrs: `gen_ai.operation.name`, `gen_ai.provider.name`** | These are the discriminators; without them the trace is unattributed | LOW | Set in adapter wrapper; provider name is `openai` / `anthropic` / `ollama` |
| **Request attrs: `gen_ai.request.model`, `gen_ai.request.temperature`, `gen_ai.request.max_tokens`, `gen_ai.request.stream`** | Vendor backends auto-build dashboards keyed on these | LOW | Read from `GenerateRequest` |
| **Response attrs: `gen_ai.response.model`, `gen_ai.response.id`, `gen_ai.response.finish_reasons`** | Used by backends to correlate failures and detect length-truncation | LOW | Surface from provider response |
| **Token attrs: `gen_ai.usage.input_tokens`, `gen_ai.usage.output_tokens`** | Cost dashboards key off these. Plus cache attrs (`gen_ai.usage.cache_read.input_tokens`, `gen_ai.usage.cache_creation.input_tokens`) when prompt caching is on | LOW | Already on `StreamUsage`; emit on span end |
| **Tool span: `execute_tool` operation, `gen_ai.tool.name`, `gen_ai.tool.call.id`, `gen_ai.tool.type`** | Required for tool-call success ratios in dashboards | LOW | One span per tool invocation in `agents/builtin` and registry |
| **Agent span: `invoke_agent {gen_ai.agent.name}`, `gen_ai.agent.id`, `gen_ai.conversation.id`** | Lets vendors group spans into a single conversation timeline (Datadog AI Observability does this) | MEDIUM | Wrap each `agent.Run`; thread `conversation.id` through `context.Context` |
| **Metrics: `gen_ai.client.token.usage` (histogram), `gen_ai.client.operation.duration` (histogram)** | Spec-mandated; backends compute cost-per-route and p99 latency from these | LOW | Emit alongside spans |
| **Streaming metrics: `gen_ai.client.operation.time_to_first_chunk`, `time_per_output_chunk`** | TTFT is the user-visible latency in chat UIs; everyone instruments it | LOW | Hook stream channel reads |
| **Error metric: `error.type` attribute set on failed spans** | Spec-mandated; lets backends aggregate error rates by category | LOW | Map from typed error taxonomy (above) |
| **Logs via slog bridge** | Go 1.21+ apps already use `log/slog`; adapter must be a `slog.Handler` decorator, not a parallel logger | LOW | `go.opentelemetry.io/contrib/bridges/otelslog` |
| **OTLP/HTTP + OTLP/gRPC exporters** | These two are the only ones that matter; `grafana/otel-lgtm` and Datadog accept both | LOW | Ship config-presets, no custom exporter |

#### C. Reference Customer-Support Service — Production Service Surface

What `genkit start` and `eino` examples do; what users replicate when they read your reference service.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **HTTP API (chat endpoint)** | "Service" not "CLI demo." Even genkit/eino samples expose `POST /chat` | LOW | `net/http` only; align with PROJECT.md stdlib bias for refsvc surface (refsvc itself can take deps) |
| **Health checks (`/healthz`, `/readyz`)** | k8s, docker compose `healthcheck:`, load balancers all expect these | LOW | Distinguish liveness (process alive) from readiness (provider reachable, OTel exporter connected) |
| **Graceful shutdown on SIGTERM** | Drains in-flight requests; `http.Server.Shutdown` | LOW | Required by `docker compose down` and k8s rolling updates |
| **Config via env vars** | 12-factor; PROJECT.md already calls out "Provider switch via env" | LOW | `LLM_PROVIDER=openai|anthropic|ollama`, `OPENAI_API_KEY`, `OTEL_EXPORTER_OTLP_ENDPOINT` |
| **`docker compose up` brings up everything** | PROJECT.md explicit requirement; service + Ollama + OTel collector + Grafana | MEDIUM | Use `grafana/otel-lgtm` (single container = LGTM stack) to avoid 5 separate services |
| **Pre-provisioned Grafana dashboards** | "Out-of-box telemetry" is the diff between "demo" and "reference." Mount JSON via Grafana provisioning | MEDIUM | One dashboard with: latency p50/p99, tokens/min, cost/min, error rate, tool-call success ratio |
| **Per-request trace ID exposed to client** | Modern APIs return `X-Trace-Id` so users can paste into Tempo/Jaeger and find the trace | LOW | Read from current span context, set response header |
| **Session state (conversation memory)** | Customer support requires multi-turn; expected from any "chat service" | MEDIUM | Reuse existing `memory.Manager`; key on `conversation.id` from request body or cookie |
| **Provider-switch via env (one variable)** | PROJECT.md explicit; demonstrates the abstraction is real | LOW | Factory: `provider, _ := factory.New(os.Getenv("LLM_PROVIDER"))` |

#### D. Cross-Cutting (Required by All Three)

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Capability interfaces (ToolCaller, Embedder)** | PROJECT.md explicitly lists this; agents that embed must be able to ask "do you support embeddings?" before calling. `any-llm-go` (Mozilla) and `eino` both do this | MEDIUM | Composition: providers implement `llm.Client` (always) + optionally `llm.ToolCaller` + `llm.Embedder`. Agents type-assert |
| **Provider Author Guide** | PROJECT.md explicit; without it, third parties can't write conforming providers and your conformance tests are tribal knowledge | MEDIUM | A `docs/PROVIDER_AUTHOR_GUIDE.md` with the wire-format expectations + the httptest fixtures used by core tests |
| **Wire-format conformance test suite (httptest)** | PROJECT.md explicit; cost-free CI requires it. langchaingo and genkit both have similar suites | HIGH | One `providertest.Conformance(t, factory)` shared across all three adapters, replays canned OpenAI/Anthropic/Ollama HTTP fixtures |
| **Slog-friendly error wrapping** | Go ecosystem expectation; errors must serialize cleanly into structured logs with stable keys | LOW | `errors.Is` / `errors.As` for typed-error switches |

### Differentiators (Competitive Advantage)

Features where `llm-agent` v0.3 can pull ahead of `langchaingo` (loose, mostly OpenAI) and match or exceed `eino` (heavier, dep-heavy). All differentiators here are consistent with PROJECT.md's "stdlib-only core, opt-in sister repos" thesis.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Stdlib-only core, opt-in providers** | langchaingo pulls 30+ transitive deps the moment you `go get`. eino pulls Sonic / hertz / kitex. `llm-agent`'s core stays readable in one sitting; deps live behind import boundary the user explicitly crosses | LOW | This is the *existing* core value — NOT new work. v0.3 must protect it: providers/otel/refsvc in sister repos |
| **Conformance-tested provider abstraction** | Most Go LLM frameworks have 1 wire-format that works (OpenAI), 1 that lags. Three-provider parity in v0.3 is a marketable claim — "switch with one env var, see Grafana panel still works" | HIGH | The conformance suite IS the differentiator |
| **Capability-negotiated agents** | langchaingo agents assume tool calling exists and crash on Ollama llama3-base. `llm-agent` agents that type-assert and degrade gracefully (e.g., FunctionCall → ReAct text-parser fallback) is a real win | MEDIUM | Build on the v2 `ToolCaller` interface |
| **OTel GenAI semconv-conformant out of box** | Most frameworks emit *some* OTel; very few emit semantic-conventions-conformant attribute names. Datadog / Grafana / New Relic dashboards Just Work with conformant emitters | MEDIUM | One-shot work; main risk is keeping up as the spec moves from experimental → stable |
| **Reference service deployable in one command** | PROJECT.md explicit. langchaingo has no reference service; eino's references are ByteDance-internal-shaped; genkit Go is sample-shaped not deploy-shaped. A `docker compose up` reference deployment with Grafana panels is rare | MEDIUM | Bundles `grafana/otel-lgtm` |
| **Prompt-injection guardrail demo** | Customer support is *the* injection-prone domain. Showing a working input-sanitization step in the reference service teaches users a pattern they can copy. Non-trivial in 2026 | MEDIUM | Implement as a `Tool`-like middleware; pre-LLM check + post-LLM output filter. Document as a recipe; don't ship an opinionated guardrail engine |
| **Per-request cost limit guardrail** | "Budget caps must be multi-layered: per-session, per-user, per-tool." Most frameworks handwave this. A working `MaxCostUSD` knob on `agent.Run` is real production utility | MEDIUM | Plug into the metrics: count tokens, multiply by per-model rate table, abort when exceeded |
| **Provider-rate-table built-in** | If you already track tokens for OTel, computing cost is trivial. langchaingo doesn't; eino requires you to plug your own. A maintained `cost.Table` (versioned per release) is differentiating | LOW | Tiny YAML/Go-map keyed on `(provider, model)` |
| **Prompt-caching support (Anthropic explicit, OpenAI implicit)** | Cuts cost 90% / latency 85% on prefix-stable prompts (system prompt + RAG context). Anthropic requires `cache_control` markers — must be a first-class concept on `Message` not buried in raw-options. OpenAI is automatic | MEDIUM | Add `Message.CacheControl *CacheControl` (optional); adapter passes through. Surface cache hit/miss in OTel `gen_ai.usage.cache_read.input_tokens` |
| **Structured-output / JSON-schema mode** | All 3 providers ship it (OpenAI strict mode, Anthropic input_schema, Ollama via OpenAI compat layer). Critical for tool calling reliability and for non-tool structured extraction | MEDIUM | Add `GenerateRequest.ResponseFormat` enum (`text` / `json` / `json_schema`) + schema field. Pass through per-provider |
| **Replay testing via request fingerprinting** | Make agent runs deterministic in test by hashing `(messages, tools, schema)` and replaying canned response. Critical for CI cost-zero policy. langchaingo has nothing; eino has callbacks | MEDIUM | Build on the existing `ScriptedLLM` pattern + add a `RecordingLLM` that captures real provider responses to fixture files |

### Anti-Features (Commonly Requested, Often Problematic)

These map directly to PROJECT.md "Out of Scope" or are scope-creep traps. Crossed-checked against `Out of Scope` list at PROJECT.md:63-72.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| **Vision / multimodal in v0.3** | Users see GPT-4o / Claude Sonnet vision and want parity | Wire-format for multipart messages is provider-specific (OpenAI nested arrays vs Anthropic source.type=base64); doubles the abstraction surface; tracking per PROJECT.md OoS | Defer to v0.4; document in CHANGELOG that text-only is intentional |
| **Real RL training in-process** | "RL is hot in 2026"; users assume `rl/` package will train | Different toolchain (PyTorch / TRL); concentrating budget on three provider integrations is the v0.3 priority per PROJECT.md OoS | Keep `rl/` as evaluation-only + Python TRL bridge stub |
| **Cross-framework interop (LangChain / LlamaIndex bridges)** | Users want to bring their LangChain prompts | Bidirectional schema translation is a maintenance treadmill; bloats stdlib-only core | Document "compose at the agent level" — users can wrap a LangChain call as a `Tool` if needed (PROJECT.md OoS) |
| **Distributed multi-agent (production a2a/anp)** | "Service mesh for agents" sounds enterprise | Service discovery + retry + circuit-breaking + security review is its own milestone; v0.3 keeps these toy-level | Defer to a later milestone (PROJECT.md OoS) |
| **Single-repo monolith with build tags** | "Tags would be cleaner than 4 repos" | Hidden imports = users still pay the import-cost the moment they `go build`. The differentiating value of zero-dep core depends on physical repo separation | Multi-repo + `go.work` for local dev (PROJECT.md OoS) |
| **In-process LLM inference (llama.cpp Go bindings)** | "Why not embed the model?" | CGO + 5GB model weights = breaks stdlib-only core, breaks `go get`. Ollama-as-sidecar covers this without breaking the import contract | Use Ollama provider for local-model use cases |
| **Auto-retry agent loops with model fallback** | "If GPT-4 fails, fall back to Claude" | Cross-provider state translation (tool-call IDs, message history shapes) on retry is brittle; users want it but bake-time is high | Document a recipe using the `Client` interface + a wrapper agent; don't ship a `MultiProviderClient` |
| **GUI / Studio / playground UI** | genkit ships a local Studio UI. Users will ask | Big TS/React surface area; not stdlib-only-friendly; orthogonal to "framework" mission | Reference service exposes a minimal `/chat` HTML page if any UI is needed; full Studio is out of scope for v0.3 |
| **Vector store backends (Pinecone, Weaviate, Pgvector)** | RAG users want managed vector DBs | InMemoryStore is sufficient for the reference service; per-vendor SDKs explode dependency surface | Document the `rag.Store` interface + ship one reference impl (sister-repo if needed); 3rd parties write the rest |
| **Multi-tenancy / API key rotation / per-user quotas** | Sounds production-grade | This is gateway concern (Bifrost, Portkey, AgentGateway exist); not framework concern. Bloating the core to support it is mission creep | Document "deploy behind an LLM gateway" as the recommended pattern |
| **Provider auto-detection from API keys / model names** | Users want `client := llm.New("gpt-4o")` magic | Magic kills explicitness; once you support 3 providers explicitly, the factory is 30 lines and clearer than auto-detect heuristics | Explicit `factory.NewOpenAI(cfg)` etc.; document the env-var pattern in refsvc |

## Feature Dependencies

```
[Embeddings (Provider B.4)]
    └──unlocks──> [Real RAG with non-Hash backing]
                       └──enables──> [Reference service knowledge lookup (C)]

[ToolCaller capability interface (Cross-Cutting D)]
    └──required-by──> [Native tool calling (Provider A.3)]
    └──required-by──> [Capability-negotiated agents (Differentiator)]
    └──required-by──> [Tool span emission (Observability B)]

[Token usage exposure (Provider A.5)]
    └──required-by──> [gen_ai.usage.* attrs (Observability B)]
    └──required-by──> [Cost-per-trace metric (Differentiator)]
    └──required-by──> [Per-request cost limit guardrail (Differentiator)]

[Wire-format conformance test suite (Cross-Cutting D)]
    └──required-by──> [Provider Author Guide (Cross-Cutting D)]
    └──required-by──> [Three-provider parity claim (Differentiator)]

[Streaming (Provider A.2)]
    └──required-by──> [TTFT metric: gen_ai.client.operation.time_to_first_chunk (Observability B)]
    └──required-by──> [Real chat UX in reference service (Reference C)]

[Conversation.id threading (Cross-Cutting)]
    └──required-by──> [Agent span attrs (Observability B)]
    └──required-by──> [Session state (Reference C)]

[Prompt caching (Differentiator)]
    └──enhances──> [Cost-per-trace metric]
    └──enhances──> [TTFT metric]
    └──conflicts──> [Naive request fingerprinting] — fingerprinter must hash *post-cache-marker* canonical form

[OTel adapter (sister: llm-agent-otel)]
    └──required-by──> [Reference service Grafana dashboards (Reference C)]
    └──required-by──> [Cost-per-trace, error rate, tool-call success metrics (Differentiator)]

[Capability interfaces (Cross-Cutting D)]
    └──conflicts──> [Auto-retry with model fallback (Anti-Feature)]
       (because cross-provider tool-call state translation is the hard part fallback can't solve)
```

### Dependency Notes

- **Embeddings unlocks RAG package switch:** v0.2 ships HashEmbedder (toy). The moment a real provider exposes `Embedder`, the reference service can plug it into the existing `rag.RAGSystem` without changing `rag` internals — a clean validation that the v0.2 abstraction was correct. If something breaks, that's an early signal for the abstraction-evolution work.
- **ToolCaller is the keystone:** Three downstream features (tool-call wire format, capability-negotiated agents, OTel tool spans) all hinge on the same v2 capability interface. This is the single most important abstraction decision in v0.3.
- **Token usage threads through three layers:** Provider must surface it → OTel must emit it → cost-limit guardrail must consume it. A break anywhere in the chain disables both observability and cost control. Test top-to-bottom.
- **Conformance suite is upstream of everything else provider-shaped:** Provider Author Guide *is* the conformance suite plus prose; nightly Ollama-live tests *are* the conformance suite against a real binary. Build it once, reuse it.
- **Prompt-caching design has a request-fingerprinting trap:** If replay testing hashes the raw `Messages` array including `cache_control` markers, two semantically-identical requests will mis-match. Hash a canonical form (cache markers stripped, then re-applied per-provider before send).

## MVP Definition

The "MVP" framing here is the v0.3 milestone scope from PROJECT.md, decomposed into launch / iterate / defer.

### Launch With (v0.3)

The minimum that delivers PROJECT.md's stated goal: "library you can deploy."

- [ ] **OpenAI provider** — Generate + Stream + native tool calling + Embeddings — required for "production-grade" claim
- [ ] **Anthropic provider** — Generate + Stream + native tool calling (Embeddings: document gap, point to Voyage/3rd party) — validates abstraction against a non-OpenAI wire format
- [ ] **Ollama provider** — Generate + Stream + Tool calling (where the model supports it; capability-degrade otherwise) + Embeddings — local-model story; required for nightly CI
- [ ] **`llm/v2` capability interfaces** — `Client` (always) + `ToolCaller` (optional) + `Embedder` (optional). Dual-track BC per PROJECT.md
- [ ] **Cross-provider streaming + tool-call interleaving conformance suite** — httptest fixtures replayed against all three adapters
- [ ] **Provider Author Guide** — markdown doc + the conformance suite as the contract
- [ ] **OTel adapter (sister repo)** — spans for LLM call / tool call / agent step + spec-conformant attributes + token/duration/TTFT metrics + slog bridge + OTLP exporters
- [ ] **Span semantics for each agent paradigm** — ReAct loop spans, PlanSolve plan/solve phase spans, StateGraph node-transition spans
- [ ] **Reference customer-support service (sister repo)** — `POST /chat`, `/healthz`, `/readyz`, graceful shutdown, env-var config, session state via `memory.Manager`, `X-Trace-Id` response header
- [ ] **`docker compose up` bring-up** — service + Ollama + `grafana/otel-lgtm` (LGTM bundle) + reference Grafana dashboard JSON
- [ ] **Provider-switch via one env var** — `LLM_PROVIDER=openai|anthropic|ollama`
- [ ] **Token usage exposed end-to-end** — provider response → OTel attrs → `gen_ai.usage.*` metrics
- [ ] **Typed error taxonomy** — `RateLimitError` / `AuthError` / `InvalidRequestError` / `TransientError`
- [ ] **`go.work` for local cross-repo dev** + nightly Ollama-live CI + httptest wire-format CI on PRs

### Add After Validation (v0.3.x patches / v0.4)

Once the v0.3 surface is shipped and a real user has tried it.

- [ ] **Structured-output / JSON-schema mode** — `GenerateRequest.ResponseFormat` enum + per-provider passthrough. Trigger: first user reports they hand-roll JSON parsing
- [ ] **Prompt-caching first-class support** — `Message.CacheControl` field for Anthropic, OTel cache attrs for both. Trigger: cost dashboards show repeated prefix tokens
- [ ] **Per-request cost-limit guardrail** — `agent.Run(ctx, ..., agents.WithMaxCostUSD(0.10))`. Trigger: first user reports a runaway loop
- [ ] **Provider rate-table** — versioned `cost.Table` for `(provider, model)`. Trigger: cost-limit feature requires it
- [ ] **Prompt-injection guardrail recipe** — pre-LLM input filter + post-LLM output filter, documented as middleware in the reference service. Trigger: refsvc users ask "how do I sanitize input?"
- [ ] **Replay testing utility** — `RecordingLLM` that wraps a real provider and captures fixtures. Trigger: 3rd-party provider authors need to reproduce conformance failures

### Future Consideration (v0.4+ or later milestones)

Defer until v0.3 lessons-learned are in.

- [ ] **Vision / multimodal** — wire-format work is its own milestone (PROJECT.md OoS)
- [ ] **Streaming-tool-call mid-stream cancellation** — provider-specific edge case; document the limitation in v0.3
- [ ] **Distributed a2a/anp production-shape** — separate milestone (PROJECT.md OoS)
- [ ] **Vector-store providers** — reference service uses InMemoryStore in v0.3; sister-repo `llm-agent-vectorstores` is a future milestone
- [ ] **Kubernetes manifests / Helm chart** — `docker compose` is the v0.3 commitment; k8s is "optional" in PROJECT.md and best deferred
- [ ] **GUI / playground UI** — out of scope for a Go-only framework; the `/chat` HTML page in refsvc is sufficient
- [ ] **Real RL training** — TRL bridge stub remains; production RL is a separate milestone (PROJECT.md OoS)
- [ ] **Cross-framework bridges (LangChain / LlamaIndex)** — out of scope (PROJECT.md OoS)

## Feature Prioritization Matrix

User Value × Implementation Cost. P1 = must ship in v0.3; P2 = nice in v0.3 if time permits, otherwise v0.3.x; P3 = explicitly deferred.

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| OpenAI provider (Gen + Stream + Tools + Embed) | HIGH | MEDIUM | P1 |
| Anthropic provider (Gen + Stream + Tools) | HIGH | MEDIUM | P1 |
| Ollama provider (Gen + Stream + Tools + Embed) | HIGH | MEDIUM | P1 |
| `llm/v2` capability interfaces | HIGH | MEDIUM | P1 |
| Conformance test suite (httptest) | HIGH | HIGH | P1 |
| OTel spec-conformant span/metric/log emission | HIGH | MEDIUM | P1 |
| Reference service `docker compose up` | HIGH | MEDIUM | P1 |
| Provider-switch via env | HIGH | LOW | P1 |
| Pre-provisioned Grafana dashboard | HIGH | MEDIUM | P1 |
| Token usage exposure end-to-end | HIGH | LOW | P1 |
| Typed error taxonomy | MEDIUM | LOW | P1 |
| Provider Author Guide | MEDIUM | MEDIUM | P1 |
| Conversation.id threading | HIGH | MEDIUM | P1 |
| Health/readiness endpoints + graceful shutdown | HIGH | LOW | P1 |
| Structured-output / JSON-schema mode | HIGH | MEDIUM | P2 |
| Prompt-caching first-class support | HIGH | MEDIUM | P2 |
| Per-request cost-limit guardrail | MEDIUM | MEDIUM | P2 |
| Prompt-injection guardrail recipe | MEDIUM | MEDIUM | P2 |
| Provider rate-table | MEDIUM | LOW | P2 |
| Replay testing (`RecordingLLM`) | MEDIUM | MEDIUM | P2 |
| Capability-negotiated agent fallbacks | MEDIUM | MEDIUM | P2 |
| Vision / multimodal | HIGH | HIGH | P3 (deferred per OoS) |
| Vector-store providers (Pinecone/Weaviate/Pgvector) | MEDIUM | HIGH | P3 |
| Kubernetes manifests / Helm | LOW | MEDIUM | P3 |
| Distributed a2a/anp | LOW | HIGH | P3 (deferred per OoS) |
| GUI / Studio | LOW | HIGH | P3 (out of scope) |
| Real RL training | LOW | HIGH | P3 (deferred per OoS) |
| Cross-framework bridges | LOW | HIGH | P3 (out of scope) |

**Priority key:**
- **P1**: Must have for v0.3 launch — listed in PROJECT.md `### Active`
- **P2**: Should ship in v0.3 if cycles allow; otherwise v0.3.x patches
- **P3**: Explicitly deferred or out-of-scope per PROJECT.md

## Competitor Feature Analysis

Comparison against the four most relevant competitors. `langchaingo` is the closest direct competitor (Go, multi-provider). `eino` is the heaviest production framework in Go. `genkit` is Google's polyglot framework with a Go SDK and is what users will compare against for "deployable demo." LangChain Python and LlamaIndex are the reference bars users carry over from Python.

| Feature | langchaingo (Go) | eino / CloudWeGo (Go) | genkit Go (Google) | llm-agent v0.3 (target) |
|---------|------------------|------------------------|---------------------|--------------------------|
| **Stdlib-only core** | No (deep dep tree) | No (Sonic + Hertz + Kitex deps) | No (Firebase deps) | **Yes** (this is THE differentiator) |
| **OpenAI provider** | Yes (mature) | Yes (eino-ext) | Yes | Yes (sister repo) |
| **Anthropic provider** | Yes (mature) | Yes | Yes | Yes (sister repo) |
| **Ollama provider** | Yes | Yes | Yes | Yes (sister repo) |
| **Native tool calling** | Yes — but uneven across providers | Yes — built into ChatModelAgent | Yes | Yes — capability-interface gated |
| **Streaming** | Yes — `GenerateContentStream` | Yes — automatic across orchestration | Yes — `Stream()` method on flows | Yes — already on `llm.Client` |
| **Embeddings** | Yes — 6+ providers | Yes — Embedding component | Yes | Yes (OpenAI + Ollama; Anthropic via 3rd party) |
| **Structured output / JSON mode** | Partial | Yes | Yes | P2 — defer to v0.3.x |
| **Prompt caching** | Manual (raw options pass-through) | Manual | Partial | P2 — first-class via `Message.CacheControl` |
| **Vision / multimodal** | Partial | Yes | Yes | **No (PROJECT.md OoS for v0.3)** |
| **Retry / backoff** | Per-provider | Per-provider | Yes | Yes — delegate to SDK defaults |
| **Capability negotiation** | No (assumes features exist) | Partial — components declare types | Partial | **Yes — first-class via interface composition (differentiator)** |
| **OTel emission** | Partial / community | Yes — callbacks for tracing | Yes — built-in observability dashboard | Yes — sister repo `llm-agent-otel` |
| **OTel GenAI semconv conformance** | No (custom names) | Partial | Partial (Genkit-specific names) | **Yes — strict conformance (differentiator)** |
| **Cost tracking** | No (manual) | Manual | Yes (in dashboard) | Yes — provider rate-table + OTel-derived (P2) |
| **Multi-agent orchestration** | Limited | Yes — Graph + ADK | Yes — Flows | Yes — already in v0.2 (Pipeline / FanOutFanIn / RoundRobin / RolePlay / StateGraph) |
| **Reference deployable service** | No | ByteDance-internal samples | Sample shapes | **Yes — `docker compose up` (differentiator)** |
| **Pre-provisioned Grafana dashboards** | No | Partial | Partial (Firebase console) | **Yes — JSON shipped (differentiator)** |
| **Prompt-injection guardrails** | No (community recipes only) | No | No | P2 — recipe in refsvc |
| **Cost-limit guardrail per request** | No | No | Limited | P2 — `WithMaxCostUSD()` |
| **Replay / fingerprint testing** | No | Callbacks-based | Limited | P2 — `RecordingLLM` |
| **Conformance test suite for 3rd-party providers** | No | Component test interfaces | No | **Yes — `providertest.Conformance(t, factory)` (differentiator)** |

**Reading of the matrix:**
- `langchaingo` is the breadth leader (most providers, most adapters) but the **least production-shaped** — no reference service, partial OTel, no conformance suite. v0.3 wins on production posture and Go ergonomics, not breadth.
- `eino` is the production-shape leader in Go (callbacks, ADK, graph) but **heavy** — Sonic/Hertz/Kitex pulled in. v0.3 wins on minimalism and audit-ability, not feature count.
- `genkit Go` has the best **deployable-demo** story (Studio UI, dashboards, flows) but is **Google-Cloud-shaped** — Firebase opinions leak in. v0.3 wins on cloud-neutrality and stdlib-core.
- The unique cross-product position v0.3 occupies: **stdlib-only core + three-provider conformance + OTel-semconv + deployable reference**. None of the four competitors has all four properties.

## Sources

### OpenTelemetry GenAI Semantic Conventions (HIGH confidence)
- [OpenTelemetry GenAI semantic conventions overview](https://opentelemetry.io/docs/specs/semconv/gen-ai/) — official spec
- [GenAI client spans (chat / text_completion / embeddings / execute_tool)](https://opentelemetry.io/docs/specs/semconv/gen-ai/gen-ai-spans/) — exact attribute list
- [GenAI agent and framework spans (invoke_agent / invoke_workflow)](https://opentelemetry.io/docs/specs/semconv/gen-ai/gen-ai-agent-spans/) — agent-level attributes
- [GenAI metrics (token usage, operation duration, TTFT, time-per-output-chunk)](https://opentelemetry.io/docs/specs/semconv/gen-ai/gen-ai-metrics/) — exact metric names
- [OpenTelemetry GenAI Semantic Conventions — The Standard for LLM Observability (DEV)](https://dev.to/x4nent/opentelemetry-genai-semantic-conventions-the-standard-for-llm-observability-1o2a) — vendor adoption notes (Datadog v1.37, Grafana Loki)

### Provider Go SDKs (HIGH confidence — verified at official repos)
- [openai/openai-go](https://github.com/openai/openai-go) — Responses API, streaming, tool calling, structured outputs, embeddings, batch, retry/backoff defaults
- [anthropics/anthropic-sdk-go](https://github.com/anthropics/anthropic-sdk-go) — Messages API, streaming, tool calling, prompt caching (`cache_control`), batches, retry, MCP support
- [ollama/ollama Go API client](https://pkg.go.dev/github.com/ollama/ollama/api) — Generate, Stream, tool calling (model-dependent), Embed
- [Ollama OpenAI compatibility layer](https://docs.ollama.com/api/openai-compatibility) — explicit OpenAI-shape compatibility for tools and embeddings

### Competitor Frameworks (MEDIUM confidence — README-level)
- [tmc/langchaingo](https://pkg.go.dev/github.com/tmc/langchaingo) — multi-provider Go LangChain port
- [LangChainGo embeddings docs](https://tmc.github.io/langchaingo/docs/modules/model_io/models/embeddings/) — provider list (OpenAI, HF, Jina, Voyage, Vertex, Bedrock)
- [tmc/langchaingo DeepWiki](https://deepwiki.com/tmc/langchaingo) — feature surface
- [cloudwego/eino on GitHub](https://github.com/cloudwego/eino) — ADK, callback aspects, graph orchestration, ChatModelAgent / DeepAgent
- [Eino overview at CloudWeGo](https://www.cloudwego.io/docs/eino/overview/) — component model, observability via callbacks
- [firebase/genkit Go module](https://pkg.go.dev/github.com/firebase/genkit/go) — flows, streaming, tool calling, dev UI
- [Announcing Genkit for Python and Go (Firebase blog)](https://firebase.blog/posts/2025/04/genkit-python-go/) — observability dashboard claim

### Cross-Cutting (MEDIUM confidence)
- [any-llm-go: One Interface for LLMs in Go (Mozilla AI)](https://blog.mozilla.ai/run-openai-claude-mistral-llamafile-and-more-from-one-interface-now-in-go/) — capability-flag pattern reference
- [Prompt caching for Anthropic and OpenAI (DigitalOcean)](https://www.digitalocean.com/blog/prompt-caching-with-digital-ocean) — 90% cost / 85% latency reduction figures
- [Anthropic prompt caching docs](https://platform.claude.com/docs/en/build-with-claude/prompt-caching) — `cache_control` mechanics
- [OpenAI prompt caching docs](https://developers.openai.com/api/docs/guides/prompt-caching) — implicit/automatic mechanics
- [JSON Mode vs Function Calling vs Structured Output: 2026 Guide](https://www.buildmvpfast.com/blog/structured-output-llm-json-mode-function-calling-production-guide-2026) — every-major-provider parity
- [grafana/docker-otel-lgtm](https://github.com/grafana/docker-otel-lgtm) — single-container LGTM bundle for the reference service compose file
- [Building a Complete Grafana LGTM Observability Platform with Docker Compose](https://blog.samzhu.dev/2025/03/25/Building-a-Complete-Grafana-LGTM-Observability-Platform-with-Docker-Compose/) — wiring reference

### Production / Guardrails (MEDIUM confidence)
- [How to Deploy AI Agents to Production: Budget Limits, Guardrails, and Monitoring (MindStudio)](https://www.mindstudio.ai/blog/deploy-ai-agents-production-budget-guardrails-monitoring) — multi-layered cost limits framing
- [AI Agent Security: Guardrails and Preventing Prompt Injection (Collabnix)](https://collabnix.com/ai-agent-security-guardrails-and-preventing-prompt-injection/) — pre-LLM/post-LLM filter pattern
- [Token-Based Rate Limiting (Zuplo)](https://zuplo.com/learning-center/token-based-rate-limiting-ai-agents) — TPM/TPD framing
- [Bifrost open-source LLM gateway in Go](https://www.dsinnovators.com/blog/golang/ai-apis-golang-concurrency-llm-2026/) — supports the "framework ≠ gateway" position in Anti-Features

---
*Feature research for: production-grade Go LLM agent framework v0.3 — providers + OTel + reference service*
*Researched: 2026-05-10*

# llm-agent

## What This Is

`llm-agent` is a stdlib-only Go framework for building LLM-driven agents — 5 classic paradigms (Simple/ReAct/Reflection/Plan-and-Solve/FunctionCall), Memory, RAG, Context engineering, communication protocols (MCP/A2A/ANP), multi-agent orchestration, and benchmarks. It currently ships v0.2.0 as a learning/prototype-stage library with deterministic mock-LLM demos; this milestone takes it from "library you can read" to "library you can deploy" by adding real provider adapters, OpenTelemetry observability, and a reference customer-support service — all in sister repos so the core stays stdlib-only.

## Core Value

**The core `llm-agent` module stays stdlib-only and zero-dep — anyone can `go get` it and read every line.** Every other decision in this milestone bends to preserve that property: providers, telemetry, and reference services live in sister repos, so users opt into dependencies one package at a time.

## Requirements

### Validated

<!-- Existing capabilities in the repo as of v0.2.0. -->

- ✓ **Agent paradigms** — Simple / ReAct / Reflection / Plan-and-Solve / FunctionCall constructors over a unified `Agent` interface — existing
- ✓ **`llm.Client` minimal contract** — `Generate` + `GenerateStream` + `Message` / `Tool` / `ToolCall` / `StreamChunk` types — existing
- ✓ **Builtin tools** — Calculator, MockSearch, NoteTool, TerminalTool — existing
- ✓ **Memory subsystem** — WorkingMemory / EpisodicMemory / SemanticMemory + Manager + MemoryTool — existing
- ✓ **RAG subsystem** — HashEmbedder, Chunker, InMemoryStore, RAGSystem, MQE, HyDE — existing
- ✓ **Context engineering** — GSSC ContextBuilder (Gather → Select → Structure → Compress) — existing
- ✓ **Communication protocols** — Envelope + Transport (HTTP/Stdio); toy MCP / A2A / ANP coverage — existing
- ✓ **Multi-agent orchestration** — Pipeline, FanOutFanIn, RoundRobinChat, RolePlay, StateGraph + Termination — existing
- ✓ **Agentic-RL scaffold** — Dataset, Trajectory, Reward, Evaluator, TrainerProxy (no training; Python TRL bridge planned) — existing
- ✓ **Benchmarks** — BFCL, GAIA, LLM-as-Judge, Win Rate, Reporter (mini fixtures) — existing
- ✓ **Runnable demos** — 5 standalone `go run .`-able programs in `/examples/`, deterministic mock LLM, no API key required — existing
- ✓ **Stdlib-only build** — zero non-stdlib deps in core module; CI compatible with no `go.sum` — existing

### Active

<!-- v0.3 milestone scope. Hypotheses until shipped. -->

**Provider adapters (sister repo: `llm-agent-providers`)**
- [ ] OpenAI adapter — Generate + Stream + native Tool calling + Embeddings
- [ ] Anthropic adapter — Generate + Stream + native Tool calling (Embeddings via 3rd party / N-A)
- [ ] Ollama adapter — Generate + Stream + Tool calling (where supported by model) + Embeddings
- [ ] Cross-provider streaming + tool-call interleaving conformance (httptest contract tests)

**Core abstraction evolution (this repo: `llm-agent`)**
- [ ] New `llm/v2` (or successor package) extending `Client` with `ToolCaller` + `Embedder` capability interfaces
- [ ] Dual-track BC: existing `llm.Client` retained and marked Deprecated; new code in parallel package; one-minor-cycle deprecation window
- [ ] Provider Author Guide — how to write a conforming provider, including streaming + tool-call wire format expectations
- [ ] Capability negotiation — agents/RAG ask provider what it supports rather than assuming

**Observability (sister repo: `llm-agent-otel`)**
- [ ] OpenTelemetry adapter — traces (per agent step / tool call / LLM call), metrics (token usage, latency, cost), logs (slog bridge)
- [ ] Span semantics for each paradigm (ReAct loop, PlanSolve plan/solve phases, StateGraph node transitions)
- [ ] Exporter wiring for the major collectors (OTLP/HTTP, OTLP/gRPC) — defaults that work with `otel-collector` in compose

**Reference service (sister repo: `llm-agent-customer-support`)**
- [ ] Multi-agent customer-support service — RAG-backed knowledge lookup + StateGraph triage routing + tool use
- [ ] Single-command bring-up — `docker compose up` brings up the service + Ollama + OTel collector + Grafana
- [ ] Provider switch via env — same service runs against OpenAI / Anthropic / Ollama with one variable
- [ ] HTTP API surface (chat endpoint), demonstrating real request handling, not just `go run .`

**Multi-repo infrastructure (umbrella concern)**
- [ ] `go.work` for local cross-repo dev
- [ ] CI strategy: PR runs mock-only; nightly job runs Ollama-live in a container; OpenAI/Anthropic verified by httptest wire-format tests
- [ ] Versioning across the four repos — `llm-agent` v0.3.x bumps; sister repos start at v0.1.x

### Out of Scope

<!-- Explicit v0.3 exclusions. Reasoning included so they don't get re-added. -->

- **Real RL training** — `rl/` keeps the Python TRL bridge stub; no in-process trainer. *Why:* concentrate budget on the three new directions; training is a wholly different scope and best left to a Python toolchain.
- **Vision / multimodal** — text-only adapters in v0.3. *Why:* dilutes the wire-format work; vision message structure is provider-specific enough to warrant its own milestone.
- **Cross-framework interop** — no LangChain / LlamaIndex / CrewAI bridges or schema converters. *Why:* keep the surface area pure-Go; let users compose at the agent level if they need it.
- **Production-grade distributed a2a/anp** — `comm/a2a` + `comm/anp` stay at toy/demo level. *Why:* "production distributed multi-agent" is a separate milestone needing service discovery, rate limiting, retry/circuit-breaking, and security review.
- **v1.0 stability commitment** — this milestone targets v0.3 (incremental, BC may break per existing 0.x policy). *Why:* "real-world feedback" gating v1.0 hasn't accumulated yet; ship v0.3, learn from real users running real workloads, then promote.
- **Single-repo monolith with build tags or hard provider deps** — providers/telemetry/refsvc do NOT live in this repo. *Why:* core's stdlib-only contract is the differentiating value (per Core Value).
- **Kubernetes manifests / Helm for the reference service** — defer to v0.4+ with kind/k3d CI from day one. *Why:* half-shipped K8s is worse than no K8s; v0.3 only promises the local compose demo stack.

## Context

**Project history:**
- Extracted from `costa92/aics-core/pkg/llm/agents` (commit `eadfe3c`)
- v0.2.0 just shipped — 5 examples landed in `/examples/`, CI wired for stdlib-only modules
- Original design specs lived in the parent AICS repo (`docs/superpowers/specs/2026-04-27` and `2026-05-06`)

**Codebase shape:**
- ~5000 LOC, 12 top-level packages
- Strict stdlib-only — no `go.sum`, CI compatible without one
- ScriptedLLM in `scriptedllm_test.go` is the de-facto mock; all examples use it
- `agents.NewRegistry` + the `Tool` interface are the extension points for builtins

**Why now:**
- The library is feature-complete enough on the abstraction side; what's missing is "hook it up to anything real and watch it work"
- Real provider integration is the validation step that gets v0.x → v1.0
- Observability + a deployable reference service is the difference between "interesting library" and "library people can adopt"

**Solo, side-project pace:**
- No external deadline; quality and clarity over velocity
- Prefer many small phases over a few big ones — each commit should be independently reviewable
- Cross-repo work means the .planning roadmap is umbrella-style; phases tag which repo they land in

## Constraints

- **Tech stack (core)**: Go ≥ 1.26, stdlib-only — no exception in `llm-agent` repo, ever. Sister repos may take deps but should justify each one.
- **Tech stack (providers)**: prefer official Go SDKs over hand-rolled HTTP where they exist (`github.com/openai/openai-go`, `github.com/anthropics/anthropic-sdk-go`); Ollama acceptable to hand-roll (simple HTTP).
- **Tech stack (telemetry)**: OpenTelemetry SDK (`go.opentelemetry.io/otel`) — fact-of-life standard.
- **Compatibility (BC)**: dual-track — keep `llm.Client` callable through one minor-version deprecation window before removal. Document migration in CHANGELOG `### Breaking` section per existing project policy.
- **CI cost**: mock-by-default. Live OpenAI/Anthropic calls forbidden in PR CI (cost + flakiness). Ollama-live nightly only.
- **Module layout**: 4 separate Go modules in 4 separate repos. `go.work` for local dev, `replace` directives in repo READMEs for downstream iteration.
- **Versioning**: this milestone targets v0.3.x in `llm-agent`; sister repos start fresh at v0.1.x. Per-line BC policy from README applies.
- **Out-of-band**: no commitment to API stability until v1.0 — `0.x major bump` may break per repo policy.

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Core stays stdlib-only; providers/telemetry/refsvc go to sister repos | Differentiating value of `llm-agent` is "zero-dep, readable in one sitting"; bending it once means bending it forever | — Pending |
| Three providers in parallel for v1 (OpenAI + Anthropic + Ollama) | Validates abstraction against three different wire formats and three different feature sets at once; better to discover abstraction holes early than ship + repeatedly retro-fit | — Pending |
| Full-depth providers (Generate + Stream + Tools + Embeddings) | Tool calling is what makes ReAct/FunctionCall paradigms actually useful; embeddings unlock RAG with non-Hash backing — both required for "production-grade" framing | — Pending |
| OpenTelemetry over custom interface | OTel is the de-facto standard; competing minimal interface costs more docs/ecosystem than it saves | — Pending |
| Reference service as a deployable service (`docker compose up`), not just a CLI demo | "Library people can adopt" requires showing it runs as a service with telemetry — CLI demo doesn't prove that | — Pending |
| Customer-support multi-agent as the reference scenario | Reuses the existing `support_triage` example as the seed; exercises RAG + Tools + StateGraph + multi-agent in one realistic flow | — Pending |
| Dual-track BC for `llm.Client` evolution | Avoids forcing existing callers (and the parent AICS repo) to migrate in one shot; matches Go ecosystem norms | — Pending |
| v0.3 incremental, not v1.0 jump | "Real-world feedback gating v1.0" hasn't accumulated; ship v0.3 to generate that feedback | — Pending |
| Umbrella .planning/ in `llm-agent` covering all 4 repos | Single source of truth for milestone progress; cross-repo dependencies tracked in one ROADMAP rather than fragmenting | — Pending |
| Mock-first CI; nightly Ollama-live; httptest contract tests | Cost-free PRs; some real-LLM verification; full coverage of wire-format conformance | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-05-11 after Phase 6 closeout cleanup*

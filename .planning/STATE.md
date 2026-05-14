# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-05-13)

**Core value:** The core `llm-agent` module stays stdlib-only and zero-dep — anyone can `go get` it and read every line. Providers, telemetry, and reference services live in sister repos so users opt into deps one package at a time.
**Current focus:** post-Phase-10 compatibility alignment and Phase 11
preparation for structure-aware retrieval

## Current Position

Phase: 10 — retrieval policies, hybrid recall, and context packing — complete 2026-05-14
Previous phase: 9 — source-aware ingestion and lifecycle controls — complete 2026-05-14
Plan: open Phase 11 for structure-aware retrieval now that the standalone
release is cut and the core compatibility facade has been aligned to
`llm-agent-rag v0.1.4`.
Status: milestone `v0.5` is active. Phases 8, 9, and 10 are complete. The
standalone repo shipped `v0.1.4`, and the core repo now consumes it while
preserving the historical `rag/` facade.
Last activity: 2026-05-14 — upgraded the core repo to `llm-agent-rag v0.1.4`,
aligned facade retrieval/tool plumbing to the standalone policy path, and
verified the full core test suite.

Progress: [██████████░] 85% of `v0.5` planned milestone executed

## Performance Metrics

**Velocity:**
- Total plans completed: 40
- Average duration: -
- Total execution time: -

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 0 | 6 | - | - |
| 1 | 7 | - | - |
| 2 | 4 | - | - |
| 3 | 5 | - | - |
| 4 | 5 | - | - |
| 5 | 5 | - | - |
| 6 | 8 | - | - |

**Recent Trend:**
- Last 5 plans: 06-04, 06-05, 06-06, 06-07, 06-08 completed
- Trend: milestone closeout is complete; focus has shifted to post-release transition and next-milestone setup

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Phase 0 (planned): Hybrid walking-skeleton build order RATIFIED (Conflict A) — Generate → Stream → Tools → Embeddings across all 3 providers in lockstep; OpenAI leads each gate.
- Phase 0 (planned): `ProviderInfo` per-(provider × model) via construction-time model binding RATIFIED (Conflict B; locks CORE-06 + K2).
- Phase 0 (planned): `replace` directives = README escape hatch (INFRA-06) AND CI release-gate (INFRA-04). No conflict — both ship.
- Phase 6 (planned): K8s OUT of v0.3 scope (Conflict D resolved). PROJECT.md Active list flagged for cleanup at next `/gsd-transition`.
- Phase 7 gate override: on 2026-05-12 the original calendar gate was manually opened by operator instruction; this does not broaden scope beyond `DEPRC-01..04`.
- Phase 7 `07-01` audit start: a repo scan shows the remaining internal legacy-surface usage is concentrated in `rag/`, `bench/`, `context/`, `rl/`, docs/examples, and the deprecated symbol definitions under `llm/legacy.go`.
- Phase 7 `07-02` close: runtime packages `rag/`, `context/`, `bench/`, and `rl/` now depend only on `llm.ChatModel`; targeted and full `go test` verification passed on 2026-05-12.
- Phase 7 `07-03` close: `llm/legacy.go` has been removed, docs/examples have been rewritten to the current API, and a regenerated `docs/api-snapshot.txt` now reflects the post-compat-removal surface.
- Phase 7 `07-04` audit: on 2026-05-13, a local workspace rooted at `/tmp/phase7-v04-audit/go.work` verified that `llm-agent-providers`, `llm-agent-otel`, and `llm-agent-customer-support` all pass their full test suites against the current core API with no source edits required.
- Phase 7 `07-05` closeout: an early 2026-05-13 attempt to bump sister-repo
  dependencies hit `unknown revision v0.4.0` before the core tag propagated,
  but the released core tag now resolves remotely, sister-repo `v0.4.0` bumps
  are landed, full verification passed, and coordinated tags were cut:
  `llm-agent-providers v0.1.1`, `llm-agent-otel v0.1.1`,
  `llm-agent-customer-support v0.1.0`.
- Governance bootstrap closeout: on 2026-05-13, all 3 sister repos converged on
  the same idempotent `pr-governance.yml` owner auto-merge behavior. The final
  stable shape is: job-level `contents: write` + `pull-requests: write`,
  `autoMergeRequest` pre-check before `gh pr merge --auto`, and an explicit
  understanding that `pull_request_target` bootstrap PRs may need one manual
  merge/auto-merge intervention before the fixed workflow reaches `main`.
- Phase 1 close: live Ollama verification is nightly-only by design; PR CI remains fixture-driven and Docker-free.
- Phase 1 close: `PROVIDER_AUTHORING.md` is now the canonical Generate-only third-party adapter contract.
- Phase 2 close: shared streaming conformance is now the contract gate before Phase 3 native tool-calling work.
- Phase 3 open: provider-native tools land before any core agent refactor; agent constructors consume capability interfaces, not provider names.
- Phase 3 plan 01 close: OpenAI tool support is modeled as a truthful capability on the bound provider/model, while actual tool attachment remains immutable per `WithTools(...)`.
- Phase 3 plan 02 close: Anthropic stays on the low-level Messages API rather than SDK `BetaToolRunner`; this preserves our adapter contract and concurrent-safety expectations.
- Phase 3 plan 03 close: Ollama tool support is now driven by a per-model strategy table; unsupported models fail honestly instead of silently degrading to free-text.
- Phase 3 plan 04 close: shared conformance now enforces calculator tool calls, parallel/multi-block behavior, capability-degrade, and dedupe-key invariants across providers.
- Phase 3 plan 05 close: core agents now bind to `llm.ChatModel`; `ReAct` selects native tools only when `ToolCaller` and `Capabilities.Tools` both hold, while `FunctionCallAgent` is native-only and rejects chat-only models at construction.
- Phase 4 close: provider embeddings now use the same capability-negotiation idiom as chat and tools; Anthropic's absence remains explicit contract data, not a hidden limitation.
- Phase 5 planning open: OTel stays in the sister repo, wrappers preserve capability interfaces, and semconv/content-capture/cardinality rules must land before any refsvc integration.
- Phase 5 plan 01 close: `otelmodel.Wrap(...)` now preserves `ToolCaller` / `Embedder` / `StructuredOutputs`, rewraps immutable bound models with the same tracer provider, and emits single-operation spans for generate/stream paths.
- Phase 5 plan 02 close: `otelagent.Wrap(...)` now preserves the public `Agent` contract and emits an `invoke_agent` root with bounded `chat` / `execute_tool` child spans driven only from streamed step events.
- Phase 5 plan 03 close: `gen_ai.*` constants and opt-in gates now live in one root file, metrics are emitted through a strict allowlist that excludes `user.id` / `session.id`, and content capture stays off by default with redaction support when enabled.
- Phase 5 plan 04 close: `otelslog.NewHandler(...)` now decorates any `slog.Handler` with `trace_id` / `span_id` correlation while preserving existing structured fields, including `gen_ai.*` keys.
- Phase 5 plan 05 close: OTLP exporter wiring now defaults to HTTP on `:4318`, a compose demo using `grafana/otel-lgtm` exists, and the README documents wrapper usage, opt-in semantics, defaults, and demo verification.
- Phase 6 plan 01 close: the reference service repo now has a thin but runnable bootstrap layer. Config loading, provider-aware model construction, OTel wrappers, signal-aware startup, and graceful shutdown are in place before the HTTP API layer lands.
- Phase 6 plan 02 close: the first transport surface is now wired. JSON chat, SSE chat streaming, health/readiness probes, and `X-Trace-Id` response propagation all share the same wrapped runtime.
- Phase 6 plan 03 close: chat and embedding provider selection are now independent. Anthropic chat plus OpenAI/Ollama embeddings is an explicit supported bootstrap combination, and provider selection logic is centralized in `internal/providers`.
- Phase 6 plan 04 close: the reference service now runs a real support flow. Explicit triage lives in `StateGraph`, refund knowledge lookup is tool-backed through RAG, and the HTTP transport now drives that flow instead of a `SimpleAgent`.
- Phase 6 plan 05 close: conversation state now lives outside agent instances. A shared session-store contract backs both SQLite and Postgres, the HTTP layer propagates/mints session IDs, and supportflow reloads prior transcript context across calls.
- Phase 6 plan 06 close: K7 guardrails now fail closed in the running service. Request caps are config-driven, panic-switch checks are live per request, and HTTP transports now surface `429`/`503` guard outcomes directly.
- Phase 6 plan 07 close: prompt-injection defense is now layered in the support flow. Suspicious inputs fail closed to a safe fallback, tool arguments no longer trust model-supplied identity, and RAG content is explicitly marked untrusted in the system prompt path.
- Phase 6 plan 08 close: the reference service now ships a local demo compose stack with app + Ollama + collector + Grafana assets, a pre-provisioned dashboard, tail-sampling config, and README startup / caveat guidance. Cold-stack runtime verification remains sensitive to first-run Docker and model download time.
- Phase 6 closeout follow-up: the compose collector asset now matches the roadmap's `decision_wait=30s` contract, blocked injection attempts set `prompt_injection_attempt=true` on the active trace, and v0.3 closeout should proceed through verification/audit rather than Phase 7 implementation.
- Phase 6 runtime verification: on 2026-05-12, a locally built server running against the live local dependency stack returned `200` from `/readyz` and `/chat`, emitted real `X-Trace-Id` headers, and confirmed that Grafana had provisioned the `Customer Support Observability` dashboard.
- Phase 6 observability closeout: on 2026-05-12, live verification exposed two real gaps in the demo observability path — the collector OTLP receiver was bound to loopback inside the container, and error spans recorded exceptions without setting `STATUS_CODE_ERROR`. After fixing both, a direct OTLP probe confirmed the configured tail-sampling branches: fast baseline retained 2/30 traces, while error and >5s traces were both retained 1/1.
- Phase 6 compose-native proof: on 2026-05-12, `/tmp/llm-agent-customer-support` built the compose `app` image successfully with `docker compose -f compose/compose.yaml build app`, and a compose-built `app` container then returned `200` from `/readyz` and `/chat` with real `X-Trace-Id` / `X-Session-Id` headers. The host still showed demo-environment sensitivity around `11434` port binding and `ollama-init` DNS resolution, but those no longer block milestone close.
- Milestone close decision was superseded on 2026-05-12 by an explicit operator override opening Phase 7 early.

### Pending Todos

- open Phase 11 for structure-aware retrieval and explainability
- keep standalone `llm-agent-rag` and core `llm-agent/rag` compatibility in lockstep

### Blockers/Concerns

No immediate implementation blocker. The standing constraint is still to
preserve the zero-dependency core value while pushing retrieval capability into
the standalone module first.

## Deferred Items

Items acknowledged and carried forward from previous milestone close:

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| *(none — fresh v0.3 milestone)* | | | |

## Session Continuity

Last session: 2026-05-14
Stopped at: Phase 10 closeout including core facade alignment to
`llm-agent-rag v0.1.4`; next action is Phase 11 planning/execution.
Resume file: .planning/ROADMAP.md

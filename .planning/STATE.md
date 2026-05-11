# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-05-10)

**Core value:** The core `llm-agent` module stays stdlib-only and zero-dep — anyone can `go get` it and read every line. Providers, telemetry, and reference services live in sister repos so users opt into deps one package at a time.
**Current focus:** Phase 4 planning ready; next execution wave is embeddings + RAG regression

## Current Position

Phase: 4 of 7 (Embeddings on OpenAI + Ollama; Anthropic gap documented) — planning ready 2026-05-11
Previous phase: 3 — Native tool calling on all 3 providers + agent refactor — ✓ COMPLETE 2026-05-11
Plan: 0 of 5 in Phase 4
Status: Phase 3 is complete and the Phase 4 planning bundle is now in place. Next execution wave is OpenAI/Ollama embeddings plus Anthropic documented-gap handling.
Last activity: 2026-05-11 — completed `03-05` in `llm-agent` and created the Phase 4 planning bundle for embeddings, conformance, and RAG regression.

Progress: [████░░░░░░] 50% (4 of 8 phases complete)

## Performance Metrics

**Velocity:**
- Total plans completed: 11
- Average duration: -
- Total execution time: -

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 1 | 7 | - | - |
| 2 | 4 | - | - |

**Recent Trend:**
- Last 5 plans: 03-01, 03-02, 03-03, 03-04, 03-05 completed
- Trend: tool-calling work is closed; embeddings are queued with a clean two-wave plan

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Phase 0 (planned): Hybrid walking-skeleton build order RATIFIED (Conflict A) — Generate → Stream → Tools → Embeddings across all 3 providers in lockstep; OpenAI leads each gate.
- Phase 0 (planned): `ProviderInfo` per-(provider × model) via construction-time model binding RATIFIED (Conflict B; locks CORE-06 + K2).
- Phase 0 (planned): `replace` directives = README escape hatch (INFRA-06) AND CI release-gate (INFRA-04). No conflict — both ship.
- Phase 6 (planned): K8s OUT of v0.3 scope (Conflict D resolved). PROJECT.md Active list flagged for cleanup at next `/gsd-transition`.
- Phase 7 is calendar-gated, not effort-gated — depends on a complete v0.3.0 cycle and one minor cycle of deprecation window.
- Phase 1 close: live Ollama verification is nightly-only by design; PR CI remains fixture-driven and Docker-free.
- Phase 1 close: `PROVIDER_AUTHORING.md` is now the canonical Generate-only third-party adapter contract.
- Phase 2 close: shared streaming conformance is now the contract gate before Phase 3 native tool-calling work.
- Phase 3 open: provider-native tools land before any core agent refactor; agent constructors consume capability interfaces, not provider names.
- Phase 3 plan 01 close: OpenAI tool support is modeled as a truthful capability on the bound provider/model, while actual tool attachment remains immutable per `WithTools(...)`.
- Phase 3 plan 02 close: Anthropic stays on the low-level Messages API rather than SDK `BetaToolRunner`; this preserves our adapter contract and concurrent-safety expectations.
- Phase 3 plan 03 close: Ollama tool support is now driven by a per-model strategy table; unsupported models fail honestly instead of silently degrading to free-text.
- Phase 3 plan 04 close: shared conformance now enforces calculator tool calls, parallel/multi-block behavior, capability-degrade, and dedupe-key invariants across providers.
- Phase 3 plan 05 close: core agents now bind to `llm.ChatModel`; `ReAct` selects native tools only when `ToolCaller` and `Capabilities.Tools` both hold, while `FunctionCallAgent` is native-only and rejects chat-only models at construction.
- Phase 4 planning open: embeddings will land on OpenAI and Ollama in parallel, Anthropic's gap stays explicit, then shared embedding conformance and RAG regression close the gate.

### Pending Todos

[From .planning/todos/pending/ — ideas captured during sessions]

- **PROJECT.md cleanup**: move "Optional Kubernetes manifests / Helm chart variant" from `### Active` to `### Out of Scope` at next `/gsd-transition` (per Conflict D resolution).
- ~~**Out-of-band Phase 0 close**: `git tag v0.3.0-pre.1 && git push --tags`~~ — ✓ done 2026-05-10.
- **Manual GitHub UI**: enable branch protection on `main` for the 3 sister repos (Settings → Branches → require status checks).
- **Post-merge workflow smoke test**: trigger `nightly-ollama-live` via `workflow_dispatch` after merge to validate GitHub-hosted Docker + cache behavior on the first real run.

### Blockers/Concerns

No current blocker. Next logical work is executing `04-01`, `04-02`, and `04-03` in parallel.

## Deferred Items

Items acknowledged and carried forward from previous milestone close:

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| *(none — fresh v0.3 milestone)* | | | |

## Session Continuity

Last session: 2026-05-11
Stopped at: Phase 4 planning ready; Wave 1 embeddings work is next.
Resume file: .planning/phases/04-embeddings-rag-regression/04-01-PLAN.md

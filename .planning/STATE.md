# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-05-10)

**Core value:** The core `llm-agent` module stays stdlib-only and zero-dep — anyone can `go get` it and read every line. Providers, telemetry, and reference services live in sister repos so users opt into deps one package at a time.
**Current focus:** Phase 2 planning - streaming on all 3 providers

## Current Position

Phase: 2 of 7 (Streaming on all 3 providers + StreamEvent validation) — PLANNING READY
Previous phase: 1 — Three-provider walking skeleton — Generate sync only — ✓ COMPLETE 2026-05-10
Plan: 2 of 4 in Phase 2
Status: OpenAI and Anthropic streaming landed. Ollama streaming remains in Wave 1 before shared streaming conformance.
Last activity: 2026-05-10 — completed `02-02` in `llm-agent-providers`: Anthropic `Stream()` now maps content-block events, preserves `partial_json` chunking, and emits final usage on `message_stop`.

Progress: [██▒░░░░░░░] 25% (2 of 8 phases complete)

## Performance Metrics

**Velocity:**
- Total plans completed: 9
- Average duration: -
- Total execution time: -

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 1 | 7 | - | - |
| 2 | 2 | - | - |

**Recent Trend:**
- Last 5 plans: 01-05, 01-06, 01-07, 02-01, 02-02 completed
- Trend: Phase 2 Wave 1 is half complete; next useful work is `02-03`, then `02-04`

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

### Pending Todos

[From .planning/todos/pending/ — ideas captured during sessions]

- **PROJECT.md cleanup**: move "Optional Kubernetes manifests / Helm chart variant" from `### Active` to `### Out of Scope` at next `/gsd-transition` (per Conflict D resolution).
- ~~**Out-of-band Phase 0 close**: `git tag v0.3.0-pre.1 && git push --tags`~~ — ✓ done 2026-05-10.
- **Manual GitHub UI**: enable branch protection on `main` for the 3 sister repos (Settings → Branches → require status checks).
- **Post-merge workflow smoke test**: trigger `nightly-ollama-live` via `workflow_dispatch` after merge to validate GitHub-hosted Docker + cache behavior on the first real run.

### Blockers/Concerns

No current blocker. Next logical work is executing `02-03` in the sister repo, then `02-04`.

## Deferred Items

Items acknowledged and carried forward from previous milestone close:

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| *(none — fresh v0.3 milestone)* | | | |

## Session Continuity

Last session: 2026-05-10
Stopped at: Phase 2 `02-02` complete; Anthropic streaming shipped.
Resume file: .planning/phases/02-streaming-stream-event-validation/02-02-SUMMARY.md

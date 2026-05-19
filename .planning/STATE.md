---
gsd_state_version: 1.0
milestone: v1.0
milestone_name: API stabilization and the compatibility promise
status: shipped
stopped_at: v1.0 milestone closed — committed, tagged v1.0.0, pushed, transitioned (2026-05-21)
last_updated: "2026-05-21T04:00:00.000Z"
last_activity: 2026-05-21 — v1.0 milestone close: committed, tagged v1.0.0, pushed, transitioned
progress:
  total_phases: 3
  completed_phases: 3
  total_plans: 9
  completed_plans: 9
  percent: 100
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-05-21)

**Core value:** The core `llm-agent` module stays stdlib-only and zero-dep — anyone can `go get` it and read every line. Providers, telemetry, and reference services live in sister repos so users opt into deps one package at a time.
**Current focus:** between milestones. v1.0 API stabilization shipped — `llm-agent-rag` has a frozen, documented, gate-protected `v1.0.0` public API. The next milestone is not yet scoped.

## Current Position

Milestone: `v1.0` API stabilization and the compatibility promise —
**shipped and closed 2026-05-21**.
Previous milestone: `v0.9` GraphRAG refinements — shipped 2026-05-20
(`llm-agent-rag v0.6.0`).
Status: v1.0 is fully closed. All three phases (28-30, 9 slices) executed
and verified green; milestone audit PASS (6/6 requirements
`RAG-API-01..06`, `.planning/v1.0-MILESTONE-AUDIT.md`, KS-1..KS-8
honored). The v1.0 work was committed to `llm-agent-rag` master
(`a76896d` feat + `170b944` changelog), tagged `v1.0.0` (on the feat
commit, the established layout), and pushed to origin. Milestone
transition done: v1.0 ROADMAP/REQUIREMENTS frozen to
`.planning/milestones/v1.0-{ROADMAP,REQUIREMENTS}.md`;
PROJECT/ROADMAP/REQUIREMENTS/STATE updated to "between milestones."

v1.0 froze the `llm-agent-rag` public API: a written exported-surface
audit and the pre-freeze breaking renames (`eval.Evaluator`→
`RetrievalEvaluator`, `eval.Result`→`RetrievalResult`; the `ragkit` root
repurposed as a documented doc-anchor); full package + exported-symbol
doc-comment coverage and a written `docs/compatibility.md` Go-module
compatibility promise; a pure-stdlib `internal/apisnapshot`
exported-surface gate (`api/v1.snapshot.txt` + a `go test`
regeneration-diff) plus a `-tags llmagent` CI step closing a long-standing
adapter CI gap. No new dependency, no behavior change. Scope was
`llm-agent-rag` only — the core module and sister repos stay on their own
version tracks.

Next step: scope the next milestone (candidates: the `llm-agent-rag`
deployment layer — HTTP service / CLI / caching; incremental community
maintenance; live-Postgres CI wiring; a core `llm-agent` or sister-repo
milestone). Awaiting operator direction.
Last activity: 2026-05-21 — v1.0 milestone close: committed
(`llm-agent-rag` `a76896d`/`170b944`), tagged `v1.0.0`, pushed master +
tag, ran the milestone transition (archive + planning-doc updates).

Progress: [██████████████] v1.0 API stabilization — shipped and closed
(`llm-agent-rag v1.0.0`, 6/6 requirements). v0.9 (`v0.6.0`), v0.8
(`v0.5.0`), v0.7 (`v0.4.0`), v0.6 (`v0.3.0`). Between milestones.

## Performance Metrics

**Velocity:**

- Total plans completed: 86 (40 through v0.5 + 14 in v0.6 phases 14-19 +
  8 in v0.7 phases 20-22 + 9 in v0.8 phases 23-25 + 6 in v0.9 phases 26-27
  + 9 in v1.0 phases 28-30)
- Average duration: -
- Total execution time: -

**By Phase:**

| Phase | Plans | Status |
|-------|-------|--------|
| 8 | 4 | complete |
| 9 | 3 | complete |
| 10 | 4 | complete |
| 11 | 13 | complete |
| 12 | 3 | complete |
| 13 | 4 | complete |
| 14 | 3 | complete |
| 15 | 2 | complete |
| 16 | 2 | complete |
| 17 | 3 | complete |
| 18 | 2 | complete |
| 19 | 2 | complete |
| 20 | 2 | complete |
| 21 | 3 | complete |
| 22 | 3 | complete |
| 23 | 3 | complete |
| 24 | 3 | complete |
| 25 | 3 | complete |
| 26 | 3 | complete |
| 27 | 3 | complete |
| 28 | 3 | complete |
| 29 | 3 | complete |
| 30 | 3 | complete |

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- 2026-05-15: new non-stdlib deps are allowed in `llm-agent-rag` only,
  isolated behind a subpackage/build tag like `postgres`. Core
  `llm-agent` stays stdlib-only.

- 2026-05-19: `v0.7` GraphRAG Tier-1 shipped — `llm-agent-rag` tagged
  `v0.4.0`, no new dependency.

- 2026-05-20: `v0.8` GraphRAG Tier-3 shipped — `llm-agent-rag` tagged
  `v0.5.0`, no new dependency.

- 2026-05-20: `v0.9` GraphRAG refinements shipped — `llm-agent-rag` tagged
  `v0.6.0`, no new dependency.

- 2026-05-21: `v1.0` API stabilization shipped — `llm-agent-rag` tagged
  `v1.0.0`, no new dependency. The public API is frozen and committed to
  the Go import-compatibility promise (KS-5); pre-freeze renames applied
  (KS-4); a stdlib `internal/apisnapshot` gate protects the surface
  (KS-6). Scope was `llm-agent-rag` only (KS-1); breaking changes from
  here require a `/v2` import path.

### Pending Todos

- Live-Postgres CI wiring (testcontainers-go or GH Actions services) —
  carried forward from v0.5; the Phase 14 `tsvector` path, the Phase 21
  `postgres` graph path, and the v0.8 `postgres`
  `_communities`/`_community_reports` paths are all unverified against a
  live DB.

- v0.7, v0.8, v0.9, and v1.0 milestone-closes are all fully complete —
  committed, tagged (`v0.4.0`/`v0.5.0`/`v0.6.0`/`v1.0.0`), pushed,
  transitioned, and archived. No follow-ups remain.

- `llm-agent-otel`'s `require llm-agent-rag` stays at `v0.3.0` —
  v0.7..v1.0 added no API `otelrag` depends on, so no bump is needed. A
  future bump to `v1.0.0` is optional housekeeping, not required.

- Incremental community maintenance is deferred (v0.9 keystone KG4-5) —
  v0.8's full re-detection on re-ingest stays until profiling shows it a
  bottleneck. Now a `v1.x`-additive candidate.

- Next milestone not yet scoped — see Current Position "Next step".

- Environment note: `git config --global url."git@github.com:".insteadOf
  "https://github.com/"` was set (operator-authorized) so `go mod` fetches
  the private `github.com/costa92/*` modules over SSH. It persists on this
  machine; harmless (mirrors the pre-existing `code.hellotalk.com` rewrite).

### Blockers/Concerns

- No immediate blocker — v1.0 is fully closed; `llm-agent-rag` has a
  stable, frozen, gate-protected `v1.0.0` API.

- Post-v1.0 discipline: the API is frozen. Within `v1.x` the surface is
  additive-only; any breaking change requires a `/v2` import path. The
  `internal/apisnapshot` gate (`api/v1.snapshot.txt`) and the cross-repo
  `contract` gate together fail any PR that breaks the promise — a
  deliberate `-update` regeneration is required for additive changes.

## Session Continuity

Last session: 2026-05-21
Stopped at: v1.0 milestone closed (2026-05-21)

v0.6, v0.7, v0.8, v0.9, and v1.0 are all fully shipped and closed —
`llm-agent-rag` tagged `v0.3.0`/`v0.4.0`/`v0.5.0`/`v0.6.0`/`v1.0.0`, each
audited PASS, committed, tagged, pushed, transitioned, and archived. The
GraphRAG arc is complete across v0.7-v0.9 (Tier-1, Tier-3, refinements);
v1.0 then froze and documented the whole `llm-agent-rag` public surface.

The `v1.0` API-stabilization milestone (phases 28-30, 9 slices) executed
and verified green, audited **PASS** (`.planning/v1.0-MILESTONE-AUDIT.md`,
6/6 `RAG-API-01..06`, KS-1..KS-8 honored), committed to `llm-agent-rag`
(`a76896d` feat + `170b944` changelog), tagged `v1.0.0`, and pushed.
Milestone transition done: v1.0 ROADMAP/REQUIREMENTS frozen to
`.planning/milestones/`, PROJECT/ROADMAP/REQUIREMENTS/STATE updated to
"between milestones." The core repo `.planning/` tree of the v1.0 close
is committed by the operator.

The project is now between milestones. `llm-agent-rag` has a frozen,
fully-documented, gate-protected `v1.0.0` public API and a written
compatibility promise. Next step: scope the next milestone — candidates
are the `llm-agent-rag` deployment layer (HTTP service, CLI, caching — the
first obvious `v1.x` additive direction), incremental community
maintenance (deferred from v0.9 by KG4-5), live-Postgres CI wiring, or a
milestone on the core `llm-agent` module / a sister repo. Awaiting
operator direction.
Resume file: .planning/ROADMAP.md

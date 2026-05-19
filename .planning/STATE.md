---
gsd_state_version: 1.0
milestone: v0.9
milestone_name: GraphRAG refinements — DRIFT search and path-ranking
status: shipped
stopped_at: v0.9 milestone closed — committed, tagged v0.6.0, pushed, transitioned (2026-05-20)
last_updated: "2026-05-20T18:00:00.000Z"
last_activity: 2026-05-20 — v0.9 milestone close: committed, tagged v0.6.0, pushed, transitioned
progress:
  total_phases: 2
  completed_phases: 2
  total_plans: 6
  completed_plans: 6
  percent: 100
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-05-18)

**Core value:** The core `llm-agent` module stays stdlib-only and zero-dep — anyone can `go get` it and read every line. Providers, telemetry, and reference services live in sister repos so users opt into deps one package at a time.
**Current focus:** between milestones. v0.9 GraphRAG refinements shipped — the next milestone is not yet scoped.

## Current Position

Milestone: `v0.9` GraphRAG refinements — **shipped and closed 2026-05-20**.
Previous milestone: `v0.8` GraphRAG Tier-3 — shipped 2026-05-20
(`llm-agent-rag v0.5.0`).
Status: v0.9 GraphRAG refinements is fully closed. Both phases (26-27, 6
slices) executed and verified green; milestone audit PASS (4/4 requirements
`RAG-GRAPH4-01..04`, `.planning/v0.9-MILESTONE-AUDIT.md`, KG4-1..KG4-7
honored). The v0.9 work was committed to `llm-agent-rag` master
(`5d16007` feat + `1d6e206` changelog), tagged `v0.6.0` (on the feat
commit, the established layout), and pushed to origin. Milestone transition
done: v0.9 ROADMAP/REQUIREMENTS frozen to
`.planning/milestones/v0.9-{ROADMAP,REQUIREMENTS}.md`;
PROJECT/ROADMAP/REQUIREMENTS/STATE updated to "between milestones."
The GraphRAG refinements shipped: deterministic stdlib path ranking
(`graph.PathRanker` + an opt-in `retrieve.GraphRetriever` mode with
subgraph-as-evidence) and DRIFT hybrid search (`rag.System.AskDrift` +
`eval.DriftEvaluator`). `llm-agent-rag` gained no new dependency.
Incremental community maintenance was deferred to v1.0+ (KG4-5).

The GraphRAG arc is complete across v0.7-v0.9: Tier-1 (relationship-traversal
retrieval), Tier-3 (communities + global search + fuzzy resolution), and
refinements (path-ranking + DRIFT). The SDK spans the full practical
GraphRAG spectrum.
Next step: scope the next milestone (candidates: incremental community
maintenance; the `llm-agent-rag` deployment layer; live-Postgres CI wiring;
a v1.0 API-stability pass). Awaiting operator direction.
Last activity: 2026-05-20 — v0.9 milestone close: committed
(`llm-agent-rag` `5d16007`/`1d6e206`), tagged `v0.6.0`, pushed master +
tag, ran the milestone transition (archive + planning-doc updates).

Progress: [██████████████] v0.9 GraphRAG refinements — shipped and closed
(`llm-agent-rag v0.6.0`, 4/4 requirements). v0.8 (`v0.5.0`), v0.7
(`v0.4.0`), v0.6 (`v0.3.0`). Between milestones.

## Performance Metrics

**Velocity:**

- Total plans completed: 77 (40 through v0.5 + 14 in v0.6 phases 14-19 +
  8 in v0.7 phases 20-22 + 9 in v0.8 phases 23-25 + 6 in v0.9 phases 26-27)
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

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- 2026-05-15: `v0.5` shipped — `llm-agent-rag` tagged `v0.2.0`,
  `llm-agent-otel` consumes it (`replace` removed), core `llm-agent/rag`
  facade aligned.

- 2026-05-15: v0.6 scope deepens six 🟡 Partial seams (retrieval, rerank,
  eval, observability, security, agentic); deployment layer (HTTP/CLI/cache)
  deferred past v0.6.

- 2026-05-15: new non-stdlib deps for v0.6 are allowed in `llm-agent-rag`
  only, isolated behind a subpackage/build tag like `postgres`. Core
  `llm-agent` stays stdlib-only.

- 2026-05-19: `v0.7` GraphRAG Tier-1 shipped — `llm-agent-rag` tagged
  `v0.4.0`, no new dependency.

- 2026-05-20: `v0.8` GraphRAG Tier-3 shipped — `llm-agent-rag` tagged
  `v0.5.0`, no new dependency.

- 2026-05-20: `v0.9` GraphRAG refinements shipped — `llm-agent-rag` tagged
  `v0.6.0`, no new dependency. Path-ranking + DRIFT delivered; incremental
  community maintenance deferred to v1.0+ (KG4-5).

### Pending Todos

- Live-Postgres CI wiring (testcontainers-go or GH Actions services) —
  carried forward from v0.5; the Phase 14 `tsvector` path, the Phase 21
  `postgres` graph path, and the v0.8 `postgres`
  `_communities`/`_community_reports` paths are all unverified against a
  live DB.

- v0.7, v0.8, and v0.9 milestone-closes are all fully complete — committed,
  tagged (`v0.4.0`/`v0.5.0`/`v0.6.0`), pushed, transitioned, and archived.
  No follow-ups remain.

- `llm-agent-otel`'s `require llm-agent-rag` stays at `v0.3.0` —
  v0.7/v0.8/v0.9 added no API `otelrag` depends on, so no bump is needed.

- Incremental community maintenance is deferred to v1.0+ (v0.9 keystone
  KG4-5) — v0.8's full re-detection on re-ingest stays until profiling
  shows it a bottleneck.

- Next milestone not yet scoped — see Current Position "Next step".

- Environment note: `git config --global url."git@github.com:".insteadOf
  "https://github.com/"` was set (operator-authorized) so `go mod` fetches
  the private `github.com/costa92/*` modules over SSH. It persists on this
  machine; harmless (mirrors the pre-existing `code.hellotalk.com` rewrite).

### Blockers/Concerns

- No immediate implementation blocker — the v0.6 milestone is fully closed
  and the 17-03 cross-repo dependency debt is resolved.

- (Resolved 2026-05-18) The `llm-agent-otel` `require` bump to
  `llm-agent-rag v0.3.0` was initially blocked — `go mod` fetches the
  private `github.com/costa92/*` modules over HTTPS, which has no
  credentials in this environment. Fixed (operator-authorized) with
  `git config --global url."git@github.com:".insteadOf "https://github.com/"`
  plus `GOPRIVATE=github.com/costa92/*` to skip the proxy/checksum-DB;
  `go mod tidy` then fetched `v0.3.0` over SSH. `otelrag` now builds under
  `GOWORK=off` against `llm-agent-rag v0.3.0` (`4ddbc4c`, pushed).

## Session Continuity

Last session: 2026-05-20
Stopped at: v0.9 milestone closed (2026-05-20)

v0.6, v0.7, v0.8, and v0.9 are all fully shipped and closed —
`llm-agent-rag` tagged `v0.3.0`/`v0.4.0`/`v0.5.0`/`v0.6.0`, each audited
PASS, committed, tagged, pushed, transitioned, and archived. The GraphRAG
arc is complete across v0.7-v0.9: Tier-1 (entity/relation extraction +
relationship-traversal retrieval), Tier-3 (community detection + global
search + fuzzy resolution), and refinements (path-ranking + DRIFT). The
SDK spans the full practical GraphRAG spectrum.

The `v0.9` GraphRAG refinements milestone (phases 26-27) executed and
verified green, audited **PASS** (`.planning/v0.9-MILESTONE-AUDIT.md`, 4/4
`RAG-GRAPH4-01..04`, KG4-1..KG4-7 honored), committed to `llm-agent-rag`
(`5d16007`/`1d6e206`), tagged `v0.6.0`, and pushed. Milestone transition
done: v0.9 ROADMAP/REQUIREMENTS frozen to `.planning/milestones/`,
PROJECT/ROADMAP/REQUIREMENTS/STATE updated to "between milestones." The
core repo `.planning/` tree of the v0.9 close is committed separately by
the operator.

The project is now between milestones. Next step: scope the next
milestone — candidates are incremental community maintenance (deferred
from v0.9 by KG4-5), the `llm-agent-rag` deployment layer (HTTP service,
CLI, caching), live-Postgres CI wiring, or a v1.0 API-stability pass.
Awaiting operator direction.
Resume file: .planning/ROADMAP.md

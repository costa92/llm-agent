---
gsd_state_version: 1.0
milestone: v0.8
milestone_name: GraphRAG Tier-3 — communities, global search, fuzzy resolution
status: shipped
stopped_at: v0.8 milestone closed — committed, tagged v0.5.0, pushed, transitioned (2026-05-20)
last_updated: "2026-05-20T04:00:00.000Z"
last_activity: 2026-05-20 — v0.8 milestone close: committed, tagged v0.5.0, pushed, transitioned
progress:
  total_phases: 3
  completed_phases: 3
  total_plans: 9
  completed_plans: 9
  percent: 100
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-05-18)

**Core value:** The core `llm-agent` module stays stdlib-only and zero-dep — anyone can `go get` it and read every line. Providers, telemetry, and reference services live in sister repos so users opt into deps one package at a time.
**Current focus:** between milestones. v0.8 GraphRAG Tier-3 shipped — the next milestone is not yet scoped.

## Current Position

Milestone: `v0.8` GraphRAG Tier-3 — **shipped and closed 2026-05-20**.
Previous milestone: `v0.7` GraphRAG Tier-1 — shipped 2026-05-19
(`llm-agent-rag v0.4.0`).
Status: v0.8 GraphRAG Tier-3 is fully closed. All three phases (23-25, 9
slices) executed and verified green; milestone audit PASS (6/6 requirements
`RAG-GRAPH3-01..06`, `.planning/v0.8-MILESTONE-AUDIT.md`, KG3-1..KG3-8
honored). The v0.8 work was committed to `llm-agent-rag` master
(`fd58ef0` feat + `00e9fb3` changelog), tagged `v0.5.0` (on the feat
commit, mirroring the v0.3.0/v0.4.0 layout), and pushed to origin.
Milestone transition done: v0.8 ROADMAP/REQUIREMENTS frozen to
`.planning/milestones/v0.8-{ROADMAP,REQUIREMENTS}.md`;
PROJECT/ROADMAP/REQUIREMENTS/STATE updated to "between milestones."
The Tier-3 GraphRAG stack shipped: deterministic stdlib community detection
(Louvain), the `store.CommunityStore` capability (in-memory + postgres),
lazy community summaries, the `rag.System.AskGlobal` map-reduce
global-search answer path, an opt-in `EmbeddingEntityResolver` fuzzy-merge
pre-pass, and `eval.GlobalEvaluator`. `llm-agent-rag` gained no new
dependency and no graph database. (The postgres community/report paths are
env-gated — unverified against a live DB, carried-forward debt.)
Next step: scope the next milestone (candidates: v0.9 GraphRAG refinements
— DRIFT search / incremental community maintenance / path-ranking; the
`llm-agent-rag` deployment layer; live-Postgres CI wiring). Awaiting
operator direction.
Last activity: 2026-05-20 — v0.8 milestone close: committed
(`llm-agent-rag` `fd58ef0`/`00e9fb3`), tagged `v0.5.0`, pushed master +
tag, ran the milestone transition (archive + planning-doc updates).

Progress: [██████████████] v0.8 GraphRAG Tier-3 — shipped and closed
(`llm-agent-rag v0.5.0`, 6/6 requirements). v0.7 shipped (`v0.4.0`), v0.6
shipped (`v0.3.0`). Between milestones.

## Performance Metrics

**Velocity:**

- Total plans completed: 71 (40 through v0.5 + 14 in v0.6 phases 14-19 +
  8 in v0.7 phases 20-22 + 9 in v0.8 phases 23-25)
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

- 2026-05-19: `v0.8` opened — GraphRAG Tier-3. Keystone KG3-2: community
  summarization is **lazy by default** (LazyGraphRAG); KG3-3: community
  detection is **pure stdlib** (Louvain), store-agnostic; KG3-4: global
  search is a separate `AskGlobal` path, not a `Retriever`. DRIFT deferred
  to v0.9. Expected to need no new dependency.

### Pending Todos

- Live-Postgres CI wiring (testcontainers-go or GH Actions services) —
  carried forward from v0.5; the Phase 14 `tsvector` path, the Phase 21
  `postgres` graph path, and the upcoming v0.8 `postgres`
  `_communities`/`_community_reports` paths are all unverified against a
  live DB.

- v0.7 milestone-close is fully complete — committed (`llm-agent-rag`
  `12d303f` feat + `ac119f8` changelog), tagged `v0.4.0` (on `12d303f`,
  one commit before the changelog — same layout as `v0.3.0`/`798bf3f`),
  pushed master + tag, milestone transitioned and archived. No v0.7
  follow-ups remain.

- `llm-agent-otel`'s `require llm-agent-rag` stays at `v0.3.0` — v0.7
  added no API `otelrag` depends on, so no bump is needed. Bump only if a
  future change requires it.

- v0.8 GraphRAG Tier-3 milestone-close is fully complete — committed
  (`llm-agent-rag` `fd58ef0` feat + `00e9fb3` changelog), tagged `v0.5.0`,
  pushed master + tag, milestone transitioned and archived. No v0.8
  follow-ups remain.

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

Last session: 2026-05-18T14:30:07.317Z
Stopped at: context exhaustion at 75% (2026-05-18)
executed and verified green; milestone audit PASS (12/12 requirements,
`.planning/v0.6-MILESTONE-AUDIT.md`). Committed, merged to `master`, tagged
`llm-agent-rag v0.3.0`, and pushed all three repos to origin. Milestone
transition done: v0.6 ROADMAP/REQUIREMENTS archived to
`.planning/milestones/`, PROJECT/ROADMAP/STATE updated to "between
milestones."
v0.6 and v0.7 are both fully shipped and closed. The `v0.7` GraphRAG
milestone (phases 20-22) executed and verified green, audited **PASS**
(`.planning/v0.7-MILESTONE-AUDIT.md`, 6/6 `RAG-GRAPH-01..06`, KG-1..KG-7
honored), committed to `llm-agent-rag` (`12d303f`/`ac119f8`), tagged
`v0.4.0`, and pushed. Milestone transition done: v0.7 ROADMAP/REQUIREMENTS
frozen to `.planning/milestones/`, PROJECT/ROADMAP/REQUIREMENTS/STATE
updated to "between milestones." `.planning/` tree of this core repo is
itself not yet committed — the operator commits core-repo planning docs
separately.
The `v0.8` GraphRAG Tier-3 milestone is in progress. Scoped 2026-05-19 from
`.planning/research/v0.8-graphrag-tier3-SUMMARY.md`; REQUIREMENTS
(`RAG-GRAPH3-01..06`) + ROADMAP (Phases 23-25, keystones KG3-1..KG3-8). It
delivers the Tier-3 GraphRAG v0.7 deferred: hierarchical community detection
(pure-stdlib Louvain), lazy community summaries, a map-reduce global-search
answer path (`rag.System.AskGlobal`), and embedding-similarity fuzzy entity
resolution. DRIFT search is deferred to v0.9.

**v0.8 is fully shipped and closed.** All three phases (23-25, 9 slices)
executed and verified green, audited PASS
(`.planning/v0.8-MILESTONE-AUDIT.md`, 6/6 `RAG-GRAPH3-01..06`, KG3-1..KG3-8
honored). Committed to `llm-agent-rag` (`fd58ef0` feat + `00e9fb3`
changelog), tagged `v0.5.0`, pushed. Milestone transition done: v0.8
ROADMAP/REQUIREMENTS frozen to `.planning/milestones/`,
PROJECT/ROADMAP/REQUIREMENTS/STATE updated to "between milestones." The
core repo `.planning/` tree (v0.8 close) is committed separately by the
operator.

The project is now between milestones. Next step: scope the next
milestone — candidates are v0.9 GraphRAG refinements (DRIFT search,
incremental community maintenance, path-ranking), the `llm-agent-rag`
deployment layer, or live-Postgres CI wiring. Awaiting operator direction.
Resume file: .planning/ROADMAP.md

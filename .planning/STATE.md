---
gsd_state_version: 1.0
milestone: v1.1
milestone_name: Ecosystem Alignment
status: between-milestones
stopped_at: Phase 34 complete — v1.1 audit PASS 5/5; pending operator milestone-close commit + /gsd-transition (2026-05-20)
last_updated: "2026-05-20T06:10:00.000Z"
last_activity: 2026-05-20 — Phase 34 closed (9-slice expansion); v1.1 audit PASS
progress:
  total_phases: 4
  completed_phases: 4
  total_plans: 19
  completed_plans: 19
  percent: 100
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-05-21)

**Core value:** The core `llm-agent` module stays stdlib-only and zero-dep — anyone can `go get` it and read every line. Providers, telemetry, and reference services live in sister repos so users opt into deps one package at a time.
**Current focus:** **between milestones.** v1.1 (ecosystem alignment) shipped and closed 2026-05-20 (audit PASS 5/5); pending the operator milestone-close commit + `/gsd-transition`. Next milestone unscoped.

## Current Position

Milestone: `v1.1` Ecosystem Alignment — **shipped and closed 2026-05-20**
(audit PASS 5/5).
Previous milestone: `v1.0` API stabilization — shipped and closed
2026-05-21 (`llm-agent-rag v1.0.0`, audit PASS 6/6, fully archived).
Plan: `v1.1` Ecosystem Alignment scoped 2026-05-21 from
`.planning/research/v1.1-ecosystem-alignment-SUMMARY.md`. 5 requirements
(`ECO-01..05`) across 4 phases:
Phase 31 — core RAG facade re-alignment to `llm-agent-rag v1.0.0` (the dep
is 8 minors + a major stale; bumping it surfaces a 7-test
`vector dimension mismatch` regression in the facade adapters).
Phase 32 — sister-repo branch landing & hygiene (merge `otel`'s stranded
4-commit `otelrag` feature branch and `customer-support`'s 2-commit CI-fix
branch to `main`; prune stale branches).
Phase 33 — coordinated dependency-bump & re-tag wave (core `v0.5.0`,
`otel`/`providers`/`customer-support` `v0.2.0`, dependency-ordered, no
`replace` directives).
Phase 34 — umbrella dependency-currency CI gate, 5-repo coherence
verification, milestone close.
Keystone calls KE-1..KE-7 (research doc + ROADMAP): alignment not features;
`llm-agent-rag` is the untouched fixed point (no rag re-tag, back-edge
left as-is); the core facade is repaired and proven stdlib-only; branches
land before tags; a coordinated no-`replace` tag wave; the umbrella gains
a drift-detection gate; live-Postgres CI stays deferred.
Status: **all 4 phases (31-34) are complete; v1.1 audit PASS 5/5.**
Phases 31-33 shipped the original alignment arc (core facade repair,
sister-repo branch landing, coordinated base-line re-tag wave); Phase 34
expanded from 3 to 9 slices mid-flight to satisfy the strict
dep-currency gate (cascade of patch tags through the back-edge refresh)
and to correct a topological-order miss in Phase 33's cascade
(`customer-support` re-tagged once `providers` was in place).

**Final coordinated v1.1 tag set** (post-cascade): `llm-agent v0.5.1`,
`llm-agent-rag v1.0.1`, `llm-agent-otel v0.2.1`,
`llm-agent-providers v0.2.1`, `llm-agent-customer-support v0.2.2`. All
pushed. Zero `replace` directives across all tagged branches. Umbrella
dep-currency CI gate shipped at `acb3253` (the core HEAD is 1 commit
past `v0.5.1`; gate is build-system infrastructure, no new tag).

Three architectural trade-offs documented honestly in the audit
(`.planning/v1.1-MILESTONE-AUDIT.md` §Trade-offs): (i) the v1.0.0 →
v1.0.1 freeze-day-after re-tag — chore-only patch, KE-2 holds; (ii)
the topological-order miss in the cascade — future cascades must `tsort`
against the dep DAG, not intuition; (iii) the rag↔core cycle exemption
— the one auditable strict-equality exemption in the gate, narrow on
purpose.

Next step: operator runs `git add .planning/ && git commit -m 'docs(planning): close v1.1 ecosystem alignment milestone (audit PASS 5/5)' && git push origin main` (mirrors v1.0 close `48cbbc9`), then `/gsd-transition`.
Last activity: 2026-05-20 — Phase 34 closed (9 slices); v1.1 audit PASS.

Progress: [██████████████] v1.1 Ecosystem Alignment — 4/4 phases
complete. v1.0 shipped (`llm-agent-rag v1.0.0`), v0.9 (`v0.6.0`), v0.8
(`v0.5.0`).

## Performance Metrics

**Velocity:**

- Total plans completed: 105 (40 through v0.5 + 14 in v0.6 phases 14-19 +
  8 in v0.7 phases 20-22 + 9 in v0.8 phases 23-25 + 6 in v0.9 phases 26-27
  + 9 in v1.0 phases 28-30 + 19 in v1.1 phases 31-34 — Phase 34 expanded
  from 3 to 9 slices mid-flight)
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
| 31 | 3 | complete |
| 32 | 3 | complete |
| 33 | 4 | complete |
| 34 | 9 | complete |

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

- 2026-05-21: `v1.1` Ecosystem Alignment opened — bring the core
  `llm-agent` + the three sister repos current with `llm-agent-rag
  v1.0.0`. Operator-chosen direction ("全部都需要执行" — all repos).
  Keystone calls KE-1..KE-7: alignment not features (KE-1); `llm-agent-rag`
  is the untouched fixed point — no rag re-tag, back-edge left as-is
  (KE-2); the core RAG facade repaired and proven stdlib-only (KE-3);
  branches land before tags (KE-4); a coordinated no-`replace` tag wave —
  core `v0.5.0`, sister repos `v0.2.0` (KE-5); an umbrella
  dependency-currency CI gate (KE-6). No new feature, no new dependency.

- 2026-05-20: `v1.1` shipped and closed — audit PASS 5/5
  (`.planning/v1.1-MILESTONE-AUDIT.md`). Final coordinated tag set
  `llm-agent v0.5.1` / `llm-agent-rag v1.0.1` /
  `llm-agent-otel v0.2.1` / `llm-agent-providers v0.2.1` /
  `llm-agent-customer-support v0.2.2`. Phase 34 expanded from 3 slices
  to 9 mid-flight to satisfy the strict dep-currency gate (cascade of
  patch tags through the back-edge refresh) and correct a
  topological-order miss in Phase 33's cascade. Three trade-offs
  documented honestly: v1.0.0 → v1.0.1 freeze-day-after (KE-2 holds —
  chore-only patch, no exported-symbol move); topological-order miss in
  the cascade (future cascades must `tsort` against the dep DAG); the
  rag↔core cycle exemption in the dep-currency gate (one narrow
  auditable strict-equality exemption). Pending the operator
  milestone-close commit.

### Pending Todos

- Live-Postgres CI wiring (testcontainers-go or GH Actions services) —
  carried forward from v0.5; the Phase 14 `tsvector` path, the Phase 21
  `postgres` graph path, and the v0.8 `postgres`
  `_communities`/`_community_reports` paths are all unverified against a
  live DB.

- v0.7, v0.8, v0.9, and v1.0 milestone-closes are all fully complete —
  committed, tagged, pushed, transitioned, and archived. The core repo
  `.planning/` tree of the v1.0 close is committed (`48cbbc9`) and pushed.

- v1.1 fully shipped and closed (audit PASS 5/5, 2026-05-20). All 5
  coordinated tags pushed (final post-cascade set: `llm-agent v0.5.1`,
  `llm-agent-rag v1.0.1`, `llm-agent-otel v0.2.1`,
  `llm-agent-providers v0.2.1`, `llm-agent-customer-support v0.2.2`).
  Umbrella dep-currency CI gate shipped (commit `acb3253`). Pending the
  operator milestone-close commit (mirrors v1.0 `48cbbc9`) and
  `/gsd-transition` to between-milestones state.

- Operator-authorized environment changes during Phase 33:
  `llm-agent-rag` repo visibility flipped to **public** to unblock
  cross-repo CI on the public sister repos. Aligned with v1.0's
  `docs/compatibility.md` public-SDK framing.

- Stale REMOTE branches across the sister repos (`chore/bump-llm-agent-v0.4.0`,
  `docs/link-governance-guides`, merged `fix/*`, `verify/*`) are recorded
  in the 32-0x SUMMARYs for the operator to prune on the remotes — Phase
  32 pruned only local branches.

- v1.1 will resolve the long-standing sister-repo dep staleness: the core
  `llm-agent` `go.mod` pins `llm-agent-rag v0.1.4` (8 minors + a major
  stale); `llm-agent-otel` pins `v0.3.0`; `customer-support` pins stale
  `otel`/`providers` tags. The coordinated re-tag wave (Phase 33) and the
  dependency-currency gate (Phase 34) close this.

- Incremental community maintenance is deferred (v0.9 keystone KG4-5) and
  is NOT in v1.1 scope (KE-1 — v1.1 is alignment, not features). A
  `v1.x`-additive candidate for a later rag milestone.

- Environment note: `git config --global url."git@github.com:".insteadOf
  "https://github.com/"` was set (operator-authorized) so `go mod` fetches
  the private `github.com/costa92/*` modules over SSH. It persists on this
  machine; harmless (mirrors the pre-existing `code.hellotalk.com` rewrite).

### Blockers/Concerns

- No immediate blocker — v1.0 is fully closed; v1.1 is defined and ready
  to plan. The research confirmed all five repos build/test green and
  every cross-repo bump resolves.

- One known v1.1 code fix: bumping the core `llm-agent` to `llm-agent-rag
  v1.0.0` surfaces 7 facade-test `vector dimension mismatch` failures — a
  real regression in the compatibility facade (preliminary root cause:
  the facade adapter wires a mismatched store/query embedding dimension).
  Phase 31 owns the exact dimension-contract diff and the fix. It must be
  fixed inside the facade — never by adding a dependency (KE-3).

- Post-v1.0 discipline (`llm-agent-rag`): the API is frozen — additive-only
  within `v1.x`, breaking ⇒ `/v2`. v1.1 does not touch rag (KE-2).

## Session Continuity

Last session: 2026-05-20
Stopped at: Phase 34 complete — v1.1 audit PASS 5/5; pending operator milestone-close commit + `/gsd-transition` (between-milestones)

v1.0 is fully shipped and closed — `llm-agent-rag` tagged `v1.0.0`, audit
PASS 6/6, committed (`a76896d` feat + `170b944` changelog), pushed,
transitioned and archived; the core repo `.planning/` close commit
(`48cbbc9`) is committed and pushed. `llm-agent-rag` now has a frozen,
fully-documented, gate-protected `v1.0.0` public API and a written
compatibility promise.

The **`v1.1` Ecosystem Alignment milestone is shipped and closed**
(audit PASS 5/5, 2026-05-20). All 4 phases (31-34) complete; 19 slices
executed total (Phase 34 expanded from 3 to 9 mid-flight).

**Final coordinated v1.1 tag set** (post-cascade, all pushed):

- `llm-agent v0.5.1` @ `88db43e` (HEAD on `main` is `acb3253` — the
  Wave 7 umbrella-gate commit, build-system infrastructure, no new tag)
- `llm-agent-rag v1.0.1` @ `09697ca`
- `llm-agent-otel v0.2.1` @ `c7ebda7`
- `llm-agent-providers v0.2.1` @ `efdef5a`
- `llm-agent-customer-support v0.2.2` @ `ca62e5b`

Zero `replace` directives across all tagged branches. Dep-currency gate
exit 0. 5/5 vet/build/test -short green. 4/4 sister working trees
clean. Audit: `.planning/v1.1-MILESTONE-AUDIT.md` — verdict ✅ PASS.

Three trade-offs surfaced and documented honestly: (i) the v1.0.0 →
v1.0.1 freeze-day-after re-tag (KE-2 holds — chore-only patch, no
exported-symbol move); (ii) the topological-order miss in Phase 33's
cascade (Wave 6 follow-up re-tagged `cs v0.2.2` once `providers v0.2.1`
existed; future cascades must `tsort` against the dep DAG); (iii) the
rag↔core cycle exemption in the dep-currency gate (the one auditable
strict-equality exemption — narrow on purpose).

**Pending operator action — milestone-close commit.** The planning-tree
edits authored by slice 34-09 are staged (`.planning/v1.1-MILESTONE-AUDIT.md`,
`.planning/PROJECT.md`, `.planning/STATE.md`, `.planning/ROADMAP.md`,
`.planning/REQUIREMENTS.md`, `.planning/milestones/v1.1-ROADMAP.md`,
`.planning/milestones/v1.1-REQUIREMENTS.md`, plus the 34-09 PLAN/SUMMARY).
The slice does not run `git commit` — that move mirrors the v1.0 close
commit `48cbbc9` and is the operator's explicit action.

Next step: operator runs
`git add .planning/ && git commit -m 'docs(planning): close v1.1 ecosystem alignment milestone (audit PASS 5/5)' && git push origin main`,
then `/gsd-transition` to between-milestones state.
Resume file: .planning/v1.1-MILESTONE-AUDIT.md

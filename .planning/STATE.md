---
gsd_state_version: 1.0
milestone: v1.2
milestone_name: Core Capability Deepening
status: executing
stopped_at: Phase 35 (CC-1 budget) complete in 4 waves; v0.6.0 partial
last_updated: "2026-05-21T08:27:17.737Z"
last_activity: 2026-05-21 -- Phase 37 execution started
progress:
  total_phases: 4
  completed_phases: 2
  total_plans: 13
  completed_plans: 9
  percent: 50
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-05-21)

**Core value:** The core `llm-agent` module stays stdlib-only and zero-dep — anyone can `go get` it and read every line. Providers, telemetry, and reference services live in sister repos so users opt into deps one package at a time.
**Current focus:** Phase 37 — orchestrate-supervisor

## Current Position

Phase: 37 (orchestrate-supervisor) — EXECUTING
Milestone: **`v1.2` Core Capability Deepening — active, opened 2026-05-20.**
The first **core-feature** milestone since v0.3. Theme: **Core v0.6**
— capability additions to core `llm-agent`; memory tiering deferred to
v1.3 per KC-2. Core module bump: `v0.5.1 → v0.6.0` (minor — additive
only). 4 requirements (`CC-1..04`) across 4 phases (35-38):

- **Phase 35 — Budget / cancellation context (`CC-1`)** — `budget`
  package; ctx-keyed propagation + `budget.Tracker` enforcement;
  integration at the `generateFromPrompt` chokepoint; cost-table
  opt-in / outside core (KC-4).

- **Phase 36 — Policy / safety middleware (`CC-2`)** — `policy`
  package; capability-preserving `policy.Wrap(model) ChatModel`
  decorator mirroring `otelmodel.Wrap` (K3); 3 built-in regex gates
  (PII redaction, injection detection, max-input-length); documented
  composition stack `policy.Wrap(otelmodel.Wrap(provider))` (KC-3).

- **Phase 37 — Multi-agent coordination (`CC-3`)** —
  `orchestrate.Supervisor` shipped as a thin `StateGraph[S]` facade;
  iterative supervisor↔worker; honors **CC-1**'s budget + **CC-2**'s
  policy (KC-1).

- **Phase 38 — v1.2 milestone audit + close (`CC-4`)** — tag
  `llm-agent v0.6.0`; CHANGELOG entry; archive v1.2 ROADMAP/REQUIREMENTS
  to `.planning/milestones/`; ship `v1.2-MILESTONE-AUDIT.md`; refresh
  PROJECT/STATE/ROADMAP/REQUIREMENTS to between-milestones.

Keystone calls KC-1..KC-5 (research doc + v1.2 ROADMAP): Supervisor in
`orchestrate/` as a `StateGraph[S]` facade (KC-1); memory tiering OUT
of v1.2 scope, deferred to v1.3 with `ScopedMemory` decorator
pre-decided (KC-2); policy middleware as capability-preserving model
decorator (KC-3); budget ctx-keyed propagation + `Tracker` interface
enforcement with cost-table opt-in (KC-4); every new surface additive,
no `/v2` (KC-5). Core stdlib-only **preserved**.

Scope is the core repo only — `llm-agent-rag` stays a fixed point
(KS-5); sister repos stay on v0.2.x (the umbrella dep-currency gate
will fire when they bump core to v0.6.0, but that's a future
ecosystem-alignment task — not v1.2's job).

Previous milestone: `v1.1` Ecosystem Alignment — shipped and closed
2026-05-20 (audit PASS 5/5; `ECO-01..05`, `KE-1..KE-7`). Final
coordinated tag set: `llm-agent v0.5.1`, `llm-agent-rag v1.0.1`,
`llm-agent-otel v0.2.1`, `llm-agent-providers v0.2.1`,
`llm-agent-customer-support v0.2.2`.
Previous milestone (rag): `v1.0` API stabilization — shipped and closed
2026-05-21 (`llm-agent-rag v1.0.0`, audit PASS 6/6, fully archived).
Plan: 1 of 4
`.planning/research/v1.2-core-capability-deepening-SUMMARY.md`. 4
requirements (`CC-1..04`) across 4 phases (35-38). See ROADMAP /
REQUIREMENTS / phase-block above for the per-phase shape.

Status: Executing Phase 37
ROADMAP / REQUIREMENTS authored, PROJECT.md updated with v1.2 active
block + KC-1..KC-5 keystones table, STATE.md flipped from
between-milestones to v1.2 active. **No code/CI YAML changes yet** —
that begins in Phase 35.

Next step: run `/gsd-plan-phase 36` to plan the `policy` package
(CC-2 — capability-preserving model decorator with PII / injection /
max-input gates).
Last activity: 2026-05-21 -- Phase 37 execution started
`581caea`/`d141bf6`/`39950e2`/`535375f`); v0.6.0 partial cap cut
(also includes additive `agentstest` test-helper sub-package);
v1.2 cap retargeted to v0.7.0.

Progress: [███░░░░░░░░░░░] v1.2 Core Capability Deepening — 1/4 phases
complete (Phase 35 shipped, Phase 36 next). v1.1 shipped
(`llm-agent v0.5.1` + 4 sister tags), v1.0 (`llm-agent-rag v1.0.0`),
v0.9 (core `v0.6.0` legacy nomenclature), v0.8 (`v0.5.0`).

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
  auditable strict-equality exemption).

- 2026-05-20: `v1.2` Core Capability Deepening opened — the umbrella's
  first **core-feature** milestone since v0.3. Operator-chosen
  direction: take the core to **v0.6.0** with three additive
  capabilities (budget + policy + Supervisor); memory-tiering
  candidate **deferred to v1.3** per KC-2. Keystone calls KC-1..KC-5
  ratified: Supervisor as `StateGraph[S]` facade in `orchestrate/`
  (KC-1); memory tiering OUT of v1.2 (KC-2); policy middleware mirrors
  `otelmodel.Wrap` (KC-3); budget is ctx-keyed propagation +
  `Tracker` interface enforcement, cost-table opt-in (KC-4); every new
  surface additive, no `/v2` (KC-5). Core stdlib-only **preserved**:
  every new gate uses stdlib `regexp`; every new test uses
  `ScriptedLLM`; no edit to validated public types. Scope is the core
  repo only (sister repos stay on v0.2.x).

- 2026-05-21: **v0.6.0 cut as partial v1.2 cap** — contains Phase 35
  (CC-1 budget package, integrated at the `generateFromPrompt`
  chokepoint) plus the additive `agentstest` test-helper sub-package
  (stdlib-only, intended for sister-repo `*_test.go` consumption). Per
  semver, an additive new sub-package warrants a minor bump; v0.6.0
  was the natural name. **v1.2 milestone cap is retargeted from v0.6.0
  to v0.7.0** — Phase 36 lands on v0.6.1 (patch, additive `policy`
  sub-package), Phase 37 on v0.6.2 (patch, additive
  `orchestrate.Supervisor`), Phase 38 closes v1.2 with v0.7.0. KC-5
  (additive only, no `/v2`) preserved unchanged.

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
  Umbrella dep-currency CI gate shipped (commit `acb3253`). Milestone
  close was committed by the operator after the slice authored this
  STATE.md edit.

- v1.2 milestone artifacts laid down 2026-05-20; **Phase 35 shipped
  2026-05-21** in v0.6.0 partial cap. Next concrete action: operator
  runs `/gsd-plan-phase 36` (policy package, CC-2).

- Sister-repo dep-currency follow-up (post-v1.2): when v1.2 closes
  with `llm-agent v0.7.0`, the umbrella dep-currency gate will fail
  against the sister repos (`llm-agent-otel`, `llm-agent-providers`,
  `llm-agent-customer-support` all pin core `v0.5.1`). That's a future
  ecosystem-alignment milestone (v1.3?), **not v1.2's job**. Surfaced
  as a *known* follow-up, not a blocker. The gate will already fail
  against core v0.6.0 (sisters trail by one minor); a brief grace
  window during v1.2 mid-flight is acceptable.

- Memory tiering / scoping (session / project / user) — **deferred to
  v1.3** per KC-2. The reframed shape (`ScopedMemory` decorator +
  `memory.WithScope(ctx, scope)` ctx-keyed propagation) is pre-decided
  in the v1.2 research; v1.3 will implement without relitigating the
  design call.

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

- No immediate blocker — v1.1 is closed; v1.2 ROADMAP / REQUIREMENTS
  are laid down and ready to plan. The research
  (`.planning/research/v1.2-core-capability-deepening-SUMMARY.md`)
  confirmed all four candidate capabilities are stdlib-feasible at HIGH
  confidence; design proposals (the keystone calls) are MEDIUM
  confidence — first-slice prototypes will refine field-level details.

- KC-1 (Supervisor as `StateGraph[S]` facade) is MEDIUM confidence:
  the pattern is sound but the first-slice prototype (Phase 37-01) will
  refine whether StateGraph is the right substrate or if Supervisor
  needs its own state shape. Planner may go separate.

- KC-4 (ctx-keyed + `Tracker` hybrid) has a few unknowns at the
  streaming boundary — Phase 35-03 will pin down how partial-stream
  cancellation surfaces (`ctx.Err()` vs `budget.ErrExhausted`).

- Post-v1.0 discipline (`llm-agent-rag`): the API is frozen —
  additive-only within `v1.x`, breaking ⇒ `/v2`. v1.2 does not touch
  rag (KS-5).

- v1.2 stays under the breaking-change ceiling for core (KC-5): every
  new surface is in a new package or a new optional interface; no edit
  to validated public types; no `/v2` import path.

## Session Continuity

Last session: 2026-05-21
Stopped at: Phase 35 (CC-1 budget) complete in 4 waves; v0.6.0 partial
v1.2 cap cut (also includes additive `agentstest`); ready for
`/gsd-plan-phase 36` (policy package, CC-2). v1.2 cap retargeted to
v0.7.0.

v1.1 (Ecosystem Alignment) shipped and closed 2026-05-20 — audit PASS
5/5 (`ECO-01..05`, `KE-1..KE-7`); final coordinated tag set: `llm-agent
v0.5.1`, `llm-agent-rag v1.0.1`, `llm-agent-otel v0.2.1`,
`llm-agent-providers v0.2.1`, `llm-agent-customer-support v0.2.2`. One
auditable finding documented: the rag↔core cycle exemption in the
dep-currency gate (narrow strict-equality exemption — KE-2 holds).
Three architectural trade-offs documented honestly in the audit:
v1.0.0 → v1.0.1 freeze-day-after re-tag (chore-only patch);
topological-order miss in Phase 33's cascade (future cascades must
`tsort` against the dep DAG); the rag↔core cycle exemption itself.

**`v1.2` Core Capability Deepening milestone opened 2026-05-20** (this
slice). Theme: **Core v0.6** — capability additions to core
`llm-agent`; memory tiering deferred to v1.3 per KC-2. Core module
bump: `v0.5.1 → v0.6.0` (minor — additive only). Operator-chosen
direction: take the three picks (budget + policy + Supervisor) deeply
finished over four shallow. 4 phases (35-38), 4 requirements
(`CC-1..04`).

v1.2 scope:

- **Phase 35 — Budget / cancellation context (`CC-1`)**: `budget`
  package; ctx-keyed propagation via `budget.WithBudget(ctx, *Tracker)`

  + `budget.From(ctx)`; built-in `NewStrict`/`NewSoft` trackers;
  integration at the `generateFromPrompt` chokepoint. Cost-table is
  opt-in / outside core (KC-4).

- **Phase 36 — Policy / safety middleware (`CC-2`)**: `policy`
  package; capability-preserving `policy.Wrap(model) ChatModel`
  decorator mirroring `otelmodel.Wrap` (K3); typed `Gate` event union;
  3 built-in regex gates; documented stack
  `policy.Wrap(otelmodel.Wrap(provider))` (KC-3).

- **Phase 37 — Multi-agent coordination (`CC-3`)**:
  `orchestrate.Supervisor` shipped as a thin `StateGraph[S]` facade;
  honors **CC-1**'s budget + **CC-2**'s policy (KC-1).

- **Phase 38 — Milestone close (`CC-4`)**: tag `llm-agent v0.6.0`;
  CHANGELOG entry; archive `v1.2-*.md` to `.planning/milestones/`;
  audit; refresh planning artifacts to between-milestones.

**v1.1 closed (audit PASS 5/5, cycle-exemption finding documented);
v1.2 scope = Budget → Policy → Supervisor (4 phases); Phase 35
shipped 2026-05-21 in v0.6.0 partial cap; next concrete action =
`/gsd-plan-phase 36`; v1.2 milestone cap tag retargeted to v0.7.0;
memory tiering still deferred to v1.3 per operator decision
2026-05-20.**

Next step: operator runs `/gsd-plan-phase 36` to plan the `policy`
package (CC-2 — capability-preserving `policy.Wrap(model) ChatModel`
decorator with PII redaction, injection-detection, and max-input-length
gates).
Resume file: .planning/v1.2-ROADMAP.md

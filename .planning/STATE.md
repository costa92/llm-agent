---
gsd_state_version: 1.0
milestone: between-milestones
milestone_name: between-milestones (v1.2 closed 2026-05-25 with v0.7.0 + v1.3 brought-forward)
status: closed
stopped_at: v1.2 closed by v0.7.0 (combined v1.2 + v1.3-brought-forward)
last_updated: "2026-05-25"
last_activity: 2026-05-25 -- v0.7.0 cut; v1.2 milestone closed; v1.3 memory work brought forward (combined tag)
progress:
  total_phases: 4
  completed_phases: 4
  total_plans: 13
  completed_plans: 13
  percent: 100
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-05-25)

**Core value:** The core `llm-agent` module stays stdlib-only and zero-dep — anyone can `go get` it and read every line. Providers, telemetry, and reference services live in sister repos so users opt into deps one package at a time.
**Current focus:** between-milestones — v1.2 closed; next milestone not yet scoped.

## Current Position

Phase: none — between-milestones.
Milestone: **`v1.2` Core Capability Deepening — closed 2026-05-25 by `llm-agent v0.7.0`** (combined v1.2 close + v1.3 brought-forward).

## Recent activity (2026-05-25)

`v0.7.0` cut as **combined v1.2 close + v1.3 brought-forward**:

- Phase 35 (CC-1 budget) shipped 2026-05-21 in v0.6.0
- Phase 36 (CC-2 policy) shipped 2026-05-21 in v0.6.1
- Phase 37 (CC-3 `orchestrate.Supervisor`) shipped 2026-05-23 in v0.6.2
- Phase 38 (CC-4 close) + v1.3 brought-forward (4 memory PRs: #9 / #10
  / #11 / #12) shipped 2026-05-25 in v0.7.0

The v1.3 brought-forward was scoped significantly beyond the original
KC-2 plan (`ScopedMemory` decorator only) — it delivers a 5-pillar
ChatGPT-style memory feature suite. KC-2's "ScopedMemory" was reframed
to "ScopedManager" (Manager-level decorator) after discuss-phase
verification revealed `Manager.Consolidate` / `storeOf` bypass the
`Memory` interface, making a Memory-level decorator un-pluggable.

See `.planning/v1.2-MILESTONE-AUDIT.md` for the full audit + honest
deviation record.

### v0.7.0 verification gate (re-run at close)

- `GOWORK=off go vet ./...` — PASS
- `GOWORK=off go test ./... -count=1` — PASS (17/17 packages)
- `GOWORK=off go test -race -count=1 ./memory/...` — PASS (race-clean)
- `git diff v0.5.1 v0.7.0 -- go.mod go.sum` — dep set **strictly
  smaller** (rag back-edge dropped in v0.6.2 commit `6029565`); the
  core module now has zero `require` lines and an empty `go.sum`.
  Stdlib-only invariant strengthened across v1.2.

Status: v1.2 closed; ready for next-milestone scoping.

Previous milestone: `v1.1` Ecosystem Alignment — shipped and closed
2026-05-20 (audit PASS 5/5; `ECO-01..05`, `KE-1..KE-7`). Final
coordinated tag set: `llm-agent v0.5.1`, `llm-agent-rag v1.0.1`,
`llm-agent-otel v0.2.1`, `llm-agent-providers v0.2.1`,
`llm-agent-customer-support v0.2.2`.
Previous milestone (rag): `v1.0` API stabilization — shipped and closed
2026-05-21 (`llm-agent-rag v1.0.0`, audit PASS 6/6, fully archived).

Last activity: 2026-05-25 -- v0.7.0 cut; v1.2 closed combined with
v1.3 brought-forward.

Progress: [████████████████] v1.2 Core Capability Deepening — 4/4
phases complete. Tag set: v0.6.0 (Phase 35), v0.6.1 (Phase 36),
v0.6.2 (Phase 37), v0.7.0 (Phase 38 close + v1.3 brought-forward).

## Performance Metrics

**Velocity:**

- Total plans completed: 109 (40 through v0.5 + 14 in v0.6 phases 14-19 +
  8 in v0.7 phases 20-22 + 9 in v0.8 phases 23-25 + 6 in v0.9 phases 26-27
  + 9 in v1.0 phases 28-30 + 19 in v1.1 phases 31-34 + 13 in v1.2 phases
  35-38; v1.3 memory work brought forward as 4 standalone PRs without
  formal phase scoping).

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
| 35 | 4 | complete |
| 36 | 3 | complete |
| 37 | 3 | complete |
| 38 | 3 | complete |

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- 2026-05-15: new non-stdlib deps are allowed in `llm-agent-rag` only,
  isolated behind a subpackage/build tag like `postgres`. Core
  `llm-agent` stays stdlib-only.

- 2026-05-21: `v0.6.0` cut (CC-1 budget + `agentstest`); v1.2 cap
  retargeted from v0.6.0 to v0.7.0 (additive sub-package pattern; each
  v1.2 phase rolls a patch).

- 2026-05-25: `v1.2` closed by `v0.7.0` — combined release: v1.2 close
  (Phase 38, CC-4) **plus** v1.3 memory work brought forward
  (5-pillar ChatGPT-style memory across 4 PRs: #9 profile metadata,
  #10 Scope + ScopedManager (KC-2 reframe — Manager-level decorator),
  #11 Lister + Sanitizer + 5 tool actions, #12 persistence
  (Snapshot/Restore + FilesystemStore + 2 tool actions)). Operator
  decision (response "f2", 2026-05-25): accept the merge, reframe
  v0.7.0 as the combined tag. KC-5 (additive-only, no `/v2`)
  preserved. memory package: 38 → 139 tests (+101) across PRs #9 /
  #10 / #11 / #12; race-clean.

### Pending Todos

- **Sister-repo dep-currency follow-up:** cross-repo-build CI red —
  `llm-agent-otel` / `llm-agent-providers` / `llm-agent-customer-support`
  all pin core `v0.5.1` (2 minors behind even before v0.7.0). The
  umbrella `dep-currency-check.sh` continues to fail. Recommend a
  future ecosystem-alignment milestone (v1.4 candidate?) to bump all
  sister repos to pin `llm-agent v0.7.0`. **Known carry-forward, not a
  v0.7.0 blocker.**

- **v1.4 / next-milestone scoping:** not started; planning artifacts not
  yet authored. v1.3 memory work was brought forward into v0.7.0
  without formal ROADMAP / REQUIREMENTS scoping artifacts — future
  memory-extension milestone should re-establish formal scoping even
  though core impl is already shipped.

- **Memory v0.7 scope-awareness limitation:** `Manager.Consolidate`,
  `Manager.Forget`, and `Manager.StatsAll` on `ScopedManager` pass
  through to the inner Manager and operate on all stored items
  regardless of scope. These methods bypass the `Memory` abstraction
  to access the underlying store directly. Documented in
  `memory/doc.go` and `CHANGELOG.md`; scope-aware variants are
  deferred to a future memory-extension release.

- Live-Postgres CI wiring (testcontainers-go or GH Actions services) —
  carried forward from v0.5; the Phase 14 `tsvector` path, the Phase 21
  `postgres` graph path, and the v0.8 `postgres`
  `_communities`/`_community_reports` paths are all unverified against
  a live DB.

- Memory tiering / scoping (session / project / user) — **brought
  forward into v0.7.0**; the v1.3 plan to ship via `ScopedMemory`
  decorator was *reframed at discuss-phase* to `ScopedManager`
  (Manager-level decorator) because executor verification surfaced
  that `Manager.Consolidate` / `storeOf` bypass the `Memory`
  interface, making Memory-level decoration un-pluggable for the
  ChatGPT-style scope use case.

- v0.7, v0.8, v0.9, and v1.0 milestone-closes are all fully complete —
  committed, tagged, pushed, transitioned, and archived. The core
  repo `.planning/` tree of the v1.0 close is committed (`48cbbc9`)
  and pushed.

- v1.1 fully shipped and closed (audit PASS 5/5, 2026-05-20). All 5
  coordinated tags pushed (final post-cascade set: `llm-agent v0.5.1`,
  `llm-agent-rag v1.0.1`, `llm-agent-otel v0.2.1`,
  `llm-agent-providers v0.2.1`, `llm-agent-customer-support v0.2.2`).
  Umbrella dep-currency CI gate shipped (commit `acb3253`). Milestone
  close was committed by the operator after the slice authored that
  STATE.md edit.

- Operator-authorized environment changes during Phase 33:
  `llm-agent-rag` repo visibility flipped to **public** to unblock
  cross-repo CI on the public sister repos. Aligned with v1.0's
  `docs/compatibility.md` public-SDK framing.

- Incremental community maintenance is deferred (v0.9 keystone KG4-5).

- Environment note: `git config --global url."git@github.com:".insteadOf
  "https://github.com/"` was set (operator-authorized) so `go mod`
  fetches the private `github.com/costa92/*` modules over SSH. It
  persists on this machine; harmless (mirrors the pre-existing
  `code.hellotalk.com` rewrite).

### Blockers/Concerns

- No immediate blocker — v1.2 is closed; next-milestone scoping is open
  for operator direction. Memory v0.7-scope-awareness limitation and
  the sister-repo dep-currency staleness are recorded follow-ups, not
  blockers.

- Post-v1.0 discipline (`llm-agent-rag`): the API is frozen —
  additive-only within `v1.x`, breaking ⇒ `/v2`. v1.2 did not touch
  rag (KS-5).

- v1.2 stayed under the breaking-change ceiling for core (KC-5): every
  new surface is in a new package or a new optional interface; no edit
  to validated public types; no `/v2` import path. The v1.3
  brought-forward memory work honors KC-5 verbatim (zero changes to
  the `Memory` interface, `MemoryItem` fields, or `Manager` method
  signatures).

## Session Continuity

Last session: 2026-05-25
Stopped at: v1.2 closed by v0.7.0 (combined v1.2-close + v1.3-brought-
forward); audit + STATE + PROJECT artifacts authored on the
`release/v0.7.0` branch.

**`v1.2` Core Capability Deepening shipped and closed 2026-05-25** —
audit PASS 4/4 (`CC-01..04`, `KC-1..KC-5` with KC-2 reframed). Tag
sequence: `v0.6.0` (Phase 35, CC-1 budget), `v0.6.1` (Phase 36, CC-2
policy), `v0.6.2` (Phase 37, CC-3 `orchestrate.Supervisor`), `v0.7.0`
(Phase 38, CC-4 close + v1.3 memory brought forward).

v1.3 brought-forward shipped 4 memory PRs as the combined-tag content:

- PR #9 — profile metadata (`Source` / `Category` / `Pin` / `Disable`
  helpers + `SavedBoost`)
- PR #10 — `Scope` + `ScopedManager` (KC-2 reframe; Manager-level
  decorator)
- PR #11 — `Lister` + `Sanitizer` + 5 tool actions (`list`, `pin`,
  `unpin`, `disable`, `enable`)
- PR #12 — persistence (`Snapshot` / `Exporter` / `Importer` /
  `SnapshotStore` / `FilesystemStore` + `RestoreWorking` /
  `RestoreEpisodic` / `RestoreSemantic` + `Manager.ExportAll` /
  `ImportAll` + 2 tool actions `export`, `import`)

**v1.1 closed (audit PASS 5/5); v1.2 closed by v0.7.0 (audit PASS 4/4
with KC-2 reframe to ScopedManager); v1.3 memory work brought forward
into v0.7.0 as the combined tag content. Next concrete action: operator
chooses next-milestone direction (v1.4 candidate: sister-repo
dep-currency follow-up + bump to pin v0.7.0).**

Next step: operator scopes the next milestone (e.g. v1.4 sister-repo
dep-currency follow-up or a new core-feature direction).
Resume file: .planning/v1.2-MILESTONE-AUDIT.md (for the v1.2 close
record) + .planning/PROJECT.md (for the rolling project context).

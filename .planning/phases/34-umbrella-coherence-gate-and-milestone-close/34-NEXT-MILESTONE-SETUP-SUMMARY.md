# 34-NEXT-MILESTONE-SETUP — Summary

**Authored:** 2026-05-20
**Slice scope:** lay down the v1.2 milestone planning artifacts after the
operator picked **v1.2 — Core Capability Deepening (theme: Core v0.6)**
with the memory-tiering candidate deferred to v1.3. Pure `.planning/`
tree writes; no code, no CI YAML, no `git commit`.

## Artifacts created

- `.planning/v1.2-ROADMAP.md` — **251 lines** — Phases 35-38, KC-1..KC-5
  ratified, carry-forward debt restated verbatim from
  `.planning/v1.1-MILESTONE-AUDIT.md` plus the new memory-tiering
  deferral row.
- `.planning/v1.2-REQUIREMENTS.md` — **153 lines** — 4 CC-* with
  Pending checkboxes; Out-of-Scope table; traceability table 4/4
  mapped.

## Names ratified

- **Requirements:** `CC-1` (Budget / cancellation context, Phase 35),
  `CC-2` (Policy middleware, Phase 36), `CC-3` (Multi-agent
  coordination — `orchestrate.Supervisor`, Phase 37), `CC-4` (v1.2
  milestone audit + close, Phase 38).
- **Keystones:** `KC-1` (Supervisor as `StateGraph[S]` facade in
  `orchestrate/`), `KC-2` (memory tiering OUT of v1.2 — deferred to
  v1.3 with `ScopedMemory` decorator shape pre-decided), `KC-3` (policy
  middleware mirrors `otelmodel.Wrap` decorator), `KC-4` (budget
  ctx-keyed propagation + `Tracker` enforcement; cost-table opt-in /
  outside core), `KC-5` (every new surface additive — no `/v2`, no edit
  to validated public types).

## ROADMAP / REQUIREMENTS stubs

Phases 35-38 stubbed into `.planning/ROADMAP.md` `## Milestone v1.2`
section with status `not started` and per-phase goal + planned-work
bullets. v1.1 section moved into the archived list. v1.2 4 CC-* rows
added to `.planning/REQUIREMENTS.md` Traceability table; v1.1 ECO-*
rows preserved under "Archived Requirements".

## Memory tiering deferral

Documented in:

- `v1.2-ROADMAP.md` "Known Carry-forward Debt" — explicit row citing
  KC-2 and the pre-decided `ScopedMemory` decorator + ctx-keyed
  propagation shape.
- `v1.2-REQUIREMENTS.md` "Out of Scope" — top row citing KC-2.
- `.planning/ROADMAP.md` "Known Carry-forward Debt" — same row added.
- `.planning/PROJECT.md` v1.2 active block — explicit "memory tiering
  deferred to v1.3 per KC-2".

## Live planning artifacts updated

1. `.planning/PROJECT.md` — v1.2 active block added to Current State,
   Active Milestone Goals replaced (between-milestones → v1.2 active),
   KC-1..KC-5 keystone table appended to Key Decisions.
2. `.planning/STATE.md` — frontmatter flipped (`status: active`,
   `milestone: v1.2`, `stopped_at` updated, progress 0/4); Current
   Position rewritten for v1.2; Decisions / Pending Todos / Blockers
   refreshed; Session Continuity block records v1.1 closed + v1.2
   scope + next concrete action (`/gsd-plan-phase 35`).
3. `.planning/ROADMAP.md` — v1.2 `## Milestone v1.2` section added
   above the v1.1 (archived) section; v1.1 added to archived list;
   carry-forward debt updated.
4. `.planning/REQUIREMENTS.md` — rewritten to v1.2 (CC-1..CC-4
   Pending); v1.1 ECO-* archived block preserved; traceability table
   carries both.

## Verify gates (all green)

```
ROADMAP-OK    REQS-OK    PROJECT-OK    STATE-OK    REQS-LIVE-OK
ROADMAP-LIVE-OK    Leak count: 0 (no *.go / go.mod / go.sum / .github/ changes)
```

## Deviations

None — slice executed exactly as specified.

## Operator gate

Commit the v1.2 setup before `/gsd-plan-phase 35` fires:

```
git add .planning/ && git commit -m 'docs(planning): open v1.2 core capability deepening milestone' && git push origin main
```

Then run `/gsd-plan-phase 35` (Budget / cancellation context).

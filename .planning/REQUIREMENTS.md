# Requirements: llm-agent — active milestone (v1.2 Core Capability Deepening)

**Defined:** 2026-05-20
**Core Value:** the core `llm-agent` module stays stdlib-only and zero-dep
— anyone can `go get` it and read every line. v1.2's three new
capabilities (`budget`, `policy`, `orchestrate.Supervisor`) all ship inside
the core module and **preserve that property**: every new gate uses
stdlib `regexp`, every new test uses the existing `ScriptedLLM` mock, and
any cost-table required by `Budget.Cost` enforcement is **outside core**
(user-supplied `CostMapper` or a future providers sister-repo capability).
The four validated public types (`llm.ChatModel`, `agents.Agent`,
`memory.Memory`, `orchestrate.NodeFunc[S]`) are not edited.

## Milestone Scope

v1.2 is an **additive feature milestone** — the first **core-feature**
milestone since v0.3 shipped. Theme: **Core v0.6** — capability additions
to core `llm-agent`; memory tiering deferred to v1.3 per KC-2. Three new
**agent-runtime governance primitives** — budget/cancellation context,
policy/safety middleware, and a `Supervisor` multi-agent coordinator —
land without touching the validated public types. Core module bumps
**`v0.5.1 → v0.6.0`** (minor, additive only); **no `/v2` import path** —
v1.2 stays under the breaking-change ceiling.

Reference: `.planning/research/v1.2-core-capability-deepening-SUMMARY.md`
— the 4-candidate sweep, current-state audit, and the KC-1..KC-5
keystone decisions this milestone ratifies.

## v1.2 Requirements

### Budget / cancellation context

- [ ] **CC-1**: a `budget` package is shipped in core with
      `budget.WithBudget(ctx, *Tracker) context.Context`,
      `budget.From(ctx) *Tracker` (safe no-op on absent budget),
      `type Tracker interface{ Charge(Usage) error; Remaining() Budget }`,
      built-in trackers `budget.NewStrict` and `budget.NewSoft`, and is
      integrated at the `generateFromPrompt` chokepoint so every existing
      agent paradigm (Simple/ReAct/Reflection/PlanSolve/FunctionCall)
      honors it with zero behavior change when no budget is set. Cost is
      **opt-in / outside core** — core ships `Budget.Cost float64`
      plumbing only; no provider→$ table in core. **The core stays
      stdlib-only** (`go list -deps ./...` lists zero third-party modules;
      no edit to `llm.ChatModel` or `agents.Agent` — KC-5). Phase 35.

### Policy / safety middleware

- [ ] **CC-2**: a `policy` package is shipped in core with a
      capability-preserving `policy.Wrap(inner llm.ChatModel, gates
      ...Gate) llm.ChatModel` decorator that mirrors `otelmodel.Wrap`
      (K3 — handles `ToolCaller` / `Embedder` / `StructuredOutputs`
      assertions), a typed `Gate` event union (`PreGenerate`/
      `PostGenerate`/`PreStream`/`StreamDelta`/`PostStream`), a sentinel
      `policy.ErrBlocked`, and 3 built-in gates (`PIIRedactor`,
      `InjectionScanner`, `MaxInputLen`). The documented composition
      stack `policy.Wrap(otelmodel.Wrap(provider))` is verified by an
      integration test. **The core stays stdlib-only** — every built-in
      gate uses stdlib `regexp`; no rag import (regex patterns lifted
      to a separate file — KC-3). Phase 36.

### Multi-agent coordination

- [ ] **CC-3**: an `orchestrate.Supervisor` primitive is shipped as a
      thin facade over `StateGraph[S]` (per KC-1) with `NewSupervisor`,
      `SupervisorOptions{ Planner, Workers, MaxRounds, ParseDispatch,
      BuildAggregate }`, where workers are `agents.Agent` (composable —
      a Supervisor can supervise another Supervisor). The Supervisor
      honors **CC-1**'s ctx-keyed budget (rounds count against
      `Budget.Calls`; ctx propagates to workers) and supports policy
      attachment via **CC-2** (documented pattern for policy-wrapping a
      worker's underlying model). A `compose-with-StateGraph` test
      proves the facade works both directions. **The core stays
      stdlib-only**; no edit to `agents.Agent` or
      `orchestrate.NodeFunc[S]` (KC-5). Phase 37.

### Milestone close

- [ ] **CC-4**: the v1.2 milestone-close gate is green —
      `.planning/v1.2-MILESTONE-AUDIT.md` ships verdict ✅ PASS, the
      core `llm-agent` is tagged `v0.6.0` from `main` and pushed, the
      `CHANGELOG.md` records the additive feature set (`budget`,
      `policy`, `orchestrate.Supervisor`), and the v1.2 ROADMAP /
      REQUIREMENTS are archived to `.planning/milestones/v1.2-*.md`.
      The active planning artifacts (`PROJECT.md`, `STATE.md`,
      `ROADMAP.md`, `REQUIREMENTS.md`) are refreshed to
      between-milestones. **The core remains stdlib-only post-tag**
      (`go list -deps ./...` proof in audit). Phase 38.

## Archived Requirements

### v1.1 Ecosystem Alignment (5/5 Done — shipped 2026-05-20)

- [x] **ECO-01** — core `llm-agent` RAG facade repaired against
      `llm-agent-rag v1.0.0`; the 7 facade-test failures fixed inside
      the facade adapters; core proven stdlib-only. Phase 31.
- [x] **ECO-02** — every sister repo's `main` reflects reality —
      stranded branches merged, stale branches pruned. Phase 32.
- [x] **ECO-03** — coordinated dependency-ordered re-tag wave shipped
      — final post-cascade tags: `llm-agent v0.5.1`,
      `llm-agent-rag v1.0.1`, `llm-agent-otel v0.2.1`,
      `llm-agent-providers v0.2.1`, `llm-agent-customer-support v0.2.2`.
      Zero `replace` directives. Phase 33.
- [x] **ECO-04** — umbrella dependency-currency CI gate shipped
      (`scripts/dep-currency-check.sh` + `.github/workflows/umbrella.yml`).
      Phase 34.
- [x] **ECO-05** — full 5-repo coherence verification PASS
      (`34-08-RESULTS.md`); milestone audited
      (`.planning/v1.1-MILESTONE-AUDIT.md`). Phase 34.

Archive: `.planning/milestones/v1.1-REQUIREMENTS.md`.

## Out of Scope

| Feature | Reason |
|---------|--------|
| Memory tiering / scoping (session / project / user) | **Deferred to v1.3 per KC-2.** The reframed shape (`ScopedMemory` decorator + ctx-keyed scope) is pre-decided in the v1.2 research. |
| Anything in sister repos (`llm-agent-providers`, `llm-agent-otel`, `llm-agent-customer-support`) | v1.2 is a **core-only** milestone. Sister repos stay on v0.2.x; the umbrella dep-currency gate will fire when they bump core to v0.6.0 — future ecosystem-alignment milestone. |
| Anything in `llm-agent-rag` | Frozen v1.0.x track (KS-5 compatibility promise). v1.2 does not touch rag. |
| Live-Postgres CI wiring | Carried-forward debt (KE-7); a CI capability project of independent size. Stays deferred. |
| `llm-agent-rag` deployment layer (HTTP service, CLI, caching) | A feature milestone deferred since v0.6. |
| Productionizing the `customer-support` demo | Intentionally demo-grade (PROJECT.md "Known Tech Debt"). |
| K8s / Helm packaging | Standing hard-rule non-goal. |
| Any non-stdlib dependency in core | CLAUDE.md Rule 1 — absolute. New gates use stdlib `regexp`; new tests use `ScriptedLLM`. |
| Cost-table (provider → $/token) | KC-4 corollary — core ships `Budget.Cost` plumbing; cost-table is **outside core** (user-supplied `CostMapper`). |
| OWASP / NIST safety category framework in `policy` | v1.2 ships 3 built-in gates; full taxonomy is a future expansion. |
| `replace` directives in tagged-release branches | CLAUDE.md Rule 3 — absolute. |
| Any new `/v2` import path | KC-5 — v1.2 stays under the breaking-change ceiling. |
| Per-sister-repo dependency-currency gate | v1.1 KE-1 carry-forward — broadening the gate is a v1.3-style ecosystem-alignment task. |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| CC-1 | Phase 35 | Pending |
| CC-2 | Phase 36 | Pending |
| CC-3 | Phase 37 | Pending |
| CC-4 | Phase 38 | Pending |
| ECO-01 | Phase 31 | Done (v1.1) |
| ECO-02 | Phase 32 | Done (v1.1) |
| ECO-03 | Phase 33 | Done (v1.1) |
| ECO-04 | Phase 34 | Done (v1.1) |
| ECO-05 | Phase 34 | Done (v1.1) |

**Coverage:**
- v1.2 requirements: 4 total
- Mapped to phases: 4
- Unmapped: 0

---
*Requirements defined: 2026-05-20 — v1.2 Core Capability Deepening
milestone, scoped from
`.planning/research/v1.2-core-capability-deepening-SUMMARY.md`. Previous
milestone v1.1 archived to `.planning/milestones/v1.1-REQUIREMENTS.md`.*

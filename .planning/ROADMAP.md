# Roadmap: llm-agent

**Last updated:** 2026-05-12
**Current state:** `v0.3` shipped and archived
**Active scope:** no active implementation milestone; only future gated work
remains

## Archived Milestones

- [x] **v0.3: Deployable multi-repo release** — shipped 2026-05-12.
  Delivered `llm/v2`, three real provider adapters, OTel wrappers, and the
  customer-support demo stack across 4 repos.
  - Archive: `.planning/milestones/v0.3-ROADMAP.md`
  - Requirements archive: `.planning/milestones/v0.3-REQUIREMENTS.md`
  - Audit: `.planning/v0.3-MILESTONE-AUDIT.md`

## Active Forward Work

### Phase 7: Deprecation removal & `v0.4` cut

**Status**: calendar-gated, not started

**Goal**: Honor the deprecation window promised in `v0.3` by removing the old
`llm.Client` surface only after a full minor cycle has elapsed.

**Depends on**:

- `v0.3` shipped and archived
- a real deprecation window for downstream users
- coordinated cross-repo release timing

**Repos**: `llm-agent`, `llm-agent-providers`, `llm-agent-otel`,
`llm-agent-customer-support`

**Requirements carried forward**: `DEPRC-01`, `DEPRC-02`, `DEPRC-03`,
`DEPRC-04`

**Planned work**:

- `07-01` Audit zero remaining internal `llm.Client` usage
- `07-02` Remove deprecated v0.2 client/types and document the breaking change
- `07-03` Bump sister repos to `llm-agent v0.4.x` and coordinate tags

**Gate**:

Do not start this phase just because implementation capacity is available.
This work begins only after the post-`v0.3` deprecation window is intentionally
opened.

## Next Milestone Setup

- `v0.4` requirements have not been defined yet.
- When the gate opens, start with a fresh `.planning/REQUIREMENTS.md` and
  expand the roadmap from there.
- Any non-gated new feature work should be introduced as a new milestone,
  not smuggled into Phase 7.

## Known Carry-forward Debt

- Formal verification artifacts are still uneven after Phase 0.
- Nyquist validation artifacts are still missing for Phases 2-6.
- The refsvc demo remains intentionally demo-grade in observability fidelity
  and packaging.

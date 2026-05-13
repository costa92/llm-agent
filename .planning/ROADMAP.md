# Roadmap: llm-agent

**Last updated:** 2026-05-13
**Current state:** `v0.3` shipped and archived; Phase 7 manually opened
**Active scope:** Phase 7 deprecation-removal cycle (`v0.4` cut)

## Archived Milestones

- [x] **v0.3: Deployable multi-repo release** — shipped 2026-05-12.
  Delivered `llm/v2`, three real provider adapters, OTel wrappers, and the
  customer-support demo stack across 4 repos.
  - Archive: `.planning/milestones/v0.3-ROADMAP.md`
  - Requirements archive: `.planning/milestones/v0.3-REQUIREMENTS.md`
  - Audit: `.planning/v0.3-MILESTONE-AUDIT.md`

## Active Forward Work

### Phase 7: Deprecation removal & `v0.4` cut

**Status**: active by explicit early gate override on 2026-05-12

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
- `07-02` Migrate runtime packages off the deprecated surface
- `07-03` Delete the compatibility layer and rewrite core docs/examples
- `07-04` Verify sister repos against the removed-compatibility core API
- `07-05` Bump sister repos to the final `llm-agent v0.4.x` tag and coordinate release tags

**Gate**:

The original calendar gate was explicitly overridden on 2026-05-12 by operator
instruction. Phase 7 is now active, but its scope remains constrained to
`DEPRC-01..04` only.

## Next Milestone Setup

- `v0.4` is now the active deprecation-removal cycle.
- Any non-deprecation feature work should still be introduced as a separate
  milestone, not smuggled into Phase 7.

## Known Carry-forward Debt

- Formal verification artifacts are still uneven after Phase 0.
- The refsvc demo remains intentionally demo-grade in observability fidelity
  and packaging.

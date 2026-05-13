# Phase 7: Deprecation Removal & v0.4 Cut - Context

**Gathered:** 2026-05-12  
**Status:** Active by explicit early gate override

<domain>
## Phase Boundary

Phase 7 removes the deprecated v0.2 `llm.Client` surface from the core repo
and coordinates the resulting `v0.4` cut across the umbrella repos.

This phase covers:

- internal audit of all remaining legacy-surface usage
- core symbol removal from `llm/legacy.go`
- migration of any remaining internal packages/tests/examples to `llm.ChatModel`
- breaking-change documentation
- coordinated sister-repo version bump and tagging

Phase 7 does NOT cover:

- new provider capabilities
- new agent paradigms
- observability feature expansion
- demo hardening beyond compatibility updates needed for `v0.4`

</domain>

<decisions>
## Implementation Decisions

### D-01: The operator override opens the gate but not the scope

- The original roadmap treated Phase 7 as calendar-gated.
- On 2026-05-12, that gate was explicitly opened by operator instruction.
- This override authorizes deprecation-removal work only.

### D-02: Audit before deletion

- `07-01` must enumerate every remaining internal legacy-surface usage before
  any symbol deletion lands.
- The audit output becomes the dependency map for `07-02` and `07-03`.

### D-03: Core repo stays stdlib-only

- Removing the legacy surface must not weaken the core repo's zero-dependency
  invariant.
- Migrations should target existing `llm.ChatModel` / capability seams rather
  than new abstraction layers.

### D-04: Cross-repo release remains coordinated

- Even though the gate opened early, the `v0.4` bump still needs coordinated
  updates in sister repos before tags are cut.

</decisions>

<canonical_refs>
## Canonical References

- `.planning/ROADMAP.md`
- `.planning/REQUIREMENTS.md`
- `.planning/STATE.md`
- `.planning/milestones/v0.3-REQUIREMENTS.md`
- `DEPRECATIONS.md`
- `docs/migration-v0.2-to-v0.3.md`
- `llm/legacy.go`

</canonical_refs>

<specifics>
## Initial Audit Findings

The first repo scan on 2026-05-12 shows legacy-surface usage still present in:

- `rag/`
- `bench/`
- `context/`
- `rl/`
- examples and example tests
- docs snapshots and migration docs
- the deprecated symbol definitions themselves in `llm/legacy.go`

This confirms `DEPRC-01` is not yet satisfied and the phase should begin with
an explicit audit artifact and plan rather than direct deletion.

</specifics>


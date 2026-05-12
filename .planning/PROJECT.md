# llm-agent

## What This Is

`llm-agent` is a stdlib-only Go framework for building LLM-driven agents.
At `v0.3`, the project now spans four coordinated repos:

- `llm-agent` keeps the zero-dependency core, agent paradigms, memory, RAG,
  and the new `llm/v2` capability surface.
- `llm-agent-providers` ships real OpenAI, Anthropic, and Ollama adapters.
- `llm-agent-otel` ships capability-preserving OpenTelemetry wrappers.
- `llm-agent-customer-support` ships a demo customer-support service that ties
  the stack together.

`v0.3` is now shipped and archived. The active roadmap is no longer "build the
deployable stack"; it is "hold the line on the deprecation window, then define
the next milestone cleanly."

## Core Value

**The core `llm-agent` module stays stdlib-only and zero-dep.** Providers,
telemetry, and reference services remain opt-in sister repos so the primary
module stays readable, portable, and cheap to adopt.

## Current State

- `v0.3` shipped on 2026-05-12 and is archived in
  `.planning/milestones/v0.3-ROADMAP.md`.
- The shipped stack includes real Generate, Stream, Tool, and Embedding paths
  across the targeted provider set.
- OpenTelemetry wrappers and the reference customer-support service are part of
  the released milestone state.
- Phase 7 is intentionally not started because it is calendar-gated
  post-`v0.3` work.

## Requirements

### Validated

- ✓ The core repo still builds as a stdlib-only module.
- ✓ `llm/v2` capability negotiation is live in the core repo.
- ✓ Three real provider adapters exist in sister repos.
- ✓ Capability-preserving OTel wrappers exist in a sister repo.
- ✓ A runnable customer-support demo service exists in a sister repo.

### Active

- None. `v0.3` scope has been archived and no new milestone requirements are
  active yet.

### Out of Scope

- Kubernetes packaging is still out of scope until a future milestone plans it
  explicitly.
- Multimodal/vision support is still out of scope.
- A v1.0 stability promise is still out of scope pending real-world feedback.

## Next Milestone Goals

- Honor the `llm.Client` deprecation window before starting Phase 7.
- Decide whether the next planned work is the gated deprecation-removal cycle
  or a distinct feature milestone.
- Raise archive quality by strengthening milestone-close verification quality
  beyond the newly backfilled validation artifacts.

## Known Tech Debt

- Formal `*-VERIFICATION.md` coverage is uneven after Phase 0.
- The refsvc observability demo is intentionally demo-grade rather than
  production-billing-grade.

## Operational Follow-ups

- Manual GitHub branch protection still needs to be enabled on the sister
  repos.
- The first post-merge `nightly-ollama-live` workflow smoke test should still
  be observed once changes are pushed.

## Archived Milestone Definition

<details>
<summary>v0.3 milestone snapshot</summary>

`v0.3` was the "library you can deploy" milestone:

- add real OpenAI, Anthropic, and Ollama integrations
- extend the core contract to capability-based `llm/v2`
- add OpenTelemetry observability
- ship a `docker compose` customer-support reference stack

Archive references:

- Roadmap: `.planning/milestones/v0.3-ROADMAP.md`
- Requirements: `.planning/milestones/v0.3-REQUIREMENTS.md`
- Audit: `.planning/v0.3-MILESTONE-AUDIT.md`

</details>

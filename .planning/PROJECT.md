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

`v0.3` is now shipped and archived. As of 2026-05-12, the Phase 7 gate has
been opened early by explicit operator decision, so the active roadmap is now
"execute the deprecation-removal cycle cleanly and cut `v0.4` without dragging
new feature scope into it."

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
- Phase 7 has been explicitly opened early on 2026-05-12 by operator override
  despite the original calendar gate.
- As of 2026-05-12, the core repo itself has completed the compatibility
  removal: runtime packages use `llm.ChatModel`, `llm/legacy.go` is gone, and
  only cross-repo release coordination remains.
- As of 2026-05-13, local workspace verification against `/tmp` sibling repos
  shows that providers, OTel wrappers, and the reference service all already
  pass against the removed-compatibility core without source changes.
- As of 2026-05-13, the core `v0.4.0` tag exists remotely, all three sister
  repos have their `go.mod` files on `github.com/costa92/llm-agent v0.4.0`,
  and coordinated sister-repo release tagging is the final Phase 7 closeout
  action.

## Requirements

### Validated

- ✓ The core repo still builds as a stdlib-only module.
- ✓ `llm/v2` capability negotiation is live in the core repo.
- ✓ Three real provider adapters exist in sister repos.
- ✓ Capability-preserving OTel wrappers exist in a sister repo.
- ✓ A runnable customer-support demo service exists in a sister repo.

### Active

- `DEPRC-01`: Audit complete — zero internal users of `llm.Client` remain.
- `DEPRC-02`: `llm.Client` and v0.2-era types removed in `v0.4.0` core.
- `DEPRC-03`: CHANGELOG `### Breaking` documents the removal.
- `DEPRC-04`: Sister repos bumped to `llm-agent v0.4.0` and coordinated tags
  complete the release line.

### Out of Scope

- Kubernetes packaging is still out of scope until a future milestone plans it
  explicitly.
- Multimodal/vision support is still out of scope.
- A v1.0 stability promise is still out of scope pending real-world feedback.

## Next Milestone Goals

- Open the next milestone after the completed coordinated `v0.4` release.
- Raise archive quality by strengthening milestone-close verification quality
  beyond the newly backfilled validation artifacts.

## Known Tech Debt

- Formal `*-VERIFICATION.md` coverage is uneven after Phase 0.
- The refsvc observability demo is intentionally demo-grade rather than
  production-billing-grade.

## Operational Follow-ups

- Archive Phase 7 and open the next milestone when the next scoped feature set
  is ready.

## Key Decisions

- 2026-05-12: Phase 7 gate opened early by explicit operator instruction even
  though the original roadmap treated it as calendar-gated post-`v0.3` work.
  This locks the next active work to `DEPRC-01..04` only; no unrelated feature
  milestone is being opened in parallel.
- 2026-05-12: Phase 7 execution was split into three core-repo slices:
  `07-01` audit, `07-02` runtime migration, and `07-03` compatibility removal
  + documentation rewrite. Cross-repo coordination is deferred to `07-04`.
- 2026-05-13: a local 4-repo `go.work` audit proved that `llm-agent-providers`,
  `llm-agent-otel`, and `llm-agent-customer-support` already pass against the
  post-compat-removal core API with no source patches required.
- 2026-05-13: Phase 7 closeout verification confirmed that the released core
  `v0.4.0` tag resolves remotely, all sister repos pass `go test ./...`
  against the coordinated release line, and coordinated sister-repo tags can be
  cut from the already-landed `v0.4.0` bump commits.

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

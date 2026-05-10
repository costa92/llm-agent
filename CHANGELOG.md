# Changelog

All notable changes to `github.com/costa92/llm-agent` —
a standalone Go LLM agents framework module.

<!-- Keep a Changelog format: https://keepachangelog.com/en/1.1.0/ -->
<!-- Semver: https://semver.org/ -->
<!-- Sections per release: Added | Changed | Deprecated | Removed | Fixed | Security | Breaking -->
<!-- 0.x BC policy: minor/patch within a 0.x line are BC-compatible; 0.x→0.y (y>x) may break -->
<!-- Breaking changes: include "### Breaking" section + migration notes in the release entry -->

## [Unreleased]

Phase 0 of the v0.3 milestone — multi-repo infra + capability-aware `llm/` interfaces.
The new `llm/` types coexist with the v0.2 `llm.Client` surface (now renamed to
`llm.LegacyClient` with a `type Client = LegacyClient` alias for source compatibility).
Existing callers continue to compile unchanged.

### Added

- New capability-aware interfaces in `llm/`:
  - `llm.ChatModel` — base contract (`Generate` + `Stream` + `Info`)
  - `llm.ToolCaller` — capability for native function-calling (`WithTools`, immutable)
  - `llm.Embedder` — capability for vector embeddings (NOT embedding `ChatModel`)
  - `llm.StructuredOutputs` — capability for JSON-schema-constrained generation
- Typed streaming union in `llm/stream.go`:
  - `llm.StreamReader` — iterator-style streaming (`Next` + `Close`)
  - `llm.StreamEvent` + `llm.StreamEventKind` — typed union with `EventTextDelta`,
    `EventToolCallStart`, `EventToolCallArgsDelta`, `EventToolCallEnd`,
    `EventThinkingDelta`, `EventDone`
  - `llm.ToolCallDelta` — per-tool-call streaming state with stable `Index` field
  - `llm.AccumulateStream` — convenience for consumers that want a flat `Response`
- Per-(provider x model) identity:
  - `llm.ProviderInfo` — bound provider+model identity returned by `Info()`
  - `llm.Capabilities` — JSON-serializable feature struct (`Tools`, `Embeddings`,
    `StructuredOutputs`, `PromptCaching` bool fields)
- New chat-layer request/response types:
  - `llm.Request` (replaces `GenerateRequest`)
  - `llm.Response` (replaces `GenerateResponse`)
  - `llm.Vector` (`[]float32`)
  - `llm.Usage` + `llm.UsageSource` (`Reported` / `Estimated` / `Unknown`) for
    K4 three-state cost tracking
- New mocks (promoted from `_test.go`):
  - `llm.ScriptedLLM` — full-capability deterministic mock with functional options
    (`WithProvider`, `WithModel`, `WithCapabilities`, `WithResponses`,
    `WithEmbedDimensions`); helpers `TextResponse`, `ToolCallResponse`
  - `llm.ChatOnlyMock` — `ChatModel`-only mock for capability-degradation testing
- Sentinel errors:
  - `llm.ErrCapabilityNotSupported` — wrap with `fmt.Errorf("...: %w", ...)`
  - `llm.ErrScriptExhausted` — emitted by `ScriptedLLM` when the script runs out
- Migration guide at `docs/migration-v0.2-to-v0.3.md` with one worked Simple-paradigm
  example + a generic type-rename mapping table.
- `DEPRECATIONS.md` at repo root — single source of truth for symbol → target
  removal version; Phase 7 audits this file before the v0.4 cut (Pitfall 15).

### Deprecated

The following symbols are retained for v0.3.x source compatibility but will be
**removed in v0.4.0**. See [`docs/migration-v0.2-to-v0.3.md`](docs/migration-v0.2-to-v0.3.md)
for migration steps; full table in [`DEPRECATIONS.md`](DEPRECATIONS.md).

- `llm.Client` (interface) — now an alias for `llm.LegacyClient`. Use `llm.ChatModel`.
- `llm.LegacyClient` (interface) — renamed from `llm.Client`. Use `llm.ChatModel`.
- `llm.GenerateRequest` (struct) — use `llm.Request`.
- `llm.GenerateResponse` (struct) — use `llm.Response`.
- `llm.StreamChunk` (struct) — use `llm.StreamEvent` (typed union).
- `llm.StreamUsage` (struct) — use `llm.Usage` (with `Source` field).
- `agents.scriptedLLM` (root-package test helper) — use `llm.NewScriptedLLM`.
  Removed in Phase 3 (~v0.3.3) once agent paradigms migrate to `llm.ChatModel`.

### Versioning policy (INFRA-07)

**Versioning policy across 4 repos:** `llm-agent` v0.3.x core; sister repos start
at v0.1.x; CHANGELOG `### Breaking` per repo. See README §Versioning for the full
BC matrix.

The v0.3 milestone covers a 4-repo umbrella:

| Repo | v0.3 track | Notes |
|---|---|---|
| `github.com/costa92/llm-agent` (this repo) | `v0.3.x` | Stdlib-only core. Pre-release tag `v0.3.0-pre.1` cut at end of Phase 0; final `v0.3.0` after Phase 6. |
| `github.com/costa92/llm-agent-providers` | `v0.1.x` | Sister repo created in Phase 0; first content lands in Phase 1. |
| `github.com/costa92/llm-agent-otel` | `v0.1.x` | Sister repo created in Phase 0; first content lands in Phase 5. |
| `github.com/costa92/llm-agent-customer-support` | `v0.1.x` | Sister repo created in Phase 0; first content lands in Phase 6. |

- 0.x BC policy applies per repo: minor/patch within a 0.x line are BC-compatible;
  0.x→0.y (y>x) may break. Each repo declares breaking changes with a `### Breaking`
  section in its CHANGELOG.
- Sister repos pin `require github.com/costa92/llm-agent vX.Y.Z` per
  Phase / sister-repo release; coordinated tags during the v0.4 cut (Phase 7).
- `replace` directives are FORBIDDEN on any branch matching `release/**` —
  enforced by the `release-precheck` CI workflow in every repo.
- `go.work` is `.gitignore`d in every repo; CI runs with `GOWORK=off`.

## [v0.1.0] — 2026-04-28

Initial module release. Framework was implemented as 9 phases inside
the parent repo between 2026-04-27 and 2026-04-28; this release extracts it
into its own Go module so external users can `go get` it without pulling
the AICS main module's transitive dependencies (Kratos, GORM, Redis, etc.).

### Added

- Standalone Go module: `github.com/costa92/llm-agent`
- New `agents/llm` subpackage owning the LLM contract:
  `Client`, `GenerateRequest`, `GenerateResponse`, `Message`, `Tool`,
  `ToolCall`, `StreamChunk`, `StreamUsage`, `FinishReason` + 6 const
- 12 packages exposed:
  `agents` (root), `agents/llm`, `agents/builtin`, `agents/memory`,
  `agents/rag`, `agents/context`, `agents/comm`, `agents/comm/mcp`,
  `agents/comm/a2a`, `agents/comm/anp`, `agents/orchestrate`, `agents/rl`,
  `agents/bench`
- Stdlib-only — zero third-party Go dependencies

### Notes

- v0.1.0 is **学習 / 原型** — API may break between 0.x releases.
  Wait for **v1.0** for stability commitment.
- Source design spec:
  `docs/superpowers/specs/2026-04-27-pkg-llm-agents-design.md`
  (in parent AICS repo).
- Extraction design spec:
  `docs/superpowers/specs/2026-04-28-pkg-llm-agents-module-extraction-design.md`
  (in parent AICS repo).

## [v0.2.0] — 2026-05-08

Standalone repository extraction. The framework was developed inside the parent
AICS monorepo (`github.com/costa92/aics-core/pkg/llm/agents`) through Phase R;
this release lifts it into its own GitHub repository and Go module so the import
path is no longer nested.

### Changed (Breaking)

- **Module path** — `github.com/costa92/aics-core/pkg/llm/agents` →
  `github.com/costa92/llm-agent`. Subpackage paths follow the same flattening
  (e.g. `.../aics-core/pkg/llm/agents/llm` → `.../llm-agent/llm`).
  Callers must update import statements accordingly.

### Added

- `pkg/fanout` — concurrent task runner (`fanout.Task[T]` / `fanout.Run` /
  `WithFailFast`), copied from aics-core; previously a transitive dep, now a
  first-class subpackage.
- `internal/testenv` — test-only HTTP listen helper, copied from aics-core to
  keep this module zero-third-party.

### Fixed

- Updated `doc.go` portability contract — drops references to the old
  monorepo's `internal/*` and `pkg/*` constraints; now reads as a standalone
  stdlib-only module.

### Migration

```diff
- import "github.com/costa92/aics-core/pkg/llm/agents"
+ import "github.com/costa92/llm-agent"

- import "github.com/costa92/aics-core/pkg/llm/agents/llm"
+ import "github.com/costa92/llm-agent/llm"
```

```bash
go get github.com/costa92/llm-agent@v0.2.0
```

---

### Versioning

Tags on this repo are flat (`vX.Y.Z`) — no `<module-subpath>/` prefix.
0.x line: minor/patch are BC; 0.x → 0.y (y > x) may break.

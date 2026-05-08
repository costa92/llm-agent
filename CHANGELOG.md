# Changelog

All notable changes to `github.com/costa92/llm-agent` —
a standalone Go LLM agents framework module.

<!-- Keep a Changelog format: https://keepachangelog.com/en/1.1.0/ -->
<!-- Semver: https://semver.org/ -->
<!-- Sections per release: Added | Changed | Deprecated | Removed | Fixed | Security | Breaking -->
<!-- 0.x BC policy: minor/patch within a 0.x line are BC-compatible; 0.x→0.y (y>x) may break -->
<!-- Breaking changes: include "### Breaking" section + migration notes in the release entry -->

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

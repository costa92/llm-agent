# Changelog

All notable changes to `github.com/costa92/llm-agent` —
a standalone Go LLM agents framework module.

<!-- Keep a Changelog format: https://keepachangelog.com/en/1.1.0/ -->
<!-- Semver: https://semver.org/ -->
<!-- Sections per release: Added | Changed | Deprecated | Removed | Fixed | Security | Breaking -->
<!-- 0.x BC policy: minor/patch within a 0.x line are BC-compatible; 0.x→0.y (y>x) may break -->
<!-- Breaking changes: include "### Breaking" section + migration notes in the release entry -->

## [Unreleased]

### Added

- `memory`: ChatGPT-style profile metadata helpers layered on the
  existing `MemoryItem.Metadata` map. No changes to `MemoryItem`
  struct fields or the `Memory` interface — all state lives under a
  reserved `_`-prefixed key namespace (`_source`, `_category`,
  `_pinned`, `_disabled`; `_scope` reserved for a future PR).
  - Types: `Source` (with `SourceUserSaved`, `SourceAgentInferred`,
    `SourceSystem`, `SourceUnknown`); `Category` (with `CategoryUser`,
    `CategoryFeedback`, `CategoryProject`, `CategoryReference`).
  - Constructors: `NewSavedMemory(content, cat)` (Importance=0.9,
    Pinned=true, Source=SourceUserSaved); `NewInferredMemory(content,
    cat, confidence)` (confidence clamped to [0,1] → Importance,
    Source=SourceAgentInferred).
  - Accessors: `GetSource` / `SetSource`, `GetCategory` /
    `SetCategory`, `IsPinned` / `SetPinned`, `IsDisabled` /
    `SetDisabled`. Getters are zero-value safe on nil / missing /
    type-mismatched metadata; setters initialize `Metadata` when nil.
- `memory`: `WorkingOptions.SavedBoost`, `EpisodicOptions.SavedBoost`,
  `SemanticOptions.SavedBoost` — multiplicative score factor applied
  at `Search` time when the item is `IsPinned` or
  `GetSource(it) == SourceUserSaved`. The zero value (or any
  non-positive value) is a strict no-op, preserving pre-v0.7 scoring.

### Changed

- `memory`: `Search` across all three memory types now skips items
  flagged with `IsDisabled(it) == true`. Disabled items remain in
  storage (Get / Stats / Forget still see them); they are only hidden
  from query results.
- `memory`: `Manager.Forget` strategies (`ForgetByImportance`,
  `ForgetByAge`, `ForgetByCapacity`) now skip items flagged with
  `IsPinned(it) == true`. Pinned items are excluded from the
  candidate set; under `ForgetByCapacity` they neither count toward
  `Keep` nor get evicted.

## [v0.6.2] - 2026-05-23

Bundled release: introduces a stdlib-only `orchestrate.Supervisor` facade
over `StateGraph[S]` for planner/worker coordination, and closes three
correctness fixes from the v1.3 K1-closure wave — Phase 2 Gap B
(AccumulateStream Index-keyed merge per K1), P1-4 (RunStream cancel emits
terminal Done event), and P1-3 (a2a server worker DELETE cancel). All
signatures unchanged; observable behavior is strictly more correct per
the documented K1 and cancellation contracts. Callers compiling against
`v0.6.1` compile unchanged against `v0.6.2` (KC-5 preserved).

### Added

- New `orchestrate.Supervisor` surface — `NewSupervisor`, `SupervisorOptions`,
  `Dispatch`, `WorkerResult`, `DispatchParser`, `Aggregator`, `Run`,
  `RunStream`, and the sentinel family for validation/dispatch errors.
- New `orchestrate/supervisor.go` implementation plus deterministic tests for
  budget propagation, policy composition, and runtime composition with
  `StateGraph[S]`.
- New `examples/08-supervisor/` demo — basic coordination, budget gate,
  and compose-with-graph smoke test.

### Fixed

- `llm.AccumulateStream` now merges streaming tool-call deltas by
  `ToolCallDelta.Index` (the stable per-tool-call key per the K1
  contract), not by `ID`. The previous ID-keyed implementation
  silently dropped `EventToolCallArgsDelta` chunks whose `ID` field
  was empty — the standard OpenAI / Anthropic / Ollama wire shape
  where `ID` is populated only on the `EventToolCallStart` event. A
  separate symptom: two parallel tool calls with the same `ID` /
  `Name` at distinct `Index` (Ollama's `ID==Name` fallback) collapsed
  into one entry. Function signature is unchanged. The unexported
  `appendToolCallDelta` helper is removed; its logic is inlined into
  `AccumulateStream` with the new Index-keyed map and a deterministic
  first-Start ordering. (Phase 2 Gap B, closes the K1 "production
  accumulator" disclaimer at `llm/stream.go`.)
- `runStreamFromBlocking` and `Supervisor.RunStream` no longer silently close
  the `StepEvent` channel when `ctx` is canceled mid-run. Both now emit a
  terminal `StepEvent{Done: true, Err: ctx.Err()}` before close so SSE
  handlers and any `for ev := range ch` consumer can distinguish a clean
  finish from a mid-stream cancel. Terminal-event priority is
  `err > ctx.Err > Final` to guarantee exactly one Done event even when
  `runFn` races with cancel. (P1-4)
- `comm/a2a` server-side worker goroutine now cancels via
  `DELETE /tasks/{id}` instead of being unkillable. The worker previously
  ran with `context.Background()`; it now uses a per-task `WithCancel`
  whose cancel funcval lives on the Task and is invoked by the DELETE
  handler. Cancel reuses `TaskFailed` with `Error="canceled by DELETE"`
  to avoid adding a new enum state (clients that switch on `TaskState`
  remain exhaustive). (P1-3)

### Compatibility

- `llm.AccumulateStream` signature unchanged (`func(StreamReader)
  (Response, error)`). Observable behavior is strictly more correct
  per the K1 contract: no production caller relied on the prior
  broken behavior (existing production paths feed text-only streams;
  no tool-streaming path consumed `AccumulateStream` against a
  provider with a non-empty `ID` on Start + empty on subsequent
  ArgsDelta chunks).
- stdlib-only invariant preserved (no new third-party imports).

## [v0.6.1] - 2026-05-21

Additive release: introduces a stdlib-only `policy` sub-package — a
capability-preserving `llm.ChatModel` decorator that runs typed `Gate`
events at request, response, and stream boundaries. No behavior changes
to any existing package; consumers can stay on `v0.6.0` if they don't
need policy enforcement. Strict-additive: callers compiling against
`v0.6.0` compile unchanged against `v0.6.1` (KC-5 honored verbatim —
`llm/`, paradigm files, `agent_chatmodel.go`, `memory/`, `orchestrate/`,
`go.mod`, `go.sum` all byte-identical to the pre-Phase-36 state).

### Added

- New `policy` sub-package — capability-preserving `llm.ChatModel`
  decorator. Mirrors `otelmodel.Wrap` shape (KC-3) with the 8-wrapper
  type-switch tree + 21 compile-time interface assertions so
  `ToolCaller` / `Embedder` / `StructuredOutputs` capabilities are
  preserved through the wrap.
  - `policy.Wrap(model, gates...)` — convenience entry point.
  - `policy.WrapConfig(model, Config{Gates: ..., OnDecision: f})` —
    structured entry with optional audit callback (synchronous,
    nil-safe, panic-recovered).
  - `policy.Gate` interface + `policy.Event` struct + `policy.EventKind`
    enum with 5 kinds (`PreGenerate` / `PostGenerate` / `PreStream` /
    `StreamDelta` / `PostStream`).
  - `policy.Decision` struct + `policy.DecisionAction` enum with 4
    actions (`Allow` / `Block` / `Redact` / `Replace`).
  - `policy.ErrBlocked` sentinel + `policy.BlockedError` rich error
    pair (`errors.Is` umbrella + `errors.As` detail with embedded
    `Decision` copy).
- Three built-in gates (all stdlib-only):
  - `policy.NewPIIRedactor()` — redacts email / phone / IPv4 patterns
    (US-locale ssn / credit_card deferred to a future
    `NewUSLocalePIIRedactor` additive). `WithStreamRedaction` opt-in
    enables per-delta scanning (default OFF per Q4 — per-delta regex
    is expensive and cross-delta PII can leak by design).
  - `policy.NewInjectionScanner()` — pre-call block on the 4 canonical
    prompt-injection patterns (lifted from `llm-agent-rag/guard` by
    copy, not import, per KS-5).
  - `policy.NewMaxInputLen(n int)` — pre-call block when prompt size
    exceeds n bytes (Q3 — bytes are the operative cap for provider
    HTTP budgets; a future `MaxInputLenRunes` is a v1.3 additive
    candidate).
- Composition with `otelmodel.Wrap` is documented in
  `examples/07-policy/README.md` — canonical v1.2+ stack is
  `policy.Wrap(otelmodel.Wrap(provider), ...)`: outer denies before
  observed, middle observes, inner calls. The example main.go
  intentionally does NOT import the otel sister repo (Decision G —
  the sister-repo example ships in v1.3 when `llm-agent-otel` bumps
  to match core `v0.6.x`).
- New `examples/07-policy/` — three deterministic demos
  (`demoPIIRedaction`, `demoInjectionBlock`, `demoMaxInputLen`)
  driven by `llm.ScriptedLLM` proving each gate's decision action
  end-to-end. Run with `cd examples && go run ./07-policy`.

## [v0.6.0] - 2026-05-21

Additive release: introduces a stdlib-only shared test-helper sub-package.
No behavior changes to any existing package; consumers can stay on
`v0.5.1` if they don't need the new helpers.

### Added

- New `agentstest` sub-package — stdlib-only shared test helpers for
  `agents.Tool`. Provides `StubTool` / `NewStubTool` / `NewErrorTool` for
  building fake tools and `RecordingTool` (a thread-safe decorator that
  records every Execute call). Intended for sister-repo `*_test.go`
  consumption (analogous to `net/http/httptest` for `net/http`); avoids
  the previous pattern where each repo locally re-stubbed `agents.Tool`.
  See `agentstest/doc.go` for the bridging note (use
  `flow.FromAgentTool` to adapt to the narrower `flow.Tool` interface).

## [v0.5.1] - 2026-05-20

### Changed

- Bump `llm-agent-rag` to `v1.0.1` (back-edge refresh, no public-API change).

## [v0.5.0] - 2026-05-21

Post-`v0.4` RAG compatibility maintenance plus the Phase-31 alignment to the
standalone SDK's frozen `v1.0` API. The core stays stdlib-only; the public
`rag` facade API is unchanged.

### Changed

- bumped `github.com/costa92/llm-agent-rag` from `v0.1.4` to `v1.0.0`
  (the standalone SDK's frozen `v1.0` API).
- repaired the core `rag/` compatibility facade for the `v1.0.0` store
  contract — `storeAdapter` now enumerates documents via a real list route
  (`*InMemoryStore.ListDocuments` + an optional `lister` interface + an
  id-index fallback) instead of a `nil`-vector similarity search, which
  `v1.0.0`'s stricter `store.InMemoryStore.Search` rejects.
- bumped `github.com/costa92/llm-agent-rag` from `v0.1.2` to `v0.1.4` (the
  intermediate post-`v0.4` maintenance bump, superseded by the `v1.0.0`
  bump above).
- aligned the core `rag/` compatibility facade with the standalone retrieval
  policy path:
  - MQE / HyDE now delegate to standalone retrieval orchestration
  - `Ask(...)` delegates to standalone rerank + context packing flow
  - tool-facing `enable_rerank` is now plumbed through to standalone ask/search
- extended the core facade's internal store adapter for standalone contract
  parity:
  - `List(...)`
  - `RemoveByFilter(...)`

### Breaking

- Removed the deprecated v0.2 compatibility symbols from `llm/`:
  - `llm.Client`
  - `llm.LegacyClient`
  - `llm.GenerateRequest`
  - `llm.GenerateResponse`
  - `llm.StreamChunk`
  - `llm.StreamUsage`
- Any downstream still compiling against the removed surface must migrate to:
  - `llm.ChatModel`
  - `llm.Request`
  - `llm.Response`
  - `llm.StreamReader`
  - `llm.StreamEvent`

### Changed

- Core runtime packages now depend only on `llm.ChatModel`:
  - `rag`
  - `context`
  - `bench`
  - `rl`
- Repository examples, test helpers, and quick-start docs now show only the
  current `ChatModel` API.
- `rag` has been externally split into `github.com/costa92/llm-agent-rag`.
  Main-repo `rag/` now acts as a compatibility facade while this repo depends on:
  - `github.com/costa92/llm-agent-rag v0.1.0`

### Removed

- Deleted `llm/legacy.go`.
- Removed alias-only tests that existed to prove `Client`/`LegacyClient`
  round-tripping.

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

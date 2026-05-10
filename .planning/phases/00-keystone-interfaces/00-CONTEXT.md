# Phase 0: Multi-repo infra + keystone interfaces - Context

**Gathered:** 2026-05-10
**Status:** Ready for planning

> Note: ROADMAP.md names this phase "Multi-repo infra + `llm/v2` keystone interfaces". Per **D-01** below, the new package is NOT at `llm/v2/` — it's a reboot of `llm/`. Phase title carries the `llm/v2` shorthand for continuity with research SUMMARY's K1/K2 keystone language; the actual import path is `github.com/costa92/llm-agent/llm`.

<domain>
## Phase Boundary

Lock the capability-negotiation contract (K1 typed `StreamEvent`, K2 per-(provider × model) `ProviderInfo`, K3 decorator-friendly capability interfaces) and the multi-repo CI discipline (K6) **before any provider adapter is written**. Phase 0 produces:

- A rebooted `llm/` package with `ChatModel` + `ToolCaller` + `Embedder` + `StructuredOutputs` + `StreamEvent` + `ProviderInfo` types
- A deprecated `llm.LegacyClient` (renamed from current `llm.Client`) targeting removal in v0.4
- Migration guide for v0.2 → v0.3 callers
- 3 sister GitHub repos created with `go.mod` skeletons + LICENSE + OWNERS + README + per-repo CI
- Umbrella CI in `llm-agent` building all 4 repos against `llm-agent` HEAD
- `release-precheck` CI gate rejecting `replace` directives on tagged-release branches

No provider adapter code, no agent-paradigm refactor (that's Phase 3), no real network calls.

</domain>

<decisions>
## Implementation Decisions

### Package Path for New Capability Interfaces (Area 1)

- **D-01: Reboot the `llm/` package.** New types (`ChatModel`, `ToolCaller`, `Embedder`, `StructuredOutputs`, `StreamEvent`, `StreamReader`, `ProviderInfo`, `Capabilities`) live IN `llm/`. The current `llm.Client` is renamed to `llm.LegacyClient` with a `// Deprecated:` godoc comment naming `v0.4.0` as the target removal version.
  - **Why:** v0.x explicitly allows BC breaks (per repo policy); the project is small enough that `sed`-style migrations are realistic; nesting at `llm/v2/` is unusual within a single Go module (the `/v2` convention is for module-level major bumps, not subpackages); inventing a fresh name like `model/` or `chat/` adds churn without buying clarity.
  - **Import path stays:** `github.com/costa92/llm-agent/llm`.
  - **Old types coexist:** in the same package, side-by-side with new types, until v0.4 removal.
  - **Cascades into:** CORE-01..09 file layout, migration guide diff examples, deprecation notices in CHANGELOG.

### `ProviderInfo` and `Capabilities` Shape (Area 2)

- **D-02: `Capabilities` is an embedded struct field on `ProviderInfo`** (NOT methods, NOT bitmask).
  - Concrete shape:
    ```go
    type ProviderInfo struct {
        Provider     string       // "openai", "anthropic", "ollama"
        Model        string       // "gpt-4o-mini", "claude-3-5-haiku", "llama3.1:8b"
        Capabilities Capabilities
    }

    type Capabilities struct {
        Tools             bool
        Embeddings        bool
        StructuredOutputs bool
        PromptCaching     bool
    }
    ```
  - **Why struct over methods:** JSON-serializable for OTel attributes (`gen_ai.provider.capabilities.*`), enables future "capabilities are data" patterns (e.g., capability-degradation logging in agent layer), simpler exploration via godoc.
  - **Why struct over bitmask:** Self-documenting (no opaque `0b00011` in test failures), godoc shows field purposes, easier to extend with non-bool capabilities (e.g., `MaxToolsPerCall int` later).
  - **Type assertion stays as PRIMARY signal at compile-time** (Eino's `BindTools` lesson — small interfaces let agents write `if tc, ok := model.(ToolCaller); ok { ... }`). `Capabilities` is the **runtime** signal for per-(provider × model) variation that type assertion can't see (Pitfall 6, K2).
  - **Cascades into:** CORE-06 (per-(provider × model) granularity); OTel span/metric attribute design (Phase 5); agent fallback paths (Phase 3).

### Mock Organization (Area 3)

- **D-03: One full-capability `ScriptedLLM` v2 + small per-capability mocks for fallback testing.**
  - `ScriptedLLM` (in `llm/scriptedllm.go` or `llm/mock.go` — promoted out of `_test.go` since adapters need it too) implements `ChatModel + ToolCaller + Embedder + StructuredOutputs`. Replaces the current `scriptedllm_test.go` shape.
  - `ChatOnlyMock` (alongside, e.g., `llm/chat_only_mock.go`) implements only `ChatModel` — used to test capability-degradation paths (e.g., "ReAct falls back to scratchpad templating when `model.(ToolCaller)` assertion fails").
  - **Why both:** ScriptedLLM is the default for happy-path agent tests (1-line setup); `ChatOnlyMock` exists specifically to verify that agents handle missing capabilities gracefully — a critical Phase 3 test surface.
  - **Why ScriptedLLM is non-test code:** sister repo conformance suites also need it (`internal/contract` runs the same fixtures against real adapters AND ScriptedLLM as a sanity baseline). Living in `_test.go` prevents reuse.
  - **Cascades into:** Phase 1 conformance suite design (CONF-01); Phase 3 agent fallback tests (CORE-10).

### Sister Repo Creation Strategy (Area 4)

- **D-04: Phase 0 creates all 3 sister GitHub repos AND pushes the skeleton.**
  - Repos: `github.com/costa92/llm-agent-providers`, `github.com/costa92/llm-agent-otel`, `github.com/costa92/llm-agent-customer-support`.
  - Each gets a Phase-0 skeleton:
    - `go.mod` declaring the module path; depends on `github.com/costa92/llm-agent` at the v0.3.0-pre tag once available
    - `LICENSE` (MIT — same as core)
    - `OWNERS` (single owner — costa)
    - `README.md` documenting purpose + cross-repo iteration pattern (`go.work` recommended; `replace` only as documented escape hatch — INFRA-06)
    - `.github/workflows/test.yml` (per-repo CI: `GOWORK=off go build ./... && go test ./...`; release-precheck for `replace` ban — INFRA-04)
    - `.gitignore` including `go.work` (INFRA-02)
  - **Why upfront:** umbrella CI (INFRA-05) needs the 3 repos to exist (it clones them via a sibling-aware `go.work`) — physically impossible to land Phase 0 without the repos. Discovering this in Phase 1 means re-doing Phase 0.
  - **Why GitHub-public from day 1:** `go get` resolution depends on the repos being public; nightly Ollama-live CI in Phase 1 needs PR-time CI access; downstream users iterating on a sister repo via `go.work` need to be able to `git clone` them.
  - **Cost:** ~1-2 hours setup work for the 3 repos (creation, branch protection, secrets — though no secrets needed in Phase 0).
  - **Cascades into:** every subsequent phase (each lands in a now-existing repo); umbrella CI design; release-tag coordination.

### Claude's Discretion

- **Migration guide depth (CORE-09):** target 1 worked example for the most-used paradigm (Simple, since it's the docs example) + a generic `llm.Client → llm.ChatModel` mapping table covering all type renames and method-signature shifts. Other 4 paradigms covered by the mapping table; full per-paradigm worked examples deferred until users ask.
- **`StreamEvent` exact field names:** locked by D-01/D-02 directionally (typed Kind enum + per-tool-call Index + delta fields), but precise field naming (`Args` vs `Arguments` vs `ArgsDelta`) is researcher/planner discretion — match Anthropic SDK's `partial_json` and OpenAI SDK's `function.arguments` ergonomics where possible.
- **Sister repo READMEs (INFRA-06):** Claude drafts; user reviews at PR time. Stick to: purpose, install command, cross-repo iteration pattern, link back to the core repo's CLAUDE.md.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project-level (always)
- `.planning/PROJECT.md` — milestone scope, Core Value, Validated/Active/Out of Scope, Key Decisions
- `.planning/REQUIREMENTS.md` — 65 v1 requirements; CORE-01..09 + INFRA-01..07 are the Phase 0 surface
- `.planning/ROADMAP.md` §"Phase 0: Multi-repo infra + `llm/v2` keystone interfaces" — phase scope, success criteria, requirement mapping, pitfalls guarded
- `.planning/STATE.md` — current position, accumulated decisions

### Research bundle (Phase 0 directly consumes)
- `.planning/research/SUMMARY.md` §"The 5–7 Keystone Decisions" — K1, K2, K3, K6 ALL land in Phase 0
- `.planning/research/SUMMARY.md` §"Conflicts to Resolve" — Conflicts B (`ProviderInfo` granularity) and C (`replace` directives) settled in Phase 0
- `.planning/research/ARCHITECTURE.md` §"Capability negotiation" — small interfaces + type assertion + `ProviderInfo` hint pattern
- `.planning/research/ARCHITECTURE.md` §"Streaming + tool calls" — typed `StreamEvent` union, per-tool-call indexing
- `.planning/research/ARCHITECTURE.md` §"Multi-repo Go architecture" — sibling-not-cousin dependency direction (`go.opentelemetry.io/contrib` precedent)
- `.planning/research/PITFALLS.md` Pitfall 6 — capabilities are per-(provider × model)
- `.planning/research/PITFALLS.md` Pitfalls 12–14 — `replace`-in-tag, `go.work` commit, cross-repo break
- `.planning/research/PITFALLS.md` Pitfall 15 — deprecation lifecycle
- `.planning/research/PITFALLS.md` Pitfall 22 — architectural-drift baseline (`go doc ./...` snapshot at phase exit)
- `.planning/research/STACK.md` §"Multi-repo Go modules" — `go.work` placement, gitignore policy, replace banishment
- `.planning/research/FEATURES.md` §"P1 / Cross-Cutting D — Capability interfaces" — feature taxonomy

### External specs / SDK references (consulted, not primary)
- [Eino `ChatModel` interface](https://github.com/cloudwego/eino/blob/main/components/model/interface.go) — confirmed pattern of small capability interfaces (`ToolCallingChatModel` extends `BaseChatModel`)
- [Genkit Go `ai.ModelSupports`](https://genkit.dev/go/docs/plugin-authoring-models) — capability hint precedent
- [OpenTelemetry `gen_ai.*` semconv](https://opentelemetry.io/docs/specs/semconv/gen-ai/) — Phase 0 designs ChatModel + Wrap to compose with these (Phase 5 implements)
- Existing `github.com/costa92/llm-agent/llm/client.go` — types being rebooted

### Codebase artifacts to reference at planning time
- `llm/client.go` — current `Client` interface, types being renamed/replaced
- `agent.go`, `react.go`, `function_call.go`, `simple.go`, `plan_solve.go`, `reflection.go` — agent paradigms whose paradigm-refactor blocker is the new interfaces (refactor itself is Phase 3, but Phase 0 designs interfaces with these consumers in mind)
- `scriptedllm_test.go` — current mock; ScriptedLLM v2 replaces this
- `.github/workflows/test.yml` — current CI shape; Phase 0 adds umbrella CI, refactors existing CI
- `CHANGELOG.md` — Phase 0 adds Deprecated entry; v0.4 cut adds Breaking entry (Phase 7)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets

- **`llm/client.go`** — `Tool`, `ToolCall`, `Message`, `FinishReason`, `StreamUsage` types are mostly directly portable to v0.3. `Client` interface itself is the thing being renamed to `LegacyClient` and superseded; supporting types stay (with possible field additions for `Capabilities`-bound metadata).
- **`scriptedllm_test.go`** — design template for `ScriptedLLM v2`; promote to non-test code, extend to satisfy all v0.3 capability interfaces.
- **`agents.NewRegistry` + `Tool` interface in `tool.go`** — agent-side tool dispatch is unaffected by Phase 0; `ToolCall` type from `llm` package flows through this unchanged.
- **`example_simple_test.go`** — minimal agent invocation; the migration guide's worked example tracks this file's diff.
- **`.github/workflows/test.yml`** — already handles stdlib-only modules with no `go.sum`; the umbrella CI extends rather than replaces it.

### Established Patterns

- **Stdlib-only, no `go.sum`** — Phase 0 must preserve this in core. Sister repos can take deps; their `go.mod`s have `go.sum`s normally.
- **One package per concept, no init()** — observed across all 12 packages; new `llm/` types follow.
- **Test files colocated with code** — `*_test.go` per package, no `_test/` subdirectory; ScriptedLLM v2 stays in `llm/` even after promotion to non-test code.
- **Minimal interfaces** — `Client` is 2 methods; `Tool` is 3; the agents framework's seam is intentionally narrow. v0.3 capability interfaces follow: `ChatModel` is the smallest possible (Generate + Stream); each capability is a separate interface.
- **`json` tag everywhere on public types** — current `Client`, `GenerateRequest`, `Tool`, `ToolCall` all carry `json:"..."` tags. New types continue this; `Capabilities` JSON-serializability (D-02) requires it.
- **README has design-spec links to parent AICS repo** — informative but external; Phase 0 doesn't need to consult them since `.planning/research/` covers what's needed.

### Integration Points

- **Where new types connect to existing system:**
  - `llm/` package — internal namespace where new types LIVE
  - `agent.go` — `Agent` interface accepts `Client` today; in Phase 3 will accept `ChatModel`. Phase 0 only defines the new types; agents stay on `LegacyClient` until Phase 3.
  - `examples/` — 5 demos use ScriptedLLM via `llm.Client`; they continue working through Phase 0 because `LegacyClient` is alias-equivalent. Phase 3 migrates them.
  - `scriptedllm_test.go` — promoted to `llm/scripted.go` (non-test); test file becomes a thin import test. Existing tests across the repo that construct `ScriptedLLM{}` get a one-line import change.
  - **Sister repo connection:** every sister repo has `require github.com/costa92/llm-agent vX.Y.Z` in `go.mod`. In Phase 0, the version is `v0.3.0-pre.1` or similar pre-release tag. Sister repos exercise the new `llm.ChatModel` interface (in their CI mocks) before any provider implementation lands.

</code_context>

<specifics>
## Specific Ideas

- **Migration guide format:** prefer one canonical worked example over a long "every type one row" mapping. Users learn faster from "see how `simple_test.go` changed" than from a 30-row table.
- **Sister repo OWNERS format:** mirror the existing `llm-agent` OWNERS (single owner: costa).
- **Per-repo CI:** mirror existing `.github/workflows/test.yml` shape (matrix over Go versions if applicable, lint + vet + test). Sister repos may take `go.uber.org/goleak` as a TEST-only dep starting in Phase 1.
- **Deprecation comment format:** `// Deprecated: Use llm.ChatModel instead. LegacyClient will be removed in v0.4.0. See docs/migration-v0.2-to-v0.3.md.`
- **Pre-release tagging:** core repo tags `v0.3.0-pre.1` after Phase 0 lands, so sister repos can `require` it during Phases 1–6. Final `v0.3.0` tag waits for Phase 6 completion.

</specifics>

<deferred>
## Deferred Ideas

- **Provider Author Guide v0.1** — explicitly Phase 1 scope (CORE-11). Phase 0 provides the *types* the guide will document; the guide itself waits until at least one adapter is built (Phase 1's first parallel plan) to ground the prose in real-world wire-format gotchas.
- **Agent paradigm refactor (CORE-10)** — Phase 3 scope. Phase 0 designs interfaces with the 5 paradigms' needs in mind but does NOT touch `simple.go`, `react.go`, etc.
- **OTel decorator implementation** — Phase 5 scope. Phase 0 ensures the interfaces COMPOSE under wrapping (capability re-implementation is feasible) but does NOT write the decorator.
- **Conformance test harness** — Phase 1 scope (`internal/contract/`). Phase 0 promotes ScriptedLLM v2 out of `_test.go` so the harness in Phase 1 can import it; the harness itself isn't built here.
- **Anti-features re-checked:** vision, RL training, K8s, cross-framework bridges remain Out of Scope per PROJECT.md and REQUIREMENTS.md. None resurfaced during this discussion.

</deferred>

---

*Phase: 0-keystone-interfaces*
*Context gathered: 2026-05-10*

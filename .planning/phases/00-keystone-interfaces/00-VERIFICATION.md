---
phase: 00-keystone-interfaces
verified: 2026-05-10T00:00:00Z
status: passed
score: 16/16 must-haves verified
overrides_applied: 0
re_verification:
  previous_status: none
  previous_score: n/a
  gaps_closed: []
  gaps_remaining: []
  regressions: []
gaps: []
deferred: []
human_verification: []
---

# Phase 0: Multi-repo infra + `llm/v2` keystone interfaces — Verification Report

**Phase Goal:** Lock the capability-negotiation contract (K1 typed `StreamEvent`,
K2 per-(provider × model) `ProviderInfo`, K3 decorator-friendly capability
interfaces) and the multi-repo CI discipline (K6) BEFORE any provider adapter
is written.

**Verified:** 2026-05-10
**Status:** PASS
**Re-verification:** No — initial verification (no previous VERIFICATION.md present)

---

## Verdict

# PASS

All 4 keystones (K1, K2, K3, K6) are implemented and locked in code. All 4
implementation decisions (D-01..D-04) honored. All 16 phase requirements
(INFRA-01..07 + CORE-01..09) have concrete code/CI/doc artifacts that satisfy
their contracts. The repo builds clean (`go vet ./...` + `go build ./...` +
`go test ./... -count=1` all green; 15 packages pass), the stdlib-only
invariant on the core module is intact (zero `require` directives in
`go.mod`), and the Pitfall 22 architectural-drift baseline has been captured
in `docs/api-snapshot.txt` (3495 lines). Phase 0 is ready to close; Phase 1
(walking-skeleton Generate adapters) is unblocked.

Two items remain explicitly out-of-band, exactly as the plans documented and
the user opted into deferral:

1. `git tag v0.3.0-pre.1` on the core repo — must be pushed so sister-repo
   CI can resolve their `require github.com/costa92/llm-agent v0.3.0-pre.1`
   line. Until then, sister-repo `test` workflows are RED by design (00-03
   SUMMARY.md §"First CI Run Status").
2. Branch protection on `main` for the 3 sister repos and live smoke-test
   of umbrella.yml + release-precheck.yml — deferred at the user's request
   per 00-05 SUMMARY.md §"Smoke-test deferral" and 00-03 SUMMARY.md §"Task 4
   Awaiting (Branch Protection)".

Both items are tracked human-actionable follow-ups, not gaps in code-shipped
deliverables. They do not block Phase 1 starting.

---

## Goal Achievement

### Observable Truths (Keystones)

| #  | Truth                                                         | Status     | Evidence                                                                                                                                                                                                       |
| -- | ------------------------------------------------------------- | ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| K1 | Typed `StreamEvent` union with stable per-tool-call `Index`   | ✓ VERIFIED | `llm/stream.go` defines `StreamEvent` struct (line 41), `StreamEventKind uint8` enum with 6 variants (lines 22–31: `EventTextDelta`, `EventToolCallStart`, `EventToolCallArgsDelta`, `EventToolCallEnd`, `EventThinkingDelta`, `EventDone`), `ToolCallDelta.Index int` field (line 66), and `StreamReader` iterator interface (lines 13–16: `Next() (StreamEvent, error)` + `Close() error`). `var _ StreamReader = (*scriptedStream)(nil)` satisfaction implicit via `newScriptedStream` returning `StreamReader` (line 224). |
| K2 | Per-(provider × model) `ProviderInfo`                         | ✓ VERIFIED | `llm/info.go`: `ProviderInfo` struct has `Provider, Model string` + embedded `Capabilities Capabilities` (lines 8–12). `Capabilities` struct has 4 bool fields with snake_case JSON tags: `Tools`, `Embeddings`, `StructuredOutputs`, `PromptCaching` (lines 25–30). `ChatModel` interface declares `Info() ProviderInfo` method (`llm/chatmodel.go:20`). Construction-time model binding documented in `llm/info.go` lines 4–7 ("Provider instances bind a model at construction time"). |
| K3 | Decorator-friendly capability interfaces                      | ✓ VERIFIED | `llm/chatmodel.go` defines `ChatModel` as the smallest possible interface (Generate + Stream + Info — 3 methods, lines 17–21). `llm/capabilities.go` defines `ToolCaller`, `Embedder`, `StructuredOutputs` as separate interfaces (lines 13–16, 27–30, 39–42). `WithTools` is documented and implemented as IMMUTABLE (returns new value): `llm/capabilities.go:6–9`, `llm/scripted.go:103–122` (field-by-field copy creates a fresh `*ScriptedLLM`, not `cp := *s`). `WithSchema` likewise immutable: `llm/capabilities.go:35–42`, `llm/scripted.go:154–174`. `TestToolCallerImmutable` (`llm/llm_test.go:115`) asserts `a != b` after `WithTools(toolsA)` and `WithTools(toolsB)` from same base, plus concurrent-safety check. |
| K6 | Multi-repo CI discipline                                      | ✓ VERIFIED | `.gitignore:21` lists `go.work`; `.gitignore:22` lists `go.work.sum`. `.github/workflows/test.yml:14` sets workflow-level `GOWORK: off`. `.github/workflows/umbrella.yml` exists (104 lines, checks out all 4 repos via `actions/checkout@v4` with `repository: costa92/...`, then `go work init` over all 4 modules). `.github/workflows/release-precheck.yml` exists (37 lines, parses `go mod edit -json` for `Replace` and rejects non-empty). All 3 sister repos verified PUBLIC on GitHub via `gh repo view`: `llm-agent-providers`, `llm-agent-otel`, `llm-agent-customer-support`. |

**Score: 4/4 keystones verified.**

### Implementation Decisions (D-01..D-04)

| #    | Decision                                                                                                                                       | Status     | Evidence                                                                                                                                                                                                                                                                                                                              |
| ---- | ---------------------------------------------------------------------------------------------------------------------------------------------- | ---------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| D-01 | Reboot `llm/` package in-place; rename `Client` → `LegacyClient`; `// Deprecated:` comment names v0.4.0 target                                 | ✓ VERIFIED | `llm/legacy.go` exists (renamed from `client.go`; commit 7abbeb9 = `git mv`). `LegacyClient` interface defined at `llm/legacy.go:8–11`. `type Client = LegacyClient` alias at line 16. 6 `// Deprecated:` comments naming `v0.4.0` (`grep -c "// Deprecated:" llm/legacy.go` = 6). Format matches CONTEXT.md spec exactly. Import path unchanged (still `github.com/costa92/llm-agent/llm`). New types coexist in same package: `chatmodel.go`, `capabilities.go`, `info.go`, `stream.go`, `errors.go`, `types.go`, `scripted.go`, `chat_only_mock.go`. |
| D-02 | `Capabilities` is an embedded struct field on `ProviderInfo` (NOT methods, NOT bitmask)                                                        | ✓ VERIFIED | `llm/info.go:11`: `Capabilities Capabilities `json:"capabilities"` ` — struct field. Shape exactly matches CONTEXT.md: 4 bool fields named `Tools`, `Embeddings`, `StructuredOutputs`, `PromptCaching` (lines 26–29). JSON tags use snake_case (`tools`, `embeddings`, `structured_outputs`, `prompt_caching`). `TestProviderInfo_JSONRoundTrip` (`llm/llm_test.go:219`) asserts the exact wire-format: `{"provider":"openai","model":"gpt-4o-mini","capabilities":{"tools":true,...}}`. |
| D-03 | One full-capability `ScriptedLLM` v2 (non-test code) + `ChatOnlyMock` (ChatModel-only)                                                         | ✓ VERIFIED | `llm/scripted.go` exists (256 LOC, NOT `_test.go`). `ScriptedLLM` satisfies all 4 capability interfaces — compile-time `var _` block at lines 43–48: `_ ChatModel`, `_ ToolCaller`, `_ Embedder`, `_ StructuredOutputs`. `llm/chat_only_mock.go` exists (34 LOC, NOT `_test.go`). `ChatOnlyMock` implements ONLY `ChatModel` — single `var _ ChatModel = (*ChatOnlyMock)(nil)` at line 18. `TestChatOnlyMockExcludesCapabilities` (`llm/llm_test.go:28`) asserts the negative claims (NOT `ToolCaller`, NOT `Embedder`, NOT `StructuredOutputs`) at runtime via type-assertion. |
| D-04 | All 3 sister GitHub repos exist publicly                                                                                                       | ✓ VERIFIED | `gh repo view costa92/llm-agent-providers --json visibility` → `PUBLIC`. Same for `llm-agent-otel` and `llm-agent-customer-support`. 00-03-SUMMARY.md confirms 8-file Phase-0 skeleton (go.mod / LICENSE / OWNERS / README.md / .gitignore / scripts/workspace.sh / .github/workflows/test.yml / .github/workflows/release-precheck.yml) on each repo's `main`. SHA256 of `scripts/workspace.sh` (`8eda10c3e7a337a5551eef68d43732d71533663f0aaa66e1c0c729be796a09ec`) byte-identical across all 4 repos (asserted in `umbrella.yml:52–69`). |

**Score: 4/4 decisions honored.**

### Required Artifacts (Code surface in `llm/`)

| Artifact                       | Expected                                                            | Status     | Details                                                                                                  |
| ------------------------------ | ------------------------------------------------------------------- | ---------- | -------------------------------------------------------------------------------------------------------- |
| `llm/chatmodel.go`             | `ChatModel` interface (Generate + Stream + Info)                    | ✓ VERIFIED | 21 LOC; 3-method interface with concurrent-use contract documented (line 16)                             |
| `llm/capabilities.go`          | `ToolCaller`, `Embedder`, `StructuredOutputs` interfaces            | ✓ VERIFIED | 42 LOC; immutable WithTools/WithSchema; Embedder deliberately does NOT embed ChatModel (line 26)         |
| `llm/info.go`                  | `ProviderInfo` + `Capabilities` struct (D-02 shape)                 | ✓ VERIFIED | 30 LOC; 4 bool fields with snake_case JSON tags; D-02 rationale comments (lines 14–24)                   |
| `llm/stream.go`                | `StreamReader` + `StreamEvent` + `StreamEventKind` + `ToolCallDelta`| ✓ VERIFIED | 145 LOC; 6-variant Kind enum; per-tool-call Index field; Pitfall-1 documentation (lines 51–55)            |
| `llm/types.go`                 | `Request`, `Response`, `Message`, `Tool`, `ToolCall` (with ID), `Vector`, `Usage`, `UsageSource`, `FinishReason` alias | ✓ VERIFIED | 105 LOC; ToolCall.ID is NEW vs v0.2 (line 66); UsageSource has 3 constants (Reported/Estimated/Unknown); FinishReason is `type FinishReason = legacyFinishReason` alias |
| `llm/errors.go`                | `ErrCapabilityNotSupported` + `ErrScriptExhausted` sentinels        | ✓ VERIFIED | 21 LOC; `errors.New` sentinels; canonical `errors.Is` round-trip tested in `llm/llm_test.go:180`         |
| `llm/legacy.go`                | Renamed from client.go; LegacyClient + Client alias + 6 Deprecated  | ✓ VERIFIED | 85 LOC; `git mv` from `client.go` (commit 7abbeb9); `type Client = LegacyClient` alias; 6 Deprecated comments naming v0.4.0 |
| `llm/scripted.go`              | ScriptedLLM v2 (full capability mock)                               | ✓ VERIFIED | 256 LOC (production code); 4 capability `var _` assertions at file scope; functional options pattern; concurrent-safe via sync.Mutex |
| `llm/chat_only_mock.go`        | ChatOnlyMock (ChatModel-only)                                       | ✓ VERIFIED | 34 LOC (production code); `var _ ChatModel = (*ChatOnlyMock)(nil)` only; Capabilities all-false in Info() |
| `llm/doc.go`                   | Package doc with capability-negotiation idiom                       | ✓ VERIFIED | 62 LOC; documents all 16 exported types; canonical `if tc, ok := model.(ToolCaller); ok && model.Info().Capabilities.Tools { ... }` idiom (lines 40–46) |
| `llm/llm_test.go`              | 8 tests covering interface satisfaction + JSON + concurrency        | ✓ VERIFIED | 243 LOC; 8 tests all green: TestLegacyClientAlias, TestChatOnlyMockExcludesCapabilities, TestScriptedLLM_Capabilities, TestToolCallerImmutable, TestStreamReaderClosesIdempotent, TestSentinelErrors_ErrorsIs, TestStreamEventKind_Variants, TestProviderInfo_JSONRoundTrip |

### Required Artifacts (Multi-repo + Docs)

| Artifact                                                       | Expected                                                  | Status     | Details                                                                                                  |
| -------------------------------------------------------------- | --------------------------------------------------------- | ---------- | -------------------------------------------------------------------------------------------------------- |
| `.gitignore`                                                   | Includes `go.work` (Pitfall 13)                           | ✓ VERIFIED | Lines 21–22 list `go.work` and `go.work.sum`                                                              |
| `.github/workflows/test.yml`                                   | Workflow-level `GOWORK: off` (INFRA-02)                   | ✓ VERIFIED | Line 14: `GOWORK: off  # INFRA-02: CI never picks up a workspace file silently`                            |
| `.github/workflows/umbrella.yml`                               | 4-repo cross-build on PR (INFRA-05)                       | ✓ VERIFIED | 104 LOC; checks out all 4 sibling repos; runs `go vet/build/test` against each module against this PR's `llm-agent`; SHA256 cross-check on `scripts/workspace.sh` (lines 52–69) |
| `.github/workflows/release-precheck.yml`                       | Replace-ban on `release/**` branches (INFRA-04)           | ✓ VERIFIED | 37 LOC; triggers on push/PR to `release/**`; uses `go mod edit -json` parsed by python3 to count `Replace` entries; fails loudly if > 0 |
| `scripts/workspace.sh`                                         | Sibling-aware go.work writer (INFRA-03)                   | ✓ VERIFIED | 40 LOC; 100755 mode; SHA256 = `8eda10c3...` byte-identical with sister repos; idempotent (rm -f + go work init) |
| `docs/migration-v0.2-to-v0.3.md`                               | Migration guide with worked example (CORE-09)             | ✓ VERIFIED | 207 LOC; Quick reference table (13 rows); Simple paradigm worked example (3 variants: v0.2 / v0.3 transitional / v0.3 idiomatic); capability detection + streaming + when-to-migrate sections |
| `DEPRECATIONS.md`                                              | Single source of truth, symbol → removal version (Pitfall 15) | ✓ VERIFIED | 56 LOC; 4-column table covers `llm.Client`, `llm.LegacyClient`, `llm.GenerateRequest`, `llm.GenerateResponse`, `llm.StreamChunk`, `llm.StreamUsage`, `agents.scriptedLLM` shim; removal procedure + adding-new-deprecations howto |
| `CHANGELOG.md` `[Unreleased]` section                          | INFRA-07 versioning policy + Added/Deprecated subsections | ✓ VERIFIED | +83 LOC append (commit efd417d); ### Added (16 symbols); ### Deprecated (7 symbols, v0.4.0 + Phase-3 targets); ### Versioning policy with 4-repo table |
| `docs/api-snapshot.txt`                                        | Pitfall 22 architectural-drift baseline                   | ✓ VERIFIED | 3495 lines; covers all 12+ packages' godoc surface (`go list ./...` × `go doc -all`); contains all keystone type names (`ChatModel`, `ToolCaller`, `Embedder`, `StructuredOutputs`, `StreamReader`, `StreamEvent`, `ProviderInfo`, `Capabilities`) at lines 1676+ |

### Key Link Verification

| From                                         | To                                                | Via                                                                                                              | Status     | Details                                                                                                  |
| -------------------------------------------- | ------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- | ---------- | -------------------------------------------------------------------------------------------------------- |
| `ChatModel.Info()`                           | `ProviderInfo` + `Capabilities`                   | Method signature `Info() ProviderInfo` (`llm/chatmodel.go:20`)                                                   | ✓ WIRED    | `*ScriptedLLM.Info()` (`llm/scripted.go:95–99`) returns concrete value; `*ChatOnlyMock.Info()` (`llm/chat_only_mock.go:28–34`) returns concrete value with all-false Capabilities (per D-03 contract) |
| `ToolCaller.WithTools`                       | New ToolCaller (immutable)                        | Method signature returns `(ToolCaller, error)` not `error` (`llm/capabilities.go:15`)                            | ✓ WIRED    | `*ScriptedLLM.WithTools` (`llm/scripted.go:103–122`) constructs new `&ScriptedLLM{...}` field-by-field (avoids `go vet copylocks` on sync.Mutex); `TestToolCallerImmutable` asserts `a != b` |
| `StructuredOutputs.WithSchema`               | New ChatModel (schema-bound)                      | Method signature returns `(ChatModel, error)` (`llm/capabilities.go:41`)                                         | ✓ WIRED    | `*ScriptedLLM.WithSchema` (`llm/scripted.go:154–174`) constructs new value; test (`llm/llm_test.go:104–111`) asserts return value implements `ChatModel`                                              |
| `Capabilities` JSON                          | OTel attribute serialization                      | Snake-case JSON tags on all 4 fields (`llm/info.go:26–29`)                                                       | ✓ WIRED    | `TestProviderInfo_JSONRoundTrip` (`llm/llm_test.go:219–243`) asserts byte-exact `{"provider":"openai","model":"gpt-4o-mini","capabilities":{"tools":true,"embeddings":true,"structured_outputs":false,"prompt_caching":false}}` |
| `LegacyClient` (deprecated)                  | `ChatModel` (forward path)                        | `// Deprecated:` godoc comments name v0.4.0 + migration guide path (`llm/legacy.go:7,15,20,35,67`)                | ✓ WIRED    | All 6 deprecation comments link to `docs/migration-v0.2-to-v0.3.md`. `type Client = LegacyClient` alias preserves source compat. Migration guide exists with concrete worked example. |
| Umbrella CI (`umbrella.yml`)                 | All 4 repos' build outputs                        | `actions/checkout@v4` with `repository: costa92/...`; `go work init`; per-repo `go vet/build/test`               | ✓ WIRED    | Lines 22–45 check out all 4; lines 71–104 run vet/build/test in each module; SHA256 cross-check (line 52–69) asserts `scripts/workspace.sh` byte-identical                                          |
| Release-precheck (`release-precheck.yml`)    | `replace` directive ban                           | `go mod edit -json` parsed by python3 to count `Replace` array entries (`release-precheck.yml:24–35`)             | ✓ WIRED    | Trigger: `push` + `pull_request` on `release/**`. Exit 1 with `::error::` annotation if `replace_count != 0`. Byte-identical workflow shipped to all 3 sister repos (SHA256 verified).               |
| Sister repos                                 | Core `llm-agent` v0.3.0-pre.1                     | `require github.com/costa92/llm-agent v0.3.0-pre.1` in each sister repo's `go.mod`                                | ⚠ PARTIAL  | Sister go.mod files have the require line (per 00-03 SUMMARY); however core repo has NOT yet tagged `v0.3.0-pre.1`. Sister-repo `test` workflows are RED until tag is pushed. **This is documented intentional out-of-band work** — see "Out-of-band remaining" section below. |

The `Sister repos → core` link is "PARTIAL" only on the live-CI dimension
(sister CI green) — the wiring on disk (`require` line, gitignore, GOWORK=off,
release-precheck) is complete. The remaining step (`git tag v0.3.0-pre.1`)
is single-command and documented as the Phase-0 close signal in 00-05
SUMMARY.md §"Out-of-band". This is expected per RESEARCH.md Q3 RESOLVED and
not a code gap.

### Behavioral Spot-Checks

| Behavior                                                                | Command                                              | Result                                          | Status |
| ----------------------------------------------------------------------- | ---------------------------------------------------- | ----------------------------------------------- | ------ |
| Core repo builds cleanly                                                | `go vet ./...`                                       | exit 0, no output                               | ✓ PASS |
| Core repo tests pass across all packages                                | `go test ./... -count=1`                             | 15 packages green; `internal/testenv` no tests  | ✓ PASS |
| `llm` package godoc lists all keystone types                            | `go doc ./llm/`                                      | Lists: ChatModel, ChatOnlyMock, Embedder, FinishReason, LegacyClient, ProviderInfo, Request, Response, ScriptedLLM, StreamEvent, StreamEventKind, StreamReader, StructuredOutputs, Tool, ToolCall, ToolCallDelta, ToolCaller, Usage, UsageSource, Capabilities, Message, Vector, Client (alias), GenerateRequest, GenerateResponse, StreamChunk, StreamUsage | ✓ PASS |
| Core go.mod is stdlib-only                                              | `grep -c '^require' go.mod`                          | 0                                               | ✓ PASS |
| `// Deprecated:` count on `llm.LegacyClient` and companion types       | `grep -c "// Deprecated:" llm/legacy.go`             | 6 (covers LegacyClient, Client alias, GenerateRequest, GenerateResponse, StreamChunk, StreamUsage) | ✓ PASS |
| Pitfall 22 baseline non-empty                                           | `wc -l docs/api-snapshot.txt`                        | 3495                                            | ✓ PASS |
| 3 sister repos exist publicly on GitHub                                 | `gh repo view costa92/llm-agent-providers --json visibility` (and 2 more) | All return `"visibility":"PUBLIC"` | ✓ PASS |

### Anti-Patterns Found

No anti-patterns detected.

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| _(none)_ | _(none)_ | _(none)_ | _(none)_ | _(none)_ |

Notes on apparent flagged-but-not-anti-pattern items:

- `llm/scripted.go:151–153` — `WithSchema` is documented as a no-op for the
  mock ("does not validate JSON schemas"). This is **intentional** per CONTEXT.md
  D-03 / RESEARCH.md (mocks are deterministic shells, not validators); the
  godoc says so explicitly. Not a stub — it returns a new immutable
  `*ScriptedLLM` exactly as `WithTools` does, satisfying the StructuredOutputs
  contract.

- `llm/stream.go:96–103` — `AccumulateStream`'s tool-call branch is
  documented as a "compile-and-pass-trivial-smoke-test" implementation
  (lines 95–100); the production accumulator lands in Phase 2 along with
  the actual streaming adapters. This is documented and intentional —
  Phase 0 is the contract layer, not the accumulator implementation. Both
  the `EventDone` branch (line 108) and the helper itself are real working
  code (the Phase-0-scoped happy path of `EventTextDelta` → `EventDone`
  fully works and is exercised by `TestScriptedLLM_Capabilities` line
  68–76).

### Requirements Coverage (16 Phase-0 requirements)

| Requirement   | Description                                                                              | Status       | Evidence                                                                                                                                                                                                                                                                                  |
| ------------- | ---------------------------------------------------------------------------------------- | ------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **INFRA-01**  | 4 sibling Go modules exist with their own `go.mod`                                       | ✓ SATISFIED  | Core: `go.mod` declares `module github.com/costa92/llm-agent`. 3 sister repos verified via `gh repo view`: `llm-agent-providers`, `llm-agent-otel`, `llm-agent-customer-support` (all PUBLIC). Each contains its own `go.mod` per 00-03 SUMMARY §"Files in Each Repo's Initial Commit".  |
| **INFRA-02**  | `go.work` is `.gitignore`d in every repo; CI runs with `GOWORK=off`                      | ✓ SATISFIED  | Core: `.gitignore:21–22` (`go.work`, `go.work.sum`); `.github/workflows/test.yml:14` (workflow-level `GOWORK: off`). Sister repos: same pattern per 00-03 SUMMARY §"Files in Each Repo's Initial Commit". `git check-ignore -q go.work` returns 0 in core (tested in 00-04 SUMMARY).      |
| **INFRA-03**  | A `Makefile` or shell script in each sister repo writes a sibling-aware `go.work`        | ✓ SATISFIED  | `scripts/workspace.sh` (40 LOC, 100755). SHA256 `8eda10c3e7a337a5551eef68d43732d71533663f0aaa66e1c0c729be796a09ec` byte-identical across all 4 repos (asserted in `umbrella.yml:52–69`).                                                                                                  |
| **INFRA-04**  | CI gate rejects `replace` directives on tagged-release branches                          | ✓ SATISFIED  | `.github/workflows/release-precheck.yml` (37 LOC): triggers on `push` + `pull_request` to `release/**`; parses `go mod edit -json` for `Replace`; fails on count > 0. Byte-identical SHA256 (`2b507c8804852fb4cf82f40dabb159daab3ebb3838d6352821b825be3e16a96c`) shipped to all 3 sisters. |
| **INFRA-05**  | Umbrella CI in `llm-agent` builds all 4 repos against `llm-agent` HEAD on every PR       | ✓ SATISFIED  | `.github/workflows/umbrella.yml` (104 LOC): triggers on `pull_request: branches: [main]` + `workflow_dispatch`; checks out all 4 repos via `actions/checkout@v4`; runs `go work init` to point all 4 modules at this PR's `llm-agent`; runs `go vet/build/test` per module.              |
| **INFRA-06**  | README in each sister repo documents the cross-repo iteration pattern                    | ✓ SATISFIED  | 00-03 SUMMARY §"Files in Each Repo's Initial Commit" confirms `README.md` shipped to each sister with "INFRA-06 cross-repo pattern". `gh api` spot checks in 00-03 confirmed README content (e.g., customer-support has `K8s manifests are NOT part of v0.3` banner per Pitfall 16).      |
| **INFRA-07**  | Versioning policy documented across all 4 repos                                          | ✓ SATISFIED  | `CHANGELOG.md` (line 71+) `### Versioning policy (INFRA-07)` subsection with 4-repo table (`llm-agent` v0.3.x; sister repos v0.1.x; CHANGELOG `### Breaking` per repo). Sister repos document the same policy per 00-02 / 00-03 SUMMARYs.                                                |
| **CORE-01**   | `llm/v2` package defines `ChatModel` base interface (Generate + Stream)                  | ✓ SATISFIED  | `llm/chatmodel.go:17–21`: 3-method interface (Generate + Stream + Info). Per D-01, the package path is `llm/` (not `llm/v2/`) — same import path. Concurrent-use contract documented (line 16).                                                                                          |
| **CORE-02**   | `ToolCaller` defines `WithTools(tools) ToolCaller` (immutable)                           | ✓ SATISFIED  | `llm/capabilities.go:13–16`: `ToolCaller` interface declares `WithTools(tools []Tool) (ToolCaller, error)` returning a new value. Implementation in `llm/scripted.go:103–122` constructs new `&ScriptedLLM{...}`. `TestToolCallerImmutable` asserts `a != b` and concurrent safety.       |
| **CORE-03**   | `Embedder` defines `Embed(ctx, []string) ([]Vector, Usage, error)` — separate interface  | ✓ SATISFIED  | `llm/capabilities.go:27–30`: `Embedder` interface declares `Embed(ctx, texts) (vectors []Vector, usage Usage, err error)` + `EmbedDimensions() int`. Embedder deliberately does NOT embed ChatModel (line 26 documents the orthogonality decision).                                       |
| **CORE-04**   | `StructuredOutputs` defines `WithSchema(schema) ChatModel`                               | ✓ SATISFIED  | `llm/capabilities.go:39–42`: `StructuredOutputs` interface declares `WithSchema(schema []byte) (ChatModel, error)`. Returns ChatModel (not StructuredOutputs) per RESEARCH §"WithSchema returns ChatModel" rationale.                                                                     |
| **CORE-05**   | Typed `StreamEvent` union with `Kind` enum + stable per-tool-call `Index`                | ✓ SATISFIED  | `llm/stream.go:22–31`: `StreamEventKind uint8` with 6 variants. `llm/stream.go:41–47`: `StreamEvent` typed-union struct. `llm/stream.go:65–70`: `ToolCallDelta.Index int` documented as "stable across chunks for a single tool call" (Pitfall 1). `TestStreamEventKind_Variants` (`llm/llm_test.go:199`) asserts iota ordering. |
| **CORE-06**   | `ProviderInfo` returned by `ChatModel.Info()` reflects bound model's capabilities (per-(provider × model)) | ✓ SATISFIED  | `llm/info.go:8–12`: `ProviderInfo` struct binds Provider + Model + Capabilities. `llm/chatmodel.go:20`: `ChatModel.Info() ProviderInfo` method. Construction-time binding documented in `llm/info.go:4–7` ("Provider instances bind a model at construction time"). D-02 ratified shape. |
| **CORE-07**   | Mock implementations (`ScriptedLLM`-style) for `ChatModel` + each capability             | ✓ SATISFIED  | `llm/scripted.go:43–48`: 4 compile-time `var _` assertions (ChatModel, ToolCaller, Embedder, StructuredOutputs). `llm/chat_only_mock.go:18`: ChatModel-only `var _` assertion. `TestChatOnlyMockExcludesCapabilities` (`llm/llm_test.go:28`) asserts negative claims.                     |
| **CORE-08**   | Existing `llm.Client` (v0.2 surface) remains callable; marked Deprecated with godoc + removal target | ✓ SATISFIED  | `llm/legacy.go:8–11`: `LegacyClient` interface (renamed via git mv). `llm/legacy.go:16`: `type Client = LegacyClient` alias preserves all v0.2 callers' `llm.Client` references. 6 `// Deprecated:` comments name v0.4.0 + migration guide. `TestLegacyClientAlias` asserts compile-time. |
| **CORE-09**   | Migration guide in `docs/migration-v0.2-to-v0.3.md` — concrete diff examples for each agent paradigm | ✓ SATISFIED  | `docs/migration-v0.2-to-v0.3.md` (207 LOC). Quick reference table (13 rows). Simple paradigm worked example (3 variants: v0.2 / v0.3 transitional / v0.3 idiomatic) covers the canonical diff. Other 4 paradigms covered by the type-rename mapping table per CONTEXT.md Claude's Discretion. |

**Score: 16/16 requirements satisfied.**

No requirements were declared in any plan's `requirements:` field that
appear orphaned. REQUIREMENTS.md `## Traceability` table maps INFRA-01..07
+ CORE-01..09 to Phase 0 — all 16 are accounted for above with concrete
artifacts. CORE-10 (agent paradigm refactor) and CORE-11 (Provider Author
Guide) are explicitly mapped to Phase 3 and Phase 1 respectively in
REQUIREMENTS.md and are NOT Phase-0 scope.

### Build / Test / Stdlib Invariant

| Check                                                          | Command                                            | Result        | Status   |
| -------------------------------------------------------------- | -------------------------------------------------- | ------------- | -------- |
| `go vet ./...` green                                           | `go vet ./...`                                     | exit 0        | ✓ PASS   |
| `go build ./...` green                                         | `go build ./...`                                   | exit 0        | ✓ PASS   |
| `go test ./... -count=1` green across 15 packages              | `go test ./... -count=1`                           | 15 ok, 0 FAIL | ✓ PASS   |
| Core go.mod is stdlib-only (no `require` block)                | `grep -c '^require' go.mod`                        | 0             | ✓ PASS   |

### Pitfall 22 Baseline

| Check                                                          | Command                                            | Result          | Status   |
| -------------------------------------------------------------- | -------------------------------------------------- | --------------- | -------- |
| `docs/api-snapshot.txt` exists, non-empty                      | `test -s docs/api-snapshot.txt`                    | exit 0          | ✓ PASS   |
| `docs/api-snapshot.txt` line count                             | `wc -l docs/api-snapshot.txt`                      | 3495 lines      | ✓ PASS   |
| Snapshot contains all keystone types                           | `grep -E "ChatModel\|ToolCaller\|Embedder\|StructuredOutputs\|StreamEvent\|StreamReader\|ProviderInfo" docs/api-snapshot.txt` | matches at lines 1676+ | ✓ PASS   |
| Snapshot tracked by git                                        | `git ls-files docs/api-snapshot.txt`               | 1 path          | ✓ PASS   |

### Out-of-band Remaining

These items are explicitly documented in plan SUMMARYs as out-of-band /
deferred. They are NOT Phase-0 code gaps; they are human-actionable
follow-ups that do not block Phase 1.

| Item                                                    | Source                                                      | Owner        | Notes                                                                                                                                              |
| ------------------------------------------------------- | ----------------------------------------------------------- | ------------ | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| `git tag v0.3.0-pre.1` on core repo                     | 00-05 SUMMARY §"Out-of-band"; RESEARCH Q3 RESOLVED          | user (gsd)   | Single-command. Once pushed, sister-repo CI flips from RED → GREEN automatically. Required for Phase 1 sister-repo work to start.                 |
| Branch protection on `main` for 3 sister repos          | 00-03 SUMMARY §"Task 4 Awaiting (Branch Protection)"        | user         | GitHub Settings UI for each repo: protect `main`, require `test / go` status check. Documented as explicit human-verify gate.                      |
| Live smoke-test of `umbrella.yml` + `release-precheck.yml` | 00-05 SUMMARY §"Smoke-test deferral"                       | user         | User opted to defer live verification of CI logic. The bash logic in the YAML files is correct by inspection; first PR after Phase 0 close will exercise umbrella naturally. |
| First sister-repo CI run currently RED                  | 00-03 SUMMARY §"First CI Run Status"                        | n/a (auto)   | Expected and intentional. Resolves automatically when `v0.3.0-pre.1` tag is pushed.                                                                |
| `.planning/STATE.md` update with Phase 0 completion     | implicit per gsd workflow                                   | gsd / user   | Tracked in `/gsd-transition` workflow, not in this verification.                                                                                  |

### Human Verification Required

None at this verification level. All Phase-0 truths and requirements were
verifiable programmatically (file existence, grep patterns, `go vet/build/test`,
godoc inspection, `gh repo view`).

The out-of-band items above ARE human-actionable, but they are explicitly
documented as deferred / out-of-plan in the SUMMARYs and do not affect the
Phase-0 PASS verdict. They are surfaced for the user's awareness, not as
verification gaps.

### Gaps Summary

No gaps. All 4 keystones implemented, all 4 decisions honored, all 16
requirements satisfied, all artifacts present, all key links wired, build /
test / stdlib invariants intact, Pitfall 22 baseline captured.

The two known follow-ups (`v0.3.0-pre.1` tag and sister-repo branch
protection / live CI smoke-test) are explicitly out-of-band per plan
SUMMARYs and acceptable for a solo side-project. They do not block Phase 1
from starting.

---

## Ready for Phase 1: YES

Phase 0 has delivered the K1 / K2 / K3 / K6 keystones in code, locked the
multi-repo CI discipline, and produced the migration / deprecation /
versioning documentation. Phase 1 (three-provider walking skeleton —
Generate sync only) can begin against the locked surface in `llm/`.

Recommended Phase-0 → Phase-1 transition steps (out-of-band, user-driven):

1. Push `git tag v0.3.0-pre.1` on the core repo so sister-repo CI goes
   green.
2. Configure branch protection on `main` for the 3 sister repos.
3. Run `/gsd-transition` to mark Phase 0 complete in `STATE.md` /
   `ROADMAP.md` progress table and to update `PROJECT.md` per the K8s
   cleanup flag (Conflict D resolution per ROADMAP.md §"PROJECT.md Cleanup
   Flag").

---

_Verified: 2026-05-10_
_Verifier: Claude (gsd-verifier) — goal-backward verification per
`.claude/get-shit-done/references/gates.md` Escalation Gate pattern_

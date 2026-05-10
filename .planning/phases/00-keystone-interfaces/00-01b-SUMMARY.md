---
phase: 00-keystone-interfaces
plan: 01b
subsystem: llm
tags:
  - llm
  - mocks
  - testing
  - capability-degradation
  - go-stdlib-only
dependency_graph:
  requires:
    - plan 00-01a (ChatModel + ToolCaller + Embedder + StructuredOutputs + errors + types + legacy)
  provides:
    - llm.ScriptedLLM (full-capability deterministic mock, production code)
    - llm.ChatOnlyMock (ChatModel-only mock for capability-degradation testing)
    - llm.NewScriptedLLM + ScriptedOption functional options
    - llm.TextResponse / llm.ToolCallResponse convenience constructors
    - llm.WithProvider / WithModel / WithCapabilities / WithResponses / WithEmbedDimensions options
    - llm/doc.go canonical capability-negotiation idiom + all v0.3 types documented
    - llm/llm_test.go: 8 tests covering interface satisfaction, JSON round-trips, sentinel errors, concurrency
    - scriptedllm_test.go: thin v0.2-shim preserving agent paradigm test compatibility
  affects:
    - all agent paradigm tests (via scriptedllm_test.go shim — no source change needed)
    - sister-repo conformance suites (Phase 1) — can now import llm.ScriptedLLM
tech_stack:
  added: []
  patterns:
    - stdlib-only (sync, context, encoding/json, errors, fmt, io)
    - functional options pattern (ScriptedOption func(*ScriptedLLM))
    - immutable capability constructors (WithTools/WithSchema field-by-field copy to satisfy go vet)
    - compile-time var _ assertions in production code (godoc-visible)
    - iterator-style scriptedStream (Next/Close idempotent; closed bool guarded by sync.Mutex)
    - v0.2-shim pattern: thin wrapper aliases sentinel, preserves legacy interface
key_files:
  created:
    - llm/scripted.go        # 256 LOC — ScriptedLLM v2: ChatModel+ToolCaller+Embedder+StructuredOutputs
    - llm/chat_only_mock.go  # 34 LOC  — ChatOnlyMock: ChatModel ONLY (capability-degradation tests)
    - llm/llm_test.go        # 243 LOC — 8 tests: interface satisfaction, JSON, concurrency, sentinel errors
  modified:
    - llm/doc.go             # 62 LOC  — replaced v0.2 doc with v0.3 capability-negotiation overview
    - scriptedllm_test.go    # 77 LOC  — rewritten as thin v0.2-shim delegating to llm.ErrScriptExhausted
decisions:
  - "go vet flags struct copy when value contains sync.Mutex — fixed WithTools/WithSchema to use field-by-field copy instead of cp := *s"
  - "scriptedllm_test.go kept as wrapper type (not type alias) because v0.2/v0.3 method signatures differ (chan vs StreamReader)"
  - "ScriptedLLM and ChatOnlyMock promoted to production code (not _test.go) so sister-repo conformance suites can import them"
metrics:
  duration: ~20min
  completed: 2026-05-10
  tasks_completed: 4
  files_created: 3
  files_modified: 2
---

# Phase 0 Plan 01b: llm/ Mock + Test + Docs Summary

**One-liner:** ScriptedLLM v2 (full-capability deterministic mock) + ChatOnlyMock (capability-degraded mock) promoted to production `llm/` package with functional options, compile-time var _ assertions, 8-test suite covering interface satisfaction/JSON/concurrency/sentinels, and a v0.2-shim that keeps all agent paradigm tests green — stdlib-only throughout.

## Tasks Completed

| # | Name | Commit | Key Files |
|---|------|--------|-----------|
| 4 | Create llm/scripted.go + llm/chat_only_mock.go | eec4ac0 | llm/scripted.go, llm/chat_only_mock.go |
| 5 | Replace llm/doc.go with new package overview | 27ce516 | llm/doc.go |
| 6 | Create llm/llm_test.go (8 interface satisfaction + capability tests) | ad13c48 | llm/llm_test.go |
| 7 | Rewrite scriptedllm_test.go as thin v0.2-shim | 641d411 | scriptedllm_test.go |

## Verification Results

- `go vet ./...` — PASS
- `go build ./...` — PASS
- `go test ./... -count=1 -race` — PASS (15 packages, all green; 8 new tests in llm/)
- `go test ./llm/... -count=1 -race -v` — PASS (all 8 tests listed)
- `grep -c '^require' go.mod` — 0 (stdlib-only invariant intact)
- `go doc ./llm/` — renders full capability-negotiation overview with all types

## File Inventory

| File | LOC | Provides |
|------|-----|---------|
| llm/scripted.go | 256 | ScriptedLLM: Generate/Stream/Info/WithTools/Embed/EmbedDimensions/WithSchema; ScriptedOption functional options; TextResponse/ToolCallResponse helpers; scriptedStream (EventTextDelta→EventDone→io.EOF); compile-time var _ for all 4 capability interfaces |
| llm/chat_only_mock.go | 34 | ChatOnlyMock: Generate/Stream/Info; compile-time var _ ChatModel only; delegates Stream to newScriptedStream |
| llm/doc.go | 62 | Package overview: all 16 exported types documented; canonical capability-negotiation idiom; streaming section; deprecation section |
| llm/llm_test.go | 243 | TestLegacyClientAlias, TestChatOnlyMockExcludesCapabilities, TestScriptedLLM_Capabilities, TestToolCallerImmutable, TestStreamReaderClosesIdempotent, TestSentinelErrors_ErrorsIs, TestStreamEventKind_Variants, TestProviderInfo_JSONRoundTrip |
| scriptedllm_test.go | 77 | Thin v0.2 shim: errScriptExhausted = llm.ErrScriptExhausted; newScriptedLLM/textResp/callCount preserved; var _ llm.Client assertion |
| **Total** | **672** | |

## go.mod Confirmation

`go.mod` was NOT modified. Content after this plan:
```
module github.com/costa92/llm-agent

go 1.26.0
```
No `require` block. Stdlib-only invariant intact.

## Test Results (llm package, -race)

```
=== RUN   TestLegacyClientAlias           --- PASS
=== RUN   TestChatOnlyMockExcludesCapabilities --- PASS
=== RUN   TestScriptedLLM_Capabilities    --- PASS
=== RUN   TestToolCallerImmutable         --- PASS
=== RUN   TestStreamReaderClosesIdempotent --- PASS
=== RUN   TestSentinelErrors_ErrorsIs     --- PASS
=== RUN   TestStreamEventKind_Variants    --- PASS
=== RUN   TestProviderInfo_JSONRoundTrip  --- PASS
PASS ok github.com/costa92/llm-agent/llm
```

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] go vet: struct copy of sync.Mutex value in WithTools and WithSchema**
- **Found during:** Task 4 verification
- **Issue:** `cp := *s` copies the entire `ScriptedLLM` struct including its `sync.Mutex` field — `go vet` correctly rejects this
- **Fix:** Replaced struct copy with field-by-field extraction under the lock, then constructed a new `&ScriptedLLM{...}` without the mutex field (it gets a fresh zero-value mutex automatically)
- **Files modified:** llm/scripted.go (WithTools and WithSchema methods)
- **Commit:** eec4ac0 (incorporated into the task commit)

**2. [Observation] Verify script case mismatch: `grep -q 'capability negotiation'` vs actual `# Capability negotiation`**
- **Found during:** Task 5 verification
- **Issue:** Plan's automated verify uses lowercase `capability negotiation` but the action body specifies `# Capability negotiation` (uppercase C, Go section heading convention)
- **Resolution:** File content matches the plan's `<action>` body exactly; the verify script has a case-sensitivity mismatch. Content is correct. The done criteria (`go vet` passes, all types documented, canonical idiom present) are fully met.

## Known Stubs

None. All 4 files contain fully implemented production code. ScriptedLLM's WithSchema is documented as a no-op (schema not validated), which is intentional and documented in the godoc.

## Threat Surface Scan

No new network endpoints, auth paths, file access patterns, or schema changes. All files are pure Go type declarations/implementations within the `llm` package and test helpers within `package agents`. The only new surface is:

- `llm.ScriptedLLM` and `llm.ChatOnlyMock` promoted from test-only to production code — intentional per D-03 so sister repos can import them for conformance suites.

No threat flags beyond those already in the plan's STRIDE register (T-00-01b-01 through T-00-01b-05, all mitigated by the test suite).

## What Comes Next (Phase 0 completion)

Phase 0 K1/K2/K3 deliverable is now complete on the core-repo side:
- K1 (typed streaming events) — StreamEvent + StreamEventKind (plan 00-01a)
- K2 (per-model capabilities) — ProviderInfo + Capabilities (plan 00-01a)
- Contract surface: ChatModel + ToolCaller + Embedder + StructuredOutputs (plan 00-01a)
- Mocks + tests + docs: ScriptedLLM v2 + ChatOnlyMock + llm_test.go + doc.go (this plan)

Next: Plan 00-02 (sister repo scaffolding), or as directed by the GSD orchestrator.

## Self-Check: PASSED

Files verified:
- FOUND: llm/scripted.go
- FOUND: llm/chat_only_mock.go
- FOUND: llm/doc.go
- FOUND: llm/llm_test.go
- FOUND: scriptedllm_test.go

Commits verified:
- FOUND: eec4ac0 (Task 4 — scripted.go + chat_only_mock.go)
- FOUND: 27ce516 (Task 5 — doc.go replacement)
- FOUND: ad13c48 (Task 6 — llm_test.go)
- FOUND: 641d411 (Task 7 — scriptedllm_test.go shim)

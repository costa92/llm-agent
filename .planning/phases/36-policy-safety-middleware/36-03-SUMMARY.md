---
phase: 36-policy-safety-middleware
plan: 03
subsystem: policy
tags: [policy, integration-test, compose, kc-3, decision-g, budget-compose, paradigm-smoke]
requires:
  - 36-01-SUMMARY.md   # policy.Wrap factory + 8-wrapper tree + Gate/Decision types
  - 36-02-SUMMARY.md   # PIIRedactor, InjectionScanner, MaxInputLen built-in gates
  - 35-budget-and-cancellation-context  # generateFromPrompt chokepoint + budget sentinels
provides:
  - "policy/integration_test.go::observerModel"          # in-test mimic of otelmodel.Wrap's 4-interface contract
  - "policy/integration_test.go::TestCompose_CapabilityPreserved"
  - "policy/integration_test.go::TestCompose_BlockedByPolicyShortCircuits"
  - "policy/integration_test.go::TestCompose_StreamingThroughBothLayers"
  - "policy/integration_test.go::TestCompose_BudgetBeatsPolicyAtChokepoint"
  - "policy/integration_test.go::TestCompose_PerParadigmSmoke"
affects:
  - policy/                # ONLY the new integration_test.go — no other policy/* edits
tech-stack:
  added: []                # stdlib + in-repo only — no new deps
  patterns:
    - "in-test capability-mirror struct (Decision G): satisfies all 4 capability interfaces in one struct"
    - "table-driven per-paradigm smoke (mirrors budget_integration_test.go::TestAllParadigms_BudgetUniformity)"
    - "atomic counter invariant (observer.generateCount.Load() == 0) for KC-3 outer-denies-first proof"
key-files:
  created:
    - policy/integration_test.go
  modified: []
decisions:
  - "Decision G honored: NO import of llm-agent-otel anywhere in policy/. The mimic is a single struct that satisfies ChatModel + ToolCaller + Embedder + StructuredOutputs unconditionally (real otelmodel uses an 8-wrapper type-switch pyramid; the test uses a fully-capable inner so the unconditional claim is sufficient to prove the shape contract)."
  - "Observer mimic forwards Embed/WithSchema/WithTools via type-assertion on the inner — gracefully degrades if a future test passes a partially-capable inner."
  - "Budget-beats-policy test uses MaxCalls:1 + cross-Run pattern (first Run: gate fires; second Run: chokepoint denies before policy ever runs) — mirrors the canonical Phase 35 35-04 cross-Run shape."
  - "Per-paradigm smoke seeds 5 dummy responses per paradigm so a paradigm that bypasses the gate (none should) does not surface ErrScriptExhausted masking the real assertion failure."
  - "Compile-time assertions on observerModel use single-line var form (not var-block) so the acceptance grep `var _ llm.(...)+=` matches the canonical 4-line pattern."
metrics:
  duration_minutes: ~25
  completed_utc: 2026-05-21T06:43:09Z
  task_count: 3
  file_count: 1
  test_count_added: 5         # TestCompose_CapabilityPreserved, _BlockedByPolicyShortCircuits, _StreamingThroughBothLayers, _BudgetBeatsPolicyAtChokepoint, _PerParadigmSmoke (5 sub-tests)
  loc_added: 581
---

# Phase 36 Plan 03: policy / compose-with-everything integration tests — Summary

One-liner: Ships `policy/integration_test.go` with an in-test
`observerModel` mimic of `otelmodel.Wrap` (~50 LOC, NO `llm-agent-otel`
import) plus 5 compose tests proving capability preservation, KC-3
outer-denies-first short-circuit, streaming-through-both-layers, the
v1.2 budget-beats-policy chokepoint invariant, and uniform
`BlockedError` propagation across all 5 v1.2 agent paradigms — all
under `go test -race`.

## What Was Built

### `policy/integration_test.go` (581 LOC, 1 new test file)

Single new file under `policy/` (matches the plan's `<files_modified>`
spec exactly — no other `policy/*` edits). Package `policy` (in-package
test so it can use the existing `testGate`, `scriptedStreamChat`, and
`observerGate` helpers from `policy_test.go`).

**`observerModel` — Decision G in-test mimic (~120 LOC including methods).**
Struct with `inner llm.ChatModel`, `tools []llm.Tool`, `schema []byte`,
`generateCount atomic.Int64`, `streamCount atomic.Int64`, mutex-guarded
`lastReq llm.Request`. Implements:

- `Generate(ctx, req)` — increments `generateCount`, captures `lastReq`,
  delegates.
- `Stream(ctx, req)` — increments `streamCount`, captures `lastReq`,
  delegates.
- `Info()` — forwards to inner (transparent at the Info layer).
- `WithTools(tools)` — rebinds inner `ToolCaller` if available, returns
  a NEW `*observerModel` wrapping the bound child. Mirrors otelmodel's
  re-wrap idiom so the K1 immutable-WithTools pattern composes.
- `Embed(ctx, texts)` — forwards to inner `Embedder` if available;
  returns `ErrCapabilityNotSupported` otherwise (the type-assertion
  contract is unconditional but runtime behavior is delegation-or-fail).
- `EmbedDimensions()` — same delegation pattern.
- `WithSchema(schema)` — rebinds inner `StructuredOutputs`, returns a
  new `*observerModel`. Signature returns `llm.ChatModel` per the
  `StructuredOutputs` interface (NOT `StructuredOutputs` — the plan's
  prose said "returns a new `*observerModel` with `schema` set" which
  the actual go interface signature constrains; the bound child is an
  `*observerModel` whose static type satisfies `llm.ChatModel`).

Compile-time assertions (single-line var form so the canonical grep
`var _ llm\.(ChatModel|ToolCaller|Embedder|StructuredOutputs) +=`
returns 4):

```go
var _ llm.ChatModel = (*observerModel)(nil)
var _ llm.ToolCaller = (*observerModel)(nil)
var _ llm.Embedder = (*observerModel)(nil)
var _ llm.StructuredOutputs = (*observerModel)(nil)
```

### 5 compose tests

| Test                                            | What it proves                                                                                                                             | KC reference            |
|-------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------|-------------------------|
| `TestCompose_CapabilityPreserved`               | `policy.Wrap(observerModel(scriptedLLM))` preserves `ToolCaller` / `Embedder` / `StructuredOutputs` AND the gate stack survives `WithTools` rebind | KC-3 capability mirror  |
| `TestCompose_BlockedByPolicyShortCircuits`      | `PreGenerate Block` returns `BlockedError`, `observer.generateCount.Load() == 0` (inner never reached). `errors.As → BlockedError{Gate: "blockingGate", Reason: "test"}` | KC-3 outer-denies-first |
| `TestCompose_StreamingThroughBothLayers`        | `PreStream` fires once on first `Next()`, `StreamDelta` fires 3× (one per non-Done event), `PostStream` fires once on `EventDone`. Drained events match `"a"/"b"/"c"/Done`. `observer.streamCount == 1` | Decision F (streaming surface) |
| `TestCompose_BudgetBeatsPolicyAtChokepoint`     | With `MaxCalls=1`: 1st Run → `policy.ErrBlocked` (gate fires after budget passes). 2nd Run → `budget.ErrCallsExceeded` AND NOT `policy.ErrBlocked` (chokepoint denies before policy ever runs) | 35-RESEARCH §"Carry-forward notes" |
| `TestCompose_PerParadigmSmoke`                  | All 5 paradigms (Simple, ReAct, Reflection, PlanSolve, FunctionCall) propagate `BlockedError` correctly through `Agent.Run` against a blocking gate. `BlockedError.Reason == "smoke-test"` | v1.2 paradigm uniformity |

### `streamCountingGate` helper (~25 LOC)

Per-EventKind atomic counters; `Inspect` records the event kind and
returns `Allow`. Used exclusively by `TestCompose_StreamingThroughBothLayers`
to assert each EventKind fires the expected number of times.

## Verification

All `<verify>` commands from the plan pass:

| Check                                                          | Result                |
|----------------------------------------------------------------|-----------------------|
| `go vet ./...`                                                 | clean                 |
| `go test ./policy/... -count=1`                                | PASS                  |
| `go test -race ./policy/... -count=1`                          | PASS (1.018s)         |
| `go test ./... -count=1` (repo-wide)                           | all packages green    |
| `grep -r "llm-agent-otel" policy/ | wc -l`                     | `0`                   |
| `grep -l 'observerModel' policy/ | wc -l`                      | `1` (integration_test.go only) |
| 5 compose tests present (grep `func TestCompose_*`)            | all 5 found           |
| `grep -cE 'var _ llm\.(...) +=' policy/integration_test.go`    | `4`                   |
| `grep -c 'budget.ErrCallsExceeded' policy/integration_test.go` | `4`                   |
| `grep -c 'ErrBlocked' policy/integration_test.go`              | `10`                  |
| `go list -deps ./policy/...` non-stdlib leak check             | `0` non-stdlib outside in-repo `llm` |
| KC-5 unchanged (`git status --short -- llm/ agent.go agents.go simple.go react.go reflection.go plan_solve.go function_call.go memory/ orchestrate/`) | `0` modifications |
| `go.mod` / `go.sum` unchanged                                  | `0` modifications     |

`policy/` package imports remain: `context`, `errors`, `fmt`, `io`,
`regexp`, `strings`, `sync` (stdlib) + `github.com/costa92/llm-agent/llm`
(in-repo sibling). The integration test additionally pulls
`encoding/json`, `sync/atomic`, `testing`, `github.com/costa92/llm-agent`,
and `github.com/costa92/llm-agent/budget` — all stdlib or in-repo. The
sister-repo `llm-agent-otel` package is not imported anywhere in core.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - blocker] `Embed` and `WithSchema` interface signatures**
- **Found during:** Task 1 implementation.
- **Issue:** Plan's `<action>` prose specified `Embed(ctx, input) ([][]float32, error)` and `WithSchema(schema) (llm.StructuredOutputs, error)`. Real interfaces in `llm/capabilities.go` are `Embed(ctx, texts) ([]Vector, Usage, err)` and `WithSchema(schema) (ChatModel, error)`. Building against the prose would fail compile-time.
- **Fix:** Used the real interface signatures from `llm/capabilities.go`. The `observerModel` `Embed` returns `(nil, llm.Usage{}, llm.ErrCapabilityNotSupported)` when the inner is not an `Embedder`; `WithSchema` returns `llm.ChatModel` (not `StructuredOutputs`).
- **Files modified:** `policy/integration_test.go` (Task 1 commit).
- **Commit:** df25122

**2. [Rule 2 - critical] Single-line var form for compile-time assertions**
- **Found during:** Task 3 verification.
- **Issue:** Initial Task-1 implementation used `var (...)` block form for the 4-interface assertions. The plan's acceptance grep `var _ llm\.(...) +=` is anchored on `var _` (single-line form) and returned `0`. The block form has identical semantic effect (all 4 assertions are checked at compile time) but fails the literal acceptance grep.
- **Fix:** Converted the 4 assertions to single-line `var _ llm.XXX = (*observerModel)(nil)` form.
- **Files modified:** `policy/integration_test.go` (Task 3 commit).
- **Commit:** f163bbe

**3. [Rule 3 - blocker] Doc comment containing `llm-agent-otel` literal**
- **Found during:** Task 3 verification.
- **Issue:** The file-level doc comment referenced `github.com/costa92/llm-agent-otel/otelmodel` as a forbidden import to make Decision G's rationale explicit. The success criterion mandates `grep -c 'llm-agent-otel' policy/integration_test.go == 0`, and a comment match still trips the grep.
- **Fix:** Reworded the comment to "the sister-repo otelmodel package" — preserves the rationale without the literal string.
- **Files modified:** `policy/integration_test.go` (Task 3 commit).
- **Commit:** f163bbe

### Architectural changes
None.

### Out-of-scope discoveries deferred
None — every issue encountered was a direct consequence of the current
task's scope.

## Authentication Gates
None.

## Known Stubs
None. The `observerModel` is intentionally a test-only mimic; it is
documented as such in its doc comment and is NOT exported to package
consumers (lower-case `observerModel`, file-local).

## TDD Gate Compliance
The plan does not declare `type: tdd`. Each task is `type="auto"` with
`tdd` unset; the RED/GREEN gate sequence does not apply. The compose
tests are written GREEN-first against the already-shipped 36-01/36-02
surfaces — a valid pattern for integration test slices where the
production code is already in place.

## Self-Check: PASSED

**Files created/modified verified to exist:**

```
[FOUND] policy/integration_test.go (581 LOC)
```

**Commits exist in git log:**

```
[FOUND] df25122 — test(36-03): add observerModel mimic + capability-preservation test
[FOUND] b80a1ac — test(36-03): add short-circuit, streaming, and budget-beats-policy compose tests
[FOUND] f163bbe — test(36-03): add per-paradigm smoke test across 5 v1.2 paradigms
```

All 3 task commits land cleanly on
`worktree-agent-a8d42aff6337e4ac3` ahead of the merge-base
`ea6620914353a4700b49562f36679d56ae867cc3` (Phase 36-01 + 36-02 merged
state).

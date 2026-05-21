---
phase: 35-budget-and-cancellation-context
plan: 02
status: paused-at-push-gate
completed_at: 2026-05-20
repo: llm-agent (core)
requirements: [CC-1]
depends_on: ["35-01"]
files_modified:
  - agent_chatmodel.go
  - agent_chatmodel_test.go
artifacts:
  - "llm-agent local commit d141bf6 on main"
  - "PUSH PENDING: git push origin main — operator-gated"
---

# 35-02 — chokepoint wiring of `budget` into `generateFromPrompt` (CC-1 / Phase 35 Wave 2, paused at push gate)

Wave 2 of Phase 35. Surgical body-only edit of
`agent_chatmodel.go::generateFromPrompt` — the one helper every agent
paradigm (Simple, ReAct, Reflection, PlanSolve, FunctionCall) × 10 call
sites funnels through. Pre-call `Tracker.Charge(Usage{Calls: 1})` and
post-call `Tracker.Charge(Usage{Tokens: resp.Usage.TotalTokens})` against
the ctx-keyed tracker installed by Wave 1's `budget.WithBudget`. Function
signature byte-identical to pre-slice (KC-5). Zero paradigm files
touched. New test file `agent_chatmodel_test.go` (278 lines, 5 tests)
proves: no-budget passthrough, pre-call deny short-circuits the network,
post-call deny returns BOTH `(resp, sentinel)`, MaxWall fires via
`context.DeadlineExceeded`, concurrent enforcement is race-clean.

## Outcome

- `agent_chatmodel.go` grew from 35 → 69 lines (`generateFromPrompt`
  body went from 8 → 42 lines including comments).
- `agent_chatmodel_test.go` is new: 278 lines, 5 tests, all green
  under `-race`.
- All 8 `<verify>` gates green: SIGNATURE-OK, BUDGET-IMPORTED, Charge
  count=2, Q2-COMMENT-OK, COST-DEFER-COMMENT-OK, vet+build+test green,
  go.mod/go.sum unchanged, no paradigm files edited.
- Repo-wide `go test ./...` green (16 packages, 0 failures).
- Stdlib-only invariant preserved — only new import is
  `github.com/costa92/llm-agent/budget` (intra-module; not a new dep).
- Single local commit `d141bf6` on `main`; not yet pushed.

## Diff summary — `agent_chatmodel.go`

| Region                                                  | Lines (after) | Change |
| ------------------------------------------------------- | ------------- | ------ |
| Import block                                            | 3–9           | Added `"github.com/costa92/llm-agent/budget"` (alphabetized above `llm`). |
| `generateFromPrompt` signature (line 11)                | 11            | **Unchanged** (byte-identical to pre-slice; verified by `grep`). |
| Request build (lines 12–17)                             | 12–17         | Unchanged. |
| **Pre-call charge** (NEW, lines 19–28)                  | 19–28         | `t, hasBudget := budget.From(ctx)`; if `hasBudget` then `t.Charge(budget.Usage{Calls: 1})` — return zero `llm.Response{}` and the sentinel on deny. |
| `model.Generate` call (NEW form, line 30)               | 30            | Split former `return model.Generate(ctx, req)` into explicit `resp, err := ...` to allow post-call branching. |
| Error short-circuit (lines 31–35)                       | 31–35         | If `err != nil` return `(resp, err)` — `ctx.Err()` from MaxWall surfaces here. |
| **Post-call charge** (NEW, lines 37–48)                 | 37–48         | If `hasBudget`, `t.Charge(budget.Usage{Tokens: resp.Usage.TotalTokens})` — on deny return BOTH `(resp, cerr)` per Decision 3. Inline comment documents v1.2 → v1.3 Cost-field gap. |
| Final return (line 50)                                  | 50            | `return resp, nil`. |
| `nativeToolCaller` + `toolCapabilityError` (lines 54–68)| 54–68         | Unchanged. |

Net: chokepoint body 8 → 42 lines (+34); signature line untouched.

## New file — `agent_chatmodel_test.go` (278 lines, package `agents`)

| Test                                                  | Lines | Asserts |
| ----------------------------------------------------- | ----: | ------- |
| `tokenResp` helper                                    |    11 | Builds an `llm.Response` with `Usage.TotalTokens` populated — canonical `textResp` (scriptedllm_test.go) leaves Usage empty. |
| `slowScriptedLLM` wrapper                             |    21 | Test-local `llm.ChatModel` shim that sleeps `delay` before delegating; honors `ctx.Done()` (mirrors real provider HTTP-client behavior). |
| `TestGenerateFromPrompt_NoBudget_Passthrough`         |    30 | 5 calls on a budget-less ctx; every response byte-identical to scripted; callCount==5; `budget.From(ctx)` returns `(nil, false)`. Proves zero behavior change. |
| `TestGenerateFromPrompt_MaxCalls_PreCallDeny`         |    33 | `Budget{MaxCalls: 3}`; calls 1–3 succeed, call 4 returns zero `llm.Response` + `ErrCallsExceeded` (which wraps `ErrBudgetExceeded`); `scriptedLLM.callCount() == 3` — **the denied attempt MUST NOT reach the LLM**. |
| `TestGenerateFromPrompt_MaxTokens_PostCallDeny_ReturnsBoth` | 54 | `Budget{MaxTokens: 100}`; 3 scripted responses × 60 tokens. Call 1 OK. Call 2 post-call denies — returns `(resp, ErrTokensExceeded)` with `resp.Text == "r2"`, `resp.Usage.TotalTokens == 60`; tracker `Snapshot.Tokens` remains 60 (no-commit-on-deny). Call 3 likewise. callCount==3 (no MaxCalls cap; all attempts reach LLM). Documents v1.2 post-hoc semantics → v1.3 Estimator enables pre-call deny on tokens. |
| `TestGenerateFromPrompt_MaxWall_ContextDeadline`      |    21 | `Budget{MaxWall: 50ms}` + slow LLM (200ms). Err `errors.Is(..., context.DeadlineExceeded)` (NOT `ErrWallExceeded` — fires via ctx). Elapsed < 180ms. Proves wall-clock surface is zero — pure ctx-deadline. |
| `TestGenerateFromPrompt_Concurrent_Race`              |    52 | 20 goroutines × 10 calls = 200 attempts vs `Budget{MaxCalls: 50}`. Succ exactly 50; succ+denied == 200; no non-budget errors; `scriptedLLM.callCount() == succ` (denied attempts never reach LLM). `go test -race` green. |

## Test results

```
$ GOWORK=off GOCACHE=/tmp/go-build go test -race -run 'TestGenerateFromPrompt' . -count=1 -v
=== RUN   TestGenerateFromPrompt_NoBudget_Passthrough
--- PASS: TestGenerateFromPrompt_NoBudget_Passthrough (0.00s)
=== RUN   TestGenerateFromPrompt_MaxCalls_PreCallDeny
--- PASS: TestGenerateFromPrompt_MaxCalls_PreCallDeny (0.00s)
=== RUN   TestGenerateFromPrompt_MaxTokens_PostCallDeny_ReturnsBoth
--- PASS: TestGenerateFromPrompt_MaxTokens_PostCallDeny_ReturnsBoth (0.00s)
=== RUN   TestGenerateFromPrompt_MaxWall_ContextDeadline
--- PASS: TestGenerateFromPrompt_MaxWall_ContextDeadline (0.05s)
=== RUN   TestGenerateFromPrompt_Concurrent_Race
--- PASS: TestGenerateFromPrompt_Concurrent_Race (0.00s)
PASS
ok      github.com/costa92/llm-agent    1.061s
```

Repo-wide `GOWORK=off go test ./... -count=1`: **16/16 packages green**.
The 5 paradigm test files (`simple_test.go`, `react_test.go`,
`reflection_test.go`, `plan_solve_test.go`, `function_call_test.go`)
all pass unchanged — they construct contexts without `WithBudget`, so
`budget.From(ctx)` returns `(nil, false)` and the chokepoint's new
branches are no-ops. **Backwards-compat empirically verified.**

## `<verify>` gates (all green)

| Gate                                       | Result |
| ------------------------------------------ | ------ |
| `SIGNATURE-OK` (signature byte-identical)  | PASS   |
| `BUDGET-IMPORTED`                          | PASS   |
| `t.Charge(budget.Usage` count ≥ 2          | PASS (=2) |
| `Q2-COMMENT-OK` (attempts comment present) | PASS   |
| `COST-DEFER-COMMENT-OK`                    | PASS   |
| `go vet ./...` + `go build ./...`          | PASS   |
| `go test ./... -count=1`                   | PASS   |
| `go test -race -run TestGenerateFromPrompt`| PASS   |
| `go.mod` / `go.sum` unchanged (count == 0) | PASS (0) |
| no paradigm-file edits (count == 0)        | PASS (0) |
| no `llm/` public-type edits (count == 0)   | PASS (0) |

## Stdlib-only confirmation

- `git status --short -- go.mod go.sum` → 0 lines (no change).
- The only new import in `agent_chatmodel.go` is
  `"github.com/costa92/llm-agent/budget"` — same module path; intra-
  module, not a new dependency. No `go.sum` created.
- `agent_chatmodel_test.go` imports only `context`, `errors`, `sync`,
  `sync/atomic`, `testing`, `time` (stdlib) + the intra-module
  `budget` and `llm` packages.

## Local commit

```
d141bf6 feat(agents): wire budget enforcement into generateFromPrompt chokepoint (CC-1 / Phase 35 Wave 2)
 agent_chatmodel.go      |  37 ++++++-
 agent_chatmodel_test.go | 278 ++++++++++++++++++++++++++++++++++++++++++++++++
 2 files changed, 314 insertions(+), 1 deletion(-)
```

## Deviations from PLAN

- **`tokenResp` helper added** — the canonical `textResp` helper in
  `scriptedllm_test.go` leaves `Usage` empty (sets only
  `Source: UsageReported`). The chokepoint's post-call charge reads
  `resp.Usage.TotalTokens`, so the budget-aware tests need a variant
  that populates the field. Added at the top of
  `agent_chatmodel_test.go` rather than extending the shared mock
  (smaller blast radius — Wave 2 is the only consumer, and the
  canonical scripted helper in `llm/` already supports Usage
  population via `llm.NewScriptedLLM(llm.WithResponses(...))` for
  future tests). No changes to `scriptedllm_test.go`. Documented per
  the prompt's "if you extend ScriptedLLM, document in SUMMARY" rule.
- **`slowScriptedLLM` test-local wrapper added** — needed for the
  MaxWall test. The canonical `scriptedLLM.Generate` returns
  immediately; ctx-deadline tests require a sleep that honors
  `ctx.Done()`. Wrapper is 21 lines, lives in the test file, no
  production-code change.
- **`llm.Response{}` zero-comparison swapped for field-by-field check**
  — `llm.Response` carries a `[]ToolCall` so direct struct comparison
  fails to compile. The test now checks the scalar fields
  (`Text`, `FinishReason`, `Provider`, `Usage.TotalTokens`) that
  `ScriptedLLM` would populate; this still proves the deny-zero
  invariant.
- **All other PLAN guidance followed exactly** — body-only edit;
  signature unchanged; Q2 + v1.3-CostMapper comments present; no
  paradigm-file edits; no `llm/` public-type edits.

## Backwards-compat audit

The "no behavior change when no budget set" invariant was empirically
verified four ways:

1. **`TestGenerateFromPrompt_NoBudget_Passthrough`** — explicit test of
   the chokepoint with `context.Background()`.
2. **All 5 paradigm test files pass unchanged** — they construct
   contexts without `WithBudget`; if the chokepoint were not properly
   guarded, every paradigm test would break.
3. **`budget.From(ctx)` returns `(nil, false)` on a budget-less ctx**
   — asserted in the passthrough test.
4. **The branch `if hasBudget { ... }` skips both `Charge` sites
   entirely** — no allocation, no lock, no atomic-load when no budget
   is installed; the function's hot-path cost when budget is absent is
   one ctx.Value lookup + one bool check.

## Next step

**PAUSED AT PUSH GATE.** Per `~/.claude/projects/.../slice_workflow.md`
and the prompt's "Do NOT push" directive: the slice is locally
committed at `d141bf6` on `main`. Operator must invoke
`git push origin main` to publish. Wave 3 (35-03: integration tests
across all 5 paradigms with budget installed) and Wave 4 (35-04: the
backwards-compat byte-identical regression net) are gated on operator
confirmation that 35-02 is publishable.

---
phase: 35-budget-and-cancellation-context
plan: 04
type: execute
wave: 4
status: complete
closes_phase: true
requirements: [CC-1]
files_modified:
  - simple_test.go
  - react_test.go
  - reflection_test.go
  - plan_solve_test.go
  - function_call_test.go
files_created:
  - budget_integration_test.go
commit: 535375f
prior_waves:
  - {plan: "35-01", subject: "budget package (NewStrict/NewSoft/WithBudget + sentinels)"}
  - {plan: "35-02", subject: "chokepoint edit in agent_chatmodel.go (commit d141bf6)"}
  - {plan: "35-03", subject: "example walkthrough (commit 39950e2)"}
gates:
  - PARADIGMS-OK
  - UNIFORMITY-OK
  - REPO-GREEN
  - REPO-RACE-OK
  - STDLIB-EXIT-GATE=0
  - BUDGET-TESTS-PRESENT
  - INTEGRATION-FILE-OK
  - KC-5-OK
verify_commands_run:
  - 'GOWORK=off GOCACHE=/tmp/go-build go test -run TestSimple_BudgetExhaustion|TestReAct_BudgetExhaustion|TestReflection_BudgetExhaustion|TestPlanSolve_BudgetExhaustion|TestFunctionCall_BudgetExhaustion . -count=1'
  - 'GOWORK=off GOCACHE=/tmp/go-build go test -run TestAllParadigms_BudgetUniformity . -count=1'
  - 'GOWORK=off GOCACHE=/tmp/go-build go vet ./...'
  - 'GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1'
  - 'GOWORK=off GOCACHE=/tmp/go-build go test -race ./... -count=1'
  - 'GOWORK=off go list -deps -f ... | sort -u | grep -vE ^(github\.com/costa92/llm-agent(-rag)?)$ | grep -v ^$ | wc -l   →  0'
---

# Phase 35 Plan 04: Cross-Paradigm Budget Uniformity Summary

Wide integration test confirming the 35-02 chokepoint edit propagates
correctly through all 5 agent paradigms. Five paradigm-local tests +
one cross-paradigm table-driven test + stdlib-only exit gate verified
green. **Phase 35 closes with this slice.**

## One-liner

Every agent paradigm (Simple / ReAct / Reflection / PlanSolve /
FunctionCall) propagates `budget.ErrCallsExceeded` (wrapping
`budget.ErrBudgetExceeded`) with a zero `agents.Result{}` when the
chokepoint denies, and the tracker's `Snapshot().Calls == cap` proves
the denied attempt did not mutate state.

## What changed

### Files modified (5 paradigm test files)

| File                    | New test                          | Cap | Strategy                                                            |
| ----------------------- | --------------------------------- | --- | ------------------------------------------------------------------- |
| `simple_test.go`        | `TestSimple_BudgetExhaustion`     | 1   | **cross-Run** (SimpleAgent is 1-call-per-Run)                       |
| `react_test.go`         | `TestReAct_BudgetExhaustion`      | 2   | scratchpad loop attempts 3rd LLM call → pre-charge denies           |
| `reflection_test.go`    | `TestReflection_BudgetExhaustion` | 2   | gen + critique pass; revise pre-charge denies                       |
| `plan_solve_test.go`    | `TestPlanSolve_BudgetExhaustion`  | 2   | plan + step-1 pass; step-2 pre-charge denies                        |
| `function_call_test.go` | `TestFunctionCall_BudgetExhaustion` | 1 | **cross-Run** (FunctionCallAgent is single-turn — 1 call per Run)   |

### File created

`budget_integration_test.go` — `package agents` at repo root. Ships
`TestAllParadigms_BudgetUniformity`, a table-driven test asserting
identical chokepoint behavior across all 5 paradigms. SimpleAgent +
FunctionCallAgent take a `crossRun: true` branch (the table preserves
them — their cross-Run enforcement is load-bearing for Phase 37
Supervisor).

## Uniform assertions (every paradigm, every test)

```go
errors.Is(err, budget.ErrCallsExceeded)   // dim-specific sentinel
errors.Is(err, budget.ErrBudgetExceeded)  // umbrella sentinel
reflect.DeepEqual(result, agents.Result{}) // paradigms zero-return on chokepoint err
tracker.Snapshot().Calls == cap            // denied attempt did NOT mutate
```

This matches the plan-checker-revised assertions; the original draft's
"partial result preserved" / `Result.Output` / `Result.Usage.LLMCalls`
assertions would have been wrong — paradigm production code returns
`Result{}` on every `generateFromPrompt` error path (`react.go:106`,
`reflection.go:81/94/110`, `plan_solve.go:81/102/119`).

## Verification (all gates green)

| Gate                       | Result                                                              |
| -------------------------- | ------------------------------------------------------------------- |
| PARADIGMS-OK               | 5/5 paradigm budget tests pass                                      |
| UNIFORMITY-OK              | 5/5 sub-tests under `TestAllParadigms_BudgetUniformity` pass        |
| REPO-GREEN                 | full `go vet ./...` + `go test ./...` clean (16 packages)           |
| REPO-RACE-OK               | full `go test -race ./...` clean (16 packages, ~16s)                |
| **Stdlib-only exit gate**  | **0** non-stdlib non-self modules — load-bearing milestone proof    |
| BUDGET-TESTS-PRESENT       | every paradigm `*_test.go` contains a `BudgetExhaustion` test       |
| INTEGRATION-FILE-OK        | `budget_integration_test.go` exists at repo root + has uniformity test |
| KC-5: production untouched | `simple.go` / `react.go` / `reflection.go` / `plan_solve.go` / `function_call.go` / `agent.go` — 0 modifications |
| KC-5: llm public types     | `llm/types.go` / `llm/chatmodel.go` — 0 modifications               |
| `go.mod` / `go.sum`        | unchanged from pre-35 baseline                                      |

## Deviations from plan

**Single deviation — FunctionCall paradigm strategy.** The plan's task 2
description for FunctionCall said "Script 2 responses (tool-binding
call + post-tool synthesis)" / "the agent should make the first call,
dispatch tool, then on the second LLM call get denied." This wording
implied a multi-turn paradigm; the actual `function_call.go` is
**single-turn** (one `generateFromPrompt`, then `AsyncRunner.Execute`
on the returned tool calls, then aggregate — no post-tool LLM call).
The plan's assertion target ("tool counter == 1, tracker Calls == 1")
is compatible with the only feasible single-turn strategy: the
**cross-Run pattern** (two `agent.Run` calls in the same ctx with
`MaxCalls: 1`; first succeeds and dispatches the tool, second is
denied at pre-call charge). This is exactly the SimpleAgent pattern
applied to FunctionCall — both single-call-per-Run paradigms reach the
same `cap` via the same proof shape. The cross-paradigm integration
table reflects this by marking both rows `crossRun: true`. No
production code was touched. The PLAN's `needs: 2` integer for
FunctionCall maps cleanly to "needs at least 2 Runs to exhaust
`MaxCalls=1`".

(No other deviations. All five paradigm-local tests landed exactly as
planned; the integration table includes all 5 rows as required.)

## Why this slice closes Phase 35

Phase 35's CC-1 acceptance was "every agent paradigm respects a budget
applied via context." The chokepoint edit in 35-02 made the enforcement
mechanical (one pre-call `Charge` in `generateFromPrompt`); this slice
proves it actually works in every paradigm. With the stdlib-only exit
gate returning 0 and `-race` clean repo-wide, the milestone's
load-bearing invariants are intact: core stays stdlib-only, no race
regressions from the chokepoint edit, every paradigm uniform.

## Phase 35 — all 4 waves complete

| Wave | Plan  | Subject                                                  | Commit  |
| ---- | ----- | -------------------------------------------------------- | ------- |
| 1    | 35-01 | `budget` package — sentinels, Tracker, WithBudget        | (Wave 1)|
| 2    | 35-02 | Chokepoint edit in `agent_chatmodel.go`                  | d141bf6 |
| 3    | 35-03 | Example walkthrough (`examples/budget/`)                 | 39950e2 |
| 4    | 35-04 | Cross-paradigm uniformity tests + stdlib-only exit gate  | 535375f |

Milestone close (planning-tree refresh, STATE.md / ROADMAP.md updates,
v1.2 → v1.3 transition prep) is owned by **Phase 38**, not this slice.

## Out of scope (deferred)

- `budget.Wrap` decorator → v1.3
- Paradigm-level `MaxTokens` / `MaxWall` / `MaxCost` tests → already
  covered at chokepoint level in 35-02; paradigm coverage redundant
- `go test -bench` runs → out of v1.2 scope
- `.planning/STATE.md` / `.planning/ROADMAP.md` refresh → Phase 38
- Multi-turn FunctionCall (would require `pkg/llm.Message.ToolCallID`)
  → out of phase

## Local commit (NOT pushed)

```
535375f test(agents): wide paradigm budget integration + stdlib-only exit gate (CC-1 / Phase 35 Wave 4)
```

**PAUSED AT PUSH GATE** — the user invokes `git push` separately
(per the hard-rule "never commit/push without explicit ask"; commit was
explicit, push is not).

## Self-Check: PASSED

- `simple_test.go` :: `TestSimple_BudgetExhaustion` — FOUND
- `react_test.go` :: `TestReAct_BudgetExhaustion` — FOUND
- `reflection_test.go` :: `TestReflection_BudgetExhaustion` — FOUND
- `plan_solve_test.go` :: `TestPlanSolve_BudgetExhaustion` — FOUND
- `function_call_test.go` :: `TestFunctionCall_BudgetExhaustion` — FOUND
- `budget_integration_test.go` :: `TestAllParadigms_BudgetUniformity` — FOUND
- Commit `535375f` — FOUND on `main` (local)
- All eight `<verify>` gates green

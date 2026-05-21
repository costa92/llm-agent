---
phase: 35-budget-and-cancellation-context
plan: 01
status: paused-at-push-gate
completed_at: 2026-05-20
repo: llm-agent (core)
requirements: [CC-1]
files_modified:
  - budget/budget.go
  - budget/budget_test.go
  - budget/doc.go
artifacts:
  - "llm-agent local commit 581caea on main"
  - "PUSH PENDING: git push origin main — operator-gated"
---

# 35-01 — `budget` package (CC-1 / Phase 35 Wave 1, paused at push gate)

Wave 1 of Phase 35 (v1.2 milestone, the first feature phase). Shipped the
stdlib-only `budget` package as a pure data + plumbing layer: `Budget`
value type, distinct `Usage` value type, concurrency-safe `Tracker`
interface with two built-in constructors (`NewStrict`, `NewSoft`),
ctx-keyed `WithBudget` / `WithTracker` / `From` plumbing, and a
sentinel-error family rooted at `ErrBudgetExceeded`. No edits outside
`budget/`. The chokepoint integration into `agent_chatmodel.go::
generateFromPrompt` is Wave 2 (slice 35-02). Push is operator-gated.

## Outcome

- New directory `budget/` at the repo root, 3 files, 696 lines total.
- Public surface: 13/13 symbols present + doc-commented (grep gate
  `SURFACE-OK`).
- All exit gates green locally: `go vet`, `go test`, `go test -race`,
  repo-wide vet + build, surface grep, doc-content grep.
- Zero non-stdlib imports in `budget/` (only `context`, `errors`, `fmt`,
  `sync`, `sync/atomic`, `time` — verified by `go list -f` grep).
- `go.mod` unchanged; no `go.sum` created or modified.
- Single local commit `581caea` on `main`; not yet pushed.

## Files

| File                 | Lines | Purpose                                                                                  |
| -------------------- | ----: | ---------------------------------------------------------------------------------------- |
| `budget/doc.go`      |    45 | Package doc — first-sentence per Go conv; Q1 (three Usage types coexist) + Q2 (MaxCalls counts attempts) cited; CC-1 / KC-4 / KC-5 referenced. |
| `budget/budget.go`   |   309 | `Budget`, `Usage`, `Tracker`, sentinel-error family, `strictTracker` / `softTracker` impls, `NewStrict`, `NewSoft`, `WithBudget`, `WithTracker`, `From`. |
| `budget/budget_test.go` | 342 | 15 tests covering every Charge path, ordering invariant, denied-charge non-mutation, Remaining arithmetic + zero-cap convention, ctx-deadline derivation, ctx-key round-trip, 100×100 concurrency race (-race green), sentinel wrapping. |

## Surface inventory (13/13)

| Symbol                | Kind          | Notes                                                          |
| --------------------- | ------------- | -------------------------------------------------------------- |
| `Budget`              | struct        | `MaxTokens` / `MaxCalls` / `MaxWall` / `MaxCost`; zero = no cap. |
| `Usage`               | struct        | Distinct from `llm.Usage` / `agents.Usage` per Q1.             |
| `Tracker`             | interface     | `Charge(Usage) error`, `Snapshot() Usage`, `Remaining(Budget) Usage`. |
| `NewStrict`           | func          | Strict-mode constructor (default; used by `WithBudget`).       |
| `NewSoft`             | func          | Soft-mode (observability-only) constructor.                    |
| `WithBudget`          | func          | One-liner: builds strict tracker + attaches; derives `context.WithDeadline` when `MaxWall > 0`. |
| `WithTracker`         | func          | Explicit attach; does NOT derive deadline.                     |
| `From`                | func          | `(Tracker, bool)` ok-comma; `(nil, false)` when absent.        |
| `ErrBudgetExceeded`   | sentinel      | Umbrella.                                                      |
| `ErrTokensExceeded`   | sentinel      | Wraps `ErrBudgetExceeded` via `fmt.Errorf("%w: tokens", …)`.   |
| `ErrCallsExceeded`    | sentinel      | Wraps umbrella; Q2 — counts attempts.                          |
| `ErrWallExceeded`     | sentinel      | Wraps umbrella; complements `context.DeadlineExceeded`.        |
| `ErrCostExceeded`     | sentinel      | Wraps umbrella.                                                |

## Verify outcomes

All exit gates from PLAN `<verify>`, run from the repo root with
`GOWORK=off GOCACHE=/tmp/go-build`:

| Check                                                                 | Result        |
| --------------------------------------------------------------------- | ------------- |
| `go vet ./budget/...`                                                 | `VET-OK`      |
| `go test ./budget/... -count=1`                                       | `TEST-OK` — 15 tests pass, 0.027s |
| `go test -race ./budget/... -count=1`                                 | `RACE-OK` — 1.035s |
| `go list -f '{{join .Imports "\n"}}' ./budget/` minus stdlib whitelist | `0` non-stdlib imports |
| `git status --short -- go.mod go.sum`                                 | empty — `go.mod` / `go.sum` untouched |
| repo-wide `go vet ./...` + `go build ./...`                           | `REPO-SMOKE-OK` |
| public-surface grep (13 symbols)                                      | `SURFACE-OK`  |
| `doc.go` cites Q1 + Q2 + "attempts"                                   | `DOC-OK`      |
| `go list -deps ./budget/...`                                          | stdlib-only (no `github.com/*` deps beyond the module's own path) |

## Implementation notes

### Check-before-commit under concurrent load

The full check-and-commit transaction in `strictTracker.Charge` is
serialized under `t.mu`. The original plan called for atomic adds on
the hot path with separate cap checks, but two goroutines can both
observe `curCalls=4999`, both compute `want=5000 ≤ cap`, and both
commit, ending at `calls=5001 > cap` — that violates "Snapshot().Calls
≤ cap" and defeats the purpose of a strict cap. Holding `t.mu` for the
whole Charge is the simplest stdlib-only fix and keeps the
check-before-commit invariant atomic. Snapshot reads remain lock-free
via `atomic.LoadInt64` for the counter fields (Wall/Cost are still
mutex-read for memory consistency with non-atomic types). The race
test (`TestCharge_Concurrent_Race`, 100 goroutines × 100 charges)
asserts both `Snapshot().Calls ≤ cap` and `successes + denials ==
attempts`; passes under `-race`.

### `context.WithDeadline` cancel func handling

`go vet`'s `lostcancel` check rejects `ctx, _ = context.WithDeadline(…)`
on the same statement. The accepted idiom is a separate
`var cancel; ctx, cancel = …; _ = cancel`, which is what `WithBudget`
does — with a comment explaining the deadline is intended to outlive
the function (the tracker is the canonical "budget exhausted" signal
for the non-wall dimensions; the deadline fires on its own when wall
expires).

### Charge ordering

`Charge` checks `MaxCalls → MaxTokens → MaxCost → MaxWall` in that
order; verified by `TestCharge_OrderingMatters` (a single Charge that
violates both Calls and Tokens caps returns `ErrCallsExceeded`, not
`ErrTokensExceeded`). This order matches the plan and means a
multi-dim runaway is reported by the smallest-incrementing dimension
first.

## Deviations

| # | Deviation | Reason | Severity |
| - | --------- | ------ | -------- |
| 1 | `Charge` is fully mutex-serialized, not atomic-add-on-hot-path | Plan §"Default tracker impl" mentioned "Zero-alloc on the hot path (`Charge`): atomic adds". Plain atomic adds cannot enforce a hard cap under concurrent load — the check-and-commit must be atomic together. Snapshot reads remain lock-free; only Charge is serialized. | Implementation detail — public contract unchanged; correctness invariant honored. |
| 2 | `cancel` from `context.WithDeadline` bound to a named var + `_ = cancel` discard | Plan said "`ctx, _ = context.WithDeadline(…)`"; `go vet` rejects that form. The named-var pattern is the standard discard idiom and preserves the deadline-outlives-function intent. | Idiomatic-syntax adjustment. |
| 3 | `Budget.Validate()` not added (Q5 of research) | Plan task 3 explicitly defers `Validate` to v1.3 unless a foot-gun surfaces in 35-04 tests. None did in this slice. | None — matches plan. |

## Stdlib-only confirmation

```
$ GOWORK=off go list -f '{{join .Imports "\n"}}' ./budget/ \
    | grep -vE '^(context|errors|fmt|sync|sync/atomic|time)$' | wc -l
0
$ git status --short -- go.mod go.sum | wc -l
0
```

`go.mod` still has its single require (`github.com/costa92/llm-agent-rag
v1.0.1`); no `go.sum` exists; the only module-qualified deps reachable
from `./budget/...` are `github.com/costa92/llm-agent/budget` itself.

## Identifiers

| Artifact                       | Value                                                            |
| ------------------------------ | ---------------------------------------------------------------- |
| Local commit SHA               | `581caea` (full: `581caea…`)                                     |
| Local commit message           | `feat(budget): add ctx-keyed budget package (CC-1 / Phase 35 Wave 1)` |
| Files created                  | `budget/budget.go`, `budget/budget_test.go`, `budget/doc.go`     |
| Lines added                    | 696 (309 + 342 + 45)                                             |
| Branch                         | `main`                                                           |
| `git log origin/main..main`    | 1 commit (this slice) — push pending                             |

## Next step

**PAUSED AT PUSH GATE** — orchestrator runs `git push origin main`
after operator confirms. After push, the slice's exit gates re-run
clean in CI; Wave 2 (slice 35-02 — chokepoint integration) unblocks.

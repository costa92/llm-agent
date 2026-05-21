---
phase: 35-budget-and-cancellation-context
plan: 03
status: paused-at-push-gate
completed_at: 2026-05-20
repo: llm-agent (core; examples submodule)
requirements: [CC-1]
depends_on: ["35-01", "35-02"]
files_modified:
  - examples/06-budget/main.go            (new, 215 lines)
  - examples/06-budget/main_test.go       (new,  67 lines)
  - examples/06-budget/README.md          (new,  78 lines)
  - examples/README.md                    (+2 lines, 2 hunks)
artifacts:
  - "llm-agent local commit 39950e2 on main"
  - "PUSH PENDING: git push origin main — operator-gated"
---

# 35-03 — deterministic `06-budget` example (CC-1 / Phase 35 Wave 3, paused at push gate)

Wave 3 of Phase 35. Ships a runnable, network-free demo at
`examples/06-budget/` that wires `budget.WithBudget` through a
`SimpleAgent` and exercises all three budget dimensions enforced by the
Wave 2 chokepoint (`agent_chatmodel.go::generateFromPrompt`). Uses the
canonical `examples/scriptedllm` mock per CLAUDE.md. New companion
`main_test.go` captures stdout via `os.Pipe` and asserts the
deterministic transcript — a deliberate divergence from 01-05 (none of
which ship tests today), recorded as precedent for enforcement-contract
demos. The `examples/README.md` index lists 06-budget alongside the
existing five. Core `go.mod`/`go.sum` unchanged; `examples/go.mod`
unchanged (the existing `replace … => ../` satisfies the new
intra-module `budget` import; no `examples/go.sum` exists — stdlib-only
intra-module).

## Outcome

- `examples/06-budget/main.go` — 215 lines. `main()` calls
  `demoMaxCalls()`, `demoMaxTokens()`, `demoMaxWall()` in sequence then
  prints "OK" so the test can assert end-to-end completion.
- `examples/06-budget/main_test.go` — 67 lines. One test:
  `TestExample_RunsToCompletion`. Captures stdout via `os.Pipe`, runs
  `main()`, restores, asserts 11 deterministic fragments covering all
  three dimensions plus the trailing "OK".
- `examples/06-budget/README.md` — 78 lines (target ≤60 narrowly missed;
  the deferred-to-v1.3 bullets + stdlib-only footer pushed it over).
  Cites `agent_chatmodel.go::generateFromPrompt` by name, calls out Q2
  (attempts vs successes), notes the v1.3-deferred CostMapper /
  Estimator / `budget.Wrap` decorator.
- `examples/README.md` — added a table row + appended `06-budget` to
  the run-loop section. Style mirrors the existing 01-05 entries.
- Single local commit `39950e2` on `main`; not yet pushed.

## How the three sub-demos illustrate the contract

| Sub-demo                                  | Budget                | What the transcript proves |
| ----------------------------------------- | --------------------- | -------------------------- |
| `demoMaxCalls()` (pre-call deny)          | `Budget{MaxCalls: 3}` | 3 of 4 attempts succeed; 4th returns `ErrCallsExceeded` (wrapping `ErrBudgetExceeded`). A `countingLLM` wrapper proves `model.Generate` was reached only 3 times — the denied attempt is short-circuited BEFORE the network. |
| `demoMaxTokens()` (post-call deny)        | `Budget{MaxTokens: 100}` (3×60-tok responses) | Call 1 OK. Call 2 returns `ErrTokensExceeded` — but the `countingLLM` shows the model WAS reached (network call succeeded; the chokepoint returned `(resp, sentinel)`). Call 3 same contract again. Tracker snapshot: `Tokens=60` — no-commit-on-deny invariant verified. |
| `demoMaxWall()` (ctx-deadline)            | `Budget{MaxWall: 50ms}` + 200 ms `slowLLM` | `errors.Is(err, context.DeadlineExceeded) = true` — wall-clock has ZERO new error surface (Decision 4). `slowLLM.Generate` honors `ctx.Done()` so the deadline fires cleanly. Elapsed < 200 ms confirms cancellation, not a slow response. |

## Demo output (deterministic, captured 2026-05-20)

```
--- MaxCalls (pre-call deny) ---
call 1: ok
call 2: ok
call 3: ok
call 4: denied — budget: exceeded: calls
4th denied with errors.Is(err, budget.ErrCallsExceeded) = true
4th denied with errors.Is(err, budget.ErrBudgetExceeded) = true
LLM Generate calls reaching the model: 3 (denied attempt never reaches LLM)
tracker snapshot: {Tokens:30 Calls:3 Wall:0s Cost:0}

--- MaxTokens (post-call deny) ---
call 1: ok
call 2: valid response but exhausted — budget: exceeded: tokens
call 3: valid response but exhausted — budget: exceeded: tokens
LLM Generate calls reaching the model: 3 (network call succeeded all 3 times — deny is post-call)
tracker snapshot: {Tokens:60 Calls:3 Wall:0s Cost:0} (no-commit-on-deny: only the successful 60 tokens are recorded)

--- MaxWall (ctx.DeadlineExceeded) ---
call: errors.Is(err, context.DeadlineExceeded) = true (wall-clock fires via ctx, not a budget sentinel)
deadline fired before response: elapsed < 4x deadline? true

OK
```

## `<verify>` gates (all green)

| Gate                                                            | Result |
| --------------------------------------------------------------- | ------ |
| `EXAMPLE-FILES-OK` (main.go + main_test.go + README.md exist)   | PASS   |
| `EXAMPLE-TEST-OK` (`cd examples && go test ./06-budget/...`)    | PASS   |
| `EXAMPLE-RUN-OK` (`cd examples && go run ./06-budget`)          | PASS   |
| `INDEX-OK` (examples/README.md mentions 06-budget)              | PASS   |
| `README-OK` (cites generateFromPrompt + attempts/Q2 + v1.3)     | PASS   |
| `BUDGET-IMPORT-OK` (main.go imports `…/llm-agent/budget`)       | PASS   |
| `SCRIPTED-OK` (main.go uses canonical scriptedllm)              | PASS   |
| `CORE-TEST-OK` (`go test ./... -count=1` core, 16/16 packages)  | PASS   |
| core `go.mod`/`go.sum` unchanged (status lines == 0)            | PASS   |
| `examples/go.mod` unchanged (no new external dep)               | PASS   |

## Stdlib-only confirmation

- Core `git status --short -- go.mod go.sum` → 0 lines.
- `examples/go.mod` unchanged — the existing
  `replace github.com/costa92/llm-agent => ../` resolves the new
  `…/llm-agent/budget` import locally. No `examples/go.sum` exists or
  was created (intra-module, no external deps).
- `examples/06-budget/main.go` imports only stdlib (`context`, `errors`,
  `fmt`, `sync/atomic`, `time`) + the intra-module `agents`, `budget`,
  `llm`, and `examples/scriptedllm` packages.
- `main_test.go` imports only `io`, `os`, `strings`, `testing` (stdlib).

## Deviations from PLAN

- **README is 78 lines, plan target ≤60.** Honest count: 16 lines for
  the intro/run section, 13 for the Budget struct, 17 for the
  chokepoint walk-through, 13 for the wall-clock section, 9 for the
  v1.3-deferred bullets, 4 for the stdlib-only footer. Trimming would
  cost clarity on the post-call-deny contract or drop the v1.3 list.
  Kept at 78. The plan's `acceptance` block does not bind the line
  count (it lives in tasks/6 narrative); recording for transparency.
- **`SimpleAgent` instead of `ReActAgent`.** The plan suggested ReAct
  but noted "if wiring a multi-turn ReAct is heavy here, use the
  simpler SimpleAgent in a manual loop — clarity trumps paradigm-
  richness for the example. Pick the cleaner of the two; document the
  choice." SimpleAgent + a manual `for i := 1; i <= 4` loop is exactly
  the cleaner shape — each iteration is one chokepoint pass, so the
  MaxCalls/MaxTokens semantics map one-to-one to call counts. ReAct
  would muddle the count (each user prompt fans out to N internal
  thought-action turns).
- **`countingLLM` wrapper added.** Not in the plan, but it's how the
  demo proves "pre-call deny short-circuits BEFORE the LLM" vs "post-
  call deny does reach the LLM" via printed counts. Pure example-local
  helper, 14 lines in `main.go`.
- **`tokenText` helper added (not the plan's verbatim name).** The
  canonical `scriptedllm.Text` leaves `Usage.TotalTokens == 0`, which
  would defeat the MaxTokens demo. Same rationale as Wave 2's
  `tokenResp` helper in `agent_chatmodel_test.go`. Kept local to the
  example file rather than extending the shared `scriptedllm` package
  (smaller blast radius).
- **All other PLAN guidance followed exactly** — three sub-demos in
  sequence, ScriptedLLM mock per CLAUDE.md, slowLLM honors ctx.Done(),
  `examples/README.md` row added without rewriting the file,
  `cd examples && go mod tidy` ran (no-op — the local replace covers
  the new import), all 8 verify gates green, no top-level README.md
  edit (the project root README does not list examples — verified —
  so the plan's escape hatch applies).

## Local commit

```
39950e2 docs(examples): add 06-budget deterministic example (CC-1 / Phase 35 Wave 3)
 examples/06-budget/README.md     |  78 +++++++++++++++++++
 examples/06-budget/main.go       | 215 +++++++++++++++++++++++++++++++++++++++++++
 examples/06-budget/main_test.go  |  67 +++++++++++++++
 examples/README.md               |   4 +-
 4 files changed, 364 insertions(+), 2 deletions(-)
```

## Next step

**PAUSED AT PUSH GATE.** Per `~/.claude/projects/.../slice_workflow.md`
and the prompt's "Do NOT push" directive: the slice is locally
committed at `39950e2` on `main`. Operator must invoke
`git push origin main` to publish. With 35-01 (`581caea` — budget pkg),
35-02 (`d141bf6` — chokepoint), and 35-03 (`39950e2` — example,
local), the CC-1 chain is functionally complete;
Wave 4 (35-04: backwards-compat byte-identical regression net) is the
remaining gate. Operator confirmation that 35-03 is publishable unlocks
the next slice.

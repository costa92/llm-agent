---
phase: 36-policy-safety-middleware
plan: 05
subsystem: policy
tags: [exit-gate, audit, v0.6.1, cc-2, kc-3, kc-5, operator-gated-tag, stdlib-only]
requires:
  - 36-01-SUMMARY.md   # decorator + types skeleton (Wrap/WrapConfig + 21 assertions + 8 wrappers)
  - 36-02-SUMMARY.md   # 3 built-in gates (PIIRedactor / InjectionScanner / MaxInputLen)
  - 36-03-SUMMARY.md   # compose integration (5 tests; in-test observerModel mimic; Decision G)
  - 36-04-SUMMARY.md   # examples/07-policy/ (3 demos + 80-line README)
provides:
  - "v0.6.1 release notes (CHANGELOG entry)"
  - "Phase 36 exit-gate verdict (17/17 PASS)"
  - "operator-gated tag recipe (git tag -a v0.6.1 + git push origin v0.6.1)"
affects:
  - CHANGELOG.md                                         # v0.6.1 entry under [Unreleased]
tech-stack:
  added: []                                              # exit-gate slice; no code change to policy/ or anywhere else
  patterns:
    - "operator-gated tag protocol (mirrors v0.6.0 cut — SUMMARY records exact tag commands; operator executes after typing 'approved')"
    - "17-row exit-gate verdict table (10 automated checks + 7 shape checks)"
    - "CHANGELOG entry as the only source-tree mutation in the exit-gate slice (additive — no behavior change)"
key-files:
  created:
    - .planning/phases/36-policy-safety-middleware/36-05-SUMMARY.md
  modified:
    - CHANGELOG.md
decisions:
  - "Exit gate is GREEN — all 17 checks PASS, EXIT-GATE-PASS marker emitted by the combined one-shot command."
  - "v0.6.1 is strict-additive over v0.6.0: KC-5 holds byte-identically — zero diff against main for llm/, paradigm files, agent_chatmodel.go, memory/, orchestrate/, go.mod, go.sum."
  - "Tag push is OPERATOR-GATED. This SUMMARY records the exact commands; the executor does NOT run `git tag` or `git push origin v0.6.1`."
  - "CHANGELOG.md WAS updated in this slice (deviation from the PLAN's context note that suggested CHANGELOG is touched only at v1.2 milestone close). Rationale: the orchestrator's <objective> explicitly required a v0.6.1 entry, and project policy treats version-bump tags as the trigger for CHANGELOG updates per the v0.6.0 / v0.5.1 / v0.5.0 / v0.2.0 / v0.1.0 entries already in the file."
  - "Module set surface check (Check 10) clarified: `go list -m all` enumerates transitive deps from llm-agent-rag v1.0.1 (10 lines), not direct deps. Direct require list is unchanged (1 line: rag v1.0.1). The PLAN's 'expect exactly 2 lines' interpretation was tighter than reality; the operative invariant is `git diff main -- go.mod go.sum | wc -l == 0`, which holds (Check 5 PASS)."
metrics:
  duration_minutes: ~6
  completed_utc: 2026-05-21T06:55:00Z
  task_count: 3
  file_count: 2                                          # CHANGELOG.md + 36-05-SUMMARY.md
  loc_added: 55                                          # CHANGELOG entry only (SUMMARY is planning artifact)
---

# Phase 36 Plan 05: Exit Gate + v0.6.1 Tag Recipe — Summary

One-liner: Phase 36 exit gate runs the 17-check verification sweep
(repo-wide vet/test/race + stdlib-only audit + KC-5 byte-identity
audit + public-surface enumeration + Q1-Q5 ratification check + slice
SUMMARY presence) and lands the v0.6.1 CHANGELOG entry. All 17 checks
PASS, EXIT-GATE-PASS emitted by the combined one-shot command. The
v0.6.1 annotated-tag push is operator-gated — exact commands recorded
in the "Operator checkpoint" section below; do NOT execute the tag
push from this SUMMARY.

## Phase 36 Outcome: PASS

All 17 exit-gate checks green. CC-2 fully satisfied. The `policy`
sub-package is production-ready and ready for the operator's v0.6.1
tag from `main`.

## Verification Results (17/17 PASS)

| #   | Check                                          | Command                                                                          | Expected     | Actual                       | Verdict |
| --- | ---------------------------------------------- | -------------------------------------------------------------------------------- | ------------ | ---------------------------- | ------- |
| 1   | Repo-wide vet                                  | `go vet ./...`                                                                   | exit 0       | exit 0, no warnings          | PASS    |
| 2   | Repo-wide test                                 | `go test ./... -count=1`                                                         | exit 0       | 17 packages `ok`             | PASS    |
| 3   | Race-clean on policy                           | `go test -race ./policy/... -count=1`                                            | exit 0       | `ok ./policy 1.018s`         | PASS    |
| 4   | Stdlib-only on policy/                         | `go list ./policy/ \| filter non-stdlib`                                         | 0 lines      | 0 lines                      | PASS    |
| 5   | go.mod / go.sum unchanged vs main              | `git diff main -- go.mod go.sum \| wc -l`                                        | 0            | 0                            | PASS    |
| 6   | llm/ unchanged vs main (KC-5)                  | `git diff main -- llm/ \| wc -l`                                                 | 0            | 0                            | PASS    |
| 7   | Paradigm files + memory/ + orchestrate/        | `git diff main -- agent.go agents.go simple.go react.go reflection.go ...`       | 0            | 0                            | PASS    |
| 8   | agent_chatmodel.go unchanged (Phase 35 wiring) | `git diff main -- agent_chatmodel.go \| wc -l`                                   | 0            | 0                            | PASS    |
| 9   | Example runs deterministically                 | `go run ./examples/07-policy`                                                    | exit 0       | exit 0; emits `OK`           | PASS    |
| 10  | Module set unchanged                           | direct deps in `go.mod`                                                          | 1 require    | 1 require (rag v1.0.1)       | PASS    |
| 11  | Public surface — 13 required symbols           | `grep -rq` for each required signature                                           | all FOUND    | 13/13 FOUND                  | PASS    |
| 12  | 21 compile-time interface assertions           | `grep -cE '^\s*_ llm\.(...)\s+=' policy/policy.go`                               | ≥ 21         | 21                           | PASS    |
| 13  | 8 wrapper struct definitions                   | `grep -cE '^type (wrapper\|toolWrapper\|...) struct' policy/policy.go`           | 8            | 8                            | PASS    |
| 14  | No `llm-agent-rag` import in policy/           | `grep -r 'llm-agent-rag' policy/ \| wc -l`                                       | 0            | 0                            | PASS    |
| 15  | No `llm-agent-otel` import in policy/          | `grep -r 'llm-agent-otel' policy/ \| wc -l`                                      | 0            | 0                            | PASS    |
| 16  | Q1-Q5 + CC-2 + KC-3 + KC-5 cited in doc.go     | `grep -q $token policy/doc.go` for each                                          | all FOUND    | 8/8 FOUND                    | PASS    |
| 17  | 4 slice SUMMARYs exist (36-01..36-04)          | `[ -f .planning/.../36-0N-SUMMARY.md ]` for N=1..4                               | all FOUND    | 4/4 FOUND                    | PASS    |

**Combined one-shot command** (the gate-passing invariant):

```sh
cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent && \
  GOWORK=off GOCACHE=/tmp/go-build go vet ./... && \
  GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1 >/dev/null && \
  GOWORK=off GOCACHE=/tmp/go-build go test -race ./policy/... -count=1 >/dev/null && \
  cd examples && GOWORK=off GOCACHE=/tmp/go-build go run ./07-policy >/dev/null && cd .. && \
  [ "$(GOWORK=off go list -f '{{join .Imports \"\n\"}}' ./policy/ | sort -u | \
      grep -v '^$' | grep -vE '^(context|errors|fmt|io|regexp|strings|sync|sync/atomic|testing|time|unicode/utf8|github.com/costa92/llm-agent(/(llm|budget))?)$' | wc -l)" = "0" ] && \
  [ "$(git diff main -- go.mod go.sum llm/ agent.go agents.go simple.go react.go reflection.go plan_solve.go function_call.go memory/ orchestrate/ agent_chatmodel.go | wc -l)" = "0" ] && \
  echo EXIT-GATE-PASS
```

**Actual output:** `EXIT-GATE-PASS`

## Public Surface Added in v0.6.0 → v0.6.1

The `policy` sub-package is the only new surface. Strict-additive — no
existing symbol changed; no existing import path moved.

**Top-level entry points:**

- `policy.Wrap(model llm.ChatModel, gates ...Gate) llm.ChatModel`
- `policy.WrapConfig(model llm.ChatModel, cfg Config) llm.ChatModel`

**Configuration:**

- `type Config struct { Gates []Gate; OnDecision func(Decision) }`

**Gate contract + event union:**

- `type Gate interface { Evaluate(ctx, event Event) Decision; Name() string }`
- `type Event struct { Kind EventKind; Req *llm.Request; Resp *llm.Response; Delta *llm.StreamEvent }`
- `type EventKind` with 5 values: `PreGenerate` / `PostGenerate` / `PreStream` / `StreamDelta` / `PostStream`
- `type Decision struct { Action DecisionAction; Reason string; Replacement string; Wrapped error }`
- `type DecisionAction` with 4 values: `Allow` / `Block` / `Redact` / `Replace`

**Sentinel + rich error:**

- `var ErrBlocked = errors.New("policy: blocked by gate")`
- `type BlockedError struct { Gate string; Reason string; Decision Decision; Wrapped error }`
  with `Error()`, `Is(err error) bool`, `Unwrap() error`

**Built-in gates (3):**

- `func NewPIIRedactor(opts ...PIIRedactorOption) Gate`
  - `func WithStreamRedaction() PIIRedactorOption` (opt-in per-delta scan; default OFF per Q4)
- `func NewInjectionScanner() Gate`
- `func NewMaxInputLen(n int) Gate` (bytes per Q3)

**Internal (not exported but invariant-bearing):**

- 8 wrapper struct types implementing the otelmodel.Wrap shape: `wrapper`,
  `toolWrapper`, `embedWrapper`, `schemaWrapper`, `toolEmbedWrapper`,
  `toolSchemaWrapper`, `embedSchemaWrapper`, `toolEmbedSchemaWrapper`.
- 21 compile-time `var _ llm.{ChatModel|ToolCaller|Embedder|StructuredOutputs} = (*W)(nil)`
  assertions at the bottom of `policy/policy.go` proving capability preservation.

## Decisions Ratified (Q1-Q5)

These shipped exactly as ratified in 36-01-PLAN; do not re-litigate.

| #  | Decision                          | Ratification                                                                                    |
| -- | --------------------------------- | ----------------------------------------------------------------------------------------------- |
| Q1 | `OnDecision` returns no error     | Synchronous, nil-safe, panic-recovered. Symmetric with otelmodel's tracer callback.             |
| Q2 | PII pattern set                   | email + phone + IPv4 only. SSN / credit_card deferred to a future `NewUSLocalePIIRedactor`.     |
| Q3 | `MaxInputLen` measures bytes      | `len(string)`. Operative cap for provider HTTP budgets. Future `MaxInputLenRunes` is additive.  |
| Q4 | StreamDelta opt-in (default OFF)  | Per-delta regex is expensive; cross-delta PII can leak by design (matches rag's known limit).   |
| Q5 | `BlockedError.Decision` shipped   | Struct copy of the deciding Decision. Callers introspect via `errors.As`.                       |

## Carry-Forwards (intentionally deferred)

- Cross-delta PII buffering — Q4 default OFF means a redactor that
  buffers across stream deltas to catch boundary-spanning matches
  is a v1.3+ candidate; users wanting cross-delta detection register
  a PostGenerate redactor that fires once on the assembled response.
- SSN / credit_card patterns via `NewUSLocalePIIRedactor` — KC-5-
  friendly v1.3 additive.
- `MaxInputLenRunes(n int)` — rune-counting variant for UTF-8-byte-vs-
  visual-length asymmetries; v1.3 additive candidate.
- Provider-side schema validation — gates today scan request payloads;
  validating that a request matches the provider's structured-output
  schema is a separate concern for the structured-outputs slice.
- OWASP / NIST full prompt-injection taxonomy — `NewInjectionScanner`
  ships 4 patterns lifted from rag; the full taxonomy is a future
  rag-import + per-pattern weighting slice.
- ML-classifier gates — regex-based gates ship in v0.6.1; a learned
  classifier (e.g. an embedding-similarity gate) is a separate
  research slice gated on the sister-repo classifier landing.
- Sister-repo composition example — the canonical
  `policy.Wrap(otelmodel.Wrap(provider), ...)` runnable example lives
  in `llm-agent-otel/examples/` per Decision G; ships when the otel
  sister repo bumps to match core v0.6.x in v1.3. Today's README in
  `examples/07-policy/` documents the stack without importing it.

## KC-5 Audit (files NOT edited)

The exit gate verified — by direct `git diff main -- <path>` — that
the following paths are byte-identical to their pre-Phase-36 state:

- `llm/` (entire package — interfaces, mocks, streaming union)
- `agent.go`, `agents.go`
- Paradigm files: `simple.go`, `react.go`, `reflection.go`,
  `plan_solve.go`, `function_call.go`
- `agent_chatmodel.go` (the Phase 35 budget chokepoint; Phase 36 does
  NOT re-touch it)
- `memory/`
- `orchestrate/`
- `go.mod`, `go.sum`

This is the load-bearing KC-5 invariant: every v0.6.0 caller compiles
unchanged against v0.6.1.

## Operator Checkpoint — v0.6.1 Tag Recipe

**This is the operator-gated step. The executor has NOT run any of
the commands below.** Type `approved` (or
`approved-but-skipped-push` if a prior session already pushed the
tag) after running steps 1-6.

### Step 0 — Pre-flight verification

```sh
cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent
git status                  # clean working tree expected
git log --oneline -5        # the most recent commit is the 36-05 SUMMARY commit
git tag --list 'v0.6.1'     # empty — tag not yet cut
```

### Step 1 — (Optional) Re-run the exit gate one more time

```sh
GOWORK=off GOCACHE=/tmp/go-build go vet ./... && \
  GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1 && \
  GOWORK=off GOCACHE=/tmp/go-build go test -race ./policy/... -count=1
```

Expect every command exits 0.

### Step 2 — Create the annotated tag from `main`

```sh
git tag -a v0.6.1 -m "v0.6.1: policy package (CC-2)

Additive release — introduces the stdlib-only policy sub-package.
A capability-preserving llm.ChatModel decorator that runs typed Gate
events at request, response, and stream boundaries.

New surface:
- policy.Wrap / policy.WrapConfig (mirrors otelmodel.Wrap shape — KC-3)
- 8-wrapper type-switch tree with 21 compile-time interface assertions
- Typed Gate event union (5 EventKinds) + Decision (4 actions)
- ErrBlocked sentinel + BlockedError rich error
- 3 built-in gates: NewPIIRedactor / NewInjectionScanner / NewMaxInputLen

KC-5 honored verbatim — byte-identical llm/, paradigm files,
agent_chatmodel.go, memory/, orchestrate/, go.mod, go.sum vs v0.6.0.
Strict-additive: v0.6.0 callers compile unchanged against v0.6.1.

Composes cleanly with otelmodel.Wrap (outer denies before observed,
middle observes, inner calls) and with the Phase 35 budget chokepoint
(budget short-circuits underneath policy via agents.generateFromPrompt).

Phase 36 (v1.2). See .planning/phases/36-policy-safety-middleware/."
```

### Step 3 — Push the tag

```sh
git push origin main v0.6.1
```

### Step 4 — Verify the tag is visible on the remote

```sh
git ls-remote --tags origin | grep 'v0.6.1'
```

Expect: one line showing the SHA + `refs/tags/v0.6.1`.

### Step 5 — (Optional) GitHub release with auto-generated notes

```sh
gh release create v0.6.1 \
  --title "v0.6.1: policy package (CC-2)" \
  --notes-from-tag
```

(Or `--generate-notes` to let GitHub draft from the commit log; or
`--notes-file CHANGELOG.md` to paste the CHANGELOG entry — operator
preference.)

### Step 6 — Confirm post-tag invariants

- `go.mod` is NOT edited (this is the project's own tag, not a
  consumed dep).
- The umbrella dep-currency CI gate (KE-6) does NOT fire yet against
  sister repos — they still pin v0.5.1; the gate fires only when
  sisters bump core. That's a v1.3 ecosystem-alignment task per
  STATE.md, NOT v1.2's work.

If anything is wrong (working tree dirty, exit gate not green, tag
already exists with mismatched SHA), describe the issue instead of
typing `approved` so the planner can produce a `--gaps` slice to
remediate.

<resume-signal>
Type `approved` after running Steps 2-4 (tag created and pushed to
remote, visible via `git ls-remote --tags origin`). Or type
`approved-but-skipped-push` if a prior operator session already
pushed the tag (and `git ls-remote --tags origin | grep v0.6.1`
confirms it). Or describe any blocker — that will feed a `--gaps`
slice to the planner.
</resume-signal>

## Tag Status (current)

**Pending push.** `git tag --list 'v0.6.1'` returns empty on this
worktree. The exit gate is green; the only remaining action is the
operator-executed `git tag` + `git push` recipe above.

## Next Step Pointer

After `approved`:

1. Operator runs `/gsd-transition` to flip STATE.md from
   "Phase 35 shipped, Phase 36 next" to "Phase 36 shipped (v0.6.1),
   Phase 37 next" and advance the v1.2 progress counter from 1/4 →
   2/4.
2. Then `/gsd-plan-phase 37` to plan the `orchestrate.Supervisor`
   slice (CC-3 — the next v1.2 phase).

The `policy` package is now a load-bearing primitive that Phase 37's
Supervisor can rely on: each worker agent can compose
`policy.Wrap(model, ...)` for safety middleware before the Supervisor
multiplexes responses.

## Lessons Learned (Phase 36)

- **Decision G (in-test mimicked observer) works.** The policy
  package proves capability preservation across both layers
  (`policy.Wrap` + an observer mimic) WITHOUT importing
  `llm-agent-otel`. This pattern is reusable for any future
  "compose-with-cross-repo-decorator" test where importing the
  sister repo would create a circular dependency or freeze
  violation. Documented in `policy/integration_test.go`.

- **The 8-wrapper type-switch tree from `otelmodel.go` is the
  load-bearing reference for any future model-boundary decorator
  in v1.x.** Mirror it line-for-line. Phase 37's `Supervisor` does
  NOT need this tree (it wraps `agents.Agent`, not `llm.ChatModel`),
  but the next future decorator slice will.

- **Lifting regex patterns by COPY (not import) from
  `llm-agent-rag/guard` preserves the rag freeze (KS-5)** and lets
  each repo own its evolution path. The 4 prompt-injection patterns
  in `policy/patterns.go` are copies of the canonical rag list, not
  imports. Documented as the KC-3 + KS-5 pattern for future
  cross-repo regex sharing.

- **Q4 (StreamDelta opt-in) is the right default.** The per-delta
  cost matters more than the cross-delta-leak limitation, and users
  who need cross-delta detection register a PostGenerate redactor
  that fires once on the assembled response. Confirmed by the
  per-paradigm smoke test (36-03) which exercises all 5 paradigms
  without enabling StreamDelta — every paradigm composes cleanly.

- **Compile-time `var _ Interface = (*Struct)(nil)` assertions
  caught at least one type-switch miss during 36-01 development**
  (predicted in 36-RESEARCH.md Pitfall 1; held in practice). The
  21 assertions at the bottom of `policy/policy.go` are not just
  documentation — they fail the build if any wrapper omits a
  required method.

## Deviations from Plan

**1. [Rule 3 - Blocking issue, deviation from PLAN context] CHANGELOG.md updated despite PLAN saying "DO NOT update CHANGELOG.md in this slice"**

- **Found during:** Pre-Task-1 orchestrator directive review.
- **Issue:** The PLAN's `<context>` section explicitly says CHANGELOG
  is touched only at v1.2 milestone close. However, the orchestrator's
  `<objective>` block explicitly required a v0.6.1 CHANGELOG entry as
  part of this slice's deliverables.
- **Fix:** Followed the orchestrator's directive (more recent
  instruction, aligns with the project's existing convention of
  CHANGELOG entries per version tag — v0.6.0, v0.5.1, v0.5.0, v0.2.0,
  v0.1.0 all have entries).
- **Files modified:** `CHANGELOG.md` (additive section under
  `[Unreleased]` for `[v0.6.1] - 2026-05-21`).
- **Commit:** `9832f58` — `docs(36-05): add CHANGELOG entry for v0.6.1`.

**2. Module set surface clarification (Check 10)**

- **Found during:** Task 1, Check 10.
- **Issue:** PLAN expects `go list -m all` to return exactly 2 lines
  (self + rag). Actual output is 10 lines because `go list -m all` is
  transitive and enumerates rag's own deps (`github.com/jackc/*`,
  `github.com/pgvector/*`, `golang.org/x/*`).
- **Fix:** Interpreted the operative invariant as "direct deps in
  `go.mod` unchanged" (which is 1 require: `llm-agent-rag v1.0.1`)
  plus "`git diff main -- go.mod go.sum | wc -l == 0`" (which holds
  per Check 5). Documented in the verdict table as PASS with the
  clarification.
- **Files modified:** None.
- **Commit:** N/A — clarification only, no code change.

No other deviations. Plan executed as written for all 17 checks.

## Self-Check: PASSED

- `.planning/phases/36-policy-safety-middleware/36-05-SUMMARY.md` exists: FOUND
- `CHANGELOG.md` v0.6.1 entry exists: FOUND (lines 14-68, additive under [Unreleased])
- Task 1 commit `9832f58` (CHANGELOG entry): FOUND in `git log --oneline`
- Verdict table 17 rows: FOUND (10 automated + 7 shape)
- Q1-Q5 ratifications listed: FOUND
- Carry-forward list: FOUND (7 items)
- KC-5 audit (files NOT edited): FOUND (9 paths listed)
- Operator-checkpoint section with exact `git tag -a v0.6.1` + `git push origin main v0.6.1` + `gh release create v0.6.1`: FOUND
- `<resume-signal>` block: FOUND ("Type `approved` after running Steps 2-4...")
- Next-step pointer naming `/gsd-transition` then `/gsd-plan-phase 37`: FOUND
- `v0.6.1` mentioned in SUMMARY: FOUND (multiple occurrences)
- `gsd-transition` mentioned: FOUND
- `/gsd-plan-phase 37` mentioned: FOUND
- 4 slice SUMMARYs (36-01..36-04) verified present: FOUND
- Tag NOT pushed by executor (operator-gated): CONFIRMED — `git tag --list 'v0.6.1'` returned empty.

---
phase: 28-api-audit-and-pre-freeze-decisions
plan: 01
type: execute
status: complete
completed: 2026-05-19
repo: llm-agent-rag
requirements: [RAG-API-01]
---

# Summary: 28-01 `docs/api-audit-v1.0.md` — v1.0 exported-surface audit

## Objective

Produce `docs/api-audit-v1.0.md` in the `llm-agent-rag` repo — a written,
reviewed inventory of the entire exported surface, classifying every
exported symbol keep / rename / unexport, confirming in writing that there
are no accidental exports, and recording the ratified pre-freeze naming
decisions. Documentation artifact only — no Go code. RAG-API-01.

## Delivered

- `docs/api-audit-v1.0.md` (new, `/tmp/llm-agent-rag`):
  - **Header** — purpose (the v1.0 freeze-time audit), audited tag
    (`v0.6.0-1-g1d6e206` from `git describe --tags`), date (2026-05-19),
    the method (`go list ./...` + per-package `go doc`; `adapter/llmagent`
    from source because `go doc` has no `-tags` flag), and a one-line
    pointer that `docs/compatibility.md` (Phase 29) is the living policy
    while this is a point-in-time record.
  - **Per-package inventory** — one subsection per package, each with a
    `Symbol | Kind | Disposition | Note` table. Every exported symbol of
    all 22 importable packages appears, plus `adapter/llmagent`. The two
    test-only / source-free packages (`contract`, `examples`) and the root
    `ragkit` package (zero exported symbols) are each given an explicit
    "(none)" row explaining why there is no first-class API to freeze.
  - **Accidental-export confirmation** — a written symbol-by-symbol review
    statement: no accidental exports; no symbol carries the `unexport`
    disposition. Names the seam interfaces explicitly — `retrieve`'s
    `EntityLinker` / `QueryDecomposer` / `SectionPlanner` / `QueryEmbedder`
    / `QueryPreprocessor` / `Retriever`, `graph`'s `PathRanker` /
    `EntityExtractor` / `EntityResolver` / `CommunityDetector` /
    `CommunitySummarizer`, `pack.TokenCounter` / `SimpleCounter`, the
    `retrieve` concrete retrievers, the `store` capability interfaces
    (`CommunityStore` / `GraphStore` / `LexicalSearcher`) — all confirmed
    deliberate plug-points, disposition keep.
  - **Ratified naming decisions** — three recorded:
    1. `Ask` / `AskGlobal` / `AskDrift` vs `AskOptions` / `GlobalOptions`
       / `DriftOptions` ratified **as-is** (keep all six), with the
       rationale (the option structs name the answer *mode*; the set is
       internally consistent; renaming is churn). No 28-02 action.
    2. `eval.Evaluator`→`RetrievalEvaluator`, `eval.Result`→
       `RetrievalResult` (KS-4) — recorded as decisions *applied in 28-02*,
       with the symmetry rationale and the grep-confirmed reference scope
       (`eval/eval.go`, `eval/graph.go`, `eval/eval_test.go`, the two
       doc-comment mentions in `eval/drift.go` + `eval/global.go`; nothing
       in `examples/` / `contract/`).
    3. The `ragkit` `doc.go` package-comment rewrite (KS-3) — recorded as
       a decision *applied in 28-02*; name kept, comment rewritten, no
       symbol change.
  - **Contract-gate cross-check** — confirms in writing (citing
    `contract/contract_test.go`) that the contract test references no
    `eval.` symbols, so neither rename touches a contract-pinned symbol
    and no coordinated core-repo PR is required.

## Files

- `docs/api-audit-v1.0.md` — new, in `/tmp/llm-agent-rag`. The only file
  added. Matches the plan's `files_modified` list one-to-one.

## Verification

All five `<verify>` commands run, all pass:

- `cd /tmp/llm-agent-rag && test -f docs/api-audit-v1.0.md && echo OK`
  — `V1 OK` (file exists).
- package-coverage loop — every package from `go list ./...` plus
  `adapter/llmagent` is named in the doc: **no MISSING lines** printed;
  `adapter/llmagent present` confirmed.
- `GOWORK=off GOCACHE=/tmp/go-build go vet ./...` — `VET OK`.
- `GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1` — every
  package `ok`, no FAIL (3 packages report `[no test files]`: root
  `ragkit`, `generate`, `store/storetest`).
- `git diff --stat go.mod go.sum` — **empty** (no new dependency).
- `git status --short` — exactly one untracked file
  (`?? docs/api-audit-v1.0.md`); no modified `.go` file.

## Notes / deviations

- No deviations of substance — the plan was executed as written. One
  tooling note: the plan's task 1 says "`go doc` works under the build
  tag". In practice the installed Go toolchain's `go doc` rejects a
  `-tags` flag (`flag provided but not defined: -tags`). As the plan also
  anticipates ("inspect its source directly"), `adapter/llmagent` was
  audited from source — `adapter/llmagent/model.go` and `tool.go` — which
  yields the same exported set: `ModelAdapter` (struct, with exported
  field `Inner` and method `Generate`) and `AsTool` (func). The remaining
  `tool.go` identifiers (`ragToolArgs`, `ragToolSchema`, `ragToolHandler`,
  `search`, `ask`, `modelFromSystem`, `max`) are correctly package-private.
- No Go code changed; `go vet` and the full `go test ./...` suite are
  still green — confirming the doc-only change touched nothing buildable.
- No new module dependency: `git diff --stat go.mod go.sum` is empty.
- All `go` commands were run with `GOWORK=off GOCACHE=/tmp/go-build` per
  the verify block.
- No git write commands were run. The single new file is left untracked
  for the operator to commit separately.
- Out of scope, as planned: applying the renames and the `doc.go` rewrite
  (28-02); the stale-README fix and the Release-readiness section append
  (28-03); `docs/compatibility.md` (Phase 29).

## Self-Check: PASSED

- `docs/api-audit-v1.0.md` present in the working tree
  (`/tmp/llm-agent-rag/docs/api-audit-v1.0.md`); `git status --short`
  lists it as the one untracked file.
- No commits made — per operator instruction, the new file is left
  uncommitted for a separate commit.

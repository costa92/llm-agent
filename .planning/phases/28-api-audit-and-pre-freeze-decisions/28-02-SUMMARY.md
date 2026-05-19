---
phase: 28-api-audit-and-pre-freeze-decisions
plan: 02
type: execute
status: complete
completed: 2026-05-19
repo: llm-agent-rag
requirements: [RAG-API-02]
---

# Summary: 28-02 — apply the ratified pre-freeze renames

## Objective

Apply the two pre-freeze breaking changes ratified when the v1.0
API-stabilization milestone was opened: rename the `eval` base evaluator
`Evaluator`/`Result` → `RetrievalEvaluator`/`RetrievalResult` (KS-4), and
rewrite the `ragkit` `doc.go` package comment to document the root as a
deliberate documentation anchor (KS-3). After `v1.0.0` these are
impossible without a `/v2` — this slice is the last opportunity.
RAG-API-02.

## Delivered

- `eval/eval.go`:
  - `type Result struct` → `type RetrievalResult struct`. The doc comment
    now starts with the new name (Go convention) and describes the
    retrieval recall/MRR/precision trace role.
  - `type Evaluator struct` → `type RetrievalEvaluator struct`. The doc
    comment starts with the new name and now states the symmetry with the
    three name-prefixed answer-side evaluators (`GlobalEvaluator`,
    `DriftEvaluator`, `TriadEvaluator`).
  - `func (e Evaluator) Run` → `func (e RetrievalEvaluator) Run`; the
    method name `Run` is unchanged. Return type and all three internal
    return literals (`Result{}` guard returns + the final populated
    `Result{...}`) updated to `RetrievalResult`.
- `eval/graph.go`:
  - The two `Evaluator{Retriever:..., Options:...}` constructions inside
    `RunGraphAB` (the GraphRAG-off and GraphRAG-on arms) updated to
    `RetrievalEvaluator{...}`.
- `eval/eval_test.go`:
  - `eval.Evaluator{...}` → `eval.RetrievalEvaluator{...}` at every site
    (the baseline-metrics CI gate `ev`, plus the two rejection tests).
  - Test funcs renamed for consistency:
    `TestEvaluatorRunRejectsNilRetriever` →
    `TestRetrievalEvaluatorRunRejectsNilRetriever`,
    `TestEvaluatorRunRejectsZeroTopK` →
    `TestRetrievalEvaluatorRunRejectsZeroTopK`.
- `eval/drift.go`, `eval/global.go`:
  - The doc-comment phrase "RunGraphAB / Evaluator measure that for the
    local path" updated to "RunGraphAB / RetrievalEvaluator measure …" in
    both `DriftEvaluator` and `GlobalEvaluator` type comments, so the
    prose stays accurate.
- `doc.go`:
  - `package ragkit` and the file are kept; the package comment is
    rewritten. It now states `ragkit` is the SDK's short brand name, that
    the module path is `github.com/costa92/llm-agent-rag`, that the root
    package is a deliberate documentation anchor exporting no symbols, and
    that callers import the sub-packages directly. The `ragkit` ≠
    module-name divergence is recorded as a deliberate decision, not an
    accidental mismatch. No exported symbols introduced.

All six files match the plan's `files_modified` list one-to-one. No new
file was created.

## Verification

All `<verify>` commands run, all green:

- `cd /tmp/llm-agent-rag && GOWORK=off GOCACHE=/tmp/go-build go build ./...`
  — BUILD OK
- `cd /tmp/llm-agent-rag && GOWORK=off GOCACHE=/tmp/go-build go vet ./...`
  — VET OK
- `cd /tmp/llm-agent-rag && GOWORK=off GOCACHE=/tmp/go-build go test ./eval/... -count=1`
  — `ok github.com/costa92/llm-agent-rag/eval`
- `cd /tmp/llm-agent-rag && GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1`
  — all 22 packages `ok`, no FAIL
- `go doc ./eval RetrievalEvaluator` / `go doc ./eval RetrievalResult` —
  both render with the new names; `(RetrievalEvaluator).Run` returns
  `RetrievalResult`.
- `! grep -rn 'eval\.Evaluator\b\|eval\.Result\b' --include=*.go .` —
  "OLD NAMES GONE": no qualified `eval.Evaluator` / `eval.Result`
  reference remains anywhere.
- `go doc . | head -5` — root package comment renders the new doc-anchor
  wording ("the root package is a deliberate documentation anchor only:
  it exports no symbols").
- `git diff --stat go.mod go.sum` — empty (no new dependency).
- `go test ./contract/... -count=1` — `ok github.com/costa92/llm-agent-rag/contract`
  (the cross-repo compile-pin gate is green; renames touched no
  contract-pinned symbol).
- core facade smoke (from the core repo `llm-agent`):
  `GOWORK=off go vet ./rag/... && go test ./rag/...` — VET OK,
  `ok github.com/costa92/llm-agent/rag`.

Defensive grep (Task 5):
`grep -rn --include='*.go' '\bEvaluator\b\|\bResult\b' . | grep -v 'Retrieval\|Global\|Drift\|Triad'`
returned only unrelated identifiers in other packages — `agentic.Result`
(the `CorrectiveAsker` outcome type) and `pack.Result` (the packer output
type). No bare `Evaluator` remains. The rename is contained entirely to
the `eval` package, as the research predicted.

## Notes / deviations

- No deviations — the plan was executed exactly as written. The
  `files_modified` list matches one-to-one; no extra file was needed.
- Scope held to the two ratified renames only (KS-2: v1.0 freezes, it
  does not redesign). The four-evaluator unification refactor was NOT
  attempted; `GlobalEvaluator`, `DriftEvaluator`, `TriadEvaluator` and
  their result types are untouched except for the one doc-comment phrase.
  The method name `Run` is preserved.
- No new module dependency: `git diff --stat go.mod go.sum` is empty.
  `go doc` / `go list` are toolchain commands, not deps.
- The untracked `docs/api-audit-v1.0.md` from slice 28-01 was left
  untouched and remains untracked, as instructed.
- No git write commands were run — all changes are left uncommitted in
  `/tmp/llm-agent-rag` (branch `master`) for the operator to commit
  separately.
- Out of scope as planned: the README `Not implemented yet:` correction
  (28-03), `docs/compatibility.md` (Phase 29), the `CHANGELOG.md`
  `[v1.0.0]` entry (30-03).

## Self-Check: PASSED

- `eval/eval.go`, `eval/graph.go`, `eval/eval_test.go`, `eval/drift.go`,
  `eval/global.go`, `doc.go` all show as modified (`M`) in
  `git status --short` for `/tmp/llm-agent-rag`.
- `go doc ./eval RetrievalEvaluator` and `go doc ./eval RetrievalResult`
  confirm the new symbols exist; `! grep eval.Evaluator|eval.Result`
  confirms the old qualified names are gone.
- No commits made — per operator instruction, all changes left
  uncommitted for a separate commit.
</content>
</invoke>

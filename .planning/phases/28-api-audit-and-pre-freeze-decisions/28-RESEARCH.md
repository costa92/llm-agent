# Phase 28 Research: API audit & pre-freeze decisions

**Researched:** 2026-05-20
**Phase:** 28 — API audit & pre-freeze decisions (first v1.0 phase)
**Requirements:** RAG-API-01, RAG-API-02
**Repos:** `llm-agent-rag`
**Upstream:** `.planning/research/v1.0-api-stabilization-SUMMARY.md` §1, §2, §5;
keystones KS-3, KS-4, KS-7.

## Phase goal

Produce a written, reviewed inventory of `llm-agent-rag`'s entire exported
surface and *make and record* every naming/consistency decision that must be
settled before the `v1.0.0` freeze. This phase changes **little code** — it
produces a decision record and a small, bounded set of breaking renames.
After `v1.0.0` every rename needs a `/v2`; Phase 28 is the last chance.

## Current state (codebase scan, `/tmp/llm-agent-rag` @ `v0.6.0-1-g1d6e206`)

- **22 packages via `go list ./...`** (default tags): root (`doc.go`),
  `advanced`, `agentic`, `contract`, `embed`, `eval`, `examples`,
  `feedback`, `generate`, `graph`, `guard`, `ingest`, `obs`, `pack`,
  `postgres`, `prompt`, `rag`, `rerank`, `retrieve`, `store`,
  `store/storetest`, `tree`. Plus `adapter/llmagent` behind `-tags llmagent`.
- **`doc.go`** — declares `package ragkit`, three lines, one package
  comment, **no exported symbols**. The module path is
  `github.com/costa92/llm-agent-rag`; the `ragkit` ≠ module-name mismatch
  is currently undocumented (audit finding A1).
- **`eval` base evaluator** — `eval/eval.go` declares `type Result struct`
  (line 65) and `type Evaluator struct` (line 81); `(Evaluator).Run` returns
  `Result`. The three later evaluators are name-prefixed
  (`GlobalEvaluator`/`GlobalEvalResult`, `DriftEvaluator`/`DriftEvalResult`,
  `TriadEvaluator`/`TriadResult`) — the base pair is the only un-prefixed one.
- **`Evaluator`/`Result` reference sites (verified by grep, repo-wide):**
  - `eval/eval.go` — the declarations + all internal returns.
  - `eval/graph.go:26,32` — `RunGraphAB` constructs `Evaluator{...}`.
  - `eval/eval_test.go:99,143,144,150,151` — `eval.Evaluator{...}` + test names.
  - `eval/drift.go:51`, `eval/global.go:48` — **doc-comment** references
    ("RunGraphAB / Evaluator measure that").
  - **No references in `examples/`, `contract/`, or any other package.**
- **`contract/contract_test.go`** — the cross-repo compile-pin. Confirmed:
  it does **not** reference `eval.Evaluator` or `eval.Result` (no `eval.`
  symbols at all). The Phase-28 renames touch **zero** contract-pinned
  symbols → **no coordinated core-repo PR is needed** (closes the v1.0
  SUMMARY open question). The 28-02 verify still runs the core-facade smoke
  as a safety net.
- **`README.md`** — line 48 `Current status: production-ready core,
  evolving ecosystem.`; lines 147-152 a `Not implemented yet:` list that
  still names `online-to-offline production-feedback workflow (planned in
  slice 13-03)` and `cross-repo contract-drift CI gates (planned in slice
  13-04)` — **both shipped** (`feedback` package + `contract` gate exist).
  Stale (audit finding, §5 of the milestone research).
- **Clean-state confirmed by the milestone audit:** zero
  `TODO`/`FIXME`/`XXX`/`HACK`/`Deprecated` in non-test code; no `replace`
  directives; consistent package-prefixed sentinel errors.

## Decision 1 — the audit inventory is a committed repo doc (RAG-API-01)

`28-01` writes `docs/api-audit-v1.0.md` **in the `llm-agent-rag` repo** — a
point-in-time, reviewed record that ships with the code. It is named
`-v1.0` to signal it is the freeze-time audit, not a living document
(`docs/compatibility.md`, written in Phase 29, is the living policy).

The inventory enumerates every package via `go list ./...` (plus
`adapter/llmagent` with `-tags llmagent`), captures each package's exported
surface via `go doc <pkg>`, and classifies every exported symbol
**keep / rename / unexport**. It records in writing:

- **No accidental exports.** The many small seam interfaces
  (`EntityLinker`, `QueryDecomposer`, `SectionPlanner`, `PathRanker`, …) are
  deliberate plug-points — recorded as keep.
- `pack.TokenCounter` / `pack.SimpleCounter` — a legitimate seam
  (`rag/ask.go` uses `SimpleCounter{}`; a caller may supply a real
  tokenizer) — recorded as keep.
- The `retrieve` concrete-retriever surface (6+ structs, 5+ seams) — the
  frozen `retrieve` surface, recorded as keep.
- **The `Ask`/`AskGlobal`/`AskDrift` vs `AskOptions`/`GlobalOptions`/
  `DriftOptions` naming** — ratified **as-is** (KS-4). Rationale recorded:
  the option structs name the *answer mode*, not the method; the set is
  internally consistent; renaming to `AskGlobalOptions`/`AskDriftOptions`
  would be churn for no clarity gain.
- **The two ratified renames** (Decision 2) — recorded as decisions, to be
  *applied* in 28-02.

This slice writes **no Go code** — it is a doc + the `go doc` evidence.

## Decision 2 — the ratified pre-freeze renames (RAG-API-02, KS-3 + KS-4)

`28-02` applies exactly two breaking changes, both ratified by the operator
when v1.0 was opened:

1. **`eval.Evaluator` → `eval.RetrievalEvaluator`,
   `eval.Result` → `eval.RetrievalResult`** (KS-4). Symmetry with the three
   prefixed answer-path evaluators. `(Evaluator).Run` keeps the method name
   `Run`. Scope (from the grep above): `eval/eval.go` (decls + returns),
   `eval/graph.go` (`RunGraphAB`), `eval/eval_test.go`, and the two
   doc-comment mentions in `eval/drift.go` + `eval/global.go`. **Contained
   entirely to the `eval` package.**
2. **Rewrite the `doc.go` package comment** (KS-3). Keep `package ragkit`
   and the name; rewrite the comment to state `ragkit` is a deliberate
   *documentation anchor* — the SDK's short brand name — and that callers
   import the sub-packages, not the root. Converts the
   `ragkit`-vs-module-name mismatch from an accident into a recorded
   decision. No symbol change; `doc.go` stays exported-symbol-free.

No other rename. KS-2: v1.0 freezes, it does not redesign — the `eval`
four-evaluator *unification* refactor is explicitly rejected (out of scope).

## Decision 3 — the stale-README correction (RAG-API-02, §5)

`28-03` corrects the `README.md` `Not implemented yet:` list (lines
147-152): **remove** `online-to-offline production-feedback workflow` and
`cross-repo contract-drift CI gates` — both shipped. The HTTP service layer
and CLI **stay** on the list (deliberate non-goals, deferred since v0.6).
Each surviving claim is verified against the codebase before the slice
closes.

The README **status line** (line 48 → "stable, v1.0") and the
compatibility-policy link are **Phase 29** work (29-03), not Phase 28 — 28
corrects factual staleness only, it does not re-brand the README.

`28-03` also appends a **Release-readiness** section to
`docs/api-audit-v1.0.md`: the zero-`TODO`/`replace`/dead-code state recorded
as a verified release-readiness fact (re-run the grep sweeps as evidence).

## Slice breakdown

- **28-01** — `docs/api-audit-v1.0.md`: full exported-symbol inventory
  across all 22 importable packages + `adapter/llmagent`; every symbol
  classified keep/rename/unexport; no-accidental-exports confirmed in
  writing; the `Ask*`/`*Options` naming ratified-as-is; the two renames
  recorded as decisions. Doc only — no Go code. (RAG-API-01)
- **28-02** — apply the ratified renames: `eval.Evaluator`→
  `RetrievalEvaluator`, `eval.Result`→`RetrievalResult` (all sites incl.
  test + doc-comment mentions); rewrite the `ragkit` `doc.go` package
  comment. (RAG-API-02)
- **28-03** — correct the stale README `Not implemented yet:` list; append
  the Release-readiness section to `docs/api-audit-v1.0.md`. (RAG-API-02)

## Risks / notes

- **Smallest-possible-code phase by design.** The risk is *inventing*
  refactor work — KS-2 forbids it. If a change is not one of the two
  ratified renames, a stale-doc fix, or a confirmed-accidental export, it
  does **not** belong in Phase 28. The 28-01 inventory's job is to *record
  keep decisions*, not trim the surface.
- The renames are mechanical and contained to `eval` — golden tests in
  `eval/eval_test.go` are renamed alongside, not rewritten.
- No new module dependency — `go doc` and `go list` are toolchain commands.
  `git diff --stat go.mod go.sum` must stay empty in every slice's verify.
- Dependencies: 28-02 and 28-03 both depend on 28-01 (the decisions must be
  recorded before they are applied; 28-03 appends to the 28-01 doc). 28-02
  and 28-03 touch disjoint files (`eval/` + `doc.go` vs `README.md` +
  `docs/api-audit-v1.0.md`) — no conflict.
- The `examples/` were grep-confirmed to hold **no** `eval.Evaluator`/
  `eval.Result` references — 28-02 still greps them defensively before
  closing.

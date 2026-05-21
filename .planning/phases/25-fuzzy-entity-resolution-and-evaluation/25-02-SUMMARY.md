---
phase: 25-fuzzy-entity-resolution-and-evaluation
plan: 02
type: execute
status: complete
completed: 2026-05-19
repo: llm-agent-rag
requirements: [RAG-GRAPH3-06]
---

> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent-rag` current docs.


# Summary: 25-02 eval global-search harness — Triad/Judge over AskGlobal

## Objective

Add a global-search evaluation harness to `eval` — run whole-corpus questions
through `rag.System.AskGlobal` and score each answer with the RAG-Triad
`Judge` (groundedness against the consulted community reports + answer
relevance). No chunk recall@k — global search has no gold chunk set.
Completes RAG-GRAPH3-06.

## Delivered

- `rag.GlobalDiagnostics.ConsultedReports []graph.CommunityReport` — a new
  additive field on the existing `Diagnostics.Global` block. `AskGlobal`
  populates it with the community reports it actually mapped over (the
  lazily-loaded or freshly-generated reports for the selected communities, in
  selection order). It is the answer's grounding context: an evaluator reads
  it straight off the `Answer`, with no `CommunityStore` plumbing.
- `rag/global.go` — `AskGlobal` now sets `ConsultedReports: reports` in the
  returned `Answer.Diagnostics.Global`. `reports` is the exact slice already
  produced by `s.communityReports` (cache hit or lazy summarize), so it is
  free — no extra store reads, no extra generations. Zero behavior change to
  `Answer.Text`, `CommunityIDs`, `MapScores`, or call counts.
- `eval/global.go` — new file, package `eval`:
  - `GlobalAsker` interface —
    `AskGlobal(ctx, question string, opts rag.GlobalOptions) (rag.Answer, error)`.
    `*rag.System` satisfies it. The global-search counterpart of `Asker`,
    kept a separate seam because global search is scored only on its
    generation side.
  - `GlobalEvalResult struct` — `MeanGroundedness`, `MeanAnswerRelevance`,
    `Examples int`, and a `PerExample []GlobalExampleResult` detail slice
    (`Example`, `Answer`, consulted `CommunityIDs`, `Judgement`). Carries
    **no** chunk recall@k / precision@k — documented in the type comment as
    the deliberate difference from `Metrics`/`TriadResult`.
  - `GlobalEvaluator struct { Asker GlobalAsker; Judge Judge; MaxCommunities
    int }` with `Run(ctx, dataset Dataset) (GlobalEvalResult, error)`:
    - errors if `Asker` or `Judge` is nil (mirrors `TriadEvaluator`).
    - per `Example`: `AskGlobal` with `GlobalOptions{Namespace: ex.Namespace,
      MaxCommunities: e.MaxCommunities}`; builds the judge context from
      `Answer.Diagnostics.Global.ConsultedReports` (one "Title: Summary"
      passage per report via `reportContext`); `Judge.Judge` for groundedness
      + answer relevance; accumulates the two means.
    - `Example.GoldDocIDs` / `GoldChunkIDs` are unused — documented in the
      `GlobalEvaluator` doc comment: global search has no chunk-recall notion.
  - `GlobalEvalResult.Summary()` — a compact human-readable scoreboard, like
    `TriadResult.Summary()`.
- `eval/global_test.go` — new file, the global-search CI gate. A
  `globalEvalModel` (a scripted `generate.Model` keyed by `SystemPrompt`
  prefix, serving fixed replies for the summarize / map / reduce steps inside
  `AskGlobal`), a `recordingJudge` (a deterministic word-overlap stub `Judge`
  that records every `JudgeRequest`), and a four-triangle knowledge graph
  detected with `graph.LouvainDetector` into a multi-level hierarchy and
  persisted on an in-memory store. Tests:
  - `TestGlobalEvalGate` — `GlobalEvaluator.Run` scores every example, the
    means land in `[0,1]`, answer relevance is `1.0` (scripted answer on
    topic), groundedness `> 0` (proof the consulted-report context reached
    the judge), every `JudgeRequest` carries non-empty context passages,
    every per-example result records the consulted community IDs, and
    `Summary()` is non-empty.
  - `TestGlobalEvaluatorRequiresAskerAndJudge` — `Run` errors on a nil
    `Asker` and on a nil `Judge`.

## Files

- `eval/global.go` — new; `GlobalAsker`, `GlobalEvaluator`,
  `GlobalEvalResult`, `GlobalExampleResult`, `reportContext`, `Summary`.
- `eval/global_test.go` — new; the scripted-model + scripted-judge CI gate.
- `rag/global.go` — modified; `AskGlobal` populates
  `GlobalDiagnostics.ConsultedReports`.
- `rag/system.go` — modified; added the `ConsultedReports
  []graph.CommunityReport` field to `GlobalDiagnostics`.

Exactly the plan's `files_modified` list — no extra files.

## Verification

All `<verify>` commands run, all green:

- `GOWORK=off GOCACHE=/tmp/go-build go build ./...` — BUILD-OK
- `GOWORK=off GOCACHE=/tmp/go-build go vet ./...` — VET-OK
- `GOWORK=off GOCACHE=/tmp/go-build go test ./eval ./rag/... -count=1` —
  `ok github.com/costa92/llm-agent-rag/eval`,
  `ok github.com/costa92/llm-agent-rag/rag`
- `GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1` — all 21
  packages `ok`, no FAIL
- `git diff --stat go.mod go.sum` — empty (no new module dependency)
- core facade (from the core repo `llm-agent`): `GOWORK=off go vet ./rag/...
  && go test ./rag/...` — `ok github.com/costa92/llm-agent/rag`

## Deviations from plan

None. The `files_modified` list is matched exactly (`eval/global.go`,
`eval/global_test.go`, `rag/global.go`, `rag/system.go`); no extra files were
needed. `eval/global_test.go` builds the test graph/store inline with exported
APIs (`store.NewInMemoryStore`, `UpsertGraph`, `graph.LouvainDetector.Detect`,
`UpsertCommunities`) rather than reusing the `rag` package's unexported
`globalTestGraph` helper — that helper is in `package rag`, unreachable from
`package eval_test`, so the four-triangle shape is reproduced locally.

## Notes

- `ConsultedReports` is the design hinge of Decision 4: the evaluator needs
  the answer's grounding context, and the only place it exists post-`AskGlobal`
  is the `reports` slice. Surfacing it on the `Answer` keeps `GlobalEvaluator`
  free of `CommunityStore` plumbing — it is a pure consumer of `rag.Answer`.
- The harness is fully deterministic with no live calls: the scripted
  `generate.Model` drives the summarizer and the map/reduce steps inside
  `AskGlobal`; the scripted `recordingJudge` replaces `LLMJudge`. The CI gate
  asserts plumbing, not LLM quality — `LLMJudge` itself is exercised by the
  existing `eval/judge_test.go`.
- Global search has no gold chunk set, so `GlobalEvaluator` measures only the
  two generation-side legs of the RAG Triad (groundedness vs. the consulted
  community reports, answer relevance vs. the question). `RunGraphAB` /
  `Evaluator` remain the chunk-recall path for local search; `GlobalEvalResult`
  deliberately carries no recall@k field.
- No new module dependency — `eval/global.go` imports only `rag`, `graph`,
  and stdlib (`context`, `errors`, `fmt`, `strings`); the `graph` import is
  already in the module and used for the `CommunityReport` type.
- Out of scope, per plan: `docs/graphrag.md` (25-03); a comprehensiveness
  judge dimension (v0.9) — the harness reuses `LLMJudge`'s groundedness +
  answer-relevance.

## Self-Check: PASSED

- `eval/global.go` — FOUND
- `eval/global_test.go` — FOUND
- `rag/global.go`, `rag/system.go` — FOUND (modified)
- All `<verify>` commands green; `go.mod`/`go.sum` diff empty.

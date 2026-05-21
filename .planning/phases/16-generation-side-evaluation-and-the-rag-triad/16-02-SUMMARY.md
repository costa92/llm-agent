---
phase: 16-generation-side-evaluation-and-the-rag-triad
plan: 02
type: execute
status: complete
completed: 2026-05-15
repo: llm-agent-rag
requirements: [RAG-EVAL2-02]
---

> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent-rag` current docs.

# Summary: 16-02 TriadEvaluator + RAG-Triad CI gate

## Objective

Assemble the full RAG Triad: a `TriadEvaluator` that runs a dataset through
the complete `Ask` pipeline, scores both retrieval and generation quality,
produces one combined `TriadResult` with a JSONL report and a summary, and
adds a CI gate.

## Delivered

- `eval.Asker` — seam for the full retrieve+generate pipeline
  (`Ask(ctx, question, rag.AskOptions) (rag.Answer, error)`); `*rag.System`
  satisfies it.
- `eval.GenerationMetrics{MeanGroundedness, MeanAnswerRelevance, Examples}` —
  generation-side scoreboard (RAG Triad legs 2 and 3).
- `eval.TriadExampleResult` — per-example detail (example, answer,
  retrieved IDs, judgement).
- `eval.TriadResult{Dataset, Retrieval, Generation, PerExample}` —
  JSON-tagged combined result.
- `eval.TriadEvaluator{Asker, Judge, Options}` — `Run` calls `Ask` per
  example, derives the four retrieval metrics from `answer.Hits` (reusing
  the existing `countMatches`/`firstGoldRank`/`anyOverlap` helpers), builds
  the judge context from hit contents, calls `Judge`, and assembles a
  `TriadResult`. Errors on a nil `Asker`/`Judge` or non-positive `TopK`.
- `eval.WriteJSONL(w, TriadResult)` — one JSON line per example plus a final
  `{"summary": {...}}` line.
- `(TriadResult).Summary()` — compact human-readable scoreboard.
- `eval/triad_test.go` — the RAG-Triad CI gate: a deterministic
  `wordOverlapJudge` stub, the seed corpus + 4-example dataset, asserting
  retrieval thresholds (precision/recall/MRR/grounding), populated
  generation metrics, and `WriteJSONL`/`Summary` output. Plus a nil-Asker/
  nil-Judge error test.

## Files

- `eval/triad.go` — new: `Asker`, `GenerationMetrics`, `TriadExampleResult`,
  `TriadResult`, `TriadEvaluator`, `WriteJSONL`, `Summary`.
- `eval/triad_test.go` — new: `wordOverlapJudge` stub + helpers, the CI
  gate, the nil-dependency error test.

## Verification

All `<verify>` commands run, all green:

- `go build ./...` — BUILD OK
- `go vet ./...` — VET OK
- `go test ./eval -v -run Triad` — `TestTriadGateMeetsBaseline` and
  `TestTriadEvaluatorRequiresAskerAndJudge` PASS
- `go test ./... -count=1` — all 14 packages ok
- `git diff --stat go.mod go.sum` — empty (no new dependency)
- core: `GOWORK=off go vet ./rag/... && go test ./rag/...` (run from the
  core repo `llm-agent`, package `github.com/costa92/llm-agent/rag`) — ok

## Notes

- The retrieval-only `Evaluator` and its gate
  (`TestSeedDatasetMeetsBaselineMetrics`) are untouched — `TriadEvaluator`
  is purely additive.
- The CI gate uses the deterministic `wordOverlapJudge`, not `LLMJudge`:
  live LLM-as-judge quality is not CI-verifiable (no model). The gate
  verifies assembly + retrieval thresholds; `LLMJudge` parsing is covered by
  16-01's unit tests. This mirrors the existing gate's use of `fakeModel`.
- The triad gate re-states the seed corpus/dataset inline rather than
  sharing the retrieval gate's inline fixture — kept self-contained to avoid
  refactoring the existing gate.
- Process note: the first core-smoke run was accidentally executed from the
  `/tmp/llm-agent-rag` working directory (re-testing the standalone `rag`
  package); it was re-run from the core repo to verify the real
  `github.com/costa92/llm-agent/rag` facade. Green.

## Phase 16 status

Both slices (16-01, 16-02) complete. RAG-EVAL2-01 (LLM-as-judge for
groundedness/answer-relevance) and RAG-EVAL2-02 (combined retrieval +
generation `TriadResult`, JSONL report, RAG-Triad CI gate) are delivered.
Phase 16 is complete, with no new module dependency.

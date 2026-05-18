---
phase: 16-generation-side-evaluation-and-the-rag-triad
plan: 01
type: execute
status: complete
completed: 2026-05-15
repo: llm-agent-rag
requirements: [RAG-EVAL2-01]
---

# Summary: 16-01 Judge seam + LLMJudge

## Objective

Add the generation-side judge to the `eval` package: a `Judge` seam that
scores an answer for groundedness and answer relevance, plus an `LLMJudge`
that implements the seam as an LLM-as-judge over `generate.Model`.

## Delivered

- `eval.JudgeRequest{Query, Answer, Context}` — one answer to score.
- `eval.Judgement{Groundedness, AnswerRelevance, Rationale}` — scores in
  `[0,1]`.
- `eval.Judge` interface — `Judge(ctx, JudgeRequest) (Judgement, error)`.
  Caller-supplied, so the eval framework stays vendor-neutral.
- `eval.LLMJudge{Model generate.Model}` — prompts the model with a strict
  RAG-evaluator system prompt asking for a JSON judgement, then parses it.
- `parseJudgement` — lenient parser: takes the first `{` through the last
  `}` as the JSON object (models wrap JSON in prose), `json.Unmarshal`s it,
  and clamps both scores to `[0,1]` via `clamp01`.
- `eval.ErrJudgeModelRequired` — returned by `LLMJudge` with a nil model.

## Files

- `eval/judge.go` — new: the seam, `LLMJudge`, `parseJudgement`, `clamp01`.
- `eval/judge_test.go` — new: clean JSON, prose-wrapped JSON, out-of-range
  clamping, no-JSON error, nil-model error — all with scripted models.

## Verification

All `<verify>` commands run, all green:

- `go build ./...` — BUILD OK
- `go vet ./...` — VET OK
- `go test ./eval -count=1` — ok
- `go test ./... -count=1` — all 14 packages ok
- `git diff --stat go.mod go.sum` — empty (no new dependency)

## Notes

- `LLMJudge` is exercised only with scripted models in tests — live
  LLM-as-judge quality is not CI-verifiable. This matches the existing
  retrieval gate's use of `fakeModel`.
- `parseJudgement` is unexported and reused by no one yet; 16-02's
  `TriadEvaluator` consumes the `Judge` interface, not the parser.
- Score comparison in tests uses `==` against decimal literals — safe
  because `json.Unmarshal("0.9")` yields the same `float64` as the Go
  literal `0.9`.

## Next slice

16-02 — `Asker` seam, `TriadEvaluator` assembling retrieval + generation
metrics into a `TriadResult`, `WriteJSONL` report + `Summary`, RAG-Triad
CI gate.

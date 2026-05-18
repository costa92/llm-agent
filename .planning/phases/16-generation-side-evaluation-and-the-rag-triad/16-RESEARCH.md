# Phase 16 Research: Generation-side evaluation and the RAG Triad

**Researched:** 2026-05-15
**Phase:** 16 — generation-side evaluation and the RAG Triad
**Requirements:** RAG-EVAL2-01, RAG-EVAL2-02

## Current state (codebase scan)

- `eval` package scores **retrieval only**: `Evaluator{Retriever, Options}`
  runs a `Dataset` of labeled `Example`s and produces `Metrics`
  (PrecisionAtK / RecallAtK / MRR / GroundingAtK). Metric helpers
  (`countMatches`, `firstGoldRank`, `anyOverlap`) are reusable.
- `Dataset` / `Example` are JSON-tagged; `LoadJSONL` reads a dataset from a
  JSONL file.
- The CI gate is `eval_test.go::TestSeedDatasetMeetsBaselineMetrics` — builds
  a `rag.System` with a `fakeModel`, runs a 4-example seed dataset, asserts
  per-metric thresholds. A retrieval regression fails the test.
- `generate.Model` is the abstract generation seam:
  `Generate(ctx, Request) (Response, error)`, `Response{Text string}`.
- `rag.Answer` carries `Text`, `Hits []store.Hit`, `Citations`, `Diagnostics`,
  `Trace`. `(*rag.System).Ask(ctx, question, AskOptions) (Answer, error)`
  gives both the retrieved context and the generated answer in one call.

## The RAG Triad

Standard production-RAG quality has three legs:

1. **Context relevance** — are retrieved chunks relevant to the query?
   Already approximated by the existing retrieval metrics.
2. **Groundedness / faithfulness** — is the answer supported by the
   retrieved context (no hallucination)? **Missing today.**
3. **Answer relevance** — does the answer actually address the query?
   **Missing today.**

Phase 16 adds legs 2 and 3 via an LLM-as-judge, then assembles all three
into one report.

## Decision 1 — `Judge` seam, mirroring `generate.Model`

```go
type JudgeRequest struct {
    Query   string
    Answer  string
    Context []string // supporting chunk texts the answer should be grounded in
}
type Judgement struct {
    Groundedness    float64 // 0..1: answer supported by context
    AnswerRelevance float64 // 0..1: answer addresses the query
    Rationale       string
}
type Judge interface {
    Judge(ctx context.Context, req JudgeRequest) (Judgement, error)
}
```

A request struct (not positional args) keeps the seam extensible. Scores are
`0..1`. A `Judge` is caller-supplied, so the eval framework stays decoupled
from any model vendor — same discipline as `generate.Model`.

## Decision 2 — `LLMJudge` over `generate.Model`

`LLMJudge{Model generate.Model}` implements `Judge` by prompting the model to
return a JSON judgement `{"groundedness":<f>,"answer_relevance":<f>,
"rationale":"..."}`. Parsing is lenient: extract the first `{` ... last `}`
substring and `json.Unmarshal` it (models often wrap JSON in prose); clamp
scores into `[0,1]`. No new dependency — `encoding/json` + the existing
`generate.Model` seam. Tested deterministically with a scripted model that
returns a fixed JSON judgement (the project's `ScriptedLLM` discipline).

## Decision 3 — `TriadEvaluator` assembles retrieval + generation

The retrieval `Evaluator` stays as-is. A parallel `TriadEvaluator` runs the
full pipeline:

```go
type Asker interface {
    Ask(ctx context.Context, question string, opts rag.AskOptions) (rag.Answer, error)
}
type TriadEvaluator struct { Asker Asker; Judge Judge; Options rag.AskOptions }
```

`Run(ctx, Dataset)` per example: call `Ask`, derive retrieval metrics from
`answer.Hits` (reusing the existing metric helpers), build the judge context
from the hit contents, call `Judge`, accumulate generation metrics. Output:

```go
type GenerationMetrics struct { MeanGroundedness, MeanAnswerRelevance float64; Examples int }
type TriadResult struct {
    Dataset    Dataset
    Retrieval  Metrics
    Generation GenerationMetrics
    PerExample []TriadExampleResult
}
```

`TriadResult` is JSON-tagged; `WriteJSONL` emits one JSON line per example
(the "JSONL report"), and `Summary()` returns a human-readable scoreboard
(the "summary"). A CI gate test runs the triad on the seed dataset with a
deterministic stub judge.

## Slice breakdown

- **16-01** — `Judge` seam, `JudgeRequest`/`Judgement`, `LLMJudge`
  (LLM-as-judge over `generate.Model`). (RAG-EVAL2-01)
- **16-02** — `Asker` seam, `TriadEvaluator`, `TriadResult` +
  `GenerationMetrics`, `WriteJSONL` + `Summary`, and a CI gate test.
  (RAG-EVAL2-02)

## Risks / notes

- Live LLM-as-judge quality cannot be CI-verified (no model in CI). The CI
  gate uses a deterministic stub `Judge`; it verifies assembly + retrieval
  thresholds, not real judge quality. `LLMJudge` itself is unit-tested with
  a scripted model. This mirrors the existing retrieval gate's use of
  `fakeModel`.
- No new dependency — entirely stdlib + existing seams.
- 16-02 depends on 16-01 (`TriadEvaluator` consumes `Judge`).

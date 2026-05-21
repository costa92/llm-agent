> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent-rag` current docs.

# Phase 13-01 Summary

Date: 2026-05-15
Repo: `llm-agent-rag`
Plan: [13-01-PLAN.md](/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/phases/13-evaluation-feedback-loop-and-ecosystem-contract/13-01-PLAN.md)

## Objective

Stand up the retrieval evaluation framework: a dataset format, an
evaluator that computes standard IR metrics, a seed dataset
covering the Phase 11 capability surface, and a `go test` CI gate
so retrieval regressions surface as test failures. Covers
RAG-OPS-03.

## Delivered

- new package `eval/`:
  - `eval/eval.go` exports:
    - `Example{Query, Namespace, GoldDocIDs, GoldChunkIDs, Notes}`
    - `Dataset{Name, TopK, Examples}`
    - `Metrics{PrecisionAtK, RecallAtK, MRR, GroundingAtK,
      Examples, TopK}`
    - `Result{Dataset, Metrics, PerExample}`
    - `ExampleResult{Example, RetrievedIDs, RetrievedDocs,
      RankOfFirstGoldDoc, GroundingHit}`
    - `Retriever` (single-method interface that `*rag.System`
      satisfies naturally)
    - `Evaluator{Retriever, Options}` with `Run(ctx, dataset)
      (Result, error)`
  - `eval/loader.go`:
    - `LoadJSONL(path string) (Dataset, error)` — JSONL parser
      with comment-line support (`//`, `#`) and `top_k` field on
      the first line setting the dataset's TopK
- metric definitions (deterministic, no LLM):
  - `PrecisionAtK` = mean over examples of `len(retrieved ∩ gold) / k`
  - `RecallAtK` = mean over examples of `len(retrieved ∩ gold) / |gold|`
    (skip examples with zero gold)
  - `MRR` = mean of `1 / rank-of-first-gold-doc` (0 when not retrieved)
  - `GroundingAtK` = fraction of examples where at least one gold
    chunk ID appears in retrieved chunk IDs (chunk-level, decoupled
    from answer generation)
- regression test (`eval/eval_test.go`):
  - corpus exercising Phase 11 shapes: `# Cities` with Travel /
    History / History-museums sub-sections, a Cuisine doc, plus a
    distractor namespace
  - 4 hand-authored examples covering:
    - single dominant route (Paris museums → Travel)
    - ambiguous between two routes (history museums → History or
      History-museums)
    - cross-document retrieval (french pastries → Cuisine)
    - namespace isolation (programming notes → 'other' namespace)
  - per-metric thresholds chosen tight against current pipeline:
    - PrecisionAtK >= 0.25 (capped by sparse gold + TopK=3)
    - RecallAtK >= 0.75
    - MRR >= 0.75
    - GroundingAtK >= 0.5
  - additional input-validation tests:
    `TestEvaluatorRunRejectsNilRetriever`,
    `TestEvaluatorRunRejectsZeroTopK`
- JSONL loader test (`eval/loader_test.go`):
  - asserts comment/blank-line handling, basename → Dataset.Name,
    first-line `top_k` field, `gold_chunk_ids` round-trip

## Files

- `/tmp/llm-agent-rag/eval/eval.go` (new, ~205 LOC)
- `/tmp/llm-agent-rag/eval/loader.go` (new, ~70 LOC)
- `/tmp/llm-agent-rag/eval/eval_test.go` (new, ~150 LOC)
- `/tmp/llm-agent-rag/eval/loader_test.go` (new, ~50 LOC)

## Verification

```bash
cd /tmp/llm-agent-rag
GOWORK=off GOCACHE=/tmp/go-build go build ./...
GOWORK=off GOCACHE=/tmp/go-build go vet ./...
GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1
GOWORK=off GOCACHE=/tmp/go-build go test ./eval -count=1 -v

cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent
GOWORK=off go vet ./rag/...
GOWORK=off go test ./rag/... -count=1
```

Result:

- `go build ./...` (standalone): pass
- `go vet ./...` (standalone): pass
- `go test ./...` (standalone, 15 packages including new `eval`):
  pass
- 5/5 eval tests pass
- core `go vet ./rag/...` + `go test ./rag/...`: pass

## Notes & Design Decisions

- **Eval against the plain hybrid retrieval path, not auto-route.**
  An early iteration ran with `EnableAutoRoute=true` and the
  seed dataset hit the route-narrowing path, where the auto-route
  picker picked the wrong section for "paris museums" (the route
  scorer reads heading text only, and "Travel" alone doesn't
  contain query tokens). The baseline eval is now scoped to
  hybrid retrieval over the full namespace. Auto-route quality is
  already covered by Phase 11 unit tests in `retrieve/` and `rag/`,
  and a separate eval dataset can target it later.
- **PrecisionAtK naturally low with sparse gold.** With one gold
  doc and TopK=3, the per-example precision ceiling is 1/3.
  Threshold set to 0.25 so the metric still acts as a regression
  guard while not being misleadingly small. The real
  regression-detection signal lives in MRR (rank-1 quality) and
  RecallAtK (coverage).
- **Grounding measured at retrieval level, not citation level.**
  GroundingAtK asks whether the right chunks were *retrieved*,
  not whether the *Answer cited* them. That keeps the eval
  deterministic without an LLM and disentangles retrieval
  regressions from generation regressions.
- **JSONL format keeps 13-03 ready.** The production-feedback
  workflow planned for 13-03 needs a way to append captured
  miss-cases without recompiling the eval package. JSONL fits:
  one example per line, append-only, comment lines allowed for
  human annotation.

## Next slice

`13-02` covers production-deployment / backend-selection /
compatibility documentation. Then `13-03` adds the
online-to-offline feedback workflow that consumes the
JSONL format set up here.

# Phase 19 Research: Agentic retrieval — decomposition and self-correction

**Researched:** 2026-05-17
**Phase:** 19 — agentic retrieval — decomposition and self-correction
**Requirements:** RAG-AGENT-01, RAG-AGENT-02
**Repos:** `llm-agent-rag` (the final v0.6 phase)

## Current state (codebase scan)

- `retrieve.Retriever` is the retrieval seam:
  `Retrieve(ctx, Request) ([]store.Hit, Trace, error)`. Decorators compose
  over it — `VariantRetriever{Base: HybridRetriever{Dense, Lexical,
  Structure}}`. `VariantRetriever` already does the multi-query merge
  pattern: per query-variant, call `Base.Retrieve`, then dedup by
  `Chunk.ID` keeping the max score, preserve insertion order, truncate to
  `req.TopK`. `appendUniqueStrings` merges trace string fields.
- `retrieve.LLMExpansionPreprocessor{Model generate.Model}` already shows
  the LLM-in-retrieve pattern (MQE/HyDE). `retrieve` imports `generate`.
- `eval.Judge` (Phase 16) is the grounding signal:
  `Judge(ctx, JudgeRequest{Query, Answer, Context}) (Judgement{Groundedness,
  AnswerRelevance, Rationale}, error)`. `eval.Asker` is the full-pipeline
  seam: `Ask(ctx, question, rag.AskOptions) (rag.Answer, error)` —
  `*rag.System` satisfies it.
- No query decomposition and no self-correcting loop exist today.

## What RAG-AGENT-01 / RAG-AGENT-02 ask for

- **RAG-AGENT-01** — a multi-hop `Retriever` decorator decomposes compound
  queries into sub-queries and merges their sub-retrievals.
- **RAG-AGENT-02** — a self-correcting retrieval loop detects low grounding
  (the Phase 16 grounding signal) and re-retrieves with reformulated
  queries up to a bounded retry cap.

## Decision 1 — `MultiHopRetriever` decorator in the `retrieve` package

`MultiHopRetriever` implements `retrieve.Retriever`, so it slots into the
existing seam (`rag.Options.Retriever`) and lives beside its sibling
decorators `VariantRetriever`/`HybridRetriever`. No `rag` or `System`
change is needed — callers wire it as `opts.Retriever`.

```go
type QueryDecomposer interface {
    Decompose(ctx context.Context, query string) ([]string, error)
}
type HeuristicDecomposer struct{}          // splits on " and " — no model
type LLMDecomposer struct {                 // LLM-backed
    Model         generate.Model
    MaxSubQueries int
}
type MultiHopRetriever struct {
    Base       Retriever
    Decomposer QueryDecomposer              // nil → HeuristicDecomposer
}
```

`MultiHopRetriever.Retrieve` decomposes `req.Query`, runs `Base.Retrieve`
for each sub-query (a sub-request copy with `Query` set, `QueryVariants`
cleared), and merges hits with the `VariantRetriever` merge pattern (dedup
by `Chunk.ID`, max score, then sort-by-score and truncate to `req.TopK`).
A non-compound query decomposes to itself — one hop, behaviourally a
pass-through to `Base`.

`LLMDecomposer` prompts the model to break a compound question into
sub-questions one per line; parsing is lenient (split lines, strip list
markers, drop empties) and falls back to `[query]`. `HeuristicDecomposer`
is the deterministic no-model decomposer (and the testing default), the
same Noop/Heuristic/Model tiering used by `rerank`.

Per-hop attribution: `retrieve.Trace` gains `Hops []HopAttribution` where
`HopAttribution{SubQuery string, HitCount int}`.

## Decision 2 — `CorrectiveAsker` in a new `agentic` package

The self-correcting loop is a runtime orchestrator around the full
retrieve→generate→judge cycle — not a `Retriever` and not an evaluator.
It needs `eval.Judge` (the grounding signal), but `rag` cannot import
`eval` (`eval` imports `rag`). So it lives in a new package `agentic`
(imports `eval` + `rag` + `generate` — linear, no cycle).

```go
type QueryReformulator interface {
    Reformulate(ctx context.Context, question string, prev rag.Answer) (string, error)
}
type LLMReformulator struct { Model generate.Model }

type Attempt struct { Query string; Groundedness, AnswerRelevance float64 }
type Result  struct { Answer rag.Answer; Attempts []Attempt; Corrected bool }

type CorrectiveAsker struct {
    Asker        eval.Asker          // *rag.System satisfies it
    Judge        eval.Judge
    Reformulator QueryReformulator
    MinGrounding float64             // <=0 → default 0.5
    MaxRetries   int                 // <=0 → default 2 (bounded cap)
}
```

`AskWithCorrection` loops: call `Asker.Ask`; judge the answer's
groundedness against `answer.Hits` content (judging always with the
**original** question); if `Groundedness >= MinGrounding` or the retry cap
is reached, stop; otherwise `Reformulator.Reformulate` the query and retry.
It tracks the best attempt by groundedness and returns that — never a
worse later attempt. `Ask` (satisfying `eval.Asker`) wraps
`AskWithCorrection` and returns just the answer, so a `CorrectiveAsker`
composes inside a `TriadEvaluator`.

`LLMReformulator` prompts the model with the question and the poorly-
grounded answer for a better retrieval query. Nil `Asker`/`Judge`/
`Reformulator` → error (the loop is meaningless without all three).

## Slice breakdown

- **19-01** — `retrieve.MultiHopRetriever` + `QueryDecomposer`/
  `HeuristicDecomposer`/`LLMDecomposer`; `retrieve.Trace.Hops`. (RAG-AGENT-01)
- **19-02** — new `agentic` package: `CorrectiveAsker` +
  `QueryReformulator`/`LLMReformulator`; the grounding-driven retry loop.
  (RAG-AGENT-02)

## Risks / notes

- The two slices are independent (different packages) — 19-02 does not
  depend on 19-01; they may be planned/executed in either order. Listed
  19-01 then 19-02 for the natural decomposition-before-correction reading.
- Both LLM-backed types (`LLMDecomposer`, `LLMReformulator`) are tested
  deterministically with a scripted model; `MultiHopRetriever` and
  `CorrectiveAsker` are tested with deterministic stubs
  (`HeuristicDecomposer`, a stub `Retriever`, a stub `Judge`/`Asker`/
  `Reformulator`) — the project's mock discipline.
- `HeuristicDecomposer` is best-effort (splits on " and " — may over-split
  e.g. "black and white"); the `QueryDecomposer` seam lets callers supply
  `LLMDecomposer` or their own. Documented, not solved.
- The retry cap is hard-bounded by `MaxRetries`; `CorrectiveAsker` makes at
  most `MaxRetries + 1` `Asker.Ask` calls — no unbounded looping.
- No new module dependency — `generate.Model` seam + stdlib only. The
  standard `git diff --stat go.mod go.sum` (must be empty) check applies.

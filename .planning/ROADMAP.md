# Roadmap: llm-agent

**Last updated:** 2026-05-17
**Current state:** `v0.6` retrieval-quality milestone — all six phases
(14-19) executed; pending milestone-close (commit, tag, audit)
**Active scope:** `v0.6` production-grade retrieval quality and safety

## Archived Milestones

- [x] **v0.3: Deployable multi-repo release** — shipped 2026-05-12.
  Delivered `llm/v2`, three real provider adapters, OTel wrappers, and the
  customer-support demo stack across 4 repos.
  - Archive: `.planning/milestones/v0.3-ROADMAP.md`
  - Requirements archive: `.planning/milestones/v0.3-REQUIREMENTS.md`
  - Audit: `.planning/v0.3-MILESTONE-AUDIT.md`
- [x] **v0.5: RAG productionization and standalone SDK evolution** — shipped
  2026-05-15. Delivered structure-aware retrieval, a PostgreSQL + pgvector
  backend with a shared conformance suite, tracing hooks, an evaluation
  framework, a feedback loop, and cross-repo contract gates. `llm-agent-rag`
  tagged `v0.2.0`.
  - Archive: `.planning/milestones/v0.5-ROADMAP.md`
  - Requirements archive: `.planning/milestones/v0.5-REQUIREMENTS.md`

## Milestone v0.6: Production-grade retrieval quality and safety

**Goal**: deepen the six retrieval-quality seams that v0.5 left thin —
lexical/hybrid retrieval, reranking, evaluation, observability, content
safety, and agentic retrieval — turning minimal interfaces into
production-grade implementations. This is a quality milestone, not a
packaging one: no new deployment surface.

**Repos**: `llm-agent-rag` (primary), `llm-agent-otel` (RED metrics wiring),
`llm-agent` (compatibility-facade lockstep)

**Requirements in scope**:

- `RAG-RETR2-01..02`
- `RAG-RERANK-01..02`
- `RAG-EVAL2-01..02`
- `RAG-OBS-01..02`
- `RAG-SEC-01..02`
- `RAG-AGENT-01..02`

## Active Forward Work

### Phase 14: Lexical retrieval and principled hybrid fusion

**Status**: complete 2026-05-15

**Goal**: replace token-overlap lexical scoring with a real BM25 model and
fuse dense/lexical/structure signals through a principled method with
per-signal score attribution.

**Depends on**:

- v0.5 milestone complete (Phase 13)

**Repos**: `llm-agent-rag`

**Requirements covered**:

- `RAG-RETR2-01`
- `RAG-RETR2-02`

**Planned work**:

- `14-01` Okapi BM25 in-memory lexical retriever plus an optional
  `store.LexicalSearcher` capability interface (covers RAG-RETR2-01)
- `14-02` Postgres `tsvector`/`ts_rank_cd` lexical path implementing
  `store.LexicalSearcher`, with an opt-in lexical conformance suite
  (covers RAG-RETR2-01)
- `14-03` configurable RRF constant plus per-signal fusion attribution in the
  retrieval `Trace` (covers RAG-RETR2-02)

### Phase 15: Model-based reranking and rerank explainability

**Status**: complete 2026-05-15

**Goal**: add a model-based reranker behind the existing `rerank.Reranker`
seam and make rerank decisions auditable through score/rank-delta trace data.

**Depends on**:

- Phase 14

**Repos**: `llm-agent-rag`

**Requirements covered**:

- `RAG-RERANK-01`
- `RAG-RERANK-02`

**Planned work**:

- `15-01` rerank explainability: `rerank.RerankScore` + `Trace.Scores`
  (input/output score, rank, delta), surfaced through
  `rag.Diagnostics.RerankScores` (covers RAG-RERANK-02)
- `15-02` `rerank.ScoringModel` seam + `ModelReranker` + `HTTPScoringModel`
  (`net/http` rerank-API client; no new dependency) (covers RAG-RERANK-01)

### Phase 16: Generation-side evaluation and the RAG Triad

**Status**: complete 2026-05-15

**Goal**: complete the RAG Triad by adding generation-side evaluation —
groundedness/faithfulness and answer-relevance via LLM-as-judge — and assemble
retrieval + generation scores into one report.

**Depends on**:

- Phase 14
- Phase 15

**Repos**: `llm-agent-rag`

**Requirements covered**:

- `RAG-EVAL2-01`
- `RAG-EVAL2-02`

**Planned work**:

- `16-01` `eval.Judge` seam (`JudgeRequest`/`Judgement`) + `LLMJudge`
  LLM-as-judge over `generate.Model`, lenient JSON parsing (covers
  RAG-EVAL2-01)
- `16-02` `eval.TriadEvaluator` assembling retrieval + generation metrics
  into a `TriadResult`, `WriteJSONL` report + `Summary`, and a RAG-Triad CI
  gate (covers RAG-EVAL2-02)

### Phase 17: Cost and latency observability

**Status**: complete 2026-05-16

**Goal**: instrument every import/retrieve/ask flow with token counts,
per-stage durations, and call counts, and emit RED + cost metrics from the
`otelrag` sister-repo wrapper.

**Depends on**:

- Phases 14-16

**Repos**: `llm-agent-rag`, `llm-agent-otel`

**Requirements covered**:

- `RAG-OBS-01`
- `RAG-OBS-02`

**Planned work**:

- `17-01` `obs` package (`Metrics`/`StageTiming`/`CallCounts`/`TokenUsage` +
  context-scoped `Counter`); `countingEmbedder`/`countingModel` decorators;
  per-stage durations + call counts into `Diagnostics`, `retrieve.Trace`,
  `ImportResult`, `ImportTrace` (covers RAG-OBS-01 — measurement)
- `17-02` `generate.Usage` on `generate.Response`; ask flow records token
  cost into `obs.Metrics.Tokens` — reported usage or a `pack.TokenCounter`
  estimate flagged `Estimated` (covers RAG-OBS-01 — tokens)
- `17-03` `otelrag` RED + cost metrics: `MeterProvider` in `Config`, four
  instruments on `Wrapper`, emitted per `Import`/`Retrieve`/`Ask`; verified
  locally via `go.work` (covers RAG-OBS-02)

**Cross-repo note**: `17-03` references RAG-SDK fields in the untagged
`llm-agent-rag` working tree — verified locally via a temporary `go.work`;
the `otelrag/go.mod` `require` bump waits on an `llm-agent-rag` re-tag at
v0.6 close (the v0.5 pattern).

### Phase 18: Content safety — PII redaction and injection defense

**Status**: complete 2026-05-17

**Goal**: add a content-safety layer — PII redaction on ingestion and a
prompt-injection filter on retrieved content before prompt assembly.

**Depends on**:

- Phase 14

**Repos**: `llm-agent-rag`

**Requirements covered**:

- `RAG-SEC-01`
- `RAG-SEC-02`

**Planned work**:

- `18-01` `guard` package + PII redaction: `Redactor`/`PIIRedactor` with a
  configurable rule set (`NewPIIRedactor`: email/phone/credit-card/SSN/
  IPv4); wired into `Import` before chunking; per-kind `Redactions` on
  `ImportResult`/`ImportTrace` (covers RAG-SEC-01)
- `18-02` `guard` injection scanner: `InjectionScanner`/`PatternScanner` +
  `SanitizeMode` (Neutralize/Drop) + `Neutralize`; wired into `Ask` before
  prompt assembly; `InjectionFindings` on `Diagnostics` (covers RAG-SEC-02)

### Phase 19: Agentic retrieval — decomposition and self-correction

**Status**: complete 2026-05-17

**Goal**: add agentic retrieval patterns — multi-hop query decomposition and a
self-correcting retrieval loop driven by the grounding signal.

**Depends on**:

- Phase 16

**Repos**: `llm-agent-rag`

**Requirements covered**:

- `RAG-AGENT-01`
- `RAG-AGENT-02`

**Planned work**:

- `19-01` `retrieve.MultiHopRetriever` decorator + `QueryDecomposer`
  (`HeuristicDecomposer`/`LLMDecomposer`): decompose a compound query into
  sub-queries, retrieve per sub-query through the wrapped `Retriever`, merge
  (dedup + `TopK`); `Trace.Hops` per-hop attribution (covers RAG-AGENT-01)
- `19-02` new `agentic` package: `CorrectiveAsker` + `QueryReformulator`
  (`LLMReformulator`) — judge groundedness via `eval.Judge`, reformulate and
  retry under a bounded `MaxRetries` cap, return the best attempt (covers
  RAG-AGENT-02)

## Known Carry-forward Debt

- Formal verification artifacts are still uneven after Phase 0.
- The refsvc demo remains intentionally demo-grade in observability fidelity
  and packaging.
- Deployment-layer surface for `llm-agent-rag` (HTTP service, CLI, caching) is
  intentionally deferred past v0.6 — v0.6 is a retrieval-quality milestone.
- Live-Postgres CI wiring (testcontainers-go or GH Actions services) is still
  pending from v0.5.

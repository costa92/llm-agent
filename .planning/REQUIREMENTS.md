# Requirements: v0.6 Production-grade retrieval quality and safety

**Defined:** 2026-05-15
**Core Value:** the core `llm-agent` module stays stdlib-only and zero-dep
while `llm-agent-rag` deepens its retrieval, reranking, evaluation,
observability, safety, and agentic seams from minimal interfaces into
production-grade implementations.

## Milestone Scope

v0.6 does **not** add new deployment-layer surface (HTTP service, CLI,
caching are deferred). It hardens the six seams that v0.5 left as thin or
partial: real lexical/hybrid retrieval, model-based reranking, generation-side
evaluation, cost/latency observability, content safety, and agentic retrieval.

Reference: the v0.6 gap analysis against
[Awesome-RAG-Production](https://github.com/Enes830/Awesome-RAG-Production)
classified these six areas as 🟡 Partial — seams exist, implementations are
weak. Each area becomes one phase.

## v0.6 Requirements

### Retrieval quality

- [x] **RAG-RETR2-01**: lexical retrieval uses a real BM25 ranking model in the
      in-memory path and a `tsvector`/`ts_rank` lexical path in `postgres.Store`,
      replacing the current token-overlap scoring.
- [x] **RAG-RETR2-02**: hybrid retrieval fuses dense, lexical, and structure
      signals through a principled method (reciprocal rank fusion / normalized
      score fusion) with per-signal score attribution exposed in the trace.

### Reranking

- [x] **RAG-RERANK-01**: a model-based reranker implements the existing
      `rerank.Reranker` interface by calling an external cross-encoder/rerank
      model, isolated behind a subpackage/build tag like `postgres`.
- [x] **RAG-RERANK-02**: rerank decisions are auditable — pre/post-rerank
      scores and rank deltas are surfaced in `Diagnostics` / retrieval trace.

### Evaluation

- [x] **RAG-EVAL2-01**: a `Judge` interface supports LLM-as-judge scoring of
      groundedness/faithfulness and answer-relevance, completing the RAG Triad
      alongside the existing retrieval metrics.
- [x] **RAG-EVAL2-02**: retrieval and generation scores are assembled into a
      single evaluation report (JSONL + summary) wired into the `eval` package
      and the existing CI regression gate.

### Observability

- [x] **RAG-OBS-01**: token counts, per-stage durations, and embedding/
      generation call counts are recorded in `Trace` / `Diagnostics` for every
      import, retrieve, and ask flow.
- [x] **RAG-OBS-02**: the `otelrag` sister-repo wrapper emits rate/error/
      duration plus cost metrics derived from those fields.

### Content safety

- [x] **RAG-SEC-01**: a `guard` package redacts PII from ingested content
      before chunking/embedding, with configurable entity rules.
- [x] **RAG-SEC-02**: retrieved chunks pass an injection-pattern filter before
      prompt assembly; untrusted content is neutralized or dropped fail-safe.

### Agentic retrieval

- [x] **RAG-AGENT-01**: a multi-hop `Retriever` decorator decomposes compound
      queries into sub-queries and merges their sub-retrievals.
- [x] **RAG-AGENT-02**: a self-correcting retrieval loop detects low grounding
      (using the Phase 16 grounding signal) and re-retrieves with reformulated
      queries up to a bounded retry cap.

## Out of Scope

| Feature | Reason |
|---------|--------|
| HTTP service layer / CLI for `llm-agent-rag` | Deployment-layer work; deferred to a later milestone — v0.6 is a quality milestone, not a packaging one |
| Embedding / retrieval caching | Useful, but downstream of getting retrieval quality and instrumentation right first |
| GraphRAG / relationship traversal | Large architectural addition; defer past v0.6 |
| PDF/OCR ingestion stack | Ingestion robustness is not in the six chosen areas |
| Embedding or vector-store deps in core `llm-agent` | Violates the zero-dependency core value |
| Kubernetes packaging | Still out of scope until a future milestone plans it explicitly |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| RAG-RETR2-01 | Phase 14 | Delivered |
| RAG-RETR2-02 | Phase 14 | Delivered |
| RAG-RERANK-01 | Phase 15 | Delivered |
| RAG-RERANK-02 | Phase 15 | Delivered |
| RAG-EVAL2-01 | Phase 16 | Delivered |
| RAG-EVAL2-02 | Phase 16 | Delivered |
| RAG-OBS-01 | Phase 17 | Delivered |
| RAG-OBS-02 | Phase 17 | Delivered |
| RAG-SEC-01 | Phase 18 | Delivered |
| RAG-SEC-02 | Phase 18 | Delivered |
| RAG-AGENT-01 | Phase 19 | Delivered |
| RAG-AGENT-02 | Phase 19 | Delivered |

**Coverage:**
- v0.6 requirements: 12 total
- Mapped to phases: 12
- Unmapped: 0

---
*Requirements defined: 2026-05-15 after opening the v0.6 retrieval-quality milestone*

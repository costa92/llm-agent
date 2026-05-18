# llm-agent

## What This Is

`llm-agent` is a stdlib-only Go framework for building LLM-driven agents.
The project now spans four coordinated repos plus a standalone RAG SDK:

- `llm-agent` keeps the zero-dependency core, agent paradigms, memory, RAG,
  and the new `llm/v2` capability surface.
- `llm-agent-providers` ships real OpenAI, Anthropic, and Ollama adapters.
- `llm-agent-otel` ships capability-preserving OpenTelemetry wrappers.
- `llm-agent-customer-support` ships a demo customer-support service that ties
  the stack together.
- `llm-agent-rag` is the standalone RAG SDK that owns import, retrieval, and
  answer-generation primitives while the core repo preserves a compatibility
  facade.

`v0.3` shipped, `v0.4` closed the deprecation-removal cycle, and `v0.5`
turned the extracted RAG work into a production-oriented standalone SDK.
The next milestone (`v0.6`) deepens RAG retrieval quality, evaluation,
observability, and safety without violating the zero-dependency contract of
the core module.

## Core Value

**The core `llm-agent` module stays stdlib-only and zero-dep.** Providers,
telemetry, and reference services remain opt-in sister repos so the primary
module stays readable, portable, and cheap to adopt.

## Current State

- `v0.3` shipped on 2026-05-12 and is archived in
  `.planning/milestones/v0.3-ROADMAP.md`.
- The shipped stack includes real Generate, Stream, Tool, and Embedding paths
  across the targeted provider set.
- OpenTelemetry wrappers and the reference customer-support service are part of
  the released milestone state.
- `v0.4.0` completed the deprecation-removal cycle and is now the stable base
  line across the sister repos.
- As of 2026-05-14, the RAG code has been extracted into the standalone repo
  `llm-agent-rag`, released independently, and re-consumed from the core repo
  through module dependency instead of a vendored copy.
- `v0.5` shipped on 2026-05-15: structure-aware retrieval, a PostgreSQL +
  pgvector backend with a shared conformance suite, tracing hooks, an
  evaluation framework, a feedback loop, and cross-repo contract gates.
  `llm-agent-rag` is tagged `v0.2.0`.
- The active next step (`v0.6`) is a retrieval-quality milestone: deepen the
  six seams v0.5 left thin — lexical/hybrid retrieval, reranking, evaluation,
  observability, content safety, and agentic retrieval.

## Requirements

### Validated

- ✓ The core repo still builds as a stdlib-only module.
- ✓ `llm/v2` capability negotiation is live in the core repo.
- ✓ Three real provider adapters exist in sister repos.
- ✓ Capability-preserving OTel wrappers exist in a sister repo.
- ✓ A runnable customer-support demo service exists in a sister repo.

### Active (v0.6)

- `RAG-RETR2-01..02`: real BM25 lexical retrieval and principled hybrid fusion
  with per-signal score attribution.
- `RAG-RERANK-01..02`: a model-based reranker behind the existing seam, plus
  rerank-decision explainability in the trace.
- `RAG-EVAL2-01..02`: generation-side LLM-as-judge evaluation completing the
  RAG Triad, assembled into a combined report.
- `RAG-OBS-01..02`: cost/latency instrumentation in `Trace`/`Diagnostics` and
  RED metrics emitted from `otelrag`.
- `RAG-SEC-01..02`: PII redaction on ingestion and prompt-injection defense on
  retrieved content.
- `RAG-AGENT-01..02`: multi-hop query decomposition and a self-correcting
  retrieval loop.

### Out of Scope

- HTTP service layer, CLI, and caching for `llm-agent-rag` are deferred past
  v0.6 — v0.6 is a retrieval-quality milestone, not a packaging one.
- GraphRAG / relationship traversal is deferred past v0.6.
- PDF/OCR ingestion is out of scope for v0.6.
- Kubernetes packaging is still out of scope until a future milestone plans it
  explicitly.
- Multimodal/vision support is still out of scope.
- A v1.0 stability promise is still out of scope pending real-world feedback.
- Moving provider or vector-store dependencies into the core `llm-agent` repo
  remains out of scope because it would violate the zero-dependency core value.

## Next Milestone Goals (v0.6)

- Replace token-overlap lexical scoring with real BM25 and fuse retrieval
  signals through a principled method.
- Add a model-based reranker behind the existing seam, with auditable rerank
  decisions.
- Complete the RAG Triad with generation-side LLM-as-judge evaluation.
- Instrument cost/latency end-to-end and emit RED metrics from `otelrag`.
- Add a content-safety layer: PII redaction and prompt-injection defense.
- Add agentic retrieval: query decomposition and self-correcting retrieval.

## Known Tech Debt

- Formal `*-VERIFICATION.md` coverage is uneven after Phase 0.
- The refsvc observability demo is intentionally demo-grade rather than
  production-billing-grade.
- `llm-agent-rag` lexical retrieval is still token-overlap scoring, not BM25;
  reranking is heuristic-only; evaluation covers the retrieval side but not
  generation; there is no cost/latency instrumentation or content-safety
  layer. v0.6 closes these gaps.
- Live-Postgres CI wiring (testcontainers-go or GH Actions services) is still
  pending from v0.5.

## Operational Follow-ups

- Run the next milestone through the standalone `llm-agent-rag` repo first
  wherever possible, then keep `llm-agent/rag` aligned as a compatibility
  facade.
- Keep the core repo stdlib-only while expanding RAG through sister-repo-style
  opt-in dependencies.

## Key Decisions

- 2026-05-12: Phase 7 gate opened early by explicit operator instruction even
  though the original roadmap treated it as calendar-gated post-`v0.3` work.
  This locks the next active work to `DEPRC-01..04` only; no unrelated feature
  milestone is being opened in parallel.
- 2026-05-12: Phase 7 execution was split into three core-repo slices:
  `07-01` audit, `07-02` runtime migration, and `07-03` compatibility removal
  + documentation rewrite. Cross-repo coordination is deferred to `07-04`.
- 2026-05-13: a local 4-repo `go.work` audit proved that `llm-agent-providers`,
  `llm-agent-otel`, and `llm-agent-customer-support` already pass against the
  post-compat-removal core API with no source patches required.
- 2026-05-13: Phase 7 closeout verification confirmed that the released core
  `v0.4.0` tag resolves remotely, all sister repos pass `go test ./...`
  against the coordinated release line, and coordinated sister-repo tags can be
  cut from the already-landed `v0.4.0` bump commits.
- 2026-05-14: the RAG subsystem now has a standalone repository
  `llm-agent-rag`; future feature growth should land there first, while the
  core repo preserves the historical API through adapters and compatibility
  wrappers.
- 2026-05-14: the next active milestone is RAG productionization rather than
  another core-API transition; the main architectural constraint is preserving
  the zero-dependency core while expanding retrieval capability externally.
- 2026-05-15: `v0.5` shipped (`llm-agent-rag v0.2.0`). The `v0.6` milestone
  was scoped after a gap analysis against the Awesome-RAG-Production taxonomy.
  The operator explicitly chose to deepen the six 🟡 Partial seams — retrieval,
  reranking, evaluation, observability, security, agentic — over building the
  ❌ Missing deployment layer (HTTP service, CLI, caching). v0.6 is therefore a
  retrieval-quality milestone; deployment-layer surface is deferred.
- 2026-05-15: new non-stdlib deps needed by v0.6 (e.g. a rerank-model HTTP
  client) are permitted in `llm-agent-rag` but must follow the `postgres`
  subpackage pattern — isolated behind a subpackage/build tag so the core SDK
  stays publishable. The stdlib-only rule remains absolute for core `llm-agent`.

## Archived Milestone Definition

<details>
<summary>v0.3 milestone snapshot</summary>

`v0.3` was the "library you can deploy" milestone:

- add real OpenAI, Anthropic, and Ollama integrations
- extend the core contract to capability-based `llm/v2`
- add OpenTelemetry observability
- ship a `docker compose` customer-support reference stack

Archive references:

- Roadmap: `.planning/milestones/v0.3-ROADMAP.md`
- Requirements: `.planning/milestones/v0.3-REQUIREMENTS.md`
- Audit: `.planning/v0.3-MILESTONE-AUDIT.md`

</details>

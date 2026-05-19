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

`v0.3` shipped, `v0.4` closed the deprecation-removal cycle, `v0.5` turned
the extracted RAG work into a production-oriented standalone SDK, `v0.6`
deepened RAG retrieval quality, reranking, evaluation, observability, and
safety, `v0.7` added Tier-1 GraphRAG — knowledge-graph construction and
relationship-traversal retrieval — to `llm-agent-rag`, and `v0.8` extended
that to Tier-3: hierarchical community detection, lazy community summaries,
map-reduce global search, and fuzzy entity resolution, and `v0.9` refined
it with DRIFT hybrid search and path-ranked subgraph evidence. The project
is currently **between milestones**; the next milestone is not yet scoped.

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
- `v0.6` shipped on 2026-05-18: the six retrieval-quality seams v0.5 left
  thin are now production-grade — BM25 lexical retrieval + principled RRF
  fusion, model-based reranking with explainability, the generation-side
  RAG Triad, cost/latency observability, content safety (PII redaction +
  injection defense), and agentic retrieval. `llm-agent-rag` is tagged
  `v0.3.0`; 12/12 requirements delivered (audit
  `.planning/v0.6-MILESTONE-AUDIT.md`).
- `v0.7` shipped on 2026-05-19: Tier-1 GraphRAG for `llm-agent-rag` —
  knowledge-graph construction (`graph` package, dual-mode extraction), a
  `store.GraphStore` optional capability (in-memory + `postgres`
  recursive-CTE) with re-ingest reconciliation, and a `GraphRetriever`
  fused as a fourth RRF signal. `llm-agent-rag` is tagged `v0.4.0`; 6/6
  requirements delivered (audit `.planning/v0.7-MILESTONE-AUDIT.md`); no
  new dependency, no graph database.
- `v0.8` shipped on 2026-05-20: GraphRAG Tier-3 for `llm-agent-rag` —
  hierarchical community detection (deterministic stdlib Louvain), a
  `store.CommunityStore` capability, lazy community summaries, the
  `rag.System.AskGlobal` map-reduce global-search answer path, and an
  opt-in `EmbeddingEntityResolver` fuzzy entity-resolution pre-pass.
  `llm-agent-rag` is tagged `v0.5.0`; 6/6 requirements delivered (audit
  `.planning/v0.8-MILESTONE-AUDIT.md`); no new dependency, no graph
  database.
- `v0.9` shipped on 2026-05-20: GraphRAG refinements for `llm-agent-rag` —
  path-ranked subgraph evidence (a deterministic stdlib `graph.PathRanker`
  + an opt-in `GraphRetriever` mode) and DRIFT hybrid search
  (`rag.System.AskDrift` + `eval.DriftEvaluator`). `llm-agent-rag` is
  tagged `v0.6.0`; 4/4 requirements delivered (audit
  `.planning/v0.9-MILESTONE-AUDIT.md`); no new dependency, no graph
  database. Incremental community maintenance is deferred to v1.0+.
- The project is now between milestones; the next milestone is not yet
  scoped.

## Requirements

### Validated

- ✓ The core repo still builds as a stdlib-only module.
- ✓ `llm/v2` capability negotiation is live in the core repo.
- ✓ Three real provider adapters exist in sister repos.
- ✓ Capability-preserving OTel wrappers exist in a sister repo.
- ✓ A runnable customer-support demo service exists in a sister repo.
- ✓ `llm-agent-rag` (`v0.3.0`) has production-grade retrieval: real BM25
  lexical retrieval + principled RRF fusion with per-signal attribution,
  a model-based reranker behind the existing seam with rerank
  explainability, the generation-side RAG Triad (LLM-as-judge), cost/
  latency observability with `otelrag` RED metrics, content safety (PII
  redaction + prompt-injection defense), and agentic retrieval (multi-hop
  decomposition + self-correcting loop).
- ✓ `llm-agent-rag` (`v0.4.0`) has Tier-1 GraphRAG: a `graph` package with
  dual-mode (LLM + deterministic) entity/relation extraction and
  exact-match canonicalization, a `store.GraphStore` optional capability
  (stdlib in-memory + `postgres` recursive-CTE) with hard-bounded traversal
  and re-ingest reconciliation, and a `retrieve.GraphRetriever` fused as a
  fourth RRF signal — no graph database, no new dependency.
- ✓ `llm-agent-rag` (`v0.5.0`) has Tier-3 GraphRAG: hierarchical community
  detection (a deterministic pure-stdlib `graph.CommunityDetector` Louvain
  seam), a `store.CommunityStore` capability, lazy LLM community summaries
  (content-hash-cached), the `rag.System.AskGlobal` map-reduce
  global-search answer path, and an opt-in `EmbeddingEntityResolver`
  fuzzy-merge pre-pass — no graph database, no new dependency.

- ✓ `llm-agent-rag` (`v0.6.0`) has the GraphRAG refinements: deterministic
  pure-stdlib path ranking (`graph.PathRanker` + an opt-in
  `retrieve.GraphRetriever` path mode with structured
  subgraph-as-evidence), and DRIFT hybrid search (`rag.System.AskDrift` — a
  global primer + bounded local loop + synthesis — with
  `eval.DriftEvaluator`) — no graph database, no new dependency.

### Active

None — the project is between milestones. v0.9 GraphRAG refinements is
shipped and archived; the next milestone is not yet scoped.

### Out of Scope

- Incremental community maintenance is deferred to v1.0+ — v0.8's full
  re-detection on re-ingest is correct and fast at SDK scale.
- A dedicated graph database (Neo4j etc.) — `GraphStore`/`CommunityStore`
  stay interfaces so a graph-DB impl can be added later; the SDK uses
  recursive-CTE traversal and in-Go community detection.
- HTTP service layer, CLI, and caching for `llm-agent-rag` remain deferred.
- PDF/OCR ingestion is out of scope.
- Kubernetes packaging is still out of scope until a future milestone plans it
  explicitly.
- Multimodal/vision support is still out of scope.
- A v1.0 stability promise is still out of scope pending real-world feedback.
- Moving provider or vector-store dependencies into the core `llm-agent` repo
  remains out of scope because it would violate the zero-dependency core value.

## Active Milestone Goals

None — the project is between milestones. v0.9 GraphRAG refinements shipped
2026-05-20 (`llm-agent-rag v0.6.0`). After v0.9 the SDK spans the full
practical GraphRAG spectrum: lightweight local (v0.7), path-ranked local
(v0.9), community global (v0.8), and DRIFT hybrid (v0.9).

Candidate next directions (not yet scoped):

- **incremental community maintenance** — update only the communities a
  re-ingest perturbs (deferred from v0.9 by keystone KG4-5; revisit if
  profiling shows `Detect` dominating re-ingest).
- the `llm-agent-rag` **deployment layer** — HTTP service, CLI, caching —
  deferred since v0.6.
- **live-Postgres CI wiring** — carried-forward infra debt.
- a **v1.0** stability pass to lock the public API.

Still deferred: PDF/OCR ingestion, claim/covariate extraction, a dedicated
graph database.

## Known Tech Debt

- Formal `*-VERIFICATION.md` coverage is uneven after Phase 0.
- The refsvc observability demo is intentionally demo-grade rather than
  production-billing-grade.
- Live-Postgres CI wiring (testcontainers-go or GH Actions services) is still
  pending from v0.5; the Phase 14 Postgres `tsvector` lexical path, the
  Phase 21 `postgres` graph path, and the Phase 23-24 `postgres`
  `_communities`/`_community_reports` paths remain unverified against a live
  database.
- `EmbeddingEntityResolver` (v0.8) has documented false-positive risk; it
  ships conservative (high threshold, same-type-only) and opt-in.
- Regex-based content safety (`guard`) is best-effort — it catches known PII
  and injection patterns, not novel/obfuscated ones.

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
- 2026-05-18: `v0.6` shipped — `llm-agent-rag` tagged `v0.3.0`, milestone
  audit PASS (12/12 requirements). In the event, v0.6 needed **no** new
  dependency at all: every new capability (BM25, RRF, rerank HTTP client,
  LLM-as-judge, `obs` metrics, `guard` safety, agentic retrieval) was built
  on the stdlib plus existing seams — the `postgres` subpackage remains the
  SDK's only non-stdlib island.
- 2026-05-18: `v0.7` opened — GraphRAG for `llm-agent-rag`, scoped from
  `.planning/research/v0.7-graphrag-SUMMARY.md`. Keystone calls (KG-1..KG-7):
  v0.7 targets **Tier-1 lightweight GraphRAG** (LightRAG-end: entity/relation
  extraction + neighborhood-traversal retrieval) — community detection and
  global search are v0.8. The graph is a `store.GraphStore` **optional
  capability** (mirroring `store.LexicalSearcher`): a stdlib in-memory impl
  plus a `postgres` recursive-CTE impl — **no graph database**, so the
  milestone again adds no new module dependency. Extraction is dual-mode
  (LLM + deterministic); graph retrieval fuses as a fourth RRF signal and
  never replaces dense/lexical; traversal is hard-bounded (depth ≤ 2).
- 2026-05-19: `v0.7` shipped — `llm-agent-rag` tagged `v0.4.0`, milestone
  audit PASS (6/6 requirements). As with v0.6, v0.7 needed **no** new
  dependency: the `graph` package, `store.GraphStore` (in-memory +
  `postgres` recursive-CTE), and `retrieve.GraphRetriever` were all built
  on the stdlib plus existing seams — no graph database. The KG-1..KG-7
  keystone calls held in the delivered code; the `postgres` graph path is
  env-gated and joins the carried-forward live-DB verification debt.
- 2026-05-19: `v0.8` opened — GraphRAG Tier-3 for `llm-agent-rag`, scoped
  from `.planning/research/v0.8-graphrag-tier3-SUMMARY.md`. Keystone calls
  (KG3-1..KG3-8): community report generation is **lazy by default**
  (LazyGraphRAG — detect communities at ingest, summarize at query time and
  cache; Microsoft's own ~0.1%-indexing-cost data drives this); community
  detection is **pure stdlib** (a deterministic Louvain `CommunityDetector`
  seam, store-agnostic — no graph database, no new dependency); global
  search is a **separate `rag.System.AskGlobal` answer path**, not a
  `Retriever`, and never passes through rerank/pack; fuzzy entity resolution
  is an **opt-in pre-pass** before `Canonicalize` (`NoopEntityResolver`
  default). DRIFT search is deferred to v0.9.
- 2026-05-20: `v0.9` shipped — `llm-agent-rag` tagged `v0.6.0`, milestone
  audit PASS (4/4 requirements). As with v0.6/v0.7/v0.8, v0.9 needed **no**
  new dependency: path ranking is a stdlib graph computation over the
  existing `Subgraph`; DRIFT (`rag.System.AskDrift`) is orchestration over
  `AskGlobal`'s helpers + direct graph traversal + `generate.Model`. The
  KG4-1..KG4-7 keystone calls held; incremental community maintenance was
  deferred again to v1.0+ (KG4-5) — v0.8's full re-detection is correct and
  fast at SDK scale. After v0.9 the SDK spans the full practical GraphRAG
  spectrum (lightweight/path-ranked local, community global, DRIFT hybrid).
- 2026-05-20: `v0.8` shipped — `llm-agent-rag` tagged `v0.5.0`, milestone
  audit PASS (6/6 requirements). As with v0.6 and v0.7, v0.8 needed **no**
  new dependency: community detection is pure stdlib (a deterministic
  Louvain `CommunityDetector`), summarization reuses `generate.Model`,
  fuzzy resolution reuses `embed.Embedder` — no graph database. The
  KG3-1..KG3-8 keystone calls held in the delivered code; the `postgres`
  `_communities`/`_community_reports` paths are env-gated and join the
  carried-forward live-DB verification debt.
- 2026-05-20: `v0.9` opened — GraphRAG refinements for `llm-agent-rag`,
  scoped from `.planning/research/v0.9-graphrag-refinements-SUMMARY.md`.
  Keystone calls (KG4-1..KG4-7): v0.9 ships **two** of v0.8's three
  deferrals — DRIFT hybrid search and path-ranking / subgraph-as-evidence —
  and **defers incremental community maintenance again** (v0.8's full
  re-detection is correct and fast at SDK scale; incremental Louvain is a
  large subtle second algorithm solving a non-problem). DRIFT is a third
  answer path (`rag.System.AskDrift`) orchestrating `AskGlobal` + the local
  `GraphRetriever`, with a hard round cap; path ranking is a deterministic
  pure-stdlib opt-in mode on `GraphRetriever`. No new dependency.

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

<details>
<summary>v0.6 milestone snapshot</summary>

`v0.6` was the "production-grade retrieval quality and safety" milestone —
six phases (14-19), one per retrieval-quality seam v0.5 left thin:

- Phase 14 — BM25 lexical retrieval + principled RRF fusion
- Phase 15 — model-based reranking + rerank explainability
- Phase 16 — generation-side evaluation (the RAG Triad)
- Phase 17 — cost/latency observability + `otelrag` RED metrics
- Phase 18 — content safety: PII redaction + injection defense
- Phase 19 — agentic retrieval: decomposition + self-correction

Shipped 2026-05-18; `llm-agent-rag` tagged `v0.3.0`; no new dependency.

Archive references:

- Roadmap: `.planning/milestones/v0.6-ROADMAP.md`
- Requirements: `.planning/milestones/v0.6-REQUIREMENTS.md`
- Audit: `.planning/v0.6-MILESTONE-AUDIT.md`

</details>

<details>
<summary>v0.7 milestone snapshot</summary>

`v0.7` was the "GraphRAG — relationship-traversal retrieval" milestone —
three phases (20-22) adding Tier-1 lightweight GraphRAG to `llm-agent-rag`:

- Phase 20 — knowledge-graph construction: the `graph` package, dual-mode
  (LLM + deterministic) entity/relation extraction, exact-match
  canonicalization
- Phase 21 — graph storage: the `store.GraphStore` optional capability
  (stdlib in-memory + `postgres` recursive-CTE), hard-bounded traversal,
  re-ingest reconciliation
- Phase 22 — graph-traversal retrieval: `GraphRetriever` fused as a fourth
  RRF signal, graph-on/off eval A/B

Shipped 2026-05-19; `llm-agent-rag` tagged `v0.4.0`; no new dependency, no
graph database.

Archive references:

- Roadmap: `.planning/milestones/v0.7-ROADMAP.md`
- Requirements: `.planning/milestones/v0.7-REQUIREMENTS.md`
- Audit: `.planning/v0.7-MILESTONE-AUDIT.md`

</details>

<details>
<summary>v0.8 + v0.9 milestone snapshots</summary>

`v0.8` was the "GraphRAG Tier-3" milestone — three phases (23-25):
community detection (stdlib Louvain `CommunityDetector` + `store.CommunityStore`),
lazy community summaries, the `rag.System.AskGlobal` map-reduce
global-search path, and the opt-in `EmbeddingEntityResolver` fuzzy-merge
pre-pass. Shipped 2026-05-20; `llm-agent-rag` tagged `v0.5.0`.

`v0.9` was the "GraphRAG refinements" milestone — two phases (26-27):
path-ranked subgraph evidence (`graph.PathRanker` + an opt-in
`GraphRetriever` mode) and DRIFT hybrid search (`rag.System.AskDrift` +
`eval.DriftEvaluator`). Shipped 2026-05-20; `llm-agent-rag` tagged
`v0.6.0`. Both milestones added no new dependency and no graph database.

Archive references:

- Roadmaps: `.planning/milestones/v0.8-ROADMAP.md`,
  `.planning/milestones/v0.9-ROADMAP.md`
- Requirements: `.planning/milestones/v0.8-REQUIREMENTS.md`,
  `.planning/milestones/v0.9-REQUIREMENTS.md`
- Audits: `.planning/v0.8-MILESTONE-AUDIT.md`,
  `.planning/v0.9-MILESTONE-AUDIT.md`

</details>

<details>
<summary>v0.8 milestone snapshot</summary>

`v0.8` was the "GraphRAG Tier-3 — communities, global search, fuzzy
resolution" milestone — three phases (23-25) extending the v0.7 Tier-1
graph:

- Phase 23 — community detection: a deterministic stdlib Louvain
  `CommunityDetector` seam, the `store.CommunityStore` capability, detection
  wired into `Import`
- Phase 24 — community summaries and global search: lazy
  content-hash-cached community reports, the `rag.System.AskGlobal`
  map-reduce global-search answer path
- Phase 25 — fuzzy entity resolution and evaluation: the opt-in
  `EmbeddingEntityResolver` pre-pass, the `eval.GlobalEvaluator` harness

Shipped 2026-05-20; `llm-agent-rag` tagged `v0.5.0`; no new dependency, no
graph database.

Archive references:

- Roadmap: `.planning/milestones/v0.8-ROADMAP.md`
- Requirements: `.planning/milestones/v0.8-REQUIREMENTS.md`
- Audit: `.planning/v0.8-MILESTONE-AUDIT.md`

</details>

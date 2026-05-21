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
it with DRIFT hybrid search and path-ranked subgraph evidence. `v1.0`
stabilized the `llm-agent-rag` public API and committed it to a
Go-module compatibility promise — a quality milestone, no new features.
`v1.1` is **ecosystem alignment — shipped and closed 2026-05-20** (audit
PASS 5/5): the core `llm-agent` and the three sister repos brought
current with `llm-agent-rag v1.0.0` and each cut a coherent, tagged
baseline (`llm-agent v0.5.1`, `llm-agent-rag v1.0.1`, sisters
`v0.2.1`/`v0.2.1`/`v0.2.2`); an umbrella dependency-currency CI gate
shipped to prevent future drift. Alignment / housekeeping milestone — no
new features, no new dependency.

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
- `v1.0` shipped on 2026-05-21: API stabilization for `llm-agent-rag` —
  a written exported-surface audit and the pre-freeze breaking renames
  (`eval.Evaluator`→`RetrievalEvaluator`, `eval.Result`→`RetrievalResult`;
  the `ragkit` root repurposed as a documented doc-anchor); full package +
  exported-symbol doc-comment coverage and a written `docs/compatibility.md`
  Go-module compatibility promise; a pure-stdlib `internal/apisnapshot`
  exported-surface gate (`api/v1.snapshot.txt` + a `go test`
  regeneration-diff) plus a `-tags llmagent` CI step. `llm-agent-rag` is
  tagged `v1.0.0`; 6/6 requirements delivered (audit
  `.planning/v1.0-MILESTONE-AUDIT.md`); no new dependency, no behavior
  change. Scope was `llm-agent-rag` only — the core module and sister
  repos stay on their own version tracks.
- `v1.1` opened on 2026-05-21: an ecosystem-alignment milestone — the core
  `llm-agent` `go.mod` pinned `llm-agent-rag v0.1.4` (8 minors + a major
  stale) and three sister repos lagged too. v1.1 repaired the core RAG
  facade against rag `v1.0.0` (a 7-test `vector dimension mismatch`
  regression), landed two stranded sister-repo branches, walked a
  coordinated dependency-bump + re-tag wave, and added an umbrella
  dependency-currency CI gate. Scoped from
  `.planning/research/v1.1-ecosystem-alignment-SUMMARY.md`; 5 requirements
  (`ECO-01..05`) across 4 phases (31-34). `llm-agent-rag` is the untouched
  fixed point; no new features, no new dependency.
- `v1.1` shipped and closed on 2026-05-20: audit PASS 5/5 (`ECO-01..05`,
  `KE-1..KE-7`). Final coordinated tag set: `llm-agent v0.5.1`,
  `llm-agent-rag v1.0.1`, `llm-agent-otel v0.2.1`,
  `llm-agent-providers v0.2.1`, `llm-agent-customer-support v0.2.2`.
  Phase 34 expanded from 3 slices to 9 slices mid-flight to honor the
  strict dep-currency gate (cascade of patch tags through the back-edge)
  and to correct a topological-order miss in Phase 33's cascade (cs sink
  re-tagged once providers was in place). Three architectural trade-offs
  documented honestly in `.planning/v1.1-MILESTONE-AUDIT.md`: the
  `v1.0.0 → v1.0.1` freeze-day-after re-tag (KE-2 holds — chore-only patch,
  no exported-symbol move), the topological-order miss (future cascades
  must `tsort` against the dep DAG), and the rag↔core cycle exemption
  (the one auditable strict-equality exemption in the gate, narrowly
  scoped). Audit: `.planning/v1.1-MILESTONE-AUDIT.md`.
- `v1.2` opened on 2026-05-20: **Core Capability Deepening** milestone
  — the umbrella's first **core-feature** milestone since v0.3. Theme:
  **Core v0.6** — capability additions to core `llm-agent`; memory
  tiering deferred to v1.3 per KC-2. Core module bump: `v0.5.1 →
  v0.6.0` (minor — additive only). Three capability additions:
  `budget` (ctx-keyed budget + `Tracker` interface enforcement at the
  `generateFromPrompt` chokepoint — CC-1, Phase 35), `policy`
  (capability-preserving `llm.ChatModel` decorator mirroring
  `otelmodel.Wrap` (K3); 3 built-in gates — PII redaction, injection
  detection, max-input-length — CC-2, Phase 36), and
  `orchestrate.Supervisor` (iterative supervisor↔worker primitive as a
  `StateGraph[S]` facade; honors budget + policy — CC-3, Phase 37).
  Phase 38 closes the milestone (tag `llm-agent v0.6.0`, archive, audit
  — CC-4). Memory tiering / scoping (session/project/user) is **OUT of
  v1.2 scope** per KC-2 — deferred to v1.3 with the `ScopedMemory`
  decorator shape pre-decided in research. Scoped from
  `.planning/research/v1.2-core-capability-deepening-SUMMARY.md`. Scope
  is the core repo only; sister repos stay on v0.2.x. Core stdlib-only
  **preserved**: every new gate uses stdlib `regexp`; every new test
  uses `ScriptedLLM`; no edit to validated public types
  (`llm.ChatModel`, `agents.Agent`, `memory.Memory`,
  `orchestrate.NodeFunc[S]`); no `/v2` import path (KC-5). Cost-table
  is **opt-in / outside core** (KC-4).

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

### Shipped (v1.1) — 2026-05-20

- ✓ `ECO-01`: core `llm-agent` RAG facade repaired against
  `llm-agent-rag v1.0.0` (and subsequently `v1.0.1` via the back-edge
  cascade); the 7 facade-test failures fixed inside the facade adapters;
  the core proven stdlib-only.
- ✓ `ECO-02`: every sister repo's `main` reflects reality — stranded
  branches merged, stale branches pruned.
- ✓ `ECO-03`: coordinated dependency-ordered re-tag wave shipped — final
  post-cascade tags: `llm-agent v0.5.1`, `llm-agent-rag v1.0.1`,
  `llm-agent-otel v0.2.1`, `llm-agent-providers v0.2.1`,
  `llm-agent-customer-support v0.2.2`. Zero `replace` directives.
- ✓ `ECO-04`: umbrella dependency-currency CI gate shipped
  (`scripts/dep-currency-check.sh` + `.github/workflows/umbrella.yml`),
  with one auditable rag↔core cycle exemption.
- ✓ `ECO-05`: full 5-repo coherence verification PASS
  (`34-08-RESULTS.md`); milestone audited
  (`.planning/v1.1-MILESTONE-AUDIT.md`).

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
- Moving provider or vector-store dependencies into the core `llm-agent` repo
  remains out of scope because it would violate the zero-dependency core value.

## Active Milestone Goals

**v1.2 — Core Capability Deepening (active, opened 2026-05-20).** The
first **core-feature** milestone since v0.3. Theme: **Core v0.6** —
capability additions to core `llm-agent`; memory tiering deferred to
v1.3. Core module bump: `v0.5.1 → v0.6.0` (minor — additive only).

Phases 35-38 ship three new agent-runtime governance primitives:

- **Phase 35 — Budget / cancellation context (`CC-1`)**: a `budget`
  package; ctx-keyed propagation via `budget.WithBudget(ctx, *Tracker)`
  + `budget.From(ctx)`; built-in `NewStrict`/`NewSoft` trackers;
  integration at the `generateFromPrompt` chokepoint so every existing
  agent paradigm honors it. Cost-table is **opt-in / outside core**.

- **Phase 36 — Policy / safety middleware (`CC-2`)**: a `policy`
  package; capability-preserving `policy.Wrap(model) ChatModel`
  decorator mirroring `otelmodel.Wrap` (K3); typed `Gate` event union;
  3 built-in gates (PII redaction, injection detection,
  max-input-length); documented composition stack
  `policy.Wrap(otelmodel.Wrap(provider))`.

- **Phase 37 — Multi-agent coordination (`CC-3`)**:
  `orchestrate.Supervisor` shipped as a thin `StateGraph[S]` facade;
  iterative supervisor↔worker primitive; honors **CC-1**'s budget and
  attaches **CC-2**'s policy gates to workers; workers are
  `agents.Agent` (composable — a Supervisor can supervise another
  Supervisor).

- **Phase 38 — Milestone close (`CC-4`)**: tag `llm-agent v0.6.0`;
  CHANGELOG entry; archive `v1.2-ROADMAP.md`/`v1.2-REQUIREMENTS.md` to
  `.planning/milestones/v1.2-*.md`; ship
  `.planning/v1.2-MILESTONE-AUDIT.md`; refresh PROJECT/STATE/ROADMAP/
  REQUIREMENTS to between-milestones.

**Core stdlib-only preserved** — every new gate uses stdlib `regexp`;
every new test uses `ScriptedLLM`; no edit to the four validated
public types (`llm.ChatModel`, `agents.Agent`, `memory.Memory`,
`orchestrate.NodeFunc[S]`); no `/v2` import path (KC-5). Scope is the
core repo only; sister repos stay on v0.2.x — the umbrella dep-currency
gate (KE-6) will fire when they bump core from `v0.5.1 → v0.6.0`, but
that's a future ecosystem-alignment milestone (v1.3-style), not v1.2's
work.

Still deferred (carried forward through v1.1 + v1.2, candidates for a
future milestone): memory tiering / scoping (deferred to v1.3 per
KC-2; `ScopedMemory` decorator shape pre-decided), the `llm-agent-rag`
deployment layer (HTTP service, CLI, caching), incremental community
maintenance, live-Postgres CI wiring, PDF/OCR ingestion,
claim/covariate extraction, a dedicated graph database, productionizing
the customer-support demo.

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
  wherever possible.
- The removed core `llm-agent/rag` facade is not an active development target.
  Any historical planning or research text that mentions it should be treated
  as archival record only, not as implementation guidance.
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
- 2026-05-20: `v1.0` opened — API stabilization for `llm-agent-rag`, scoped
  from `.planning/research/v1.0-api-stabilization-SUMMARY.md`. A quality
  milestone, not a feature one. Keystone calls (KS-1..KS-8): scope is
  `llm-agent-rag` v1.0.0 **only** (the core module + sister repos stay on
  their own tracks; the `contract` package is the cross-repo seam); v1.0 is
  a **freeze** — the only code changes are pre-freeze naming fixes
  (`eval.Evaluator`→`RetrievalEvaluator`, the `ragkit` doc-anchor comment),
  doc comments, and a stdlib API-snapshot gate; a written
  `docs/compatibility.md` states the Go import-compatibility promise; no
  new dependency. The audit found a clean codebase — no `TODO`/`replace`/
  dead code — so "cleanup" is light and the risk is *inventing* refactor
  work, which v1.0 explicitly resists.
- 2026-05-21: `v1.0` shipped — `llm-agent-rag` tagged `v1.0.0`, audit PASS
  6/6 (`RAG-API-01..06`, KS-1..KS-8). The public API is frozen and
  committed to the Go import-compatibility promise; breaking changes from
  here require a `/v2` import path.
- 2026-05-21: `v1.1` opened — ecosystem alignment, scoped from
  `.planning/research/v1.1-ecosystem-alignment-SUMMARY.md`. An alignment /
  housekeeping milestone, not a feature one. Keystone calls (KE-1..KE-7):
  scope is the core `llm-agent` + the three sister repos; `llm-agent-rag`
  is the **untouched fixed point** (KE-2 — no rag re-tag, the back-edge
  `require llm-agent v0.4.0` left as-is); the core RAG facade is repaired
  against rag `v1.0.0` and **proven stdlib-only** (KE-3 — never fix by
  adding a dependency); branches land before tags (KE-4); a coordinated
  dependency-ordered re-tag wave with no `replace` directives (KE-5); the
  umbrella gains a dependency-currency CI gate (KE-6); live-Postgres CI
  wiring stays deferred (KE-7). No new feature, no new dependency.
- 2026-05-20: `v1.1` shipped and closed — audit PASS 5/5
  (`ECO-01..05`, `KE-1..KE-7`). Final coordinated tag set: `llm-agent
  v0.5.1`, `llm-agent-rag v1.0.1`, `llm-agent-otel v0.2.1`,
  `llm-agent-providers v0.2.1`, `llm-agent-customer-support v0.2.2`.
  Phase 34 expanded from 3 slices to 9 mid-flight to satisfy the strict
  dep-currency gate (cascade of patch tags through the back-edge refresh)
  and to correct a topological-order miss in Phase 33's cascade. Three
  architectural trade-offs documented honestly in the audit: the v1.0.0 →
  v1.0.1 freeze-day-after re-tag (KE-2 holds — chore-only patch, no
  exported-symbol move); the topological-order miss (future cascades must
  `tsort` against the dep DAG, not intuition); the rag↔core cycle
  exemption (the one auditable strict-equality exemption — narrow on
  purpose). Umbrella dep-currency gate
  (`scripts/dep-currency-check.sh` + `umbrella.yml` step) is shipped and
  runs green against the live state. Audit:
  `.planning/v1.1-MILESTONE-AUDIT.md`.
- 2026-05-20: `v1.2` opened — Core Capability Deepening (theme: **Core
  v0.6** — capability additions to core `llm-agent`; memory tiering
  deferred to v1.3). Scoped from
  `.planning/research/v1.2-core-capability-deepening-SUMMARY.md`. The
  umbrella's first **core-feature** milestone since v0.3. Keystone
  calls (KC-1..KC-5): Supervisor lives in `orchestrate/` as a
  `StateGraph[S]` facade — not a new `agents/coord` package (KC-1);
  memory tiering is OUT of v1.2 scope and deferred to v1.3 with the
  `ScopedMemory` decorator shape + ctx-keyed scope propagation
  pre-decided (KC-2); policy middleware mirrors `otelmodel.Wrap`'s
  capability-preserving decorator pattern (K3), lives at the model
  boundary, with typed `Gate` event union + sentinel `ErrBlocked` and
  3 built-in regex-based gates (KC-3); budget is **ctx-keyed for
  propagation + a `Tracker` interface for enforcement** with the
  cost-table opt-in / outside core, integrated at the
  `generateFromPrompt` chokepoint (KC-4); every new surface is
  additive — no edit to validated public types, no `/v2` import path
  (KC-5). Core module bumps `v0.5.1 → v0.6.0` (minor, additive).
  Core stdlib-only **preserved**; 4 requirements (`CC-1..04`) across
  4 phases (35-38).

### v1.2 Keystone Decisions (KC-1..KC-5)

| ID | Decision | Phase |
|----|----------|-------|
| KC-1 | Multi-agent coordination lives in `orchestrate/` as `orchestrate.Supervisor` (thin `StateGraph[S]` facade); workers are `agents.Agent` (composable). | Phase 37 |
| KC-2 | Memory tiering is OUT of v1.2 scope — deferred to v1.3 with the `ScopedMemory` decorator + ctx-keyed scope shape pre-decided. | (deferred, v1.3 target) |
| KC-3 | Policy middleware is a `policy` package — capability-preserving `policy.Wrap(model) ChatModel` decorator (mirrors `otelmodel.Wrap`); typed `Gate` event union; sentinel `ErrBlocked`; built-in gates use stdlib `regexp`. | Phase 36 |
| KC-4 | Budget is ctx-keyed for propagation + a `budget.Tracker` interface for enforcement; integrated at `generateFromPrompt` chokepoint; cost-table is opt-in / outside core. | Phase 35 |
| KC-5 | Every new surface is additive — no edit to `llm.ChatModel` / `agents.Agent` / `memory.Memory` / `orchestrate.NodeFunc[S]`; no `/v2` import path; `v0.5.1 → v0.6.0` is a minor bump. | Phases 35-37 |

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

<details>
<summary>v1.1 milestone snapshot</summary>

`v1.1` was the "ecosystem alignment" milestone — four phases (31-34)
bringing the core `llm-agent` + three sister repos current with
`llm-agent-rag v1.0.0` and each to a coherent tagged baseline:

- Phase 31 — core RAG facade re-alignment to `llm-agent-rag v1.0.0`
  (the 7-test `vector dimension mismatch` regression fixed inside the
  facade adapters; an additive `*InMemoryStore.ListDocuments` + a
  `lister` interface + an id-index fallback; core proven stdlib-only)
- Phase 32 — sister-repo branch landing & hygiene (the stranded
  `otel`/`customer-support` branches merged; `providers` confirmed clean
  on `main`; stale branches pruned)
- Phase 33 — coordinated dependency-bump + re-tag wave (`llm-agent
  v0.5.0`, sisters `v0.2.0`)
- Phase 34 — umbrella coherence gate + milestone close (expanded
  mid-flight from 3 to 9 slices to accommodate the strict dep-currency
  gate's cascade-through-the-back-edge requirement and a
  topological-order correction; final coordinated tag set
  `llm-agent v0.5.1`, `llm-agent-rag v1.0.1`,
  `llm-agent-otel v0.2.1`, `llm-agent-providers v0.2.1`,
  `llm-agent-customer-support v0.2.2`;
  `scripts/dep-currency-check.sh` + umbrella CI gate with one auditable
  rag↔core cycle exemption)

Shipped 2026-05-20; 5/5 requirements delivered (`ECO-01..05`); audit PASS
5/5 (`.planning/v1.1-MILESTONE-AUDIT.md`). No new dependency, no new
feature. `llm-agent-rag`'s frozen v1.0.0 API untouched (KE-2); v1.0.1 is
chore-only patch (back-edge refresh, no exported-symbol move).

Archive references:

- Roadmap: `.planning/milestones/v1.1-ROADMAP.md`
- Requirements: `.planning/milestones/v1.1-REQUIREMENTS.md`
- Audit: `.planning/v1.1-MILESTONE-AUDIT.md`

</details>

<details>
<summary>v1.0 milestone snapshot</summary>

`v1.0` was the "API stabilization and the compatibility promise"
milestone — three phases (28-30), a quality milestone freezing the
`llm-agent-rag` public API (no new features):

- Phase 28 — API audit & pre-freeze decisions: the written
  exported-surface inventory (`docs/api-audit-v1.0.md`), the ratified
  breaking renames (`eval.Evaluator`→`RetrievalEvaluator`,
  `eval.Result`→`RetrievalResult`), the `ragkit` root repurposed as a
  documented doc-anchor, stale-README corrections
- Phase 29 — documentation completeness & the compatibility policy: a
  package comment on every package, a name-prefixed comment on every
  exported symbol, the repo made `gofmt`-clean, the written
  `docs/compatibility.md` Go-module compatibility promise
- Phase 30 — API-stability gate, freeze & the tag: the pure-stdlib
  `internal/apisnapshot` exported-surface gate (`api/v1.snapshot.txt` +
  a `go test` regeneration-diff), the `-tags llmagent` CI step, the
  `CHANGELOG.md` `[v1.0.0]` entry

Shipped 2026-05-21; `llm-agent-rag` tagged `v1.0.0`; 6/6 requirements
delivered (audit PASS); no new dependency, no behavior change. Scope was
`llm-agent-rag` only — the core module and sister repos stay on their own
version tracks.

Archive references:

- Roadmap: `.planning/milestones/v1.0-ROADMAP.md`
- Requirements: `.planning/milestones/v1.0-REQUIREMENTS.md`
- Audit: `.planning/v1.0-MILESTONE-AUDIT.md`

</details>

---
phase: 25-fuzzy-entity-resolution-and-evaluation
plan: 03
type: execute
status: complete
completed: 2026-05-19
repo: llm-agent-rag
requirements: [RAG-GRAPH3-06]
---

# Summary: 25-03 docs/graphrag.md updated for v0.8 GraphRAG Tier-3

## Objective

Extend `docs/graphrag.md` for v0.8 GraphRAG Tier-3 — document community
detection, lazy community summaries, the `AskGlobal` map-reduce global-search
path, and fuzzy entity resolution, with the honest tradeoffs (lazy-vs-eager,
fuzzy false-positive risk) called out, and rewrite the deferral section as
"Deferred to v0.9". Docs-only — no code change. Completes RAG-GRAPH3-06 and
Phase 25 / the v0.8 milestone.

## Delivered

`docs/graphrag.md` rewritten from the v0.7 Tier-1 doc into a Tier-1 + Tier-3
doc (+324 lines, −36):

- **Intro reframed** — v0.7 Tier-1 vs v0.8 Tier-3; the "opt-in and additive"
  promise restated for the Tier-3 seams (defaults are no-ops; a Tier-1-only
  SDK is byte-identical to v0.7).
- **Tier-1 section** — the existing three pieces (`graph.EntityExtractor`,
  `store.GraphStore`, `retrieve.GraphRetriever`, `SearchOptions.EnableGraph`)
  kept intact, with one additive line noting `GraphTrace.CommunityIDs` when
  the store also carries detected communities.
- **Tier-3 section** — five new subsections, every type/method name
  cross-checked against the shipped code in `/tmp/llm-agent-rag`:
  1. **Community detection** — `graph.CommunityDetector`;
     `graph.LouvainDetector{Resolution}` (hierarchical default, two-phase
     Louvain, `Level`/`ParentID` hierarchy) and
     `graph.LabelPropagationDetector{}` (single-level alternative);
     `graph.Community` fields (`ID`, `Level`, `ParentID`, sorted
     `EntityIDs`/`RelationIDs`); deterministic + pure stdlib (KG3-6);
     `rag.Options.CommunityDetector`; detection at `Import` when the store is
     a `store.CommunityStore`; full per-namespace re-detection on re-ingest
     (`UpsertCommunities` is replace-all).
  2. **Community summaries** — `graph.CommunitySummarizer` /
     `graph.LLMCommunitySummarizer{Model}`; `graph.CommunityReport` (title +
     paragraph); lazy-by-default model — reports generated at query time and
     cached on the `CommunityStore` keyed by `graph.CommunityContentHash`
     (SHA-256 over sorted membership); the opt-in eager
     `System.PrewarmCommunityReports`; the honest **lazy-vs-eager cost
     tradeoff** spelled out; `rag.ErrCommunitySummarizerRequired` /
     `graph.ErrCommunitySummarizerModelRequired` named.
  3. **Global search** — `System.AskGlobal(ctx, question, GlobalOptions)`;
     the *select → lazy report → map → reduce* flow; a SEPARATE answer path
     from `Ask` (no retrieve/rerank/pack); `GlobalOptions` (`Namespace`,
     `MaxCommunities`); `Answer.Diagnostics.Global` /
     `rag.GlobalDiagnostics` (`CommunityIDs`, `MapScores`, `MapCalls`,
     `ReduceCalls`, `ConsultedReports`); graceful degradation; a short
     **wiring code example** (`rag.New` → `Import` with
     `ingest.ImportOptions` → optional `PrewarmCommunityReports` →
     `AskGlobal`).
  4. **Fuzzy entity resolution** — `graph.EntityResolver`; the opt-in
     pre-pass before `Canonicalize` (KG3-8); `graph.NoopEntityResolver{}`
     default (byte-identical to pre-fuzzy behavior);
     `graph.EmbeddingEntityResolver{Embedder, Threshold}` (same-type-only
     cosine clustering, longest-name canonical form, rewrites entities AND
     relation endpoints so no relation is orphaned); deterministic by
     construction (KG3-6); `rag.Options.EntityResolver`; the **false-positive
     caveat** — opt-in, high `0.92` default threshold, same-type-only,
     "ship conservative, precision over recall".
  5. **Evaluating global search** — `eval.GlobalAsker`,
     `eval.GlobalEvaluator{Asker, Judge, MaxCommunities}`,
     `eval.GlobalEvalResult` (Triad groundedness + answer relevance, NO
     chunk recall@k); `RunGraphAB` stays the local-path chunk-recall harness.
- **"Deferred to v0.9"** — the old "Deferred to v0.8" section rewritten:
  DRIFT search, incremental community maintenance (v0.8 does full
  per-namespace re-detection), path-ranking / subgraph-as-evidence, and
  fuzzy-resolution quality improvements. The "Neo4j is a future
  `GraphStore` impl" note kept (extended to mention `CommunityStore`).

## Files

- `docs/graphrag.md` — modified; extended for v0.8 Tier-3. The only file
  this slice touched.

## Verification

All `<verify>` commands run, all green:

- `GOWORK=off GOCACHE=/tmp/go-build go build ./...` — BUILD-OK
- `GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1` — all 21
  packages `ok`, no FAIL
- `git diff --stat go.mod go.sum` — empty (no new module dependency)
- `git diff --stat` — `docs/graphrag.md` is the only file this slice
  changed. (The other entries in the worktree diff — `graph/graph.go`,
  `rag/import.go`, `rag/options.go`, `rag/system.go`, `retrieve/graph.go`,
  `store/*.go`, `postgres/*.go`, etc. — are the uncommitted Phase 23/24/25
  *code* slices already present in the worktree before 25-03 ran; 25-03 did
  not touch them.)
- core facade (from the core repo `llm-agent`): `GOWORK=off go vet ./rag/...
  && go test ./rag/...` — VET-OK, `ok github.com/costa92/llm-agent/rag`

## Deviations from plan

None — plan executed exactly as written. `docs/graphrag.md` is the only file
in `files_modified` and the only file changed.

One implementation note: the wiring code example was cross-checked against
the real constructor signatures rather than guessed — `rag.New(opts Options)
*System` returns no error, and `System.Import` takes `ingest.ImportOptions`
(not a `rag.ImportOptions`). The example reflects the shipped signatures.

## Notes

- Every type, method, field, error, and keystone reference in the doc was
  cross-checked against the shipped v0.8 code in `/tmp/llm-agent-rag`
  (`graph/community.go`, `graph/louvain.go`, `graph/summary.go`,
  `graph/resolve.go`, `store/community.go`, `store/store.go`,
  `rag/global.go`, `rag/options.go`, `rag/system.go`, `rag/errors.go`,
  `eval/global.go`) and against the Phase 23/24/25 SUMMARYs — no API name in
  the doc is a guess.
- Docs-only: no code change, no new dependency. `go.mod`/`go.sum` diff is
  empty.
- The honest tradeoffs the plan asked for are both stated explicitly: the
  lazy-vs-eager summarization cost tradeoff under "Community summaries", and
  the fuzzy-resolution false-positive caveat ("Apple" company vs fruit;
  precision over recall) under "Fuzzy entity resolution".

## Self-Check: PASSED

- `docs/graphrag.md` — FOUND (modified, +324/−36)
- All `<verify>` commands green; `go.mod`/`go.sum` diff empty; `docs/graphrag.md`
  is the only file 25-03 changed.

## Phase 25 status

All three slices complete:

- **25-01** — `graph.EntityResolver` seam + `NoopEntityResolver` +
  `EmbeddingEntityResolver` (same-type-only cosine clustering over an
  `embed.Embedder`, deterministic); wired as an opt-in pre-pass before
  `Canonicalize` in `Import`; scripted-embedder tests. (RAG-GRAPH3-05)
- **25-02** — `eval` global-search harness: `GlobalAsker` +
  `GlobalEvaluator` + `GlobalEvalResult` over the Triad/`LLMJudge` path;
  `GlobalDiagnostics.ConsultedReports`; a scripted-model CI gate. (RAG-GRAPH3-06)
- **25-03** — `docs/graphrag.md` extended for Tier-3 (communities, global
  search, lazy-vs-eager, fuzzy false-positive caveat, v0.9 deferral list).
  (RAG-GRAPH3-06)

**RAG-GRAPH3-05 and RAG-GRAPH3-06 are delivered.** Phase 25 — the final
v0.8 GraphRAG-tier-3 phase, fuzzy entity resolution and evaluation — is
complete.

## v0.8 milestone status

The v0.8 GraphRAG Tier-3 milestone is **code-complete** across Phases 23-25:

- **Phase 23** — community detection + graph-store persistence:
  `graph.CommunityDetector` / `LouvainDetector` / `LabelPropagationDetector`,
  `store.CommunityStore`, `Import`-time detection. (RAG-GRAPH3-01,
  RAG-GRAPH3-02)
- **Phase 24** — community summaries + global search:
  `graph.CommunitySummarizer` / `LLMCommunitySummarizer` /
  `CommunityContentHash`, `System.AskGlobal` map-reduce, the opt-in eager
  `PrewarmCommunityReports`, local-search community attribution.
  (RAG-GRAPH3-03, RAG-GRAPH3-04)
- **Phase 25** — fuzzy entity resolution + evaluation:
  `graph.EntityResolver` / `EmbeddingEntityResolver`, the `eval`
  global-search harness, and the Tier-3 documentation. (RAG-GRAPH3-05,
  RAG-GRAPH3-06)

All six v0.8 requirements **RAG-GRAPH3-01..06** are delivered. With 25-03
the Tier-3 work is fully documented as well — the v0.8 GraphRAG Tier-3
milestone is code-complete and doc-complete.

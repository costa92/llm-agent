# Phase 24 Research: Community summaries and global search

**Researched:** 2026-05-19
**Phase:** 24 — community summaries & global search (second v0.8 phase)
**Requirements:** RAG-GRAPH3-03, RAG-GRAPH3-04
**Repos:** `llm-agent-rag`
**Upstream:** `.planning/research/v0.8-graphrag-tier3-SUMMARY.md` §4 (lazy vs
eager) and §5 (global search); `23-RESEARCH.md`; Phase 23 code at HEAD.

## Current state (codebase scan, `/tmp/llm-agent-rag` after Phase 23)

- `graph` package — `Community{ID, Level, ParentID, EntityIDs, RelationIDs}`,
  `CommunityDetector` seam, `LouvainDetector`/`LabelPropagationDetector`.
  `graph` already imports `generate` (the `LLMEntityExtractor` precedent in
  `graph/extract.go`) — an LLM-backed summarizer fits the same package.
- `store.CommunityStore` (sibling capability) — `GraphSnapshot`,
  `UpsertCommunities`, `Communities`. In-memory impl `store/community.go`,
  postgres impl `postgres/community.go` (`_communities` table).
- `generate.Model` — `Generate(ctx, Request) (Response, error)`;
  `Request{SystemPrompt, Messages, Metadata}`, `Response{Text, Usage}`.
- `rag.System.Ask` — retrieve → rerank → pack → generate; returns
  `Answer{Text, Hits, Prompt, Citations, Diagnostics, Trace}`.
  `Diagnostics` carries per-stage attribution incl. `GraphTrace`.
- `rag.Options.CommunityDetector` (Phase 23) is already carried as
  `System.communityDetector` and run in `Import`.

## Decision 1 — `CommunityReport` + `CommunitySummarizer` seam (in `graph`)

```go
package graph

// CommunityReport is an LLM-written summary of one community.
type CommunityReport struct {
    CommunityID string
    Title       string
    Summary     string
    ContentHash string // hash of the community's membership; cache key
}

// CommunitySummarizer writes a report for one community.
type CommunitySummarizer interface {
    Summarize(ctx context.Context, c Community, g Graph) (CommunityReport, error)
}
// LLMCommunitySummarizer{Model generate.Model} — scripted-model tested
```

`Summarize` takes the `Community` (member IDs) plus the namespace `Graph`
(`GraphSnapshot`) to look member entity/relation descriptions up by ID. The
LLM impl mirrors `LLMEntityExtractor`: a fixed prompt, lenient parsing of a
title line + summary body, scripted-model tested incl. malformed output.

`graph.CommunityContentHash(c Community) string` — a deterministic
`crypto/sha256` over the sorted `EntityIDs`+`RelationIDs`. The report stores
the hash of the community it was built from; a re-detected community with
the same membership reuses its cached report, a changed one misses.

## Decision 2 — reports persist on `CommunityStore` (lazy cache)

Extend the v0.8 `store.CommunityStore` capability (introduced Phase 23,
unreleased — safe to extend) with two report methods:

```go
PutCommunityReport(ctx, namespace string, report graph.CommunityReport) error
CommunityReport(ctx, namespace, communityID string) (graph.CommunityReport, bool, error)
```

Both the in-memory store and postgres (`_community_reports` table) implement
them. **The `CommunityStore` is the report cache** — no separate in-memory
`System` cache is needed, because global search requires communities, which
require a `CommunityStore`. The cache is *lazy* (KG3-2): reports are
generated at query time on a miss, not at ingest. Staleness is the caller's
check — `System` compares the stored report's `ContentHash` against the live
community's hash and regenerates on mismatch.

## Decision 3 — `rag.System.AskGlobal` is a separate answer path (KG3-4)

```go
func (s *System) AskGlobal(ctx context.Context, question string, opts GlobalOptions) (Answer, error)
type GlobalOptions struct { Namespace string; MaxCommunities int }
```

It does **not** implement `retrieve.Retriever`, does **not** call
`s.retrieve`/rerank/pack. Flow:

1. **Select.** `CommunityStore.Communities(ns)`; take the **coarsest**
   level (highest `Level` — the broadest themes; v0.8 fixes on coarsest,
   level selection is a later refinement). If more than `MaxCommunities`,
   lexically rank by query-token overlap with member entity names and cap.
2. **Lazy reports.** For each selected community: `CommunityReport` lookup;
   on miss or `ContentHash` mismatch, `Summarizer.Summarize` then
   `PutCommunityReport`.
3. **Map.** Per report, `generate.Model`: "given this community summary,
   what does it contribute to Q?" → a partial answer + a self-rated
   helpfulness score (0-100), leniently parsed.
4. **Reduce.** Drop zero-score partials, rank by score, `generate.Model`
   once more to synthesize the survivors into the final answer.

It reuses the `Answer` struct, the `obs.Counter` shell, and `Diagnostics`
(a new global block: communities consulted, per-community map scores,
map/reduce token counts). `Ask` is untouched. This mirrors how v0.7 kept
`MultiHopRetriever` and `GraphRetriever` distinct — different shape, different
type.

## Decision 4 — eager prewarm + local enrichment (24-03)

- `System.PrewarmCommunityReports(ctx, namespace)` — walk every community,
  generate+cache any report that is missing or stale. The thin opt-in eager
  mode (KG3-2): same summarizer, same cache, called eagerly.
- Local-search enrichment: `AskGlobal`'s sibling is v0.7 `GraphRetriever`
  (entity-anchored local search). v0.8 enrichment is modest — surface, in
  the retrieval `Diagnostics`/`GraphTrace`, the community IDs the reached
  entities belong to, so a local answer can be traced to its communities.
  No new retriever, no `HybridRetriever` change.

## Slice breakdown

- **24-01** — `graph`: `CommunityReport` + `CommunitySummarizer` seam +
  `LLMCommunitySummarizer` + `CommunityContentHash`; extend
  `store.CommunityStore` with `PutCommunityReport`/`CommunityReport`;
  in-memory + postgres (`_community_reports` table) impls; conformance
  extension. (RAG-GRAPH3-03)
- **24-02** — `rag.System.AskGlobal` map-reduce global search: select →
  lazy report → map → reduce; `GlobalOptions`; `Diagnostics` global block;
  scripted-model tests on a fixed graph. (RAG-GRAPH3-04)
- **24-03** — opt-in `PrewarmCommunityReports` (eager mode); local-search
  community attribution in `GraphTrace`/`Diagnostics`; a deterministic
  scripted-model global-search worked example. (RAG-GRAPH3-04)

## Risks / notes

- The map/reduce/summarize prompts are LLM-backed → scripted-`generate.Model`
  tests on a fixed tiny graph; lenient parsing per the v0.7 extractor
  precedent. Determinism lives in selection, hashing, and ranking — all
  sorted/deterministic.
- `AskGlobal` over a store that is not a `CommunityStore`, or with no
  communities, returns an empty `Answer` and no error — graceful, like the
  graph signal degrading in v0.7.
- 24-02 depends on 24-01; 24-03 depends on 24-01+02.
- No new module dependency — `crypto/sha256` is stdlib; summary/map/reduce
  reuse `generate.Model`.

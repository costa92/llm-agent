---
phase: 24-community-summaries-and-global-search
plan: 03
type: execute
status: complete
completed: 2026-05-19
repo: llm-agent-rag
requirements: [RAG-GRAPH3-04]
---

> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent-rag` current docs.


# Summary: 24-03 eager prewarm, local community attribution, global-search worked example

## Objective

Add the opt-in eager `System.PrewarmCommunityReports`, surface community
attribution on local (entity-anchored) graph retrieval via
`GraphTrace.CommunityIDs`, and ship a deterministic global-search worked
example. Completes RAG-GRAPH3-04 and Phase 24.

## Delivered

- `rag/global.go` — `func (s *System) PrewarmCommunityReports(ctx, namespace
  string) (int, error)`: the thin opt-in eager mode (keystone KG3-2). It
  type-asserts `store.CommunityStore` (absent → `0, nil`, graceful), reads
  the namespace's community set, and for every community whose cached report
  is missing or stale (`ContentHash` mismatch against the live
  `graph.CommunityContentHash`) calls the configured `s.communitySummarizer`
  and `PutCommunityReport`s the result. It returns the count of reports
  generated; a community whose report is already fresh is left untouched and
  not counted. The namespace graph snapshot is loaded lazily and at most
  once. Same summarizer, same `CommunityStore`-backed cache as the lazy
  `AskGlobal` path — just called ahead of time so the first global query is
  all cache hits. A cache miss with no configured summarizer returns
  `ErrCommunitySummarizerRequired`, mirroring the lazy path.
- `retrieve/graph.go` — `GraphTrace` gains an additive `CommunityIDs
  []string` field. `GraphRetriever.Retrieve` populates it via a new
  `communitiesOf` helper: when the store implements `store.CommunityStore`
  and the namespace has detected communities, it records — sorted and
  deduped — the IDs of the communities the reached entities belong to. A
  store that is not a `CommunityStore`, or a namespace with no communities,
  yields `nil` — no behavior change, graceful degradation, the same contract
  shape as the v0.7 graph signal degrading. A community-lookup error is
  swallowed: community attribution is best-effort enrichment in the
  retrieval `Trace`, never a retrieval failure.
- `examples/graphrag_global_example_test.go` — `Example_graphRAGGlobal`: a
  fully deterministic end-to-end global-search wiring. A
  `DictionaryEntityExtractor` gazetteer builds the graph at ingest,
  `LouvainDetector` is wired as `Options.CommunityDetector`, and a single
  scripted `generate.Model` (`globalExampleModel`) serves all three
  generation kinds — the community summarizer, the per-community map step,
  and the reduce step — routed by request `SystemPrompt` prefix. The example
  imports a fixed four-document corpus, calls `PrewarmCommunityReports`, then
  `AskGlobal`, and prints whether reports were prewarmed, whether communities
  were consulted, and the final answer. Stable `// Output:`, no live model —
  the project's example discipline.
- `rag/global_test.go` — `TestPrewarmCommunityReports` (prewarm fills the
  cache and returns the community count; a second prewarm regenerates
  nothing; a subsequent `AskGlobal` is all cache hits — no summarizer call;
  after one community's membership changes, prewarm regenerates exactly the
  one now-stale report) plus `TestPrewarmCommunityReportsNonCommunityStore`,
  `TestPrewarmCommunityReportsNoCommunities`, and
  `TestPrewarmCommunityReportsSummarizerRequired` (graceful-degradation and
  error-path coverage).
- `retrieve/graph_test.go` — `TestGraphRetrieverCommunityIDs` (a
  `CommunityStore` fixture with two communities partitioning the
  Alpha-Bravo-Charlie chain → `GraphTrace.CommunityIDs` is the sorted,
  deduped `[comm-0 comm-1]`) and `TestGraphRetrieverCommunityIDsNil` (a
  `GraphStore` with no detected communities → `CommunityIDs` is nil).

## Files

- `rag/global.go` — `PrewarmCommunityReports`.
- `rag/global_test.go` — four prewarm tests.
- `retrieve/graph.go` — `GraphTrace.CommunityIDs`; `communitiesOf` helper.
- `retrieve/graph_test.go` — two `CommunityIDs` tests.
- `examples/graphrag_global_example_test.go` — new; `Example_graphRAGGlobal`.

## Verification

All `<verify>` commands run, all green:

- `GOWORK=off GOCACHE=/tmp/go-build go build ./...` — BUILD OK
- `GOWORK=off GOCACHE=/tmp/go-build go vet ./...` — VET OK
- `GOWORK=off GOCACHE=/tmp/go-build go test ./rag/... ./retrieve/...
  ./examples/... -count=1` — all three packages ok (new prewarm, `CommunityIDs`,
  and `Example_graphRAGGlobal` tests PASS)
- `GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1` — every package
  ok, no FAIL
- `git diff --stat go.mod go.sum` — empty (no new dependency)
- core facade (from the core repo `llm-agent`): `GOWORK=off go vet ./rag/...
  && go test ./rag/...` — VET OK, TEST OK

## Deviations from plan

None — plan executed exactly as written. The `files_modified` list was
matched precisely (`rag/global.go`, `rag/global_test.go`, `retrieve/graph.go`,
`retrieve/graph_test.go`, `examples/graphrag_global_example_test.go`); no
extra file was touched.

One implementation note on the prewarm staleness test: `LouvainDetector`
keeps the fixture's dense triangles in stable Level-0 communities, so
re-detecting over a graph with one extra bridge edge does not shift any
community's `ContentHash` (a cross-community edge belongs to no community's
relation set). The test therefore exercises the stale-report path by
re-`UpsertCommunities`-ing the set with one community's `EntityIDs` mutated
directly — a genuine `CommunityContentHash` flip — which is the precise
condition `PrewarmCommunityReports` checks. This is a test-fixture choice,
not a plan deviation: the plan asked for "prewarm after a graph change
regenerates only the stale reports", which is exactly what is asserted (one
stale community → one regeneration, the rest untouched).

## Notes

- `communitiesOf` is best-effort: a `Communities` lookup error degrades to
  `nil` rather than failing the retrieval, consistent with v0.8's "graph
  signal degrades, never crashes" stance. A local answer over a store
  without a community hierarchy is unchanged.
- The worked example's single `globalExampleModel` proves the three
  generation kinds in `AskGlobal` are cleanly separable by `SystemPrompt`
  prefix — the same scripted-model discipline the `rag/global_test.go`
  `globalScriptedModel` already relies on.
- No new module dependency — `PrewarmCommunityReports` reuses
  `graph.CommunityContentHash` (stdlib `crypto/sha256`) and the existing
  summarizer/cache seam; `communitiesOf` is pure stdlib.

## Phase 24 status

All three slices complete:

- **24-01** — `graph`: `CommunityReport` + `CommunitySummarizer` seam +
  `LLMCommunitySummarizer` + `CommunityContentHash`; `store.CommunityStore`
  extended with `PutCommunityReport`/`CommunityReport`; in-memory + postgres
  (`_community_reports` table) impls; conformance extension. (RAG-GRAPH3-03)
- **24-02** — `rag.System.AskGlobal` map-reduce global search: select →
  lazy report → map → reduce; `GlobalOptions`; the `Diagnostics` global
  block; scripted-model tests on a fixed graph. (RAG-GRAPH3-04)
- **24-03** — opt-in `System.PrewarmCommunityReports` (eager mode);
  local-search community attribution via `retrieve.GraphTrace.CommunityIDs`;
  a deterministic scripted-model global-search worked example. (RAG-GRAPH3-04)

**RAG-GRAPH3-03 and RAG-GRAPH3-04 are delivered.** Phase 24 — the second
v0.8 GraphRAG-tier-3 phase, community summaries and global search — is
complete. The Tier-3 `docs/graphrag.md` update and the global-search eval
harness are explicitly Phase 25 (25-03 and 25-02 respectively).

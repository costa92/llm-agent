---
phase: 26-path-ranking-and-subgraph-as-evidence
plan: 01
type: execute
status: complete
completed: 2026-05-19
repo: llm-agent-rag
requirements: [RAG-GRAPH4-01]
---

> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent-rag` current docs.


# Summary: 26-01 Path ranking in the `graph` package

## Objective

Add path ranking to the `graph` package: a `RankedPath` type, a
`PathRanker` seam, and a deterministic pure-stdlib `WeightedPathRanker` that
enumerates and scores simple multi-hop paths within a `graph.Subgraph`. Pure
`graph` work — no store, no LLM, no new dependency. RAG-GRAPH4-01.

## Delivered

- `graph/path.go` (new):
  - `RankedPath struct { EntityIDs []string; RelationIDs []string; Score
    float64 }` — `EntityIDs` is the ordered traversal path[0]..path[n];
    `RelationIDs` the n edges between consecutive entities; `Score` the
    deterministic composite, higher = stronger.
  - `PathRanker` seam — `RankPaths(sub Subgraph, seedPairs [][2]string)
    []RankedPath`. The doc comment states the determinism contract: same
    `Subgraph` + same `seedPairs` => same `[]RankedPath`, byte-for-byte.
  - `WeightedPathRanker struct { LengthDecay float64 }` — the default
    stdlib `PathRanker`:
    - `maxPathLen` const = 2 edges — the `graph`-package path-length cap.
      `graph` cannot import `store` (cycle), so it carries its own bound,
      matching `store`'s `maxGraphDepth`.
    - For each `[2]string` seed pair, a bounded DFS (`enumPaths`) over the
      `Subgraph`'s relations — treated **undirected** for connectivity —
      enumerates *simple* paths (no repeated entity) up to `maxPathLen`
      edges. Adjacency lists are pre-sorted by `(neighbor ID, relation ID)`
      so the DFS visits neighbors in a total, reproducible order. A
      self-loop or a relation whose endpoint is not a reached entity is
      skipped; an empty/`from==to`/unknown-endpoint pair yields nothing.
    - `scorePath` is a composite of three signals already present in the
      graph, no randomness: `LengthDecay^(edges-1)` (length decay; a
      single edge has no penalty), times the product of `edgeWeight` over
      the path's relations (reusing the in-package `edgeWeight` helper from
      `community.go`), times one `(1 + provenanceBonus)` factor per
      consecutive relation pair that shares a `SourceChunkID` (co-attested
      provenance overlap). `LengthDecay <= 0` selects `defaultLengthDecay`
      (0.5); `provenanceBonus` is 0.1.
    - The returned `[]RankedPath` is sorted by `Score` descending,
      tie-broken by `pathKey` — the `EntityIDs` sequence joined on a
      `"\x00"` separator that never appears in an entity ID — a total,
      reproducible order, the same discipline as `Canonicalize` /
      `LouvainDetector`. Every map (`adj`) is drained into a sorted slice
      before iteration.
- `graph/path_test.go` (new): golden-output unit tests —
  - **short-vs-long**: a fixed `Subgraph` with a direct high-weight `a—d`
    edge and a longer low-weight `a—b—d` detour; asserts the short strong
    path ranks first with golden `EntityIDs`/`RelationIDs` order and golden
    absolute scores (5.0 vs 0.5).
  - **determinism**: `RankPaths` on the same input twice is byte-identical
    (`reflect.DeepEqual`).
  - **provenance overlap**: two structurally identical 2-hop routes — one
    co-attested by a shared chunk `c1`, one scattered — asserts the
    co-attested route scores higher, golden scores 0.55 vs 0.5.
  - **no path**: a disconnected pair yields no `RankedPath`, no panic.
  - **empty inputs**: empty `Subgraph` and nil `seedPairs` yield empty
    results.
  - **`maxPathLen` cap**: a pair reachable only via a 3-edge chain yields
    no path; a 2-edge pair on the same chain is still found.
  - **simple path**: a triangle yields exactly the direct and one-detour
    path, and no path repeats an entity.

## Files

- `graph/path.go` — new: `RankedPath`, `PathRanker`, `WeightedPathRanker`,
  the `maxPathLen`/`defaultLengthDecay`/`provenanceBonus` consts, the
  `enumPaths` DFS, `scorePath`, `shareChunk`, `pathKey` helpers.
- `graph/path_test.go` — new: golden-output `WeightedPathRanker` tests.

Both files match the plan's `files_modified` list one-to-one — no extra
file was needed.

## Verification

All six `<verify>` commands run, all green:

- `cd /tmp/llm-agent-rag && GOWORK=off GOCACHE=/tmp/go-build go build ./...`
  — BUILD OK
- `cd /tmp/llm-agent-rag && GOWORK=off GOCACHE=/tmp/go-build go vet ./...`
  — VET OK
- `cd /tmp/llm-agent-rag && GOWORK=off GOCACHE=/tmp/go-build go test ./graph
  -count=1` — `ok github.com/costa92/llm-agent-rag/graph`
- `cd /tmp/llm-agent-rag && GOWORK=off GOCACHE=/tmp/go-build go test ./...
  -count=1` — all packages `ok`, no FAIL
- `cd /tmp/llm-agent-rag && git diff --stat go.mod go.sum` — empty (no new
  dependency)
- core facade (from the core repo `llm-agent`):
  `GOWORK=off go vet ./rag/... && go test ./rag/...` — VET OK, `ok`

## Notes / deviations

- No deviations — the plan was executed exactly as written. The
  `files_modified` list matches one-to-one.
- No new module dependency: `graph/path.go` imports only `sort` and
  `strings` from stdlib, reuses the in-package `edgeWeight` helper, and
  does not import `store`. `git diff --stat go.mod go.sum` is empty.
- Determinism (keystones KG4-4, KG4-6) is structural, not incidental:
  adjacency lists are pre-sorted by `(neighbor ID, relation ID)` so the DFS
  visits neighbors in a total order; the only map drained for iteration
  (`adj`) is sorted before use; the scoring function is a pure arithmetic
  composite with no randomness; and the final `[]RankedPath` sort is total
  — `Score` descending, tie-broken by the `"\x00"`-joined `EntityIDs`
  sequence. A dedicated test ranks the same input twice and asserts
  byte-identical output.
- Path enumeration is bounded by construction: the `Subgraph` is already
  depth-2/fan-out-64 limited upstream, and `enumPaths` additionally caps at
  `maxPathLen` (2) edges over simple paths only — no unbounded recursion.
- Out of scope as planned: wiring into `GraphRetriever` (26-02); the worked
  example and `docs/graphrag.md` path-evidence section (26-03).

## Self-Check: PASSED

- `graph/path.go` and `graph/path_test.go` present in the working tree
  (`/tmp/llm-agent-rag/graph/`).
- No commits made — per operator instruction, all changes left uncommitted
  for a separate commit.

---
phase: 23-community-detection-and-graph-store-persistence
plan: 01
type: execute
status: complete
completed: 2026-05-19
repo: llm-agent-rag
requirements: [RAG-GRAPH3-01]
---

# Summary: 23-01 Community detection in the graph package

## Objective

Add community detection to the `graph` package: a `Community` type, an
optional `Graph.Communities` field, a `CommunityDetector` seam, a
deterministic pure-stdlib hierarchical `LouvainDetector`, and a
single-level `LabelPropagationDetector` alternative behind the same seam.
Pure `graph` work — no store, no LLM, no embedder, no new dependency.
RAG-GRAPH3-01.

## Delivered

- `graph/graph.go`: `Graph` gained an additive `Communities []Community`
  field — a zero-value `Graph` behaves exactly as in v0.7.
- `graph/community.go` (new):
  - `Community{ID, Level, ParentID, EntityIDs, RelationIDs}` — Level 0 is
    the finest partition; `EntityIDs`/`RelationIDs` always sorted.
  - `CommunityDetector` seam — `Detect(ctx, Graph) ([]Community, error)`.
  - `LabelPropagationDetector{}` — deterministic single-level (Level 0)
    label propagation: each entity starts in its own community; entities
    are swept in sorted ID order and each adopts the heaviest-weight label
    among its neighbors; the sweep repeats until stable or a 100-iteration
    cap.
  - shared helpers: `edgeWeight` (`Relation.Weight` when > 0, else 1.0),
    `buildAdjacency` (undirected, parallel edges summed, self-loops and
    dangling endpoints skipped), `entityIDs` (sorted iteration order),
    `communityID` (the deterministic `L{level}-{lowestMember}` rule), and
    `communitiesFromLabels` (groups a label map into sorted communities,
    attaching level-0 relation membership).
- `graph/louvain.go` (new):
  - `LouvainDetector{Resolution float64}` (Resolution <= 0 → 1.0)
    implementing `CommunityDetector`.
  - standard two-phase Louvain — `louvainPass` runs greedy
    modularity-gain moves (nodes visited in sorted ID order, ties broken
    toward the lowest community ID); `coarsen` collapses each community
    into a super-node with aggregated edge weights; the loop repeats, one
    hierarchy level per pass, until a pass makes no merges (cap 64
    levels). `assignParents` links each level-N community to the
    level-(N+1) community holding the same finest entities; the top level
    keeps `ParentID ""`.
  - no randomness, no random restarts; every map is drained into a sorted
    slice before iteration.
- `graph/community_test.go` (new): a fixed two-triangle-plus-bridge
  fixture and a four-triangle hierarchical fixture; golden assertions on
  Louvain's two-cluster split and member/relation IDs; byte-identical
  determinism checks for both detectors; the label-propagation two-cluster
  split; a Louvain hierarchy test asserting >1 level with `ParentID`s that
  resolve to real superset communities one level up; empty-graph and
  single-entity no-panic cases for both detectors.

## Files

- `graph/graph.go` — `Graph.Communities` field (additive).
- `graph/community.go` — new: `Community`, `CommunityDetector`,
  `LabelPropagationDetector`, shared helpers.
- `graph/louvain.go` — new: deterministic hierarchical `LouvainDetector`.
- `graph/community_test.go` — new: golden-output and determinism tests.

## Verification

All `<verify>` commands run, all green:

- `GOWORK=off GOCACHE=/tmp/go-build go build ./...` — BUILD OK
- `GOWORK=off GOCACHE=/tmp/go-build go vet ./...` — VET OK
- `GOWORK=off GOCACHE=/tmp/go-build go test ./graph -count=1` — ok
- `GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1` — all
  packages ok, no FAIL
- `git diff --stat go.mod go.sum` — empty (no new dependency)
- core facade (from the core repo `llm-agent`):
  `GOWORK=off go vet ./rag/... && go test ./rag/... -count=1` — ok

## Notes / deviations

- One deviation from the plan's literal label-propagation spec. The plan
  said "move each to the community most common among its neighbors
  (tie-break by lowest community ID)". Implemented verbatim, synchronous
  label propagation collapses the two-cluster fixture into a single
  community: a bridge node sees three distinct, equally-weighted neighbor
  labels and the "lowest ID" tie-break drags it across the bridge before
  the far cluster has converged. The two-cluster acceptance test (also
  required by the plan) then fails. To satisfy both: `dominantNeighborLabel`
  keeps the current label when it is among the weight maxima, and
  otherwise breaks ties toward the maximal label sharing the longest
  common prefix with the node's current label (then lexically lowest).
  This is fully deterministic — the scan runs over a sorted label slice,
  no randomness — and is what stops a single bridge edge from leaking a
  node into the far cluster. The detector still produces the plan's
  required deterministic single-level partition; the change is a stricter
  tie-break, not an algorithm swap. (Deviation Rule 1 — the literal spec
  produced a broken result that failed the plan's own acceptance test.)
- `LouvainDetector` produces a genuine multi-level hierarchy: the
  four-cluster fixture coarsens into Level 0 (four triangles) and a Level
  1 grouping, with `ParentID`s verified to resolve to real superset
  communities. Community IDs are level-prefixed (`L0-…`, `L1-…`) so a
  community that survives unmerged into a higher level gets a distinct ID
  per level — no cross-level ID collision.
- Determinism is verified directly: `TestLouvainDeterministic` and
  `TestLabelPropagationDeterministic` call `Detect` twice on the same
  graph and assert `reflect.DeepEqual` on the full `[]Community`.
- No new module dependency — `graph` stays a stdlib-only leaf package
  (plus the pre-existing `generate` seam, unused by this slice).
- Out of scope as planned: `store.CommunityStore` persistence (23-02),
  `rag.Import` wiring (23-03), Leiden (a documented future swap behind
  the `CommunityDetector` seam).

## Self-Check: PASSED

- `graph/graph.go`, `graph/community.go`, `graph/louvain.go`,
  `graph/community_test.go` all present in the working tree.
- No commits made — per operator instruction, all changes left
  uncommitted for separate commit.

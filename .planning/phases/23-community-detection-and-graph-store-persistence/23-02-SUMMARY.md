---
phase: 23-community-detection-and-graph-store-persistence
plan: 02
type: execute
status: complete
completed: 2026-05-19
repo: llm-agent-rag
requirements: [RAG-GRAPH3-02]
---

# Summary: 23-02 store.CommunityStore capability + in-memory implementation

## Objective

Add the `store.CommunityStore` optional capability — whole-graph snapshot
read plus community persistence — and implement it for the in-memory store,
with a shared `storetest.RunCommunityConformance` suite. Pure stdlib, no
postgres edit. Completes RAG-GRAPH3-02.

## Delivered

- `store.CommunityStore` — a new optional capability interface, sibling of
  `GraphStore` (Decision 3): `GraphSnapshot`, `UpsertCommunities`,
  `Communities`. Consumers type-assert for it and degrade gracefully. The
  v0.7 `store.GraphStore` interface is left byte-identical — not touched.
- `*InMemoryStore` implements `CommunityStore` (in `store/community.go`):
  - `GraphSnapshot` reconstructs a `graph.Graph` from the namespace's
    `nsGraph` adjacency — entities and relations in deterministic
    sorted-by-ID order, deep-copied; an unknown namespace yields an empty
    `graph.Graph` and no error.
  - `UpsertCommunities` is replace-all: it stores a deep copy of the
    community slice for the namespace (an empty/nil set deletes the entry).
  - `Communities` returns a deep copy sorted by community ID; an unknown
    namespace yields `nil` and no error.
  - `var _ CommunityStore = (*InMemoryStore)(nil)` compile assertion.
- `InMemoryStore` gained a `communities map[string][]graph.Community` field,
  initialized in `NewInMemoryStore`.
- `storetest.RunCommunityConformance(t, factory)` — shared suite mirroring
  `RunGraphConformance`: skips when the store is not a `CommunityStore`;
  otherwise exercises snapshot round-trip, community round-trip, replace
  (not append) semantics, namespace isolation, and unknown-namespace
  empty/no-error.

## Files

- `store/store.go` — added the `CommunityStore` interface.
- `store/community.go` — new; the in-memory `CommunityStore` implementation
  (`GraphSnapshot`, `UpsertCommunities`, `Communities`, deep-copy helpers,
  `var _` assertion).
- `store/community_test.go` — new; runs `RunCommunityConformance` against
  the in-memory store, plus a direct test that a `graph.LouvainDetector`
  hierarchy (detected over a `GraphSnapshot`) survives an
  `UpsertCommunities` -> `Communities` round-trip byte-identically, and a
  deep-copy / no-alias test in both directions.
- `store/storetest/storetest.go` — added `RunCommunityConformance` and its
  five conformance subtests; added the `sort` import.
- `store/inmemory.go` — added the `communities` map field + its
  initialization in `NewInMemoryStore`; added the `graph` import.

## Verification

All `<verify>` commands run, all green:

- `GOWORK=off GOCACHE=/tmp/go-build go build ./...` — BUILD-OK
- `GOWORK=off GOCACHE=/tmp/go-build go vet ./...` — VET-OK
- `GOWORK=off GOCACHE=/tmp/go-build go test ./store/... -count=1` — `ok
  github.com/costa92/llm-agent-rag/store`;
  `TestInMemoryStoreCommunityConformance` (5 subtests),
  `TestCommunityDetectedHierarchyRoundTrip`,
  `TestCommunityUpsertDoesNotAliasCallerSlice` all PASS
- `GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1` — all 20
  packages `ok`, no FAIL
- `git diff --stat go.mod go.sum` — empty (no new module dependency)
- core facade (from the core repo `llm-agent`): `GOWORK=off go vet
  ./rag/... && go test ./rag/... -count=1` — `ok
  github.com/costa92/llm-agent/rag`

## Notes

- `inmemory.go` did not previously import `graph`; the import was added
  alongside the new `communities` field. The plan's reference to a `New`
  constructor is the actual `NewInMemoryStore` — the field is initialized
  there.
- `UpsertCommunities` deep-copies on the way in and `Communities` deep-copies
  on the way out, so stored state is isolated from caller state in both
  directions — verified by `TestCommunityUpsertDoesNotAliasCallerSlice`.
- postgres `CommunityStore`, `Import` wiring, and `Options.CommunityDetector`
  are deliberately out of scope — slice 23-03. The build stays green because
  `CommunityStore` is a new interface; the postgres store simply does not
  satisfy it yet, which is no compile break.
- No new module dependency — the capability reuses `graph` types and the
  in-memory implementation is pure stdlib.

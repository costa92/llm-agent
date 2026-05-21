---
phase: 22-graph-traversal-retrieval
plan: 01
type: execute
status: complete
completed: 2026-05-18
repo: llm-agent-rag
requirements: [RAG-GRAPH-05]
---

> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent-rag` current docs.


# Summary: 22-01 EntityLinker + GraphRetriever

## Objective

Add the graph-traversal retriever — an `EntityLinker` seam, a
`LexicalEntityLinker`, and a `GraphRetriever` that links a query to seed
entities, traverses their bounded neighborhood, and returns the provenance
chunks scored by graph proximity. First half of RAG-GRAPH-05.

## Delivered

- `retrieve/graph.go` (new):
  - `EntityLinker` seam — `Link(ctx, query, namespace, gs store.GraphStore)
    ([]graph.Entity, error)`. It lives in `retrieve` (not `graph`):
    `Link` references `store.GraphStore`, and `store` imports `graph`, so
    `graph` must not import `store`.
  - `LexicalEntityLinker` — resolves a query to seeds by matching its
    whitespace tokens (and the whole query) via `gs.FindEntities`, deduped
    by entity ID. No embedder — the zero-LLM default.
  - `GraphTrace{SeedEntityIDs, ReachedEntityIDs, MaxHop}`.
  - `GraphRetriever{Linker, Store, MaxDepth, HopDecay}` implementing
    `retrieve.Retriever`: type-asserts the store for `store.GraphStore`
    (empty result + no error when absent); links the query; `Neighborhood`
    from the seed IDs (depth `<=0`→1, hard cap 2); maps each reached
    entity's `SourceChunkIDs` to its best (lowest-hop) score
    `HopDecay^hop`; fetches chunks via `Store.Get` (an `ErrNotFound` is
    skipped, not fatal); sorts, truncates to `req.TopK`; returns a `Trace`
    with the `Graph` block.
- `retrieve.Trace` gained a `Graph GraphTrace` field.

## Files

- `retrieve/graph.go` — new: linker seam + `GraphRetriever`.
- `retrieve/retrieve.go` — `Trace.Graph` field.
- `retrieve/graph_test.go` — new: `graphFixture` (in-memory store + a
  3-entity chain graph); linker, retriever (proximity decay: seed > 1-hop
  > 2-hop; `Trace.Graph` populated), and non-`GraphStore` degradation tests.

## Verification

All `<verify>` commands run, all green:

- `GOWORK=off go build ./...` — BUILD OK
- `GOWORK=off go vet ./...` — VET OK
- `GOWORK=off go test ./retrieve -count=1` — ok
- `GOWORK=off go test ./... -count=1` — all packages ok (no FAIL)
- `git diff --stat go.mod go.sum` — empty (no new dependency)
- core facade (from the core repo `llm-agent`): `GOWORK=off go vet
  ./rag/... && go test ./rag/...` — ok

## Notes

- The `Trace.Graph` field was added here (22-01), not 22-02 — the
  `GraphRetriever` must return its trace through it. 22-02's plan listed
  it; this slice took it because of the compile dependency.
- `EmbeddingEntityLinker` is deferred (no stored entity embeddings in
  v0.7) — the `EntityLinker` seam keeps it a clean future addition.
- No new module dependency — `retrieve` already imports `store`; `graph`
  is a new per-file import in `retrieve/graph.go`.

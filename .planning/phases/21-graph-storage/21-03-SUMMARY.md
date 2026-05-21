---
phase: 21-graph-storage
plan: 03
type: execute
status: complete
completed: 2026-05-18
repo: llm-agent-rag
requirements: [RAG-GRAPH-04]
---

> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent-rag` current docs.


# Summary: 21-03 Import graph persistence + re-ingest reconciliation

## Objective

Complete RAG-GRAPH-04: wire `Import` to persist the canonicalized graph
into a `store.GraphStore`-capable store and reconcile it on a
`ReplaceSource` re-ingest — provenance-based removal of the old
contributions before the re-extracted subgraph is union-merged in.

## Delivered

- `rag/import.go` `Import`:
  - type-asserts `s.store` for `store.GraphStore` once
    (`graphStore, isGraphStore`); `persistGraph` = an extractor is
    configured **and** the store is a `GraphStore`.
  - on a `ReplaceSource` re-ingest, before `RemoveByFilter` deletes the
    source's chunks, `List`s them to capture their prior chunk IDs into
    `staleGraphChunkIDs`.
  - after `Canonicalize`: when `persistGraph`, `RemoveGraphBySource` the
    stale chunk IDs (reconcile — drop the old contributions, GC orphans),
    then `UpsertGraph` the re-extracted subgraph. Remove-then-upsert, so a
    reused chunk ID ends up with the new content's provenance.
  - a `GraphStore` error aborts the import with a wrapped error.
  - a store that is not a `GraphStore` degrades gracefully — `res.Graph` is
    still produced and returned, persistence is skipped.

## Files

- `rag/import.go` — graph-persistence + re-ingest reconciliation.
- `rag/graph_test.go` — `plainStore` (a `store.Store` that is not a
  `GraphStore`); persistence, re-ingest-reconciliation, and
  graceful-degradation tests.

## Verification

All `<verify>` commands run, all green:

- `GOWORK=off go build ./...` — BUILD OK
- `GOWORK=off go vet ./...` — VET OK
- `GOWORK=off go test ./rag ./store/... -count=1` — ok;
  `TestImportReingestReconcilesGraph` PASS (Paris dropped on re-ingest,
  Berlin added)
- `GOWORK=off go test ./... -count=1` — all packages ok (no FAIL)
- `git diff --stat go.mod go.sum` — empty (no new dependency)
- core facade (from the core repo `llm-agent`): `GOWORK=off go vet
  ./rag/... && go test ./rag/...` — ok

## Notes

- Re-ingest reconciliation uses `store.List` (filtered by `source_id`)
  before `RemoveByFilter` to recover the prior chunk IDs — `RemoveByFilter`
  returns only a count, not IDs. One extra `List` per re-ingested source.
- The re-ingest path is verified on the in-memory `GraphStore`; the
  postgres path is env-gated (carried-forward live-Postgres debt — see
  21-02-SUMMARY).
- No new module dependency.

## Phase 21 status

All three slices complete. RAG-GRAPH-03 (the `store.GraphStore` capability,
in-memory + `postgres` implementations, shared `RunGraphConformance`) and
RAG-GRAPH-04 (incremental re-ingest reconciliation) are delivered. Phase 21
is complete; the graph is now persisted and traversable, ready for Phase 22
(`GraphRetriever`).

# Phase 09-01 Summary

Date: 2026-05-14
Repo: `llm-agent-rag`
Plan: [09-01-PLAN.md](/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/phases/09-source-aware-ingestion-and-index-lifecycle/09-01-PLAN.md)

## Objective

Move standalone ingestion beyond flat text import by introducing source-aware
lineage metadata, basic markdown section awareness, and explicit re-import
lifecycle semantics for the default store path.

## Delivered

- Added additive source-lineage fields to `ingest.Document`:
  - `SourceID`
  - `Version`
  - `Checksum`
  - `EmbeddingVersion`
- Standardized lineage metadata propagation into chunks using metadata keys:
  - `source_id`
  - `version`
  - `checksum`
  - `embedding_version`
- Added `MarkdownSplitter` with section-aware metadata:
  - `heading`
  - `heading_level`
  - `section_path`
- Kept non-markdown or heading-free text on the existing char-splitting
  fallback path.
- Added lifecycle contract support for source replacement:
  - `ingest.ImportOptions.ReplaceSource`
  - `store.Store.RemoveByFilter(...)`
  - `InMemoryStore.RemoveByFilter(...)`
- Updated `rag.System.Import(...)` so re-importing with `ReplaceSource=true`
  removes existing chunks for the same `source_id` before upserting the new
  ones.

## Files

- `/tmp/llm-agent-rag/ingest/types.go`
- `/tmp/llm-agent-rag/ingest/import.go`
- `/tmp/llm-agent-rag/ingest/splitter.go`
- `/tmp/llm-agent-rag/ingest/splitter_test.go`
- `/tmp/llm-agent-rag/ingest/markdown_splitter_test.go`
- `/tmp/llm-agent-rag/store/store.go`
- `/tmp/llm-agent-rag/store/inmemory.go`
- `/tmp/llm-agent-rag/store/inmemory_test.go`
- `/tmp/llm-agent-rag/rag/import.go`
- `/tmp/llm-agent-rag/rag/system_test.go`

## Verification

```bash
cd /tmp/llm-agent-rag
GOWORK=off GOCACHE=/tmp/go-build go test ./ingest ./rag -count=1
GOWORK=off GOCACHE=/tmp/go-build go test ./store ./rag ./ingest -count=1
GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1
```

Result:

- targeted ingest/rag tests: pass
- targeted store/rag/ingest tests: pass
- standalone full test suite: pass

## Notes

- The original Phase 9 planning thread began at lineage metadata (`09-01`) but
  the first execution pass also completed the minimum viable markdown splitter
  and replace-by-source lifecycle contract needed for later retrieval phases.
- The next natural step is to release these standalone changes and then decide
  whether the core repo needs any compatibility surface updates for the new
  ingest semantics.

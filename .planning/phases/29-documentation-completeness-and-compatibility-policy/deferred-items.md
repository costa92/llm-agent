# Deferred items — Phase 29

Out-of-scope discoveries logged during execution. Not fixed by the slice that found them.

## Pre-existing `gofmt` non-compliance (found during 29-01)

`gofmt -l` flags 8 files in `/tmp/llm-agent-rag`. All are **pre-existing** at
tag `v0.6.0` — none were introduced by slice 29-01 (a comment-only slice). The
common issue is import-block ordering (`corellm` sorted before
`llm-agent-rag/generate`, etc.).

Flagged files:

- `ingest/splitter.go`
- `rag/community_test.go`
- `rag/drift.go`
- `retrieve/graph.go`
- `store/inmemory.go`
- `adapter/llmagent/model.go` (import block only — the 29-01 package comment is gofmt-clean)
- `adapter/llmagent/model_test.go`
- `adapter/llmagent/tool_test.go`

Slice 29-01 did not reformat them — out of scope for a comment-only slice and
unrelated to the package-doc work. A future formatting-cleanup slice (or the
operator) should run `gofmt -w` over these files.

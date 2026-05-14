# Phase 10-01 Summary

Date: 2026-05-14
Repo: `llm-agent-rag`
Plan: [10-01-PLAN.md](/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/phases/10-retrieval-policies-hybrid-recall-and-context-packing/10-01-PLAN.md)

## Objective

Introduce the first retrieval-policy seam in the standalone RAG SDK so query
preprocessing and retriever selection become explicit contracts instead of
staying hardcoded inside `rag.System.Retrieve(...)`.

## Delivered

- Added a new standalone `retrieve/` package.
- Added a `QueryPreprocessor` seam with a default `NoopPreprocessor`.
- Added a `Retriever` seam with a default `DenseRetriever`.
- Added structured retrieval request and trace types for the new policy layer.
- Updated `rag.Options` so a caller can inject:
  - `Preprocessor`
  - `Retriever`
- Updated `rag.System` to initialize safe defaults when neither seam is
  provided.
- Rewired `rag.System.Retrieve(...)` to delegate through the new policy layer
  while preserving the old dense-retrieval behavior by default.
- Added tests proving:
  - no-op preprocessing preserves the incoming query
  - dense retrieval still respects the store contract and filters
  - `System.Retrieve(...)` can consume a custom preprocessor

## Files

- `/tmp/llm-agent-rag/retrieve/retrieve.go`
- `/tmp/llm-agent-rag/retrieve/retrieve_test.go`
- `/tmp/llm-agent-rag/rag/options.go`
- `/tmp/llm-agent-rag/rag/system.go`
- `/tmp/llm-agent-rag/rag/retrieve.go`
- `/tmp/llm-agent-rag/rag/system_test.go`

## Verification

```bash
cd /tmp/llm-agent-rag
GOWORK=off GOCACHE=/tmp/go-build go test ./rag ./retrieve -count=1
GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1
```

Result:

- `go test ./rag ./retrieve`: pass
- `go test ./...`: pass

## Notes

- This is the seam-setting slice only. Lexical retrieval, hybrid fusion, and
  MQE/HyDE migration into the policy layer remain follow-up work in the rest of
  Phase 10.
- The old public retrieval path still behaves like dense retrieval unless a
  caller opts into a custom preprocessor or retriever.

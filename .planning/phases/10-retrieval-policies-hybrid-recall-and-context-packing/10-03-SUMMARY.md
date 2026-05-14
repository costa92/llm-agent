# Phase 10-03 Summary

Date: 2026-05-14
Repo: `llm-agent-rag`
Plan: [10-03-PLAN.md](/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/phases/10-retrieval-policies-hybrid-recall-and-context-packing/10-03-PLAN.md)

## Objective

Move MQE and HyDE into the standalone retrieval policy layer so callers enable
query expansion through `rag.SearchOptions` and the retrieval stack handles
query generation plus result merging internally.

## Delivered

- Extended standalone retrieval inputs with:
  - `EnableMQE`
  - `EnableHyDE`
  - `MQECount`
- Added `retrieve.LLMExpansionPreprocessor`:
  - uses the configured standalone `generate.Model`
  - performs MQE expansion and HyDE hypothetical generation when enabled
  - emits ordered query variants for downstream retrieval
- Added `retrieve.VariantRetriever`:
  - runs each query variant through a wrapped base retriever
  - merges duplicate hits by chunk ID
  - keeps the strongest score per chunk
  - trims deterministically to `TopK`
- Updated default `rag.System` construction so standalone retrieval now uses:
  - `LLMExpansionPreprocessor`
  - `VariantRetriever`
  - `DenseRetriever` as the default base retriever
- Updated `rag.SearchOptions` and `rag.System.Retrieve(...)` to propagate MQE /
  HyDE controls into the standalone retrieval layer.
- Simplified `adapter/llmagent` so `search` and `ask` no longer implement their
  own MQE/HyDE query loops.
- Added tests proving:
  - LLM expansion builds ordered variants from MQE and HyDE
  - model-less MQE requests fail cleanly
  - multi-query retrieval merges and ranks results deterministically

## Files

- `/tmp/llm-agent-rag/retrieve/retrieve.go`
- `/tmp/llm-agent-rag/retrieve/retrieve_test.go`
- `/tmp/llm-agent-rag/rag/options.go`
- `/tmp/llm-agent-rag/rag/retrieve.go`
- `/tmp/llm-agent-rag/rag/system.go`
- `/tmp/llm-agent-rag/adapter/llmagent/tool.go`
- `/tmp/llm-agent-rag/README.md`
- `/tmp/llm-agent-rag/CHANGELOG.md`

## Verification

```bash
cd /tmp/llm-agent-rag
GOWORK=off GOCACHE=/tmp/go-build go test ./retrieve ./rag -count=1
GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1
```

Result:

- `go test ./retrieve ./rag`: pass
- `go test ./...`: pass

## Notes

- The default standalone retrieval path now owns MQE/HyDE orchestration, but
  the same seams can wrap lexical or hybrid base retrievers later.
- Phase `10-04` remains the next retrieval-policy follow-up for reranking and
  context-packing.

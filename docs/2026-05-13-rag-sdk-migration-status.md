# RAG SDK Migration Status

Date: 2026-05-13
Project: `github.com/costa92/llm-agent`
Embedded SDK path: `third_party/llm-agent-rag`
Status: in-repo extraction complete, external release pending

## Summary

The original `rag/` package in `llm-agent` has been split into two layers:

- `third_party/llm-agent-rag`
  - standalone SDK core
  - provider-agnostic import / retrieve / ask orchestration
  - `advanced/` LLM-assisted retrieval helpers
  - optional `adapter/llmagent` bridge
- `rag/`
  - compatibility facade for existing `llm-agent` callers
  - preserves historical public API and tests
  - delegates most implementation to the standalone SDK

## What Moved

The following responsibilities now primarily live in the embedded SDK:

- chunk splitting
- hash embedding
- in-memory vector storage
- import orchestration
- retrieval orchestration
- prompt rendering for default QA
- LLM-assisted query expansion and HyDE prompt logic
- optional `llm-agent` adapter code

The main repo `rag/` package now mainly provides:

- compatibility types and error values
- compatibility method signatures
- adapter/conversion glue for legacy callers
- tool facade behavior expected by existing tests and downstream packages

## Current Compatibility Shape

Existing packages such as `memory/` and `context/` still import:

- `rag.Embedder`
- `rag.NewHashEmbedder`
- `rag.CosineSimilarity`
- `rag.Document`
- `rag.SearchHit`
- `rag.NewInMemoryStore`
- `rag.RAGSystem`
- `rag.AsTool`

These entry points continue to work, but they are no longer the source of truth
for implementation logic.

## Completed Milestones

Completed in this repo:

- design and implementation planning docs created
- standalone SDK scaffolded and embedded under `third_party/llm-agent-rag`
- `rag.RAGSystem` converted into a compatibility facade over the SDK
- `MQE` / `HyDE` prompt logic moved into SDK `advanced/`
- tool-level namespace support wired through the facade
- `rag/chunk.go`, `rag/embedder.go`, and `rag/store.go` converted into SDK-backed compatibility wrappers
- `adapter/llmagent/tool.go` extended to support:
  - `namespace`
  - `enable_mqe`
  - `enable_hyde`
  - `mqe_count`

## Known Boundaries

### 1. Embedded SDK is still in-repo

The standalone SDK currently lives at:

- `third_party/llm-agent-rag`

It has not yet been pushed and versioned as an external repository release from
this working tree.

### 2. `adapter/llmagent` is dev-only in the standalone module

The core SDK intentionally does not keep a hard dependency on
`github.com/costa92/llm-agent` in its publishable `go.mod`.

That means:

- default SDK tests pass without `llm-agent`
- tagged adapter tests require a temporary local `require` / `replace`

This is deliberate and preserves the standalone core boundary.

### 3. Main-repo `rag/tool.go` is still the default public entry point

Although the SDK adapter has now reached near-feature parity for the tool path,
the main repo still exposes `rag.AsTool` from the compatibility layer.

That is acceptable for now because:

- existing callers do not need to change
- default repo tests stay simple
- the facade keeps historical behavior stable

## Recommended Next Step

The next high-value step is externalization, not more in-repo refactoring.

Recommended order:

1. create/push the standalone repository from `third_party/llm-agent-rag`
2. tag an initial version
3. replace the local `replace github.com/costa92/llm-agent-rag => ./third_party/llm-agent-rag`
   with a real module version in `llm-agent`
4. decide whether `rag/` remains as a permanent compatibility package or begins
   a deprecation path

## Verification Snapshot

Verified during migration:

- main repo:
  - `GOWORK=off GOCACHE=/tmp/go-build go test ./...`
- embedded SDK core:
  - `GOWORK=off GOCACHE=/tmp/go-build go test ./...`

`adapter/llmagent` tagged tests are intentionally excluded from the default
standalone verification path until a temporary local dependency is added.

# Phase 13-04 Summary

Date: 2026-05-15
Repos: `llm-agent-rag` + `llm-agent`
Plan: [13-04-PLAN.md](/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/phases/13-evaluation-feedback-loop-and-ecosystem-contract/13-04-PLAN.md)

## Objective

Close RAG-ECO-02 by making the cross-repo contract explicit:
contract drift between standalone `llm-agent-rag` and the core
`llm-agent/rag` facade now surfaces as a compile-time test failure
on every `go test` run. No new workflow plumbing required.

## Delivered

- **Standalone-side gate**:
  `/tmp/llm-agent-rag/contract/contract_test.go` —
  `package contract_test` referencing every standalone symbol the
  core `llm-agent/rag` facade consumes. Pinned via `_ = symbol`
  pattern (compile-pin without runtime cost):
  - `embed.Vector`, `embed.Embedder`, `embed.HashEmbedder`,
    `embed.NewHashEmbedder`, `embed.CosineSimilarity`
  - `generate.{Message, Request, Response, Model}`
  - `ingest.{Document, Chunk, ImportOptions, Splitter,
    CharSplitter}` + the four `Metadata*Key` constants the core
    relies on
  - `prompt.Template`, `prompt.DefaultQATemplate`
  - `rag.{System, Options, SearchOptions, AskOptions, Trace,
    Diagnostics, Answer, Citation, Observer, ImportTrace, New,
    ErrEmptyQuery}`
  - `store.{Store, StoredChunk, Query, Hit, Filter, Stats,
    InMemoryStore, NewInMemoryStore, ErrNotFound,
    ErrDimensionMismatch}`
  - `retrieve.{Trace, RoutePolicyTrace, RouteCandidate,
    TrajectoryStep, Request}` (consumed transitively via
    `rag.Trace`)
- **Core-side gate**:
  `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/contract_test.go` —
  `package rag` (internal test, sees unexported state too)
  pinning every exported facade symbol:
  - types: `RAGSystem`, `Options`, `SearchOptions`, `Document`,
    `SearchHit`, `VectorStore`, `StoreStats`, `InMemoryStore`
  - constructors: `New`, `NewInMemoryStore`, `AsTool`
  - errors: `ErrEmptyQuery`, `ErrLLMRequired`,
    `ErrStoreNotFound`, `ErrDimMismatch`
  - methods on `*RAGSystem` and `*InMemoryStore` pinned via
    method-value references so a signature change breaks
    compilation, not just a method-removal
- Both contract files include a header comment establishing
  that the file IS the contract: additions widen the surface
  deliberately; removals are breaking changes requiring
  coordinated PRs.

## Files

- `/tmp/llm-agent-rag/contract/contract_test.go` (new, ~110 LOC)
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/contract_test.go` (new, ~80 LOC)

## Verification

```bash
cd /tmp/llm-agent-rag
GOWORK=off GOCACHE=/tmp/go-build go build ./...
GOWORK=off GOCACHE=/tmp/go-build go vet ./...
GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1

cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent
GOWORK=off go vet ./rag/...
GOWORK=off go test ./rag/... -count=1
```

Result:

- standalone `go test ./...` (17 packages including new `contract`): pass
- standalone `TestContract_ConsumedByCoreFacade`: pass (compile-pin gate)
- core `TestContract_PublicFacade`: pass (compile-pin gate)
- core `go test ./rag/...`: pass

## Notes & Design Decisions

- **Why two files, not one.** The contract has two sides. The
  standalone gate pins what the core consumes; the core gate pins
  what the core exports. Without both, a rename inside the core
  facade would still pass the standalone gate (correctly — the
  standalone has no opinion on what the core exports). Both files
  must shift for a coordinated breaking change.
- **Method-value references catch signature drift.** Pinning a
  method by name (`_ = rs.AddText`) catches removal. Pinning the
  method *value* (`_ = rs.AddText` as a function value) catches
  signature changes too — a different parameter list won't be
  assignable to the same type.
- **No workflow files touched.** CLAUDE.md still says to leave
  `.github/workflows/test.yml` alone; the contract tests fire
  during the normal `go test` runs the workflow already executes,
  so no plumbing is needed to enforce the gate.
- **Compile success is the assertion.** No runtime checks beyond
  `_ = t` to keep the test body referenced — the gate is purely
  structural.

## Phase 13 close-out

With 13-04 landed, Phase 13's stated goal (evaluation + feedback
loop + ecosystem contract) is fully delivered:

- `13-01` shipped the eval framework + JSONL format + seed dataset CI gate
- `13-02` shipped production / backend / compatibility docs
- `13-03` shipped the online-to-offline feedback workflow that
  writes captured Asks into the eval format
- `13-04` shipped the cross-repo contract-drift gates

Phase 13 status in ROADMAP is now `complete 2026-05-15`.

## v0.5 milestone status

All 13 numbered phases in the v0.5 milestone (Phase 8 → 13) are
now complete. Remaining items on the pending list are
operational (live-Postgres CI wiring, `adapter/llmagent` test
triage, OTel sister-repo wiring) and are not gating the milestone.

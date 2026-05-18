---
phase: 14-lexical-retrieval-and-principled-hybrid-fusion
plan: 03
type: execute
status: complete
completed: 2026-05-15
repo: llm-agent-rag
requirements: [RAG-RETR2-02]
---

# Summary: 14-03 Configurable RRF + per-signal fusion attribution

## Objective

Make `HybridRetriever`'s reciprocal rank fusion configurable (RRF constant)
and expose per-signal fusion attribution in the retrieval `Trace`, so callers
can see how dense, lexical, and structure signals each ranked every chunk.

## Delivered

- `retrieve.FusionAttribution{ChunkID, DenseRank, LexicalRank, StructureRank,
  RRFScore}` — records how each signal ranked one chunk. Rank `0` means the
  signal did not return that chunk.
- `Trace.Fusion []FusionAttribution` — additive trace field, sorted by
  `RRFScore` descending (tie-break by `ChunkID`).
- `HybridRetriever.RRFConstant float64` — the `k` in `1/(k + rank)`. Zero
  resolves to the standard default `60`, reproducing pre-slice fusion scores
  bit-for-bit (`float64(i+1+60)` and `60.0 + float64(i+1)` are exact-equal for
  these integers).
- `HybridRetriever.Retrieve` now records each signal's 1-based rank per chunk
  while applying RRF, and emits `Trace.Fusion`. Existing trace fields and hit
  ordering are unchanged at the default constant.

## Files

- `retrieve/retrieve.go` — `FusionAttribution` type, `Trace.Fusion` field,
  `HybridRetriever.RRFConstant` field, per-signal rank recording in `Retrieve`.
- `retrieve/retrieve_test.go` — `hitList` helper; tests for fusion
  attribution + multi-signal boost, default-constant behavior preservation,
  and RRF-constant configurability (an ordering flip between k=60 and k=1).

## Verification

All `<verify>` commands run, all green:

- `go build ./...` — BUILD OK
- `go vet ./...` — VET OK
- `go test ./retrieve ./rag ./contract -count=1` — ok (contract gate passes)
- `go test ./... -count=1` — all 14 packages ok
- core: `GOWORK=off go vet ./rag/... && go test ./rag/...` — ok

## Notes

- The `contract` package gate passes in both repos — adding an exported field
  to `Trace` did not break the pinned cross-repo surface.
- RRF remains rank-based and weightless by design; unequal per-signal weights
  are a possible future knob, deliberately not added here.
- `Trace.Fusion` is on the standalone retrieval trace only; facade exposure
  through `llm-agent/rag` can follow when a consumer needs it.

## Phase 14 status

All three slices (14-01, 14-02, 14-03) complete. RAG-RETR2-01 (real BM25
lexical retrieval, in-memory + Postgres) and RAG-RETR2-02 (principled,
configurable, attributable hybrid fusion) are delivered. Phase 14 is complete
pending the still-deferred live-Postgres verification of the `tsvector` path.

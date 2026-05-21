---
phase: 22-graph-traversal-retrieval
plan: 02
type: execute
status: complete
completed: 2026-05-18
repo: llm-agent-rag
requirements: [RAG-GRAPH-05]
---

> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent-rag` current docs.


# Summary: 22-02 graph fused as a fourth RRF signal

## Objective

Fuse graph retrieval into `HybridRetriever` as a fourth RRF signal —
alongside dense, lexical, and structure — gated by an `EnableGraph` toggle,
with the graph's contribution attributed in the trace and `rag.Diagnostics`.
Completes RAG-GRAPH-05.

## Delivered

- `retrieve.HybridRetriever` gained a `Graph Retriever` field;
  `retrieve.Request` gained `EnableGraph bool`; `FusionAttribution` gained
  `GraphRank int`.
- `HybridRetriever.Retrieve`: when `req.EnableGraph && r.Graph != nil`, the
  graph retriever runs and `apply(graphHits, graphRank)` folds it into the
  RRF fusion with the identical `1/(k+rank)` formula; `FusionAttribution`
  records the per-chunk `GraphRank`; the graph sub-trace's `Graph` block is
  carried onto the returned `Trace`. With graph off/unset, hybrid
  retrieval is byte-for-byte the prior behavior.
- `rag.SearchOptions` gained `EnableGraph`; `rag/retrieve.go` maps it onto
  `retrieve.Request.EnableGraph`.
- `rag.Diagnostics` gained `GraphTrace retrieve.GraphTrace`, populated in
  `Ask` from the retrieval trace.

## Files

- `retrieve/retrieve.go` — `HybridRetriever.Graph`, `Request.EnableGraph`,
  `FusionAttribution.GraphRank`, the graph signal in `Retrieve`.
- `rag/options.go` — `SearchOptions.EnableGraph`.
- `rag/retrieve.go` — `EnableGraph` mapping.
- `rag/system.go` — `Diagnostics.GraphTrace`.
- `rag/ask.go` — populate `Diagnostics.GraphTrace`.
- `retrieve/graph_test.go` — hybrid graph-on (fuses in, `GraphRank > 0`,
  `Trace.Graph` carried) and graph-off (signal does not run) tests.

## Verification

All `<verify>` commands run, all green:

- `GOWORK=off go build ./...` — BUILD OK
- `GOWORK=off go vet ./...` — VET OK
- `GOWORK=off go test ./retrieve ./rag -count=1` — ok;
  `TestHybridRetrieverFusesGraphSignal` PASS
- `GOWORK=off go test ./... -count=1` — all packages ok (no FAIL)
- `git diff --stat go.mod go.sum` — empty (no new dependency)
- core facade (from the core repo `llm-agent`): `GOWORK=off go vet
  ./rag/... && go test ./rag/...` — ok

## Notes

- `HybridRetriever.Graph` is wired by the caller — the default `rag.New`
  retriever wiring is unchanged (KG-4: default behavior unaffected unless
  graph retrieval is explicitly provided and `EnableGraph` set). The
  worked example (22-03) shows the wiring.
- No new module dependency.

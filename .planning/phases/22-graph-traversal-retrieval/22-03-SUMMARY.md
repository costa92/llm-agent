---
phase: 22-graph-traversal-retrieval
plan: 03
type: execute
status: complete
completed: 2026-05-18
repo: llm-agent-rag
requirements: [RAG-GRAPH-06]
---

> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent-rag` current docs.


# Summary: 22-03 graph eval A/B + worked example + docs

## Objective

Close the v0.7 GraphRAG milestone — graph-retrieval evaluation (a
graph-on/off A/B), a deterministic worked example, and the GraphRAG docs.
RAG-GRAPH-06.

## Delivered

- `eval/graph.go` — `RunGraphAB`: scores a `Dataset` twice through an
  `eval.Retriever` (once `EnableGraph` off, once on) and returns
  `GraphABResult{GraphOff, GraphOn Metrics; RecallDelta, MRRDelta}`. It
  reuses `eval.Evaluator` entirely — no forked metric code.
- `eval/graph_test.go` — a CI gate: `abStubRetriever` returns an extra
  gold chunk when `EnableGraph` is set; the test asserts `RunGraphAB` runs
  both arms, graph-on recall does not regress, and `RecallDelta > 0`. Plus
  a nil-retriever error test.
- `examples/graphrag_example_test.go` — `Example_graphRAG`: a deterministic
  end-to-end wiring — `DictionaryEntityExtractor` + a `GraphRetriever`-wired
  `HybridRetriever`, import, `Ask` with `EnableGraph: true` — with a
  stable `// Output:`. No live model.
- `docs/graphrag.md` — the GraphRAG usage doc: the three seams
  (`EntityExtractor`, `GraphStore`, `GraphRetriever`), the `EnableGraph`
  toggle, and the explicit v0.8 deferral list (community detection /
  summaries, global/DRIFT search, fuzzy entity resolution) + the
  "Neo4j is a future `GraphStore` impl" note.

## Fix found by the worked example

The `Example_graphRAG` end-to-end wiring exposed an integration gap:
`VariantRetriever` (the default outer retriever) rebuilds the retrieval
`Trace` from its sub-traces and was **not** carrying the new `Graph` field
through — so `Diagnostics.GraphTrace` came back empty even though
`HybridRetriever` populated it. Fixed: `VariantRetriever.Retrieve` now
propagates the `Graph` sub-trace (first non-empty wins, mirroring how it
already carries `RoutePolicy`). This is exactly the kind of fusion-plumbing
gap a worked example is meant to catch.

## Files

- `eval/graph.go`, `eval/graph_test.go` — new.
- `examples/graphrag_example_test.go` — new.
- `docs/graphrag.md` — new.
- `retrieve/retrieve.go` — `VariantRetriever` propagates `Trace.Graph`.

## Verification

All `<verify>` commands run, all green:

- `GOWORK=off go build ./...` — BUILD OK
- `GOWORK=off go vet ./...` — VET OK
- `GOWORK=off go test ./eval ./examples -count=1` — ok (`Example_graphRAG`
  output matches; `TestRunGraphAB` PASS)
- `GOWORK=off go test ./... -count=1` — all packages ok (no FAIL)
- `git diff --stat go.mod go.sum` — empty (no new dependency)
- core facade (from the core repo `llm-agent`): `GOWORK=off go vet
  ./rag/... && go test ./rag/...` — ok

## Notes

- The `eval` CI gate verifies the `RunGraphAB` harness mechanics with a
  deterministic stub retriever — faithful end-to-end graph retrieval is
  covered by `retrieve/graph_test.go` (22-01/02). This mirrors the Phase 16
  eval gate's use of stubs for what is not CI-verifiable.
- No new module dependency.

## Phase 22 status

All three slices complete. RAG-GRAPH-05 (`EntityLinker` + `GraphRetriever`
fused as a fourth RRF signal) and RAG-GRAPH-06 (graph A/B eval, worked
example, docs) are delivered. **Phase 22 completes the v0.7 GraphRAG
milestone** — Phases 20-22, RAG-GRAPH-01..06.

# Phase 10: Retrieval Policies, Hybrid Recall, and Context Packing - Research

**Researched:** 2026-05-14
**Status:** Seeded from the roadmap, Phase 9 ingestion work, and the current
standalone retrieval flow

## Locked Inputs

- current standalone retrieval is a single dense-vector path in `rag.Retrieve`
- MQE and HyDE already exist as model helpers in `advanced/llm.go`
- the best first step is to separate orchestration from dense retrieval logic
  before adding more retrieval modes

## Known High-Risk Areas

1. Baking dense retrieval assumptions too deeply into the new policy seam
2. Making query preprocessing mandatory instead of additive
3. Creating a policy abstraction that is too generic to be useful

## Research Conclusions

- `QueryPreprocessor` should return one or more query variants plus trace data
- `Retriever` should accept a retrieval request object rather than only raw
  query strings
- default `Retrieve(...)` should route through a dense retriever implementation
  using the existing embedder/store path

## Research Tasks Deferred Into Execution

- define the smallest useful request/trace structures
- keep default behavior backward-compatible while moving internals behind the
  new seams

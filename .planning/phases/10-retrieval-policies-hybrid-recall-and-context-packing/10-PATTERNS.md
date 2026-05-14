# Phase 10: Retrieval Policies, Hybrid Recall, and Context Packing - Pattern Map

**Mapped:** 2026-05-14

## Reuse From Prior Work

- additive-contract pattern from Phases 8 and 9
- standalone-first implementation pattern
- diagnostics and trace pattern already established in `rag.Answer`

## New Patterns to Add

### Query preprocessor pattern

- preprocessors can leave the query unchanged by default
- preprocessors may emit multiple query variants later
- trace output must record what changed

### Retriever policy pattern

- a retriever consumes a structured request
- dense retrieval is one implementation, not the contract itself
- later lexical and hybrid retrievers fit the same call path

### Default-policy bridge pattern

- public `System.Retrieve(...)` continues to work
- internally it delegates to the configured or default retriever


# Phase 12-02 Summary

Date: 2026-05-14
Repo: `llm-agent-rag`
Plan: [12-02-PLAN.md](/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/phases/12-persistence-tracing-and-backend-conformance/12-02-PLAN.md)

## Objective

Add a shared conformance suite under `store/storetest/` so every
`store.Store` implementation can be tested against the same contract
with one call. Wire the in-memory store (always-on) and the postgres
store (env-gated) into the suite.

Tracing-hook work moved to `12-03` so this slice stays focused.

## Delivered

- new package `store/storetest/`:
  - `Factory func(t *testing.T) store.Store` — fresh store per
    subtest
  - `Option`, `WithDimensionStrict()` for opt-in capability checks
  - `RunConformance(t, factory, opts...)` entry point
  - 12 conformance subtests covering:
    - Upsert and Get round-trip (all `StoredChunk` fields,
      `SectionPath`, `HeadingLevel`, metadata)
    - Search returns nearest first (cosine ranking)
    - Search respects namespace
    - Filter narrows results (metadata equality)
    - Security filter intersects with caller filter (AND
      semantics — conflicting filters yield zero hits;
      security-only narrows correctly)
    - List returns namespace chunks (with and without filters)
    - Get on missing returns `store.ErrNotFound`
    - Remove on missing returns `store.ErrNotFound`
    - Remove deletes (round-trip Upsert → Remove → Get fails)
    - RemoveByFilter returns count and removes matching chunks
    - Stats reports namespace count and dim
    - Dimension mismatch returns `store.ErrDimensionMismatch`
      (opt-in via `WithDimensionStrict()`)
- in-memory store wiring:
  - new `store/inmemory_conformance_test.go` with
    `TestInMemoryStoreConformance` — all 12 subtests green
  - existing bespoke tests in `store/inmemory_test.go` left
    untouched and still pass; conformance is additive
- postgres store wiring (env-gated):
  - new `postgres/postgres_conformance_test.go` with
    `TestPostgresStoreConformance`
  - factory creates a fresh table per subtest derived from
    `t.Name()` + random hex suffix, registered via `t.Cleanup` to
    `DROP TABLE` after each subtest
  - skips cleanly when `LLM_AGENT_RAG_PG_URL` is unset
  - existing 12-01 smoke tests (`TestPostgresStore_LiveSmoke`,
    `TestPostgresStore_LiveMetadataFilter`) preserved as the
    canonical AfterConnect / RegisterTypes reference

## Files

- `/tmp/llm-agent-rag/store/storetest/storetest.go` (new, ~250 LOC)
- `/tmp/llm-agent-rag/store/inmemory_conformance_test.go` (new, ~15 LOC)
- `/tmp/llm-agent-rag/postgres/postgres_conformance_test.go` (new, ~95 LOC)

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

- `go build ./...` (standalone): pass
- `go vet ./...` (standalone): pass
- `go test ./...` (standalone, 14 packages): pass — in-memory
  conformance runs all 12 subtests; postgres conformance skips
  cleanly without `LLM_AGENT_RAG_PG_URL`
- `go vet ./rag/...` / `go test ./rag/...` (core compat): pass

Live-Postgres conformance (when a local PG with pgvector is available):

```bash
cd /tmp/llm-agent-rag
LLM_AGENT_RAG_PG_URL=postgres://localhost/llm_agent_rag_test?sslmode=disable \
  GOWORK=off GOCACHE=/tmp/go-build go test ./postgres -count=1 -v \
  -run TestPostgresStoreConformance
```

## Notes

- the conformance subpackage follows the stdlib `httptest`/`fstest`/
  `iotest` convention — it imports `testing` so it cannot live in
  `store/` itself
- per-subtest isolation: in-memory creates a fresh `*InMemoryStore`
  per call; postgres creates a fresh table per call with a random
  suffix and drops it on cleanup. No shared state between subtests
  in either backend
- security-filter semantics were nailed down explicitly in the suite
  as **AND** (intersection), not override. This matches the
  in-memory store's existing behavior and the postgres
  implementation's `metadata @> $N AND metadata @> $M` shape from
  12-01
- subtest names use underscores instead of spaces or punctuation so
  `t.Name()` is directly usable as a postgres identifier with
  minimal sanitization
- the `WithDimensionStrict()` opt-in keeps the door open for future
  backends that don't enforce dim on Upsert (e.g. variable-dim
  pgvector via `vector` type without a fixed N) — they'll wire up
  without the option and the dimension-mismatch subtest skips

## Next slice

`12-03` covers tracing hooks for `Import`, `Retrieve`, `Pack`, `Ask`
(RAG-OPS-02). Today `Retrieve` and `Ask` already carry rich `Trace`
data; `Pack` has partial trace; `Import` has none. The slice will
add the missing pieces and define an OTel-friendly hook surface so
external tracing systems can attach without rewiring the rag facade.

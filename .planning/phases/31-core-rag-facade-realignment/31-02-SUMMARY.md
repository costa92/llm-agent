---
phase: 31-core-rag-facade-realignment
plan: 02
type: execute
wave: 2
status: complete
completed: 2026-05-19
repo: llm-agent
depends_on: ["31-01"]
requirements: [ECO-01]
files_modified:
  - rag/rag.go
  - rag/store.go
---

# Summary: 31-02 — repair the core `rag/` facade against `llm-agent-rag v1.0.0`

## Objective

Repair the core `llm-agent` `rag/` compatibility facade so it works against
`llm-agent-rag v1.0.0`: replace the `nil`-vector wide-search enumeration hack
in `storeAdapter.List` with a real `List` route. All facade tests pass; no new
dependency; the public `VectorStore` interface is unchanged. ECO-01.

## Root cause (from 31-01 diagnosis, confirmed)

`storeAdapter.List` (`rag/rag.go`) had no real enumerate primitive, so it
emulated one with `a.inner.Search(ctx, nil, stats.Count)`. A `nil` query
vector has length 0. `llm-agent-rag v1.0.0` `store/inmemory.go:56-57` hardened
`Search` to strictly reject any query vector whose length ≠ store dimension
(`if len(q.Vector) != s.dim { return nil, ErrDimensionMismatch }`). In
`v0.1.4` the store did not length-check, so the hack worked by accident.

31-01's key refinement: `storeAdapter.List` is reached on **every** search,
not just namespaced/filtered ones — the v1.0.0 default `HybridRetriever`
calls `r.Lexical.Retrieve` unconditionally, the facade's `storeAdapter` does
not implement the optional `store.LexicalSearcher`, so `LexicalRetriever`
falls through to `r.Store.List(...)` for every query. That is why all 7
facade tests failed, including the most trivial plain `New(Options{})` search.

## Delivered

Two facade files changed in the core repo, left **uncommitted** for the
operator. The fix is facade Go code only — no new import, no third-party
module, `go.mod`/`go.sum` untouched by this slice.

### `rag/store.go` — additive `ListDocuments` on `*InMemoryStore`

Added one additive method to the already-public concrete type
`*InMemoryStore`:

```go
func (s *InMemoryStore) ListDocuments(ctx context.Context) ([]Document, error)
```

It enumerates every stored document by delegating to the inner SDK store's
**real** `List` primitive — `s.inner.List(ctx, "", nil, nil)` (the SDK
`store.Store.List`, `inmemory.go:99`) — and maps the SDK `StoredChunk`s to
facade `Document`s. No similarity search. Namespace/filter scoping is
**deliberately not** delegated to the SDK `List`: the facade tracks namespace
in document metadata (`__rag_namespace` key), not the SDK `StoredChunk.Namespace`
field, so all scoping stays in the `storeAdapter`'s own metadata-based matching.

This is purely additive to an already-public type — it is **not** a new
`VectorStore` interface method — so the `contract_test.go` cross-repo gate
stays green.

### `rag/rag.go` — repaired `storeAdapter`

- **Optional `lister` capability interface** (unexported):
  `interface { ListDocuments(ctx) ([]Document, error) }`. `storeAdapter.List`
  type-asserts `a.inner` to it; the default `*InMemoryStore` satisfies it.
- **`idIndex` fallback** (unexported, mutex-guarded) for a custom
  `VectorStore` that does not implement the optional `lister`. The
  `storeAdapter` observes every chunk through `Upsert`/`Remove`/`RemoveByFilter`
  and maintains an id set. Because `storeAdapter` is passed **by value** into
  `ragcore.New`, the index is held behind a pointer (`*idIndex`) so
  `Upsert`/`Remove` mutations stay visible to `List`. A new `newStoreAdapter`
  constructor initialises the shared index; `New` now calls `newStoreAdapter(store)`.
- **`storeAdapter.enumerate`** — the single non-search enumeration helper:
  uses the optional `lister` when present, otherwise walks the id-index and
  fetches each document via `a.inner.Get`. A vanished id (store mutated
  outside the facade) is skipped and pruned from the index rather than
  failing the whole enumeration.
- **`storeAdapter.List`** rewritten to call `enumerate` then apply the
  facade's metadata-based namespace/filter/security matching. The
  `a.inner.Search(ctx, nil, stats.Count)` hack is **deleted**.
- **`storeAdapter.Upsert`** records each upserted id in the index.
- **`storeAdapter.Remove`** removes the id from the index after a successful
  inner remove.
- **`storeAdapter.RemoveByFilter`** now routes deletions through `a.Remove`
  (was calling `a.inner.Remove` directly) so the id-index stays consistent.

`storeAdapter.Search`'s namespace/filter/security branches were audited:
they set `pool = a.inner.Stats().Count` and feed `a.inner.Search(ctx, q.Vector, pool)`
— `q.Vector` there is the **real, non-nil** query vector, so those branches
are unaffected. The only `nil`-vector caller was `List`; it is now fixed.

## Verification

Every `<verify>` command run with `GOWORK=off` (core CI runs `GOWORK=off`).

- **`GOWORK=off go build ./...`** → `BUILD-OK`, no errors.
- **`GOWORK=off go vet ./...`** → `VET-OK`, clean.
- **`GOWORK=off go test ./rag/... -count=1`** → `ok github.com/costa92/llm-agent/rag` —
  all facade tests pass.
- **`GOWORK=off go test ./... -count=1`** → full suite green; every package
  `ok` (`llm-agent`, `bench`, `builtin`, `comm`, `comm/a2a`, `comm/anp`,
  `comm/mcp`, `context`, `llm`, `memory`, `orchestrate`, `pkg/fanout`, `rag`,
  `rl`). Zero failures.
- **No `nil`-vector search remains** —
  `! grep -n 'Search(ctx, nil' rag/rag.go && echo NO-NIL-SEARCH` → `NO-NIL-SEARCH`.
- **Public `VectorStore` interface unchanged** —
  `grep -A6 '^type VectorStore interface' rag/store.go` still shows exactly
  `Upsert` / `Search` / `Get` / `Remove` / `Stats`; no new method.
- **No new dependency** — `git diff go.mod go.sum` shows only the 31-01
  `llm-agent-rag v0.1.4 → v1.0.0` bump; 31-02 changed neither file.
- **`gofmt -l rag/`** → no output (formatting clean).
- **Cross-repo contract gate** — `TestContract_PublicFacade` **PASSES**; the
  facade's public surface is unchanged.

### The 7 previously-failing tests — all now pass

Run by name (`-run` over the exact 31-01 failing set):

| # | Test | 31-01 | 31-02 |
|---|------|-------|-------|
| 1 | `TestRAGSystem_WorksWithLLMEmbedderAdapter` | FAIL | PASS |
| 2 | `TestRAGSystem_AddAndSearch` | FAIL | PASS |
| 3 | `TestRAGSystem_AskHappyPath` | FAIL | PASS |
| 4 | `TestRAGSystem_SearchWithMQEMergesResults` | FAIL | PASS |
| 5 | `TestRAGTool_AddTextAndSearch` | FAIL | PASS |
| 6 | `TestRAGTool_Ask` | FAIL | PASS |
| 7 | `TestRAGTool_NamespaceIsolation` | FAIL | PASS |

Regression coverage per 31-01's recommendation: the plain-search lexical
fallback path is exercised by tests 1-6; the namespaced path by test 7
(`TestRAGTool_NamespaceIsolation`). MQE merge (test 4) also exercises the
lexical fallback for each generated sub-query. All retrieval paths that route
through the facade store's `List` now enumerate correctly.

## Deviations from plan

**None of substance.** The plan's recommended fix direction was followed
exactly: optional unexported `lister` interface + additive `ListDocuments` on
the already-public `*InMemoryStore`, with an id-index fallback for custom
`VectorStore`s. Two minor implementation details worth recording:

1. **`ListDocuments` does not delegate namespace/filter scoping to the SDK
   `List`.** The plan task 2 says "exposed in facade `Document` terms" via
   "the SDK `store.Store.List(ctx, namespace, filters, securityFilters)`".
   `ListDocuments` calls `s.inner.List(ctx, "", nil, nil)` — full enumeration,
   no SDK-side scoping — because the facade stores the namespace in document
   **metadata** (`__rag_namespace`), not in the SDK `StoredChunk.Namespace`
   field (`storeAdapter.Upsert` sets only metadata; `InMemoryStore.Upsert`
   leaves `Namespace` empty). SDK-side scoping on `Namespace`/`Filters`
   would silently drop every chunk. All scoping is correctly applied by
   `storeAdapter.List`'s existing metadata-based matching against the full
   enumerated set. This is the only way to keep facade namespace semantics
   correct and is consistent with how the old hack worked (it too scoped on
   metadata after the wide search).

2. **`storeAdapter` is now constructed via `newStoreAdapter`** rather than a
   struct literal, so the shared `*idIndex` is initialised. `RemoveByFilter`
   was changed to route through `a.Remove` (not `a.inner.Remove`) so the
   id-index fallback stays consistent. Both are internal to the unexported
   `storeAdapter`; no exported symbol changed.

No new dependency was added; `llm-agent-rag` was not changed (KE-3 honoured).
No git write commands were run — `rag/rag.go` and `rag/store.go` are modified
in the working tree and left uncommitted for the operator (alongside the
31-01 `go.mod`/`go.sum` changes).

## Out of scope (as planned)

- The `go.mod` bump — done in 31-01.
- The stdlib-only proof / verify gate / facade-doc updates — slice 31-03.
- Re-tagging the core — Phase 33.

## Self-Check: PASSED

- `rag/store.go` has the additive `ListDocuments` method on `*InMemoryStore`;
  `rag/rag.go` has the `lister` interface, `idIndex`, `newStoreAdapter`,
  `enumerate`, and the rewritten `List` — verified by reading the edited files.
- `! grep -n 'Search(ctx, nil' rag/rag.go` → `NO-NIL-SEARCH`; the hack is
  gone.
- `go build ./...` → `BUILD-OK`; `go vet ./...` → `VET-OK`;
  `go test ./... -count=1` → full suite green, zero failures.
- All 7 previously-failing facade tests pass when run by name.
- `TestContract_PublicFacade` passes — public surface unchanged.
- `git diff go.mod go.sum` shows only the 31-01 bump; 31-02 added no
  dependency. `gofmt -l rag/` is empty.
- No commits made — facade files left uncommitted for the operator per
  instruction.

---
phase: 31-core-rag-facade-realignment
plan: 01
type: execute
status: complete
completed: 2026-05-19
repo: llm-agent
requirements: [ECO-01]
---

# Summary: 31-01 — bump core to `llm-agent-rag v1.0.0` + diagnose the facade break

## Objective

Bump the core `llm-agent` module from `llm-agent-rag v0.1.4` to `v1.0.0`,
refresh `go.sum`, prove the core stays stdlib-only, reproduce the
`vector dimension mismatch` facade-test failures, and write a precise
diagnosis for slice 31-02. This slice does **not** fix the facade — it
pins down exactly what 31-02 must repair. ECO-01.

## Delivered

Two files changed in the core repo, left **uncommitted** for the operator:

- **`go.mod`** — `require github.com/costa92/llm-agent-rag` bumped
  `v0.1.4 => v1.0.0`. No other `require` added; the core still has exactly
  one third-party dependency (the sister-repo facade).
- **`go.sum`** — refreshed to the 2 `v1.0.0` lines only:
  ```
  github.com/costa92/llm-agent-rag v1.0.0 h1:58JlqUym3blPelaZNsn6cKPEybvJm9N1aJJTvV3g9xQ=
  github.com/costa92/llm-agent-rag v1.0.0/go.mod h1:m7+pFSGtENG1/cworYaIMhWeVnihzuve+GS5+XGpDqY=
  ```
  No `pgx`, no `pgvector`, no new third-party entries.

No facade source was modified (that is slice 31-02).

## Verification

Every `<verify>` command run with `GOWORK=off` (core CI runs `GOWORK=off`).
`GOPRIVATE=github.com/costa92/*,code.hellotalk.com` had to be supplied on
the `go get` invocation — see Deviations.

- **`go.mod` bumped** — `grep -q 'llm-agent-rag v1.0.0' go.mod` →
  `BUMP-OK`.
- **`go.sum` refreshed + minimal** — `grep -c 'llm-agent-rag' go.sum` → `2`;
  `! grep -E 'pgx|pgvector' go.sum` → `GOSUM-CLEAN`. Only the two
  `llm-agent-rag v1.0.0` lines; zero new third-party entries.
- **stdlib-only** —
  `go list -deps ./rag | grep -E '^github\.com/(jackc|pgvector)'` matched
  nothing → `STDLIB-ONLY-OK`. A wider scan (`grep -E '^[^/]+\.[^/]+/'`
  minus `github.com/costa92/` and `golang.org/`) returned **nothing**: the
  only external module in `./rag`'s dependency closure is
  `github.com/costa92/llm-agent-rag` itself. The `vendor/golang.org/x/*`
  entries are Go's stdlib-internal vendored packages, not module deps.
- **`go build ./...`** → `BUILD-OK`, no errors. The bump is a clean
  compile — the facade's *public surface* did not change; the break is
  purely behavioral.
- **`go test ./...`** → **7 failures, all in `github.com/costa92/llm-agent/rag`**,
  every other package `ok`. This is the expected diagnosis target, not a
  slice failure.
- **`rag/contract_test.go` cross-repo gate** — `TestContract_PublicFacade`
  **PASSES**. The facade's public surface still compiles against v1.0.0.
- **direct-store tests** — `TestInMemoryStore_*` (6 tests) all **PASS** —
  they call `InMemoryStore.Search` with correctly-sized vectors, so they
  never hit the broken path.

## Diagnosis

### Failing tests — exact list (7)

| # | File | Test | Error string |
|---|------|------|--------------|
| 1 | `rag/llm_embedder_test.go:48` | `TestRAGSystem_WorksWithLLMEmbedderAdapter` | `Search: rag: store search: rag: vector dimension mismatch` |
| 2 | `rag/rag_test.go:51` | `TestRAGSystem_AddAndSearch` | `Search: rag: store search: rag: vector dimension mismatch` |
| 3 | `rag/rag_test.go:94` | `TestRAGSystem_AskHappyPath` | `Ask: rag: vector dimension mismatch` |
| 4 | `rag/rag_test.go:149` | `TestRAGSystem_SearchWithMQEMergesResults` | `Search: rag: store search: rag: vector dimension mismatch` |
| 5 | `rag/tool_test.go:32` | `TestRAGTool_AddTextAndSearch` | `search: rag: store search: rag: vector dimension mismatch` |
| 6 | `rag/tool_test.go:47` | `TestRAGTool_Ask` | `ask: rag: vector dimension mismatch` |
| 7 | `rag/tool_test.go:114` | `TestRAGTool_NamespaceIsolation` | `search alpha: rag: store search: rag: vector dimension mismatch` |

All 7 are `RAGSystem.Search` / `RAGSystem.Ask` calls (the `tool_test`
ones go through `AsTool(r)`, which is a thin wrapper over the same
`RAGSystem`). None of them set a `RoutePath`; some set a `Namespace`
(test 7) or `EnableMQE` (test 4) — the failure is identical regardless,
which is itself diagnostic (see "Code path" below).

### Code path each failure reaches — CONFIRMED

The error is **not** raised on the dense-vector search branch. It is
raised on the **lexical** branch of the v1.0.0 SDK's default hybrid
retriever, which routes through the facade's `storeAdapter.List`:

```
RAGSystem.Search / .Ask
  → ragcore.System.Retrieve            (SDK rag/retrieve.go)
    → s.ret.Retrieve  =  VariantRetriever{ Base: HybridRetriever{...} }
      → HybridRetriever.Retrieve       (SDK retrieve/retrieve.go:983)
        → r.Dense.Retrieve   → DenseRetriever → storeAdapter.Search  ... OK
        → r.Lexical.Retrieve UNCONDITIONALLY  (retrieve.go:990 — not gated)
          → LexicalRetriever.Retrieve  (retrieve/retrieve.go ~720)
            → store NOT a store.LexicalSearcher  → FALLS THROUGH to:
            → r.Store.List(ctx, ns, filters, secFilters)   (retrieve.go:761)
              → facade storeAdapter.List   (core rag/rag.go:313)
                → a.inner.Search(ctx, nil, stats.Count)   (rag/rag.go:319)  ← THE HACK
                  → InMemoryStore.Search   (core rag/store.go:69)
                    → ragstore.InMemoryStore.Search   (SDK store/inmemory.go:55)
                      → len(q.Vector)=0  !=  s.dim=32   → ErrDimensionMismatch
                                                          (SDK store/inmemory.go:56-57)
            ← mapStoreErr maps store.ErrDimensionMismatch → rag.ErrDimMismatch
              ("rag: vector dimension mismatch", core rag/store.go:121-122)
```

`storeAdapter.Search`'s namespace/filter branch (`pool = a.inner.Stats().Count`,
`rag/rag.go:269-280`) is **not** the trigger — that branch still forwards
the real, correctly-sized `q.Vector`. The trigger is `storeAdapter.List`'s
`nil`-vector wide search.

### Root cause — CONFIRMED against the real bump

`31-RESEARCH.md`'s root cause is **confirmed**, with one important
refinement.

- **The hack.** `storeAdapter.List` (`rag/rag.go:313-343`) has no real
  enumerate primitive, so it emulates one with
  `a.inner.Search(ctx, nil, stats.Count)`. A `nil` query vector is
  length 0.
- **The SDK hardening.** `llm-agent-rag v1.0.0` `store/inmemory.go:56-57`:
  `if len(q.Vector) != s.dim { return nil, ErrDimensionMismatch }`.
  v1.0.0's `Search` strictly length-checks the query vector. In `v0.1.4`
  the store did not, so a length-0 vector silently scored against every
  chunk and the hack worked by accident.
- **`Upsert` is fine.** v1.0.0 length-checks `Upsert` too
  (`inmemory.go:46-47`), but the facade always upserts a real
  `embedder`-sized vector, so ingestion is unaffected — all `AddText`
  calls in the failing tests succeed; only the subsequent search fails.

**Refinement vs. research (this is the second-order finding — read it):**
`31-RESEARCH.md` framed `storeAdapter.List` as reached only on
"namespace isolation, filtered search, structure-aware retrieval". The
real v1.0.0 bump shows it is reached on **every single search**, with no
namespace and no filters. The v1.0.0 default retriever is
`VariantRetriever{ Base: HybridRetriever{ Dense, Lexical, Structure } }`,
and `HybridRetriever.Retrieve` calls `r.Lexical.Retrieve`
**unconditionally** (not behind `EnableStructure` / any flag —
`retrieve/retrieve.go:990`). `LexicalRetriever.Retrieve` first probes the
store for the optional `store.LexicalSearcher` capability; the facade's
`storeAdapter` does **not** implement it, so the lexical retriever falls
through to `r.Store.List(...)` for **every** query. That is why even the
most trivial test — `TestRAGSystem_AddAndSearch`, plain `New(Options{})`,
no namespace, no filter — fails. The dimension-mismatch class is
correct; its blast radius is *all searches*, not just the
namespace/filter subset.

This is a **facade bug, not an SDK bug.** v1.0.0's strict length-check is
a correct hardening; the facade's `nil`-vector enumeration hack was
always relying on its absence.

### Second-order-break check — EXPLICIT

**One fix class, but it must cover three SDK call sites — not just `List`.**

- **All 7 failures are the same class** — the `nil`-vector
  `storeAdapter.List` hack hitting v1.0.0's strict length-check. No
  failure is explained by a *different* v0.1.4→v1.0.0 behavior change.
  Repairing `storeAdapter.List` so it enumerates **without** a
  `nil`-vector `Search` will clear all 7.
- **No signature break.** `go build ./...` is green; `embed.HashEmbedder`,
  `embed.NewHashEmbedder`, `store.NewInMemoryStore` are all
  signature-compatible. The break is purely behavioral.
- **The contract gate is unaffected** — `TestContract_PublicFacade`
  passes; the facade's public surface needs no change.
- **Second-order risk for 31-02 to keep in scope:** the v1.0.0 SDK calls
  the facade store's `List` from **four** retrieval sites, not one —
  `retrieve/retrieve.go:613` (`DenseRetriever.retrieveWithinRoute`),
  `:761` (`LexicalRetriever` fallback — the one all 7 tests hit), `:813`
  (`StructureRetriever.Retrieve`), and `:1198`. It also calls `List` from
  `rag/import.go:54` (re-ingest staleness check). Once
  `storeAdapter.List` is given a correct (non-`nil`-vector) enumeration,
  every one of these sites is fixed at once — but 31-02's regression test
  must exercise at least the lexical path (covered today), a
  namespaced/filtered search (test 7), and ideally `EnableStructure` to
  prove the `StructureRetriever.List` site too. No *hidden* second-order
  break was found: the strict length-check is the only relevant
  v0.1.4→v1.0.0 behavior change reaching the facade.

### Recommended fix direction for 31-02

Per `31-RESEARCH.md` Decision 1, adjusted for what 31-01 confirmed:

1. **Give the facade `*InMemoryStore` a real enumerate method.** The
   facade `InMemoryStore` (`rag/store.go`) wraps `*ragstore.InMemoryStore`,
   which in v1.0.0 has a **real `List(ctx, namespace, filters,
   securityFilters)`** (`SDK store/inmemory.go:99`). Add an additive
   method on the facade `*InMemoryStore` that delegates to the inner
   SDK store's real `List` — no `nil`-vector search. This is purely
   additive to an already-public type, so the `contract_test.go` gate
   stays green.

2. **Define an optional `lister` capability interface in `rag/`** (e.g.
   `interface { List(ctx, namespace) ([]Document, error) }`) and have
   `storeAdapter.List` type-assert `a.inner` to it. When present (the
   default `InMemoryStore`), enumerate via the real `List`; the
   `nil`-vector `Search` call at `rag/rag.go:319` is deleted.

3. **Fallback for a custom `VectorStore`** that does not implement the
   optional `lister`: `storeAdapter` already observes every chunk through
   `Upsert` / `Remove`, so it can maintain its own id-index and enumerate
   via `a.inner.Get(id)`. This keeps namespace/filter/lexical/structure
   retrieval working for **any** `VectorStore` with **no** change to the
   public `VectorStore` interface — so no breaking change for external
   implementers.

4. **Do not** add `List`/enumerate to the public `VectorStore` interface
   — that breaks external implementers and is avoidable via the
   optional-interface escape (research Decision 1, "Not chosen").

5. **Optional, lower priority:** the facade *could* additionally
   implement `store.LexicalSearcher` on `storeAdapter` so the lexical
   retriever takes its fast path instead of the `List` fallback — but
   that is an enhancement, not required for the fix. Fixing
   `storeAdapter.List` alone clears all 7 failures because the lexical
   fallback then enumerates correctly.

6. **31-02 regression coverage** must include: a plain search (lexical
   fallback path — all 7 current tests), a namespaced search (test 7
   path), and an `EnableStructure` search (to exercise the
   `StructureRetriever` `List` site at `retrieve.go:813`).

## Notes / deviations

- **Deviation — `GOPRIVATE` was not exported in the shell.** The plan
  states `GOPRIVATE=github.com/costa92/*` is "already set on this
  machine". `go env` showed `GOPRIVATE=code.hellotalk.com` only — it did
  **not** include `github.com/costa92/*`. The first
  `go get llm-agent-rag@v1.0.0` failed against `sum.golang.org`
  (`404 Not Found`) and then against `https://github.com` (`terminal
  prompts disabled`). The `git config url."git@github.com:".insteadOf`
  rewrite *was* present. Resolved by supplying
  `GOPRIVATE=github.com/costa92/*,code.hellotalk.com` on the `go get`
  invocation itself — this routes the fetch through SSH and skips the
  sum-database lookup. `go.sum` is still fully populated with real
  hashes. **Action for the operator / 31-03:** persist
  `GOPRIVATE=github.com/costa92/*` into the machine's `go env` (or the CI
  env) so the bump is reproducible without per-command overrides.
- **No facade fix applied** — out of scope; that is slice 31-02. The 7
  failing tests are the intended diagnosis output of this slice, not a
  slice failure.
- **No git write commands run.** `go.mod` and `go.sum` are modified in
  the working tree and left uncommitted for the operator
  (`git status --short` shows ` M go.mod`, ` M go.sum`).
- **Out of scope, as planned:** the facade fix (31-02); the verify gate
  and facade-doc updates (31-03); re-tagging the core (Phase 33).

## Self-Check: PASSED

- `go.mod` requires `github.com/costa92/llm-agent-rag v1.0.0`;
  `go.sum` has exactly 2 `llm-agent-rag v1.0.0` lines and no other
  third-party entries — verified by `grep`.
- `go build ./...` → `BUILD-OK`; `go list -deps ./rag` → `STDLIB-ONLY-OK`
  (no `jackc`/`pgvector`, no external module other than
  `github.com/costa92/*`).
- `go test ./...` reproduces exactly 7 facade-test failures, all in
  `rag`, all `vector dimension mismatch`; every other package `ok`.
- The diagnosis above lists each failing test (file + name), the exact
  facade→SDK code path, the confirmed `nil`-vector root cause, an
  explicit second-order-break check, and the 31-02 fix direction.
- No commits made — `go.mod` / `go.sum` left uncommitted for the operator
  per instruction.

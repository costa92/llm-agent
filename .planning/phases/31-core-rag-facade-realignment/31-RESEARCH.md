> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent-rag` current docs.

# Phase 31 Research: Core RAG facade re-alignment to `llm-agent-rag v1.0.0`

**Researched:** 2026-05-21
**Phase:** 31 — core RAG facade re-alignment (first v1.1 phase)
**Requirements:** ECO-01
**Repos:** `llm-agent` (core — THIS repo)
**Upstream:** `.planning/research/v1.1-ecosystem-alignment-SUMMARY.md` §1;
keystone KE-3.

## Phase goal

The core `llm-agent` module builds and tests green against
`llm-agent-rag v1.0.0` (currently pinned at `v0.1.4` — 8 minors + a major
stale), the `rag/` compatibility facade behaves identically to callers,
and the core stays **provably stdlib-only**.

## Current state (codebase scan — core repo `/home/hellotalk/code/go/src/github.com/costa92/llm-agent`)

- `go.mod`: `require github.com/costa92/llm-agent-rag v0.1.4`; `go.sum`
  pins only `v0.1.4` (2 lines). No other third-party require — the core
  is stdlib-only apart from this one sister-repo facade dependency.
- `rag/` is the compatibility facade — `rag.go` (the `RAGSystem` + the
  `splitterAdapter`/`embedderAdapter`/`storeAdapter`/`modelAdapter` that
  bridge to the SDK's `rag.System`), `store.go` (the `VectorStore`
  interface + `InMemoryStore` wrapper over `ragstore.InMemoryStore`),
  `embedder.go` (`Embedder` + `HashEmbedder` over `ragembed.HashEmbedder`),
  `chunk.go`, `tool.go`, `advanced.go`, `contract_test.go`.
- `rag/contract_test.go` — the core side of the cross-repo contract gate;
  pins the facade's public surface. Its mirror is
  `llm-agent-rag/contract/contract_test.go`.
- Today the facade builds and tests **green against the stale `v0.1.4`**.

## Root cause — CONFIRMED (not preliminary)

The v1.1 SUMMARY flagged 7 facade-test `rag: vector dimension mismatch`
failures on a trial bump to `v1.0.0` with a *preliminary* root cause. This
phase research **confirmed it** by reading the v1.0.0 SDK source:

- `llm-agent-rag` `store/inmemory.go` (v1.0.0), `InMemoryStore.Search`,
  **lines 56-57**: `if len(q.Vector) != s.dim { return nil,
  ErrDimensionMismatch }`. v1.0.0's store **strictly rejects any query
  vector whose length ≠ the store dimension.**
- The core facade's `storeAdapter.List` (`rag/rag.go`) emulates a
  list/enumerate primitive with a **`nil`-vector wide search**:
  `a.inner.Search(ctx, nil, stats.Count)`. A `nil` query vector has
  length 0; under v1.0.0 that is `0 != 32` → `ErrDimensionMismatch` →
  the facade maps it (`mapStoreErr`) to `ErrDimMismatch`
  (`"rag: vector dimension mismatch"`).
- The same hazard exists in `storeAdapter.Search`'s namespace/filter
  branch and anywhere the SDK's v1.0.0 retrieval path calls the facade
  store's `List` (namespace isolation, filtered search, structure-aware
  retrieval). In `v0.1.4` the SDK store did not strictly length-check the
  query vector, so the `nil`-vector hack silently worked. v1.0.0 made the
  check strict — a correct SDK hardening that the facade's emulation hack
  was relying on the absence of.

**This is a facade bug, not an SDK bug.** Emulating enumeration with a
`nil`-vector similarity search was always a hack; v1.0.0 merely stopped
tolerating it. The fix belongs entirely in the core `rag/` facade.

## The v1.0.0 SDK surface the facade must align to

- `store.Store` (v1.0.0) is a 7-method interface — `Upsert`, `Search`,
  **`List(ctx, namespace, filters, securityFilters)`**, `Get`, `Remove`,
  `RemoveByFilter`, `Stats`. It already has a **real `List`** — the
  facade does not need to emulate one; it needs to *route to a real one*.
- `embed.HashEmbedder` / `embed.NewHashEmbedder(dim)` / `Dimension()` /
  `Embed(ctx, text) (Vector, error)` — signature-compatible with what the
  facade uses (`dim <= 0` → 32). No embed-side break.
- `store.NewInMemoryStore(dim)` — signature-compatible (`dim <= 0` → 32).

The break is **behavioral and localized** to the `nil`-vector enumeration
hack — not a signature break across the facade.

## Decision 1 — the fix routes `List` to a real enumeration (KE-3)

The facade's public `VectorStore` interface (`rag/store.go`) is
`Upsert`/`Search`/`Get`/`Remove`/`Stats` — it has **no enumerate
primitive**, which is why `storeAdapter.List` resorted to the `nil`-vector
hack. The fix must give `storeAdapter.List` a real source of chunks
**without** the `nil`-vector search and **without** adding a dependency
(KE-3 — never fix by adding a dep).

Recommended direction (31-02 confirms against 31-01's findings):

- **An optional `lister` interface + a real `List` on the default
  `InMemoryStore`.** The facade's `InMemoryStore` wraps
  `ragstore.InMemoryStore`, which *has* the SDK's real `List`. Expose it:
  give the facade `*InMemoryStore` a method that enumerates via the inner
  SDK store's real `List` (or `Stats`-bounded id walk), and have
  `storeAdapter.List` **type-assert** `a.inner` to that optional
  interface and use it when present.
- **Fallback for a custom `VectorStore`** that does not implement the
  optional interface: `storeAdapter` already sees every chunk through
  `Upsert`/`Remove` — it can maintain its own id index and enumerate via
  `a.inner.Get(id)`. This keeps namespace/filter ops working for *any*
  `VectorStore` with **no** change to the public `VectorStore` interface
  (so no breaking change to external facade implementers).
- The `nil`-vector call sites (`storeAdapter.List`, and the
  `pool = Stats().Count` namespace/filter branches of `storeAdapter.Search`
  that feed it) are the surface to repair. 31-01 enumerates the exact set.

**Not chosen:** adding `List`/enumerate to the public `VectorStore`
interface — that is a breaking change for any external `VectorStore`
implementer and is avoidable via the optional-interface escape.

## Decision 2 — stdlib-only is a phase exit gate (KE-3)

The core `llm-agent` is stdlib-only apart from the one `llm-agent-rag`
facade dependency. The v1.1 SUMMARY verified that bumping to `v1.0.0`
keeps it so — `go list -deps ./rag` pulls **no** `pgx`/`pgvector` (the
facade imports only the stdlib-only rag subpackages: `rag`, `embed`,
`store`, `ingest`, `advanced`, `generate`, `prompt`). Phase 31's exit
gate **re-proves** this: `go list -deps ./rag` lists zero third-party
modules and `go.sum` contains only the `llm-agent-rag` lines.

## Slice breakdown

- **31-01** — bump `go.mod` to `llm-agent-rag v1.0.0`, refresh `go.sum`,
  reproduce the failures; write a precise diagnosis: confirm the
  `nil`-vector `storeAdapter.List` root cause, enumerate the exact failing
  tests + the code path each reaches, and check whether repairing the
  `nil`-search reveals any *second-order* mismatch. No facade fix yet.
  (ECO-01)
- **31-02** — apply the facade fix: replace the `nil`-vector enumeration
  hack with a real `List` route (the optional-`lister` interface +
  fallback id-index, per Decision 1); all 7 failing facade tests pass.
  (ECO-01)
- **31-03** — verify gate: `go vet ./... && go test ./...` green;
  `go list -deps ./rag` proves zero third-party modules; `go.sum`
  minimal; `rag/contract_test.go` compiles; update the facade docs that
  name the old SDK version. (ECO-01)

## Risks / notes

- The core stays stdlib-only — the fix adds **no dependency**; it is
  facade Go code only. `go.sum` keeps only the `llm-agent-rag` lines.
- 31-01 must check for second-order breaks: once the `nil`-search is
  fixed, a different v0.1.4→v1.0.0 behavior change could surface. The
  v1.1 SUMMARY's trial run saw only the dimension-mismatch class, but
  31-01 confirms against the real bump.
- The `rag/contract_test.go` cross-repo gate must still compile after the
  bump — the facade's *public* surface does not change in this phase (the
  fix is internal: `storeAdapter` is unexported; the optional `lister`
  may be a new *additive* method on the already-public `*InMemoryStore`).
- Phase 31 does **not** re-tag the core — tagging is Phase 33 (`v0.5.0`).
  31 leaves the core on `main` with the bump applied and green.
- `go get llm-agent-rag@v1.0.0` fetches over SSH (the
  `git config url.insteadOf` + `GOPRIVATE` rewrite is set on this machine).

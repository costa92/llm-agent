---
phase: 29-documentation-completeness-and-compatibility-policy
plan: 03
type: execute
status: complete
completed: 2026-05-19
repo: llm-agent-rag
requirements: [RAG-API-04]
---

> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent-rag` current docs.

# Summary: 29-03 — docs/compatibility.md + README v1.0 status line

## Objective

Write `docs/compatibility.md` — the Go-module compatibility promise
`llm-agent-rag` commits to at `v1.0.0` — and update the `README.md`
status line to present the SDK as stable v1.0, linking the policy.
Documentation-only: no Go code change, no new module dependency.
RAG-API-04.

## Delivered

### 1. `docs/compatibility.md` (new file)

A new policy document with one section per required topic:

- **Import compatibility** — states the additive-only `v1.x` rule (no
  removed/renamed exported symbols, no signature changes, no removed
  struct fields, no changed exported constants callers depend on);
  enumerates what a `v1.x` release *may* do (new funcs/types/packages,
  new struct fields).
- **The interface-method gotcha** — called out explicitly as its own
  sub-section: adding a method to an exported interface breaks every
  external implementer and is therefore a breaking change. Names the
  frozen `v1.x` interfaces (`store.Store`, `embed.Embedder`,
  `generate.Model`, `retrieve.Retriever`, `rerank.Reranker`,
  `ingest.Splitter`, `ingest.Source`, `prompt.Template`) and the
  optional-interface escape pattern (`store.LexicalSearcher`).
- **Semantic versioning** — `v1.MINOR.PATCH` as Go's module system
  enforces it; PATCH = fixes, MINOR = additions, MAJOR = breaking.
- **Breaking changes and `/v2`** — the only mechanism is a new import
  path `github.com/costa92/llm-agent-rag/v2`; there is no other.
- **The `contract` sub-contract** — `contract/contract_test.go` pins the
  exact surface the core `llm-agent/rag` facade consumes; changes only
  via coordinated PRs with `llm-agent`. Links `core-compatibility.md`
  for the full cross-repo story (does not duplicate it).
- **External dependencies (`postgres`)** — `pgx`/`pgvector-go` are the
  only non-stdlib deps, isolated in the `postgres` package; `go.mod`
  declares minimum versions; external major bumps surface via `go.mod`
  and are never hidden in a patch release.
- **`adapter/llmagent` coverage** — the build-tagged adapter is covered
  by the same additive-only promise; a build tag controls *when* a
  package compiles, not whether its API is stable.
- **`go.sum`** — committed, and correct for this module (it has the
  postgres-island deps); noted as an intentional difference from the
  stdlib-only core `llm-agent`.
- **Minimum Go version** — `go 1.26` is the v1.0 floor; minor-Go bumps
  within `v1.x` are permitted (treated as backward-compatible).
- **Deprecation procedure** — `// Deprecated:` in a `v1.x` minor;
  removal only in a future `/v2`, never within `v1.x`.

A scope note at the top links `docs/core-compatibility.md` for the
cross-repo story, so this document covers `llm-agent-rag`'s own API
only and does not duplicate the sister-repo material.

### 2. `README.md` status line

The single line at ~48 — `Current status: production-ready core,
evolving ecosystem.` — was replaced with a stable-v1.0 status that
states the public API is frozen under an additive-only `v1.x` promise
and links `docs/compatibility.md`. No other part of the README was
touched (the "Not implemented yet" / "Implemented" list was already
corrected in slice 28-03 and was left as-is).

## Files

`/tmp/llm-agent-rag`:

- `docs/compatibility.md` — new file (the v1.0 compatibility promise).
- `README.md` — status line only (1 line replaced with 4 lines).

`go.mod` / `go.sum` — unchanged. No Go code changed.

## Verification

All `<verify>` commands run in `/tmp/llm-agent-rag`; go commands with
`GOWORK=off GOCACHE=/tmp/go-build`:

- `test -f docs/compatibility.md` → **`DOC-OK`**.
- Section coverage — `for s in 'import' 'ersion' '/v2' 'contract'
  'postgres' 'llmagent' 'go.sum' 'Deprecat'; do grep -qi …; done` →
  **no `MISSING-SECTION` lines** (all 8 required sections present).
- Interface gotcha — `grep -qi 'interface' docs/compatibility.md` →
  **`INTERFACE-OK`**.
- README links the policy — `grep -q 'compatibility.md' README.md` →
  **`README-LINK-OK`**.
- Old status removed — `! grep -q 'production-ready core, evolving
  ecosystem' README.md` → **passes** (old text gone).
- `go vet ./...` → **`VET-OK`**, no errors.
- `go test ./... -count=1` → every package `ok`, no `FAIL`. 3 packages
  report `[no test files]` (root `ragkit`, `generate`,
  `store/storetest`), unchanged. `contract` package `ok` — confirms no
  exported symbol moved.
- `git diff --stat go.mod go.sum` → **empty**. No new module dependency.

## Notes / deviations

- **No deviations.** The plan executed exactly as written: one new
  markdown file plus a one-line README status change.
- **No Go code changed** — documentation-only slice; `go vet` and the
  full `go test ./...` (including the `contract` gate) stay green,
  confirming the build is untouched.
- **No new module dependency** — `git diff --stat go.mod go.sum` empty.
- `docs/core-compatibility.md` is **linked, not duplicated** — the
  scope note and the `contract`/`go.sum` sections reference it for the
  cross-repo story.
- All `go` commands ran with `GOWORK=off GOCACHE=/tmp/go-build` per the
  verify block.
- **No git write commands were run.** The new `docs/compatibility.md`
  and the edited `README.md` are left uncommitted in the working tree
  for the operator to commit separately, alongside the untouched
  Phase-28 changes and slices 29-01/29-02's doc comments.
- Out of scope, as planned and confirmed untouched: the README
  "Not implemented yet" list (fixed in 28-03), package/symbol doc
  comments (29-01/29-02), the `CHANGELOG.md` `[v1.0.0]` entry (30-03),
  and the API-snapshot gate (Phase 30).

## Self-Check: PASSED

- `docs/compatibility.md` present in the working tree (`test -f` →
  `DOC-OK`); all 8 required sections + the interface-method gotcha
  verified by grep.
- `README.md` links `docs/compatibility.md` and no longer carries the
  old status text — both verified by grep.
- `go vet ./...` and `go test ./... -count=1` (incl. `contract`) green.
- `git diff --stat go.mod go.sum` empty — no new dependency, no Go code
  change.
- No commits made — per operator instruction, changes left uncommitted
  for a separate commit.

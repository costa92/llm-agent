# Phase 29 Research: Documentation completeness & the compatibility policy

**Researched:** 2026-05-20
**Phase:** 29 — documentation completeness & the compatibility policy
**Requirements:** RAG-API-03, RAG-API-04
**Repos:** `llm-agent-rag`
**Upstream:** `.planning/research/v1.0-api-stabilization-SUMMARY.md` §3, §4;
keystones KS-5, KS-7. Phase 28: `docs/api-audit-v1.0.md` (the symbol
inventory this phase documents).

## Phase goal

Bring every package and every exported symbol of `llm-agent-rag` up to the
v1.0 documentation bar, and write down the Go-module compatibility promise.
KS-7: the doc bar is **release-blocking** — a v1.0 with blank `pkg.go.dev`
pages does not ship. No Go API surface changes in this phase — only doc
comments, one new `docs/` file, and the README status line.

## Current state (codebase scan, `/tmp/llm-agent-rag` @ `v0.6.0` + Phase 28)

### Package doc-comment coverage (verified — `go doc ./pkg` line 3)

**Has a package comment (9):** `agentic`, `eval`, `graph`, `guard`, `obs`,
`feedback`, `postgres`, `store/storetest`, and the root `ragkit` (`doc.go`,
rewritten in 28-02).

**MISSING a package comment — 11 importable packages:** `advanced`,
`embed`, `generate`, `ingest`, `pack`, `prompt`, `rag`, `rerank`,
`retrieve`, `store`, `tree`.

**Also missing — special cases (3):**
- `adapter/llmagent` — build-tagged (`//go:build llmagent`); `model.go` +
  `tool.go` have no package comment.
- `contract` — test-only package (only `contract_test.go`); no comment.
- `examples` — test-only worked-examples package (only `*_test.go`); no
  comment.

So **14 package comments** must be added. `rag` (the front door) is the
highest priority — it currently renders blank on `pkg.go.dev`.

### Package-comment placement convention (observed)

Every existing package comment lives on the file named `<pkg>.go`
(`eval/eval.go`, `graph/graph.go`, `obs/obs.go`, `feedback/feedback.go`,
`postgres/postgres.go`) — or, when there is no `<pkg>.go`, on a
representative primary file (`guard/redact.go`, `agentic/correct.go`). The
root uses a dedicated `doc.go`. **Phase 29 follows this convention** — no
new `doc.go` files except where a package has no natural anchor.

- `store` has `store/store.go` → the comment goes there.
- `rag` has **no** `rag.go`; its central type `System` + `New` live in
  `rag/system.go` → the package comment goes on `rag/system.go`.
- The other 9 importable packages have a `<pkg>.go` or an obvious primary
  file — the executor places the comment per the convention.
- `adapter/llmagent` — the comment goes above `package llmagent` in
  `model.go` (after the `//go:build` line — Go allows the doc comment
  between the build constraint's blank line and the `package` clause).
- `contract` / `examples` — a one-line `// Package …` comment above the
  `package` clause in the existing `_test.go` file.

### Existing documentation assets

- `README.md` — 188 lines; status line (~48) `Current status:
  production-ready core, evolving ecosystem.`; the stale "Not implemented
  yet" claims were already corrected in slice 28-03.
- `docs/` — `backend-selection.md`, `core-compatibility.md`,
  `graphrag.md`, `production-deployment.md`, and `api-audit-v1.0.md` (new,
  Phase 28). **`docs/compatibility.md` does not exist** — Phase 29 creates it.
- `docs/core-compatibility.md` already covers the *cross-repo* story —
  `docs/compatibility.md` links it rather than duplicating it.
- 5 worked `examples/*_test.go` — one per primary answer path (`Ask`,
  `AskGlobal`, `AskDrift`, graph, path). They compile + run as `go test`.

## Decision 1 — package comments (slice 29-01, RAG-API-03)

`29-01` adds all 14 package comments. Each comment: starts with
`// Package <name>`, states the package's role in the SDK in 1-4 lines, and
(for seam packages) names the central interface(s). Content is sourced from
`docs/api-audit-v1.0.md` (the Phase-28 inventory already describes every
package's role). No symbol changes — package clauses only.

## Decision 2 — exported-symbol doc sweep (slice 29-02, RAG-API-03)

`29-02` fills every gap so **every exported type, func, method, `Err*`
value, and const** has a doc comment that starts with the symbol name (Go
convention — `go doc` and `pkg.go.dev` rely on the name prefix). The
Phase-28 `docs/api-audit-v1.0.md` inventory is the **checklist**: it
enumerates every exported symbol of every package. The sweep is
package-by-package; `go doc <pkg>` shows each symbol with its current
first comment line (or nothing) — bare symbols are the gaps.

The stdlib has no missing-doc linter (`golint` is deprecated and would be a
new tool — rejected, KS-8). The gate is therefore: the executor walks every
package against the audit-doc checklist, and the slice's verify spot-checks
a sample of previously-bare symbols via `go doc <pkg>.<Symbol>`. The
executor records "every package swept" in the SUMMARY. `gofmt`/`go vet`
stay green throughout.

Scope discipline (KS-2): this is a **comment-only** sweep. No rename, no
signature change, no symbol added or removed — the surface was frozen by
Phase 28. If documenting a symbol reveals it is misnamed, that is recorded
as a `/v2` note in the SUMMARY, **not** fixed here.

**Freeze-hygiene addendum (found during slice 29-01 execution).** Slice
29-01 surfaced that `llm-agent-rag` was never fully `gofmt`-clean: ~10
files at `v0.6.0` carry pre-existing trivial non-compliance — comment-column
and struct-field alignment, import ordering within a group, one
trailing-whitespace line. Phase 28's edited files (`eval/*.go`, `doc.go`)
are clean — this is genuine pre-`v1.0` debt, not regression. A v1.0
*stable* release must ship `gofmt`-clean, and 29-02's own `gofmt -l .`
verify cannot pass while the debt exists. Decision: 29-02 folds in a
repo-wide `gofmt -w .` as explicit freeze hygiene (task 5) — whitespace-only,
zero-risk, no token/symbol/behavior change. This is a planner-level scope
correction recorded here so it is a deliberate, documented decision.

## Decision 3 — `docs/compatibility.md` + README status (slice 29-03, RAG-API-04)

`29-03` writes `docs/compatibility.md` (KS-5). Contents:
- The Go **import-compatibility rule** — additive-only within `v1.x`: no
  removing/renaming exported symbols, no signature changes, no struct-field
  removals; **the interface-method gotcha called out explicitly** (adding a
  method to an exported interface breaks every external implementer).
- **Semver** as Go enforces it (`v1.MINOR.PATCH`).
- The **`/v2` import-path rule** — a breaking change requires
  `github.com/costa92/llm-agent-rag/v2`; there is no other path.
- The **`contract`-package sub-contract** — the cross-repo surface the
  core `llm-agent/rag` facade pins; links `docs/core-compatibility.md`.
- The **`postgres` external-dependency policy** — `pgx`/`pgvector-go`
  minimum-version policy; their majors surface via `go.mod`.
- The **`adapter/llmagent` coverage** — build tags do not exempt a package
  from the promise.
- The **`go.sum`-committed note** — correct for this module (it has the
  postgres-island deps); intentionally differs from the stdlib-only core
  `llm-agent`.
- The **minimum Go version floor** — `go 1.26` is the v1.0 floor;
  minor-Go bumps are allowed within `v1.x`.
- The **deprecation procedure** — `// Deprecated:` in a `v1.x` minor,
  removal only in a future `/v2`.

`29-03` also updates the `README.md` status line (~48) from
"production-ready core, evolving ecosystem" to a **"stable — v1.0"**
status, with a link to `docs/compatibility.md`. (The stale "Not
implemented yet" list was already fixed in 28-03 — 29-03 does not re-touch
it.)

## Slice breakdown

- **29-01** — add 14 package doc comments (11 importable packages +
  `adapter/llmagent` + the test-only `contract`/`examples`), placed per the
  observed convention. (RAG-API-03)
- **29-02** — exported-symbol doc-comment sweep across every package,
  checklisted against `docs/api-audit-v1.0.md`; every exported symbol gets
  a name-prefixed comment. (RAG-API-03)
- **29-03** — write `docs/compatibility.md` (KS-5); update the `README.md`
  status line + link the policy. (RAG-API-04)

## Risks / notes

- No new module dependency — doc comments and a markdown file only.
  `git diff --stat go.mod go.sum` must stay empty.
- Comment-only changes cannot break the build, but can break **examples**
  if a doc comment is accidentally placed mid-declaration — every slice
  runs the full `go test ./...` to catch that.
- 29-02 is the largest slice by edit count (every package) but the lowest
  risk per edit. It must not drift into renaming — Phase 28 froze the
  surface; 29-02 documents it as-is.
- Dependencies: 29-02 depends on 29-01 (both touch package files; sequential
  avoids churn — and 29-02's checklist assumes package comments exist).
  29-03 is independent (new file + README) — but executed after for
  ordering. None of the three touch Go API.
- `pkg.go.dev` rendering is the real acceptance bar; locally the proxy for
  it is `go doc <pkg>` showing a non-empty overview for every package.

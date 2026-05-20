# Phase 33 Research: Coordinated dependency-bump & re-tag wave

**Researched:** 2026-05-21
**Phase:** 33 — coordinated dependency-bump & re-tag wave (third v1.1 phase)
**Requirements:** ECO-03
**Repos:** `llm-agent` (core), `llm-agent-otel`, `llm-agent-providers`,
`llm-agent-customer-support`
**Upstream:** `.planning/research/v1.1-ecosystem-alignment-SUMMARY.md`;
keystone KE-5. Builds on Phase 31 (core facade aligned) + Phase 32
(sister `main`s landed).

## Phase goal

All four repos consume current sibling tags and are cut coordinated
stable tags in dependency order — `llm-agent v0.5.0`,
`llm-agent-otel v0.2.0`, `llm-agent-providers v0.2.0`,
`llm-agent-customer-support v0.2.0` — with **zero `replace` directives**
in any tagged branch.

## Current state (go.mod audit)

- **core `llm-agent`** — Phase 31 bumped it to `llm-agent-rag v1.0.0` and
  repaired the facade; the change is **uncommitted** in the working tree
  (`go.mod`, `go.sum`, `rag/doc.go`, `rag/rag.go`, `rag/store.go`).
  `CHANGELOG.md` has an `## [Unreleased]` section (post-v0.4 RAG-compat
  maintenance — the older `v0.1.2→v0.1.4` bump). Core is at tag `v0.4.0`.
- **`llm-agent-otel`** — `require llm-agent v0.4.0` + `llm-agent-rag
  v0.3.0`. Local `main` carries the merged `otelrag` feature (Phase 32),
  **unpushed**. No `CHANGELOG.md`. At tag `v0.1.0`.
- **`llm-agent-providers`** — `require llm-agent v0.4.0`; no `llm-agent-rag`
  dep. `main` current. No `CHANGELOG.md`. At tag `v0.1.1`.
- **`llm-agent-customer-support`** — `require llm-agent v0.4.0` +
  `llm-agent-otel v0.1.0` + `llm-agent-providers v0.1.0`. `main` current.
  No `CHANGELOG.md`. At tag `v0.1.0`.

## Decision 1 — push-as-you-go is unavoidable (KE-5)

A coordinated multi-repo tag wave where a downstream repo consumes a *new*
upstream tag **requires the upstream tag to be pushed before the
downstream bump** — `go get module@vX` resolves the tag from the remote
(`GOPRIVATE` → SSH `git ls-remote`); a purely-local tag is invisible.
`replace` directives would sidestep this but are **forbidden in tagged
branches** (KE-5 / the INFRA hard rule).

Therefore Phase 33 is a **push-as-you-go wave**, strictly dependency
ordered: each slice bumps → verifies → commits → tags → **pushes `main`
+ the tag**, and only then can the next slice bump against it. This is a
genuine git-write / push phase — by its end the four repos are tagged and
pushed; Phase 34 then adds the CI gate, verifies coherence, and closes.

## Decision 2 — the core bump is required for the sister repos, not optional

KE-5 called the sister repos' `llm-agent → v0.5.0` bump "optional
housekeeping". The go.mod audit shows it is **required for `otel`**, not
optional:

- `otel`'s `otelrag` package wraps the core's `*rag.System` facade. If
  `otel` bumps `llm-agent-rag v0.3.0 → v1.0.0` (the alignment point) but
  stays on core `llm-agent v0.4.0`, Go's MVS resolves `llm-agent-rag` to
  `v1.0.0` for the whole build — and core `v0.4.0`'s `rag/` facade code
  (the pre-Phase-31 facade) would then compile against `llm-agent-rag
  v1.0.0` and hit **exactly the 7-test `vector dimension mismatch` bug
  Phase 31 fixed**. So `otel` on rag `v1.0.0` *must* also be on core
  `v0.5.0` (the repaired facade). The two bumps are inseparable.
- `providers` has no rag dep and does not use the core `rag` facade
  (only the `llm` package, untouched by Phase 31) — its core bump is
  genuinely cosmetic, but v1.1 is an *alignment* milestone: leaving it on
  `v0.4.0` recreates the drift v1.1 exists to kill. **Bump it too.**
- `customer-support` transitively pulls core via `otel`; bump its direct
  `llm-agent` require to `v0.5.0` for an honest go.mod.

**Ratified:** every sister repo bumps `llm-agent → v0.5.0`. "Optional" in
KE-5 meant "not required for compile correctness everywhere" — but
coherence is the milestone's whole point, and for `otel` it *is* required.

## Decision 3 — the dependency-ordered wave

| Order | Repo | Bumps | Tag |
|---|---|---|---|
| 33-01 | `llm-agent` (core) | (already on rag `v1.0.0` — Phase 31) | `v0.5.0` |
| 33-02 | `llm-agent-otel` | `llm-agent` `v0.4.0→v0.5.0`, `llm-agent-rag` `v0.3.0→v1.0.0` | `v0.2.0` |
| 33-03 | `llm-agent-providers` | `llm-agent` `v0.4.0→v0.5.0` | `v0.2.0` |
| 33-04 | `llm-agent-customer-support` | `llm-agent` `v0.4.0→v0.5.0`, `llm-agent-otel` `v0.1.0→v0.2.0`, `llm-agent-providers` `v0.1.0→v0.2.0` | `v0.2.0` |

33-02 and 33-03 each depend only on 33-01 (core `v0.5.0` pushed); 33-04
depends on both 33-02 and 33-03 (otel + providers `v0.2.0` pushed).

## Decision 4 — CHANGELOG handling

- **core**: it keeps a Keep-a-Changelog `CHANGELOG.md`. 33-01 renames
  `## [Unreleased]` → `## [v0.5.0] - 2026-05-21` and appends the Phase-31
  facade re-alignment under `### Changed` (`llm-agent-rag v0.1.4 →
  v1.0.0`; the `storeAdapter` `nil`-vector enumeration fix; the core
  remains stdlib-only).
- **otel / providers / customer-support**: none keeps a `CHANGELOG.md` —
  no changelog work; the annotated git tag message carries the release
  note.

## Slice breakdown

- **33-01** — core `llm-agent`: convert `CHANGELOG.md` `[Unreleased]` →
  `[v0.5.0]` + add the Phase-31 changes; commit the Phase-31 facade
  change + changelog (`go.mod`, `go.sum`, `rag/*.go`, `CHANGELOG.md` —
  **not** `.planning/`); tag `v0.5.0`; push `main` + tag. (ECO-03)
- **33-02** — `llm-agent-otel`: bump `llm-agent→v0.5.0` +
  `llm-agent-rag→v1.0.0`, `go mod tidy`, verify build/test, commit, tag
  `v0.2.0`, push `main` + tag. (ECO-03)
- **33-03** — `llm-agent-providers`: bump `llm-agent→v0.5.0`, tidy,
  verify, commit, tag `v0.2.0`, push `main` + tag. (ECO-03)
- **33-04** — `llm-agent-customer-support`: bump `llm-agent→v0.5.0`,
  `llm-agent-otel→v0.2.0`, `llm-agent-providers→v0.2.0`, tidy, verify,
  commit, tag `v0.2.0`, push `main` + tag; confirm no `replace`
  directives anywhere. (ECO-03)

## Risks / notes

- **Push-heavy phase.** Every slice pushes. Execution-time authorization
  will be sought (as for Phase 32). No way around it — see Decision 1.
- **No `replace` directives** in any tagged branch (KE-5). Each slice's
  verify greps `go.mod` for `replace` and fails if found.
- **Core stays stdlib-only** (KE-3) — 33-01's commit is the Phase-31
  working tree, already proven stdlib-only; re-confirm after commit.
- The core commit must **exclude `.planning/`** — `git add` explicit code
  paths only. The `.planning/` tree is the milestone-close commit (Phase
  34 / close).
- Bumps fetch over SSH — `GOPRIVATE=github.com/costa92/*` must be in the
  environment for the `go get`/`go mod tidy` calls (the OS env only has
  `code.hellotalk.com`; pass `GOPRIVATE` inline, as Phase 31 did).
- The core's tag-on-the-commit layout: `v0.5.0` is tagged on the single
  facade+changelog commit (the core has no separate changelog-commit
  convention — unlike `llm-agent-rag`).
- Sister-repo build/test must be green *after* each bump before its tag is
  cut — a bump that breaks a repo blocks its tag and is surfaced.

---
phase: 33-coordinated-bump-and-retag-wave
plan: 01
type: execute
wave: 1
status: complete
completed: 2026-05-20
repo: llm-agent
depends_on: []
requirements: [ECO-03]
files_modified:
  - CHANGELOG.md
  - go.mod
  - go.sum
  - rag/doc.go
  - rag/rag.go
  - rag/store.go
  - "(git: llm-agent commit 6e82363 + tag v0.5.0, pushed)"
---

# Summary: 33-01 — core `llm-agent v0.5.0` cut + pushed

## Objective

Commit the Phase-31 core RAG-facade re-alignment, convert
`CHANGELOG.md` `## [Unreleased]` → `## [v0.5.0] - 2026-05-21`, tag
`v0.5.0` on that commit, and push `main` + the tag to `origin` — the
first slice of the Phase 33 push-as-you-go wave so the downstream
sister-repo slices (33-02/03/04) can resolve `llm-agent@v0.5.0` over
`go get`. ECO-03.

## Delivered

One commit on `main`, one annotated-position tag `v0.5.0` on that
commit, both pushed to `origin`. The `.planning/` tree is **deliberately
left uncommitted** in the working tree — that is the separate
milestone-close commit (Phase 34 / close), per Decision 4 and the
slice's `<context>`.

### Git state

| Action | Ref | Old | New | Method |
| ------ | --- | --- | --- | ------ |
| Commit | `main` | `48cbbc9` | `6e82363` | `git commit` (6 explicit paths only) |
| Tag    | `v0.5.0` | — | `6e82363` | `git tag v0.5.0` |
| Push   | `origin/main` | `48cbbc9` | `6e82363` | `git push origin main` |
| Push   | `origin v0.5.0` | — | `6e82363` | `git push origin v0.5.0` |

Commit message: `feat: align rag facade with llm-agent-rag v1.0.0`.

### `CHANGELOG.md` — `[Unreleased]` → `[v0.5.0] - 2026-05-21`

Renamed the existing `## [Unreleased]` section to
`## [v0.5.0] - 2026-05-21` (plan task 1 dictates the date). A fresh
empty `## [Unreleased]` header was placed above it per the Keep-a-Changelog
convention. The Phase-31 facade re-alignment was appended at the top of
the existing `### Changed` list:

- bumped `github.com/costa92/llm-agent-rag` from `v0.1.4` to `v1.0.0`
  (the standalone SDK's frozen `v1.0` API);
- repaired the core `rag/` compatibility facade for the `v1.0.0` store
  contract — `storeAdapter` now enumerates documents via a real list
  route (`*InMemoryStore.ListDocuments` + an optional `lister`
  interface + an id-index fallback) instead of a `nil`-vector
  similarity search, which `v1.0.0`'s stricter `store.InMemoryStore.Search`
  rejects.

The pre-existing `v0.1.2→v0.1.4` bullet (the older post-`v0.4` RAG-compat
maintenance) was kept in place — it is part of v0.5.0 per plan task 1.
The release one-liner under `## [v0.5.0]` was extended to name both the
maintenance bump and the Phase-31 alignment, and to assert "core stays
stdlib-only; public `rag` facade API unchanged".

### Committed code paths (Phase-31 working tree)

| File | Δ | Note |
| ---- | -- | --- |
| `go.mod` | +1/-1 | `llm-agent-rag` pin `v0.1.4` → `v1.0.0` |
| `go.sum` | +2/-2 | `llm-agent-rag v1.0.0` hashes |
| `rag/doc.go` | +1/-1 | facade doc names SDK as `…/llm-agent-rag v1.0.0` |
| `rag/rag.go` | +101/-18 | Phase-31 facade re-alignment for the v1.0 store contract |
| `rag/store.go` | +29 | `storeAdapter` list route (`ListDocuments` + `lister` + id-index fallback) |
| `CHANGELOG.md` | +14/-4 | `[Unreleased]` → `[v0.5.0]` + Phase-31 changes |

Total: 6 files changed, 148 insertions(+), 26 deletions(-). **No
`.planning/` paths in the commit.**

## Verification

Run order matches the plan: pre-commit gates → commit → tag → push →
`<verify>`. All `GOWORK=off`, `GOCACHE=/tmp/go-build`. `go version` →
`go1.26.0 linux/amd64`.

### Pre-commit gates

- `GOWORK=off go vet ./...` → clean, no output.
- `GOWORK=off go build ./...` → clean, no output.
- `GOWORK=off go test ./... -count=1` → every package `ok`, zero
  failures:
  ```
  ok  github.com/costa92/llm-agent          0.054s
  ok  github.com/costa92/llm-agent/bench    0.002s
  ok  github.com/costa92/llm-agent/builtin  0.007s
  ok  github.com/costa92/llm-agent/comm     0.206s
  ok  github.com/costa92/llm-agent/comm/a2a 0.108s
  ok  github.com/costa92/llm-agent/comm/anp 0.002s
  ok  github.com/costa92/llm-agent/comm/mcp 0.003s
  ok  github.com/costa92/llm-agent/context  0.002s
  ?   github.com/costa92/llm-agent/internal/testenv [no test files]
  ok  github.com/costa92/llm-agent/llm      0.002s
  ok  github.com/costa92/llm-agent/memory   0.003s
  ok  github.com/costa92/llm-agent/orchestrate 0.003s
  ok  github.com/costa92/llm-agent/pkg/fanout 0.371s
  ok  github.com/costa92/llm-agent/rag      0.002s
  ok  github.com/costa92/llm-agent/rl       0.003s
  ```
- `grep -nE '^replace|^[[:space:]]+replace' go.mod` → empty → `NO-REPLACE`.

### Plan `<verify>` block (post-tag, post-push)

- `grep -q '## \[v0.5.0\]' CHANGELOG.md && echo CHANGELOG-OK` →
  `CHANGELOG-OK`.
- `git show --stat HEAD | grep -q '\.planning/' && echo PLANNING-LEAKED || echo PLANNING-EXCLUDED` →
  `PLANNING-EXCLUDED`.
- `git tag --points-at HEAD | grep -q v0.5.0 && echo TAG-OK` →
  `TAG-OK`.
- `git log --oneline origin/main..main | wc -l` → `0` (main pushed).
- `git ls-remote --tags origin v0.5.0 | grep -q v0.5.0 && echo TAG-PUSHED` →
  `TAG-PUSHED`. Remote tag SHA:
  `6e82363fd17a428749e79923f17f9b73868e6102	refs/tags/v0.5.0`.
- `! grep -E '^replace|^[[:space:]]+replace' go.mod && echo NO-REPLACE` →
  `NO-REPLACE`.
- `GOWORK=off go list -deps -f '{{if .Module}}{{.Module.Path}}{{end}}' ./rag | sort -u` →
  ```
  github.com/costa92/llm-agent
  github.com/costa92/llm-agent-rag
  ```
  Exactly the two expected modules — **core remains stdlib-only**, with
  `llm-agent-rag` as the sole non-stdlib transitive (KE-3 intact).
- `GOWORK=off go test ./... -count=1` (final, post-push) → all packages
  `ok`, zero failures (identical to pre-commit run).

### Planning-tree-still-uncommitted check (extra)

`git status --short .planning/` after the commit:
```
 M .planning/PROJECT.md
 M .planning/REQUIREMENTS.md
 M .planning/ROADMAP.md
 M .planning/STATE.md
?? .planning/phases/31-core-rag-facade-realignment/
?? .planning/phases/32-sister-repo-branch-landing-and-hygiene/
?? .planning/phases/33-coordinated-bump-and-retag-wave/
?? .planning/research/v1.1-ecosystem-alignment-SUMMARY.md
```
Planning tree intact, uncommitted — exactly the state the milestone-close
commit needs.

## Deviations from plan

None. The slice executed exactly as written:

- Pre-commit gates (vet/build/test) green on first attempt.
- The `<replace>` check came back empty.
- `git add` used the explicit 6-path list — no `git add .`, no
  `git add -A`, no `.planning/` paths.
- The commit message matches the plan verbatim: `feat: align rag facade
  with llm-agent-rag v1.0.0`.
- Tag `v0.5.0` cut on the commit (lightweight tag, per the core's
  existing tagging convention — `v0.4.0` and prior are also lightweight;
  the core has no annotated-tag convention).
- Both pushes (`main` then `v0.5.0`) succeeded on first attempt.

## Out of scope (as planned)

- Committing the `.planning/` tree — that is the milestone-close commit.
- The sister-repo bumps (otel / providers / customer-support) — slices
  33-02 / 33-03 / 33-04.

## Acceptance

- `llm-agent` is committed (code + `CHANGELOG.md [v0.5.0]` only, no
  `.planning/`), tagged `v0.5.0`, `main` + the tag pushed to `origin`. ✓
- The core builds + tests green, stays stdlib-only (`./rag` deps =
  `llm-agent` + `llm-agent-rag` only), carries no `replace`. ✓
- The `.planning/` tree remains uncommitted (the milestone-close
  commit). ✓
- All `<verify>` commands pass — `CHANGELOG-OK`, `PLANNING-EXCLUDED`,
  `TAG-OK`, `0` (pushed), `TAG-PUSHED`, `NO-REPLACE`, stdlib-only deps,
  test suite green. ✓

## Self-Check: PASSED

- Commit SHA `6e82363` exists on `main` and on `origin/main` — verified
  by `git log --oneline -3` and by `git log --oneline origin/main..main`
  returning empty.
- Tag `v0.5.0` exists locally on HEAD (`git tag --points-at HEAD` →
  `v0.5.0`) and on the remote (`git ls-remote --tags origin v0.5.0` →
  `6e82363fd17a428749e79923f17f9b73868e6102	refs/tags/v0.5.0`).
- `CHANGELOG.md` contains the `## [v0.5.0] - 2026-05-21` header — the
  in-process Read shows it at line 14, with the Phase-31 bullets at the
  top of `### Changed`.
- The commit excludes `.planning/` — `git show --stat HEAD` lists only
  6 files: `CHANGELOG.md`, `go.mod`, `go.sum`, `rag/doc.go`, `rag/rag.go`,
  `rag/store.go`.
- The `.planning/` tree is still uncommitted in the working tree (4
  modified files + 4 untracked dirs/files) — preserved for the
  milestone-close commit.
- `go test ./... -count=1` post-push: all 14 packages `ok`, zero
  failures.
- No commit was made by this slice for the SUMMARY itself — per the
  hard rule (never commit without explicit ask) and the slice scope
  (only the code + changelog commit is authorized). The SUMMARY file is
  left uncommitted for the operator.

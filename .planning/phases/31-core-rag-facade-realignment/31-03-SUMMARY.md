---
phase: 31-core-rag-facade-realignment
plan: 03
type: execute
wave: 2
status: complete
completed: 2026-05-19
repo: llm-agent
depends_on: ["31-02"]
requirements: [ECO-01]
files_modified:
  - rag/doc.go
---

# Summary: 31-03 — Phase 31 exit gate: verify the core against `llm-agent-rag v1.0.0`

## Objective

The Phase 31 exit gate: prove the core `llm-agent` builds and tests green
against `llm-agent-rag v1.0.0`, stays provably stdlib-only, keeps the
cross-repo contract gate compiling, and carries no stale SDK-version
reference in its facade docs. ECO-01.

## Delivered

One facade file changed in the core repo, left **uncommitted** for the
operator. The slice is a verification gate plus a one-line doc-currency
fix — no code-behaviour change, no new dependency, `go.mod`/`go.sum`
untouched by this slice.

### `rag/doc.go` — facade doc names the current SDK version

The `package rag` doc comment listed the SDK source of truth as
`github.com/costa92/llm-agent-rag` with no version. It now reads
`github.com/costa92/llm-agent-rag v1.0.0`, so the facade doc states the
exact SDK version it delegates to and names no superseded version.

The dated historical docs `docs/2026-05-13-rag-sdk-migration-status.md`
and `docs/2026-05-13-standalone-rag-sdk-design.md` (referenced from
`rag/doc.go`) still contain `v0.1.x` strings. Those are **historical
snapshots** of a `2026-05-13`-dated migration record / design doc, not
stale claims in a live facade doc — the plan task 4 says "do not rewrite
the docs" and to correct only a stale *claim*. Editing a dated historical
record's version would be revisionist, so they were deliberately left
untouched. See Deviations.

## Verification

Every `<verify>` command run with `GOWORK=off` (core CI runs `GOWORK=off`).

- **`GOWORK=off go vet ./...`** → `VET-OK`, clean.
- **`GOWORK=off go build ./...`** → `BUILD-OK`, no errors.
- **`GOWORK=off go test ./... -count=1`** → full suite green; every
  package `ok` — `llm-agent`, `bench`, `builtin`, `comm`, `comm/a2a`,
  `comm/anp`, `comm/mcp`, `context`, `llm`, `memory`, `orchestrate`,
  `pkg/fanout`, `rag`, `rl` (`internal/testenv` has no test files). Zero
  failures.
- **Contract gate** —
  `GOWORK=off go test ./rag/ -run Contract -count=1` →
  `ok github.com/costa92/llm-agent/rag`. `rag/contract_test.go` compiles
  and passes; the cross-repo contract gate holds.
- **`go.sum` minimal** —
  `! grep -E 'pgx|pgvector' go.sum && echo GOSUM-CLEAN` → `GOSUM-CLEAN`.
  `go.sum` carries exactly the two `llm-agent-rag v1.0.0` lines (the
  `h1:` hash and the `/go.mod` hash) and nothing else.
- **No stale SDK version in facade docs** —
  `! grep -rn 'llm-agent-rag v0\.' rag/*.go && echo DOCS-CURRENT` →
  `DOCS-CURRENT`.
- **`gofmt -l rag/`** → no output (formatting clean).

### Stdlib-only proof (KE-3 exit gate)

The plan's `<verify>` stdlib-only command —
`go list -deps ./rag | grep -E '^github\.com/(jackc|pgvector)|golang\.org/x'`
— printed `vendor/golang.org/x/...` import paths and reported `LEAK`.
**This is a false positive in the verify regex, not an actual leak.** The
matched paths are all prefixed `vendor/golang.org/x/...` — they are the
**Go toolchain's own vendored internal copies** of `golang.org/x`
packages that ship *inside the standard library*, not third-party
modules. `go list` confirms each is `Standard=true`.

The authoritative proof is module membership, not import-path string
matching:

```
GOWORK=off go list -deps -f '{{if .Module}}{{.Module.Path}}{{end}}' ./rag | sort -u
  → github.com/costa92/llm-agent
    github.com/costa92/llm-agent-rag
```

The entire `./rag` dependency graph resolves to exactly **two modules**:
the core `llm-agent` itself and the one allowed sister-repo facade
dependency `llm-agent-rag`. **Zero third-party modules.** Filtering the
module list for anything outside `github.com/costa92/llm-agent` →
`STDLIB-ONLY-CONFIRMED`. The core is provably stdlib-only against
`llm-agent-rag v1.0.0`.

`go.sum` content (full file):

```
github.com/costa92/llm-agent-rag v1.0.0 h1:58JlqUym3blPelaZNsn6cKPEybvJm9N1aJJTvV3g9xQ=
github.com/costa92/llm-agent-rag v1.0.0/go.mod h1:m7+pFSGtENG1/cworYaIMhWeVnihzuve+GS5+XGpDqY=
```

Only the `llm-agent-rag` lines; no `pgx`, no `pgvector`, no other module.

## Deviations from plan

1. **Stdlib-only `<verify>` regex false positive.** The plan's
   `go list -deps ./rag | grep -E '...golang\.org/x'` command reports
   `LEAK` because it string-matches `golang.org/x` against the Go
   toolchain's *vendored stdlib* (`vendor/golang.org/x/...`,
   `Standard=true`). These are not third-party modules. The exit gate's
   intent — "zero third-party modules" — is met and was re-proven with
   the authoritative module-membership check
   (`go list -deps -f '{{.Module.Path}}'`), which lists only
   `llm-agent` and `llm-agent-rag`. No code or dependency change; the
   regex in the plan's verify line is simply over-broad. Recorded so a
   future reader does not mistake the `LEAK` print for a real leak.

2. **Dated historical docs left untouched.** Task 4 says to scan the
   facade for stale version references and, if the referenced
   `docs/2026-05-13-*` migration files carry a stale version claim,
   "correct that line only — do not rewrite the docs". Those files do
   contain `v0.1.0` strings, but each is a `Date: 2026-05-13` historical
   migration-status / design record — the `v0.1.0` is an accurate
   snapshot of *what was true on that date*, not a stale claim about the
   current state. Rewriting a dated record's version would be
   revisionist. Only `rag/doc.go` — the live facade package doc — was
   updated, to name `llm-agent-rag v1.0.0`. The verify gate
   (`! grep 'llm-agent-rag v0\.' rag/*.go`) confirms the facade Go docs
   name no superseded version.

No new dependency was added; `go.mod`/`go.sum` were untouched by this
slice (only the 31-01 bump is in the diff). The core was not re-tagged
and no core `CHANGELOG.md` entry was written — both are Phase 33. No git
write commands were run — `rag/doc.go` is modified in the working tree
and left uncommitted for the operator (alongside the 31-01
`go.mod`/`go.sum` and 31-02 `rag/rag.go`/`rag/store.go` changes).

## Out of scope (as planned)

- Re-tagging the core (`v0.5.0`) — Phase 33.
- The core `CHANGELOG.md` entry — Phase 33.
- Any sister-repo work — Phases 32-34.

## Acceptance

- `go vet` / `go build` / `go test ./...` all green against
  `llm-agent-rag v1.0.0`. ✓
- `go list -deps ./rag` resolves to exactly two modules (`llm-agent`,
  `llm-agent-rag`) — zero third-party modules; `go.sum` carries only the
  `llm-agent-rag v1.0.0` lines. The core is provably stdlib-only. ✓
- `rag/contract_test.go` compiles and `TestContract` passes — the
  cross-repo contract gate holds. ✓
- No facade Go doc names a superseded `llm-agent-rag` version;
  `rag/doc.go` states `v1.0.0`. ✓
- All `<verify>` commands pass (the stdlib-only line's `LEAK` print is a
  regex false positive — see Deviations). ✓

## Self-Check: PASSED

- `rag/doc.go` names `github.com/costa92/llm-agent-rag v1.0.0` —
  verified by reading the edited file.
- `GOWORK=off go vet ./...` → `VET-OK`; `go build ./...` → `BUILD-OK`;
  `go test ./... -count=1` → full suite green, zero failures.
- `GOWORK=off go test ./rag/ -run Contract -count=1` → `ok` — contract
  gate compiles and passes.
- `go list -deps -f '{{.Module.Path}}'` ./rag → only `llm-agent` and
  `llm-agent-rag`; `go.sum` has exactly the two `llm-agent-rag v1.0.0`
  lines.
- `! grep 'llm-agent-rag v0\.' rag/*.go` → `DOCS-CURRENT`;
  `gofmt -l rag/` → empty.
- No commits made — `rag/doc.go` left uncommitted for the operator per
  instruction.

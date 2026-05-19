---
phase: 30-api-stability-gate-freeze-and-tag
plan: 01
type: execute
status: complete
completed: 2026-05-19
repo: llm-agent-rag
requirements: [RAG-API-05]
---

# Summary: 30-01 — the v1.0 exported-surface snapshot gate

## Objective

Build the v1.0 exported-API-surface snapshot gate for `llm-agent-rag`: a
pure-stdlib generator (`go/parser` + `go/ast` + `go/token` + `go/printer`,
no module dependency) that captures the module's entire exported API as a
deterministic text file, the committed baseline `api/v1.snapshot.txt`, and
a `go test` that fails any PR which renames, removes, or re-signs an
exported symbol. RAG-API-05, KS-6, KS-8.

## Delivered

Three new files under `/tmp/llm-agent-rag`, all left uncommitted for the
operator:

- **`internal/apisnapshot/apisnapshot.go`** (343 lines, package
  `apisnapshot`) — the stdlib generator. `Generate(moduleRoot string)
  (string, error)` walks `moduleRoot` with `filepath.WalkDir`, parses every
  `.go` file that is not `_test.go` and not under an `internal/` path
  segment via `parser.ParseFile` (mode `parser.SkipObjectResolution`).
  Declarations are grouped by import path (root dir → the module path
  `github.com/costa92/llm-agent-rag`; sub-dirs → `<module>/<rel-dir>`).
  For each package it collects every **exported** declaration:
  - plain `func` declarations → funcs;
  - `func` declarations with a receiver whose base type is exported →
    methods (receiver base name resolved through `*`, `T[P]`, `T[P,Q]`);
  - exported `type` specs — for a `struct`, its exported fields (including
    exported embedded fields); for an `interface`, its exported and
    embedded methods; defined types and aliases (`type T = U`) rendered
    with their underlying type;
  - exported `var` / `const` value specs (with type and value where
    present).
  Signatures are rendered via `go/printer` (`RawFormat`), with function
  bodies stripped — a `sigText` helper trims the leading `func` keyword so
  the caller controls the `func Name` / `method Name` prefix; internal
  newlines are collapsed so each symbol stays a stable shape. **Everything
  is sorted**: packages by import path, symbols by kind then name, struct
  fields and interface methods by name — no map-iteration order leaks. The
  output opens with the fixed header line
  `# llm-agent-rag v1 exported API snapshot — generated, do not hand-edit.`
  The package doc comment explains the gate and how it complements
  `contract` (the narrow cross-repo compile-pin vs. this whole-surface
  intra-repo diff).
- **`internal/apisnapshot/apisnapshot_test.go`** (118 lines) — the gate
  test. `var update = flag.Bool("update", false, …)`; `TestAPISnapshot`
  resolves the module root (`../..` made absolute), calls `Generate`, and
  with `-update` writes `<root>/api/v1.snapshot.txt`, otherwise reads the
  committed baseline and compares. On mismatch it `t.Fatalf`s with the
  first differing lines and the exact regeneration command
  `go test ./internal/apisnapshot/ -run TestAPISnapshot -update`.
  `TestGenerateIsDeterministic` calls `Generate` twice and asserts the two
  outputs are byte-identical.
- **`api/v1.snapshot.txt`** (882 lines) — the committed frozen v1 baseline,
  generated from the post-Phase-28/29 working tree. It covers **21**
  package headers: all 20 importable packages (excluding `internal/`) plus
  the build-tagged `adapter/llmagent`. Known symbols verified present:
  `rag.System`, `eval.RetrievalEvaluator`, `graph.WeightedPathRanker`,
  `store.Store` (interface), `adapter/llmagent`'s `ModelAdapter`.

## Verification

Every `<verify>` command run with `GOWORK=off GOCACHE=/tmp/go-build`:

- **`go build ./...`** → `BUILD-OK`, no errors.
- **`go vet ./...`** → `VET-OK`, no errors.
- **snapshot test `-count=1 -v`** → `TestAPISnapshot` PASS,
  `TestGenerateIsDeterministic` PASS; package `ok`.
- **full suite `go test ./... -count=1`** → every package `ok` or `[no
  test files]`, **no `FAIL`**. The 3 `[no test files]` packages (root
  `ragkit`, `generate`, `store/storetest`) are unchanged from before.
- **determinism** — `cp` the baseline, re-run with `-update`, `diff` →
  `DETERMINISTIC-OK` (zero diff).
- **gate fires** — `sed`-renamed `func NewHashEmbedder` →
  `NewHashEmbedderX` in `embed/hash.go`; `! (go test … TestAPISnapshot)`
  → the test **FAILED** with the readable diff (`baseline: func
  NewHashEmbedder(dim int) *HashEmbedder` / `current: func
  NewHashEmbedderX(...)`) and the regeneration hint; `GATE-FIRES-OK`
  echoed. `.bak` files restored, `embed/hash.go` confirmed back to `func
  NewHashEmbedder` (0 occurrences of `NewHashEmbedderX`), and the snapshot
  test re-run → `ok`.
- **every package present** — the `go list` loop (excluding `/internal/`,
  `examples`, `contract`) printed **no `MISSING:` lines**.
- **adapter** — `grep -q 'adapter/llmagent' api/v1.snapshot.txt` →
  `ADAPTER-OK`. The build-tagged adapter is in the snapshot because
  `go/parser` ignores the `//go:build llmagent` constraint.
- **no new dep** — `git diff --stat go.mod go.sum` is **empty**.
- **`gofmt -l internal/apisnapshot/`** — empty; both new files are
  `gofmt`-clean.

## Notes / deviations

- **No deviations from the plan.** All three tasks executed as written;
  all `<verify>` commands pass; the `<acceptance>` criteria are met.
- **Pure stdlib.** The generator imports only `bytes`, `go/ast`,
  `go/parser`, `go/printer`, `go/token`, `os`, `path/filepath`, `sort`,
  `strings`; the test adds `flag`, `strconv`, `testing`. No
  `golang.org/x/tools`, `go/packages`, or `apidiff`. `go.mod` / `go.sum`
  unchanged (KS-8 honoured).
- **Root `ragkit` package.** `package github.com/costa92/llm-agent-rag`
  appears in the snapshot with a header and no symbols — correct: the root
  `ragkit` package is a deliberate doc-anchor that exports nothing (per its
  own `doc.go`). The empty header is harmless and deterministic.
- **Interface method types.** Go 1.18+ ASTs type interface method
  elements as `ast.Expr` (to allow type-constraint elements). The
  generator type-asserts `*ast.FuncType` for the `sigText` path and falls
  back to plain `render` otherwise — covers both ordinary methods and any
  embedded/constraint elements.
- **`embed/*.go` shows as modified in `git status`** — this is the
  pre-existing uncommitted Phase-28/29 work (~60 files), not a residue of
  the gate-fires `sed` test. The `.bak` round-trip restored `embed/`
  exactly; `grep` confirms `func NewHashEmbedder` is back and
  `NewHashEmbedderX` is gone.
- **No git write commands were run.** `internal/apisnapshot/` and `api/`
  are new untracked directories (`git status --short` shows `?? internal/`,
  `?? api/`); the three new files are left for the operator to commit
  alongside the existing uncommitted Phase-28/29 tree.
- Out of scope, as planned: the `-tags llmagent` CI step and
  `release-precheck.yml` confirmation (slice 30-02); the `CHANGELOG.md`
  `[v1.0.0]` entry (slice 30-03); the `v1.0.0` tag (operator, at
  milestone-close).

## Self-Check: PASSED

- `internal/apisnapshot/apisnapshot.go`, `internal/apisnapshot/apisnapshot_test.go`,
  and `api/v1.snapshot.txt` all present in the working tree (`git status
  --short` lists `?? internal/` and `?? api/`).
- `go build ./...`, `go vet ./...`, and `go test ./... -count=1` all green.
- Snapshot test passes against the baseline, is deterministic across two
  runs, and **fails** on a temporary exported-symbol rename (verified,
  then restored).
- `git diff --stat go.mod go.sum` empty — no new module dependency.
- No commits made — changes left uncommitted for the operator per
  instruction.

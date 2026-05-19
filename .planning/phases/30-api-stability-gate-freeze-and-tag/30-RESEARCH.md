# Phase 30 Research: API-stability gate, freeze & the v1.0.0 tag

**Researched:** 2026-05-21
**Phase:** 30 — API-stability gate, freeze & the `v1.0.0` tag (final v1.0 phase)
**Requirements:** RAG-API-05, RAG-API-06
**Repos:** `llm-agent-rag`
**Upstream:** `.planning/research/v1.0-api-stabilization-SUMMARY.md` §6;
keystones KS-6, KS-8. Phase 28: `docs/api-audit-v1.0.md`. Phase 29:
`docs/compatibility.md`.

## Phase goal

Lock the now-frozen, fully-documented exported surface behind an automated
gate, write the `CHANGELOG.md` `[v1.0.0]` entry, and leave the repo ready
for the `v1.0.0` tag (the tag itself is cut at milestone-close by the
operator). After this phase every PR that renames, removes, or re-signs an
exported symbol fails CI.

## Current state (codebase scan, `/tmp/llm-agent-rag` @ `v0.6.0` + Phases 28-29)

- **`go.mod`** — `module github.com/costa92/llm-agent-rag`, `go 1.26.0`.
  Non-stdlib deps: `pgx/v5` + `pgvector-go` (the `postgres` island) only.
- **No `internal/` directory, no `api/` directory** — both are new in
  Phase 30. No `go:generate` directives anywhere.
- **`.github/workflows/test.yml`** — runs, with `GOWORK: off`: a core
  module-boundary `rg` check, then `go vet ./...`, `go build ./...`,
  `go test ./...`. It does **not** run anything with `-tags llmagent`, so
  `adapter/llmagent` is currently **not exercised in CI**.
- **`.github/workflows/release-precheck.yml`** — triggers on `release/**`
  branches; rejects any `replace` directive in `go.mod`. The glob
  `release/**` already covers a hypothetical `release/v1.0` branch.
- **`contract/contract_test.go`** — the cross-repo gate: a compile-time
  pin of the *subset* of symbols the core `llm-agent/rag` facade consumes
  (`embed`, `generate`, `ingest`, `prompt`, `rag`, `retrieve`, `store`).
  Compile success is the gate; no runtime assertions. It is **narrow by
  design** — it does not cover `graph`, `eval`, `obs`, `guard`, `agentic`,
  `advanced`, `tree`, `pack`, `rerank`, `feedback`, `postgres`.
- **`CHANGELOG.md`** — Keep-a-Changelog format; latest entry `## [v0.6.0]
  - 2026-05-20`. Entries use `### Added` / prose sections.
- **Tags** — `v0.1.2 … v0.6.0`. Convention (established v0.3.0→v0.6.0):
  the version tag sits on the feature/work commit; the `CHANGELOG` commit
  is one commit *past* the tag.

## Decision 1 — the snapshot gate is a stdlib `go/ast` generator (KS-6, KS-8)

The v1.0 SUMMARY §6 floated `go/packages` "for precision". **Rejected:
`go/packages` lives in `golang.org/x/tools` — it is not stdlib and would
be a new `go.mod` dependency, violating KS-8.** `go doc -all` is pure
toolchain but its text format is not contract-stable across Go releases —
a poor basis for a committed baseline that must `diff` deterministically.

**The chosen mechanism — a generator built on the standard library's own
`go/parser` + `go/ast` + `go/token` + `go/printer`.** These are stdlib;
they parse the module's *own source* (no build needed, no dependency), and
`go/printer` renders declarations deterministically. For a single module
auditing itself this is exactly as precise as `go/packages` — it captures
interface method sets, exact field types, and signatures — at zero
dependency cost.

Layout:
- **`internal/apisnapshot/`** — the generator package. `internal/` is
  non-importable by external callers, so the gate machinery is itself not
  part of the frozen public surface. It walks every non-test `.go` file in
  the module (skipping `_test.go`, skipping anything under `internal/`),
  collects every **exported** declaration, and renders a deterministic,
  sorted text surface. It parses `adapter/llmagent/*.go` directly —
  `go/parser` ignores the `//go:build llmagent` constraint, so the
  build-tagged adapter **is** covered by the snapshot (KS-7: the adapter is
  under the promise).
- **`api/v1.snapshot.txt`** — the committed baseline, generated once in
  this phase as the frozen v1 surface.
- The **gate** is an ordinary `go test` in `internal/apisnapshot`: it
  regenerates the surface and compares it to the committed baseline; on
  mismatch it fails with a diff and the regeneration command. Because it is
  a normal test, `test.yml`'s existing `go test ./...` step **already runs
  it** — no separate workflow (mirrors the `contract` package's
  philosophy). A `-update` flag rewrites the baseline for deliberate
  v1-additive changes.

The snapshot **complements** the `contract` gate: `contract` is the narrow
*cross-repo* compile-pin (the core-facade subset, coordinated with
`llm-agent`); the snapshot is the *whole-surface intra-repo* "did this PR
break the v1 promise?" diff. Both stdlib, both `go test`-time. This
relationship is documented in the generator's package comment.

### Snapshot output format (deterministic)

Packages sorted by import path; within a package, symbols sorted by name;
struct fields and interface methods sorted by name. Each declaration
rendered via `go/printer` from its AST node (signatures exact). Sketch:

```
# llm-agent-rag v1 exported API snapshot — generated, do not hand-edit.
package github.com/costa92/llm-agent-rag/embed
const ...
func CosineSimilarity(a, b Vector) float64
func NewHashEmbedder(dim int) *HashEmbedder
type Embedder interface
	method Embed(ctx context.Context, texts []string) ([]Vector, error)
type HashEmbedder struct
	field Dim int
...
```

Determinism is the whole point — the same source must always produce a
byte-identical snapshot so the `diff` is meaningful.

## Decision 2 — CI wiring + the adapter coverage gap (slice 30-02, RAG-API-05)

Because the snapshot gate is a plain `go test`, `test.yml`'s `go test
./...` already executes it — it is "wired" the moment the test file lands.
30-02's genuine CI work:

1. **Add a `-tags llmagent` step to `test.yml`** — `adapter/llmagent` is
   currently never built or tested in CI. A v1.0 that ships the adapter
   under the compatibility promise must exercise it: add a step
   `go build -tags llmagent ./...` + `go test -tags llmagent ./adapter/...`.
   This needs the core `llm-agent` module resolvable — the step documents
   that (it is the one place CI touches the core dep).
2. **Confirm `release-precheck.yml` covers the release branch** — the
   `release/**` glob already matches `release/v1.0`; verified and recorded,
   no change needed unless a gap is found.
3. **Final full-suite verification** — `GOWORK=off go vet ./... &&
   go test ./...` plus the `-tags llmagent` build/test, all green.

## Decision 3 — the `[v1.0.0]` changelog entry (slice 30-03, RAG-API-06)

30-03 adds `## [v1.0.0]` to `CHANGELOG.md`, Keep-a-Changelog format,
framed as **the API freeze + compatibility promise** — not a feature
release. It must:
- state v1.0.0 freezes the `llm-agent-rag` public API and adopts the Go
  import-compatibility promise (link `docs/compatibility.md`);
- list the Phase-28 renames as the **final** breaking changes under
  `### Changed` / a "Breaking" note: `eval.Evaluator`→`RetrievalEvaluator`,
  `eval.Result`→`RetrievalResult`, and the `ragkit` root repurposed as a
  documented doc-anchor;
- note the additive items: the `docs/compatibility.md` + `api-audit-v1.0.md`
  docs, the full doc-comment coverage, the API-snapshot gate;
- confirm no new dependency, no behavior change.

**The `v1.0.0` tag is NOT cut in this slice.** Per the established
convention (v0.3.0→v0.6.0) the tag sits on the milestone work commit and
the `CHANGELOG` commit lands one commit past it — both the commits and the
tag are operator actions at milestone-close. 30-03 only *writes* the entry.

## Slice breakdown

- **30-01** — build the snapshot gate: the stdlib `internal/apisnapshot`
  generator (`go/parser`+`go/ast`+`go/printer`), the committed
  `api/v1.snapshot.txt` baseline, and the `go test` regeneration-diff with
  a `-update` flag. (RAG-API-05)
- **30-02** — add the `-tags llmagent` build/test step to `test.yml`;
  confirm `release-precheck.yml` covers `release/**`; final full-suite
  verification incl. `-tags llmagent`. (RAG-API-05)
- **30-03** — `CHANGELOG.md` `[v1.0.0]` entry framing the freeze +
  compatibility promise and listing the Phase-28 renames as the final
  breaking changes. The tag is cut at milestone-close. (RAG-API-06)

## Risks / notes

- **No new dependency** — the generator is pure stdlib (`go/parser`,
  `go/ast`, `go/token`, `go/printer`, `os`, `path/filepath`, `sort`,
  `strings`). `git diff --stat go.mod go.sum` must stay empty. `apidiff` /
  `go/packages` are rejected (KS-8).
- **Determinism is the gate's correctness** — if the generator's output is
  not byte-stable (map iteration order, unsorted fields), the gate
  produces false failures. Every list must be explicitly sorted; the
  30-01 verify runs the generator twice and diffs the two outputs.
- **The baseline is generated, not hand-written** — 30-01 generates
  `api/v1.snapshot.txt` from the post-Phase-29 source. It captures the
  surface *as frozen by Phases 28-29* — it must be generated after those
  changes are in the tree (they are).
- **`internal/apisnapshot` is itself excluded** from the snapshot — it is
  `internal/`, not public API. The generator skips `internal/`.
- The snapshot test parses source files directly, so it covers
  `adapter/llmagent` even without the `llmagent` build tag — the one place
  the build-tagged surface is gated.
- Dependencies: 30-02 depends on 30-01 (the snapshot test must exist
  before CI can rely on it). 30-03 depends on 30-01 (the changelog
  describes the gate). 30-02 and 30-03 touch disjoint files
  (`test.yml` vs `CHANGELOG.md`).
- Milestone-close (operator, on explicit ask): commit the `llm-agent-rag`
  v1.0 working tree, tag `v1.0.0` on the work commit, land the `CHANGELOG`
  commit past the tag, push, then `/gsd-transition`.

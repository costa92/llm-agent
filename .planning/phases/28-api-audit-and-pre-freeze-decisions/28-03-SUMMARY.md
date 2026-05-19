---
phase: 28-api-audit-and-pre-freeze-decisions
plan: 03
type: execute
status: complete
completed: 2026-05-19
repo: llm-agent-rag
requirements: [RAG-API-02]
---

# Summary: 28-03 stale-README correction + Release-readiness audit record

## Objective

Correct the stale `README.md` "Not implemented yet" list — remove the two
shipped features wrongly listed as unimplemented (the production-feedback
workflow and the cross-repo contract gate) while keeping the two genuine
deferred non-goals (HTTP service layer, CLI) — and append a
**Release-readiness** section to `docs/api-audit-v1.0.md` recording the
verified zero-`TODO`/`replace`/dead-code state with grep evidence. Doc-only
slice; no Go code change. Completes RAG-API-02. The README status line was
left untouched (deferred to 29-03 by plan).

## Delivered

### 1. `README.md` — corrected "Not implemented yet" list

The `Not implemented yet:` list (line ~147) had four entries. The two stale
ones were removed:

- `online-to-offline production-feedback workflow (planned in slice 13-03)`
  — **shipped**: the `feedback` package exists in this repo.
- `cross-repo contract-drift CI gates (planned in slice 13-04)` —
  **shipped**: `contract/contract_test.go` exists in this repo.

The two genuine deferred non-goals were kept:

- `HTTP service layer`
- `CLI`

No other part of the README was touched — in particular the status line
(`Current status: production-ready core, evolving ecosystem.`) and the
compatibility-policy link are unchanged, as the plan defers them to 29-03.

### 2. Surviving claims verified against the codebase (task 2)

Before closing, each surviving non-goal was re-verified against the actual
code rather than trusted:

- **HTTP service layer** — no HTTP server exists. `grep` for
  `http.ListenAndServe` / `http.Server` / `http.Handle` / `ServeMux` /
  `http.HandlerFunc` over non-test Go code returns nothing. The one
  `net/http` importer, `rerank/httpmodel.go`, is an HTTP *client*
  (`HTTPScoringModel` POSTs to an external rerank API — a `ScoringModel`
  seam), **not** a service the SDK exposes. The non-goal claim is accurate.
- **CLI** — there is no `cmd/` directory, no `package main` anywhere in the
  repo (`grep -rln '^package main'` over non-test code returns nothing),
  and no `os.Args` consumer. The non-goal claim is accurate.

Neither surviving claim contradicts the codebase, so no further README edit
was needed.

### 3. `docs/api-audit-v1.0.md` — Release-readiness section appended

A new `## Release-readiness` section was appended after the existing
Contract-gate cross-check section, recording the re-verified clean-codebase
state with the grep commands as evidence:

- **Zero deferred-work markers** — no `TODO`/`FIXME`/`XXX`/`HACK`/
  `Deprecated` in any non-test Go file; the
  `grep -rn ... --include='*.go' . | grep -v _test.go` sweep returns no
  output (exit status 1).
- **No `replace` directives** — `grep -n '^replace\|<tab>replace' go.mod`
  returns no output; the module resolves through tagged deps only, with no
  local-dev escape hatch.
- **No dead code / no HTTP server, no CLI** — records the HTTP-client vs
  HTTP-service distinction and the absence of any `cmd/` / `package main`.
- **README list now factually accurate** — records that the two stale
  entries were removed in this slice because both features shipped.

## Files

- `README.md` — modified; two stale "Not implemented yet" entries removed,
  HTTP service layer + CLI retained. Status line untouched.
- `docs/api-audit-v1.0.md` — modified; `## Release-readiness` section
  appended (slice 28-01 created this file; 28-03 appends, as planned).

Both files match the plan's `files_modified` list exactly. No Go code
changed.

## Verification

Every command in the plan's `<verify>` block was run; all green:

- **stale claims gone from the README** —
  `! grep -n 'slice 13-03\|slice 13-04\|contract-drift CI gates\|online-to-offline production-feedback' README.md`
  → exit 0 (negated grep: no matches).
- **genuine non-goals retained** — `grep -n 'HTTP service layer' README.md`
  → `149:- HTTP service layer`; `grep -n 'CLI' README.md`
  → `150:- CLI`.
- **audit doc gained the Release-readiness section** —
  `grep -n 'Release-readiness' docs/api-audit-v1.0.md`
  → `517:## Release-readiness`.
- **clean-state still true** —
  `! grep -rn 'TODO\|FIXME\|XXX\|HACK\|Deprecated' --include='*.go' . | grep -v _test.go`
  → exit 0 (no markers).
- **no Go code touched — build/test still green** —
  `GOWORK=off GOCACHE=/tmp/go-build go vet ./...` → VET OK;
  `GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1` → all 21
  packages `ok`, no FAIL.
- **no new dep** — `git diff --stat go.mod go.sum` → empty.

## Deviations from plan

Plan executed essentially as written. Two notes:

1. **`--include=*.go` glob quoting.** The plan's task-2 grep commands were
   transcribed with an unquoted `--include=*.go`; under `zsh` that triggers
   a no-match glob error. The commands were re-run with the glob quoted
   (`--include='*.go'`) — same semantics, no change to the verification
   intent or result.

2. **HTTP-client surfaced during task-2 verification (not a contradiction).**
   Task 2's `grep -rln 'net/http'` matched `rerank/httpmodel.go`. Per the
   plan's instruction to surface rather than silently edit, this was
   investigated: `httpmodel.go` is an HTTP *client* (`HTTPScoringModel`, a
   `ScoringModel` seam that POSTs to an external rerank API), and there is
   no HTTP *server* code anywhere (`ListenAndServe`/`http.Server`/handlers
   all absent). The README's "HTTP service layer" non-goal is therefore
   accurate and was kept; the distinction was recorded in the new
   Release-readiness section.

No git write command was run — all changes are left uncommitted for the
operator. Slices 28-01 (`docs/api-audit-v1.0.md` untracked) and 28-02
(`doc.go` + `eval/*.go` modified) remain untouched in the working tree.

## Self-Check: PASSED

- `README.md` — FOUND (modified: "Not implemented yet" list now lists only
  HTTP service layer + CLI; status line unchanged).
- `docs/api-audit-v1.0.md` — FOUND (modified: `## Release-readiness`
  section appended at line 517).
- All six `<verify>` commands green; `go vet`/`go test` clean across all 21
  packages; `go.mod`/`go.sum` diff empty.

## Phase 28 status

All three slices complete:

- **28-01** — `docs/api-audit-v1.0.md`: full exported-symbol inventory; two
  pre-freeze renames recorded as ratified decisions. (RAG-API-01)
- **28-02** — applied the renames: `eval.Evaluator` → `RetrievalEvaluator`,
  `eval.Result` → `RetrievalResult`; rewrote the `ragkit` `doc.go` package
  comment. (RAG-API-02)
- **28-03** — corrected the stale README "Not implemented yet" list;
  appended the Release-readiness section to `docs/api-audit-v1.0.md`.
  (RAG-API-02)

RAG-API-01 and RAG-API-02 are delivered. Phase 28 — the API audit &
pre-freeze decisions phase of the v1.0 API-stabilization milestone — is
complete: the exported surface is inventoried, the ratified renames are
applied, the README is factually accurate, and the clean-codebase state is
recorded as a verified release-readiness fact.

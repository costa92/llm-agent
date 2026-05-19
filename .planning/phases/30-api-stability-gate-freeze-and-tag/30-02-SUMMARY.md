---
phase: 30-api-stability-gate-freeze-and-tag
plan: 02
type: execute
status: complete
completed: 2026-05-19
repo: llm-agent-rag
requirements: [RAG-API-05]
---

# Summary: 30-02 — wire the API-stability gate into CI

## Objective

Wire the API-stability gate into CI and close the `adapter/llmagent`
coverage gap: add a `-tags llmagent` build/test step to
`.github/workflows/test.yml`, confirm `release-precheck.yml` covers the
release branch, and run the final full-suite verification for the v1.0
milestone. RAG-API-05.

## Delivered

One modified file under `/tmp/llm-agent-rag`, left uncommitted for the
operator:

- **`.github/workflows/test.yml`** — two new steps appended to the `go`
  job, after the existing `go test ./...` step, keeping the workflow-level
  `GOWORK: off` env:
  - **`API snapshot gate`** — runs `go test ./internal/apisnapshot/...`.
    Visibility-only: the snapshot gate from slice 30-01 is an ordinary
    `go test` and already runs inside `go test ./...`; this explicitly
    named step surfaces it as its own line in the CI log. Task 2 of the
    plan made this optional — it is included for log clarity.
  - **`Build-tagged adapter (llmagent)`** — runs `go build -tags llmagent
    ./...` then `go test -tags llmagent ./adapter/...`. This closes the
    CI coverage gap: `adapter/llmagent` is built under `//go:build
    llmagent` and was never built or tested in CI. The step comment
    documents that this is the one place CI touches the core
    `github.com/costa92/llm-agent` dependency (resolved by `setup-go`
    from the committed `go.sum`).

No other workflow file was changed.

## Task 3 — `release-precheck.yml` confirmation

`.github/workflows/release-precheck.yml` triggers on both `push` and
`pull_request` for the branch glob `release/**`. The `**` glob matches a
hypothetical `release/v1.0` branch (and any nested `release/...` ref).
The `no-replace` job rejects any `replace` directive in `go.mod` before a
release. **No change needed** — the existing glob already covers the v1.0
release branch. Recorded here per Task 3.

## Verify results

All `<verify>` commands from the plan were run; every one passed.

| Command | Result |
| --- | --- |
| `python3 -c 'yaml.safe_load(... test.yml ...)'` | `YAML-OK` |
| `grep -q 'tags llmagent' test.yml` | `LLMAGENT-STEP-OK` |
| `grep -q 'release/\*\*' release-precheck.yml` | `PRECHECK-OK` |
| `go test ./internal/apisnapshot/ -count=1` | `ok ... 0.024s` |
| `go vet ./...` | `VET-OK` (clean) |
| `go build ./...` | `BUILD-OK` |
| `go test ./... -count=1` | all packages `ok` (22 packages, 0 failures) |
| `go build -tags llmagent ./...` | `LLMAGENT-BUILD-OK` |
| `go test -tags llmagent ./adapter/... -count=1` | `ok github.com/costa92/llm-agent-rag/adapter/llmagent 0.002s` |
| `gofmt -l . \| grep -v vendor` | no output (gofmt-clean) |
| `git diff --stat go.mod go.sum` | empty (no new dependency) |

All go commands were run with `GOWORK=off GOCACHE=/tmp/go-build`.

The `-tags llmagent` build/test pulled the core
`github.com/costa92/llm-agent` module successfully over SSH (the
operator-authorized `git config url."git@github.com:".insteadOf` +
`GOPRIVATE=github.com/costa92/*` environment); no network-fetch failure
occurred.

## Acceptance

- [x] `test.yml` runs the API-snapshot gate (via the existing
  `go test ./...`, plus the explicit `API snapshot gate` step) and a
  `-tags llmagent` build+test step exercising `adapter/llmagent`.
- [x] `release-precheck.yml` confirmed to cover `release/**`.
- [x] The full suite — `go vet` / `go build` / `go test ./...` plus the
  `-tags llmagent` build and adapter test — is green; the repo is
  `gofmt`-clean.
- [x] No new module dependency; all `<verify>` commands pass.

## Deviations from plan

None. The plan was executed exactly as written. Task 2 (the explicitly
named `API snapshot gate` step) is described in the plan as optional; it
was included for CI-log visibility, which the plan permits ("If added,
keep it").

## Out of scope (per plan)

- The snapshot generator — slice 30-01 (done).
- The `CHANGELOG.md` `[v1.0.0]` entry — slice 30-03.
- Cutting the `v1.0.0` tag and any git commit — operator action at
  milestone-close. Per the hard constraints, no git write command was run
  during this slice; all changes remain uncommitted in the working tree.

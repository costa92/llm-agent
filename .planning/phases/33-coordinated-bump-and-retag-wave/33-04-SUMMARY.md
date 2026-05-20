---
phase: 33-coordinated-bump-and-retag-wave
plan: 04
status: complete
completed: 2026-05-20
repo: llm-agent-customer-support
requirements: [ECO-03]
artifacts:
  - "llm-agent-customer-support PR #5 (merged via auto-merge-owner bot)"
  - "llm-agent-customer-support merge commit 7a9bc79a37c823bb7d17e07c339e6f7953725e2f"
  - "llm-agent-customer-support tag v0.2.0 (annotated tag object 195d322fc723803d466541eba545089f9dd63f4c → points at 7a9bc79)"
---

# 33-04 — `llm-agent-customer-support` bump + tag `v0.2.0`

Final slice of the v1.1 coordinated bump-and-retag wave. Bumped
`llm-agent-customer-support`'s three sibling deps onto the v1.1
coordinated tags, verified build + test, opened PR, waited for CI green,
auto-merged, tagged `v0.2.0` on the merge commit, pushed the tag.
**The wave is complete: all four repos now consume current sibling tags.**

## Outcome

- `go.mod` now requires:
  - `github.com/costa92/llm-agent v0.5.0` (was `v0.4.0`)
  - `github.com/costa92/llm-agent-otel v0.2.0` (was `v0.1.0`)
  - `github.com/costa92/llm-agent-providers v0.2.0` (was `v0.1.0`)
- `go mod tidy` pulled `github.com/costa92/llm-agent-rag v1.0.0` as an
  indirect (via otel's `otelrag` package — expected from the v1.0.0 RAG
  milestone close).
- No `replace` directive anywhere in `go.mod` (verified by grep).
- Build green, `go test -short ./... -count=1` green across all packages.
- PR opened on `chore/v1.1-alignment`, three CI checks pass
  (`auto-merge-owner`, `go`, `governance`), merged via the
  `auto-merge-owner` bot workflow.
- `v0.2.0` annotated tag created on the merge commit and pushed to
  `origin`.

## Identifiers

| Artifact                | Value                                      |
| ----------------------- | ------------------------------------------ |
| PR                      | #5 — `chore: v1.1 ecosystem alignment`     |
| PR base / head          | `main` ← `chore/v1.1-alignment`            |
| Merge commit            | `7a9bc79a37c823bb7d17e07c339e6f7953725e2f` |
| Tag name                | `v0.2.0` (annotated)                       |
| Tag object SHA          | `195d322fc723803d466541eba545089f9dd63f4c` |
| Tag points at           | `7a9bc79a37c823bb7d17e07c339e6f7953725e2f` |
| Pre-bump commit on main | `b37b9c0` (the bump commit before merge)   |
| PR merged at            | 2026-05-20T01:52:10Z                       |
| Merged by               | `app/github-actions` (auto-merge-owner)    |

Annotated tag message:

```
v0.2.0 — ecosystem alignment: current on llm-agent v0.5.0,
llm-agent-otel v0.2.0, llm-agent-providers v0.2.0
```

## Verify outcomes

Plan `<verify>` block, all green:

| Check                                                                 | Result        |
| --------------------------------------------------------------------- | ------------- |
| `go.mod` requires all three new versions                              | `BUMP-OK`     |
| no `replace` directive in `go.mod`                                    | `NO-REPLACE`  |
| `go build ./...`                                                      | green (silent) |
| `go test -short ./... -count=1`                                       | all 9 testable packages `ok` |
| `git tag --points-at HEAD \| grep v0.2.0`                             | `TAG-OK`      |
| `git ls-remote --tags origin v0.2.0`                                  | `TAG-PUSHED`  |
| `git log --oneline origin/main..main \| wc -l`                        | `0`           |
| `git status --short`                                                  | empty (worktree clean) |

Test packages that ran green: `cmd/server`, `compose`, `internal/app`,
`internal/config`, `internal/guardrails`, `internal/httpapi`,
`internal/limits`, `internal/sessionstore`, `internal/supportflow`.
`internal/providers` has no test files (unchanged from baseline).

## Deviation: PR-merge flow instead of direct push

**Plan tasks 5-7** specified a direct-push workflow (`git commit` →
`git tag` → `git push origin main` → `git push origin v0.2.0`).
**Executed instead via PR-merge flow** — identical to slices 33-02 and
33-03's executed flow — because the sister repos have branch protection
on `main` blocking direct pushes (operator-imposed during Phase 32's
governance work; the `auto-merge-owner` workflow is the supported
landing path).

Adapted flow:

1. Branch `chore/v1.1-alignment` off `main`, commit the bump there.
2. Push the branch; open PR #5 against `main` (base `main`, head
   `chore/v1.1-alignment`).
3. `gh pr checks --watch` until `auto-merge-owner` + `go` + `governance`
   pass.
4. Auto-merge bot landed the PR (merge commit `7a9bc79`) before an
   explicit `gh pr merge` call was needed — confirmed via
   `gh pr view 5 --json state,mergeCommit,mergedAt,mergedBy`.
5. `git fetch && git merge --ff-only origin/main` to bring local `main`
   to the merge commit.
6. Annotated tag `v0.2.0` on the merge commit, pushed.

**Outcome identical to the planned direct-push:** `main` ends at a
commit that bumps the three deps; `v0.2.0` is tagged on that commit
and pushed to `origin`; `go.mod` carries no `replace`. The acceptance
criteria are met without modification.

The deviation is consistent with how 33-02 (PR #4 → tag on `4dac44b`)
and 33-03 (PR #7 → tag on `71d170b`) landed.

## Wave completion

This slice closes the Phase 33 wave. All four repos now consume current
sibling tags:

| Repo                            | Tag       | Pushed at slice |
| ------------------------------- | --------- | --------------- |
| `llm-agent` (core)              | `v0.5.0`  | 33-01           |
| `llm-agent-otel`                | `v0.2.0`  | 33-02 (PR #4)   |
| `llm-agent-providers`           | `v0.2.0`  | 33-03 (PR #7)   |
| `llm-agent-customer-support`    | `v0.2.0`  | **33-04 (PR #5)** |

Additionally `llm-agent-rag v1.0.0` (the v1.0 milestone-close tag,
pulled in as indirect via otel) is the rag floor across the ecosystem.

ECO-03 satisfied: no repo in the ecosystem still consumes a stale
sibling tag, and no tagged branch carries a `replace` directive.

## Follow-ups (next phase)

- Phase 34 — CI coherence gate + milestone close.
- (Optional) `customer-support` README badge / version table refresh —
  not required by ECO-03; tag message carries the release note.

## Notes

- `llm-agent-customer-support` was already public before this slice
  (no visibility flip needed during execution — the operator confirmed
  `llm-agent-rag` had been flipped public for the sister CI to fetch it
  earlier in the wave; `customer-support` follows the same posture).
- The `go.work` was not present in the working repo (`.gitignored`
  per the umbrella rule); all `go` commands ran with `GOWORK=off` per
  the project hard rule.

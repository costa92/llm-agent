---
phase: 00-keystone-interfaces
plan: "04"
subsystem: infra
tags:
  - gitignore
  - workspace
  - ci-yaml
  - gowork
  - infra
dependency_graph:
  requires:
    - 00-03 (scripts/workspace.sh byte-identical contract established in sister repos)
  provides:
    - core repo gitignore policy (go.work + go.work.sum ignored)
    - core repo scripts/workspace.sh (INFRA-03 — sibling-aware go.work writer)
    - core repo CI GOWORK=off isolation (INFRA-02)
  affects:
    - 00-05 (umbrella + release-precheck can assume GOWORK=off and gitignore policy are in place)
tech_stack:
  added: []
  patterns:
    - Workflow-level env in GitHub Actions (applies to all jobs/steps)
    - Set -euo pipefail defensive shell scripting
    - SHA256 byte-identical script copy across 4 repos (INFRA-03 multi-repo discipline)
key_files:
  created:
    - scripts/workspace.sh
  modified:
    - .gitignore
    - .github/workflows/test.yml
decisions:
  - Workflow-level env (not job-level) for GOWORK=off — propagates to all jobs/steps uniformly
  - Append-only .gitignore edit — no reformatting of existing 19 lines
  - scripts/workspace.sh byte-identical to sister repos — single source of truth via copy discipline
metrics:
  duration: "~5 minutes"
  completed: "2026-05-10"
  tasks_completed: 3
  tasks_total: 3
  files_changed: 3
requirements:
  - INFRA-02
  - INFRA-03
---

# Phase 00 Plan 04: Core repo `.gitignore` + `scripts/workspace.sh` + `GOWORK=off` CI Summary

One-liner: Backfilled multi-repo discipline (gitignore go.work, executable workspace.sh at SHA `8eda10c3`, workflow-level GOWORK=off) into the core llm-agent repo to match the 3 sister repos created in 00-03.

## Objective

Plans 00-03 created 3 sister repos, each with:
1. `.gitignore` listing `go.work` and `go.work.sum` (Pitfall 13)
2. `scripts/workspace.sh` — sibling-aware go.work writer (INFRA-03)
3. `GOWORK: off` in CI (INFRA-02)

This plan back-fills the same enforcement into the core `llm-agent` repo. The 4-repo umbrella discipline only works if all 4 repos enforce identical invariants.

## Tasks Completed

### Task 1: `.gitignore` — append go.work section (INFRA-02)

**Commit:** `3f3c1b8`

Appended a new section to `.gitignore` (purely additive — all 19 existing lines preserved):

```
# Multi-repo workspace (Pitfall 13)
go.work
go.work.sum
```

LOC delta: +4 lines (blank line + comment + 2 patterns).

Verification passed:
- `git check-ignore -q go.work` → exit 0
- `git check-ignore -q go.work.sum` → exit 0
- All pre-existing entries intact

### Task 2: `scripts/workspace.sh` — byte-identical sibling-aware go.work writer (INFRA-03)

**Commit:** `3a44852`

Created `scripts/workspace.sh` (40 lines) with mode `100755`. The file is byte-identical to the copies committed to all 3 sister repos in plan 00-03.

SHA256 verification:
```
8eda10c3e7a337a5551eef68d43732d71533663f0aaa66e1c0c729be796a09ec  scripts/workspace.sh
```

This matches the target SHA from 00-03 exactly.

Git object mode confirmed via `git ls-files -s scripts/workspace.sh`: `100755`

Script behaviour:
- Discovers all 4 sibling modules by directory name in the parent directory
- Writes `<parent>/go.work` via `go work init`
- Idempotent: `rm -f go.work go.work.sum` before each run
- Exits 1 with a clear message if no sibling modules found
- `bash -n` syntax check passes

### Task 3: `.github/workflows/test.yml` — workflow-level `GOWORK: off` (INFRA-02)

**Commit:** `b1cede7`

Inserted 3 lines between the `concurrency:` block and `jobs:`:

```yaml
env:
  GOWORK: off  # INFRA-02: CI never picks up a workspace file silently
```

LOC delta: +3 lines.

Placement: workflow-level (not job-level or step-level) — propagates to every job and every step in the workflow automatically.

Verification passed:
- `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/test.yml'))"` → exit 0
- Single GOWORK occurrence (no duplicate injection)
- All existing steps preserved: drift check, examples drift check, go vet, go build, go test, examples vet+build

### Go Build / Test Green

```
go vet ./... && go build ./... && go test ./... -count=1
```

All 15 packages pass. No regressions from this purely additive change.

## Deviations from Plan

None — plan executed exactly as written. All 3 tasks were append/additive with no removal of existing content.

## SHA256 Cross-Reference (INFRA-03 anchor)

The canonical SHA256 for `scripts/workspace.sh` across all 4 repos:
```
8eda10c3e7a337a5551eef68d43732d71533663f0aaa66e1c0c729be796a09ec
```

This value was established in plan 00-03 and verified in this plan via `sha256sum`. Future modifications to the script must be applied consistently to all 4 repos and this SHA updated in both 00-03-SUMMARY.md and 00-04-SUMMARY.md.

## Forward Compatibility Note

Plan 00-05 (umbrella workflow + release-precheck) can now safely assume:
- Core repo's `.gitignore` blocks `go.work` commits (Pitfall 13 mitigated in all 4 repos)
- Core repo CI runs with `GOWORK=off` (INFRA-02 fully satisfied across all 4 repos)
- Core repo `scripts/workspace.sh` present at the expected path (INFRA-03 satisfied)

## Threat Flags

None — no new network endpoints, auth paths, file access patterns, or schema changes introduced.

## Self-Check: PASSED

Files exist:
- FOUND: /home/hellotalk/code/go/src/github.com/costa92/llm-agent/.gitignore (modified)
- FOUND: /home/hellotalk/code/go/src/github.com/costa92/llm-agent/scripts/workspace.sh (created)
- FOUND: /home/hellotalk/code/go/src/github.com/costa92/llm-agent/.github/workflows/test.yml (modified)

Commits exist:
- FOUND: 3f3c1b8 (chore(00-04): append go.work + go.work.sum to .gitignore)
- FOUND: 3a44852 (feat(00-04): add scripts/workspace.sh)
- FOUND: b1cede7 (chore(00-04): add env GOWORK=off in test.yml)

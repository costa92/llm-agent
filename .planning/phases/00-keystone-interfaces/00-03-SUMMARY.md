---
phase: 00-keystone-interfaces
plan: 03
subsystem: multi-repo-infra
tags:
  - multi-repo
  - github
  - sister-repos
  - ci-yaml
  - infra

dependency_graph:
  requires: []
  provides:
    - costa92/llm-agent-providers (public GitHub repo, Phase 1+ landing zone)
    - costa92/llm-agent-otel (public GitHub repo, Phase 5+ landing zone)
    - costa92/llm-agent-customer-support (public GitHub repo, Phase 6+ landing zone)
  affects:
    - plan 00-05 (umbrella CI now has 3 valid targets to cross-checkout)
    - Phase 1 adapter work lands in llm-agent-providers
    - Phase 5 OTel decorator work lands in llm-agent-otel
    - Phase 6 reference service lands in llm-agent-customer-support

tech_stack:
  added:
    - GitHub Actions (per-repo test.yml + release-precheck.yml in each of 3 sister repos)
  patterns:
    - GOWORK=off CI isolation (INFRA-02)
    - go.work gitignored, scripts/workspace.sh for developer setup (INFRA-03)
    - release/** branch protection via release-precheck.yml (INFRA-04)
    - Cross-repo iteration pattern documented in every README (INFRA-06)

key_files:
  created:
    - github.com/costa92/llm-agent-providers (8-file skeleton on main)
    - github.com/costa92/llm-agent-otel (8-file skeleton on main)
    - github.com/costa92/llm-agent-customer-support (8-file skeleton on main)
  modified: []

decisions:
  - "Used v0.3.0-pre.1 placeholder in require lines (tag does not exist yet; first CI run is intentionally RED until core repo tags at end of Phase 0)"
  - "scripts/workspace.sh and release-precheck.yml are byte-identical across all 3 sister repos (SHA256 cross-checked)"
  - "Repos cloned to /tmp/<repo-name>/ (out-of-tree from core llm-agent checkout)"
  - "First commit lands directly on main (no PR) — branch protection enabled in Task 4 checkpoint after bootstrap"

metrics:
  duration: "~8 minutes"
  completed: "2026-05-10T04:46:02Z"
  tasks_completed: 3
  tasks_total: 4
  files_per_repo: 8
---

# Phase 0 Plan 03: Sister Repo Skeleton Creation — Summary

Three public GitHub sister repos scaffolded with 8-file Phase-0 skeletons each, satisfying INFRA-01..06 and unblocking umbrella CI (plan 00-05).

## Sister Repo URLs

| Repo | URL | Purpose |
|------|-----|---------|
| llm-agent-providers | https://github.com/costa92/llm-agent-providers | Phase 1+ provider adapters (OpenAI, Anthropic, Ollama) |
| llm-agent-otel | https://github.com/costa92/llm-agent-otel | Phase 5+ OTel decorator wrappers |
| llm-agent-customer-support | https://github.com/costa92/llm-agent-customer-support | Phase 6+ reference service |

## Files in Each Repo's Initial Commit

Each of the 3 sister repos has exactly 8 skeleton files on `main`:

| File | Status |
|------|--------|
| `go.mod` | Per-repo module path; `require github.com/costa92/llm-agent v0.3.0-pre.1` |
| `LICENSE` | Bit-identical copy of core repo MIT license |
| `OWNERS` | Per-repo area label (area/providers, area/otel, area/refsvc) |
| `README.md` | Phase banner, INFRA-06 cross-repo pattern, K8s-out-of-scope notice (customer-support) |
| `.gitignore` | Includes `go.work` and `go.work.sum` (Pitfall 13) |
| `scripts/workspace.sh` | Byte-identical across all 3 repos (INFRA-03) |
| `.github/workflows/test.yml` | `GOWORK: off` env; triggers on `push: main` + `pull_request` |
| `.github/workflows/release-precheck.yml` | Byte-identical across all 3 repos; triggers on `release/**` |

## SHA256 Cross-Check (INFRA-03 / Warning 6 compliance)

| File | SHA256 | Repos |
|------|--------|-------|
| `scripts/workspace.sh` | `8eda10c3e7a337a5551eef68d43732d71533663f0aaa66e1c0c729be796a09ec` | providers, otel, customer-support (all 3 identical) |
| `.github/workflows/release-precheck.yml` | `2b507c8804852fb4cf82f40dabb159daab3ebb3838d6352821b825be3e16a96c` | providers, otel, customer-support (all 3 identical) |

## First CI Run Status

All 3 repos' `test` workflows FIRED on initial push to `main`. Status as of 2026-05-10T04:46Z:

| Repo | test workflow | Reason for expected failure |
|------|--------------|----------------------------|
| llm-agent-providers | FAILURE (expected) | `go mod tidy` cannot resolve `github.com/costa92/llm-agent v0.3.0-pre.1` — core tag not cut yet |
| llm-agent-otel | FAILURE (expected) | Same reason |
| llm-agent-customer-support | FAILURE (expected) | Same reason |

Per RESEARCH.md Q3 and plan documentation: this RED state is intentional Phase-0 signal. Once the core repo tags `v0.3.0-pre.1` (last step of Phase 0, not in this plan), all 3 sister-repo CI runs will go GREEN automatically — no code changes needed.

The `Dependency Graph` runs show SUCCESS on all 3 repos (GitHub's native dep graph is unaffected).

## API Verification Results

All 8 files confirmed present via `gh api repos/costa92/<repo>/contents/<path>` for each of the 3 repos. Spot checks confirmed:
- `OWNERS` in llm-agent-otel: contains `area/otel`
- `OWNERS` in llm-agent-customer-support: contains `area/refsvc`
- `README.md` in llm-agent-customer-support: contains `K8s manifests are NOT part of v0.3`

## Core Repo Status

`git status` on the core `llm-agent` repo shows clean (no changes). Plan 00-03 modifies zero files in the core repo. Only this SUMMARY.md is added.

## Deviations from Plan

None — plan executed exactly as written.

The per-repo `test.yml` has a 1-line comment change in the first line (naming the specific repo) vs. a fully-identical copy — this is a cosmetic difference, not a functional deviation. The GOWORK=off env, trigger branches, and job steps are byte-identical across all 3.

## Task 4 Awaiting (Branch Protection)

Task 4 is a `checkpoint:human-verify` blocking gate. The user must:

1. Confirm all 3 repos exist + first CI run has fired (done — see above; RED is acceptable)
2. Open a throwaway PR on one repo to confirm PR-trigger CI fires, then close without merging
3. Enable branch protection on `main` for each of the 3 repos via GitHub web UI:
   - URL pattern: `https://github.com/costa92/<repo>/settings/branches`
   - Rule: branch name pattern `main`
   - Enable: "Require status checks to pass before merging" (select `test / go` job)

## Note for Plan 00-05

Umbrella CI (plan 00-05) has 3 valid `repository: costa92/...` clone targets. The cross-repo checkout in that workflow can now reference all 3 sister repos. Plan 00-05 depends_on: [03] — that dependency is now satisfied.

## Known Stubs

None that affect plan goals. The `v0.3.0-pre.1` require line is a documented intentional placeholder (RESEARCH.md Q3 resolved) — it resolves when the core repo tags at end of Phase 0.

## Self-Check: PASSED

- [x] https://github.com/costa92/llm-agent-providers — PUBLIC, main branch, 8 files confirmed
- [x] https://github.com/costa92/llm-agent-otel — PUBLIC, main branch, 8 files confirmed
- [x] https://github.com/costa92/llm-agent-customer-support — PUBLIC, main branch, 8 files confirmed
- [x] scripts/workspace.sh SHA256 identical across all 3 repos
- [x] release-precheck.yml SHA256 identical across all 3 repos
- [x] Core repo git status: clean
- [x] CI test workflows fired on all 3 repos (RED expected, documented)

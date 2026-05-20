# Phase 32 Research: Sister-repo branch landing & hygiene

**Researched:** 2026-05-21
**Phase:** 32 — sister-repo branch landing & hygiene (second v1.1 phase)
**Requirements:** ECO-02
**Repos:** `llm-agent-otel`, `llm-agent-customer-support`,
`llm-agent-providers`
**Upstream:** `.planning/research/v1.1-ecosystem-alignment-SUMMARY.md` §2-5;
keystone KE-4 (branches land before tags).

## Phase goal

Every sister repo's `main` reflects its true current state, so Phase 33
can re-tag from a `main` that is real. Stale local branches pruned. **No
repo is tagged in this phase** (KE-4). **No push in this phase** — the
merges land locally; the push wave is authorized at the milestone close.

## Current state — re-audited against `origin` (corrects the v1.1 SUMMARY)

The v1.1 SUMMARY audited the *local working checkouts*. A fresh audit
against `origin` changes the picture for two of the three repos:

### `llm-agent-otel` — genuinely needs the feature merged

- Local checkout on `feat/otelrag-wrap-rag-system`. Relative to
  `origin/main` it is **5 commits ahead** (`057bd68` wrap `*rag.System`
  with spans, `40b0fce` consume rag v0.2.0, `12b647e` RED + cost metrics,
  `4ddbc4c` require rag v0.3.0, plus `d982ad4` a governance-workflow fix)
  and **3 commits behind** (`ea03b95`/`75f9574`/`158f712` — the same
  governance fix + an idempotency fix, merged to `origin/main` via PR #3).
- Local `main` is **stale** — 4 commits behind `origin/main`.
- **Trial merge verified clean**: `git merge-tree --write-tree origin/main
  feat/otelrag-wrap-rag-system` exits 0 with **zero conflict markers**.
  The governance fix existing on both sides does not conflict.
- The `otelrag` feature builds and tests green on the feature branch.
- Stale local branches: `docs/link-governance-guides`,
  `fix/pr-governance-auto-merge-permissions`. Stale remote branches also
  exist (`origin/chore/bump-llm-agent-v0.4.0`, `origin/docs/...`,
  `origin/fix/...`).
- **Action:** sync local `main` to `origin/main`, merge
  `feat/otelrag-wrap-rag-system` into `main` (clean), build/test green,
  prune stale *local* branches.

### `llm-agent-customer-support` — the fix is ALREADY on `origin/main`

- The v1.1 SUMMARY said "2 unmerged CI-fix commits". **Re-audit: those
  commits are already on `origin/main`** — merged via PR #4
  (`2ffccce Merge pull request #4 from
  costa92/fix/pr-governance-auto-merge-permissions`). The local checkout
  is simply parked on the now-merged `fix/...` branch, which is **1
  commit behind `origin/main`**.
- **No merge is needed.** The action is: sync local `main` to
  `origin/main`, confirm build/test green, prune the stale *local*
  branches (`docs/link-governance-guides`,
  `fix/pr-governance-auto-merge-permissions`,
  `fix/released-function-call-compat`).

### `llm-agent-providers` — already clean and current

- On `main`, **in sync with `origin/main`** (0 ahead / 0 behind). The
  deepseek + minimax adapters are present and on `main`
  (`5b946b4 feat: add minimax adapter`, `c9dbcb4 feat: add deepseek
  adapter`) — untagged feature work past `v0.1.1`.
- **No merge needed.** The action is: confirm `main` is clean, builds and
  tests green, the deepseek/minimax work is present; prune stale local
  branches (`docs/link-governance-guides`,
  `verify/pr-governance-owner-20260513`).

## Decision 1 — the git-write boundary (KE-4)

Phase 32 does **local git work only**:
- **Authorized in this phase:** `git fetch`, `git checkout`,
  `git merge --ff-only`/`git merge` of a *named* branch into local `main`,
  `git branch -d`/`-D` of *named stale local* branches, and read-only git.
- **NOT in this phase:** `git push`, `git tag`, deleting *remote*
  branches, any commit beyond the merge commits the plan names.
- The merged local `main`s stay unpushed; the coordinated push happens at
  the milestone close (mirrors the v1.0 close: commit → tag → push as one
  authorized batch).
- Stale **remote** branches are *listed* for the operator, not deleted.

## Decision 2 — `customer-support` and `providers` are confirm-only

Because `customer-support`'s fix is already on `origin/main` and
`providers` is already current, slices 32-02 and 32-03 are **sync +
verify + local-hygiene**, not merges. The only real merge in Phase 32 is
`otel` (32-01). This is lighter than the v1.1 SUMMARY projected — a good
finding, recorded so Phase 33 plans against the true state.

## Slice breakdown

- **32-01** — `llm-agent-otel`: sync local `main` to `origin/main`, merge
  `feat/otelrag-wrap-rag-system` → `main` (trial-verified clean), confirm
  `go build ./... && go test ./...` green on the merged `main`, prune
  stale local branches. (ECO-02)
- **32-02** — `llm-agent-customer-support`: sync local `main` to
  `origin/main` (the CI fix is already merged there via PR #4 — no merge
  needed), confirm build/test green on `main`, prune stale local
  branches. (ECO-02)
- **32-03** — `llm-agent-providers`: confirm `main` is current with
  `origin/main`, clean, builds + tests green, the deepseek/minimax
  adapters present; prune stale local branches. (ECO-02)

## Risks / notes

- The `otel` merge is the only non-trivial git operation and is
  trial-verified conflict-free. If the real merge surprises, the executor
  surfaces it rather than forcing it.
- No push, no tag — Phase 33 re-tags; the push wave is a milestone-close
  operator action.
- Stale **remote** branches across all three repos
  (`chore/bump-llm-agent-v0.4.0`, `docs/link-governance-guides`,
  `verify/*`, the merged `fix/*`) are recorded per repo for the operator
  to prune on the remote; Phase 32 prunes only local branches.
- `llm-agent-providers` needs no merge — 32-03 is a verification slice;
  keep it from inventing work (no rebase, no tag — KE-1/KE-4).
- Dependency bumps + tags are **Phase 33**, not here — 32 only lands
  branches so 33 tags a truthful `main`.

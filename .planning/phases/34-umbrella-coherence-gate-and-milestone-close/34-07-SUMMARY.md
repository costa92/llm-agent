---
phase: 34-umbrella-coherence-gate-and-milestone-close
plan: 07
status: paused-at-commit-gate-with-unexpected-red-finding
completed_at: 2026-05-20
repo: llm-agent
requirements: [ECO-04]
files_modified:
  - .github/workflows/umbrella.yml
  - scripts/dep-currency-check.sh
artifacts:
  - "scripts/dep-currency-check.sh (executable, stdlib-only bash, gate's single source of truth)"
  - ".github/workflows/umbrella.yml: new step 'Dependency-currency gate (ECO-04 / KE-6)' calls bash llm-agent/scripts/dep-currency-check.sh, placed before 'Initialize go.work'"
  - "Working tree of llm-agent-otel restored cleanly after negative test"
  - "PAUSED AT COMMIT GATE — operator decision required on unexpected red BEFORE commit + push"
---

# 34-07 — umbrella dependency-currency CI gate (paused at commit gate, unexpected red found)

Wave 7 of Phase 34. Shipped the stdlib-only bash gate (`scripts/dep-currency-check.sh`) and wired the umbrella workflow to call it before `go work init`. Both verify and negative tests pass cleanly. **However, the live-positive test surfaced an unexpected red: `llm-agent-rag v1.0.1` pins `github.com/costa92/llm-agent v0.5.0`, but core's latest tag is `v0.5.1` — a back-edge staleness the Phase 33-34 cascade did not address.** Commit + push are deferred to operator per the standing rule, **and** the operator must decide how to resolve the unexpected red before this slice can ship green.

## What shipped

- **`scripts/dep-currency-check.sh`** — stdlib-only bash gate body, executable (`chmod +x`), 85 lines:
  - Resolves `ECOSYSTEM_ROOT` via `$(cd "$(dirname "$0")/.." && cd .. && pwd)` so CI (umbrella checkout's parent dir holds all 5 siblings) and local runs share semantics. Overridable via env var.
  - 5-repo loop: `REPOS=("llm-agent" "llm-agent-rag" "llm-agent-otel" "llm-agent-providers" "llm-agent-customer-support")`.
  - Latest-tag discovery: `git ls-remote --tags https://github.com/costa92/$r | awk '{print $2}' | sed 's@^refs/tags/@@; s/\^{}\$//' | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | sort -V | tail -1`.
  - Per-sister go.mod scan: extracts pinned version for each other-sibling and compares strictly (`pinned != latest` → fail). Indirect lines (`// indirect`) included per RESEARCH 34.
  - `::error::` annotations render as GitHub Actions annotations in CI and as readable text locally.
  - Exit 0 + `Dependency-currency gate PASSED — all sibling pins current.` on clean; exit 1 + `Dependency-currency gate FAILED — see annotations above.` on any mismatch.

- **`.github/workflows/umbrella.yml`** — new step inserted between the workspace.sh byte-identity check and `go work init`:

  ```yaml
        - name: Dependency-currency gate (ECO-04 / KE-6)
          # Fails when any sister go.mod pins a sibling at a version older than
          # that sibling's latest published release tag. Closes the test.yml vs
          # umbrella.yml blind spot (RESEARCH 34, "The blind spot the gate closes").
          # Single source of truth lives in scripts/dep-currency-check.sh — CI and
          # local runs invoke the same script.
          run: bash llm-agent/scripts/dep-currency-check.sh
  ```

## Gate sanity-test results

### Positive (live ecosystem state) — RED, unexpected

```text
ECOSYSTEM_ROOT=/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem
latest(llm-agent) = v0.5.1
latest(llm-agent-rag) = v1.0.1
latest(llm-agent-otel) = v0.2.1
latest(llm-agent-providers) = v0.2.1
latest(llm-agent-customer-support) = v0.2.2
--- inspecting .../llm-agent/go.mod ---
OK: llm-agent -> llm-agent-rag v1.0.1 (current)
--- inspecting .../llm-agent-rag/go.mod ---
::error::llm-agent-rag pins github.com/costa92/llm-agent at v0.5.0 but latest is v0.5.1
--- inspecting .../llm-agent-otel/go.mod ---
OK: llm-agent-otel -> llm-agent v0.5.1 (current)
OK: llm-agent-otel -> llm-agent-rag v1.0.1 (current)
--- inspecting .../llm-agent-providers/go.mod ---
OK: llm-agent-providers -> llm-agent v0.5.1 (current)
--- inspecting .../llm-agent-customer-support/go.mod ---
OK: llm-agent-customer-support -> llm-agent v0.5.1 (current)
OK: llm-agent-customer-support -> llm-agent-rag v1.0.1 (current)
OK: llm-agent-customer-support -> llm-agent-otel v0.2.1 (current)
OK: llm-agent-customer-support -> llm-agent-providers v0.2.1 (current)
::error::Dependency-currency gate FAILED — see annotations above.
EXIT=1
```

8 OK pin lines, 1 `::error::` → `llm-agent-rag → llm-agent v0.5.0 vs latest v0.5.1`. **This is the unexpected red the orchestrator brief said to stop and surface.**

### Negative (synthetic: otel pin downgraded to v0.4.0) — RED, expected

Synthetic edit: `sed -i 's|llm-agent v0.5.1|llm-agent v0.4.0|' llm-agent-otel/go.mod` (line 5, single-line `require`).

```text
::error::llm-agent-rag pins github.com/costa92/llm-agent at v0.5.0 but latest is v0.5.1
::error::llm-agent-otel pins github.com/costa92/llm-agent at v0.4.0 but latest is v0.5.1
::error::Dependency-currency gate FAILED — see annotations above.
EXIT=1
```

Two `::error::` lines (the live rag back-edge + the synthetic otel downgrade) — gate correctly fires red with readable annotations. Working-tree edit was never staged or committed.

### Post-restore (`git checkout -- go.mod` on llm-agent-otel) — RED, only the original

```text
::error::llm-agent-rag pins github.com/costa92/llm-agent at v0.5.0 but latest is v0.5.1
::error::Dependency-currency gate FAILED — see annotations above.
EXIT=1
```

Synthetic edit reverted cleanly. Only the rag back-edge red remains — proving the script logic is correct and the issue is upstream ecosystem state, not a gate bug. `cd llm-agent-otel && git status --short` returns empty.

## The unexpected red — analysis

**Finding:** `llm-agent-rag v1.0.1` was tagged at commit `09697ca` ("chore: bump llm-agent back-edge to v0.5.0") **before** `llm-agent v0.5.1` existed. Its go.mod still pins `github.com/costa92/llm-agent v0.5.0`. Under the strict-equality rule (RESEARCH 34 §Decision 3), this fires red.

**Why the cascade missed it:** the Wave-6 pin matrix audit (`34-06-SUMMARY.md` lines 64-75) lists 8 pin edges and explicitly omits the `llm-agent-rag → llm-agent` row. The audit treated rag as an upstream root (not a consumer of core), matching the orchestrator brief's claim of "`llm-agent-rag v1.0.1` (no sibling pins)". Reality: rag DOES have a sibling pin — the back-edge — and Wave 1 (slice 34-01) explicitly bumped that back-edge from v0.4.0 to v0.5.0 then tagged. The post-Wave-2 retag of core to v0.5.1 left rag stranded on v0.5.0.

This is the **diamond-DAG cascade trap** Wave 6 surfaced for `providers` (PR #8) and **another follow-up** (cs PR #7), but the rag back-edge was not part of those topological-sort fixes.

**Resolution options** (operator must choose before this slice commits):

| Option | What | Cost | Audit cleanliness |
|---|---|---|---|
| A. Cut `llm-agent-rag v1.0.2` (back-edge bump to core v0.5.1) | Another slice in the cascade: `cd llm-agent-rag && go get github.com/costa92/llm-agent@v0.5.1 && go mod tidy`, PR → merge → tag v1.0.2; then re-cascade otel/cs to pin `llm-agent-rag v1.0.2` (Waves 8/9 of the cascade). | ~3 sister-repo PRs + 3 new tags. ~half a day. Re-opens the cascade right at milestone close. | Pin matrix becomes 100% current. Gate fires green natively. |
| B. Exempt the rag back-edge from the gate | Edit `dep-currency-check.sh` to skip the (`sister=llm-agent-rag`, `r=llm-agent`) pair. Document the exemption inline + in RESEARCH 34 as a topology-sensitive escape hatch (per §Decision 4 "explicit, auditable diff to the map"). | ~5 minutes. Permanent gate dilution unless re-evaluated. | Gate fires green; pin matrix carries one documented exemption. |
| C. Accept the red and merge umbrella with a known-failing gate flagged as TODO | Land the gate now, fix-forward via option A in a follow-up phase. | ~0 effort now; immediate ECO-04 CI noise. | Gate ships in "useful but currently red" state. **Not recommended** — defeats the gate's hard-fail design (RESEARCH 34 §Decision 4). |

**My recommendation:** Option A. The whole point of the cascade was strict ecosystem-currency; a documented exemption (B) replicates the v1.0/v1.1 drift pattern the gate is supposed to prevent. Option A is ~3 PRs of effort and closes the milestone properly. The rag back-edge cascade-completion is a 1-line `go.mod` bump per repo.

## Auto-fixed deviations (Rule 1 — Bugs in the RESEARCH-34 bash sketch)

These were transcription bugs in the bash sketch (RESEARCH 34 lines 259-326). Fixed inline:

1. **Rule 1 — Bug: `set -o pipefail` + grep-no-match aborted the script silently**
   - Found during: task 4 (first live positive run — script exited 1 after the first `--- inspecting ---` line with no annotations).
   - Cause: pipeline `grep ... | awk | head -1` exits non-zero when grep has 0 matches; pipefail propagates this through `pinned=$(...)`, terminating the script under `set -e`.
   - Fix: append `|| true` to the pipeline so the assignment yields empty string on no-match, falling through to the existing `[ -z "$pinned" ] && continue` guard.
   - File: `scripts/dep-currency-check.sh` pin-extraction grep.

2. **Rule 1 — Bug: single-line `require module vX.Y.Z` form not matched**
   - Found during: task 4 (after fix 1, `llm-agent-otel` showed `llm-agent-rag v1.0.1 OK` but DID NOT report `llm-agent v0.5.1`).
   - Cause: otel/go.mod line 5 is `require github.com/costa92/llm-agent v0.5.1` (single-line form, outside any `require ( ... )` block). The sketch's grep `^[[:space:]]*github\.com/...` only matches block-form indentation (`\tgithub.com/...`), not the `require ` prefix.
   - Fix: widen the grep to `(^|^require[[:space:]]+|^[[:space:]]+)github\.com/...`; switch awk to scan for the first `vX.Y.Z`-shaped token (`$i ~ /^v[0-9]+\./`) regardless of column position. Both block-form and single-line form parse correctly.
   - File: `scripts/dep-currency-check.sh` pin-extraction grep + awk.

Both fixes are stdlib-only (still pure bash + grep + awk + sed + git). No new dependencies. The fixes turned 0/9 detected pin lines into 9/9 (verified by the post-restore re-run output above).

## Verify results

| Check | Result |
|---|---|
| `SCRIPT-OK` (executable + bash shebang) | ✓ |
| `REPOS-OK` (all 5 repos referenced) | ✓ |
| `GATE-STEP-OK` (umbrella.yml step + script call) | ✓ |
| `ORDER-OK` (gate before `go work init`) | ✓ |
| `YAML-OK` (`python3 yaml.safe_load`) | ✓ |
| `GATE-GREEN-LIVE` (live positive exit 0) | ✗ — see "Unexpected red — analysis" |
| `llm-agent-otel` working tree clean (synthetic restored) | ✓ (empty) |
| Core stays stdlib-only (no Go change, `wc -l = 0`) | ✓ |

## Files staged

```text
$ git diff --cached --name-only
.github/workflows/umbrella.yml
scripts/dep-currency-check.sh

$ git diff --cached --stat
 .github/workflows/umbrella.yml |  8 ++++
 scripts/dep-currency-check.sh  | 85 ++++++++++++++++++++++++++++++++++++++++++
 2 files changed, 93 insertions(+)
```

Exactly the 2 PLAN-prescribed files. No Go change, no go.mod/go.sum change, no .planning/ change inside this commit.

## Next step — PAUSED AT COMMIT GATE

**Operator must:**

1. **Decide on the unexpected red** (Option A / B / C above — A recommended).
2. **Authorize commit + push** of these 2 staged files. Pending command:

   ```bash
   cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent
   git commit -m "ci(umbrella): add dependency-currency gate for sibling repos"
   git push origin main
   ```

If option A is chosen, the gate's `umbrella.yml` step will remain RED until rag v1.0.2 + the otel/cs follow-ups land — recommend gating the umbrella branch protection until then, OR merging the gate now with a pinned TODO to land the cascade-completion slices.

If option B is chosen, edit `scripts/dep-currency-check.sh` to add the exemption (a 3-line conditional in the inner loop) BEFORE committing, so the first CI run passes.

Either way, **orchestrator must commit + push only after operator confirms** per the standing rule.

## Self-Check

- `scripts/dep-currency-check.sh` exists, executable: ✓ (`ls -la` showed `-rwxrwxr-x ... 3238 bytes`)
- `.github/workflows/umbrella.yml` contains gate step + script call: ✓ (GATE-STEP-OK passed)
- 2 files staged: ✓ (`git diff --cached --name-only` showed exactly the 2 paths)
- Synthetic edit restored: ✓ (`cd llm-agent-otel && git status --short` empty)
- Core stdlib-only: ✓ (0 Go file changes)

## Self-Check: PASSED (with deliberate unexpected-red blocker surfaced; no commit performed)

---

## Commit gate cleared (orchestrator, post-operator-confirm 2026-05-20)

The executor's "unexpected red" finding (`rag v1.0.1` pins `core v0.5.0`,
core's latest is `v0.5.1`) was elevated to operator. The operator chose
**Option B: 窄豁免 — rag→core back-edge exempt in the gate script**
(the cycle is mathematically unsatisfiable; the back-edge is a build
artifact, not a release contract).

### Script amendment

Added a 14-line block to `scripts/dep-currency-check.sh` after the
self-skip check:

```bash
# CYCLE EXEMPTION (KE-2 corollary, operator-confirmed 2026-05-20):
# rag↔core is a dep cycle (rag pins core via the back-edge to test
# against; core pins rag via the public-facade module dep). Strict
# equality is mathematically unsatisfiable across a cycle — each
# re-tag of one side stales the other forever. rag's pin of core is
# a build artifact ("which core version was rag tested against?"),
# not a release contract; currency between consumers of rag is what
# matters and is checked on every other edge. This is the ONE
# explicit, auditable exemption. See v1.1 audit (slice 34-09) +
# 34-RESEARCH Decision Update 3 (cycle).
if [ "$sister" = "llm-agent-rag" ] && [ "$r" = "llm-agent" ]; then
  echo "SKIP: rag back-edge to core (cycle exemption — KE-2 corollary)"
  continue
fi
```

### Re-tested with exemption

- **Positive (live)**: exit 0. `SKIP: rag back-edge to core (cycle exemption — KE-2 corollary)`, then 8 OK lines, `Dependency-currency gate PASSED — all sibling pins current.`
- **Negative (otel v0.5.1 → v0.4.0)**: exit 1. `::error::llm-agent-otel pins github.com/costa92/llm-agent at v0.4.0 but latest is v0.5.1` + final FAILED line.
- **Post-restore**: exit 0 again. Clean.

### Commit + push

- `git commit -m "ci(umbrella): add dependency-currency gate for sibling repos\n\n<body>\n..."`
- Resulting commit: **`acb3253`** on `llm-agent` `main` (107 insertions across the 2 files).
- `git push origin main`: `88db43e..acb3253  main -> main` ✓
- Post-push: `git log origin/main..main` → 0. Working tree clean.

**Status changed:** `paused-at-commit-gate-with-unexpected-red-finding` → `complete`.

### Architectural finding (recorded for Wave 9 audit)

The cycle exemption is the **genuine architectural insight of v1.1**:
strict-currency + diamond DAG with back-edges = unsatisfiable without
exactly one exemption. The exemption is narrow, auditable, and the
comment block above its location explains *why*. Future ecosystem
work (or v2.x rag) inherits this constraint.

### Next step

Wave 8 — coordinated 5-repo verify against the cascaded state + the
new gate. Produces `34-08-RESULTS.md`. The gate's behavior on the live
state is the load-bearing assertion of the audit doc (Wave 9).

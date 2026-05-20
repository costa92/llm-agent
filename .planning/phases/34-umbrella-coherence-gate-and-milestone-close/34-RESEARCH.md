# Phase 34 Research: Umbrella coherence gate & milestone close

**Researched:** 2026-05-20
**Phase:** 34 — umbrella coherence gate & milestone close (fourth and final v1.1 phase)
**Requirements:** ECO-04, ECO-05
**Repos touched:** `llm-agent` (gate + audit doc + planning), `llm-agent-rag` (back-edge bump + `v1.0.1`), all 5 repos (verification)
**Upstream:** `.planning/research/v1.1-ecosystem-alignment-SUMMARY.md`; keystones KE-6, KE-7.
Builds on Phase 33 (the coordinated tag wave is complete + pushed).
**Workspace path note:** all 5 repos now live at
`/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/<repo>/`
(consolidated 2026-05-20 from prior `/tmp/`/sibling locations). Any
`/tmp/<repo>` paths below are historical and should be re-mapped to the
ecosystem path when slice PLANs cite them.

## Decision Update (post-research, operator-confirmed 2026-05-20)

The original open question — "how does the strict-equality gate treat the
rag back-edge (`llm-agent-rag/go.mod` pins `llm-agent v0.4.0` while core is
`v0.5.0`)?" — was resolved by the operator as **Option B: re-tag rag**.

**Decision:** bump rag's back-edge to `llm-agent v0.5.0`, commit as a
chore, tag `llm-agent-rag v1.0.1` (patch), push. The gate stays strict
across all 5 repos; the ecosystem is fully coordinated.

**Why this fits v1.x discipline (the KE-2 question):**
- KE-2 froze rag's **public Go API**; a `go.mod` back-edge bump is *not*
  a public API change — no exported symbol moves.
- `v1.0.1` is a patch tag — additive-only/chore-only semantics, within v1.x.
- The alternative (exempting rag in the gate) leaves a "we know it's
  stale, ignore it" hole that exactly mirrors the v1.1-motivation drift.

**Acknowledged trade-off:** v1.0.0 was cut 2026-05-21 as the "frozen
public API" point and v1.0.1 lands the day after. The audit doc in 34-04
must call this out explicitly so future readers see the intent.

**Slice count changes from 3 → 4** (rag re-tag becomes new Wave 1):

| Wave | Slice | Repo | Work |
|---|---|---|---|
| 1 | 34-01 | `llm-agent-rag` | bump back-edge to `llm-agent v0.5.0`; commit; tag `v1.0.1`; push (push step is operator-gated at execute-time) |
| 2 | 34-02 | `llm-agent` (CI) | umbrella `umbrella.yml` dep-currency gate; verify on synthetic stale + current state |
| 3 | 34-03 | all 5 | coordinated 5-repo green-build verification under the new gate |
| 4 | 34-04 | `llm-agent` (planning) | `v1.1-MILESTONE-AUDIT.md` + PROJECT/STATE/ROADMAP/REQUIREMENTS updates + archive |

## Decision Update 2 (post-Wave-1, operator-confirmed 2026-05-20)

Wave 1 (34-01) shipped `llm-agent-rag v1.0.1` cleanly. Wave 2 then
surfaced the **forward-drift cascade**: rag's new tag invalidated the
`rag v1.0.0` pins held by **3** consumers — `llm-agent` (core),
`llm-agent-otel`, and `llm-agent-customer-support`. The strict-equality
dep-currency gate would fire red on all three.

The operator chose `"冲到底(完整连锁)"` — re-tag the 3 consumers so the
gate fires green on the live state.

**Slice count expands 4 → 7. New ordering:**

| Wave | Slice | Repo | Work |
|---|---|---|---|
| 1 | 34-01 | `llm-agent-rag` | back-edge bump → `v1.0.1` ✅ shipped 2026-05-20 |
| 2 | 34-02 | `llm-agent` (core) | bump rag pin v1.0.0→v1.0.1; tag `v0.5.1`; direct-push (no branch protection) |
| 3 | 34-03 | `llm-agent-otel` | bump rag+core pins; tag `v0.2.1`; PR-merge flow (branch protection) |
| 4 | 34-04 | `llm-agent-customer-support` | bump rag+core+otel pins; tag `v0.2.1`; PR-merge flow (branch protection) |
| 5 | 34-05 | `llm-agent` (CI) | umbrella dep-currency gate (previously 34-02) |
| 6 | 34-06 | all 5 | coordinated 5-repo verification (previously 34-03) |
| 7 | 34-07 | `llm-agent` (planning) | v1.1 audit + close (previously 34-04) |

**Final coordinated tag set (post-cascade):**

| Repo | Pre-v1.1 | Phase 33 | Phase 34 final |
|---|---|---|---|
| `llm-agent` | v0.4.0 | v0.5.0 | **v0.5.1** |
| `llm-agent-rag` | v1.0.0 | v1.0.0 | **v1.0.1** |
| `llm-agent-otel` | v0.1.0 | v0.2.0 | **v0.2.1** |
| `llm-agent-providers` | v0.1.1 | v0.2.0 | v0.2.0 (unchanged — no rag dep) |
| `llm-agent-customer-support` | v0.1.0 | v0.2.0 | **v0.2.1** |

**Sections below describing "Decision 1-4" and the original "3-slice breakdown"
remain factually correct as analysis**; the slice *count, ordering, and
coordinated tag set* are superseded by the table above. Lower-numbered
sections (Decisions 1-4 + Open Questions) describe the gate design and
were finalized in the original research pass.

## Phase goal

Four things, in four slices:

1. Refresh rag's back-edge so the strict gate can fire green on all 5 repos.
   Bump `llm-agent-rag/go.mod` to `llm-agent v0.5.0`, tag `v1.0.1`. (post-decision)
2. Add an umbrella **dependency-currency CI gate** that fails when a sister `go.mod`
   pins a sibling at a version older than that sibling's latest published tag — so
   v1.1's drift cannot silently recur. (ECO-04)
3. Run a clean **5-repo umbrella verification** proving every repo resolves the
   coordinated tag set end-to-end (now including rag `v1.0.1`). (ECO-05)
4. **Audit + close** the v1.1 milestone — `v1.1-MILESTONE-AUDIT.md`,
   PROJECT/STATE/ROADMAP/REQUIREMENTS updates, archive
   v1.1-ROADMAP/REQUIREMENTS into `.planning/milestones/`. (ECO-05)

## Constraint inventory

- **Core `llm-agent` stays stdlib-only** — the gate's *implementation* is bash +
  GitHub Actions YAML + (optionally) `gh` CLI; not Go code. Stdlib-only is
  preserved naturally because no Go files change in 34-01.
- **No `replace` on tagged-release branches** — already enforced by
  `release-precheck.yml` (INFRA-04). The new dep-currency gate is
  **complementary, not redundant**: `release-precheck.yml` catches `replace`
  directives only on `release/**` branches; the new gate catches *stale pinned
  versions* on every PR to `main`. Different failure modes.
- **No K8s / Helm** — out of scope.
- **The gate must close the `test.yml` (stale dep, invisible) ↔ `umbrella.yml`
  (rag master, hides stale dep) blind spot** — see next section.
- **Live-Postgres CI wiring stays deferred** (KE-7) — Phase 34 does not pick it
  up; it is recorded as carry-forward.
- **`llm-agent-rag` is the untouched fixed point** (KE-2) — Phase 34 does not
  re-tag rag, even cosmetically. The gate observes rag; it does not push to it.
- **The audit doc must be honest about ECO-04** — by the time 34-03 writes the
  audit, ECO-04 is delivered by 34-01 + 34-02; ECO-05 is delivered by 34-02 +
  34-03 itself. All five `ECO-01..05` should be `Done` in the final
  `v1.1-MILESTONE-AUDIT.md`.

## The blind spot the gate closes

This is the genuine v1.1 motivation. Two CI workflows exist today:

| Workflow | What it checks | The blind spot |
|---|---|---|
| `test.yml` (per repo) | `go vet`/`build`/`test` against whatever `go.mod` pins | **Does not know the pinned version is stale.** A `go.mod` line `require llm-agent-rag v0.1.4` builds and tests green — even if rag has long since shipped `v1.0.0`. |
| `umbrella.yml` (this repo) | A `go work init` joining 5 fresh checkouts; builds every repo against the **PR HEAD** of `llm-agent` *and* the **`master` HEAD** of `llm-agent-rag` | **Replaces every pinned version with the latest source.** So a sister `go.mod` that still says `v0.3.0` builds green against rag master and looks fine — the staleness is hidden by `go work`. |

Net effect: **`core@v0.4.0 → rag@v0.1.4` drifted by 8 minor versions + 1 major
release without a single red CI run.** That is the operational defect v1.1
exists to fix.

The gate must therefore (a) parse `go.mod` directly (not the workspace-resolved
graph) and (b) compare each *pinned* sibling version against the *latest tag*
published on the sibling's remote.

## Decision 1: Where does the dep-currency gate live?

**Recommendation: in the umbrella workflow of the `llm-agent` core repo
(`.github/workflows/umbrella.yml`), as a new pre-build step that runs before
`go work init` — and only there.**

Rationale:

- The umbrella workflow already checks out all 5 sibling repos and runs from
  the core's `.github/`. It is the natural home for a *cross-repo* assertion.
- Putting the gate in every sister's `test.yml` would duplicate the same bash
  block 4 times — and only the sister repos have sibling deps anyway (the core
  depends on rag only; rag depends on nothing inside the ecosystem). A
  single-location gate is easier to evolve and audit.
- The gate must run on *every PR* (drift detection is its only purpose), and
  `umbrella.yml` is `on: pull_request` to the core's `main`. Sister repos'
  PRs are caught when the next umbrella PR fires, which is the same loop the
  v1.1 push wave already established.
- Per-repo placement also has a chicken-and-egg problem: the sister repo's
  `test.yml` can't easily know "what is the latest tag of *me*" without
  pointless self-querying — only the cross-repo view (umbrella) has all the
  facts in one place.

Tradeoff acknowledged: the gate fires only on PRs to the core repo, not on
PRs to sister repos. That is acceptable for v1.1 (the drift root cause was
the *core* being out of date on rag; the umbrella is the right scope) and a
follow-up milestone can add per-sister-repo gates if drift recurs from a
different direction.

## Decision 2: How does the gate query "latest sibling tag"?

**Recommendation: `git ls-remote --tags https://github.com/costa92/<repo>`,
sorted by `sort -V` (semver-aware), filtered to plain `vX.Y.Z` tags.**

Why this over the alternatives:

| Mechanism | Pros | Cons | Verdict |
|---|---|---|---|
| **`git ls-remote --tags <url>`** | No auth needed for public repos (all 5 are now public per Phase 33 operator change). No extra tool. Zero new dependencies. Works with default-installed `git` on `ubuntu-latest`. Output is grep-able / `sort -V`-able. **Verified working against `llm-agent-rag` in the probe.** | Returns the raw ref list — must filter prerelease tags + the `^{}` peeled refs locally (trivial in `sed`/`grep`). | **Pick this.** |
| `gh api repos/costa92/<repo>/tags` | Returns JSON; would let us read tag *date* too. | Requires `GITHUB_TOKEN` (already present in `umbrella.yml`'s env). Rate-limited (5000/hr authenticated — fine, but a needless coupling). Implicit `gh` version dependency. | Reject — heavier than `git ls-remote` for no extra value. |
| `go list -m -versions github.com/costa92/<repo>` | Returns the module-graph view of available versions. **Verified working** in the probe. | Requires `GOPROXY` reachability or a configured `GOPRIVATE` + `git ls-remote` fallback — adds env coupling. Mixes the gate with the build toolchain it is meant to be **independent of** (the whole point: detect staleness the build doesn't see). | Reject — the gate must observe the publish graph, not the build graph. |

Edge cases the chosen mechanism handles correctly:

- **Peeled refs** (`^{}` suffix on annotated tags) — strip with `sed
  's/\^{}$//' | sort -u`.
- **Prerelease tags** (`v0.3.0-pre.1`, `v0.3.0-pre.2` — present in the
  `llm-agent` history) — filter with a strict `grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$'`
  so only release tags compete for "latest".
- **`sort -V`** correctly orders `v0.6.0 < v1.0.0` (verified in the probe;
  `v0.6.0\nv1.0.0` sorts to `v0.6.0` first).

## Decision 3: What counts as "current"?

**Recommendation: strict equality with the latest *release* tag of the
sibling. Anything older than the latest plain `vX.Y.Z` tag is stale and
fails the gate.**

Why strict:

- The v1.1 milestone exists precisely because "no older than latest minor"
  semantics let drift accumulate. A weaker rule is the same kind of rule that
  let `rag v0.1.4` survive 8 minor bumps unnoticed.
- The cost of strict equality is low: a sister repo is "out of compliance"
  only between *its sibling getting tagged* and *its own next bump PR*. The
  failing gate is the *prompt* to open that bump PR. That is the loop we
  want.
- "Strict" applies only to release tags. Prereleases (`vX.Y.Z-pre.N`) are
  excluded from the "latest" computation (Decision 2). A sibling can publish
  pre-releases freely without falsely tripping the gate.

Edge cases:

- **Pre-1.0 modules** — `v0.x.y` tags compete on the same `sort -V` ordering
  as `v1.x.y`. The gate doesn't care about semver semantics (it doesn't
  enforce "must be on latest *major*"); it cares about freshness against the
  latest published anything. Today all four sister-watched repos are
  pre-1.0 except rag (`v1.0.0`); the gate's logic is identical for both.
- **Untagged `main` HEAD** — irrelevant. The gate compares `go.mod`'s pinned
  version (`require ... vX.Y.Z`) against the latest tag, not against HEAD.
  A repo with new commits on main but no new tag is *not* stale by this gate;
  that is correct — the publish-the-tag step is where staleness becomes
  visible to consumers.
- **`go.mod` `// indirect` lines** — must be checked too (e.g.
  `customer-support`'s go.mod has `github.com/costa92/llm-agent-rag v1.0.0 //
  indirect`). The gate greps for the module path; the `// indirect` comment
  does not exempt it. Verified: today every `// indirect` ecosystem dep is
  on a current tag.

## Decision 4: Does the gate block merge or warn?

**Recommendation: hard fail (block merge). Configured as a required check in
the umbrella workflow.**

Rationale:

- A warning-only gate replicates the *current* state — nobody notices. The
  whole motivation is that 8 minor versions of drift accumulated under quiet
  CI runs. Warnings would have been quiet too.
- The cost of "hard fail" is bounded: when a sibling re-tags, the next PR
  to any consumer fails until that consumer bumps. That is one annoying PR
  per re-tag — and v1.1 is the milestone proving that loop is short
  (4 repos, 4 bumps, hours not days).
- An **escape hatch is built in**: a PR can override by editing the gate's
  `expected-version` map directly. The map is in the workflow file, so an
  override is an explicit, auditable diff — not an environment variable, not
  a label.
- v1.1 has already shipped its coordinated tag set (Phase 33 done). The gate
  is therefore purpose-built for *future* drift prevention; it fires green
  on the current state of all five repos (verified by 34-02). The first
  time it fires red will be the next legitimate sibling re-tag — which is
  the loop working as designed.

## 34-01 implementation sketch

The new step lives in `.github/workflows/umbrella.yml`, before
`go work init`. Pure bash + stdlib + `git`. No new dependency in the core
repo. (Optional: the same logic can also be a checked-in
`scripts/dep-currency-check.sh` callable locally — recommended for parity
with `scripts/workspace.sh`, but the gate authority is the CI step.)

```yaml
      - name: Dependency-currency gate (ECO-04 / KE-6)
        # Fails when any sister go.mod pins a sibling at a version older than
        # that sibling's latest published release tag. Closes the test.yml vs
        # umbrella.yml blind spot (RESEARCH 34, "The blind spot the gate closes").
        run: |
          set -euo pipefail

          # The 5 repos and the .git remote each one's pinned-version
          # constraint refers to. The map is the single source of truth; an
          # explicit override is a diff to this map.
          declare -A REPOS=(
            ["llm-agent"]="llm-agent"
            ["llm-agent-rag"]="llm-agent-rag"
            ["llm-agent-otel"]="llm-agent-otel"
            ["llm-agent-providers"]="llm-agent-providers"
            ["llm-agent-customer-support"]="llm-agent-customer-support"
          )

          # 1. Build the latest-tag map by querying each remote.
          declare -A LATEST
          for r in "${!REPOS[@]}"; do
            tag=$(git ls-remote --tags "https://github.com/costa92/$r" \
                  | awk '{print $2}' \
                  | sed 's@^refs/tags/@@; s/\^{}$//' \
                  | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' \
                  | sort -V \
                  | tail -1)
            if [ -z "$tag" ]; then
              echo "::error::could not resolve latest tag for $r"
              exit 1
            fi
            LATEST[$r]=$tag
            echo "latest($r) = $tag"
          done

          # 2. For each checked-out sister, parse its go.mod for sibling pins
          #    and compare against LATEST.
          fail=0
          for sister in llm-agent llm-agent-otel llm-agent-providers llm-agent-customer-support; do
            gomod="$sister/go.mod"
            [ -f "$gomod" ] || { echo "::error::missing $gomod"; exit 1; }
            echo "--- inspecting $gomod ---"
            for r in "${!REPOS[@]}"; do
              # Skip self-pin (a repo cannot pin itself).
              [ "$sister" = "$r" ] && continue
              # Extract the pinned version for this sibling (may be in
              # `require ( ... )` block, may be `// indirect`).
              pinned=$(grep -E "^[[:space:]]*github\.com/costa92/$r[[:space:]]+v" "$gomod" \
                       | awk '{print $2}' | head -1)
              [ -z "$pinned" ] && continue  # sister does not depend on r
              latest="${LATEST[$r]}"
              if [ "$pinned" != "$latest" ]; then
                echo "::error::$sister pins github.com/costa92/$r at $pinned but latest is $latest"
                fail=1
              else
                echo "OK: $sister -> $r $pinned (current)"
              fi
            done
          done

          if [ "$fail" -ne 0 ]; then
            echo "::error::Dependency-currency gate FAILED — see annotations above."
            echo "To fix: bump the stale go.mod entries, refresh go.sum, and commit."
            exit 1
          fi
          echo "Dependency-currency gate PASSED — all sibling pins current."
```

Verification protocol for the gate step (35-01 will execute these):

- **Negative test (the gate fires red):** synthetically rewrite one sister's
  `go.mod` to pin a stale version (e.g. `llm-agent-otel`'s `llm-agent` line
  from `v0.5.0` to `v0.4.0`), run the step locally with `act` or in a draft
  PR — expect exit 1 with the readable annotation. Revert the synthetic
  change.
- **Positive test (the gate passes green):** with `go.mod` files at their
  current Phase-33 state, run the step — expect "all sibling pins current".

## 34-02 verification protocol

Phase 33 already proved each repo green individually (post-bump build + test
on each `main`). 34-02 is the *coordinated* re-verify with the gate now in
place. The verification is bash-only — no new tags cut, no commits made
outside the planning tree.

```bash
set -euo pipefail
ECO=/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem

# 1. Confirm every repo's `main` is at its Phase-33 tag (no drift since).
for r in llm-agent llm-agent-rag llm-agent-otel llm-agent-providers llm-agent-customer-support; do
  ( cd "$ECO/$r" && \
    head=$(git rev-parse HEAD) && \
    tag=$(git describe --tags --exact-match HEAD 2>/dev/null || echo "none") && \
    echo "$r: HEAD=$head tag=$tag" )
done
# Expect: llm-agent v0.5.0, llm-agent-rag v1.0.0, otel v0.2.0,
# providers v0.2.0, customer-support v0.2.0.

# 2. Per-repo clean build + test (GOWORK=off, exactly how test.yml runs).
for r in llm-agent llm-agent-rag llm-agent-otel llm-agent-providers llm-agent-customer-support; do
  echo "=== $r ==="
  ( cd "$ECO/$r" && \
    GOWORK=off GOCACHE=/tmp/go-build go vet ./... && \
    GOWORK=off GOCACHE=/tmp/go-build go build ./... && \
    GOWORK=off GOCACHE=/tmp/go-build go test -short ./... -count=1 )
done

# 3. Every tagged-branch go.mod is `replace`-free (INFRA-04 mirror).
for r in llm-agent llm-agent-otel llm-agent-providers llm-agent-customer-support; do
  ( cd "$ECO/$r" && \
    if grep -E '^replace|^[[:space:]]+replace' go.mod >/dev/null 2>&1; then
      echo "FAIL: $r has replace directives"; exit 1
    else
      echo "OK: $r no replace directives"
    fi )
done

# 4. Sibling pins are at current tags (mirrors the gate's logic, locally).
# Reuse the same loop from 34-01 — confirms gate passes green on current state.
```

GOPRIVATE note: **`llm-agent-rag` is now public** (operator-authorized in
Phase 33 to unblock cross-repo sister CI; see STATE.md). `GOPRIVATE` is no
longer required for fetching `llm-agent-rag@v1.0.0` — the default `GOPROXY`
serves it. The other four `costa92/*` repos are also public per their CI
runs. 34-02 therefore does not need to set `GOPRIVATE` at all; the
verification commands run with the OS default proxy.

Verification artifacts: capture the step output (all 5 build+test runs
green, gate step green) into the 34-02 SUMMARY. Nothing is committed by
34-02 itself.

## 34-03 audit doc structure

`v1.1-MILESTONE-AUDIT.md` lives at `llm-agent/.planning/v1.1-MILESTONE-AUDIT.md`
(the same location as `v1.0-MILESTONE-AUDIT.md`). The template *is* the
v1.0 audit; the v1.1 audit mirrors its sections one-for-one. Required
sections:

1. **Header** — title, audit date, milestone tag set (proposed table below),
   verdict (✅ PASS / ⚠️ PARTIAL / ❌ FAIL).
2. **Verification gate (re-run at audit time)** — re-run the 34-02 protocol
   commands; record exit codes, test counts, and any annotations. Mirrors
   v1.0's "all commands run in `<path>` ... | Command | Result |" table.
3. **Requirement-by-requirement verdict** — a 5-row table:

   | Req | Phase | Delivered artifact | Verdict |
   |-----|-------|--------------------|---------|
   | ECO-01 | 31 | core go.mod bumped to llm-agent-rag v1.0.0; the 7 facade-test `vector dimension mismatch` failures fixed; `go list -deps ./rag` lists zero third-party modules | ✅ |
   | ECO-02 | 32 | otel's `feat/otelrag-wrap-rag-system` merged to main; customer-support's `fix/pr-governance-auto-merge-permissions` merged to main; stale local branches pruned; every sister's main builds + tests green | ✅ |
   | ECO-03 | 33 | core v0.5.0 pushed; otel v0.2.0 / providers v0.2.0 / customer-support v0.2.0 pushed; zero `replace` directives anywhere; the umbrella consumes its own current sibling tags end-to-end | ✅ |
   | ECO-04 | 34-01 | umbrella.yml gained the dependency-currency gate; verified to fire red on a synthetic stale pin and green on the current coordinated tag set | ✅ |
   | ECO-05 | 34-02 + 34-03 | full 5-repo coordinated verification green (34-02); this audit doc + planning-tree updates + milestone archive (34-03); coordinated tag set recorded | ✅ |

4. **Keystone-decision compliance (KE-1..KE-7)** — one bullet per keystone,
   citing the artifact that proves compliance. Critical checks:
   - **KE-1** scope-is-alignment: no new feature in any repo (cross-check
     against git diffs phase by phase).
   - **KE-2** rag-untouched: `llm-agent-rag` HEAD remains at `v1.0.0` (`git
     describe --tags --exact-match` confirms no commits past the tag in
     v1.1's window).
   - **KE-3** core-stdlib-only: `cd llm-agent && go list -deps ./rag |
     grep -v '^github.com/costa92/' | sort -u` returns only stdlib modules.
   - **KE-4** branches-before-tags: each Phase-33 tag commit is descended
     from the merge commits Phase 32 produced (`git merge-base --is-ancestor`).
   - **KE-5** no-replace + coordinated tags: `grep -l replace go.mod`
     across 4 repos returns empty; the 4 coordinated tags exist.
   - **KE-6** gate-installed: `.github/workflows/umbrella.yml` contains the
     `Dependency-currency gate` step; it ran green on the audit-time PR.
   - **KE-7** live-Postgres-deferred: recorded as carry-forward; not
     touched.

5. **Coordinated tag set recorded** — a small table:

   | Repo | Pre-v1.1 | v1.1 tag | Tag SHA | Pushed |
   |---|---|---|---|---|
   | llm-agent | v0.4.0 | v0.5.0 | (fill at audit) | yes |
   | llm-agent-rag | v1.0.0 | v1.0.0 (unchanged) | a76896d | yes |
   | llm-agent-otel | v0.1.0 | v0.2.0 | 4dac44b | yes (PR #4) |
   | llm-agent-providers | v0.1.1 | v0.2.0 | 71d170b | yes (PR #7) |
   | llm-agent-customer-support | v0.1.0 | v0.2.0 | 7a9bc79 | yes (PR #5) |

6. **Findings** — informational notes only; no blocking findings expected.
   Examples likely to surface:
   - The umbrella gate is *core-PR-scoped*; sister-repo PRs are not gated
     individually (Decision 1 tradeoff). Recorded for future consideration.
   - `llm-agent-rag`'s back-edge `require llm-agent v0.4.0` remains as-is
     per KE-5 + KE-2 (bumping it would force a rag re-tag for cosmetic
     reasons; rag stays the untouched fixed point).

7. **Carry-forward debt** — restate from PROJECT.md:
   - Live-Postgres CI wiring (deferred since v0.5; KE-7).
   - Incremental community maintenance (deferred since v0.9 KG4-5).
   - `llm-agent-rag` deployment layer (HTTP service, CLI, caching) —
     deferred since v0.6.
   - Regex-based content safety (`guard`) is best-effort.
   - `EmbeddingEntityResolver` (v0.8) false-positive risk.
   - The refsvc demo remains intentionally demo-grade.

8. **Close steps (pending operator ask)** — mirrors v1.0's:
   1. Commit the v1.1 planning-tree changes (`.planning/v1.1-MILESTONE-AUDIT.md`
      created, PROJECT/STATE/ROADMAP/REQUIREMENTS updated, v1.1 archive
      moved into `.planning/milestones/`).
   2. `/gsd-transition` — archive the v1.1 ROADMAP/REQUIREMENTS to
      `.planning/milestones/`, mark traceability `Done`, move to
      between-milestones state.
   3. Push the close commit to `llm-agent` `main`.

Milestone close criteria (the audit may file PASS only if all hold):

- All 5 requirements `ECO-01..05` verified `Done`.
- All 7 keystones `KE-1..KE-7` honored with citable artifact.
- The 34-02 verification protocol re-ran green at audit time.
- No `replace` directive anywhere; every coordinated tag pushed and visible
  on the remote (`git ls-remote --tags origin v0.5.0` /`v0.2.0` /
  `v1.0.0`).

## Slice breakdown

Three slices, three waves (each waits on the previous — they are strictly
sequential by data dependency):

| Slice | Wave | Type | Repo(s) touched | Files modified | Purpose |
|---|---|---|---|---|---|
| **34-01** | 1 | execute | `llm-agent` | `.github/workflows/umbrella.yml` (and optionally `scripts/dep-currency-check.sh`) | Add the dependency-currency gate; verify red-fire on synthetic stale pin and green-pass on current state. Covers ECO-04. |
| **34-02** | 2 | execute | none (verification across all 5 checkouts) | none | Run the coordinated 5-repo verification protocol; capture output in SUMMARY. Depends on 34-01 (the gate must already pass green on current state before this slice trusts the coherence claim). Covers half of ECO-05. |
| **34-03** | 3 | execute | `llm-agent` (planning tree only) | `.planning/v1.1-MILESTONE-AUDIT.md` (create); `.planning/PROJECT.md`, `.planning/STATE.md`, `.planning/ROADMAP.md`, `.planning/REQUIREMENTS.md` (update); archive `v1.1-ROADMAP.md` + `v1.1-REQUIREMENTS.md` under `.planning/milestones/` | Write the milestone audit; close the milestone in planning. Depends on 34-02 (audit cites 34-02's green run). Covers the other half of ECO-05. |

Notes on the slice shape:

- **34-01** is the only slice that modifies CI. All other phases either
  modify Go code (31), git state (32, 33), or planning docs (34-03). The
  separation keeps the gate's diff small and reviewable.
- **34-02** is verification-only — no `git add`, no commits in any repo.
  The slice's deliverable is a verification log that 34-03 cites.
- **34-03** is planning-tree-only — no Go code, no CI. Its commit is the
  milestone-close commit in the core repo, the same shape as v1.0's
  `48cbbc9` close commit.

## Out of scope

- **Bumping any tag.** Phase 33 already cut the coordinated tag set. Phase
  34 *observes* the tags; it does not produce new ones. No `v0.5.1` core,
  no `v0.2.1` sister tags, no rag re-tag.
- **Live-Postgres CI wiring** (KE-7). Genuine carry-forward debt, but a CI
  capability project of its own size. Phase 34 records the deferral in the
  audit; it does not pick up the work.
- **Per-sister-repo dep-currency gates.** Decision 1 places the gate in the
  umbrella only. Extending it to per-repo CI is a defensible follow-up but
  is *not* in v1.1 scope.
- **Productionizing the `customer-support` demo.** A standing non-goal;
  v1.1 has only ever aligned its deps.
- **Any change to `llm-agent-rag`.** KE-2 — rag is the untouched fixed
  point. Audit observes it; nothing more.
- **K8s / Helm packaging.** Standing hard-rule non-goal.

## Open questions

1. **The rag back-edge: `llm-agent-rag` `require llm-agent v0.4.0` (one
   major + a minor stale of core's now-released `v0.5.0`).** KE-2 + KE-5
   left this as-is (bumping it would force a rag re-tag for cosmetic
   reasons; rag stays untouched). But the *dependency-currency gate* by
   Decision 3 would flag this as stale — `rag/go.mod` pins `llm-agent
   v0.4.0`, latest is `v0.5.0`. Two options:

   - **(a) Exempt rag from the gate entirely** — drop `llm-agent-rag` from
     the gate's sister loop. Easiest; preserves KE-2 absolutely; the
     back-edge stays as-is until a future rag milestone bumps it
     organically. Recommended.
   - **(b) Force a rag re-tag** to bring its back-edge current. Violates
     KE-2. Not recommended.
   - **(c) Special-case rag in the gate as "back-edge allowed up to N
     versions stale"** — too much policy in CI for a one-off; rejected.

   **Default recommendation: option (a).** The gate's sister loop should
   skip `llm-agent-rag` for its back-edge entry. This is a one-line
   skip in the bash and is documented in the gate's comments + in the
   34-03 audit's "Findings" section. The discuss-phase or planner should
   confirm this before 34-01 ships.

2. **Should `scripts/dep-currency-check.sh` ship alongside the CI step?**
   The umbrella has a precedent (`scripts/workspace.sh` is the local
   developer companion to the umbrella's `go work init` step). A
   `scripts/dep-currency-check.sh` would let a maintainer test-run the
   gate locally before opening a PR. **Recommended yes**, mirrored
   byte-identically across the 4 core repos via the same `INFRA-03`
   sha256 enforcement that `scripts/workspace.sh` already uses. The
   planner can fold this into 34-01 as an additive task; it is not
   load-bearing for the gate itself.

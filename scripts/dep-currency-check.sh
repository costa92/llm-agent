#!/usr/bin/env bash
# scripts/dep-currency-check.sh — strict ecosystem dependency-currency gate.
#
# Closes the test.yml (stale dep, invisible) vs umbrella.yml (rag master, hides
# staleness) blind spot that let the v1.1 drift accumulate silently.
# ECO-04 / KE-6 / RESEARCH 34 "The blind spot the gate closes".
#
# Usage:
#   - From CI (.github/workflows/umbrella.yml): `bash scripts/dep-currency-check.sh`
#     The umbrella workflow checks out all 5 sibling repos adjacent to this one.
#   - Locally: `cd <ecosystem-root> && bash llm-agent/scripts/dep-currency-check.sh`
#     where <ecosystem-root> is the parent dir containing all 5 sibling clones.
#
# Resolution:
#   ECOSYSTEM_ROOT defaults to the parent of this script's repo
#   (`<this-script>/../../` ⇒ `<ecosystem-root>/llm-agent/scripts/.. ⇒ <ecosystem-root>/llm-agent`,
#   then up one more level). Override by exporting ECOSYSTEM_ROOT explicitly.
#
# Stdlib-only bash + git. No new dependency in the core repo.
set -euo pipefail

ECOSYSTEM_ROOT=${ECOSYSTEM_ROOT:-$(cd "$(dirname "$0")/.." && cd .. && pwd)}
echo "ECOSYSTEM_ROOT=$ECOSYSTEM_ROOT"

# The 5 ecosystem repos. The list is the single source of truth; an explicit
# override (e.g. exempting a back-edge) must be an auditable diff to this list.
REPOS=("llm-agent" "llm-agent-rag" "llm-agent-otel" "llm-agent-providers" "llm-agent-customer-support")

# 1. Build the latest-tag map by querying each remote.
declare -A LATEST
for r in "${REPOS[@]}"; do
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

# 2. For each checked-out sister, parse its go.mod for sibling pins and
#    compare against LATEST. Strict equality per RESEARCH 34 §Decision 3.
fail=0
for sister in "${REPOS[@]}"; do
  gomod="$ECOSYSTEM_ROOT/$sister/go.mod"
  if [ ! -f "$gomod" ]; then
    echo "::error::missing $gomod"
    exit 1
  fi
  echo "--- inspecting $gomod ---"
  for r in "${REPOS[@]}"; do
    # Skip self-pin (a repo cannot pin itself).
    [ "$sister" = "$r" ] && continue
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
    # Extract the pinned version for this sibling. The grep handles BOTH forms:
    #   - block form:        `\tgithub.com/costa92/<r> vX.Y.Z` (inside require())
    #   - single-line form:  `require github.com/costa92/<r> vX.Y.Z`
    # Indirect `// indirect` lines are included (per RESEARCH 34 "go.mod //
    # indirect lines"). The // indirect comment does not exempt.
    # `awk '{print $(NF-1)}'` picks the version regardless of leading tokens
    # (`require` vs. nothing). `|| true` tolerates "sister does not depend on r"
    # under `set -o pipefail`.
    pinned=$(grep -E "(^|^require[[:space:]]+|^[[:space:]]+)github\.com/costa92/$r[[:space:]]+v" "$gomod" \
             | awk '{for(i=1;i<=NF;i++) if($i ~ /^v[0-9]+\./) {print $i; exit}}' | head -1 || true)
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

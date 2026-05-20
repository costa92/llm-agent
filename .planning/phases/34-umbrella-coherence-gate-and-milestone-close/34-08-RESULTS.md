# Wave 8 — Coordinated 5-repo verification results

**Verification date:** 2026-05-20
**Ecosystem path:** `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/`
**Operator:** costa (costalong92@gmail.com)
**Slice:** 34-08 (Wave 8, post-cascade coordinated-tag-set verification, ECO-05)

This is the evidence base that the milestone-close audit (Wave 9) cites. Pure
verification: zero `git add`, zero commits, zero pushes, zero source edits in
any sister repo. The only file written is this one, inside the planning tree.

> **Note on naming.** The PLAN frontmatter calls this `34-06-RESULTS.md`
> because it was authored before the cascade was complete. The cascade promoted
> this to Wave 8; the executor wrote it as `34-08-RESULTS.md` to match the
> wave numbering used by 34-06 (cs follow-up) and 34-07 (umbrella gate).
> Same file, same evidence — just renamed to match the live wave layout.

## Coordinated tag set

The post-cascade coordinated tag set (supersedes the PLAN body's stale list of
`v0.5.0 / v1.0.1 / v0.2.0 x3`). Every HEAD listed matches what the orchestrator's
spawn message named as the "ACTUAL coordinated tag set after Phase 34 cascade".

| Repo                          | Branch | HEAD SHA   | Tag at HEAD                              |
| ----------------------------- | ------ | ---------- | ---------------------------------------- |
| llm-agent                     | main   | `acb3253`  | (1 commit past `v0.5.1` — see note)      |
| llm-agent-rag                 | master | `09697ca`  | `v1.0.1`                                 |
| llm-agent-otel                | main   | `c7ebda7`  | `v0.2.1`                                 |
| llm-agent-providers           | main   | `efdef5a`  | `v0.2.1`                                 |
| llm-agent-customer-support    | main   | `ca62e5b`  | `v0.2.2`                                 |

**core-1-commit-ahead caveat (expected, not drift):**
`llm-agent`'s coordinated tag is `v0.5.1` pointing at `88db43e` (the Wave 2
commit). The current HEAD is `acb3253` — Wave 7's umbrella-gate commit (the
shipped `.github/workflows/umbrella.yml` + `scripts/dep-currency-check.sh`).
That gate commit lands in-tree without bumping a new core tag because the gate
is build-system infrastructure (no behavior change to consumers of the core
module). The v1.1 coordinated tag for core is therefore `v0.5.1` as designed,
and every sister repo pins `llm-agent v0.5.1` — confirmed green by the gate
output below. Recognized in the spawn message; not flagged as drift.

All other repos are exact-at-tag.

## Per-repo build/test results

Per-repo: `cd $repo && GOWORK=off GOCACHE=/tmp/go-build go vet ./... && go build ./... && go test -short ./... -count=1`.

| Repo                          | vet | build | test -short | Test summary                                                |
| ----------------------------- | --- | ----- | ----------- | ----------------------------------------------------------- |
| llm-agent                     | 0   | 0     | 0           | 14 packages OK (root, bench, builtin, comm*, context, llm, memory, orchestrate, pkg/fanout, rag, rl); 1 `?` (`internal/testenv`) |
| llm-agent-rag                 | 0   | 0     | 0           | 19 packages OK; 3 `?` (root, generate, store/storetest)     |
| llm-agent-otel                | 0   | 0     | 0           | 6 packages OK (root, otelagent, otelmetrics, otelmodel, otelrag, otelslog); 1 `?` (compose/demo) |
| llm-agent-providers           | 0   | 0     | 0           | 6 packages OK (anthropic, deepseek, internal/contract, minimax, ollama, openai) |
| llm-agent-customer-support    | 0   | 0     | 0           | 9 packages OK (cmd/server, compose, internal/{app,config,guardrails,httpapi,limits,sessionstore,supportflow}); 1 `?` (internal/providers) |

All 5 repos: **green across vet + build + test -short** under `GOWORK=off` with
a fresh `/tmp/go-build` cache.

## `replace` directive scan

`grep -E '^replace|^[[:space:]]+replace' $repo/go.mod` — zero matches across
all 5 repos. Confirms no `replace` escape hatches leaked into any tagged branch.

- OK: llm-agent no replace
- OK: llm-agent-rag no replace
- OK: llm-agent-otel no replace
- OK: llm-agent-providers no replace
- OK: llm-agent-customer-support no replace

## Dep-currency gate run

Invocation: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem && bash llm-agent/scripts/dep-currency-check.sh`.

Script source: `llm-agent/scripts/dep-currency-check.sh` (shipped in Wave 7,
single source of truth used by both local verification here and CI in
`.github/workflows/umbrella.yml`).

**Exit code: `0`** (PASS).

```text
ECOSYSTEM_ROOT=/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem
latest(llm-agent) = v0.5.1
latest(llm-agent-rag) = v1.0.1
latest(llm-agent-otel) = v0.2.1
latest(llm-agent-providers) = v0.2.1
latest(llm-agent-customer-support) = v0.2.2
--- inspecting /home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/go.mod ---
OK: llm-agent -> llm-agent-rag v1.0.1 (current)
--- inspecting /home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent-rag/go.mod ---
SKIP: rag back-edge to core (cycle exemption — KE-2 corollary)
--- inspecting /home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent-otel/go.mod ---
OK: llm-agent-otel -> llm-agent v0.5.1 (current)
OK: llm-agent-otel -> llm-agent-rag v1.0.1 (current)
--- inspecting /home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent-providers/go.mod ---
OK: llm-agent-providers -> llm-agent v0.5.1 (current)
--- inspecting /home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent-customer-support/go.mod ---
OK: llm-agent-customer-support -> llm-agent v0.5.1 (current)
OK: llm-agent-customer-support -> llm-agent-rag v1.0.1 (current)
OK: llm-agent-customer-support -> llm-agent-otel v0.2.1 (current)
OK: llm-agent-customer-support -> llm-agent-providers v0.2.1 (current)
Dependency-currency gate PASSED — all sibling pins current.
```

Strict-currency matrix verified: every cross-repo pin matches the latest tag of
its target repo. The `rag → core` back-edge is correctly skipped under the
KE-2 cycle exemption (rag's dependency on core is intentional and ratchets
forward separately to avoid a circular pin between the v1.1 coordinated tags).

## Working-tree cleanliness

Per task spec, the 4 sister repos must stay clean (no source-code edits
leaked). Confirmed:

- CLEAN: llm-agent-rag
- CLEAN: llm-agent-otel
- CLEAN: llm-agent-providers
- CLEAN: llm-agent-customer-support

`llm-agent` has GSD planning-tree changes (`.planning/PROJECT.md`,
`.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md`, `.planning/STATE.md`,
plus untracked `.planning/phases/31-…` through `34-…` and a research summary).
These are **planning-tree only** — not Go source, not vendored anywhere, not
part of any tagged module. The PLAN's `<verify>` block specifically excludes
`llm-agent` from the tree-clean check for exactly this reason. Expected and
correct.

## Verdict

**✅ PASS — coordinated tag set internally consistent across all 5 repos.**

- 5/5 repos at the post-cascade coordinated tags (core 1-commit ahead with
  the gate-in-tree caveat, recognized and not drift).
- 5/5 green on vet + build + test -short under `GOWORK=off`.
- 5/5 free of `replace` directives.
- Dep-currency gate exit 0 with every pin reported `(current)` and the
  cycle exemption correctly applied.
- No source-code edits leaked into any sister repo's working tree.

The v1.1 coordinated tag set is internally consistent end-to-end. Wave 9
(milestone audit + close) may cite this document as the evidence base.

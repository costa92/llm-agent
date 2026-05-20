# Roadmap: llm-agent

**Last updated:** 2026-05-20
**Current state:** `v1.2` Core Capability Deepening milestone — opened
2026-05-20; Phases 35-38 not yet planned. The umbrella's first
**core-feature** milestone since v0.3. v1.1 (ecosystem alignment)
shipped and closed 2026-05-20 (audit ✅ PASS 5/5).
**Active scope:** `v1.2` — Core Capability Deepening (theme: **Core
v0.6** — capability additions to core `llm-agent`; memory tiering
deferred to v1.3). Core module bump: `v0.5.1 → v0.6.0` (minor —
additive).

## Archived Milestones

- [x] **v0.3: Deployable multi-repo release** — shipped 2026-05-12.
  Delivered `llm/v2`, three real provider adapters, OTel wrappers, and the
  customer-support demo stack across 4 repos.
  - Archive: `.planning/milestones/v0.3-ROADMAP.md`
  - Requirements archive: `.planning/milestones/v0.3-REQUIREMENTS.md`
  - Audit: `.planning/v0.3-MILESTONE-AUDIT.md`
- [x] **v0.5: RAG productionization and standalone SDK evolution** — shipped
  2026-05-15. Structure-aware retrieval, a PostgreSQL + pgvector backend
  with a shared conformance suite, tracing hooks, an evaluation framework, a
  feedback loop, and cross-repo contract gates. `llm-agent-rag` tagged
  `v0.2.0`.
  - Archive: `.planning/milestones/v0.5-ROADMAP.md`
  - Requirements archive: `.planning/milestones/v0.5-REQUIREMENTS.md`
- [x] **v0.6: Production-grade retrieval quality and safety** — shipped
  2026-05-18. Six phases (14-19): BM25 lexical retrieval + RRF fusion,
  model-based reranking with explainability, the RAG Triad, cost/latency
  observability (`obs` + `otelrag` RED metrics), content safety (`guard`),
  and agentic retrieval (`MultiHopRetriever` + `CorrectiveAsker`).
  `llm-agent-rag` tagged `v0.3.0`; no new dependency.
  - Archive: `.planning/milestones/v0.6-ROADMAP.md`
  - Requirements archive: `.planning/milestones/v0.6-REQUIREMENTS.md`
  - Audit: `.planning/v0.6-MILESTONE-AUDIT.md`
- [x] **v0.7: GraphRAG — relationship-traversal retrieval** — shipped
  2026-05-19. Three phases (20-22): Tier-1 lightweight GraphRAG.
  `llm-agent-rag` tagged `v0.4.0`; no new dependency, no graph database.
  - Archive: `.planning/milestones/v0.7-ROADMAP.md`
  - Requirements archive: `.planning/milestones/v0.7-REQUIREMENTS.md`
  - Audit: `.planning/v0.7-MILESTONE-AUDIT.md`
- [x] **v0.8: GraphRAG Tier-3 — communities, global search, fuzzy
  resolution** — shipped 2026-05-20. Three phases (23-25): community
  detection, lazy summaries, `AskGlobal` map-reduce global search, fuzzy
  entity resolution. `llm-agent-rag` tagged `v0.5.0`; no new dependency.
  - Archive: `.planning/milestones/v0.8-ROADMAP.md`
  - Requirements archive: `.planning/milestones/v0.8-REQUIREMENTS.md`
  - Audit: `.planning/v0.8-MILESTONE-AUDIT.md`
- [x] **v0.9: GraphRAG refinements — DRIFT search and path-ranking** —
  shipped 2026-05-20. Two phases (26-27): path-ranked subgraph evidence and
  DRIFT hybrid search. `llm-agent-rag` tagged `v0.6.0`; no new dependency.
  - Archive: `.planning/milestones/v0.9-ROADMAP.md`
  - Requirements archive: `.planning/milestones/v0.9-REQUIREMENTS.md`
  - Audit: `.planning/v0.9-MILESTONE-AUDIT.md`
- [x] **v1.0: API stabilization and the compatibility promise** — shipped
  2026-05-21. Three phases (28-30): a written exported-surface audit and
  the pre-freeze breaking renames (`eval.Evaluator`→`RetrievalEvaluator`,
  `eval.Result`→`RetrievalResult`; the `ragkit` root repurposed as a
  documented doc-anchor); full package + exported-symbol doc-comment
  coverage and a written `docs/compatibility.md` Go-module compatibility
  promise; a pure-stdlib `internal/apisnapshot` exported-surface gate
  (`api/v1.snapshot.txt` + a `go test` regeneration-diff) plus a
  `-tags llmagent` CI step. `llm-agent-rag` tagged `v1.0.0`; no new
  dependency, no behavior change.
  - Archive: `.planning/milestones/v1.0-ROADMAP.md`
  - Requirements archive: `.planning/milestones/v1.0-REQUIREMENTS.md`
  - Audit: `.planning/v1.0-MILESTONE-AUDIT.md`
- [x] **v1.1: Ecosystem Alignment** — shipped and closed 2026-05-20.
  Four phases (31-34, Phase 34 expanded to 9 slices mid-flight): core
  RAG facade re-alignment to `llm-agent-rag v1.0.0` (the 7-test
  `vector dimension mismatch` regression fixed inside the facade
  adapters); sister-repo branch landing (`otel`'s `otelrag` feature +
  `customer-support`'s CI-fix); coordinated dependency-bump + re-tag
  wave; umbrella dep-currency CI gate. Final coordinated tag set
  (post-cascade): `llm-agent v0.5.1`, `llm-agent-rag v1.0.1`,
  `llm-agent-otel v0.2.1`, `llm-agent-providers v0.2.1`,
  `llm-agent-customer-support v0.2.2`. Zero `replace` directives. 5/5
  requirements (`ECO-01..05`); audit ✅ PASS.
  - Archive: `.planning/milestones/v1.1-ROADMAP.md`
  - Requirements archive: `.planning/milestones/v1.1-REQUIREMENTS.md`
  - Audit: `.planning/v1.1-MILESTONE-AUDIT.md`

## Milestone v1.2: Core Capability Deepening

**Goal**: take core `llm-agent` to **v0.6.0** with three additive
agent-runtime governance primitives — budget/cancellation, policy/safety
middleware, and a multi-agent `Supervisor` — all stdlib-only, all
purely additive (no `/v2` import path, no edit to validated public
types). The umbrella's first **core-feature** milestone since v0.3.

**Theme**: **Core v0.6** — capability additions to core `llm-agent`;
memory tiering deferred to v1.3 per KC-2.

**Repos**: `llm-agent` (core only). `llm-agent-rag` stays a fixed point
(KS-5 freeze); the three sister repos stay on their v0.2.x tracks
(post-v1.2 ecosystem-alignment task will bump them).

**Requirements in scope**: `CC-1..04`

**Keystone decisions** (ratified from
`.planning/research/v1.2-core-capability-deepening-SUMMARY.md`):

- **KC-1** — Multi-agent coordination lives in `orchestrate/` as
  `orchestrate.Supervisor` — a thin facade over `StateGraph[S]`. Not a
  new `agents/coord` package; workers are `agents.Agent` (composable).
- **KC-2** — Memory tiering is **OUT of v1.2 scope** — deferred to
  v1.3. The reframed shape (`ScopedMemory` decorator + ctx-keyed scope
  propagation via `memory.WithScope`) is pre-decided in the v1.2
  research so v1.3 doesn't relitigate.
- **KC-3** — Policy middleware is a `policy` package — a
  capability-preserving `policy.Wrap(model) ChatModel` decorator
  mirroring `otelmodel.Wrap` (K3); typed `Gate` event union; sentinel
  `ErrBlocked`; 3 built-in regex gates (PII redaction, injection
  detection, max-input-length); documented stack
  `policy.Wrap(otelmodel.Wrap(provider))`.
- **KC-4** — Budget/cancellation is **ctx-keyed for propagation + a
  `budget.Tracker` interface for enforcement**; single integration
  chokepoint at `generateFromPrompt` in `agents/`; built-in trackers
  `NewStrict` (deny on exhaustion) and `NewSoft` (warn-only);
  `Budget.Cost` plumbing only — provider→$ cost-table is **opt-in /
  outside core**. `MaxSteps` etc. coexist with `Budget.Calls`.
- **KC-5** — Breaking-change avoidance: every new surface is in a
  *new* package or a *new* optional interface; no edit to
  `llm.ChatModel`, `agents.Agent`, `memory.Memory`, or
  `orchestrate.NodeFunc[S]`; **no `/v2` import path**; v0.5.1 → v0.6.0
  is a **minor (additive)** bump — existing v0.5.1 callers compile
  unchanged against v0.6.0.

### Phase 35: Budget / cancellation context

**Status**: not started

**Goal**: `budget` package exists, propagates through `ctx`, and
enforces token/call/wall-clock/cost budgets at the
`generateFromPrompt` chokepoint; every existing agent paradigm honors
it with zero behavior change when no budget is set. Cost-table is
opt-in / outside core.

**Depends on**: Phase 34 complete (v1.1 audit PASS)

**Repos**: `llm-agent` (core only)

**Requirements covered**: `CC-1`

**Planned work**:

- `35-01..05` — `budget` package skeleton (`Budget`, `Tracker`,
  `WithBudget`, `From`, `NewStrict`, `NewSoft`); integrate into
  `generateFromPrompt` (charge before + after every LLM call; honor
  `ctx.Err()`; preserve `MaxSteps`/`MaxTurns`); streaming integration
  (per `EventDone.Usage`); example + docs; exit gate (core stdlib-only,
  no edit to `llm.ChatModel` / `agents.Agent`).

### Phase 36: Policy / safety middleware

**Status**: not started

**Goal**: `policy` package ships as a capability-preserving
`llm.ChatModel` decorator (mirrors `otelmodel.Wrap`); 3 built-in gates
(PII redaction, injection detection, max-input-length); audit log via
`OnDecision`; composes with `otelmodel.Wrap`.

**Depends on**: Phase 35

**Repos**: `llm-agent` (core only)

**Requirements covered**: `CC-2`

**Planned work**:

- `36-01..05` — `policy` package skeleton (`Wrap`, typed `Gate` event
  union, `Decision`, `ErrBlocked`); 3 built-in gates lifted from
  `llm-agent-rag/guard` regex patterns (separate file, no rag import);
  compose-with-otel integration test; example; exit gate
  (capability-preserving assertions pass, core stdlib-only).

### Phase 37: Multi-agent coordination (`orchestrate.Supervisor`)

**Status**: not started

**Goal**: iterative supervisor↔worker primitive ships as
`orchestrate.Supervisor` — a thin facade over `StateGraph[S]` — honoring
budget (CC-1) and policy (CC-2). Workers are `agents.Agent`
(composable).

**Depends on**: Phase 35 + Phase 36

**Repos**: `llm-agent` (core only)

**Requirements covered**: `CC-3`

**Planned work**:

- `37-01..04` — `Supervisor` design + skeleton (`NewSupervisor`,
  `SupervisorOptions{ Planner, Workers, MaxRounds, ParseDispatch,
  BuildAggregate }`); budget + policy integration (rounds against
  `Budget.Calls`; ctx-propagation to workers; documented policy-wrapping
  pattern); example (`examples/supervisor/`) + compose-with-`StateGraph`
  tests; exit gate (core stdlib-only, no edit to `agents.Agent`).

### Phase 38: v1.2 milestone audit + close

**Status**: not started

**Goal**: core `v0.6.0` tagged; CHANGELOG entry; planning docs archived
to `.planning/milestones/v1.2-*.md`; milestone audited
(`.planning/v1.2-MILESTONE-AUDIT.md`); PROJECT/STATE/ROADMAP/REQUIREMENTS
refreshed to between-milestones.

**Depends on**: Phase 35 + Phase 36 + Phase 37 all green

**Repos**: `llm-agent` (core `.planning/` tree)

**Requirements covered**: `CC-4`

**Planned work**:

- `38-01..04` — verification (`go vet ./... && go test ./... && go
  list -deps ./...`); tag `llm-agent v0.6.0`; CHANGELOG; milestone
  audit; archive v1.2 ROADMAP/REQUIREMENTS; refresh active planning
  artifacts. Close commit is operator-gated.

## Milestone v1.1: Ecosystem Alignment — ✅ shipped 2026-05-20 (audit PASS)

**Goal**: make the umbrella's headline claim — "four coordinated repos plus
a standalone RAG SDK" — true at the *release* level. Bring the core
`llm-agent` and the three sister repos current with `llm-agent-rag v1.0.0`
and cut each a coherent, stable, tagged baseline. An **alignment /
housekeeping milestone — no new features**. An align → land → re-tag →
gate arc.

**Final coordinated tag set** (v1.1, post-cascade): `llm-agent v0.5.1`,
`llm-agent-rag v1.0.1`, `llm-agent-otel v0.2.1`,
`llm-agent-providers v0.2.1`, `llm-agent-customer-support v0.2.2`.

**Archive references** (pending operator milestone-close commit):

- Roadmap: `.planning/milestones/v1.1-ROADMAP.md`
- Requirements: `.planning/milestones/v1.1-REQUIREMENTS.md`
- Audit: `.planning/v1.1-MILESTONE-AUDIT.md`

**Repos**: `llm-agent` (core), `llm-agent-otel`, `llm-agent-providers`,
`llm-agent-customer-support`. `llm-agent-rag` is the untouched fixed point.

**Requirements in scope**: `ECO-01..05`

**Keystone decisions** (ratified from `.planning/research/v1.1-ecosystem-alignment-SUMMARY.md`):

- **KE-1** — scope is **alignment, not features**. v1.1 ships no new
  feature in any repo. Landing the already-written `otelrag` feature
  (it merely needs merging) is alignment, not new work.
- **KE-2** — `llm-agent-rag` is the **fixed point and is not touched**.
  Its frozen v1.0.0 API and its `internal/apisnapshot`/`contract` gates
  are inviolate; no rag re-tag in v1.1. Every other repo aligns *to* rag.
- **KE-3** — the core RAG facade is repaired **and proven stdlib-only**.
  The `vector dimension mismatch` regression is fixed inside the facade
  adapters — never by adding a dependency, never by changing rag. Exit
  gate: build + test green, `go list -deps ./rag` lists zero third-party
  modules, `go.sum` holds only the `llm-agent-rag` lines.
- **KE-4** — **branches land before tags**. No repo is re-tagged from a
  `main` that does not reflect its real state; the stranded `otel` and
  `customer-support` branches merge to `main` first.
- **KE-5** — a **coordinated, dependency-ordered tag wave**, no `replace`
  directives. Bumps flow core → otel → providers → customer-support;
  proposed tags `llm-agent v0.5.0`, `llm-agent-otel v0.2.0`,
  `llm-agent-providers v0.2.0`, `llm-agent-customer-support v0.2.0`.
  **Ratified:** `llm-agent-rag`'s back-edge `require llm-agent v0.4.0` is
  **left as-is** — bumping it would force a re-tag of the frozen rag
  v1.0.0 for a cosmetic change; rag stays untouched (KE-2).
- **KE-6** — the umbrella CI gains a **dependency-currency gate**. The
  drift went unnoticed because `test.yml` pins the stale dep while
  `umbrella.yml` checks out rag master; v1.1 adds a check that fails when
  a sister `go.mod` lags a sibling's latest tag.
- **KE-7** — **live-Postgres CI wiring stays deferred**. Genuine
  carried-forward debt, but an infra project of independent size;
  bundling it would break the tight-alignment discipline.

## Active Forward Work

### Phase 31: Core RAG facade re-alignment to `llm-agent-rag v1.0.0`

**Status**: ✅ complete — 3/3 slices. Core `go.mod` bumped to
`llm-agent-rag v1.0.0`; the facade `storeAdapter.List` `nil`-vector
enumeration hack replaced with a real list route (an additive
`*InMemoryStore.ListDocuments` + an optional `lister` interface + an
id-index fallback); all 7 `vector dimension mismatch` failures fixed.
Full suite green; the core is proven stdlib-only (`./rag` resolves to
only the two `github.com/costa92/*` modules); the cross-repo contract
gate compiles. No new dependency; the public `VectorStore` interface
unchanged. Core not re-tagged — that is Phase 33.

**Goal**: core `llm-agent` builds and tests green against rag `v1.0.0`,
the facade behaves identically to callers, and the core stays provably
stdlib-only.

**Depends on**: v1.0 milestone complete (`llm-agent-rag v1.0.0` tagged)

**Repos**: `llm-agent`

**Requirements covered**: `ECO-01`

**Planned work**:

- `31-01` diagnose the `vector dimension mismatch` — diff the v0.1.4 →
  v1.0.0 `store.InMemoryStore` / embedding dimension contract; pinpoint
  where the facade adapter wires a mismatched dimension
- `31-02` bump `go.mod` to `llm-agent-rag v1.0.0`, refresh `go.sum`,
  repair the facade adapters (`rag/store.go`, `rag/embedder.go`,
  `rag/rag.go`) so all 7 failing facade tests pass
- `31-03` verify gate: `go vet ./... && go test ./...` green;
  `go list -deps ./rag` proves zero third-party modules; `go.sum` minimal;
  `rag/contract_test.go` compiles; facade migration notes updated

### Phase 32: Sister-repo branch landing & hygiene

**Status**: ✅ complete — 3/3 slices. `otel`'s `otelrag` RAG-wrapping
feature merged into local `main` (merge `2333295`, conflict-free);
`customer-support` and `providers` synced to a current `main`. Each
repo's diverged local `main` (a patch-identical duplicate governance
commit) was reset to `origin/main` after `git cherry` proved zero unique
work. Stale local branches pruned in all three. Every `main` builds +
tests green. Local git work only — nothing pushed, no tag (KE-4); the
push wave is the milestone-close action.

**Goal**: every sister repo's `main` reflects its true current state;
stale branches pruned; nothing tagged yet.

**Depends on**: Phase 31

**Repos**: `llm-agent-otel`, `llm-agent-customer-support`,
`llm-agent-providers`

**Requirements covered**: `ECO-02`

**Planned work**:

- `32-01` `llm-agent-otel` — merge `feat/otelrag-wrap-rag-system` → `main`
  (the 4-commit `otelrag` feature); build/test green on `main`; prune
  stale governance branches
- `32-02` `llm-agent-customer-support` — merge
  `fix/pr-governance-auto-merge-permissions` → `main` (2 CI-fix commits);
  build/test green on `main`; prune stale branches
- `32-03` `llm-agent-providers` — confirm `main` clean, the untagged
  deepseek/minimax adapter work present and green; branch-hygiene pass

### Phase 33: Coordinated dependency-bump & re-tag wave

**Status**: ✅ complete — 4/4 slices. All four repos shipped:
`llm-agent v0.5.0` (direct push, no protection); `llm-agent-otel v0.2.0`
(PR #4 — bumps llm-agent + llm-agent-rag); `llm-agent-providers v0.2.0`
(PR #7); `llm-agent-customer-support v0.2.0` (PR #5 — bumps all three).
Two architectural deviations operator-resolved: the sister repos require
PR + CI on `main` (branch protection) — Phase-32 PR-merge precedent
followed; and the cross-repo CI auth gap was unblocked by flipping
`llm-agent-rag` to public visibility (matches the v1.0 docs/compatibility.md
public-SDK positioning + the sister-repo visibility pattern). Zero
`replace` directives anywhere.

**Goal**: all four repos consume current sibling tags; coordinated stable
tags cut in dependency order.

**Depends on**: Phase 32

**Repos**: `llm-agent`, `llm-agent-otel`, `llm-agent-providers`,
`llm-agent-customer-support`

**Requirements covered**: `ECO-03`

**Planned work**:

- `33-01` re-tag core `llm-agent v0.5.0` from `main` (the v1.0.0-aligned
  facade)
- `33-02` `llm-agent-otel` — bump `llm-agent-rag v0.3.0 → v1.0.0`
  (optionally `llm-agent → v0.5.0`), refresh `go.sum`, verify, tag `v0.2.0`
- `33-03` `llm-agent-providers` — optionally bump `llm-agent → v0.5.0`,
  verify, tag `v0.2.0` (captures the deepseek/minimax adapters)
- `33-04` `llm-agent-customer-support` — bump `llm-agent-otel → v0.2.0`
  and `llm-agent-providers → v0.2.0` (optionally `llm-agent → v0.5.0`),
  refresh `go.sum`, verify, tag `v0.2.0`; confirm no `replace` directives

### Phase 34: Umbrella coherence gate & milestone close

**Status**: ✅ complete (audit ✅ PASS, 2026-05-20)

**Goal**: dependency drift fails CI in future; the umbrella is provably
coherent; milestone audited and closed.

**Depends on**: Phase 33

**Repos**: `llm-agent` (umbrella CI), all repos (verification)

**Requirements covered**: `ECO-04`, `ECO-05`

**Expansion note.** Phase 34 expanded from its originally-planned 3 slices
to **9 slices** mid-flight, driven by two operator-confirmed trade-offs the
cascade surfaced: (i) the strict dep-currency gate requires a coordinated
cascade of patch tags to bring every consumer current with the back-edge
refresh (`rag v1.0.1`), and (ii) the original Phase-33 cascade had a
topological-order miss (`customer-support` was tagged before `providers`)
which forced a Wave 6 follow-up re-tag of `cs v0.2.2`. The trade-offs are
documented honestly in `.planning/v1.1-MILESTONE-AUDIT.md` §Trade-offs.

**Planned work** (9-slice expanded layout):

- `34-01` `llm-agent-rag` back-edge bump + tag `v1.0.1` (rag → core
  v0.5.0; KE-2 holds — chore-only patch, no exported-symbol move)
- `34-02` `llm-agent` cascade — bump `rag@v1.0.1`, tag `v0.5.1`
- `34-03` `llm-agent-otel` cascade — bump rag+core, tag `v0.2.1` (PR #5)
- `34-04` `llm-agent-customer-support` cascade — bump rag+core+otel,
  tag `v0.2.1` (PR #6)
- `34-05` `llm-agent-providers` transitive cascade — bump core, tag
  `v0.2.1` (PR #8)
- `34-06` `llm-agent-customer-support` cascade follow-up — bump providers,
  tag `v0.2.2` (PR #7; topology-order fix)
- `34-07` umbrella dependency-currency CI gate
  (`scripts/dep-currency-check.sh` + `.github/workflows/umbrella.yml` step)
  with the one auditable rag↔core cycle exemption (covers ECO-04)
- `34-08` coordinated 5-repo verification — vet/build/test green, no
  `replace` directives, dep-currency gate exit 0, sister working trees
  clean (covers ECO-05; evidence `34-08-RESULTS.md`)
- `34-09` milestone audit + close — `.planning/v1.1-MILESTONE-AUDIT.md`,
  planning-doc updates, archive `ROADMAP`/`REQUIREMENTS` to
  `.planning/milestones/v1.1-*.md`, record the coordinated tag set
  (covers ECO-05; this slice; close commit is operator's explicit move)

## Known Carry-forward Debt

- Live-Postgres CI wiring (testcontainers-go or GH Actions services) is
  still pending from v0.5; the Phase 14/21/23-24 `postgres` paths need
  verification against a live database.
- Incremental community maintenance is deferred (v0.9 KG4-5) — v0.8's full
  re-detection stays.
- The `llm-agent-rag` deployment layer (HTTP service, CLI, caching) remains
  a deliberate non-goal, deferred since v0.6.
- Regex-based content safety (`guard`, v0.6) is best-effort. **v1.2
  inherits this limitation** in the core `policy` package (KC-3) — same
  regex approach, same fundamental limit.
- `EmbeddingEntityResolver` (v0.8) has documented false-positive risk.
- The refsvc demo remains intentionally demo-grade.
- **Memory scoping (session / project / user) — deferred to v1.3 per
  KC-2.** The reframed shape (`ScopedMemory` decorator + ctx-keyed
  scope propagation via `memory.WithScope`) is pre-decided in the v1.2
  research; v1.3 implements without relitigating the design.
- **Sister-repo follow-up (post-v1.2)**: when v1.2 closes with
  `llm-agent v0.6.0`, the umbrella dep-currency gate will fail green
  against `llm-agent-otel` / `llm-agent-providers` /
  `llm-agent-customer-support` (all pin core `v0.5.1`). A future
  ecosystem-alignment milestone (v1.3-style) bumps them.

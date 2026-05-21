---
phase: 27-drift-search
plan: 01
type: execute
status: complete
completed: 2026-05-19
repo: llm-agent-rag
requirements: [RAG-GRAPH4-03]
---

> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent-rag` current docs.


# Summary: 27-01 `rag.System.AskDrift` — DRIFT hybrid search

## Objective

Add `rag.System.AskDrift` — DRIFT hybrid search: a global primer pass, a
hard-bounded local follow-up loop, and a synthesis step. A third answer path
alongside `Ask` and `AskGlobal`, orchestrating `AskGlobal`'s unexported
helpers and direct graph traversal — no `AskGlobal` refactor, no new
dependency. RAG-GRAPH4-03.

## Delivered

- `rag/options.go`:
  - `DriftOptions struct { Namespace string; MaxCommunities int; Rounds int;
    TopK int }` — the DRIFT answer-path config. Doc comments state every
    default and the hard round cap.
- `rag/system.go`:
  - `DriftDiagnostics struct { PrimerCommunityIDs []string; Rounds int;
    RoundEntityIDs [][]string; ConsultedReports []graph.CommunityReport }` —
    attributes one `AskDrift` run; `ConsultedReports` mirrors
    `GlobalDiagnostics.ConsultedReports` so `eval.DriftEvaluator` (27-02) can
    read grounding off the `Answer` without store plumbing.
  - An additive `Diagnostics.Drift DriftDiagnostics` field — the zero value
    for an ordinary `Ask`/`AskGlobal`.
- `rag/drift.go` (new), package `rag`:
  - `func (s *System) AskDrift(ctx, question string, opts DriftOptions)
    (Answer, error)` — the DRIFT answer path. It is SEPARATE: it never calls
    `s.retrieve`, the reranker, the `Ask` packer pipeline, or branches inside
    `AskGlobal`.
  - Defaults / hard cap consts: `driftDefaultMaxCommunities` (8),
    `driftDefaultRounds` (2), `driftMaxRounds` (3 — the hard round cap),
    `driftDefaultTopK` (8). `opts.Rounds` is clamped into
    `[1, driftMaxRounds]` *before* the loop runs, so the loop is bounded by
    construction.
  - `s.model == nil` -> `ErrModelRequired`. A fresh `obs.Counter` is
    installed on the context (mirrors `AskGlobal`/`Ask`), so the primer map
    generations, every local-round generation, the synthesis generation, and
    any lazy summarization are counted into `Diagnostics.Metrics`. Each of
    the three stages (`primer`, `local`, `synthesis`) is timed into
    `Metrics.Stages`.
  - **Primer** (`driftPrimer`): type-asserts `store.CommunityStore`; reuses
    `AskGlobal`'s unexported helpers in-package — `selectCommunities` ->
    `s.communityReports` -> the map step (one `globalMapPrompt` /
    `globalMapSystemPrompt` generation per report, scored via
    `parseGlobalMap`). The members of every community the map step scored
    `> 0` are the round-0 seed entity IDs (sorted, deduped); when no
    community scores above zero the loop falls back to seeding from every
    consulted community. A non-`CommunityStore` store or a namespace with no
    communities yields an empty primer, no error — the `AskGlobal`
    missing-community contract.
  - **Local follow-up loop** (`driftLocalLoop`): type-asserts
    `store.GraphStore`. For each round in `[0, Rounds)`: `gs.Neighborhood(ns,
    seeds, 1)` -> `graph.Subgraph`; the subgraph entities' provenance chunks
    (`SourceChunkIDs`) are deduped, `s.store.Get` loads each (skipping
    `store.ErrNotFound`, and tolerating any other single-chunk Get error),
    `pack`ed up to `TopK` into context; `s.model.Generate` with the question
    + context -> a partial answer and a leniently-parsed follow-up entity
    list; `gs.FindEntities(ns, names)` resolves the next round's seeds. The
    loop **terminates** on the first of: the round cap; an empty seed set; no
    new follow-up entities; no new reachable entities (a follow-up resolving
    only to already-explored entities). Per-round seed IDs are recorded into
    `RoundEntityIDs`.
  - **Synthesis** (`driftSynthesize`): one `s.model.Generate` folds the
    primer's score-`> 0` partials and every round's partial into
    `Answer.Text`. With nothing at all to synthesize it returns a graceful
    "No information was found to answer this question." and makes no model
    call.
  - Lenient parsing (`parseDriftLocal`, `cutFollowupPrefix`,
    `parseFollowupNames`): the first `Follow-up:` marker line (case-
    insensitive, fences tolerated) supplies a comma-separated follow-up list;
    the `none` sentinel yields no names; a response with no marker yields the
    whole text as the partial and no follow-ups — the loop then terminates,
    the safe default. This mirrors `parseGlobalMap`'s `Score:` discipline.
  - `sortedKeys` helper drains a set into a sorted slice — deterministic seed
    and chunk ordering is what makes the orchestration golden-testable.
- `rag/drift_test.go` (new): scripted-model golden tests against an in-memory
  store with a four-triangle graph (per-entity provenance chunks) and
  Louvain-detected communities —
  - `driftScriptedModel` keys responses by `SystemPrompt` so the primer map,
    each local round, and the synthesis are distinguishable; per-community
    map scores are keyed by report substring (`zeroScoreMarkers`) so a test
    can keep some communities out of the primer's seed set.
  - **TestAskDriftPrimerLoopSynthesis** — the happy path: primer maps the
    coarsest level, the loop runs two rounds (round 0 surfaces a follow-up
    resolving to a genuinely-new entity `b1`, round 1 emits none), synthesis
    output is `Answer.Text`; `Diagnostics.Drift` records the primer
    communities, consulted reports, `Rounds == 2`, and the sorted per-round
    seed IDs.
  - **TestAskDriftLoopTerminatesEarly** — the loop stops on the first
    no-follow-up round: budget 3, `Rounds` ends at 1.
  - **TestAskDriftRoundCapHolds** — the model emits a fresh follow-up every
    round; `Rounds == 10` is clamped and the loop never exceeds
    `driftMaxRounds` (3).
  - **TestAskDriftNonCommunityStore** — a store implementing neither
    capability degrades to the no-information answer, no error.
  - **TestAskDriftEmptyNamespaceLocalOnly** — a namespace with no communities
    degrades gracefully (empty primer, zero rounds), no error.
  - **TestAskDriftNoModel** — a nil model returns `ErrModelRequired`.
  - **TestAskDriftMalformedLocalOutput** — a local response with no
    `Follow-up:` marker yields no follow-ups; the loop terminates safely.

## Files

- `rag/drift.go` — new: `AskDrift` and its `driftPrimer` / `driftLocalLoop` /
  `driftChunkHits` / `driftPack` / `driftSynthesize` helpers, the prompt
  builders, the lenient follow-up parser, the `driftDefault*` /
  `driftMaxRounds` consts.
- `rag/drift_test.go` — new: scripted-model golden tests.
- `rag/options.go` — modified: added `DriftOptions`.
- `rag/system.go` — modified: added `DriftDiagnostics` and the additive
  `Diagnostics.Drift` field.

All four files match the plan's `files_modified` list one-to-one — no extra
file was needed (no new error sentinel: `AskDrift` reuses the existing
`ErrModelRequired` and `ErrCommunitySummarizerRequired`).

## Verification

All six `<verify>` commands run, all green:

- `cd /tmp/llm-agent-rag && GOWORK=off GOCACHE=/tmp/go-build go build ./...`
  — BUILD OK
- `cd /tmp/llm-agent-rag && GOWORK=off GOCACHE=/tmp/go-build go vet ./...`
  — VET OK
- `cd /tmp/llm-agent-rag && GOWORK=off GOCACHE=/tmp/go-build go test ./rag/...
  -count=1` — `ok github.com/costa92/llm-agent-rag/rag`
- `cd /tmp/llm-agent-rag && GOWORK=off GOCACHE=/tmp/go-build go test ./...
  -count=1` — all packages `ok`, no FAIL
- `cd /tmp/llm-agent-rag && git diff --stat go.mod go.sum` — empty (no new
  dependency)
- core facade (from the core repo `llm-agent`):
  `GOWORK=off go vet ./rag/... && go test ./rag/...` — VET OK, `ok`

## Notes / deviations

- No deviations — the plan was executed exactly as written. The
  `files_modified` list matches one-to-one; no extra file was needed.
- No new module dependency: `rag/drift.go` imports only `context`,
  `errors`, `sort`, `strconv`, `strings`, `time` from stdlib plus existing
  in-repo packages (`generate`, `graph`, `obs`, `pack`, `store`). `git diff
  --stat go.mod go.sum` is empty.
- `AskDrift` is a genuinely separate answer path: it calls neither `s.Ask`,
  `s.retrieve`, the reranker, nor `AskGlobal`. The primer reuses
  `AskGlobal`'s unexported helpers (`selectCommunities`,
  `s.communityReports`, `globalMapPrompt`, `globalMapSystemPrompt`,
  `parseGlobalMap`, `globalPartial`) directly in-package — `AskGlobal` itself
  is untouched.
- The local loop is hard-bounded by construction (keystone KG4-3):
  `opts.Rounds` is clamped to `[1, driftMaxRounds]` before the loop, and the
  loop additionally terminates early on an empty seed set, a no-follow-up
  round, or a round whose follow-ups resolve only to already-explored
  entities. A dedicated test makes the model emit a fresh follow-up every
  round and asserts `Rounds` never exceeds 3.
- Determinism is structural: round-0 seeds and every round's follow-up seeds
  are drained from a set via `sortedKeys`; provenance chunk IDs are sorted
  before `Get`; `Neighborhood`/`FindEntities`/`Communities` already return
  deterministically-ordered output. The orchestration (round count,
  termination, sorted per-round seed IDs, primer community IDs) is
  golden-testable against a scripted model — every test asserts exact
  counts and IDs.
- Graceful degradation: a non-`CommunityStore` store empties the primer; a
  non-`GraphStore` store empties the local loop; either way `AskDrift`
  returns a populated `Answer` (the no-information synthesis text) and no
  error — the `AskGlobal` missing-community contract.
- Out of scope as planned: `eval.DriftEvaluator` (27-02); the deterministic
  worked example and `docs/graphrag.md` finalization (27-03).
- Pre-existing unrelated modifications were already present in the
  `/tmp/llm-agent-rag` worktree (`docs/graphrag.md`, `retrieve/graph.go`,
  `retrieve/graph_test.go`, `rag/graph_test.go`, `graph/path.go`,
  `graph/path_test.go`, `examples/graphrag_path_example_test.go`) — these
  are not part of this slice and were left untouched.

## Self-Check: PASSED

- `rag/drift.go` and `rag/drift_test.go` present in the working tree
  (`/tmp/llm-agent-rag/rag/`); `rag/options.go` and `rag/system.go` modified
  in place.
- No commits made — per operator instruction, all changes left uncommitted
  for a separate commit.

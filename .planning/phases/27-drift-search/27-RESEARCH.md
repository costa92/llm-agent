# Phase 27 Research: DRIFT search

**Researched:** 2026-05-20
**Phase:** 27 — DRIFT search (final v0.9 phase)
**Requirements:** RAG-GRAPH4-03, RAG-GRAPH4-04
**Repos:** `llm-agent-rag`
**Upstream:** `.planning/research/v0.9-graphrag-refinements-SUMMARY.md` §4;
v0.8 `rag.System.AskGlobal`; v0.7 `retrieve.GraphRetriever`.

## Current state (codebase scan, `/tmp/llm-agent-rag` @ v0.5.0 + Phase 26)

- `rag.System.AskGlobal` (`rag/global.go`) — installs a fresh `obs.Counter`,
  type-asserts `store.CommunityStore`, then `selectCommunities` →
  `s.communityReports` → map (one `generate` per report → scored partial) →
  reduce (drop score-0, rank, synthesize). Helpers `selectCommunities`,
  `s.communityReports`, `globalMapPrompt`, `parseGlobalMap`,
  `globalReducePrompt`, `globalPartial` are all unexported in package `rag`
  — **directly reusable by `AskDrift` (same package), no refactor needed.**
- `Diagnostics.Global` — `{CommunityIDs, MapScores, MapCalls, ReduceCalls,
  ConsultedReports}`. The template for a `Diagnostics.Drift` block.
- `store.GraphStore` — `Neighborhood(ns, seedIDs, depth) (graph.Subgraph,
  error)`, `FindEntities(ns, names)`. `store.CommunityStore` — `Communities`,
  report get/put.
- `graph.Community.EntityIDs` — a community's member entity IDs.
- `retrieve.GraphRetriever` traverses from query-linked seeds; the DRIFT
  local loop traverses from *known* entity IDs — it calls `gs.Neighborhood`
  directly (the linking step does not apply).
- `eval.GlobalEvaluator` (`eval/global.go`) — global-search Triad harness;
  the template for `DriftEvaluator`.
- `pack` package — assembles chunks into prompt context (used by `Ask`).

## Decision 1 — `AskDrift` is a third answer path (KG4-2)

```go
// AskDrift answers by a global primer pass, a bounded local follow-up
// loop, and a synthesis step. A SEPARATE answer path — it orchestrates
// AskGlobal's pieces and graph traversal; it does not implement Retriever
// and is not a mode flag on Ask/AskGlobal.
func (s *System) AskDrift(ctx context.Context, question string, opts DriftOptions) (Answer, error)

type DriftOptions struct {
    Namespace      string
    MaxCommunities int // primer breadth (passed to selectCommunities); default 8
    Rounds         int // local follow-up rounds; default 2, hard cap 3
    TopK           int // local-retrieval chunk budget per round; default 8
}
```

`AskDrift` installs a fresh `obs.Counter` (like `AskGlobal`/`Ask`).
Graceful degradation: no `store.CommunityStore`, or no communities → the
primer is empty; `AskDrift` degrades to a pure local-loop answer (or an
empty `Answer`), no error — the `AskGlobal` missing-community contract.

## Decision 2 — composition: primer → local loop → synthesis

1. **Primer.** Reuse `AskGlobal`'s unexported pieces in-package:
   `selectCommunities` → `s.communityReports` → the map step. This yields
   the primer's scored partials AND the selected communities (with their
   `EntityIDs`) and reports. The highest-scoring communities' member
   entities are the local loop's **round-0 seed entity IDs**. No
   `AskGlobal` refactor — `AskDrift` calls the same helpers directly.
2. **Local follow-up loop**, bounded (KG4-3): for round `r` in
   `[0, Rounds)` (`Rounds <= 0` → 2; `> 3` → 3):
   - `gs.Neighborhood(ns, seedEntityIDs, depth=1)` → `graph.Subgraph`;
   - collect the subgraph entities' provenance chunks, `s.store.Get` them,
     `pack` into context;
   - `s.model.Generate` with the question + context → a partial answer
     **and** a short list of follow-up entity names;
   - resolve the follow-up names via `gs.FindEntities` → next round's seed
     entity IDs.
   - **Terminate** on the first of: round cap hit; the model emits no new
     follow-up entities; no new entities reachable. Bounded by construction.
3. **Synthesis.** One `s.model.Generate` call synthesizes the primer
   partials + every round's partial answer into the final `Answer.Text` —
   structurally `AskGlobal`'s reduce step.

## Decision 3 — `Diagnostics.Drift`

```go
type DriftDiagnostics struct {
    PrimerCommunityIDs []string                // communities the primer mapped
    Rounds             int                     // local rounds actually run
    RoundEntityIDs     [][]string              // seed entity IDs per round
    ConsultedReports   []graph.CommunityReport // primer reports — eval grounding context
}
// Diagnostics gains: Drift DriftDiagnostics
```

`ConsultedReports` mirrors v0.8's `Diagnostics.Global.ConsultedReports` (25-02)
— it lets `eval.DriftEvaluator` read the answer's grounding context off the
`Answer` without store plumbing.

## Decision 4 — `eval.DriftEvaluator` (KG4-1 / §6 of the milestone research)

DRIFT synthesizes an answer with no gold chunk set — exactly like global
search. `eval/global.go`'s `GlobalEvaluator` is the template:

```go
package eval
type DriftAsker interface {
    AskDrift(ctx context.Context, question string, opts rag.DriftOptions) (rag.Answer, error)
}
// DriftEvaluator{Asker, Judge, MaxCommunities, Rounds} — runs whole-corpus
// questions through AskDrift, scores each answer with the Judge
// (groundedness vs ConsultedReports + answer-relevance). No chunk recall@k.
```

## Slice breakdown

- **27-01** — `rag.System.AskDrift` + `DriftOptions` + `DriftDiagnostics`:
  primer (reusing the global helpers) → bounded local follow-up loop →
  synthesis; scripted-model golden tests asserting round count, termination,
  and `Diagnostics.Drift`. (RAG-GRAPH4-03)
- **27-02** — `eval.DriftEvaluator` — a Triad/`LLMJudge` harness for DRIFT
  answers, mirroring `GlobalEvaluator`; a scripted-model CI gate.
  (RAG-GRAPH4-04)
- **27-03** — deterministic DRIFT worked example; `docs/graphrag.md`
  finalized — DRIFT usage, the primer/local budget, the round cap, and the
  v1.0+ deferral list (incremental community maintenance with its trigger,
  claim extraction, graph DB). (RAG-GRAPH4-04)

## Risks / notes

- The LLM round budget is bounded: primer (`MaxCommunities` map + 1 reduce-
  free since synthesis is shared) + `Rounds` local generations + 1
  synthesis. Surfaced in `Diagnostics.Drift` + counted by `obs.Counter`.
- The local loop's follow-up parsing is lenient (a marked line of entity
  names), scripted-model tested incl. malformed output — the v0.7
  extractor-parse precedent.
- Determinism: the orchestration (round count, termination, seed
  propagation, sorted entity IDs) is golden-testable against a scripted
  model; the LLM steps themselves are scripted.
- 27-02 depends on 27-01; 27-03 depends on 27-01+02.
- No new module dependency — `AskDrift` is orchestration over existing
  seams + `generate.Model`.

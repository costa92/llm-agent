---
phase: 27-drift-search
plan: 03
type: execute
status: complete
completed: 2026-05-19
repo: llm-agent-rag
requirements: [RAG-GRAPH4-04]
---

> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent-rag` current docs.


# Summary: 27-03 DRIFT search — worked example + docs finalized

## Objective

Ship a deterministic DRIFT worked example and finalize `docs/graphrag.md`
for v0.9 — DRIFT usage, the primer/local budget, the round cap, the
`eval.DriftEvaluator` harness, and the v1.0+ deferral list. The DRIFT
feature itself shipped in 27-01 (`rag.System.AskDrift` + `DriftOptions` +
`DriftDiagnostics`) and 27-02 (`eval.DriftEvaluator`); this slice makes it
demonstrable end-to-end and reader-discoverable. Completes RAG-GRAPH4-04 and
Phase 27 / the v0.9 GraphRAG-refinements milestone.

## Delivered

### 1. `examples/graphrag_drift_example_test.go` — `Example_graphRAGDrift`

A new, fully deterministic worked example (≈155 lines), modelled on the v0.8
`graphrag_global_example_test.go` worked-example template —
`DictionaryEntityExtractor` gazetteer, an in-memory store, a
`LouvainDetector`, a single scripted `generate.Model` routed by
`SystemPrompt` prefix, a stable `// Output:`, no live model:

- A `driftExampleModel` scripted `generate.Model` serving **four** request
  kinds — the community summarizer, the primer map step, every local
  follow-up round, and the synthesis — told apart by `SystemPrompt` prefix
  exactly as the global example does. Two kinds additionally sub-route on
  request content so the example exercises a real two-round local loop:
  - the **summarizer** gives the two communities distinct titles (Babbage
    cluster vs. Turing cluster);
  - the **map step** scores only the Babbage community above zero, so just
    its member entities seed the local loop's round 0;
  - the **local step** emits `Follow-up: Alan Turing` while its context has
    not yet seen Turing, then `Follow-up: none` once it has — so the loop
    runs exactly two rounds and terminates by itself.
- The same fixed four-document corpus as the global example: two loosely
  connected clusters of early-computing history — the shape DRIFT is built
  for.
- `Example_graphRAGDrift` wires the system, calls `sys.AskDrift`, and prints
  the primer-community count, the local rounds run, and the answer — all
  read off `ans.Diagnostics.Drift`.

The `// Output:` block was **not guessed** — the example was run, the real
deterministic output captured, and the block confirmed to match it:

```
primer communities mapped: 2
local rounds run: 2
answer: DRIFT traced the corpus from Babbage's Analytical Engine through to Turing's wartime codebreaking machines.
```

The two-round behaviour was verified against the shipped 27-01 code: the
primer maps both coarsest-level communities (`PrimerCommunityIDs` has 2),
only the Babbage community scores `> 0` so round-0 seeds are its three
entities, round 0's `Follow-up: Alan Turing` resolves via `FindEntities` to
a not-yet-seen entity (`person:alan turing`), round 1 traverses from Turing
and emits `Follow-up: none`, and the loop terminates — `Drift.Rounds == 2`.

### 2. `docs/graphrag.md` — DRIFT search section

A new **"DRIFT search — the hybrid answer path (`System.AskDrift`)"**
section added under Tier-3, after the global-search-evaluation material.
Every type/method/field name was cross-checked against the shipped 27-01/02
code in `/tmp/llm-agent-rag` (`rag/drift.go`, `rag/options.go`,
`eval/drift.go`):

- DRIFT positioned as the **hybrid** between `AskGlobal` (pure global) and
  `Ask` + `GraphRetriever` (local) — a third, separate answer path, not a
  mode flag and not a `Retriever`.
- The *primer → bounded local follow-up loop → synthesis* flow spelled out
  exactly as implemented, including the graceful-degradation contract (no
  `CommunityStore` / no communities → empty primer → local-only answer).
- `DriftOptions` documented field-by-field — `Namespace`, `MaxCommunities`
  (default 8), `Rounds` (**default 2, hard cap 3**, clamped to `[1, 3]`
  before the loop runs), `TopK` (default 8) — with the LLM-budget framing.
- `Diagnostics.Drift` (`rag.DriftDiagnostics`) — `PrimerCommunityIDs`,
  `Rounds`, `RoundEntityIDs`, `ConsultedReports`.
- A short wiring code snippet and a pointer to the new
  `examples/graphrag_drift_example_test.go`.
- `eval.DriftEvaluator` — the `DriftAsker` seam, the
  `DriftEvaluator{Asker, Judge, MaxCommunities, Rounds}` struct, the
  `ConsultedReports` grounding context, and `DriftEvalResult` (no chunk
  recall@k, mirroring `GlobalEvalResult`).
- The intro paragraph updated: v0.9 now described as finishing the picture
  with path-ranked evidence and DRIFT search.

### 3. `docs/graphrag.md` — finalized "Deferred to v1.0+" section

The stale **"Deferred to v0.9"** section retitled **"Deferred to v1.0+"**
and rewritten. It now records v0.9 as closing the GraphRAG-refinements
milestone (path-ranking and DRIFT both shipped — all three answer paths
exist) and finalizes the v1.0+ list:

- **Incremental community maintenance** — with its explicit **profiling
  trigger condition**: revisit only if `CommunityDetector.Detect`
  measurably dominates re-ingest cost on a real corpus.
- **Claim / covariate extraction** — a future seam alongside
  `EntityExtractor`.
- **A dedicated graph database** — keeping the **"Neo4j is a future
  `GraphStore` implementation"** note.
- **Fuzzy-resolution quality improvements** — carried forward from v0.8.

## Files

- `examples/graphrag_drift_example_test.go` — created; the deterministic
  `Example_graphRAGDrift` worked example.
- `docs/graphrag.md` — modified; new DRIFT-search section, finalized
  "Deferred to v1.0+" section, intro paragraph updated.

Both files match the plan's `files_modified` list exactly. No code change
beyond the example test — the feature shipped in 27-01/02.

## Verification

Every command in the plan's `<verify>` block was run; all green:

- `GOWORK=off GOCACHE=/tmp/go-build go build ./...` — BUILD OK
- `GOWORK=off GOCACHE=/tmp/go-build go vet ./...` — VET OK
- `GOWORK=off GOCACHE=/tmp/go-build go test ./examples/... -count=1` — `ok`
  (`Example_graphRAGDrift` PASS with the stable `// Output:`)
- `GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1` — all 21
  packages `ok`, no FAIL
- `git diff --stat go.mod go.sum` — empty (no new module dependency)
- core facade (from the core repo `llm-agent`): `GOWORK=off go vet
  ./rag/... && go test ./rag/...` — VET OK, `ok
  github.com/costa92/llm-agent/rag`

## Deviations from plan

Plan executed essentially as written. Two notes:

1. **`// Output:` confirmed against real output.** The example was run and
   the deterministic output captured before finalizing the `// Output:`
   block — it matched the drafted block on the first run (the orchestration
   is golden-testable by construction), but the block was verified, not
   assumed, per the plan's instruction.

2. **One extra paragraph touched in `docs/graphrag.md` beyond the new
   sections.** The intro paragraph still framed the doc as v0.7/v0.8 only; a
   v0.9 sentence was added so the doc opens consistently with its new DRIFT
   content. This is within task 2's scope and the same file already in
   `files_modified`; no new file was touched.

No git write command was run — all changes are left uncommitted.

## Notes

- Every type, method, and field name in the example and the doc was
  cross-checked against the shipped 27-01/02 code: `rag.System.AskDrift`,
  `rag.DriftOptions` (`Namespace`, `MaxCommunities`, `Rounds`, `TopK`),
  `rag.DriftDiagnostics` (`PrimerCommunityIDs`, `Rounds`, `RoundEntityIDs`,
  `ConsultedReports`), `rag.Diagnostics.Drift`, `eval.DriftAsker`,
  `eval.DriftEvaluator` (`Asker`, `Judge`, `MaxCommunities`, `Rounds`),
  `eval.DriftEvalResult`, `graph.DictionaryEntityExtractor`,
  `graph.LouvainDetector`, `graph.LLMCommunitySummarizer` — no API name is a
  guess. The default/cap constants (`driftDefaultMaxCommunities = 8`,
  `driftDefaultRounds = 2`, `driftMaxRounds = 3`, `driftDefaultTopK = 8`)
  were read straight from `rag/drift.go`.
- The example is fully deterministic with no live model: a
  `DictionaryEntityExtractor` gazetteer, an in-memory store, a
  `LouvainDetector` (deterministic by construction), and a scripted
  `generate.Model` routed by `SystemPrompt` prefix + content.
- Docs + example only; no code change, no new dependency. `go.mod`/`go.sum`
  diff is empty.

## Self-Check: PASSED

- `examples/graphrag_drift_example_test.go` — FOUND (created,
  `Example_graphRAGDrift` PASS with a stable `// Output:`)
- `docs/graphrag.md` — FOUND (modified: DRIFT-search section added,
  "Deferred to v1.0+" section finalized, intro updated)
- All `<verify>` commands green; `go.mod`/`go.sum` diff empty.

## Phase 27 status

All three slices complete:

- **27-01** — `rag.System.AskDrift` + `DriftOptions` + `DriftDiagnostics`:
  the DRIFT answer path — a global primer pass (reusing `AskGlobal`'s
  unexported helpers in-package), a hard-bounded local follow-up loop
  (1-hop neighborhood traversal, lenient follow-up parsing, terminating on
  the round cap / no new follow-ups / no new reachable entities), and a
  synthesis step; scripted-model golden tests for round count, termination,
  and `Diagnostics.Drift`. (RAG-GRAPH4-03)
- **27-02** — `eval.DriftEvaluator` — a RAG-Triad / `LLMJudge` harness for
  DRIFT answers, mirroring `GlobalEvaluator` (groundedness vs.
  `ConsultedReports` + answer relevance, no chunk recall@k); a
  scripted-model + scripted-judge CI gate. (RAG-GRAPH4-04)
- **27-03** — the deterministic `Example_graphRAGDrift` worked example and
  the finalized `docs/graphrag.md` — DRIFT usage, the primer/local budget,
  the round cap, `eval.DriftEvaluator`, and the v1.0+ deferral list.
  (RAG-GRAPH4-04)

**RAG-GRAPH4-03 and RAG-GRAPH4-04 are delivered.** Phase 27 — DRIFT search,
the final v0.9 GraphRAG-refinements phase — is complete: the hybrid answer
path is shipped in `rag`, evaluable via `eval.DriftEvaluator`, demonstrated
by a deterministic worked example, and documented.

## v0.9 milestone status

The **v0.9 GraphRAG-refinements milestone is code-complete** — Phases 26 and
27, requirements RAG-GRAPH4-01..04:

- **Phase 26 — path-ranked evidence** — `RAG-GRAPH4-01` (the `graph`
  `RankedPath` / `PathRanker` / `WeightedPathRanker` seam) and
  `RAG-GRAPH4-02` (the opt-in `GraphRetriever.PathRanker` mode +
  `GraphTrace.Paths` / `EvidenceSubgraph`) — delivered, demonstrated, and
  documented.
- **Phase 27 — DRIFT search** — `RAG-GRAPH4-03` (`rag.System.AskDrift` +
  `DriftOptions` + `DriftDiagnostics`) and `RAG-GRAPH4-04`
  (`eval.DriftEvaluator` + the worked example + finalized docs) —
  delivered, demonstrated, and documented.

With path-ranked evidence and DRIFT search shipped, all three answer paths
exist — `Ask` (local), `AskGlobal` (global), and `AskDrift` (the hybrid) —
and `docs/graphrag.md` documents the complete GraphRAG surface with a
finalized v1.0+ deferral list. The v0.9 GraphRAG-refinements milestone is
done.

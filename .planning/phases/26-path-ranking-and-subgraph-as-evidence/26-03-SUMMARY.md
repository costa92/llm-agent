---
phase: 26-path-ranking-and-subgraph-as-evidence
plan: 03
type: execute
status: complete
completed: 2026-05-19
repo: llm-agent-rag
requirements: [RAG-GRAPH4-02]
---

> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent-rag` current docs.


# Summary: 26-03 path-ranked subgraph-as-evidence — worked example + docs

## Objective

Ship a deterministic worked example for path-ranked subgraph evidence and
document it in `docs/graphrag.md`. The path-ranking feature itself shipped in
26-01 (the `graph` seam) and 26-02 (the `GraphRetriever` wiring); this slice
makes it demonstrable end-to-end and reader-discoverable. Completes
RAG-GRAPH4-02 and Phase 26 / opens the v0.9 milestone work.

## Delivered

### 1. `examples/graphrag_path_example_test.go` — `Example_graphRAGPaths`

A new, fully deterministic worked example (78 lines), modelled on the v0.7
`graphrag_example_test.go` and v0.8 `graphrag_global_example_test.go`
worked-example templates — `DictionaryEntityExtractor` gazetteer, an
in-memory store, the shared `echoModel`, a stable `// Output:`, no live
model:

- A `rag.System` wired with a **bare `retrieve.GraphRetriever`** carrying a
  `graph.WeightedPathRanker{}` — path mode on (`PathRanker` non-nil).
- A small fixed corpus whose entities form a known two-hop chain:
  `Lovelace — co-occurs — Babbage — co-occurs — Engine`. Single-word entity
  names are used deliberately so the default `LexicalEntityLinker` resolves
  both chain ends from the query's whitespace tokens (multi-word names like
  "Ada Lovelace" would not match a single token).
- The query `"Lovelace Engine"` links the two chain ends; `Retrieve` ranks
  the simple path connecting them through `Babbage` and surfaces it.
- The example prints the top ranked path's entity sequence, its edge count,
  and the evidence-subgraph size — all read off
  `ans.Diagnostics.GraphTrace`.

The `// Output:` block was **not guessed** — the example was run, the real
deterministic output captured, and the block corrected to match it:

```
top ranked path: machine:engine -> person:babbage -> person:lovelace
path edges: 2
evidence subgraph entities: 3
evidence subgraph relations: 2
```

The path reads `engine -> babbage -> lovelace` (not `lovelace -> ... ->
engine`) because `GraphRetriever.seedPairs` builds the seed pair from the
sorted seed-ID list — `[2]string{machine:engine, person:lovelace}` — and
`WeightedPathRanker`'s bounded DFS enumerates the path from `pair[0]`. This
is deterministic and reproducible; the `// Output:` reflects the real shipped
behavior.

### 2. `docs/graphrag.md` — path-ranked evidence subsection

A new **"Path-ranked evidence (`GraphRetriever.PathRanker`)"** subsection
added under the Tier-1 `GraphRetriever` material (section 3), `+62` lines.
Every type/method/field name was cross-checked against the shipped 26-01/02
code in `/tmp/llm-agent-rag` (`graph/path.go`, `retrieve/graph.go`):

- The motivation — default `GraphRetriever` scores provenance *chunks* and
  says nothing about *how* entities connect; path mode surfaces the
  connecting structure as a ranked evidence artifact.
- How to enable it — set `GraphRetriever.PathRanker` to a
  `graph.PathRanker`; `graph.WeightedPathRanker{LengthDecay}` is the default
  deterministic, pure-stdlib ranker.
- The `WeightedPathRanker` scoring composite spelled out exactly as
  implemented: bounded DFS (≤ 2 edges, relations undirected), length decay
  (`LengthDecay^(edges-1)`; `≤ 0` → `0.5`), product of edge weights, and a
  provenance-overlap bonus per consecutive relation pair sharing a
  `SourceChunkID` — sorted by `Score` desc, ties broken by joined
  entity-ID sequence (keystones KG4-4, KG4-6).
- What the two new `GraphTrace` fields carry — `Paths []graph.RankedPath`
  (`EntityIDs`, `RelationIDs`, `Score`) and `EvidenceSubgraph
  *graph.Subgraph` — and that both ride through
  `Answer.Diagnostics.GraphTrace` for free, the same way
  `GraphTrace.CommunityIDs` does.
- The **opt-in / additive contract**: with `PathRanker` nil (the default)
  `Paths`/`EvidenceSubgraph` stay nil and graph retrieval is byte-identical
  to v0.7/v0.8 — chunk hits and scores untouched; path mode only *adds*
  trace output.
- A short code snippet and a pointer to the new
  `examples/graphrag_path_example_test.go`.
- The stale **"Deferred to v0.9"** bullet for path-ranking updated: it was a
  v0.8 deferral; it is now noted as **shipped in v0.9** via the
  `GraphRetriever.PathRanker` mode documented above.

## Files

- `examples/graphrag_path_example_test.go` — created; the deterministic
  `Example_graphRAGPaths` worked example.
- `docs/graphrag.md` — modified; new path-ranked-evidence subsection +
  v0.9-deferral bullet updated.

Both files match the plan's `files_modified` list exactly. No code change
beyond the example test — the feature shipped in 26-01/02.

## Verification

Every command in the plan's `<verify>` block was run; all green:

- `GOWORK=off GOCACHE=/tmp/go-build go build ./...` — BUILD OK
- `GOWORK=off GOCACHE=/tmp/go-build go vet ./...` — VET OK
- `GOWORK=off GOCACHE=/tmp/go-build go test ./examples/... -count=1` — `ok`
  (`Example_graphRAGPaths` PASS with the stable `// Output:`)
- `GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1` — all 21
  packages `ok`, no FAIL
- `git diff --stat go.mod go.sum` — empty (no new module dependency)
- core facade (from the core repo `llm-agent`): `GOWORK=off go vet
  ./rag/... && go test ./rag/...` — VET OK, `ok
  github.com/costa92/llm-agent/rag`

## Deviations from plan

Plan executed essentially as written. Two notes:

1. **`// Output:` corrected to real output.** The initial draft assumed the
   path would read `person:lovelace -> ... -> machine:engine`. The example
   was run, the real deterministic output (`machine:engine ->
   person:babbage -> person:lovelace`) captured, and the `// Output:` block
   corrected to match — per the plan's instruction to inspect real output
   rather than fake it. Path direction follows the sorted seed pair, as
   explained above.

2. **One extra line touched in `docs/graphrag.md` beyond the new
   subsection.** The "Deferred to v0.9" section still listed path-ranking as
   an unshipped v0.9 item — now contradicted by this slice. The stale bullet
   was updated to say the feature shipped, keeping the doc internally
   consistent. This is within task 2's scope ("document path-ranked
   evidence") and is the same file already in `files_modified`; no new file
   was touched.

No git write command was run — all changes are left uncommitted.

## Notes

- Every type, method, and field name in the example and the doc was
  cross-checked against the shipped 26-01/02 code: `graph.RankedPath`
  (`EntityIDs`, `RelationIDs`, `Score`), `graph.PathRanker`,
  `graph.WeightedPathRanker{LengthDecay}`, `graph.Subgraph`,
  `retrieve.GraphRetriever.PathRanker`, `retrieve.GraphTrace.Paths` /
  `.EvidenceSubgraph`, `rag.Diagnostics.GraphTrace` — no API name is a
  guess.
- The example is fully deterministic with no live model: `echoModel` (the
  shared examples-package scripted model), a `DictionaryEntityExtractor`
  gazetteer, an in-memory store, and `WeightedPathRanker`'s
  deterministic-by-construction ranking.
- Docs + example only; no code change, no new dependency. `go.mod`/`go.sum`
  diff is empty.

## Self-Check: PASSED

- `examples/graphrag_path_example_test.go` — FOUND (created, 78 lines,
  `Example_graphRAGPaths` PASS with a stable `// Output:`)
- `docs/graphrag.md` — FOUND (modified, path-ranked-evidence subsection
  added)
- All `<verify>` commands green; `go.mod`/`go.sum` diff empty.

## Phase 26 status

All three slices complete:

- **26-01** — `graph` package: the `RankedPath` type, the `PathRanker`
  seam, and `WeightedPathRanker` — bounded-DFS simple-path enumeration over
  a `Subgraph`, a composite deterministic score (length decay × edge-weight
  product × provenance-overlap bonus), a total tie-break by joined
  entity-ID sequence; golden-output unit tests. (RAG-GRAPH4-01)
- **26-02** — `retrieve.GraphRetriever` gained the opt-in `PathRanker`
  field; `GraphTrace` gained the additive `Paths` + `EvidenceSubgraph`
  fields; `Retrieve` stays byte-identical to v0.7/v0.8 when `PathRanker` is
  nil; tests cover both modes and the `rag.Diagnostics` surfacing.
  (RAG-GRAPH4-02)
- **26-03** — the deterministic `Example_graphRAGPaths` worked example and
  the `docs/graphrag.md` path-ranked-evidence section. (RAG-GRAPH4-02)

**RAG-GRAPH4-01 and RAG-GRAPH4-02 are delivered.** Phase 26 — path-ranking
and subgraph-as-evidence, the first v0.9 GraphRAG-refinements phase — is
complete: the feature is shipped in `graph`, wired into `GraphRetriever`,
demonstrated by a deterministic worked example, and documented. The next
v0.9 item — DRIFT search (query-adaptive blend of local traversal and
global community context) — is Phase 27, explicitly out of scope here.

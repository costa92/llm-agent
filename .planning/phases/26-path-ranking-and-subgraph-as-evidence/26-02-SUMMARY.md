---
phase: 26-path-ranking-and-subgraph-as-evidence
plan: 02
type: execute
status: complete
completed: 2026-05-19
repo: llm-agent-rag
requirements: [RAG-GRAPH4-02]
---

# Summary: 26-02 path mode on GraphRetriever — ranked paths + evidence subgraph on the trace

## Objective

Wire path ranking into `retrieve.GraphRetriever` as an opt-in mode and surface
the ranked paths plus the structured evidence subgraph on `GraphTrace` /
`rag.Diagnostics`. With path mode off, graph retrieval stays byte-identical to
v0.7/v0.8 (KG4-4). Completes RAG-GRAPH4-02.

## Delivered

- `retrieve.GraphRetriever.PathRanker graph.PathRanker` — a new opt-in field.
  Nil (the default) = path mode off; non-nil = path mode on. Documented like
  the v0.8 `CommunityIDs` precedent: with the field nil, `Retrieve` is
  byte-identical to v0.7/v0.8 — chunk hits, scoring, RRF behavior, and every
  other trace field untouched.
- `retrieve.GraphTrace` gains two additive fields, mirroring how v0.8 added
  `CommunityIDs`:
  - `Paths []graph.RankedPath` — the ranked simple paths connecting the
    query's seed entities, in deterministic descending-score order. Populated
    only in path mode; nil otherwise.
  - `EvidenceSubgraph *graph.Subgraph` — the bounded neighborhood the
    retriever traversed, surfaced as the structured evidence object.
    Populated only in path mode; nil otherwise.
- `retrieve/graph.go` `Retrieve` — after `sub` is built and **after** the
  existing chunk-hit / `CommunityIDs` logic (untouched), a new gated block:
  when `r.PathRanker != nil`, it builds the seed pairs via the new
  `seedPairs` helper, calls `r.PathRanker.RankPaths(sub, pairs)`, sets
  `trace.Graph.Paths`, and sets `trace.Graph.EvidenceSubgraph` to a pointer
  to a copy of `sub`. When `PathRanker` is nil the block is skipped entirely
  — hits, scores, and the rest of the trace are produced exactly as before.
- `seedPairs(seedIDs []string) [][2]string` — a new unexported helper in
  `retrieve/graph.go`: builds every unordered pair of distinct seed entity
  IDs. `seedIDs` is already sorted and deduped at the call site (the existing
  `sort.Strings(seedIDs)` in `Retrieve`), so iterating `i<j` yields pairs in
  a total, reproducible order — the determinism `WeightedPathRanker` expects.
  Fewer than two seeds yields no pairs (no path to rank).
- `retrieve/graph_test.go` — `TestGraphRetrieverPathMode`: runs the same
  query ("Alpha Charlie", linking to the two endpoints of the
  Alpha-Bravo-Charlie chain fixture) with path mode off and path mode on
  (`graph.WeightedPathRanker{}`). Asserts: path mode off leaves
  `Paths`/`EvidenceSubgraph` nil; path mode on populates `Paths` (>= 1 path),
  a non-nil `EvidenceSubgraph` with the 3 traversed entities, and a top
  ranked path connecting `t:alpha .. t:charlie`; and — the KG4-4 guard — the
  chunk `[]store.Hit` and the v0.7/v0.8 trace fields (`MaxHop`,
  `SeedEntityIDs`, `ReachedEntityIDs`) are identical between the two runs.
- `rag/graph_test.go` — `TestAskSurfacesGraphPaths`: builds an in-memory
  store with the Alpha-Bravo-Charlie chain via `UpsertGraph`, wires a
  path-mode `retrieve.GraphRetriever` (`PathRanker: graph.WeightedPathRanker{}`)
  as `Options.Retriever`, and calls `System.Ask`. Asserts the new
  `GraphTrace` fields surface through `Answer.Diagnostics.GraphTrace` —
  `Paths` is populated, `EvidenceSubgraph` is non-nil with 3 entities, and
  the top ranked path connects `t:alpha .. t:charlie`.

## Files

- `retrieve/graph.go` — modified; `GraphRetriever.PathRanker` field,
  `GraphTrace.Paths` + `EvidenceSubgraph` fields, the gated path-ranking step
  in `Retrieve`, the `seedPairs` helper.
- `retrieve/graph_test.go` — modified; `TestGraphRetrieverPathMode` added.
- `rag/graph_test.go` — modified; `TestAskSurfacesGraphPaths` added,
  `embed` and `retrieve` imports added.

Exactly the plan's `files_modified` list — no extra files.

## Verification

All `<verify>` commands run, all green:

- `GOWORK=off GOCACHE=/tmp/go-build go build ./...` — BUILD OK
- `GOWORK=off GOCACHE=/tmp/go-build go vet ./...` — VET OK
- `GOWORK=off GOCACHE=/tmp/go-build go test ./retrieve/... ./rag/... -count=1`
  — `ok github.com/costa92/llm-agent-rag/retrieve`,
  `ok github.com/costa92/llm-agent-rag/rag`
- `GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1` — all 21 packages
  `ok`, no FAIL
- `git diff --stat go.mod go.sum` — empty (no new module dependency)
- core facade (from the core repo `llm-agent`): `GOWORK=off go vet ./rag/...
  && go test ./rag/...` — `ok github.com/costa92/llm-agent/rag`

## Deviations from plan

None. The `files_modified` list is matched exactly. The plan's task 1
described the path-ranking step in `Retrieve` as building "every unordered
pair of linked seed entity IDs, in sorted order" — implemented as the
unexported `seedPairs` helper in the same file rather than inline, for
readability and so the determinism contract (sorted input, total pair order)
is documented in one place. This is the same file the plan lists, not an
extra file.

## Notes

- `EvidenceSubgraph` is set to `&evidence` where `evidence := sub` — a copy
  of the local `sub` value, not `&sub` aliasing a loop/return variable. The
  `graph.Subgraph` value type means the trace holds a stable snapshot of the
  neighborhood the retriever traversed.
- Path mode is purely additive trace output. The new block sits strictly
  after the existing chunk-hit, hit-sort, `TopK`-trim, and `CommunityIDs`
  logic, and writes only `trace.Graph.Paths` and `trace.Graph.EvidenceSubgraph`
  — so with `PathRanker` nil the function is byte-identical to v0.7/v0.8.
  `TestGraphRetrieverPathMode` proves this directly: it asserts the chunk
  `[]store.Hit` is identical between a path-mode-on and a path-mode-off run.
- The new `GraphTrace` fields reach `rag.Diagnostics` for free:
  `rag/ask.go` already sets `Diagnostics.GraphTrace = retrieveTrace.Graph`,
  and `VariantRetriever` propagates the whole `Trace.Graph` block (the v0.7
  22-03 fix). No `retrieve.go` or `rag/ask.go` change was needed —
  `TestAskSurfacesGraphPaths` confirms the fields arrive on `Answer`.
- Determinism end to end: `seedIDs` is sorted by the existing
  `sort.Strings(seedIDs)`, `seedPairs` iterates `i<j` for a total pair
  order, and `graph.WeightedPathRanker.RankPaths` is itself deterministic
  (bounded DFS in sorted neighbor order, total score tie-break). Same store
  + same query => same `Paths`, byte-for-byte.
- No new module dependency — `retrieve/graph.go` already imports the `graph`
  package; `RankedPath` / `PathRanker` / `WeightedPathRanker` (from 26-01)
  are types within it. The test additions import only already-present module
  packages (`embed`, `graph`, `retrieve`, `store`) and stdlib.
- Out of scope, per plan: the worked example and `docs/graphrag.md`
  path-evidence section (26-03); any change to chunk scoring or RRF fusion
  (path ranking is extra trace output only).

## Self-Check: PASSED

- `retrieve/graph.go` — FOUND (modified: `PathRanker` field, `Paths` +
  `EvidenceSubgraph` fields, path-ranking step, `seedPairs` helper)
- `retrieve/graph_test.go` — FOUND (modified: `TestGraphRetrieverPathMode`)
- `rag/graph_test.go` — FOUND (modified: `TestAskSurfacesGraphPaths`)
- All `<verify>` commands green; `go.mod`/`go.sum` diff empty.

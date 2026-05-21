---
phase: 25-fuzzy-entity-resolution-and-evaluation
plan: 01
type: execute
status: complete
completed: 2026-05-19
repo: llm-agent-rag
requirements: [RAG-GRAPH3-05]
---

> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent-rag` current docs.


# Summary: 25-01 Fuzzy entity resolution

## Objective

Add embedding-similarity fuzzy entity resolution: an `EntityResolver` seam,
a `NoopEntityResolver` default, an `EmbeddingEntityResolver`, wired as an
opt-in pre-pass before `graph.Canonicalize` in `rag.Import`. RAG-GRAPH3-05.

## Delivered

- `graph/resolve.go` (new):
  - `EntityResolver` seam — `Resolve(ctx, entities []Entity, relations
    []Relation) ([]Entity, []Relation, error)`. The doc comment states the
    contract that a resolver MUST rewrite relation endpoints to match the
    entity names it rewrites, because `Canonicalize` resolves endpoints by
    name and drops a relation whose endpoint matches no entity (Decision 1).
  - `NoopEntityResolver struct{}` — returns its inputs unchanged; the
    default, making `Import` byte-identical to pre-fuzzy-resolution behavior.
  - `EmbeddingEntityResolver{Embedder embed.Embedder; Threshold float64}`:
    - `Threshold <= 0` selects `defaultResolverThreshold` (0.92), a high
      conservative default — a near miss stays two nodes.
    - Embeds each distinct entity `Name` once via the `embed.Embedder`,
      clusters entities of the **same `Type`** whose cosine similarity is
      `>= Threshold`. Clustering is single-link union-find over a fixed
      sorted-by-`Name` pair scan — fully deterministic, no randomness
      (keystone KG3-6). Same-`Name` same-`Type` entities always union.
    - Per cluster picks the canonical `Name` deterministically: the longest
      member name, ties broken lexically lowest (`betterCanonical`). Every
      member entity `Name` and every relation `Source`/`Target` that named
      a member is rewritten to the canonical form. Inputs are not mutated —
      fresh slices are returned.
    - A nil `Embedder` returns the sentinel `ErrEntityResolverEmbedderRequired`;
      an embedder error is propagated; empty input returns the inputs.
  - `ErrEntityResolverEmbedderRequired` sentinel — sibling of
    `graph.ErrEntityExtractorModelRequired`.
- `graph/graph.go`: package doc comment updated — `graph` is now a
  "near-leaf" package that imports stdlib, the `generate` seam, AND the
  `embed` seam; both seams are stdlib-only leaf packages so no third-party
  dependency is added.
- `graph/resolve_test.go` (new): `EmbeddingEntityResolver` against a
  **scripted embedder** returning fixed vectors (never a live call) —
  "Acme"/"Acme Corp" of the same type collapse to one canonical name with a
  relation endpoint following suit; different-`Type` entities do NOT merge
  even at similarity 1.0; unrelated entities stay distinct; the
  longest-name canonical pick and the equal-length lexical tie-break are
  asserted; the threshold is respected (no merge below the default,
  merge with a low explicit threshold); a re-run yields identical output
  (determinism); `NoopEntityResolver` is identity; nil-embedder and
  embedder-error cases; empty input.
- `rag/options.go`: `Options.EntityResolver graph.EntityResolver` added with
  a doc comment naming the `NoopEntityResolver{}` default.
- `rag/system.go`: `System.entityResolver` field carried; `New` defaults it
  to `graph.NoopEntityResolver{}` when `Options.EntityResolver` is nil.
- `rag/import.go`: before `graph.Canonicalize`, `Import` now runs
  `s.entityResolver.Resolve(ctx, graphEnts, graphRels)`, wrapping an error
  as `rag: resolve entities: %w`. With the `Noop` default the graph is
  byte-identical to before.
- `rag/resolve_test.go` (new): an `Import` configured with an
  `EmbeddingEntityResolver` and a scripted embedder collapses the
  near-duplicate "Acme"/"Acme Corp" org entities into a single canonical
  node (`org:acme corp`, the longest surface form) with unioned provenance,
  and the `co-occurs` relation to Globex survives (its endpoint followed the
  rewrite, so `Canonicalize` does not orphan it). The default (no resolver)
  import is asserted unchanged — the two Acme surface forms stay distinct
  nodes.

## Files

- `graph/resolve.go` — new: `EntityResolver`, `NoopEntityResolver`,
  `EmbeddingEntityResolver`, `ErrEntityResolverEmbedderRequired`.
- `graph/resolve_test.go` — new: scripted-embedder resolver tests.
- `graph/graph.go` — package doc comment updated for the `embed` seam.
- `rag/options.go` — `Options.EntityResolver` field.
- `rag/system.go` — `System.entityResolver` field + `NoopEntityResolver`
  default in `New`.
- `rag/import.go` — resolver pre-pass before `Canonicalize`.
- `rag/resolve_test.go` — new: `Import`-with-resolver integration tests.

All seven files match the plan's `files_modified` list one-to-one — no
extra file was needed (the `ErrEntityResolverEmbedderRequired` sentinel
lives in `graph/resolve.go` next to its sole user, mirroring how
`ErrEntityExtractorModelRequired` lives in `graph/extract.go`).

## Verification

All `<verify>` commands run, all green:

- `cd /tmp/llm-agent-rag && GOWORK=off GOCACHE=/tmp/go-build go build ./...`
  — BUILD OK
- `cd /tmp/llm-agent-rag && GOWORK=off GOCACHE=/tmp/go-build go vet ./...`
  — VET OK
- `cd /tmp/llm-agent-rag && GOWORK=off GOCACHE=/tmp/go-build go test ./graph
  ./rag/... -count=1` — `ok` for both `graph` and `rag`
- `cd /tmp/llm-agent-rag && GOWORK=off GOCACHE=/tmp/go-build go test ./...
  -count=1` — all packages `ok`, no FAIL
- `cd /tmp/llm-agent-rag && git diff --stat go.mod go.sum` — empty (no new
  dependency)
- core facade (from the core repo `llm-agent`):
  `GOWORK=off go vet ./rag/... && go test ./rag/...` — VET OK, `ok`

## Notes / deviations

- No deviations — the plan was executed exactly as written. The
  `files_modified` list matches one-to-one.
- `graph` importing `embed` adds no module dependency: `embed` is already
  in the module and is itself a stdlib-only leaf package, so there is no
  import cycle and no new third-party dep. `git diff --stat go.mod go.sum`
  is empty.
- Determinism (KG3-6) is structural, not incidental: entities are processed
  in a sorted-by-`Name` order, clustering is single-link union-find over a
  fixed sorted pair scan, the union-find always attaches the higher root to
  the lower so the representative is scan-direction independent, and the
  canonical-name pick is longest-then-lexically-lowest. A test runs the
  resolver twice and asserts identical output.
- Conservative by design (KG3-8): same-`Type`-only merging (a person is
  never folded into an org — asserted with two identical-vector entities of
  different types that do NOT merge), a high default `Threshold` (0.92),
  and opt-in via the `NoopEntityResolver` default. The `EmbeddingEntityResolver`
  is exercised only against a scripted embedder returning fixed vectors —
  never a live embedding call.
- Out of scope as planned: the global-search eval harness (25-02);
  `docs/graphrag.md` (25-03).

## Self-Check: PASSED

- `graph/resolve.go`, `graph/resolve_test.go`, `rag/resolve_test.go` present
  in the working tree (`/tmp/llm-agent-rag`).
- `graph/graph.go`, `rag/options.go`, `rag/system.go`, `rag/import.go` all
  modified in the working tree.
- No commits made — per operator instruction, all changes left uncommitted
  for a separate commit.

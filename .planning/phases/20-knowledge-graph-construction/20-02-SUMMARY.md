---
phase: 20-knowledge-graph-construction
plan: 02
type: execute
status: complete
completed: 2026-05-18
repo: llm-agent-rag
requirements: [RAG-GRAPH-02]
---

> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent-rag` current docs.


# Summary: 20-02 deterministic extractor + Canonicalize + Import wiring

## Objective

Complete RAG-GRAPH-02 — a deterministic non-LLM `EntityExtractor`,
cross-chunk `Canonicalize` (exact-match entity merge with provenance), and
graph extraction wired into `Import` surfacing the graph on `ImportResult`.

## Delivered

- `graph/dictionary.go` — `DictionaryEntityExtractor{Terms map[string]string}`:
  a deterministic, zero-LLM `EntityExtractor` matching a caller-supplied
  gazetteer (term → type) case-insensitively against chunk text;
  co-occurring terms yield a `co-occurs` relation. Output order is stable
  (gazetteer terms sorted).
- `graph/canonicalize.go` — `Canonicalize(entities, relations) Graph`:
  merges entities by `(normalized name, type)` exact-match, assigns stable
  IDs (`type:normname`), concatenates-dedups descriptions, unions
  provenance; resolves relation endpoints (names) to canonical entity IDs
  and drops relations with an unknown endpoint; merges relations by
  `(Source, Relation, Target)` with summed `Weight` and unioned
  provenance. Deterministically ordered output.
- `ingest.ImportResult` gained `Graph *graph.Graph`.
- `rag.Options.EntityExtractor` + `System.entityExtractor` (set in `New`).
- `rag/import.go`: post-split, per-chunk `Extract` when an extractor is
  configured, accumulating raw entities/relations; after the document
  loop, `graph.Canonicalize` → `res.Graph`. A nil extractor leaves
  `res.Graph` nil and `Import` behaves exactly as before. An `Extract`
  error aborts the import with a wrapped error.

## Files

- `graph/dictionary.go`, `graph/canonicalize.go` — new.
- `graph/dictionary_test.go`, `graph/canonicalize_test.go` — new:
  deterministic extraction, cross-chunk entity merge, relation merge +
  endpoint resolution + unknown-endpoint drop.
- `ingest/types.go` — `graph` import; `ImportResult.Graph`.
- `rag/options.go` — `graph` import; `Options.EntityExtractor`.
- `rag/system.go` — `graph` import; `System.entityExtractor` wired in `New`.
- `rag/import.go` — `graph` import; post-split extraction + `Canonicalize`.
- `rag/graph_test.go` — new: `Import`-extracts-graph (provenance spanning
  documents) and no-extractor-leaves-graph-nil.

## Verification

All `<verify>` commands run, all green:

- `GOWORK=off go build ./...` — BUILD OK
- `GOWORK=off go vet ./...` — VET OK
- `GOWORK=off go test ./graph ./rag ./ingest -count=1` — ok
- `GOWORK=off go test ./... -count=1` — all packages ok (no FAIL)
- `git diff --stat go.mod go.sum` — empty (no new dependency)
- core facade (from the core repo `llm-agent`): `GOWORK=off go vet
  ./rag/... && go test ./rag/...` — ok

## Notes

- Phase boundary (per 20-RESEARCH Decision 4): Phase 20 produces and
  *returns* the canonicalized graph on `ImportResult.Graph` — it does not
  persist it. Phase 21's `store.GraphStore` is the persistence target;
  `Import` will additionally call `UpsertGraph` once the store exists.
- Graph extraction sees post-redaction text — the redactor runs before the
  splitter, so PII never reaches the graph.
- v0.7 limitation (KG-6): canonicalization is exact-match only. A name
  shared by multiple typed entities resolves deterministically (sorted —
  highest id wins) but is a known simplification; fuzzy resolution is v0.8.
- No new module dependency.

## Phase 20 status

Both slices complete. RAG-GRAPH-01 (the `graph` package, `EntityExtractor`
seam, `LLMEntityExtractor`) and RAG-GRAPH-02 (`DictionaryEntityExtractor`,
`Canonicalize`, `Import` wiring) are delivered. Phase 20 is complete; the
`graph` package is ready for Phase 21 (`store.GraphStore`).

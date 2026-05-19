---
phase: 20-knowledge-graph-construction
plan: 01
type: execute
status: complete
completed: 2026-05-18
repo: llm-agent-rag
requirements: [RAG-GRAPH-01]
---

# Summary: 20-01 graph package + LLMEntityExtractor

## Objective

Open the v0.7 GraphRAG milestone with the `graph` package — the `Entity` /
`Relation` / `Graph` types, an `EntityExtractor` seam, and an
`LLMEntityExtractor` that extracts entities and typed relations from a
chunk's text via `generate.Model`.

## Delivered

- `graph` package (new, leaf — imports only stdlib + the `generate` seam):
  - `Entity{ID, Name, Type, Description, SourceChunkIDs, Metadata}`,
    `Relation{ID, Source, Target, Relation, Description, SourceChunkIDs,
    Weight}`, `Graph{Entities, Relations}`.
  - `EntityExtractor` interface — `Extract(ctx, chunkID, text) ([]Entity,
    []Relation, error)`.
- `graph/extract.go` — `LLMEntityExtractor{Model generate.Model}`:
  - a fixed system prompt asking for *salient* entities/relations (salience
    guidance pre-empts graph bloat), pipe-delimited one record per line.
  - `parseExtraction` — lenient line parser: classifies by the first
    pipe-field (`ENTITY`/`RELATION`, case-insensitive), drops lines that
    match neither or have too few fields, never errors on malformed text.
  - extracted entities/relations carry the chunk id as provenance;
    relations get `Weight: 1` and hold endpoint *names* (resolved to ids by
    `Canonicalize` in 20-02).
  - nil `Model` → `ErrEntityExtractorModelRequired`; a model error
    propagates.

## Files

- `graph/graph.go` — new: core types + `EntityExtractor` seam.
- `graph/extract.go` — new: `LLMEntityExtractor`, prompt, lenient parser.
- `graph/graph_test.go` — new: type-shape test.
- `graph/extract_test.go` — new: `scriptedModel` stub; clean-output,
  lenient-parsing (junk lines / missing fields / too-few-field relation),
  nil-model, and model-error tests.

## Verification

All `<verify>` commands run, all green:

- `GOWORK=off go build ./...` — BUILD OK
- `GOWORK=off go vet ./...` — VET OK
- `GOWORK=off go test ./graph -count=1` — ok
- `GOWORK=off go test ./... -count=1` — all packages ok (no FAIL)
- `git diff --stat go.mod go.sum` — empty (no new dependency)
- core facade (run from the core repo `llm-agent`): `GOWORK=off go vet
  ./rag/... && go test ./rag/...` — ok

## Notes

- `LLMEntityExtractor` is tested only with a scripted model (parser
  correctness) — LLM extraction is non-deterministic; the deterministic
  `DictionaryEntityExtractor` (20-02) is what `Import`-level and later
  phases use for reproducible graphs.
- The `scriptedModel` stub lives in `extract_test.go` (package `graph`) and
  is reusable by 20-02's test files in the same package.
- No new module dependency — `graph` is stdlib + the existing `generate`
  seam.

# Phase 20 Research: Knowledge-graph construction

**Researched:** 2026-05-18
**Phase:** 20 — knowledge-graph construction
**Requirements:** RAG-GRAPH-01, RAG-GRAPH-02
**Repos:** `llm-agent-rag`
**Upstream:** `.planning/research/v0.7-graphrag-SUMMARY.md` (the GraphRAG
domain research; KG-1..KG-7 keystone decisions). This document grounds
Phase 20 in the codebase and fixes the phase-boundary decision.

## Current state (codebase scan)

- No `graph` package exists. `graph` will be a new leaf-ish package, peer
  of `retrieve`/`rerank`/`eval`.
- `advanced/llm.go` is the prompt+parse template: `ExpandQuery` /
  `GenerateHypothetical` build a `generate.Request` from a fixed prompt,
  call `model.Generate`, and parse the response line-by-line (split on
  `\n`, `TrimLeft` list markers, drop empties, dedup). `advanced` also has
  `ErrModelRequired`. The `LLMEntityExtractor` mirrors this exactly.
- `generate.Model` — `Generate(ctx, Request) (Response, error)`,
  `Request{SystemPrompt, Messages, Metadata}`, `Response{Text, Usage}`.
- `ingest.ImportResult` — `{Documents, Chunks int; ChunkIDs []string;
  Metrics obs.Metrics; Redactions []guard.Redaction}`. Phase 20 adds a
  graph field.
- `rag/import.go` `Import` loop: per document — optional `Redact`, optional
  `RemoveByFilter`, `splitter.Split(doc, maxChars)`, then per-chunk
  `embedder.Embed`; finally `store.Upsert` + the `OnImport` observer.
  Graph extraction is a **new post-split, per-chunk stage** in this loop,
  alongside (not replacing) embedding.
- The project's opt-in-seam pattern (v0.6 `Redactor`, `InjectionScanner`):
  a nil seam on `Options` means the feature is off; existing behavior is
  unchanged. Graph extraction follows this — nil `EntityExtractor` = no
  extraction.
- Determinism discipline: an LLM-backed type is unit-tested with a scripted
  `generate.Model`, never a live call (`fakeModel`/`scriptedModel`).

## Decision 1 — the `graph` package (RAG-GRAPH-01)

A new `graph` package. Phase 20 introduces:

```go
package graph

// Entity is a node — a salient real-world thing named in the corpus.
type Entity struct {
    ID             string         // canonical id (assigned by Canonicalize)
    Name           string
    Type           string         // free-form label: "person", "org", ...
    Description    string
    SourceChunkIDs []string        // provenance — drives GC on re-ingest
    Metadata       map[string]any
}

// Relation is a directed, typed edge between two entities.
type Relation struct {
    ID             string
    Source, Target string         // canonical entity IDs (post-Canonicalize)
    Relation       string         // the edge label
    Description    string
    SourceChunkIDs []string
    Weight         float64
}

// Graph is a canonicalized entity/relation set.
type Graph struct {
    Entities  []Entity
    Relations []Relation
}

// EntityExtractor turns one chunk's text into raw graph primitives.
// Implementations may be LLM-backed or deterministic.
type EntityExtractor interface {
    Extract(ctx context.Context, chunkID, text string) ([]Entity, []Relation, error)
}
```

`graph` imports only stdlib (and, in 20-01, `generate` for `LLMEntityExtractor`).

## Decision 2 — `LLMEntityExtractor`: prompt + lenient pipe-delimited parse

`LLMEntityExtractor{Model generate.Model}` mirrors `advanced.ExpandQuery`:
one `generate.Model` call per chunk, a fixed system prompt, line-by-line
parse. The wire format is **pipe-delimited, one record per line** (robust,
matches the repo's line-parse precedent):

```
ENTITY | name | type | description
RELATION | source-name | target-name | relation | description
```

Parsing is lenient: split on `|`, trim fields, skip lines that do not start
with `ENTITY`/`RELATION` (case-insensitive) or have too few fields. A
malformed line is dropped, never fatal. The extractor returns raw entities
(pre-canonical, `ID` empty, `SourceChunkIDs=[chunkID]`) and relations whose
`Source`/`Target` are entity **names** (resolved to IDs by `Canonicalize`).
Nil model → `ErrEntityExtractorModelRequired`.

## Decision 3 — deterministic extractor + canonicalization (RAG-GRAPH-02)

- `DictionaryEntityExtractor{Terms map[string]string}` — a deterministic,
  zero-LLM extractor: a caller-supplied gazetteer (term → type) matched
  case-insensitively against the chunk; matched terms become entities,
  co-occurrence within a chunk becomes a generic `mentions`/`co-occurs`
  relation. Mirrors `HeuristicDecomposer` vs `LLMDecomposer` — gives the
  default path a no-LLM story and gives every test a deterministic graph.
- `Canonicalize(perChunk ...) Graph` — merges raw per-chunk extractions:
  entities merge by `(normalizeName(Name), Type)` (case-fold + trim);
  merged entity gets a stable `ID`, concatenated-deduped `Description`,
  unioned `SourceChunkIDs`. Relation endpoints resolve from names to
  canonical entity IDs; relations merge by `(Source, Target, Relation)`
  with unioned provenance and summed/maxed `Weight`. Exact-match only —
  fuzzy/embedding resolution is deferred to v0.8 (KG-6).

## Decision 4 — phase-boundary: where the graph goes in Phase 20

Phase 21 builds `store.GraphStore`; Phase 20 has no store to persist into.
Resolution: **Phase 20 wires extraction into `Import` and surfaces the
canonicalized graph on `ImportResult`** — it is produced and returned, not
persisted. `ingest.ImportResult` gains `Graph *graph.Graph` (nil when no
extractor is configured). `Import`, when `Options.EntityExtractor` is set,
runs `Extract` per chunk post-split, `Canonicalize`s across the document
set, and attaches the result to `ImportResult.Graph`. Phase 21's
`GraphStore` then becomes the persistence target — `Import` will
additionally call `UpsertGraph` once the store exists. This keeps Phase 20
a complete, independently testable vertical slice (extract → canonicalize →
return) and leaves a clean seam for Phase 21.

Extraction is **opt-in**: a nil `Options.EntityExtractor` leaves `Import`
behaving exactly as today (KG-4 — default behavior unchanged).

## Slice breakdown

- **20-01** — `graph` package: `Entity`/`Relation`/`Graph`,
  `EntityExtractor` seam, `LLMEntityExtractor` (prompt + lenient parse),
  scripted-model unit tests incl. malformed output. (RAG-GRAPH-01)
- **20-02** — `DictionaryEntityExtractor` (deterministic); `Canonicalize`
  (exact-match merge + provenance + endpoint resolution); wire extraction
  into `Import` as a post-split stage; `ImportResult.Graph`. (RAG-GRAPH-02)

## Risks / notes

- 20-02 depends on 20-01 (`Canonicalize` consumes `EntityExtractor` output;
  the `Import` wiring needs the seam).
- LLM extraction is non-deterministic — `LLMEntityExtractor` is tested only
  via a scripted model (parser correctness); the deterministic
  `DictionaryEntityExtractor` is what `Import`-level and Phase 21/22 tests
  use for reproducible graphs.
- No new module dependency — `graph` is stdlib + the existing `generate`
  seam. The standard `git diff --stat go.mod go.sum` (empty) check applies.
- `ImportResult.Graph` carries the whole extracted graph. Acceptable — it
  is the produced artifact and mirrors `ChunkIDs` already returning all
  ids; per-import graphs at this SDK's scale are small.

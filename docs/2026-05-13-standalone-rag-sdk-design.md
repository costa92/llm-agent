# Standalone RAG SDK Design

Date: 2026-05-13
Planning repo: `github.com/costa92/llm-agent`
Target repo: `github.com/costa92/llm-agent-rag`
Status: proposed

## Goal

Design a standalone Go RAG SDK that can later live in its own repository and
support three first-class workflows:

- abstract data import
- abstract data retrieval
- answer generation with custom LLMs and custom prompt templates

The SDK must not depend on `llm-agent` core types in its core packages. Any
`llm-agent` integration must live behind adapters.

## Scope

### In scope

- a pure Go SDK layout
- abstract import pipeline
- abstract retrieval pipeline
- pluggable text-generation interface
- pluggable prompt-template interface
- one default chunker
- one default embedder
- one default store:
  - `InMemoryStore`
- one default QA prompt template
- an adapter boundary for `llm-agent`
- migration mapping from the current `rag/` package

### Out of scope for v0.1

- HTTP service layer
- CLI tooling
- built-in production vector backends such as pgvector, Qdrant, or Milvus
- MQE / HyDE in core
- rerankers
- filter DSL design beyond a minimal `map[string]any`
- async ingestion orchestration
- distributed indexing jobs

## Design Choice

Build a standalone SDK with a small core and explicit adapter packages.

Chosen shape:

- core packages own import, split, embed, store, retrieve, prompt, and ask
- core packages define their own generator interface
- `llm-agent` integration is isolated in `adapter/llmagent`

Why:

- keeps the standalone SDK reusable outside the `llm-agent` ecosystem
- preserves current good seams already present in `rag`:
  - `Chunker`
  - `Embedder`
  - `VectorStore`
- avoids leaking `llm.ChatModel` or `agents.Tool` into the core public API

## Recommended Repository Layout

```text
llm-agent-rag/
├── go.mod
├── README.md
├── doc.go
├── ingest/
│   ├── source.go
│   ├── splitter.go
│   ├── types.go
│   └── import.go
├── embed/
│   ├── embedder.go
│   ├── hash.go
│   └── vector.go
├── store/
│   ├── types.go
│   ├── store.go
│   └── inmemory.go
├── prompt/
│   ├── template.go
│   ├── default.go
│   └── types.go
├── generate/
│   ├── model.go
│   └── types.go
├── rag/
│   ├── options.go
│   ├── system.go
│   ├── import.go
│   ├── retrieve.go
│   ├── ask.go
│   └── errors.go
└── adapter/
    └── llmagent/
        ├── model.go
        └── tool.go
```

## Core Concepts

### 1. Import

The SDK must not assume that data comes from a single raw string.

Core import abstraction:

```go
type Document struct {
    ID       string
    Title    string
    Content  string
    Metadata map[string]any
}

type Source interface {
    Next(ctx context.Context) (Document, error)
}
```

Supported v0.1 import shapes:

- `Import(ctx, []Document, opts)`
- `ImportFrom(ctx, Source, opts)`

Import pipeline:

1. read document
2. split into chunks
3. derive stable chunk IDs
4. embed each chunk
5. upsert into the store

### 2. Split

Chunking remains a first-class seam:

```go
type Splitter interface {
    Split(doc Document, maxChars int) []Chunk
}
```

Default implementation:

- `CharSplitter`

Requirements:

- stable chunk ordering
- deterministic output
- overlap support
- chunk metadata enrichment:
  - source document ID
  - chunk index
  - chunk total

### 3. Embed

Embedding stays provider-agnostic:

```go
type Vector []float32

type Embedder interface {
    Embed(ctx context.Context, text string) (Vector, error)
    Dimension() int
}
```

Default implementation:

- `HashEmbedder`

Rationale:

- keeps v0.1 zero-dependency and runnable in tests
- lets downstream users swap in OpenAI, Ollama, Voyage, or custom embedders

### 4. Store

The SDK only ships one built-in store in v0.1:

- `InMemoryStore`

Store abstraction:

```go
type StoredChunk struct {
    ID        string
    Namespace string
    DocID     string
    Title     string
    Content   string
    Vector    Vector
    Metadata  map[string]any
}

type Query struct {
    Namespace string
    Vector    Vector
    TopK      int
    Filters   map[string]any
}

type Hit struct {
    Chunk StoredChunk
    Score float64
}

type Store interface {
    Upsert(ctx context.Context, chunks []StoredChunk) error
    Search(ctx context.Context, q Query) ([]Hit, error)
    Get(ctx context.Context, id string) (StoredChunk, error)
    Remove(ctx context.Context, id string) error
    Stats(ctx context.Context, namespace string) (Stats, error)
}
```

Constraints:

- store dimension must be fixed
- dimension mismatch must fail explicitly
- namespace is supported in the API from day one
- filters may remain minimal in v0.1

### 5. Retrieve

Raw retrieval must remain usable without any LLM.

Core call shape:

```go
type SearchOptions struct {
    TopK      int
    Namespace string
    Filters   map[string]any
}

func (s *System) Retrieve(ctx context.Context, query string, opts SearchOptions) ([]store.Hit, error)
```

Behavior:

- embed the query
- perform vector search
- sort hits descending
- return retrieval results without generation

This makes retrieval independently usable by applications that do not want
answer synthesis.

### 6. Generate

The core SDK must not depend on `llm.ChatModel`.

SDK-local generation interface:

```go
type Message struct {
    Role    string
    Content string
}

type Request struct {
    SystemPrompt string
    Messages     []Message
    Metadata     map[string]any
}

type Response struct {
    Text string
}

type Model interface {
    Generate(ctx context.Context, req Request) (Response, error)
}
```

This is intentionally small because current `rag` only needs prompt-in,
text-out one-shot generation.

### 7. Prompt Templates

Prompt construction must be a first-class seam, not hard-coded string
concatenation in `Ask`.

```go
type RenderContext struct {
    Question  string
    Namespace string
    Hits      []store.Hit
    Metadata  map[string]any
}

type Template interface {
    Render(ctx context.Context, rc RenderContext) (generate.Request, error)
}
```

Default implementation:

- `DefaultQATemplate`

Requirements:

- customizable system prompt
- customizable answer instructions
- deterministic chunk rendering order
- optional citation instruction

### 8. Ask

Answer generation sits on top of retrieval and templating:

```go
type AskOptions struct {
    Search   SearchOptions
    Template prompt.Template
    Metadata map[string]any
}

type Answer struct {
    Text   string
    Hits   []store.Hit
    Prompt generate.Request
}

func (s *System) Ask(ctx context.Context, question string, opts AskOptions) (Answer, error)
```

Behavior:

1. retrieve hits
2. render prompt through the selected template
3. call the configured generator
4. return answer text plus supporting retrieval context

## Stable Identity Model

Current `chunk_N` IDs are not sufficient for a standalone SDK.

v0.1 requirement:

- every imported source document has a stable source ID
- chunk IDs derive from:
  - source ID
  - chunk index

Recommended shape:

- `doc-123#chunk-0`
- `doc-123#chunk-1`

Why:

- deterministic re-import
- safe deletion
- provenance tracking
- easier dedupe

## Error Behavior

The standalone SDK needs its own narrow error set:

- `ErrEmptyQuery`
- `ErrModelRequired`
- `ErrDimensionMismatch`
- `ErrChunkerRequired` only if constructor allows nil without default
- `ErrNotFound`

Guidelines:

- keep errors typed or sentinel-based
- include pipeline stage in wrapped errors:
  - import
  - split
  - embed
  - upsert
  - search
  - generate

## Adapter Boundary

Everything that depends on `llm-agent` moves behind an adapter package.

Examples:

- `llm.ChatModel -> generate.Model`
- `rag.System -> agents.Tool`

This preserves:

- standalone SDK purity
- existing `llm-agent` ergonomics
- future provider reuse outside `llm-agent`

## Migration Mapping From Current `rag/`

Move unchanged:

- `rag/chunk.go`
- `rag/embedder.go`
- `rag/store.go`
- `rag/chunk_test.go`
- `rag/embedder_test.go`
- `rag/store_test.go`

Refactor into core:

- `rag/rag.go`
  - replace `llm.ChatModel` with SDK-local `generate.Model`
- `rag/advanced.go`
  - if retained later, make it depend on SDK-local generation
- `rag/doc.go`
  - remove `llm-agent` portability language

Move into adapter layer:

- `rag/tool.go`
- `rag/tool_test.go`
- `rag/llm_embedder_test.go` bridge pattern

## Testing Strategy

### Core SDK tests

- splitter determinism
- hash embedder dimensions and cosine behavior
- in-memory store CRUD and ranking
- import pipeline happy path
- retrieve happy path
- ask happy path with fake generator
- prompt template rendering
- dimension mismatch failures
- stable chunk ID generation

### Adapter tests

- `llm.ChatModel` adapter request/response mapping
- `agents.Tool` wrapping behavior

## v0.1 Success Criteria

The standalone SDK is successful when it can:

1. import multiple documents through an abstract source
2. split and embed them with default implementations
3. index them into an in-memory store
4. retrieve ranked hits for a query
5. generate an answer using a user-provided model adapter
6. let callers swap prompt templates without modifying retrieval code
7. integrate back into `llm-agent` through an adapter instead of direct core imports

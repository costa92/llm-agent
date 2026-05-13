# Standalone RAG SDK Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Design and scaffold a standalone Go RAG SDK with abstract import, retrieval, custom LLM generation, and custom prompt-template seams, while keeping `llm-agent` integration isolated behind adapters.

**Architecture:** The implementation extracts reusable RAG primitives into a new standalone module shape. Core packages own import, split, embed, store, retrieve, and ask orchestration using SDK-local interfaces. `llm-agent` integration points move to adapter packages so the standalone core does not import `llm-agent` types.

**Tech Stack:** Go 1.26, stdlib-only defaults, existing `rag` package as migration source, adapter seam for `llm-agent`

---

## File Map

### Planning / design artifacts

- Create: `docs/2026-05-13-standalone-rag-sdk-design.md`
- Create: `docs/2026-05-13-standalone-rag-sdk-implementation-plan.md`

### Future standalone SDK module

- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/go.mod`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/README.md`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/doc.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/ingest/source.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/ingest/splitter.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/ingest/types.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/ingest/import.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/embed/embedder.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/embed/hash.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/embed/vector.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/store/types.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/store/store.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/store/inmemory.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/prompt/template.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/prompt/default.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/prompt/types.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/generate/model.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/generate/types.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/rag/options.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/rag/system.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/rag/import.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/rag/retrieve.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/rag/ask.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/rag/errors.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/adapter/llmagent/model.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/adapter/llmagent/tool.go`

### Existing source files to reference during extraction

- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/chunk.go`
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/embedder.go`
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/store.go`
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/rag.go`
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/advanced.go`
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/tool.go`

## Task 1: Initialize standalone SDK module and public package skeleton

**Files:**
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/go.mod`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/README.md`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/doc.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/ingest/types.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/embed/vector.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/generate/types.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/prompt/types.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/store/types.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/rag/errors.go`

- [ ] **Step 1: Create the failing package-compilation test scaffold**

Add minimal tests:

```go
package rag_test

import "testing"

func TestModulePackagesCompile(t *testing.T) {}
```

- [ ] **Step 2: Run test to verify the new module is not yet wired**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./...`
Expected: FAIL with missing files or missing package symbols.

- [ ] **Step 3: Create the module and public type skeleton**

Define:

- `ingest.Document`
- `embed.Vector`
- `store.StoredChunk`, `store.Hit`, `store.Stats`
- `generate.Message`, `generate.Request`, `generate.Response`
- `prompt.RenderContext`
- sentinel errors in `rag/errors.go`

- [ ] **Step 4: Re-run package compilation**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./...`
Expected: PASS or fail only on later intentionally missing interfaces.

- [ ] **Step 5: Commit**

```bash
git add .
git commit -m "feat: scaffold standalone rag sdk module"
```

## Task 2: Implement import abstractions and default splitter

**Files:**
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/ingest/source.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/ingest/splitter.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/ingest/import.go`
- Test: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/ingest/splitter_test.go`
- Test: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/ingest/import_test.go`

- [ ] **Step 1: Write failing splitter tests**

Cover:

- empty input
- single chunk
- multi-chunk paragraph split
- overlap behavior
- deterministic output

- [ ] **Step 2: Run splitter tests to verify they fail**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./ingest -run 'Test(CharSplitter|StableChunkIDs)' -count=1`
Expected: FAIL with undefined `CharSplitter` or wrong behavior.

- [ ] **Step 3: Port and adapt the current chunker**

Base implementation on:

- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/chunk.go`

Adjust it to operate on `ingest.Document` and emit `ingest.Chunk` with stable IDs like:

- `<docID>#chunk-0`
- `<docID>#chunk-1`

- [ ] **Step 4: Re-run splitter tests**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./ingest -run 'Test(CharSplitter|StableChunkIDs)' -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add ingest
git commit -m "feat: add import abstractions and default splitter"
```

## Task 3: Implement embed abstractions and hash embedder

**Files:**
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/embed/embedder.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/embed/hash.go`
- Test: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/embed/hash_test.go`

- [ ] **Step 1: Write failing embedder tests**

Cover:

- default dimension fallback
- deterministic vectors
- cosine similarity bounds
- zero vector normalization edge case

- [ ] **Step 2: Run embed tests to verify they fail**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./embed -count=1`
Expected: FAIL with undefined `HashEmbedder` or helpers.

- [ ] **Step 3: Port the current hash embedder**

Base implementation on:

- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/embedder.go`

- [ ] **Step 4: Re-run embed tests**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./embed -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add embed
git commit -m "feat: add sdk embedder abstractions"
```

## Task 4: Implement store abstractions and in-memory backend

**Files:**
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/store/store.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/store/inmemory.go`
- Test: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/store/inmemory_test.go`

- [ ] **Step 1: Write failing store tests**

Cover:

- upsert/get/remove
- topK ranking
- dimension mismatch
- namespace isolation
- stats reporting

- [ ] **Step 2: Run store tests to verify they fail**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./store -count=1`
Expected: FAIL with undefined `Store` or `InMemoryStore`.

- [ ] **Step 3: Port and adapt the in-memory store**

Base implementation on:

- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/store.go`

Adjustments:

- store `Namespace`
- keep explicit dimension checks
- accept batch `Upsert`

- [ ] **Step 4: Re-run store tests**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./store -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add store
git commit -m "feat: add in-memory vector store"
```

## Task 5: Implement generation and prompt-template seams

**Files:**
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/generate/model.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/prompt/template.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/prompt/default.go`
- Test: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/prompt/default_test.go`

- [ ] **Step 1: Write failing prompt-template tests**

Cover:

- rendered request contains system prompt
- rendered request contains retrieved chunks in stable order
- rendered request contains question text
- configurable citation instruction

- [ ] **Step 2: Run prompt tests to verify they fail**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./prompt -count=1`
Expected: FAIL with undefined `Template` or `DefaultQATemplate`.

- [ ] **Step 3: Implement the prompt seams**

Add:

- `generate.Model`
- `prompt.Template`
- `prompt.DefaultQATemplate`

Requirements:

- deterministic rendering
- no dependency on `llm-agent`

- [ ] **Step 4: Re-run prompt tests**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./prompt -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add generate prompt
git commit -m "feat: add generation and prompt template seams"
```

## Task 6: Implement core RAG system import, retrieve, and ask flows

**Files:**
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/rag/options.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/rag/system.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/rag/import.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/rag/retrieve.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/rag/ask.go`
- Test: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/rag/system_test.go`

- [ ] **Step 1: Write failing system tests**

Cover:

- `Import` with explicit document slice
- `ImportFrom` via source iterator
- `Retrieve` happy path
- `Ask` happy path with fake generator
- `ErrEmptyQuery`
- `ErrModelRequired`

- [ ] **Step 2: Run system tests to verify they fail**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./rag -count=1`
Expected: FAIL with undefined `System` or wrong pipeline behavior.

- [ ] **Step 3: Port the current orchestration logic**

Base implementation on:

- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/rag.go`

Required changes:

- replace `llm.ChatModel` with SDK-local `generate.Model`
- split orchestration into `Import`, `Retrieve`, and `Ask`
- use stable chunk IDs from `ingest`
- return `Answer{Text, Hits, Prompt}`

- [ ] **Step 4: Re-run system tests**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./rag -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add rag
git commit -m "feat: add standalone rag system"
```

## Task 7: Add `llm-agent` adapter package

**Files:**
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/adapter/llmagent/model.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/adapter/llmagent/tool.go`
- Test: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/adapter/llmagent/model_test.go`
- Test: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/adapter/llmagent/tool_test.go`

- [ ] **Step 1: Write failing adapter tests**

Cover:

- `llm.ChatModel` to `generate.Model` request mapping
- response text passthrough
- `AsTool` action coverage:
  - add_text
  - search
  - ask
  - remove
  - stats

- [ ] **Step 2: Run adapter tests to verify they fail**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./adapter/llmagent -count=1`
Expected: FAIL with undefined adapter types.

- [ ] **Step 3: Move `tool.go` logic into the adapter**

Base implementation on:

- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/tool.go`

Required changes:

- depend on the new standalone SDK types
- keep `llm-agent` imports local to adapter package

- [ ] **Step 4: Re-run adapter tests**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./adapter/llmagent -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add adapter/llmagent
git commit -m "feat: add llm-agent rag adapters"
```

## Task 8: Write README and usage examples

**Files:**
- Modify: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/README.md`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/examples/basic_import_and_ask_test.go`

- [ ] **Step 1: Write failing example test**

Add an example that:

- builds a system with `HashEmbedder`
- imports two documents
- retrieves hits
- asks a question with a fake generator

- [ ] **Step 2: Run the example test to verify it fails**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./... -run Example -count=1`
Expected: FAIL until README/example types match implementation.

- [ ] **Step 3: Document the public API**

README sections:

- project goal
- import flow
- retrieval flow
- custom generator
- custom prompt template
- `llm-agent` adapter note
- v0.1 limits

- [ ] **Step 4: Re-run example tests**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./... -run Example -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add README.md examples
git commit -m "docs: add standalone rag sdk usage guide"
```

## Task 9: Final verification and release-readiness check

**Files:**
- Review: entire module

- [ ] **Step 1: Run the full test suite**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./... -count=1`
Expected: PASS

- [ ] **Step 2: Verify core packages do not import `llm-agent`**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && rg -n 'github.com/costa92/llm-agent' ingest embed store prompt generate rag`
Expected: no matches

- [ ] **Step 3: Verify adapter package is the only `llm-agent` integration seam**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && rg -n 'github.com/costa92/llm-agent' adapter`
Expected: matches only under `adapter/llmagent`

- [ ] **Step 4: Commit final cleanup if needed**

```bash
git add .
git commit -m "chore: finalize standalone rag sdk extraction"
```

## Spec Coverage Check

This plan covers:

- abstract import via `Import` and `ImportFrom`
- abstract retrieval via `Retrieve`
- custom LLM via `generate.Model`
- custom prompt templates via `prompt.Template`
- built-in `InMemoryStore`
- adapter isolation for `llm-agent`

Deliberately deferred from the spec:

- production vector backends
- MQE / HyDE in core
- HTTP service
- CLI

## Execution Handoff

Plan complete and saved to `docs/2026-05-13-standalone-rag-sdk-implementation-plan.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?

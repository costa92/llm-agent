> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent-rag` current docs.

# Phase 08-01 Summary

Date: 2026-05-14
Repos: `llm-agent-rag`, `llm-agent`
Plan: [08-01-PLAN.md](/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/phases/08-rag-core-contract-hardening/08-01-PLAN.md)

## Objective

Harden the initial standalone/core RAG contract by moving real filtering and
diagnostics behavior into `llm-agent-rag`, then align the core `rag` facade
against the new standalone release.

## Delivered

- In `llm-agent-rag`:
  - added real metadata filtering to the default `InMemoryStore`
  - added explicit `SecurityFilters` to the standalone retrieval contract
  - added machine-readable `Citations`, `Diagnostics`, and `Trace` to
    `rag.Answer`
  - added and expanded tests for:
    - metadata filter behavior
    - security filter enforcement
    - ask-path citations and trace propagation
- Released standalone changes as `github.com/costa92/llm-agent-rag v0.1.2`
- In core `llm-agent`:
  - bumped `github.com/costa92/llm-agent-rag` from `v0.1.0` to `v0.1.2`
  - aligned `rag.SearchOptions` with standalone filter propagation
  - updated the `rag` tool `ask` response to surface:
    - `answer`
    - `citations`
    - `diagnostics`
    - `trace`
  - kept the historical string-returning `Ask(...)` facade intact for existing
    callers

## Files

### Standalone repo

- `/tmp/llm-agent-rag/store/store.go`
- `/tmp/llm-agent-rag/store/inmemory.go`
- `/tmp/llm-agent-rag/store/inmemory_test.go`
- `/tmp/llm-agent-rag/rag/options.go`
- `/tmp/llm-agent-rag/rag/retrieve.go`
- `/tmp/llm-agent-rag/rag/system.go`
- `/tmp/llm-agent-rag/rag/ask.go`
- `/tmp/llm-agent-rag/rag/system_test.go`
- `/tmp/llm-agent-rag/CHANGELOG.md`

### Core repo

- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/go.mod`
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/go.sum`
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/rag.go`
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/tool.go`
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/tool_test.go`

## Verification

Standalone verification:

```bash
cd /tmp/llm-agent-rag
GOWORK=off GOCACHE=/tmp/go-build go test ./store ./rag -count=1
GOWORK=off GOCACHE=/tmp/go-build go test ./rag ./prompt ./examples -count=1
GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1
```

Core verification:

```bash
cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent
GOWORK=off GOCACHE=/tmp/go-build go test ./rag -count=1
GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1
```

Result:

- standalone targeted tests: pass
- standalone full test suite: pass
- core `rag` targeted tests: pass
- core full test suite: pass

## Notes

- The original Phase 8 plan split this work into `08-01` through `08-04`, but
  the first execution pass completed the minimum viable contract thread across
  both repos in one continuous cycle.
- The next natural step is Phase 9:
  source-aware ingestion metadata and index lifecycle semantics.

# Phase 12-03 Summary

Date: 2026-05-15
Repo: `llm-agent-rag`
Plan: [12-03-PLAN.md](/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/phases/12-persistence-tracing-and-backend-conformance/12-03-PLAN.md)

## Objective

Add an observation surface to the rag facade so OTel adapters in
`llm-agent-otel` (sister repo) can attach to `Import`, `Retrieve`,
and `Ask` without touching internals. Closes RAG-OPS-02.

## Delivered

- new file `rag/observer.go` exports:
  - `ImportTrace{Namespace, Documents, Chunks, ChunkIDs,
    EmbedCount, ReplaceSource, RemovedChunks}` — first trace surface
    for `Import` (which previously had none)
  - `Observer` struct with three optional `func` fields:
    `OnImport`, `OnRetrieve`, `OnAsk`. Nil-safe; unset callbacks
    skip with one branch and zero allocation
- `rag.Options.Observer` exposed as the wiring point; passed through
  to `*System` in `New(...)`
- `System.Import` populates `ImportTrace` as the loop runs
  (tracking embed calls and removed-chunk counts during
  ReplaceSource) and fires `OnImport` after the upsert succeeds
- `System.retrieve` (internal helper) fires `OnRetrieve` after the
  underlying retriever returns. This means `Ask` transitively fires
  `OnRetrieve` then `OnAsk`, matching OTel's natural span-nesting
  pattern (retrieve span as a child of ask span)
- `System.Ask` builds the `Answer` into a variable and fires
  `OnAsk` with `answer.Trace` immediately before return
- failure paths short-circuit before the callback fires — observers
  see only successful runs (errors observable via the returned
  error)
- four regression tests in `rag/observer_test.go`:
  - `TestObserverFiresOnImport` — `Documents=2`, `Chunks>0`,
    `EmbedCount==Chunks`, `ChunkIDs` length matches
  - `TestObserverFiresOnRetrieve` — trace's `OriginalQuery`
    matches the supplied query
  - `TestObserverFiresOnAsk` — both `OnRetrieve` (transitive) and
    `OnAsk` fire exactly once; the trace passed to `OnAsk` is
    identical to `answer.Trace`
  - `TestObserverNilCallbacksAreSafe` — `Observer{}` with every
    field nil; no panic across `Import`/`Retrieve`/`Ask`

## Files

- `/tmp/llm-agent-rag/rag/observer.go` (new, ~35 LOC)
- `/tmp/llm-agent-rag/rag/observer_test.go` (new, ~125 LOC)
- `/tmp/llm-agent-rag/rag/options.go` (Observer field)
- `/tmp/llm-agent-rag/rag/system.go` (observer storage on *System)
- `/tmp/llm-agent-rag/rag/import.go` (ImportTrace emission)
- `/tmp/llm-agent-rag/rag/retrieve.go` (OnRetrieve emission)
- `/tmp/llm-agent-rag/rag/ask.go` (Answer variable + OnAsk emission)

## Verification

```bash
cd /tmp/llm-agent-rag
GOWORK=off GOCACHE=/tmp/go-build go build ./...
GOWORK=off GOCACHE=/tmp/go-build go vet ./...
GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1

cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent
GOWORK=off go vet ./rag/...
GOWORK=off go test ./rag/... -count=1
```

Result:

- `go build ./...` (standalone): pass
- `go vet ./...` (standalone): pass
- `go test ./...` (standalone, 14 packages): pass — all four
  observer tests pass, every pre-existing test passes unchanged
- core `go vet ./rag/...` + `go test ./rag/...`: pass

## Notes & Design Decisions

- **Callback-struct, not interface.** `Observer` is a struct of
  optional `func` fields instead of an interface. Reasons: partial
  observers are trivial (leave a field nil), no need for a no-op
  base struct, nil-check is one branch with zero allocation, OTel
  adapter can still implement a single helper struct that assigns
  all three.
- **Transitive `OnRetrieve` from Ask.** `OnRetrieve` fires from
  the internal `retrieve` helper, so `Ask` produces both
  `OnRetrieve` and `OnAsk` events. This matches OTel's natural
  parent-child span model and gives consumers per-operation
  granularity without explicit subscription gymnastics.
- **No `OnPack` hook.** Pack output already lives inside
  `AskTrace` (`PackedChunkIDs`, `DroppedChunkIDs`,
  `RerankedChunkIDs`, `SelectedChunkIDs`). Adding a separate
  `OnPack` would duplicate that signal. Consumers needing
  pack-level granularity pull it from `AskTrace`.
- **No timing in the rag facade.** Span creation and
  `time.Now()` calls are the OTel adapter's responsibility — they
  fit naturally in the wrapped span, and putting them here would
  predetermine the span shape.
- **Success-only.** Errors short-circuit before the callback.
  This keeps the observer surface simple and matches the rag-facade
  contract (errors are first-class returns). OTel adapters that
  need error spans can wrap the rag method directly.

## Next slice

Phase 12 ROADMAP work is complete with this slice. Remaining items
in the pending todos are operational (live-Postgres CI wiring,
adapter/llmagent triage) and can run in parallel with Phase 13.

The natural next step in the milestone is Phase 13 (evaluation +
feedback loop + ecosystem contract). Before kicking that off, we
should:

1. flip Phase 12 status in ROADMAP to `complete`
2. decide whether to commit the accumulated 11-04 → 12-03 work
   before Phase 13 begins
3. align on Phase 13's first slice — likely `13-01` (retrieval +
   grounding regression datasets + CI gates per RAG-OPS-03)

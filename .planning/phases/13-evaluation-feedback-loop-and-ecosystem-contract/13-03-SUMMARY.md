> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent-rag` current docs.

# Phase 13-03 Summary

Date: 2026-05-15
Repo: `llm-agent-rag`
Plan: [13-03-PLAN.md](/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/phases/13-evaluation-feedback-loop-and-ecosystem-contract/13-03-PLAN.md)

## Objective

Close the online-to-offline loop for RAG-OPS-03: capture flagged
production Asks as eval examples in the JSONL format from 13-01, so
production misses become regression cases on the next eval run.

## Delivered

- new package `feedback/`:
  - `Recorder` — concurrent-safe JSONL writer over any `io.Writer`
  - `NewRecorder(out)` — bare constructor for in-memory / network
    sinks
  - `OpenFile(path)` — append-mode file constructor returning a
    Recorder + closer
  - `Capture(eval.Example) error` — writes one JSON line, mutex
    around the write so concurrent callers don't interleave
  - `BuildExample(trace, goldDocIDs, goldChunkIDs, notes)` — maps a
    `rag.Trace` plus caller-supplied gold info into an
    `eval.Example`
- zero new dependencies — `encoding/json`, `bufio`, `io`, `os`,
  `sync` cover everything
- 6 regression tests (`feedback/feedback_test.go`):
  - `TestBuildExampleMapsTrace` — Trace fields land in the right
    Example slots
  - `TestRecorderCaptureWritesJSONL` — two captures produce two
    valid JSON lines
  - `TestRecorderCaptureConcurrent` — 50 goroutines capture once
    each; all 50 lines present, parseable, no truncation, no
    duplication
  - `TestRoundTripObserverToLoadJSONL` — end-to-end loop: wire a
    file-backed Recorder into `Observer.OnAsk` with an opt-in
    filter (`[miss]` substring), run two Asks (one flagged, one
    ignored), close the file, read it back via `eval.LoadJSONL`,
    assert the captured Example round-tripped with the right
    Query / Namespace / gold IDs
  - `TestRecorderCaptureNilWriterFails` — graceful error when
    Recorder has no writer
  - `TestOpenFileFailsOnUnwritablePath` — surfaces filesystem
    errors

## Files

- `/tmp/llm-agent-rag/feedback/feedback.go` (new, ~100 LOC)
- `/tmp/llm-agent-rag/feedback/feedback_test.go` (new, ~205 LOC)

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

- `go build ./...` / `go vet ./...` (standalone): pass
- `go test ./...` (standalone, 16 packages including new
  `feedback`): pass
- 6/6 feedback tests pass
- `go.mod` diff confirms no new module dependencies introduced by
  this slice (only the 12-01 pgvector deps remain)
- core `go vet ./rag/...` + `go test ./rag/...`: pass

## Notes & Design Decisions

- **Opt-in per call, not by default.** `Recorder.Capture` is the
  surface; the user decides which traces are worth recording.
  There's no auto-capture-every-ask helper because that would
  encourage observability-style logging instead of
  regression-case capture.
- **Concurrency model: mutex around buffered write.** Each
  Capture serializes through a sync.Mutex, then writes the
  marshaled bytes + newline through a fresh `bufio.Writer`. The
  buffered writer is created per call so a partial write under
  contention doesn't leak into the next call's output.
- **`rag.Trace` was the natural source.** Question and Namespace
  are already on it; gold info comes from outside (human label,
  dataset lookup, etc.). No new `TraceID` field needed.
- **`OpenFile` returns a closer, not a `Close` method on Recorder.**
  Two reasons: Recorder is `io.Writer`-shaped, so non-file sinks
  (bytes.Buffer, network) compose cleanly; and the closer is just
  `f.Close`, which keeps Recorder a pure value type without
  lifecycle.
- **Test side-effect: the round-trip test proves the format
  contract.** `eval.LoadJSONL` and `feedback.Capture` are now
  pinned to each other — break the format on either side and the
  test fails.

## Next slice

`13-04` is the last numbered Phase 13 slice: cross-repo
contract-drift CI gates between standalone `llm-agent-rag` and
the core `llm-agent/rag` facade (covers RAG-ECO-02). After
13-04, every Phase 13 requirement has shipped and the milestone
work is complete.

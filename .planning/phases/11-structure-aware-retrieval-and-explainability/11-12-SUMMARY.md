> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent-rag` current docs.

# Phase 11-12 Summary

Date: 2026-05-14
Repo: `llm-agent-rag`
Plan: [11-12-PLAN.md](/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/phases/11-structure-aware-retrieval-and-explainability/11-12-PLAN.md)

## Objective

Make per-route attribution visible in retrieval traces. The earlier slices
recorded *why* the route policy fanned out or converged (11-10 rationale,
11-11 gap decision), but per-route hits and matched sections were still
flattened across executed routes via `appendUniqueStrings`. Downstream
consumers had no way to tell which route produced which hit.

## Delivered

- added `retrieve.TrajectoryStep` with `Route`, `Confidence`, `Mode`,
  `HitCount`, `HitIDs`, `MatchedSections`, `ExpandedSections`, `Rationale`
- added `retrieve.Trace.SearchTrajectory []TrajectoryStep`
- `retrieveWithRoutePolicy` now emits trajectory steps:
  - single-path early-out emits exactly one `Mode="single"` step via a new
    `singleTrajectoryStep` helper
  - fanout/converge loop emits exactly one step per executed candidate with
    per-route hit IDs and matched/expanded sections (no cross-route bleed)
- `VariantRetriever.Retrieve` now propagates per-variant trajectories into
  the merged outer trace (the flattened aggregate fields are unchanged)
- plumbed `SearchTrajectory` through the rag facade:
  - `rag.Trace.SearchTrajectory`
  - `rag.Diagnostics.SearchTrajectory`
  - `cloneTrajectory(...)` deep-copies into both
- added regression tests:
  - `retrieve.TestVariantRetrieverTrajectoryReflectsConvergedRoute`
  - `retrieve.TestVariantRetrieverTrajectoryReflectsFanoutPerRoute`
  - `retrieve.TestVariantRetrieverTrajectoryRecordsSinglePathRun`
  - `rag.TestAskExposesPerRouteSearchTrajectory`

## Files

- `/tmp/llm-agent-rag/retrieve/retrieve.go`
- `/tmp/llm-agent-rag/retrieve/retrieve_test.go`
- `/tmp/llm-agent-rag/rag/system.go`
- `/tmp/llm-agent-rag/rag/ask.go`
- `/tmp/llm-agent-rag/rag/system_test.go`

## Verification

```bash
cd /tmp/llm-agent-rag
GOWORK=off GOCACHE=/tmp/go-build go test ./retrieve ./rag -count=1
GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1

cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent
GOWORK=off go vet ./rag/...
GOWORK=off go test ./rag/... -count=1
```

Result:

- `go test ./retrieve ./rag` (standalone): pass
- `go test ./...` (standalone, 11 packages): pass
- `go vet ./rag/...` (core compatibility): pass
- `go test ./rag/...` (core compatibility): pass

## Notes

- trajectory is additive — existing flattened `SearchPath`,
  `MatchedSections`, `ExpandedSections`, `ExpandedChunkIDs`,
  `SelectedChunkIDs` remain the convenience aggregate. Consumers that need
  per-route attribution use `SearchTrajectory`; consumers that need a flat
  view can keep reading the old fields.
- trajectory steps do not yet carry latency / timestamp metadata; that can
  be added when there is a concrete consumer demanding it.
- per-variant trajectories are concatenated, so the same route can appear
  multiple times when multiple query variants pick it. That is the correct
  representation of distinct execution events; deduplication would lose
  information.
- this likely closes out Phase 11. Remaining items in the standing
  pending-todos list (section-planner skeleton, search-trajectory enrichment)
  can either form a follow-up phase or move under the upcoming AI-SPEC
  planner work; either way, Phase 11's stated goal (structure-aware
  retrieval + explainability) is now fully covered.

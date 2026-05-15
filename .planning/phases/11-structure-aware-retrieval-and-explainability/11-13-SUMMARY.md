# Phase 11-13 Summary

Date: 2026-05-14
Repo: `llm-agent-rag`
Plan: [11-13-PLAN.md](/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/phases/11-structure-aware-retrieval-and-explainability/11-13-PLAN.md)

## Objective

Extract the inline gap/fanout decision out of `retrieveWithRoutePolicy`
into a named `SectionPlanner` seam so future planner strategies have a
plug-in point. The default implementation
(`GapAwareSectionPlanner`) preserves current behavior bit-for-bit.

## Delivered

- added `retrieve.SectionPlannerDecision` (`Selected`, `Mode`, `Fanout`,
  `Gap`, `Rationale`)
- added `retrieve.SectionPlanner` interface
  (`Plan(ctx, Request, candidates) (SectionPlannerDecision, error)`)
- added `retrieve.GapAwareSectionPlanner` that implements the gap-aware
  policy previously inlined: threshold filter → optional gap collapse →
  fanout cap → mark selected
- added `Planner SectionPlanner` field on `VariantRetriever`; nil
  defaults to `GapAwareSectionPlanner{}`
- `retrieveWithRoutePolicy` now consults the planner between probe and
  execute; the inline filter/gap/fanout logic is gone from the function
  body. The single-path early-out (`AutoRouteFanout <= 1` or explicit
  `RoutePath`) is unchanged.
- `mergedTrace.RoutePolicy` now reads `Mode`/`Fanout`/`Gap`/`Rationale`
  straight from the planner decision. `ConfidenceThreshold` and
  `ConfidenceGap` keep coming from `Request` because they describe input,
  not the decision.
- added regression tests:
  - `retrieve.TestVariantRetrieverDefaultPlannerMatchesNilPlanner` —
    asserts that an explicit `GapAwareSectionPlanner{}` is equivalent to
    leaving `Planner` nil (same mode, gap, hit set)
  - `retrieve.TestVariantRetrieverHonorsCustomPlanner` — a custom
    `forcedSinglePlanner` overrides mode, fanout, and rationale; trace
    reflects `Mode="single"`, one trajectory step, custom rationale line

## Files

- `/tmp/llm-agent-rag/retrieve/retrieve.go`
- `/tmp/llm-agent-rag/retrieve/retrieve_test.go`

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

- `go test ./retrieve ./rag` (standalone): pass — every pre-existing
  test passes without modification, plus the two new seam tests
- `go test ./...` (standalone, 11 packages): pass
- `go vet ./rag/...` (core compatibility): pass
- `go test ./rag/...` (core compatibility): pass

## Notes

- the seam is intentionally scoped to `VariantRetriever`. The rag facade
  does not yet expose a `Planner` knob; rag-level configurability can
  land when there's a concrete consumer beyond the default
  gap-aware policy.
- `filterRouteCandidates` and `markSelectedCandidates` stay as package
  helpers consumed by `GapAwareSectionPlanner`. Other planners can reuse
  them or implement from scratch.
- this closes both standing pending-todo items from STATE.md ("richer
  planner decision policy" and "section planner behavior"). Phase 11's
  stated goal (structure-aware retrieval + explainability) is now
  fully covered with a clean extension seam in place.

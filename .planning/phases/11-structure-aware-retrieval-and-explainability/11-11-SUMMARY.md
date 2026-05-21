> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent-rag` current docs.

# Phase 11-11 Summary

Date: 2026-05-14
Repo: `llm-agent-rag`
Plan: [11-11-PLAN.md](/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/phases/11-structure-aware-retrieval-and-explainability/11-11-PLAN.md)

## Objective

Make route fanout adaptive to candidate evidence rather than driven only by the
static `AutoRouteFanout` cap. The system should converge on a strong top-1
candidate and only fan out when the top two route confidences are close.

## Delivered

- added `Request.AutoRouteConfidenceGap` and `RoutePolicyTrace.{ConfidenceGap, Gap}`
- `retrieveWithRoutePolicy` now consults the top1−top2 confidence gap after
  thresholding:
  - `gap >= AutoRouteConfidenceGap` collapses fanout to a single route and
    marks `Mode="converged"`
  - `gap <  AutoRouteConfidenceGap` keeps the existing fanout-up-to-N path
    with `Mode="fanout"`
- gap rationale lines (`converged: top-1 dominates by gap=...` /
  `fanout: top-2 within gap=...`) added to `RoutePolicyTrace.Rationale`
- the static early-out branch (fanout disabled or explicit route path) now
  also records the configured gap and a `gap policy inactive: fanout disabled`
  line so traces stay self-explanatory
- `AutoRouteConfidenceGap` plumbed through `rag.SearchOptions` →
  `retrieve.Request`; `Ask(...)` diagnostics surface `RoutePolicy.Gap`
- `AutoRouteConfidenceGap == 0` is the explicit opt-out (today's callers
  see identical hits and trace, no new rationale lines)
- added regression tests:
  - `retrieve.TestVariantRetrieverConvergesWhenConfidenceGapDominates`
  - `retrieve.TestVariantRetrieverFansOutWhenConfidenceGapIsSmall`
  - `retrieve.TestVariantRetrieverConfidenceGapZeroIsOptOut`
  - `rag.TestAskConvergesWhenConfidenceGapDominates`

## Files

- `/tmp/llm-agent-rag/retrieve/retrieve.go`
- `/tmp/llm-agent-rag/retrieve/retrieve_test.go`
- `/tmp/llm-agent-rag/rag/options.go`
- `/tmp/llm-agent-rag/rag/retrieve.go`
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
- `go test ./...` (standalone): pass — all 11 packages
- `go vet ./rag/...` (core compatibility): pass
- `go test ./rag/...` (core compatibility): pass

## Notes

- gap policy uses an absolute threshold on normalized confidence; it is not yet
  query- or domain-adaptive
- the converge branch always collapses to exactly the top-1 route — future work
  could allow N>1 even when the gap is dominant if downstream signals justify
  retaining a backup route
- next-step candidates: section-planner behavior (use rationale + gap to drive
  a richer search-trajectory output), or pushing this decision into the
  upcoming AI-SPEC planner so the gap threshold itself becomes a learned
  parameter

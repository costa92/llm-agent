# Phase 11-07 Summary

Date: 2026-05-14
Repo: `llm-agent-rag`
Plan: [11-07-PLAN.md](/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/phases/11-structure-aware-retrieval-and-explainability/11-07-PLAN.md)

## Objective

Upgrade structure-aware routing from a single auto-selected route to a
multi-candidate route proposal that can aggregate evidence across query
variants.

## Delivered

- added `RouteCandidate` with:
  - `Path`
  - `Score`
  - `Queries`
- added `AutoRouteMaxCandidates` so callers can bound how many route proposals
  are retained
- auto-route proposal now emits several ranked route candidates rather than
  only the selected route
- `VariantRetriever` now merges route candidates across query variants by:
  - summing candidate scores
  - unioning contributing query strings
  - preserving ranked order
- `Ask` trace and diagnostics now expose `AutoRouteCandidates`
- added regression tests proving:
  - a single retrieval call exposes ranked route candidates
  - query variants merge route candidates across sections
  - ask trace/diagnostics surface the merged candidate set

## Files

- `/tmp/llm-agent-rag/retrieve/retrieve.go`
- `/tmp/llm-agent-rag/retrieve/retrieve_test.go`
- `/tmp/llm-agent-rag/rag/options.go`
- `/tmp/llm-agent-rag/rag/retrieve.go`
- `/tmp/llm-agent-rag/rag/system.go`
- `/tmp/llm-agent-rag/rag/ask.go`
- `/tmp/llm-agent-rag/rag/system_test.go`

## Verification

```bash
cd /tmp/llm-agent-rag
GOWORK=off GOCACHE=/tmp/go-build go test ./retrieve ./rag -count=1
GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1
```

Result:

- `go test ./retrieve ./rag`: pass
- `go test ./...`: pass

## Notes

- the system still executes using the top-ranked route; this is not yet a full
  planner that fans out across several routes
- the next logical step is route-confidence signaling or multi-route execution
  policy on top of the candidate set

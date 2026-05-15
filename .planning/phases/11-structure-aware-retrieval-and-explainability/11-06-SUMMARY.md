# Phase 11-06 Summary

Date: 2026-05-14
Repo: `llm-agent-rag`
Plan: [11-06-PLAN.md](/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/phases/11-structure-aware-retrieval-and-explainability/11-06-PLAN.md)

## Objective

Add a lightweight automatic route-selection step so structure-aware corpora can
pick a relevant section subtree before running the rest of retrieval.

## Delivered

- added auto-route controls:
  - `EnableAutoRoute`
  - `AutoRouteMinScore`
- added `AutoRoutePath` to retrieval and ask trace output
- implemented automatic route proposal by scoring query-token overlap against
  section paths/headings already preserved in stored chunks
- reused the existing route-path-constrained retrieval path once a route is
  selected, instead of introducing a separate execution branch
- ensured explicit `RoutePath` still wins over automatic route selection
- added regression tests proving:
  - lexical retrieval auto-selects the matching `Travel` subtree
  - ask trace exposes both `RoutePath` and `AutoRoutePath`
  - final hits stay inside the auto-selected subtree
- updated standalone README to mention automatic section route selection

## Files

- `/tmp/llm-agent-rag/retrieve/retrieve.go`
- `/tmp/llm-agent-rag/retrieve/retrieve_test.go`
- `/tmp/llm-agent-rag/rag/options.go`
- `/tmp/llm-agent-rag/rag/retrieve.go`
- `/tmp/llm-agent-rag/rag/system.go`
- `/tmp/llm-agent-rag/rag/ask.go`
- `/tmp/llm-agent-rag/rag/system_test.go`
- `/tmp/llm-agent-rag/README.md`

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

- this is a heuristic route selector, not a full query planner
- the next Phase 11 step can build on this by selecting among multiple
  candidate routes, fusing route proposals across query variants, or exposing
  richer route confidence signals

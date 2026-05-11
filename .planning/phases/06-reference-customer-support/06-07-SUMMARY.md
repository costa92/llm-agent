# Phase 06-07 Summary

Date: 2026-05-11
Repo: `llm-agent-customer-support`
Plan: [06-07-PLAN.md](/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/phases/06-reference-customer-support/06-07-PLAN.md)

## Objective

Ship the promised day-one prompt-injection protections before the final demo
packaging plan.

## Delivered

- Added `internal/guardrails` with a layered defense surface:
  - suspicious-input heuristics
  - safe fallback reply policy
  - untrusted retrieved-content system-prompt prefix
- Extended `supportflow` to:
  - reject flagged prompt-injection input before orchestration
  - return a safe fallback response instead of invoking the model
  - prepend untrusted-RAG handling guidance to the tool-calling system prompt
- Hardened the `refund_policy` tool to reject any LLM-supplied `user_id` field,
  preserving server-side authority over identity.
- Added tests covering:
  - prompt-injection heuristic matches
  - normal support inputs still passing
  - safe fallback behavior on flagged input
  - blocked forged `user_id` tool abuse
  - untrusted-RAG marking appearing in the generated system-prompt path

## Files

- `/tmp/llm-agent-customer-support/internal/guardrails/guardrails.go`
- `/tmp/llm-agent-customer-support/internal/guardrails/guardrails_test.go`
- `/tmp/llm-agent-customer-support/internal/supportflow/supportflow.go`
- `/tmp/llm-agent-customer-support/internal/supportflow/supportflow_test.go`
- `/tmp/llm-agent-customer-support/README.md`

## Verification

Executed against the local 4-repo workspace:

```bash
GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go test ./internal/guardrails ./internal/supportflow -count=1
GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go test ./... -count=1
GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go build ./...
```

Result:

- `go test ./internal/guardrails ./internal/supportflow -count=1`: pass
- `go test ./... -count=1`: pass
- `go build ./...`: pass

## Notes

- The current heuristic filter intentionally targets obvious day-one injection
  patterns and favors deterministic explainability over broader ML-style
  classification.
- Identity remains server-owned: any future user-aware tools must derive
  identity from request context rather than trusting model-supplied arguments.

# Phase 06-04 Summary

Date: 2026-05-11
Repo: `llm-agent-customer-support`
Plan: [06-04-PLAN.md](/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/phases/06-reference-customer-support/06-04-PLAN.md)

## Objective

Replace the temporary simple-agent bootstrap with the first real customer-support
flow built from explicit triage, RAG lookup, and native tools.

## Delivered

- Added `internal/supportflow` with a typed `StateGraph` triage flow.
- Routed `chargeback` and `fraud` inputs to human escalation.
- Routed refund requests without an order ID to a clarification response.
- Added a tool-enabled self-service branch using `FunctionCallAgent`.
- Added a `refund_policy` tool backed by `rag.RAGSystem` search.
- Wired `internal/app.New(...)` to:
  - adapt the selected embedding provider into the core RAG interface
  - seed a small in-memory knowledge base
  - build `supportflow` instead of `SimpleAgent`
- Updated app-level tests to assert:
  - refund lookup returns seeded policy evidence
  - missing order IDs request more information
  - chargebacks escalate to a human path
  - transport routes now serve the real support flow
- Updated the README to describe the live support-flow behavior.

## Files

- `/tmp/llm-agent-customer-support/internal/supportflow/doc.go`
- `/tmp/llm-agent-customer-support/internal/supportflow/supportflow.go`
- `/tmp/llm-agent-customer-support/internal/supportflow/supportflow_test.go`
- `/tmp/llm-agent-customer-support/internal/app/app.go`
- `/tmp/llm-agent-customer-support/internal/app/app_test.go`
- `/tmp/llm-agent-customer-support/README.md`

## Verification

Executed against the local 4-repo workspace:

```bash
GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go test ./internal/supportflow ./internal/httpapi -count=1
GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go test ./... -count=1
GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go build ./...
```

Result:

- `go test ./internal/supportflow ./internal/httpapi -count=1`: pass
- `go test ./... -count=1`: pass
- `go build ./...`: pass

## Notes

- The service now consumes the embedding-provider seam added in `06-03`; the
  embedder is no longer bootstrap-only metadata.
- The current knowledge base is intentionally seeded in-memory for Phase 6
  velocity. Persistent storage and guardrails remain later plans.

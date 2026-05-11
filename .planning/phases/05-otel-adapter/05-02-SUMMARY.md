# Phase 05-02 Summary

Date: 2026-05-11
Repo: `llm-agent-otel`
Plan: [05-02-PLAN.md](/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/phases/05-otel-adapter/05-02-PLAN.md)

## Objective

Build `otelagent.Wrap(agents.Agent)` so high-level agent runs emit an
`invoke_agent`-rooted trace tree without changing the public `Agent` contract.

## Delivered

- Added `otelagent.Wrap(agent, opts...) agents.Agent`.
- Preserved the full `agents.Agent` contract: `Name()`, `Run(...)`,
  `RunStream(...)`.
- Instrumented agent runs with a root `invoke_agent <name>` span.
- Mapped streamed agent steps into bounded child spans:
  - `chat` for LLM-thought/final phases
  - `execute_tool <tool>` for action/observation phases
- Kept child spans parented directly under the `invoke_agent` root to preserve a
  stable tree shape.
- Added tests covering:
  - contract preservation on wrapped `SimpleAgent`
  - `SimpleAgent` trace shape: `invoke_agent -> chat`
  - `ReActAgent` scratchpad trace shape:
    `invoke_agent -> chat -> execute_tool -> chat`

## Files

- `/tmp/llm-agent-otel/otelagent/config.go`
- `/tmp/llm-agent-otel/otelagent/otelagent.go`
- `/tmp/llm-agent-otel/otelagent/otelagent_test.go`

## Verification

Executed against a temporary local `go.work` binding `llm-agent-otel` to the
current core repo checkout:

```bash
GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go test ./otelagent/... -count=1
GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go build ./otelagent/...
```

Result:

- `go test`: pass
- `go build`: pass

## Notes

- `otelagent.Wrap(...)` only relies on the public `agents.Agent` interface and
  therefore intentionally does not assume access to inner `llm.ChatModel`
  metadata such as model names.
- Child span naming is kept generic (`chat`, `execute_tool <tool>`) so the
  wrapper remains valid across all current and future agent implementations.

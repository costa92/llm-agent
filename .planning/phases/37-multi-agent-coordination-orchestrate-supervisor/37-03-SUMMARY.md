---
phase: 37-multi-agent-coordination-orchestrate-supervisor
plan: 03
status: complete
completed_at: 2026-05-21
repo: llm-agent (core)
requirements: [CC-3]
files_modified:
  - orchestrate/supervisor_compose_test.go
  - examples/08-supervisor/main.go
  - examples/08-supervisor/main_test.go
  - examples/08-supervisor/README.md
metrics:
  tests: "go test ./orchestrate -count=1; go test ./08-supervisor -count=1"
---

# 37-03 — Composition tests and example

Added runtime coverage for `Supervisor` inside `StateGraph`, `StateGraph`
inside `Supervisor`, and `Supervisor`-of-`Supervisor` composition. Added a
deterministic `examples/08-supervisor` demo with smoke test and README.

## Outcome

- Example runs deterministically and prints `OK`.
- Compose-direction tests pass.
- README documents MaxRounds vs Budget.MaxCalls and the canonical policy stack note.

## Deviations

None.


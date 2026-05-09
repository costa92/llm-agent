# Project guide for AI assistants

This project uses **GSD (Get Shit Done)** for milestone planning. The `.planning/` directory is the source of truth for what's being built and why.

## Read first (in this order)

1. `.planning/PROJECT.md` — what this project IS, core value, requirements, constraints, key decisions
2. `.planning/STATE.md` — where we are right now (current phase, plan, recent activity)
3. `.planning/ROADMAP.md` — the phase plan (8 phases, multi-repo umbrella)
4. `.planning/REQUIREMENTS.md` — 65 v1 requirements + traceability to phases
5. `.planning/research/SUMMARY.md` — the cross-cut from research; K1–K7 keystone decisions live here
6. `.planning/config.json` — workflow toggles (YOLO mode, standard granularity, parallelization on, all gates on)

When the human gives an instruction, check those files before exploring the codebase. Most "what should I do?" questions are already answered there.

## Project at a glance

- **Repo:** `github.com/costa92/llm-agent` — a stdlib-only Go LLM agents framework (v0.2.0 → v0.3.0).
- **Milestone (v0.3):** add real provider adapters (OpenAI/Anthropic/Ollama), OpenTelemetry observability, and a deployable customer-support reference service — all in **sister repos** so the core stays stdlib-only.
- **4-repo umbrella:** `llm-agent` (core, this repo), `llm-agent-providers`, `llm-agent-otel`, `llm-agent-customer-support`. The latter three don't exist yet — Phase 0 creates them.
- **Pace:** solo, side-project, no deadline. **Quality > speed.**

## Hard rules

1. **Core repo (`llm-agent`) stays stdlib-only.** No `go.sum`, no non-stdlib deps in `go.mod`. Ever. If a feature needs a dep, it goes in a sister repo.
2. **No K8s in v0.3.** Helm/K8s manifests are out of scope per `PITFALLS.md` Pitfall 16. Don't add them; flag any request to add them as scope creep against the milestone.
3. **No `replace` directives in tagged-release branches.** `replace` is a local-dev escape hatch only. CI gate enforces this (INFRA-04).
4. **`go.work` is `.gitignore`d in every repo.** CI runs with `GOWORK=off`.
5. **Capabilities are per-(provider × model), not per-provider.** A provider instance binds a model at construction time (`openai.New(openai.WithModel("gpt-4o"))`); `Info()` reflects THAT model's capabilities. (Keystone K2.)
6. **Streaming events are a typed union, not lowest-common-denominator chunks.** `StreamEvent.Kind` enum + stable per-tool-call `Index` field. (Keystone K1.)
7. **OTel attaches as decorator wrappers, never hooks.** `otelmodel.Wrap(inner) ChatModel`. (Keystone K3.)
8. **Refsvc has hard caps + `DISABLE_LLM=1` panic switch from Day 1.** Not a follow-up. (Keystone K7.)

## GSD slash commands you can invoke

These are user-facing slash commands that drive the planning lifecycle. The user runs them; you assist.

- `/gsd-plan-phase <N>` — create the detailed plan for phase N
- `/gsd-execute-phase <N>` — execute all plans in phase N (wave-based parallelization)
- `/gsd-discuss-phase <N>` — gather context before planning a phase
- `/gsd-progress` — situational check; what's next?
- `/gsd-transition` — move from one phase to the next (updates PROJECT.md, REQUIREMENTS, STATE)
- `/gsd-debug` — systematic debugging with persistent state
- `/gsd-code-review` — review source files changed during a phase

## When the user asks for code

- Trust the existing patterns in the repo. The 5 agent paradigms (Simple/ReAct/Reflection/PlanSolve/FunctionCall) and the package layout in `README.md` are validated; don't refactor unless the current phase explicitly says to.
- Before changing public types in `llm/`, `agents/`, or `orchestrate/`, check `.planning/PROJECT.md` "Validated" — those capabilities are locked.
- The `ScriptedLLM` (in `scriptedllm_test.go`) is the canonical mock. Examples in `/examples/` use it — keep new examples deterministic too.
- Tests run via `go vet ./... && go test ./...`. There is no `go.sum` by design.

## When the user asks "what's next?"

The fastest answer is `cat .planning/STATE.md` — it names the current phase + plan and the last commit. If they want to start the next concrete step, it's:

```
/gsd-plan-phase 0     # currently — Multi-repo infra + llm/v2 keystone interfaces
```

Phase 0 must complete before Phase 1 (the walking-skeleton starts).

## Files you should NOT touch without explicit ask

- `LICENSE`, `OWNERS`, `CHANGELOG.md` (only updated at version bumps)
- `.github/workflows/test.yml` (Phase 0 will refactor it; until then leave alone)
- `go.mod` (any change here must justify staying within stdlib-only)

## When in doubt

Ask. The planning artifacts are detailed (PROJECT.md alone is ~150 lines, ROADMAP.md is 8 phases × ~30 lines each, research bundle is ~13k LOC). Most "should I X?" questions are answered there — checking before asking is faster than the round-trip.

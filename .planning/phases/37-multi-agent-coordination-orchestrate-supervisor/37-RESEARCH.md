# Phase 37 Research: Multi-agent coordination (`orchestrate.Supervisor`)

**Researched:** 2026-05-21
**Phase:** 37 — multi-agent coordination (third v1.2 phase; follows Phase 36)
**Requirement:** CC-3
**Repo touched:** `llm-agent` (core only)
**Target tag:** `v0.6.2` (patch — strict-additive new `orchestrate.Supervisor` type)
**Upstream:**
- `.planning/research/v1.2-core-capability-deepening-SUMMARY.md` —
  KC-1 (Supervisor lives in `orchestrate/` as a thin `StateGraph[S]`
  facade; workers are `agents.Agent`; iterative supervisor↔worker loop) +
  KC-5 (additive only, no `/v2`, no edit to validated public types).
- `.planning/phases/35-budget-and-cancellation-context/35-RESEARCH.md` —
  §"Carry-forward notes" pins: workers receive `supervisorCtx`
  unchanged, so the parent `budget.Tracker` is found by each worker's
  `generateFromPrompt`; rounds count against `Budget.Calls`. **Hard
  rule:** no `context.WithoutCancel` or detached child contexts.
- `.planning/phases/36-policy-safety-middleware/36-RESEARCH.md` —
  composition stack `policy.Wrap(otelmodel.Wrap(provider))` is the
  v1.2 norm; budget enforces UNDERNEATH the wrappers at the
  `generateFromPrompt` chokepoint. Phase 37 inherits both layers
  unchanged.

## Scope (CC-3 verbatim)

> An `orchestrate.Supervisor` primitive is shipped as a thin facade over
> `StateGraph[S]` (per KC-1) with `NewSupervisor`, `SupervisorOptions{
> Planner, Workers, MaxRounds, ParseDispatch, BuildAggregate }`, where
> workers are `agents.Agent` (so a Supervisor can supervise another
> Supervisor — composition). The Supervisor honors **CC-1**'s ctx-keyed
> budget (rounds count against `Budget.Calls`; ctx propagates to
> workers) and supports policy attachment via **CC-2** (documented
> pattern for policy-wrapping a worker's underlying model). A
> `compose-with-StateGraph` test proves the facade works both
> directions (Supervisor inside StateGraph, StateGraph inside
> Supervisor). **The core stays stdlib-only**; no edit to `agents.Agent`
> or `orchestrate.NodeFunc[S]` (KC-5). Phase 37.

One sentence: a stdlib-only `orchestrate.Supervisor` value — implemented
as a typed `StateGraph[S]` whose state machine is *planner-emits-Dispatch
→ worker-runs → planner-observes → repeat-or-finish* — that satisfies
`agents.Agent` (so it can be a worker of another Supervisor) and
inherits cancellation + MaxRounds guard-rails from the underlying
`StateGraph[S]`.

## User Constraints

No CONTEXT.md exists for Phase 37 — this research operates directly on
KC-1 (the keystone is pre-decided at the milestone level). All hard
rules below come from CLAUDE.md (project) + KC-5 (additive-only
ceiling) + KC-1 (facade over `StateGraph[S]`).

### Locked Decisions (from KC-1 + KC-5 + CLAUDE.md — DO NOT re-litigate)

- **Live in `orchestrate/`, not `agents/coord`.** A new type inside
  the existing `orchestrate/` package, not a new sub-package. (KC-1.)
- **Thin facade over `StateGraph[S]`.** The Supervisor's loop body is a
  typed state machine: planner-emits-Dispatch → worker-runs →
  planner-observes → repeat-or-finish. Implementation MUST reuse
  `NewStateGraph[S]() + AddNode + AddConditionalEdge + Compile + Run`.
  No parallel state machine. (KC-1.)
- **Workers are `agents.Agent`.** Map keyed by worker name. (KC-1.)
- **Supervisor satisfies `agents.Agent`.** So a Supervisor may be a
  worker for another Supervisor (composition). (KC-1.)
- **`SupervisorOptions` field set, fixed by KC-1.** `Planner`,
  `Workers`, `MaxRounds`, `ParseDispatch`, `BuildAggregate`. (KC-1.)
- **Honors CC-1 (budget) via ctx propagation.** Rounds count against
  `Budget.Calls` (each planner LLM call + each worker LLM call already
  charges at the `generateFromPrompt` chokepoint shipped in Phase 35).
  Workers receive `supervisorCtx` unchanged — **NO** `context.WithoutCancel`,
  **NO** detached children, **NO** new `WithBudget` derivation inside
  Supervisor. (35-RESEARCH.md carry-forward; CC-3 verbatim.)
- **Supports CC-2 (policy) by convention.** Each worker's underlying
  `llm.ChatModel` MAY be `policy.Wrap`-ed by the caller before
  constructing the worker. Supervisor adds no policy layer itself — it
  is policy-agnostic (the chokepoint at `generateFromPrompt` is where
  budget + policy fire). (KC-3 — model boundary is the gate site.)
- **Compose with `StateGraph[S]` both directions.** A test must prove
  Supervisor inside `StateGraph[S]` (as a NodeFunc that runs Supervisor)
  AND `StateGraph[S]` inside Supervisor (as a worker — wrapping the
  CompiledGraph in a small `agents.Agent` adapter). (CC-3 verbatim.)
- **Stdlib-only.** `context`, `errors`, `fmt`, `strings`, `sync` —
  nothing else. (CLAUDE.md Rule 1; KC-5.)
- **No edit to `llm.ChatModel`, `agents.Agent`, `memory.Memory`,
  `orchestrate.NodeFunc[S]`.** New type + new exported helpers only.
  Tag is `v0.6.2` — strict patch / additive. (KC-5.)
- **Target tag: `v0.6.2`** (patch — additive). The `v0.7.0`
  milestone-cap tag is Phase 38.

### Claude's Discretion (this research recommends)

- The exact typed state shape `supervisorState` (recommended below in
  Decision A).
- The exact `Dispatch` struct shape — worker name + sub-input + opaque
  metadata (Decision B).
- The exact `WorkerResult` shape — links Dispatch + `agents.Result`
  (Decision C).
- Whether `Planner` is `agents.Agent` or a stricter `Planner` interface
  (recommendation: `agents.Agent` — uniformity; ParseDispatch handles
  the text→structured conversion; Decision D).
- Whether Supervisor charges its own per-round budget on top of
  per-call budget (recommendation: NO; inherit chokepoint-level
  enforcement; Decision E).
- Whether `BuildAggregate` is called every round or only on finish
  (recommendation: only on finish — at the FINAL node; Decision F).
- Whether Supervisor emits `StepEvent`s via `RunStream` (recommendation:
  YES, via `runStreamFromBlocking` — same pattern as the 5 paradigms;
  Decision G).
- The exact example shape (recommendation: 1 planner + 2 specialist
  workers + a deterministic 2-round script; Decision H).
- Slice breakdown (4 slices recommended — matches the ROADMAP §Planned
  work outline; Decision I).

### Deferred Ideas (OUT OF SCOPE for Phase 37)

- **A native LLM-structured-output `Planner` interface.** Most v1.2
  providers don't have first-class structured outputs; KC-1 says the
  planner emits TEXT and `ParseDispatch` converts. v1.3 can add a
  `StructuredPlanner` additive interface.
- **Parallel worker dispatch (planner emits N dispatches per round).**
  v1.2 ships **one dispatch per round** for design clarity. If the
  user wants parallel work, they use `FanOutFanIn` (already exists)
  or wrap a `FanOutFanIn` as a single worker. Multi-dispatch is a v1.3
  candidate.
- **Streaming workers (worker.RunStream).** Supervisor calls
  `worker.Run(ctx, ...)` (the blocking variant). `RunStream` on the
  Supervisor itself emits its own coordination `StepEvent`s; individual
  worker streaming is a future enhancement (out of K1 scope today
  because no paradigm streams the underlying model anyway, per
  35-RESEARCH.md §Chokepoint discovery).
- **Memory scoping per worker.** KC-2 deferred memory tiering to v1.3;
  Supervisor inherits whatever memory shape the workers carry.
- **Refund-on-policy-block budget semantics.** 36-RESEARCH.md §Decision
  D documented as carry-forward; Supervisor does not introduce new
  refund logic.
- **Cross-Supervisor distributed coordination (a3a sister-repo).** Out
  of v1.2 scope (`comm/` exists in core but is unused by Supervisor;
  Supervisor is single-process).
- **OTel emission of supervisor rounds.** `otelmodel` is a sister repo;
  v1.2 is core-only. v1.3 ecosystem alignment may add round-aware
  span emission.

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CC-3 | `orchestrate.Supervisor` primitive shipped as `StateGraph[S]` facade with `NewSupervisor`, `SupervisorOptions{Planner, Workers, MaxRounds, ParseDispatch, BuildAggregate}`; workers are `agents.Agent`; Supervisor satisfies `agents.Agent`; honors CC-1 budget (rounds count against `Budget.Calls`, ctx propagates); supports CC-2 policy attachment per worker; compose-with-StateGraph test proves the facade direction. Stdlib-only. KC-5: no edits to `agents.Agent` or `orchestrate.NodeFunc[S]`. | This document end-to-end. See §"Standard Stack" for the locked surface; §"Architecture Patterns" for the state-machine graph; §"Decision A-I" for design details; §"Slice Breakdown" for the 4-slice plan. |

## Constraint inventory

- **Stdlib-only core (CLAUDE.md Rule 1, KC-5).** `go.mod`
  (`/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/go.mod`)
  has exactly one `require` — `github.com/costa92/llm-agent-rag v1.0.1`
  for the RAG facade — and Phase 37 adds zero new requires.
  [VERIFIED: read go.mod, exactly one require directive]
- **No `/v2` import path (KC-5).** v0.6.1 → v0.6.2 is a **patch**
  (additive) bump. Existing v0.6.1 callers must compile unchanged
  against v0.6.2. New type + new exported helpers in an EXISTING
  package (`orchestrate/`) are additive (Go allows growing a
  package's surface in any release that preserves existing types).
- **Mirror `StateGraph[S]` mechanics.** `orchestrate/graph.go:14-208`
  ships the canonical builder + Compile + Run pattern. Supervisor MUST
  reuse: `NewStateGraph[S]()`, `AddNode`, `AddEdge`, `AddConditionalEdge`,
  `SetEntry`, `Compile`, `Run(ctx, initial, ...RunOption)`. Guard-rails
  inherited: `select { case <-ctx.Done(): return ctx.Err() }` per step
  (`graph.go:170-174`), `defaultMaxSteps = 100` (`graph.go:148`),
  `ErrGraphMaxSteps` on overrun (`graph.go:208`).
  [VERIFIED: read orchestrate/graph.go in this session]
- **`agents.Agent` interface — locked at `Name() / Run(ctx, input) /
  RunStream(ctx, input)`.** The Supervisor's public methods must match
  these three exactly; no extra exported method on the Supervisor type
  beyond `NewSupervisor`. [VERIFIED: read agent.go:13-21]
- **Compose with existing chokepoint.** `agent_chatmodel.go:11-54`
  ships `generateFromPrompt` — every planner / worker LLM call (which
  goes through one of the 5 paradigms) charges `budget.From(ctx)`
  pre/post automatically. Supervisor adds zero budget logic; the
  charge happens inside each worker. [VERIFIED: read
  agent_chatmodel.go]
- **Compose with FanOutFanIn vocabulary.** `orchestrate/fanout.go`
  ships `PlannedTask{Name, Input, Worker}`, `TaskResult{Task,Result}`,
  `PlanParser`, `AggregateInputBuilder`, `FanOutFanInOptions`. The
  Supervisor's vocabulary (`Dispatch`, `WorkerResult`, `ParseDispatch`,
  `BuildAggregate`) SHOULD be analogous-but-distinct — same intent,
  different shape (one-dispatch-per-round vs. N-tasks-once). DO NOT
  reuse `PlannedTask` directly — its `Worker` field is "optional with
  round-robin fallback", while Supervisor requires explicit worker
  routing. [VERIFIED: read orchestrate/fanout.go]
- **Validated public types unchanged.** `llm.ChatModel`,
  `llm.StreamReader`, `llm.StreamEvent`, `agents.Agent`,
  `agents.Result`, `orchestrate.NodeFunc[S]`, `orchestrate.StateGraph[S]`,
  `orchestrate.CompiledGraph[S]` — none edited. (KC-5.)
- **No `replace` directives in tagged-release branches** (CLAUDE.md
  Rule 3). The v0.6.2 tag must have no `replace`; CI dep-currency
  gate enforces.
- **`go.work` is `.gitignore`d** (CLAUDE.md Rule 4). All `go test`
  commands in exit-gate slice use `GOWORK=off`.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Round loop control flow | `StateGraph[S]` (existing) | Supervisor (facade) | KC-1 — Supervisor reuses graph's `select { ctx.Done } + defaultMaxSteps` guard-rails. No new loop. |
| Planner LLM call | Planner agent (e.g., SimpleAgent / ReActAgent) | `generateFromPrompt` chokepoint | Each planner is just an `agents.Agent`; the chokepoint charges budget automatically (Phase 35 wired) |
| Worker LLM call | Worker agent (e.g., SimpleAgent / ReActAgent / another Supervisor) | `generateFromPrompt` chokepoint | Same — every worker uses the existing 5-paradigm or another Supervisor, all funnel through the chokepoint |
| Dispatch parsing (text → typed struct) | `ParseDispatch func(string) (*Dispatch, error)` | Caller supplies | LLM providers without native structured outputs need a text-to-struct adapter |
| Finish detection | Planner text contains `<FINAL>` marker OR `ParseDispatch` returns `(nil, nil)` to signal finish | `ParseDispatch` is the seam | Single seam — symmetrical with `ParseDispatch`-emits-dispatch / `ParseDispatch`-emits-nil-to-finish |
| Final answer assembly | `BuildAggregate func([]WorkerResult) (string, error)` | Caller supplies | Symmetrical with FanOutFanIn's `AggregateInputBuilder`; emits the user-facing `Result.Answer` |
| Budget enforcement | `generateFromPrompt` chokepoint (Phase 35) | Supervisor passes ctx unchanged | KC-4 single chokepoint; Supervisor adds zero budget logic — ctx propagation is the only mechanism |
| Policy enforcement | Caller wraps worker's model with `policy.Wrap` BEFORE constructing the worker | Supervisor is policy-agnostic | KC-3 model boundary; documented in package doc but no code in Supervisor |
| Cancellation | `ctx.Done()` check inside `CompiledGraph[S].Run` per step (existing) | Workers honor ctx via their `agent.Run(ctx, ...)` | Inherited from StateGraph; no new mechanism |
| Total usage rollup | Supervisor's terminal node sums `WorkerResult.Result.Usage + plannerResult.Usage` per round into final `agents.Result.Usage` | Caller observes via `Result.Usage` | Symmetrical with FanOutFanIn's `TotalUsage` field; Supervisor must aggregate so callers see total spend |
| Trace emission | Supervisor builds `[]Step` from `(StepThought "round N: planner says ...") + (StepAction Tool=worker.Name(), Args=dispatch.Input) + (StepObservation Result=workerResult.Answer)` per round; `StepFinal` at end | Caller observes via `Result.Trace` | Mirrors ReAct's trace shape; no new `StepKind` (all 6 existing kinds suffice per agent.go:69-76) |
| Streaming support | `RunStream` via `runStreamFromBlocking` shared helper | Same as 5 paradigms | Decision G — uniformity |

## Standard Stack

### Core (new symbols inside the existing `orchestrate/` package)

| Symbol | Purpose | Why Standard |
|--------|---------|--------------|
| `orchestrate.Supervisor` (struct) | The supervisor↔worker coordinator | KC-1 explicit; satisfies `agents.Agent` so it composes |
| `NewSupervisor(name string, opts SupervisorOptions) *Supervisor` | Constructor | Mirrors `NewFanOutFanIn(name, opts)` from `fanout.go:67` |
| `(*Supervisor).Name() string` | `agents.Agent` method | Required interface method |
| `(*Supervisor).Run(ctx, input string) (agents.Result, error)` | `agents.Agent` method — the main entry | Required interface method; KC-1 — implemented as `StateGraph[S].Compile().Run()` under the hood |
| `(*Supervisor).RunStream(ctx, input string) (<-chan agents.StepEvent, error)` | `agents.Agent` method | Required interface method; uses `agents.RunStreamFromBlocking`-shaped pattern (Decision G — see §"Decision G") |
| `SupervisorOptions` (struct) | The constructor option payload | KC-1 names fields verbatim: `Planner, Workers, MaxRounds, ParseDispatch, BuildAggregate` |
| `Dispatch` (struct) | One unit of delegation: `WorkerName + Input + Metadata` | New — symmetric to `FanOutFanIn.PlannedTask` but distinct (one dispatch per round, not many at once); Decision B |
| `WorkerResult` (struct) | One round's worker output: `Dispatch + agents.Result` | New — symmetric to `FanOutFanIn.TaskResult`; Decision C |
| `DispatchParser` (function type — `func(string) (*Dispatch, error)`) | Text → structured dispatch from planner's free-text output | LLM providers without native structured output; symmetric to `FanOutFanIn.PlanParser` |
| `Aggregator` (function type — `func([]WorkerResult) (string, error)`) | Builds final answer from all worker results | Symmetric to `FanOutFanIn.AggregateInputBuilder` |
| Sentinels: `ErrSupervisorNilPlanner`, `ErrSupervisorNoWorkers`, `ErrSupervisorUnknownWorker`, `ErrSupervisorMaxRounds`, `ErrSupervisorParseDispatch` | Distinguishable error shapes | Mirrors `FanOutFanIn`'s sentinel set (`fanout.go:253-260`) |

### Standard library imports allowed in the new Supervisor surface

| Stdlib package | Used for |
|----------------|----------|
| `context` | `Run(ctx, ...)` signature; `ctx.Done()` check is inherited from the underlying CompiledGraph |
| `errors` | `errors.New(...)` for sentinels, `fmt.Errorf("%w: ...", base)` wrapping |
| `fmt` | Error formatting, trace string assembly |
| `strings` | Building planner prompts / parsing markers (`<FINAL>` etc.) — though most parsing is delegated to `ParseDispatch` |
| `sync` | NOT NEEDED for Supervisor itself — `CompiledGraph` is sequential, no concurrent state. Only if the example or test wants to spawn goroutines. |

**Verification:**
- All listed imports are stdlib. [VERIFIED: stdlib]
- No third-party "agent coordination" or "supervisor pattern" library
  considered — KC-1 mandates pure `StateGraph[S]` reuse; no new deps.

### Alternatives Considered (and rejected)

| Instead of | Could Use | Rejected because |
|------------|-----------|------------------|
| `StateGraph[S]` facade | Hand-rolled `for round := 0; round < MaxRounds; round++` loop | KC-1 explicit — facade preserves abstraction and inherits guard-rails |
| Single `agents.Agent` Planner | Stricter `Planner` interface returning `(*Dispatch, error)` | KC-1 says workers are `agents.Agent`; symmetry — planner is also `agents.Agent`. `ParseDispatch` handles structured-output gap |
| Multiple dispatches per round (planner emits `[]Dispatch`) | One dispatch per round | Out of scope for v1.2 (see Deferred Ideas); FanOutFanIn already covers parallel; v1.3 candidate |
| Map worker keyed by name (`map[string]agents.Agent`) | `[]NamedAgent` slice | Map gives O(1) lookup by name AND signals "name is the dispatch key"; FanOutFanIn already uses map (`fanout.go:38`) |
| `BuildAggregate` per round (running summary) | `BuildAggregate` once at end | Single seam is simpler; per-round summary is a planner-side responsibility (the planner sees the prior `[]WorkerResult` and decides what to emit next) |
| New `StepKind` for "dispatch" / "round" | Reuse existing 6 kinds | KC-5 — no edit to `Step.Kind` enum; existing `StepAction` + `StepObservation` + `StepThought` express the semantics correctly |
| Supervisor as a new sub-package (`orchestrate/coord`) | Type inside existing `orchestrate/` | KC-1 explicit; one less import path; one less doc entry |

### Package Legitimacy Audit

Not applicable — Phase 37 introduces **zero new external dependencies**.
The Supervisor uses only stdlib (`context`, `errors`, `fmt`, `strings`).
No `npm` / `pip` / `crates` / `go.sum` entries are added. `go.mod`
remains unchanged. The compose-with-StateGraph test uses only existing
`orchestrate/` symbols. The example uses `ScriptedLLM` (per CLAUDE.md).

## Architecture Patterns

### System Architecture Diagram

```
                                  caller invokes
                                 Supervisor.Run(ctx, input)
                                          ↓
                            ╔═════════════════════════════════════╗
                            ║ Supervisor (orchestrate package)    ║
                            ║                                      ║
                            ║   builds CompiledGraph[S] once       ║
                            ║   (lazy or eager — Decision A)       ║
                            ║   then graph.Run(ctx, initialState)  ║
                            ╚═════════════════════════════════════╝
                                          ↓
                            ╔═════════════════════════════════════╗
                            ║ StateGraph[supervisorState] (existing)║
                            ║                                      ║
                            ║   per step: select { ctx.Done }      ║
                            ║   per step: nodeFn(ctx, state)       ║
                            ║   per step: conditional edge → next  ║
                            ║   inherited MaxSteps cap (= MaxRounds×3 + slack)║
                            ╚═════════════════════════════════════╝
                                          ↓
   ┌──────────────────────────────────────────────────────────────────┐
   │  Nodes (NodeFunc[supervisorState]):                              │
   │                                                                  │
   │   [PLAN] planner.Run(ctx, planPrompt)                            │
   │     ↓ ParseDispatch(plan.Answer) → *Dispatch or nil-to-finish    │
   │     conditional edge:                                            │
   │       Dispatch==nil → [FINAL]                                    │
   │       round >= MaxRounds → [FINAL]   (ErrSupervisorMaxRounds)    │
   │       else → [DISPATCH]                                          │
   │                                                                  │
   │   [DISPATCH] workers[d.WorkerName].Run(ctx, d.Input)             │
   │     append WorkerResult to state.Results                         │
   │     unconditional edge → [PLAN]                                  │
   │                                                                  │
   │   [FINAL] BuildAggregate(state.Results) → final answer           │
   │     edge → NodeEnd                                               │
   └──────────────────────────────────────────────────────────────────┘
                                          ↓
                            ╔═════════════════════════════════════╗
                            ║ each agent.Run(ctx, ...) call →     ║
                            ║   generateFromPrompt(ctx, model, ...) ║
                            ║     pre-charge budget.Calls           ║
                            ║     model.Generate(ctx, req)          ║   ← policy.Wrap fires here
                            ║     post-charge budget.Tokens         ║
                            ╚═════════════════════════════════════╝

Composition properties (KC-1 + KC-5):
   - Supervisor satisfies agents.Agent → can be a worker of another Supervisor
   - Supervisor uses StateGraph[S] internally → can be a NodeFunc inside another StateGraph
   - Budget propagates by ctx through every layer (Phase 35 chokepoint)
   - Policy is opt-in per worker by the caller wrapping the worker's model (Phase 36 decorator)
```

### Recommended Project Structure

```
llm-agent/                     # repo root (flat layout — package agents at root)
├── orchestrate/
│   ├── doc.go                 # EDIT: add Supervisor entry to the paradigm list (KC-5: docstring edit is additive)
│   ├── supervisor.go          # NEW — Supervisor, SupervisorOptions, Dispatch, WorkerResult, etc.
│   ├── supervisor_test.go     # NEW — happy-path, MaxRounds, parser error, unknown worker, ctx cancel, composition
│   ├── supervisor_compose_test.go  # NEW — Supervisor inside StateGraph + StateGraph inside Supervisor
│   ├── (existing files unchanged — fanout.go / graph.go / pipeline.go / roleplay.go / roundrobin.go / termination.go / etc.)
└── examples/
    └── 08-supervisor/         # NEW — deterministic example via ScriptedLLM
        ├── main.go
        ├── main_test.go       # smoke test that example exits 0 (mirror 06-budget/main_test.go pattern)
        └── README.md          # ≤80 lines, canonical setup + composition stack note
```

**Rationale for single-file Supervisor (vs split):** the type set is
small enough (~6 types, ~5 sentinels, ~3 node funcs, ~1 prompt template)
to fit in one ~400-500 LOC file analogous to `fanout.go` (260 LOC) +
`graph.go` (208 LOC) sized contributions. Splitting would fragment the
state-machine narrative. **Use `supervisor_compose_test.go` as the
separate compose-direction file** because the assertions there are
distinct enough to warrant their own file.

### Pattern 1: Supervisor as `StateGraph[supervisorState]` facade

**What:** `NewSupervisor` constructs a `Supervisor` value. `Run(ctx,
input)` builds (lazily or eagerly) a `CompiledGraph[supervisorState]`
and calls its `Run` method, then translates the final
`supervisorState` to an `agents.Result`.

**When to use:** Always — this is the whole point of the type.

**Example (skeleton):**

```go
// Source: NEW — facade on top of orchestrate.StateGraph[S] per KC-1
package orchestrate

// supervisorState is the typed S of StateGraph[supervisorState]. The
// state machine carries the loop's full progress so each node is a
// pure function of state.
type supervisorState struct {
    // input is the user-facing prompt (constant across the run).
    input string
    // round is the current round counter (1-indexed by convention).
    round int
    // lastPlannerAnswer is the most recent planner Result.Answer
    // (used by ParseDispatch).
    lastPlannerAnswer string
    // dispatch is the most recent parsed Dispatch (nil = finish).
    dispatch *Dispatch
    // results accumulates worker outputs across rounds.
    results []WorkerResult
    // plannerUsage accumulates planner-side LLMCalls + Tokens (the
    // worker-side is already on each WorkerResult.Result.Usage).
    plannerUsage agents.Usage
    // finalAnswer is the BuildAggregate output (populated by [FINAL]
    // node).
    finalAnswer string
}

// NewSupervisor constructs a Supervisor.
func NewSupervisor(name string, opts SupervisorOptions) *Supervisor {
    if name == "" {
        name = "supervisor"
    }
    return &Supervisor{name: name, opts: opts}
}

// Name implements agents.Agent.
func (s *Supervisor) Name() string { return s.name }

// Run implements agents.Agent.
func (s *Supervisor) Run(ctx context.Context, input string) (agents.Result, error) {
    if err := s.validate(); err != nil {
        return agents.Result{}, err
    }
    cg, err := s.compileGraph()
    if err != nil {
        return agents.Result{}, err
    }
    // MaxRounds × 3 nodes per round + slack — see Decision A on the
    // MaxSteps mapping
    maxSteps := s.opts.MaxRounds*3 + 4
    final, err := cg.Run(ctx, supervisorState{input: input},
        WithMaxSteps(maxSteps))
    if err != nil {
        // ErrGraphMaxSteps from StateGraph translates to ErrSupervisorMaxRounds
        // ctx.Err() / errors from planner / worker / aggregator surface as-is
        return agents.Result{}, s.translateErr(err)
    }
    return agents.Result{
        Answer: final.finalAnswer,
        Trace:  s.buildTrace(final),
        Usage:  s.aggregateUsage(final),
    }, nil
}
```

The trick: every "round" is **three** graph steps (`[PLAN]` then
`[DISPATCH]` then back to `[PLAN]`), so the `WithMaxSteps` cap must be
`MaxRounds * 3 + slack` to give the loop room. `defaultMaxSteps = 100`
in StateGraph is too low when `MaxRounds = 50`; explicit `WithMaxSteps`
is required.

### Pattern 2: Three nodes — `[PLAN]`, `[DISPATCH]`, `[FINAL]`

**What:** Three named nodes constitute the state machine. `[PLAN]`
invokes the planner agent and ParseDispatch. `[DISPATCH]` invokes the
named worker. `[FINAL]` invokes BuildAggregate. Conditional edge from
`[PLAN]` decides dispatch vs. finish vs. maxrounds.

**When to use:** Always — this IS the state machine.

**Example (skeleton):**

```go
func (s *Supervisor) compileGraph() (*CompiledGraph[supervisorState], error) {
    g := NewStateGraph[supervisorState]()

    g.AddNode("plan", func(ctx context.Context, st supervisorState) (supervisorState, error) {
        st.round++
        // Build planner prompt: input + prior results history
        plannerInput := s.buildPlannerPrompt(st)
        res, err := s.opts.Planner.Run(ctx, plannerInput)
        if err != nil {
            return st, fmt.Errorf("supervisor %q: planner round %d: %w", s.name, st.round, err)
        }
        st.lastPlannerAnswer = res.Answer
        st.plannerUsage.LLMCalls += res.Usage.LLMCalls
        st.plannerUsage.Tokens += res.Usage.Tokens
        d, perr := s.opts.ParseDispatch(res.Answer)
        if perr != nil {
            return st, fmt.Errorf("supervisor %q: parse dispatch round %d: %w", s.name, st.round, ErrSupervisorParseDispatch)
        }
        st.dispatch = d // nil = finish signal
        return st, nil
    })

    g.AddNode("dispatch", func(ctx context.Context, st supervisorState) (supervisorState, error) {
        d := st.dispatch
        worker, ok := s.opts.Workers[d.WorkerName]
        if !ok {
            return st, fmt.Errorf("%w: %q (round %d)", ErrSupervisorUnknownWorker, d.WorkerName, st.round)
        }
        res, err := worker.Run(ctx, d.Input)
        if err != nil {
            return st, fmt.Errorf("supervisor %q: worker %q round %d: %w", s.name, d.WorkerName, st.round, err)
        }
        st.results = append(st.results, WorkerResult{Dispatch: *d, Result: res})
        return st, nil
    })

    g.AddNode("final", func(ctx context.Context, st supervisorState) (supervisorState, error) {
        ans, err := s.opts.BuildAggregate(st.results)
        if err != nil {
            return st, fmt.Errorf("supervisor %q: aggregate: %w", s.name, err)
        }
        st.finalAnswer = ans
        return st, nil
    })

    g.SetEntry("plan").
        AddConditionalEdge("plan", func(st supervisorState) string {
            if st.dispatch == nil {
                return "final" // planner said "finished"
            }
            if st.round >= s.opts.MaxRounds {
                return "final" // hit the cap; aggregate what we have
            }
            return "dispatch"
        }).
        AddEdge("dispatch", "plan").
        AddEdge("final", NodeEnd)

    return g.Compile()
}
```

The conditional edge on `[PLAN]` is the load-bearing routing decision:
either go dispatch a worker or terminate via `[FINAL]`. The check
`round >= MaxRounds` inside the conditional edge ensures the final
round still gets a `BuildAggregate` call (rather than a hard
`ErrGraphMaxSteps` from the underlying graph cap). This is the
trade-off: hitting `MaxRounds` is a **graceful** termination (aggregate
what we have) rather than an **error** termination.

**Decision call** (planner ratifies): is MaxRounds hit an **error** or
a **graceful terminus**? Recommendation: **graceful** — symmetric with
`RoundRobinChat`'s `Stopped = "max_turns"` behavior (`roundrobin.go:90`,
verified). The planner's last dispatch IS executed; only the NEXT
planner call is skipped. Document this explicitly in the doc comment.

### Pattern 3: Trace assembly from state

**What:** Translate `supervisorState.results + lastPlannerAnswer`
into the `agents.Result.Trace` shape. Each round becomes 3 steps:
`StepThought` (planner's reasoning), `StepAction` (dispatch as tool
call), `StepObservation` (worker result). Final step: `StepFinal` with
the aggregate.

**When to use:** Always — required so callers see useful trace via
`Result.Trace`.

**Example (skeleton):**

```go
func (s *Supervisor) buildTrace(st supervisorState) []agents.Step {
    trace := make([]agents.Step, 0, len(st.results)*3+2)
    for i, wr := range st.results {
        // The planner's reasoning that LED to this dispatch was captured
        // by lastPlannerAnswer at the time of the dispatch — but we don't
        // preserve per-round lastPlannerAnswer in state. Trade-off: keep
        // only the LATEST planner answer in trace (under StepThought
        // before StepFinal), and emit StepAction + StepObservation per
        // worker call. This matches FanOutFanIn's compact trace style
        // (every Round's planner answer is on planner Result.Trace
        // through StepEvents — callers wanting per-round detail wire
        // RunStream).
        actionStep := agents.Step{
            Kind: agents.StepAction,
            Tool: wr.Dispatch.WorkerName,
            Args: wr.Dispatch.Input,
        }
        obsStep := agents.Step{
            Kind:   agents.StepObservation,
            Result: wr.Result.Answer,
        }
        trace = append(trace, actionStep, obsStep)
        _ = i
    }
    if st.lastPlannerAnswer != "" {
        trace = append(trace, agents.Step{
            Kind:    agents.StepThought,
            Content: "planner: " + st.lastPlannerAnswer,
        })
    }
    trace = append(trace, agents.Step{Kind: agents.StepFinal, Content: st.finalAnswer})
    return trace
}
```

**Decision call:** the trace stays compact (no per-round planner reasoning
buffered into state); per-round detail is available via `RunStream`.
Alternative: extend `supervisorState` with `plannerAnswers []string`.
Recommendation: ship compact; if a user wants every round's planner
answer in the trace, they wire `RunStream` and observe `StepEvent`s.

### Anti-Patterns to Avoid

- **Don't reimplement the loop.** KC-1 — the whole point is to reuse
  `StateGraph[S]`. A hand-rolled `for round := ...` makes the type
  composable with neither `StateGraph` (Supervisor not a node) nor with
  itself (no shared cancellation semantics).
- **Don't add a new `Step.Kind`.** KC-5 — existing 6 kinds suffice
  (`StepThought`, `StepAction`, `StepObservation`, `StepReflection`,
  `StepPlan`, `StepFinal`). A new "StepDispatch" would be a behavior
  change for any `agents.Result.Trace` consumer.
- **Don't add a new `StreamEvent.Kind`.** K1 — Supervisor's StepEvents
  are `agents.StepEvent` (different from `llm.StreamEvent`); but the
  underlying LLM stream events from workers are still K1's locked
  union. (`agents.StepEvent` is at `agent.go:28-33` and IS the
  agent-level union — no edit either.)
- **Don't `context.WithoutCancel(ctx)` workers.** Per 35-RESEARCH.md
  §"Carry-forward notes": workers receive `supervisorCtx` UNCHANGED so
  the budget tracker (and policy decorator state, if any) propagates.
  Detached children break CC-1.
- **Don't wrap worker errors with `ErrSupervisorMaxRounds` and
  vice-versa.** Separate error class per failure mode; callers
  distinguish via `errors.Is`. (Mirror `fanout.go:253-260`.)
- **Don't share state between Supervisor instances.** A Supervisor is
  a value type with options held by reference (planner / workers are
  pointers). Two concurrent `Run` calls on the SAME Supervisor with
  the SAME ctx must be safe — verify by `go test -race`. State is
  per-Run (built fresh by `Run`); no field on Supervisor is mutated by
  Run.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Round loop control flow | A new `for round := 0; round < MaxRounds; round++` | `orchestrate.StateGraph[supervisorState]` builder + Compile + Run | KC-1 — facade reuse; inherits cancellation + MaxSteps |
| Cancellation per step | A new `select { case <-ctx.Done() }` inside each handler | StateGraph's per-step ctx check at `graph.go:170-174` | Inherited automatically |
| MaxRounds cap | A new counter check | `WithMaxSteps(s.opts.MaxRounds * 3 + 4)` + conditional edge guard | Reuses `defaultMaxSteps`/`ErrGraphMaxSteps` (`graph.go:148, 208`) |
| Per-call budget enforcement | A new budget check before/after worker.Run | `generateFromPrompt` chokepoint (Phase 35-02) charges automatically | Inherited; CC-1 explicit |
| Policy enforcement per worker | A new gate per Supervisor | Caller wraps worker's model with `policy.Wrap` BEFORE constructing the worker | Inherited; CC-2 KC-3 model boundary |
| Worker name lookup | A new resolver | `map[string]agents.Agent` direct lookup, mirror `fanout.go:184` | Simple, proven |
| Streaming Supervisor's own events | A new channel orchestration | `runStreamFromBlocking` (`agent.go:98-125`) | Mirrors all 5 paradigms — same pattern |
| Test mocks | A new mock LLM | `llm.NewScriptedLLM(...)` + `examples/scriptedllm` adapter | Canonical mock per CLAUDE.md |
| Test agents | A new stub agent | Local `stubAgent` struct mirroring `fanout_test.go:34-44` pattern | Already in test fixtures |
| Compose-with-StateGraph test | A new minimal graph | Reuse `NewStateGraph[S]()` from the existing graph_test.go shape | Same package — direct access |

**Key insight:** This phase is 80% "thread three nodes through an
existing `StateGraph[S]` builder" and 20% "trace assembly + Aggregator
calls". Almost no new design — only a new vocabulary (Dispatch,
WorkerResult, ParseDispatch, BuildAggregate) layered on the existing
graph machinery.

## Runtime State Inventory

Not applicable — Phase 37 is **strict additive new code**. No rename /
refactor / migration. The `orchestrate.Supervisor` type is brand new
in `v0.6.2`; nothing pre-exists.

**For each category, explicit "nothing found":**

| Category | Status |
|----------|--------|
| Stored data | None — Supervisor is stateless across `Run` calls; no DB/file/cache. Verified: no DB usage in any v0.6.1 core file (core is stdlib-only; no DB drivers). |
| Live service config | None — core is a library, not a service. |
| OS-registered state | None — Supervisor is a library type, no daemon / task scheduler. |
| Secrets / env vars | None — Supervisor defines no env vars; gates carry their config in-process via the constructor args (Phase 36 same pattern). |
| Build artifacts | None — adding a new exported type to an existing package doesn't invalidate any existing artifact; `go.mod` stays unchanged (no new require). |

## Decision A: `supervisorState` shape — round counter + accumulating results

**The question:** What does the typed `S` in `StateGraph[S]` carry?

**The answer (recommended):**

```go
type supervisorState struct {
    input             string             // constant across Run
    round             int                // 1-indexed; incremented at top of [PLAN]
    lastPlannerAnswer string             // most recent planner Result.Answer
    dispatch          *Dispatch          // most recent parsed Dispatch; nil = finish
    results           []WorkerResult     // accumulated per round
    plannerUsage      agents.Usage       // planner-side rollup (worker-side on each WorkerResult.Result.Usage)
    finalAnswer       string             // populated by [FINAL] node
}
```

| Field | Purpose | Why this shape |
|-------|---------|----------------|
| `input` | The user prompt the run started with. Available to every node. | Composes with FanOutFanIn's `BuildAggregate(originalInput, plan, results)` signature spiritually (the Supervisor's BuildAggregate is simpler — only `results` — because per-round planner reasoning is in `lastPlannerAnswer`/`results[i].Dispatch`) |
| `round` | Counter; the `[PLAN]` node increments at entry. | Drives both the loop-cap check (`round >= MaxRounds`) and the planner-prompt assembly ("This is round 3 of up to N…") |
| `lastPlannerAnswer` | Only the latest planner answer needed by `ParseDispatch`. | Trade-off (Decision-call below): keep latest only; if user wants per-round planner trace, they wire `RunStream` |
| `dispatch` | The parsed Dispatch from this round's planner; nil means finish. | Drives the conditional edge from `[PLAN]` |
| `results` | Accumulator; `[DISPATCH]` appends; `[FINAL]` reads. | Symmetric to `FanOutFanInResult.WorkerResults` |
| `plannerUsage` | Rollup of planner-side LLM cost. | Per-worker usage lives on `WorkerResult.Result.Usage` already; planner needs its own rollup |
| `finalAnswer` | `[FINAL]` writes; `Supervisor.Run` reads. | Avoids a second variable to return from the graph |

**Why not also store `plannerAnswers []string`:** the alternative —
buffer every round's planner answer in state — bloats memory for
long-running Supervisors (50 rounds × 4 KB answer = 200 KB just in
state) and makes the trace assembly more complex. The compact shape
keeps per-Run state O(rounds) in the `results` slice (which the user
expects to be present) and O(1) in the planner-side fields. **Decision:
keep latest only**; per-round planner detail is via `RunStream`.

**Confidence:** HIGH. Symmetry with FanOutFanIn pattern; field-by-field
maps to a concrete need; no speculative fields.

## Decision B: `Dispatch` shape — name + input + opaque metadata

**The question:** What does the planner emit per round?

**The answer (recommended):**

```go
type Dispatch struct {
    WorkerName string         // MUST be a registered worker name; ErrSupervisorUnknownWorker otherwise
    Input      string         // free-form text passed to worker.Run(ctx, Input)
    Metadata   map[string]any // optional planner-side hints; transparent to Supervisor
}
```

| Field | Purpose | Why this shape |
|-------|---------|----------------|
| `WorkerName` | Routes to `Workers[name]`. | Mirrors FanOutFanIn's `PlannedTask.Worker` (`fanout.go:19`) |
| `Input` | The sub-task prompt. | The worker is `agents.Agent.Run(ctx, input string)`; `Input` matches the signature |
| `Metadata` | Opaque to Supervisor; the planner can pass hints (e.g., `{"priority": 1}`) that a worker's wrapping layer reads. | Future extension seam — KC-5-friendly because `map[string]any` is a single-field add today and any new key is additive |

**Why not include `WorkerInputJSON` (structured JSON arg) like
function-calling tools:** the worker is a generic `agents.Agent` —
its `Run(ctx, input string)` signature is text-only. Structured input
would require `interface{}`-casting at the worker side; in v1.2 the
"planner-emits-text, worker-reads-text" symmetry is preferable. v1.3
can add a `StructuredWorker` interface additive.

**Confidence:** HIGH. Direct shape parallel to FanOutFanIn's
`PlannedTask`; no speculative fields.

## Decision C: `WorkerResult` shape — Dispatch + agents.Result

**The question:** What does Supervisor record per round?

**The answer (recommended):**

```go
type WorkerResult struct {
    Dispatch Dispatch
    Result   agents.Result
}
```

| Field | Purpose | Why this shape |
|-------|---------|----------------|
| `Dispatch` | The dispatch that produced this result (carries `WorkerName`, `Input`, `Metadata`). | The user-facing trace + `BuildAggregate` need the link |
| `Result` | The worker's full `agents.Result` (Answer + Trace + Usage). | Aggregator + trace assembly + total usage rollup all need this |

**Why not `WorkerName string + Answer string + Usage agents.Usage`:**
flattening loses the Trace; the Trace is needed for debugging
nested-Supervisor scenarios (a Supervisor-as-worker's full trace is
inside its Result.Trace). Keep the embedded `agents.Result`.

**Confidence:** HIGH. Direct shape parallel to FanOutFanIn's
`TaskResult` (`fanout.go:24-27`).

## Decision D: `Planner` is `agents.Agent`, not a stricter interface

**The question:** Should the planner be a generic `agents.Agent`, or a
narrower `Planner` interface with typed return (e.g., `Plan(ctx,
input) (*Dispatch, error)`)?

**The answer (recommended): `agents.Agent`.**

**Rationale:**
- **Symmetry.** Workers are `agents.Agent`. If Planner is a different
  interface, users compose differently (one shape for planner, another
  for worker — surface fragmentation).
- **Reuse.** Any of the 5 existing paradigms (Simple, ReAct,
  PlanSolve, Reflection, FunctionCall) can be a planner. A specialist
  `Planner` interface would prevent that without an adapter.
- **The structured-output gap is filled by `ParseDispatch`.** LLM
  providers without native structured outputs (most v1.2 providers)
  need a text-to-struct adapter; that's exactly `ParseDispatch`.
  Making the planner emit text is the lowest-friction path.
- **Supervisor-supervises-Supervisor.** Since a Supervisor is an
  `agents.Agent` (KC-1), and a Planner is an `agents.Agent`, a
  Supervisor CAN be a Planner. This composability is lost if Planner
  is a separate interface.

**Confidence:** HIGH. Uniformity is the v1.2 design principle (see KC-1
"workers are agents.Agent" + the corollary that "Supervisor is also
agents.Agent").

**Trade-off:** A user with a future-LLM that supports structured outputs
loses the typed-return convenience. v1.3 can add a `StructuredPlanner`
interface additive (a Planner that returns `(*Dispatch, agents.Result,
error)` directly — Supervisor checks for it via type assertion before
falling back to text + ParseDispatch). KC-5-friendly.

## Decision E: Supervisor does NOT charge its own budget; chokepoint suffices

**The question:** Does the Supervisor add an additional `Charge` call
per round on top of the per-LLM-call charges that happen inside each
worker / planner via the chokepoint?

**The answer (recommended): NO. Inherit only.**

**Rationale:**
- **CC-3 verbatim:** "rounds count against `Budget.Calls`" — and rounds
  ARE counted, because each round invokes the planner (= 1 chokepoint
  call) AND a worker (= ≥1 chokepoint call). So `Budget.Calls = K *
  (planner_calls_per_round + worker_calls_per_round) * rounds`. The
  rounds are counted INDIRECTLY via the per-LLM-call charges.
- **CC-1 architecture:** the single chokepoint is `generateFromPrompt`.
  Adding a Supervisor-level charge would be a second enforcement layer
  — violating KC-4's "single integration chokepoint" principle.
- **MaxRounds as a separate cap:** `SupervisorOptions.MaxRounds` IS
  the Supervisor-level cap. `Budget.MaxCalls` is the cross-agent cap.
  They COEXIST (KC-4 explicit) — MaxRounds is the supervisor-loop's
  iteration cap; Budget.MaxCalls is the user's spend cap.
- **What happens when budget exhausts mid-Supervisor:** the chokepoint
  returns `ErrBudgetExceeded`; the worker's `agent.Run` propagates;
  the Supervisor's `[DISPATCH]` node returns the error; CompiledGraph's
  `Run` returns the error from `fn(ctx, state)`; Supervisor's `Run`
  surfaces it. **Errors propagate cleanly with `errors.Is(err,
  budget.ErrBudgetExceeded) == true`** — verified by reading the
  shipped chokepoint at `agent_chatmodel.go:24-50`.

**Confidence:** HIGH. CC-1 chokepoint is the single seam; double-charging
would over-count.

**Doc-comment requirement:** the Supervisor package doc MUST explicitly
state: "Budget enforcement happens at the `generateFromPrompt`
chokepoint (Phase 35), not at the Supervisor round boundary. Each
planner round and each worker call independently charges the budget
tracker bound to ctx. MaxRounds is a separate cap on the supervisor
loop's iteration count." This avoids confusion when a user expects
"Budget.MaxCalls = 10 means at most 10 rounds" (NO — at most 10 LLM
calls total).

## Decision F: `BuildAggregate` is called ONCE at the FINAL node

**The question:** Is `BuildAggregate` called per-round (running
summary) or once at the end?

**The answer (recommended): ONCE, in the `[FINAL]` node.**

**Rationale:**
- **Simplicity.** Single seam; symmetric to FanOutFanIn's
  `AggregateInputBuilder` (called once `fanout.go:120`).
- **Performance.** A per-round summary call doubles the LLM cost
  (every round emits a summary regardless of whether the user wants
  one).
- **Per-round summary is the planner's job.** If the user wants the
  planner to consider a running summary, the planner sees the
  accumulated `WorkerResult`s in its prompt (the Supervisor's planner
  prompt template includes prior results — see Decision G below).
- **MaxRounds-hit case:** when `[PLAN]`'s conditional edge routes to
  `[FINAL]` because `round >= MaxRounds`, the `[FINAL]` node calls
  `BuildAggregate(state.results)` with whatever results were
  accumulated. The user gets a partial answer (last-good aggregation)
  rather than an error. This matches `RoundRobinChat`'s
  `Stopped="max_turns"` graceful behavior (`roundrobin.go:90-92`).

**Confidence:** HIGH. Single-seam aggregation is the v1.2 design
principle; FanOutFanIn precedent.

**Trade-off:** A user wanting a running-summary tier (e.g., emit a
checkpoint every 5 rounds) writes a custom `BuildAggregate` that's
also wired as an OnStep — but `OnStep` isn't part of
`SupervisorOptions` today (it's per-paradigm). Decision-call: should
Supervisor expose `OnStep`? Recommendation: YES, additively — see
Decision G.

## Decision G: `RunStream` emits per-round `StepEvent`s via `runStreamFromBlocking`

**The question:** How does Supervisor expose per-round progress?

**The answer (recommended): `RunStream` returns a `<-chan
agents.StepEvent` channel; the per-round events are
`StepAction`-then-`StepObservation` pairs, identical to how `ReAct`
exposes its iteration through `StepEvent`s.**

**Rationale:**
- **Symmetry with the 5 paradigms.** Every paradigm has the same
  shape: blocking `Run` + channel-based `RunStream` via
  `runStreamFromBlocking` (`agent.go:98-125`). Supervisor SHOULD match
  — uniformity is a core design value.
- **No `OnStep` field on `SupervisorOptions` initially.** Instead,
  callers wire `OnStep`-style observation through their planner /
  workers (which DO have `OnStep` fields). Adding `OnStep` to
  Supervisor would be a second observation seam without a strong need.
  **Carry-forward note**: if a v1.3 user wants Supervisor-level
  `OnStep`, it's strictly additive (new field on `SupervisorOptions`)
  and KC-5-friendly.
- **`agents.StepEvent` is the existing transport.** No new event
  type. The Supervisor's `RunStream` emits the same `StepEvent`
  union the 5 paradigms emit. Consumer compatibility: any existing
  `StepEvent`-aware UI (SSE handler, log printer) works unchanged.

**Confidence:** HIGH. Uniformity with 5 paradigms is the v1.2 design
principle.

**Implementation sketch:**

```go
func (s *Supervisor) RunStream(ctx context.Context, input string) (<-chan agents.StepEvent, error) {
    return runStreamFromBlocking(ctx, func(ctx context.Context, cb func(agents.Step)) (agents.Result, error) {
        // Hooked-up cb wired through the graph's nodes — but the graph
        // doesn't expose per-node-step callbacks. The simplest path:
        // record per-node steps in the supervisorState (compact: only
        // the steps emitted at [DISPATCH] node — one per worker call),
        // and emit them to cb after each [DISPATCH] node finishes.
        // ...
    })
}
```

**Trade-off:** Supervisor's `RunStream` uses the package-private
`runStreamFromBlocking` from `agent.go:98-125`. Reading it: it's NOT
exported. **Decision call:** either (a) export it as
`agents.RunStreamFromBlocking` (additive — KC-5-friendly; net positive
for users wanting to author their own paradigms), OR (b) reimplement
the 27-line helper locally in `orchestrate/supervisor.go`. Recommendation:
**(b) reimplement locally** — it's small, doesn't widen
`agents.` surface, and avoids a 27-line addition to the agents
package. Planner ratifies in 37-01.

## Decision H: Example shape — 1 planner + 2 specialist workers + 2 rounds

**The question:** What does the demo look like?

**The answer (recommended):** `examples/08-supervisor/main.go` —
deterministic via `ScriptedLLM`:
- Planner: a `SimpleAgent` over a `ScriptedLLM` that emits two
  responses: round 1 says "dispatch to researcher: 'find facts about
  X'", round 2 says "dispatch to summarizer: 'condense the facts
  above'", round 3 (if reached) says "FINISH".
- Workers:
  - `researcher`: a `SimpleAgent` over a `ScriptedLLM` that emits one
    response: "Fact 1; Fact 2; Fact 3."
  - `summarizer`: a `SimpleAgent` over a `ScriptedLLM` that emits one
    response: "Summary: 3 facts about X."
- `ParseDispatch`: a small text parser that extracts `dispatch to
  WORKER: 'INPUT'` from the planner's text, or returns `(nil, nil)`
  if the planner emitted `FINISH`.
- `BuildAggregate`: joins all worker results into a final answer
  string.
- The output to stdout shows: planner's round 1 → researcher's facts
  → planner's round 2 → summarizer's summary → final aggregate.

The demo also includes:
- A `demoBudget` showing Supervisor under a `budget.Budget{MaxCalls:
  3}` cap that exhausts mid-run (catches the 4th LLM call).
- A `demoComposeWithStateGraph` showing Supervisor as a node in an
  outer `StateGraph[outerState]` (proves KC-1's "facade works both
  directions").

**Confidence:** HIGH. The shape parallels `examples/06-budget/main.go`
(verified by direct read).

**Rationale:** Two specialists are the minimum to demonstrate
multi-worker routing (one would be FanOutFanIn-as-Supervisor; trivial).
Two rounds is the minimum to demonstrate the loop. Scripted responses
are deterministic per CLAUDE.md.

## Decision I: 4-slice breakdown (matches ROADMAP §"Planned work" outline)

**The question:** How many slices, in what order?

**The answer (recommended): 4 slices, all in `llm-agent` core.**

Mirrors the ROADMAP §"Planned work" outline (37-01..04) and the Phase
36 5-slice shape but compressed because Supervisor is fewer types per
slice. See §"Slice Breakdown" below for the full table.

**Confidence:** HIGH. Matches ROADMAP; matches Phase 35/36 cadence; no
new variables.

## Common Pitfalls

### Pitfall 1: Worker name lookup misses → silent fallback

**What goes wrong:** A user registers `Workers: map[string]agents.Agent{
"researcher": r}` and the planner emits `dispatch to RESEARCHER:
...` (capitalization mismatch). Lookup fails; `[DISPATCH]` node
returns `ErrSupervisorUnknownWorker`.

**Why it happens:** Case-sensitive map key vs. LLM-emitted text
inconsistency.

**How to avoid:** Document explicitly in `SupervisorOptions.Workers`
field doc: "Lookup is case-sensitive. Convention: lowercase names
(e.g., `researcher`, `summarizer`, `coder`)." `ParseDispatch` is the
seam where normalization can happen — the caller's `ParseDispatch`
implementation may `strings.ToLower(name)` if desired.

**Warning signs:** `TestUnknownWorker_Errors` test fires
`ErrSupervisorUnknownWorker`. Add a test: `TestUnknownWorker_Case` to
prove the documented case-sensitive behavior.

### Pitfall 2: MaxRounds × 3 + slack miscalculated → premature
`ErrGraphMaxSteps`

**What goes wrong:** User sets `MaxRounds: 10`. Supervisor sets
`WithMaxSteps(30)` (10 × 3). The underlying graph hits step 31 on the
30th step (`[PLAN]` → `[DISPATCH]` → `[PLAN]` → ... → step 30 = `[PLAN]`,
needs step 31 = either `[DISPATCH]` or `[FINAL]`) and surfaces
`ErrGraphMaxSteps` instead of the friendly `ErrSupervisorMaxRounds`.

**Why it happens:** Off-by-one + the `[FINAL]` node also counts as a
step.

**How to avoid:** Use `WithMaxSteps(MaxRounds * 3 + 4)` — the `+ 4`
slack covers: entry into `[PLAN]` (step 1), the final `[PLAN]` that
routes to `[FINAL]` (step 2N+1), `[FINAL]` itself (step 2N+2), and one
slot of slack. Translate `ErrGraphMaxSteps` to
`ErrSupervisorMaxRounds` in `Supervisor.Run`'s err-translator
(see Pattern 1 example).

**Warning signs:** Test `TestSupervisor_MaxRoundsExceeded` should
exercise `MaxRounds: 2` with a planner that emits 3 dispatches; assert
`errors.Is(err, ErrSupervisorMaxRounds)` AND NOT `errors.Is(err,
ErrGraphMaxSteps)`.

### Pitfall 3: `ParseDispatch` returns `(nil, nil)` ambiguity with
parse-error

**What goes wrong:** `ParseDispatch` author intends "the planner said
FINISH" but the planner emitted garbage. The author returns `(nil,
nil)` for both finish-signal and parse-failure. Supervisor can't
distinguish "loop is done" from "parser failed".

**Why it happens:** Two semantics overloaded on one return shape.

**How to avoid:** Document the contract explicitly:
- `ParseDispatch` returns `(*Dispatch, nil)` → continue; dispatch this
  worker.
- `ParseDispatch` returns `(nil, nil)` → terminate cleanly; planner
  signaled finish (this is the "user said done" path).
- `ParseDispatch` returns `(nil, err)` → terminate with error;
  surfaces as `ErrSupervisorParseDispatch` wrapping `err`.

Add a test: `TestParseDispatch_NilNilTerminates` AND
`TestParseDispatch_NilErrFailsRun`.

**Warning signs:** A test where the planner emits gibberish must see
`ErrSupervisorParseDispatch`, not silent termination.

### Pitfall 4: Worker ctx detached → budget tracker lost

**What goes wrong:** A future contributor "improves" Supervisor by
wrapping `worker.Run(ctx, ...)` in `context.WithoutCancel(ctx)` to
"prevent cancellation cascading". The budget tracker bound to the
parent ctx is no longer reachable via `budget.From(workerCtx)`;
workers see "no budget" and run unenforced.

**Why it happens:** Misunderstanding "ctx propagation must be
unbroken" (35-RESEARCH.md carry-forward).

**How to avoid:** **Hard rule** documented in package doc: "Workers
receive the supervisor's ctx UNCHANGED. Do not use
`context.WithoutCancel`, `context.Background()`, or any detached
child. The budget tracker (Phase 35) and policy decorator state
(Phase 36) propagate via ctx."

Add a test: `TestSupervisor_BudgetPropagatesToWorker` that wires a
`budget.NewStrict(Budget{MaxCalls: 3})` and asserts the 4th LLM call
across planner + workers fires `ErrCallsExceeded`.

**Warning signs:** `go test -race ./orchestrate/...` flake-free; the
budget propagation test goes red if a detached ctx slips in.

### Pitfall 5: Supervisor-supervises-Supervisor infinite loop

**What goes wrong:** User wires Supervisor B as a worker of Supervisor
A. Supervisor A's `[DISPATCH]` calls B.Run(ctx). B's planner emits a
dispatch that routes BACK to a worker that is A itself (or that
re-enters A). Mutual recursion; no cycle detector.

**Why it happens:** Composition without termination logic.

**How to avoid:** Two layers of protection:
1. Each Supervisor has its own `MaxRounds`. Eventually one of them hits
   it.
2. The shared `Budget.MaxCalls` (if any) caps the total LLM spend
   across the entire composition tree.

Document: "Supervisor-of-Supervisor composition has no cycle
detection. Both supervisors must have finite `MaxRounds`; a shared
`Budget.MaxCalls` is recommended for additional safety."

**Warning signs:** No automated test (composition shape is user
responsibility); a composition-stress test in `examples/08-supervisor/`
could demonstrate, but isn't strictly required.

## Code Examples

Verified patterns to copy / mirror in implementation:

### Example A: StateGraph[S] builder pattern (the substrate)

```go
// Source: orchestrate/graph.go:36-43, 88-131 [VERIFIED: direct read]
g := orchestrate.NewStateGraph[supervisorState]()
g.AddNode("plan", planFn).
  AddNode("dispatch", dispatchFn).
  AddNode("final", finalFn).
  SetEntry("plan").
  AddConditionalEdge("plan", routeFn).
  AddEdge("dispatch", "plan").
  AddEdge("final", orchestrate.NodeEnd)
cg, err := g.Compile()
// ...
final, err := cg.Run(ctx, supervisorState{input: input}, orchestrate.WithMaxSteps(maxSteps))
```

### Example B: FanOutFanIn pattern (the cognate to deviate from)

```go
// Source: orchestrate/fanout.go:66-129 [VERIFIED: direct read]
f := orchestrate.NewFanOutFanIn("research", orchestrate.FanOutFanInOptions{
    Planner:    planner,
    Workers:    map[string]agents.Agent{"researcher": w1, "summarizer": w2},
    Aggregator: aggregator,
    ParsePlan:  parseTasks,
})
res, err := f.Run(ctx, "investigate X")
```

The Supervisor's surface is intentionally similar but distinct:

```go
// Source: NEW — Phase 37
sup := orchestrate.NewSupervisor("research", orchestrate.SupervisorOptions{
    Planner:        planner,
    Workers:        map[string]agents.Agent{"researcher": w1, "summarizer": w2},
    MaxRounds:      10,
    ParseDispatch:  parseDispatch,   // text → *Dispatch (or nil = finish)
    BuildAggregate: buildAggregate,  // []WorkerResult → string
})
res, err := sup.Run(ctx, "investigate X")
```

Differences vs. FanOutFanIn:
- No `Aggregator agents.Agent` field — Supervisor's aggregator is a
  pure function (the planner is the LLM-driven decision-maker; the
  aggregator is text concatenation/formatting). v1.3 may add
  `LLMAggregator agents.Agent` additive if needed.
- `MaxRounds` field (new) — caps loop iterations.
- `ParseDispatch` returns ONE Dispatch per round (vs. FanOutFanIn's
  `ParsePlan` returning many tasks at once).

### Example C: agents.Agent interface implementation

```go
// Source: NEW — Phase 37, but mirrors orchestrate.FanOutFanIn shape
type Supervisor struct {
    name string
    opts SupervisorOptions
}

var _ agents.Agent = (*Supervisor)(nil) // compile-time assert

func (s *Supervisor) Name() string { return s.name }
func (s *Supervisor) Run(ctx context.Context, input string) (agents.Result, error) { /* ... */ }
func (s *Supervisor) RunStream(ctx context.Context, input string) (<-chan agents.StepEvent, error) { /* ... */ }
```

Note: FanOutFanIn does NOT satisfy `agents.Agent` today (it has a
custom `FanOutFanInResult` return type — `fanout.go:80-129`). The
Supervisor MUST satisfy `agents.Agent` per KC-1 — that's a distinct
property (Supervisor-as-worker composition).

### Example D: Compose-with-StateGraph test pattern

```go
// Source: NEW — proves KC-1's "facade works both directions"
func TestSupervisor_InsideStateGraph(t *testing.T) {
    sup := orchestrate.NewSupervisor("inner", ...)

    type outerState struct {
        input  string
        supRes string
        final  string
    }

    g := orchestrate.NewStateGraph[outerState]()
    g.AddNode("preprocess", func(_ context.Context, s outerState) (outerState, error) {
        s.input = "preprocessed: " + s.input
        return s, nil
    })
    g.AddNode("supervise", func(ctx context.Context, s outerState) (outerState, error) {
        res, err := sup.Run(ctx, s.input)
        if err != nil { return s, err }
        s.supRes = res.Answer
        return s, nil
    })
    g.AddNode("postprocess", func(_ context.Context, s outerState) (outerState, error) {
        s.final = "[" + s.supRes + "]"
        return s, nil
    })
    g.SetEntry("preprocess").
        AddEdge("preprocess", "supervise").
        AddEdge("supervise", "postprocess").
        AddEdge("postprocess", orchestrate.NodeEnd)
    cg, _ := g.Compile()
    final, err := cg.Run(context.Background(), outerState{input: "hi"})
    // assert err == nil, final.final has expected shape
}

func TestStateGraph_InsideSupervisor(t *testing.T) {
    // Construct a small CompiledGraph that does work
    type wState struct{ in, out string }
    wg := orchestrate.NewStateGraph[wState]()
    wg.AddNode("transform", func(_ context.Context, s wState) (wState, error) {
        s.out = "transformed: " + s.in
        return s, nil
    })
    wg.SetEntry("transform").AddEdge("transform", orchestrate.NodeEnd)
    cg, _ := wg.Compile()

    // Wrap the CompiledGraph in a small agents.Agent adapter
    graphWorker := &graphAsAgent{cg: cg, name: "graph-worker"}

    sup := orchestrate.NewSupervisor("outer", orchestrate.SupervisorOptions{
        Planner:        plannerStub,
        Workers:        map[string]agents.Agent{"graph-worker": graphWorker},
        MaxRounds:      3,
        ParseDispatch:  parseDispatchStub,
        BuildAggregate: buildAggregateStub,
    })
    res, err := sup.Run(context.Background(), "go")
    // assert res.Answer contains "transformed: ..."
}
```

The `graphAsAgent` adapter is ~15 LOC: it wraps a `*CompiledGraph[S]`
and exposes `Name/Run/RunStream`, picking out a final string from the
state. Defined in-test, not exported.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Hand-rolled supervisor loop (`for round := 0; round < N; round++`) | `StateGraph[S]` facade | KC-1 (2026-05-20) | Reuses cancellation + MaxSteps; composes with other graphs |
| Single coordination shape (`FanOutFanIn` plan-once + fan-out) | Iterative `Supervisor` with re-planning | CC-3 (2026-05-20) | Fills the v0.6.0 gap — `FanOutFanIn` is one-shot, `Supervisor` is loop-with-routing |
| `internal/research.Coordinator` (hand-rolled plan→summarize×N→report) | Will MIGRATE to `Supervisor` in a future ecosystem-alignment phase | v1.3 candidate | Internal-only today; the hand-roll stays until a real use case forces migration. v1.2 SUMMARY notes this is `internal/*` — out of scope for v1.2. |
| Per-paradigm `MaxSteps`/`MaxTurns`/`MaxRounds` | Per-paradigm cap COEXISTS with cross-agent `Budget.MaxCalls` | KC-4 (Phase 35) | Supervisor's `MaxRounds` is the supervisor-loop cap; `Budget.MaxCalls` is the cross-agent spend cap; both coexist |

**Deprecated / not applicable:**
- Nothing in core deprecated (this is an additive type).
- `FanOutFanIn` is NOT deprecated — it's the correct shape for
  "plan-once, fan-out, aggregate". Supervisor is the iterative
  cognate; users pick the shape that matches their workflow.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | stdlib `testing` + `go test -race` |
| Config file | None — pure stdlib |
| Quick run command | `cd .../llm-agent && GOWORK=off go test ./orchestrate/... -count=1` |
| Full suite command | `cd .../llm-agent && GOWORK=off go vet ./... && GOWORK=off go test ./... -count=1 && GOWORK=off go test -race ./orchestrate/... -count=1` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CC-3 | `Supervisor` implements `agents.Agent` (Name/Run/RunStream) | compile-time | `go vet ./orchestrate/` | ❌ Wave 0 (37-01) |
| CC-3 | `Supervisor.Run` happy path: 2 rounds × 2 workers → final aggregate | unit | `go test ./orchestrate/ -run TestSupervisor_HappyPath` | ❌ Wave 0 (37-01) |
| CC-3 | `Supervisor.Run` validates options: nil planner / nil workers map / nil ParseDispatch / nil BuildAggregate | unit | `go test ./orchestrate/ -run TestSupervisor_Validation` | ❌ Wave 0 (37-01) |
| CC-3 | `Supervisor.Run` `MaxRounds` exceeded routes to `[FINAL]` (graceful) | unit | `go test ./orchestrate/ -run TestSupervisor_MaxRoundsExceeded` | ❌ Wave 0 (37-01) |
| CC-3 | `Supervisor.Run` unknown worker name → `ErrSupervisorUnknownWorker` | unit | `go test ./orchestrate/ -run TestSupervisor_UnknownWorker` | ❌ Wave 0 (37-01) |
| CC-3 | `Supervisor.Run` `ParseDispatch` returns `(nil, err)` → `ErrSupervisorParseDispatch` | unit | `go test ./orchestrate/ -run TestSupervisor_ParseDispatchError` | ❌ Wave 0 (37-01) |
| CC-3 | `Supervisor.Run` `ParseDispatch` returns `(nil, nil)` → graceful finish before MaxRounds | unit | `go test ./orchestrate/ -run TestSupervisor_ParseDispatchFinish` | ❌ Wave 0 (37-01) |
| CC-3 | `Supervisor.Run` ctx cancellation mid-round → `ctx.Err()` | unit | `go test ./orchestrate/ -run TestSupervisor_CtxCancel` | ❌ Wave 0 (37-01) |
| CC-3 | `Supervisor.RunStream` emits StepEvent stream with `StepAction`/`StepObservation` pairs per round + terminal `StepFinal` | unit | `go test ./orchestrate/ -run TestSupervisor_RunStreamEmitsRoundEvents` | ❌ Wave 0 (37-01) |
| CC-3 | `Supervisor.Run` returns `Result.Usage` aggregating planner + all workers | unit | `go test ./orchestrate/ -run TestSupervisor_UsageRollup` | ❌ Wave 0 (37-01) |
| CC-3 | Budget propagates: `Budget{MaxCalls: 3}` on ctx → `ErrCallsExceeded` on 4th LLM call across planner + workers | integration | `go test ./orchestrate/ -run TestSupervisor_BudgetPropagatesToWorker` | ❌ Wave 0 (37-02) |
| CC-3 | Policy enforces per worker: worker's model `policy.Wrap`-ped → blocked dispatch surfaces `ErrBlocked` | integration | `go test ./orchestrate/ -run TestSupervisor_PolicyPerWorker` | ❌ Wave 0 (37-02) |
| CC-3 | Compose: Supervisor inside `StateGraph[S]` works | integration | `go test ./orchestrate/ -run TestSupervisor_InsideStateGraph` | ❌ Wave 0 (37-03) |
| CC-3 | Compose: `StateGraph[S]` (as worker via adapter) inside Supervisor works | integration | `go test ./orchestrate/ -run TestStateGraph_InsideSupervisor` | ❌ Wave 0 (37-03) |
| CC-3 | Compose: Supervisor as worker of another Supervisor (composition) | integration | `go test ./orchestrate/ -run TestSupervisor_OfSupervisor` | ❌ Wave 0 (37-03) |
| CC-3 | Race: concurrent `Run` calls on independent Supervisor instances | race | `go test -race ./orchestrate/...` | ❌ Wave 0 (37-01) |
| CC-3 | stdlib-only: `go list -deps ./orchestrate/` shows only stdlib + existing internal imports | shape | `go list -f '{{join .Imports "\n"}}' ./orchestrate/ \| grep -vE 'stdlib-or-internal'` returns 0 lines | ❌ Wave 0 (37-04) |
| CC-3 | Example runs deterministically | example | `cd examples && go run ./08-supervisor` exits 0 | ❌ Wave 0 (37-03) |
| CC-3 | go.mod unchanged after Phase 37 | shape | `git diff go.mod \| wc -l` is 0 | ❌ Wave 0 (37-04) |
| CC-3 | KC-5 verification: no edit to `agents.Agent`/`memory.Memory`/`orchestrate.NodeFunc`/`llm.ChatModel` | shape | `git diff main -- agent.go memory/ orchestrate/graph.go llm/chatmodel.go llm/stream.go llm/types.go llm/capabilities.go` shows only doc-comment additions or no changes | ❌ Wave 0 (37-04) |

### Sampling Rate

- **Per task commit:** `go test ./orchestrate/... -count=1`
- **Per wave merge:** `go test ./... -count=1 && go test -race ./orchestrate/...`
- **Phase gate:** `go vet ./... && go test ./... -count=1 && go test -race ./orchestrate/... && go list -deps ./orchestrate/ | check-stdlib-only` (the audit in 37-04)

### Wave 0 Gaps

- [ ] `orchestrate/supervisor.go` — Supervisor + SupervisorOptions + Dispatch + WorkerResult + state + nodes + Compile + sentinels (37-01)
- [ ] `orchestrate/supervisor_test.go` — happy path + 10 functional tests + race (37-01)
- [ ] `orchestrate/supervisor_compose_test.go` — Supervisor-in-StateGraph + StateGraph-in-Supervisor + Supervisor-of-Supervisor (37-03)
- [ ] `orchestrate/doc.go` — package-doc edit adding Supervisor entry (additive doc-comment only) (37-01)
- [ ] `orchestrate/supervisor_budget_test.go` (or merged into supervisor_test.go) — budget propagation + policy compose (37-02)
- [ ] `examples/08-supervisor/main.go` + `README.md` + `main_test.go` — deterministic demo (37-03)

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Library type; provider adapters handle creds. KC-5: no edit. |
| V3 Session Management | no | Library, not session-state. |
| V4 Access Control | no | Library; caller wires worker access checks via custom Gates (policy.Wrap) on the worker's model if desired. |
| V5 Input Validation | yes (via composition) | Supervisor does NOT validate input itself. The caller composes `policy.Wrap` (Phase 36) on each worker's model to validate inputs at the model boundary. Supervisor's role: propagate ctx so policy/budget bound to ctx fires uniformly across planner + all workers. |
| V6 Cryptography | no | No crypto in Supervisor. |
| V7 Error Handling & Logging | yes | Sentinel error class per failure mode (`ErrSupervisorMaxRounds`, `ErrSupervisorUnknownWorker`, `ErrSupervisorParseDispatch`, etc.) distinguishable via `errors.Is`. No information leakage in error messages (worker names + round numbers are stable identifiers; no raw planner LLM output included). |
| V8 Data Protection | yes (via composition) | `policy.Wrap(model, NewPIIRedactor())` on the worker's model scrubs PII at the LLM call boundary. Supervisor itself stores no data beyond per-Run `supervisorState` (in-memory, freed after Run returns). |
| V9 Communications | no | Network layer is the provider adapter's concern. |
| V11 Business Logic | partial | Custom workers + custom `ParseDispatch`/`BuildAggregate` express business rules. Supervisor is policy-agnostic by design. |

### Known Threat Patterns for `orchestrate.Supervisor`

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Runaway loop (planner emits dispatch forever) | Denial of Service + Cost | `MaxRounds` cap (Supervisor-level) + `Budget.MaxCalls` cap (cross-agent, Phase 35); double protection per KC-4 |
| Worker that hangs without honoring ctx | Denial of Service | Workers MUST honor `ctx.Done()` (the `agents.Agent` interface contract); StateGraph's per-step `select { ctx.Done }` catches between nodes; `Budget.MaxWall` (Phase 35) provides hard wall-clock cap |
| Cross-worker context-leak (Supervisor B as worker A learns about A's secrets via ctx values) | Information Disclosure | Supervisor passes ctx UNCHANGED — workers inherit whatever values the caller put on ctx. By Go convention, ctx values are caller-scoped; Supervisor doesn't add ANY ctx values, so no new leak. If the caller stored secrets via ctx (anti-pattern), they propagate; that's a caller hygiene issue, not a Supervisor issue. |
| Malicious planner LLM output triggers infinite worker dispatch | Denial of Service + Cost | MaxRounds + Budget; documented "planner LLM is trusted to terminate; MaxRounds is the fail-safe" |
| Supervisor of Supervisor cyclic composition | Denial of Service | Each Supervisor has its own MaxRounds; shared Budget.MaxCalls; documented composition limit |
| `BuildAggregate` reveals all worker outputs without filtering | Information Disclosure | Caller's `BuildAggregate` is user-defined; if a worker's output contains PII, the worker's model should be `policy.Wrap`-ped with `PIIRedactor` at PostGenerate. Supervisor does NOT scrub between worker output and aggregator input. |

**Carry-forward debt (acknowledged):**
- Supervisor itself adds no new security primitives — it inherits
  budget + policy from the chokepoint and the per-worker
  `policy.Wrap`. The recommended composition is documented but not
  enforced by Supervisor (a user can construct an unwrapped worker).
  Documented in package doc; recorded as "caller hygiene" rather than
  a code-side enforcement.

## Project Constraints (from CLAUDE.md)

| Constraint | Source | Impact on Phase 37 |
|------------|--------|---------------------|
| Core repo stays stdlib-only | CLAUDE.md Rule 1 | `orchestrate/supervisor.go` imports only stdlib + `github.com/costa92/llm-agent` (the `agents` package) + the existing `orchestrate/` symbols. Zero new `require` in `go.mod`. |
| No K8s | CLAUDE.md Rule 2 | n/a — Supervisor is a library type |
| No `replace` in tagged release | CLAUDE.md Rule 3 | `v0.6.2` tag (Phase 37 → 38) must have no replace directives in `go.mod`. CI dep-currency gate enforces. |
| `go.work` is `.gitignore`d | CLAUDE.md Rule 4 | All `go test` commands in exit-gate slice use `GOWORK=off`. |
| Capabilities per-(provider × model) | CLAUDE.md Rule 5 (K2) | Supervisor does NOT introspect worker model capabilities. Capability negotiation is the worker's job (its underlying agent paradigm). |
| StreamEvent typed union, stable Index | CLAUDE.md Rule 6 (K1) | No new `llm.StreamEvent.Kind` in v1.2. `agents.StepEvent` (the agent-level union) is also unchanged — Supervisor emits existing kinds. |
| OTel attaches as decorator | CLAUDE.md Rule 7 (K3) | Policy + OTel are decorator layers at the worker's model boundary; Supervisor is policy-agnostic by design. |
| Refsvc hard caps + DISABLE_LLM=1 | CLAUDE.md Rule 8 (K7) | n/a — Phase 37 is core only; refsvc is `llm-agent-customer-support` (sister, not in v1.2 scope). |
| Files NOT to touch | CLAUDE.md "Files you should NOT touch" | `LICENSE` / `OWNERS` / `.github/workflows/*` / `go.mod` (no new require) — none touched by Phase 37. |
| Use `ScriptedLLM` for examples | CLAUDE.md "When the user asks for code" | `examples/08-supervisor/main.go` MUST use `ScriptedLLM` (no real providers); deterministic. |
| Tests via `go vet ./... && go test ./...` | CLAUDE.md "When the user asks for code" | Phase exit gate is this command set, augmented with `-race` on `./orchestrate/...`. |
| No `go.sum` by design | CLAUDE.md "When the user asks for code" | Exit-gate check: `git status --short go.sum` returns empty. |

## Slice Breakdown (recommended — planner ratifies in 37-01)

Recommended **4 slices**, all in the core repo `llm-agent`. Total
estimated effort: ~1000-1300 LOC including tests + example + docs.
Mirrors the ROADMAP §"Planned work" outline (37-01..04). Sequential
(no parallelization within phase).

| Slice | Wave | Type | Repo | Files modified | Requirements | Must-haves |
|---|---|---|---|---|---|---|
| **37-01** | 1 | execute | `llm-agent` | `orchestrate/supervisor.go` (new); `orchestrate/supervisor_test.go` (new); `orchestrate/doc.go` (edit — additive paragraph) | CC-3 (Supervisor skeleton + state machine) | **Supervisor + types + state machine — no compose tests, no budget/policy integration tests yet (separate slices).** Surface: `NewSupervisor`, `SupervisorOptions{Planner, Workers, MaxRounds, ParseDispatch, BuildAggregate}`, `Dispatch{WorkerName, Input, Metadata}`, `WorkerResult{Dispatch, Result}`, `DispatchParser` and `Aggregator` type aliases, sentinels (`ErrSupervisorNilPlanner`, `ErrSupervisorNoWorkers`, `ErrSupervisorNilParseDispatch`, `ErrSupervisorNilBuildAggregate`, `ErrSupervisorUnknownWorker`, `ErrSupervisorMaxRounds`, `ErrSupervisorParseDispatch`). Three internal nodes: `[plan]` / `[dispatch]` / `[final]` wired with conditional edge from `[plan]`. `Supervisor` satisfies `agents.Agent` (Name/Run/RunStream), enforced by `var _ agents.Agent = (*Supervisor)(nil)` compile-time assertion. `RunStream` reimplements `runStreamFromBlocking` locally (Decision G recommendation b). Tests: `TestSupervisor_HappyPath` (2 rounds × 2 workers → final aggregate), `TestSupervisor_Validation` (table — nil planner / nil workers map / nil ParseDispatch / nil BuildAggregate / zero MaxRounds), `TestSupervisor_MaxRoundsExceeded` (planner emits 3 dispatches, MaxRounds=2 → graceful BuildAggregate over 2 results), `TestSupervisor_UnknownWorker`, `TestSupervisor_ParseDispatchError`, `TestSupervisor_ParseDispatchFinish` (returns nil, nil), `TestSupervisor_CtxCancel` (ctx.Cancel() during a worker call), `TestSupervisor_RunStreamEmitsRoundEvents`, `TestSupervisor_UsageRollup` (planner + all workers summed in Result.Usage), `TestSupervisor_NameDefault`, `TestSupervisor_SatisfiesAgentInterface` (the compile-time assertion is the primary test; this exists to document intent). Race: `go test -race`. Exit gate: `go vet ./orchestrate/... && go test ./orchestrate/... -count=1 && go test -race ./orchestrate/... && go list -deps ./orchestrate/` shows no new third-party module. |
| **37-02** | 2 | execute | `llm-agent` | `orchestrate/supervisor_budget_test.go` (new) OR merged into supervisor_test.go | CC-3 (budget propagation + policy compose with worker) | **Phase 35/36 integration tests for Supervisor.** Tests: (a) `TestSupervisor_BudgetPropagatesToWorker` — wire `Budget{MaxCalls: 3}` on ctx; Supervisor with 1 planner + 1 worker × MaxRounds=3 (planner counts as 3 calls + workers count as 3 more = 6 chokepoint charges in best case); assert 4th charge surfaces `ErrCallsExceeded` from within a worker's `agent.Run`; assert `errors.Is(err, budget.ErrBudgetExceeded)` AND assert tracker snapshot's `Calls == 3` (the cap). (b) `TestSupervisor_BudgetMaxWall` — wire `Budget{MaxWall: 50ms}` on ctx; Supervisor with a worker whose underlying model sleeps 200ms; assert `errors.Is(err, context.DeadlineExceeded)` (no new sentinel). (c) `TestSupervisor_PolicyPerWorker` — wrap one worker's model with `policy.Wrap(model, gateThatBlocksOnSecondCall)`; planner dispatches that worker twice; assert second dispatch surfaces `policy.ErrBlocked`; assert Supervisor doesn't swallow the error (it surfaces from the `[dispatch]` node and `Supervisor.Run` returns it). (d) `TestSupervisor_PolicyPerPlanner` — wrap the planner's model with policy; assert blocked planner surfaces `policy.ErrBlocked` from the `[plan]` node. (e) `TestSupervisor_BudgetBeatsPolicy` — same model wrapped in both; budget cap hits first (chokepoint pre-charge before policy decorator); assert `errors.Is(err, budget.ErrCallsExceeded)`. (f) Document in test file header: the composition stack is `policy.Wrap(otelmodel.Wrap(model))` (KC-3); budget enforces underneath (KC-4 chokepoint). Exit gate: `go vet && go test -race ./orchestrate/...`. |
| **37-03** | 3 | execute | `llm-agent` | `orchestrate/supervisor_compose_test.go` (new); `examples/08-supervisor/main.go` (new); `examples/08-supervisor/README.md` (new); `examples/08-supervisor/main_test.go` (new) | CC-3 (compose-with-StateGraph + example) | **The KC-1 facade verification + the user-facing demo.** Compose tests: (a) `TestSupervisor_InsideStateGraph` — build a 3-node outer StateGraph (`preprocess` → `supervise` → `postprocess`); `supervise` node calls `sup.Run(ctx, state.input)`; assert outer.Run produces the expected final string. (b) `TestStateGraph_InsideSupervisor` — build a small CompiledGraph; wrap it in a local `graphAsAgent` adapter (~15 LOC) that implements `agents.Agent`; register as a worker; assert Supervisor dispatches to it and the graph's output appears in `BuildAggregate`. (c) `TestSupervisor_OfSupervisor` — outer Supervisor with inner Supervisor as a worker; assert composition works AND each Supervisor's `MaxRounds` is independent AND a shared `Budget.MaxCalls` caps across both. (d) `TestSupervisor_AsFanOutFanInPlanner` — verify Supervisor (which is an `agents.Agent`) can be the `Planner` of a `FanOutFanIn`; expected to work because both interfaces are `agents.Agent`. Example: `examples/08-supervisor/main.go` — three demos mirroring `06-budget/main.go` style: (i) `demoBasic` (1 planner + 2 workers — researcher + summarizer — 2 rounds), (ii) `demoBudget` (Supervisor under Budget{MaxCalls: 3} → ErrCallsExceeded), (iii) `demoComposeWithStateGraph` (Supervisor inside outer StateGraph). README ≤80 lines: canonical setup, MaxRounds vs Budget.MaxCalls distinction (Decision E), composition stack note. `main_test.go`: smoke test that runs main() and asserts exit code 0 (mirror `examples/06-budget/main_test.go`). Exit gate: `cd examples && GOWORK=off go run ./08-supervisor` exits 0; `go vet ./examples/08-supervisor/...`. |
| **37-04** | 4 | execute | `llm-agent` | (verify-only — no Go source mod) | CC-3 (exit gate) | Phase exit gate, run from `llm-agent/`. (a) `go vet ./... && go test ./... -count=1 && go test -race ./orchestrate/... -count=1 && go test -race ./...` — all green. (b) `go list -deps ./orchestrate/` shows only stdlib + the package's own existing imports (no new third-party module). (c) `git diff main -- go.mod go.sum` is empty (no new require; no `go.sum` created). (d) `git diff main -- agent.go memory/ orchestrate/graph.go llm/chatmodel.go llm/stream.go llm/types.go llm/capabilities.go llm/errors.go` shows NO edits to validated public types — KC-5 verification (`orchestrate/doc.go` is allowed: a docstring-only addition listing Supervisor). (e) `git diff main -- simple.go react.go plan_solve.go reflection.go function_call.go agent_chatmodel.go agents.go` — zero edits to paradigm files (Supervisor doesn't touch them). (f) `git diff main -- budget/ policy/` — zero edits (Phase 37 doesn't touch Phases 35/36). (g) `go list -m all` — module set unchanged (only `github.com/costa92/llm-agent-rag v1.0.1`). (h) Tag and push `v0.6.2` from `main` (operator action — slice records the command but the operator runs it). |

Wave structure is strictly sequential. 37-01 → 37-02 is skeleton-then-
integration (budget/policy tests need the skeleton). 37-02 → 37-03 is
integration-then-compose-and-example (example uses the verified
compose patterns). 37-03 → 37-04 is example-then-exit-gate. No
parallelization gain available within the phase.

**Sizing.** ~400 LOC supervisor.go + state machine + types (37-01) +
~200 LOC primary tests (37-01) + ~150 LOC budget/policy integration
(37-02) + ~150 LOC compose tests (37-03) + ~200 LOC example (37-03) +
0 LOC exit gate (37-04). Total ~1000-1300 LOC. Matches the v1.2
SUMMARY's "M" estimate (~600 LOC for the Supervisor; tests + example
add the rest).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `StateGraph[S]` is sufficient as the loop substrate (no need for a parallel implementation) | Pattern 1, KC-1 | LOW — verified by direct read of `orchestrate/graph.go`; the builder + Compile + Run pattern supports the 3-node state machine cleanly |
| A2 | The 3-node state machine (`[plan]`/`[dispatch]`/`[final]`) is the minimal correct shape (vs. 2-node or 4-node alternatives) | Pattern 2 | LOW — fewer nodes can't express the conditional routing; more nodes are gratuitous |
| A3 | `MaxRounds * 3 + 4` slack for `WithMaxSteps` is sufficient | Pitfall 2 | LOW — derived from the node count per round; planner ratifies the magic number in 37-01 |
| A4 | `MaxRounds` hit is a graceful BuildAggregate-over-results-so-far (vs. an error) | Decision F + Pattern 2 | MEDIUM — operator preference call; planner can choose "error" instead by routing to ErrSupervisorMaxRounds in the conditional edge. Test `TestSupervisor_MaxRoundsExceeded` documents the chosen semantics. |
| A5 | Workers receive ctx unchanged (no detached children) | Pitfall 4 + 35-RESEARCH carry-forward | LOW — verified by reading 35-RESEARCH.md's explicit carry-forward note |
| A6 | Supervisor satisfies `agents.Agent` so it can be a worker of another Supervisor | KC-1, Standard Stack | LOW — KC-1 explicit; the compile-time assertion `var _ agents.Agent = (*Supervisor)(nil)` is the proof |
| A7 | Supervisor doesn't charge its own per-round budget; chokepoint suffices | Decision E | LOW — verified by reading the shipped chokepoint at `agent_chatmodel.go` |
| A8 | `BuildAggregate` is called ONCE at `[final]` (not per round) | Decision F | LOW — single-seam aggregation matches FanOutFanIn precedent |
| A9 | `Planner` is `agents.Agent` (not a stricter typed-return interface) | Decision D | LOW — symmetry with workers; structured outputs deferred to v1.3 |
| A10 | `RunStream` reimplements `runStreamFromBlocking` locally rather than exporting it from `agents/` | Decision G | MEDIUM — operator may prefer to widen `agents.` surface with an exported helper. Planner ratifies in 37-01. Both choices are KC-5-friendly (additive). [ASSUMED] |
| A11 | `ParseDispatch` returns `(nil, nil)` for finish vs `(nil, err)` for parse failure | Pitfall 3 | LOW — Go idiom for "two semantics on one return value, distinguished by err"; documented in package doc |
| A12 | Worker map lookup is case-sensitive | Pitfall 1 | LOW — `map[string]agents.Agent` direct lookup; `ParseDispatch` is the normalization seam (caller's responsibility) |
| A13 | The compose-with-StateGraph test uses an in-test `graphAsAgent` adapter (~15 LOC) | Example D | LOW — small, test-only, doesn't widen public surface |
| A14 | The example shape (1 planner + 2 workers + 2 rounds) is illustrative-enough | Decision H | LOW — minimal to demonstrate routing + iteration; mirror `06-budget`'s shape |
| A15 | No new `Step.Kind` or `llm.StreamEvent.Kind` is needed | Anti-Patterns | LOW — K1 + KC-5 locked; existing 6 StepKinds suffice |

## Open Questions

1. **Should `MaxRounds` hit be a graceful BuildAggregate or an
   error?**
   - What we know: A4 (recommended graceful, mirrors
     `RoundRobinChat.Stopped="max_turns"`).
   - What's unclear: Some operators prefer hard error to force the user
     to bump MaxRounds.
   - Recommendation: **Graceful** (BuildAggregate over results-so-far)
     with the partial-answer surfaced. Planner ratifies in 37-01;
     either choice is documented in test name.

2. **Should `runStreamFromBlocking` be exported from `agents/` so
   Supervisor reuses it, OR reimplemented locally in
   `orchestrate/supervisor.go`?**
   - What we know: A10 (recommended reimplement locally).
   - What's unclear: Exporting widens `agents.` surface (additive,
     KC-5-friendly); local reimplementation duplicates 27 lines.
   - Recommendation: **Reimplement locally** in 37-01. Carry-forward
     note: if a 3rd paradigm wants the same helper, refactor to
     export in v1.3.

3. **Should `SupervisorOptions` have an `OnStep` field for caller-side
   observation (in addition to `RunStream`)?**
   - What we know: 5 paradigms have `OnStep`; Supervisor should be
     uniform.
   - What's unclear: Decision G recommended NO (callers observe via
     `RunStream`); but an `OnStep` field is strictly additive and
     uniform with paradigms.
   - Recommendation: **Skip in 37-01; add in 37-02 if needed.** If the
     budget/policy tests don't need it, defer to a follow-up.
     KC-5-friendly either way.

4. **Should `Dispatch.Metadata` be `map[string]any` or omitted
   entirely?**
   - What we know: Decision B recommended include (future seam).
   - What's unclear: Including it now without a built-in consumer is
     YAGNI; planner could drop it.
   - Recommendation: **Include** — its presence enables
     planner-to-worker hints without a future package-version bump.
     Documented as "transparent to Supervisor; planner-defined
     contract". Zero-LOC cost.

5. **Should there be a `SupervisorAgent` alias for `Supervisor` so
   import sites read more naturally (`orchestrate.SupervisorAgent`
   vs `orchestrate.Supervisor`)?**
   - What we know: FanOutFanIn doesn't have an Agent suffix
     (`orchestrate.FanOutFanIn`). Symmetry.
   - What's unclear: Some readers may expect `Agent` in the name
     since Supervisor IS an `agents.Agent`.
   - Recommendation: **No alias.** Stick with `Supervisor`. The
     compile-time `var _ agents.Agent = (*Supervisor)(nil)` is the
     definitive answer to "is it an agent?". Documented in package
     doc.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `go` toolchain | All slices | ✓ (assumed) | go 1.26.0 from `go.mod` | — |
| stdlib `context` | all | ✓ | stdlib | — |
| stdlib `errors` | all | ✓ | stdlib | — |
| stdlib `fmt` | all | ✓ | stdlib | — |
| stdlib `strings` | trace assembly + prompt building | ✓ | stdlib | — |
| stdlib `testing` | tests | ✓ | stdlib | — |
| `git` | exit gate (`git diff`) | ✓ | system | — |
| `llm-agent-rag` (back-edge for RAG facade) | NOT TOUCHED | n/a — KS-5 freeze | v1.0.1 (tagged) | — |
| `ScriptedLLM` (in `llm/`) | tests + example | ✓ — verified | v0.6.1 | — |
| `examples/scriptedllm` adapter | example | ✓ — verified by listing `examples/` | v0.6.1 (Phase 35 shipped) | — |
| `budget` package | 37-02 integration tests | ✓ — Phase 35 shipped 2026-05-21 (v0.6.0) | v0.6.0+ | — |
| `policy` package | 37-02 integration tests | ✓ — Phase 36 shipped 2026-05-21 (v0.6.1) | v0.6.1+ | — |
| `llm-agent-otel` (sister) | NOT REQUIRED | n/a — Decision G of Phase 36; same approach holds for Phase 37 if needed | — | none needed; no compose-with-otel test in Phase 37 |
| `agentstest` (sub-package shipped in v0.6.0) | optional | ✓ — verified by listing `agentstest/` | v0.6.0 | not strictly needed; Supervisor tests use ScriptedLLM directly |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** none — all required dependencies are shipped.

## Sources

### Primary (HIGH confidence — direct file reads, this session)

- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/orchestrate/graph.go` — `StateGraph[S]` API, `NodeFunc[S]`, `ConditionFunc[S]`, Compile, Run, `defaultMaxSteps`, `ErrGraphMaxSteps`, `WithMaxSteps`, `NodeStart`/`NodeEnd` constants. The substrate Supervisor reuses.
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/orchestrate/fanout.go` — `FanOutFanIn` precedent (`PlannedTask`, `TaskResult`, `PlanParser`, `AggregateInputBuilder`, sentinel set). The cognate Supervisor distinguishes from.
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/orchestrate/pipeline.go` — `Pipeline` shape (`Step{Name, Agent, Adapt}`, `PipelineResult`).
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/orchestrate/roleplay.go` — `RolePlay` shape (CAMEL convention, `RolePlayResult`, `DoneMarker`).
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/orchestrate/roundrobin.go` — `RoundRobinChat` shape; the graceful `Stopped="max_turns"` precedent for Supervisor's MaxRounds.
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/orchestrate/termination.go` — `Termination`, `MaxTurns`, `TextMatch`, `And`, `Or` (not used by Supervisor but contextual).
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/orchestrate/doc.go` — paradigm-list package doc, the docstring Supervisor entry will edit additively.
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/agent.go` — `agents.Agent` interface (Name/Run/RunStream), `Result{Answer, Trace, Usage}`, `Step{Kind, Content, Tool, Args, Result}`, `StepKind` (6 values), `StepEvent`, `runStreamFromBlocking` (package-private helper), `normalizedOnStep`, sentinels (`ErrMaxStepsExceeded`, `ErrEmptyInput`, etc.).
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/agent_chatmodel.go` — the `generateFromPrompt` chokepoint shipped in Phase 35-02 (budget enforcement seam).
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/simple.go` — SimpleAgent (the planner / worker prototype for the example).
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/react.go` — ReActAgent (for trace shape inspiration and chokepoint call sites).
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/plan_solve.go` — PlanAndSolveAgent (alternate paradigm; relevant for "planner is an agents.Agent" — Decision D justification).
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/reflection.go` — ReflectionAgent (more chokepoint call sites).
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/function_call.go` — FunctionCallAgent (native tools; chokepoint at line 81).
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/budget/budget.go` — `Budget`, `Tracker`, `WithBudget`, `From`, `NewStrict`, `NewSoft`, sentinel family (`ErrBudgetExceeded`, `ErrCallsExceeded`, etc.). Phase 35 shipped surface that Phase 37 inherits.
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/policy/policy.go` and `policy/gate.go` — `Wrap`, `WrapConfig`, `Config{Gates, OnDecision}`, `Gate`, `Event`, `EventKind`, `Decision`, `DecisionAction`, `ErrBlocked`, `BlockedError`. Phase 36 shipped surface that Phase 37 inherits.
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/examples/06-budget/main.go` — example shape template (deterministic ScriptedLLM, 3 demo subroutines).
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/orchestrate/fanout_test.go` (partial) — test pattern templates (stubAgent, table-driven tests).
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/orchestrate/graph_test.go` (partial) — StateGraph test patterns.
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/.planning/phases/35-budget-and-cancellation-context/35-RESEARCH.md` — sibling phase research; the "workers receive supervisorCtx unchanged" carry-forward rule for budget propagation.
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/.planning/phases/36-policy-safety-middleware/36-RESEARCH.md` — sibling phase research; composition stack `policy.Wrap(otelmodel.Wrap(provider))`; Decision G in-test mimicked observer pattern.
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/.planning/phases/36-policy-safety-middleware/36-01-PLAN.md` and `36-01-SUMMARY.md` — slice template + decision-ratification voice for Phase 37 plans.
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/.planning/v1.2-REQUIREMENTS.md` — CC-3 verbatim.
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/.planning/v1.2-ROADMAP.md` — Phase 37 planned work (37-01..04 outline).
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/.planning/research/v1.2-core-capability-deepening-SUMMARY.md` — KC-1 keystone (the locked design: Supervisor as StateGraph[S] facade in orchestrate/).
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/CLAUDE.md` — hard rules.
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/go.mod` — verified single require directive, stdlib-only with rag back-edge.

### Secondary (MEDIUM confidence)

- ASVS category mapping — extrapolated from common library responsibilities; same approach as Phase 36's mapping. Library code is mostly outside V2/V3/V4 scope.
- Magic number `MaxRounds * 3 + 4` (slack) — derived from node count; planner ratifies in 37-01 (Pitfall 2).
- Sizing estimate (~1000-1300 LOC) — extrapolated from Phase 35 (~800 LOC) and Phase 36 (~1100-1400 LOC); planner refines.

### Tertiary (LOW confidence — none, flagged for validation)

- None. Every claim above is grounded in direct file reads or the Phase 35/36 prior research artifacts.

## Metadata

**Confidence breakdown:**

- Standard stack (Supervisor + SupervisorOptions + Dispatch + WorkerResult): HIGH — KC-1 names the fields verbatim; the shapes are direct cognates of FanOutFanIn's verified pattern.
- Architecture (3-node StateGraph[supervisorState] facade): HIGH — verified by direct read of `orchestrate/graph.go`; the 3-node pattern is the minimal correct shape.
- KC-1 facade vs separate state machine: HIGH — the v1.2 SUMMARY MEDIUM-confidence note ("first-slice prototype will refine whether StateGraph is the right substrate") is resolved by this research: yes, StateGraph IS sufficient. The 3-node state machine fits cleanly; cancellation + MaxSteps are inherited correctly. (See Decision A + Pattern 2.)
- Budget integration (CC-1 propagation): HIGH — verified by direct read of the shipped chokepoint at `agent_chatmodel.go:11-54`; workers receive ctx unchanged → tracker propagates → per-LLM-call charges fire automatically.
- Policy integration (CC-2 per-worker): HIGH — KC-3 + Phase 36's per-worker decorator pattern; Supervisor is policy-agnostic by design.
- Cancellation: HIGH — inherited from StateGraph's per-step `select { case <-ctx.Done() }` (`graph.go:170-174`).
- Slice breakdown (4 slices): HIGH — mirrors ROADMAP §"Planned work"; sequencing is dependency-forced.
- Example shape (1 planner + 2 workers + 2 rounds): MEDIUM — operator could prefer 3 workers / 3 rounds; the recommendation is minimum-illustrative. Planner ratifies in 37-03.
- Open Questions 1-5: MEDIUM — operator ratification needed (graceful MaxRounds vs error; export `runStreamFromBlocking` vs reimplement; OnStep field; Metadata map; SupervisorAgent alias). All choices are KC-5-friendly.

**Research date:** 2026-05-21
**Valid until:** 2026-06-20 (30 days — stable domain; the only mover
is the operator's ratification of A4, A10, and Open Questions 1-5 in
37-01)

---

*Researched 2026-05-21 by the Phase 37 research spawn. Voice + structure
mirror `.planning/phases/35-budget-and-cancellation-context/35-RESEARCH.md`
and `.planning/phases/36-policy-safety-middleware/36-RESEARCH.md`.
Every file path cited verified by direct read against the working tree
at `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem`.
The design mirrors `orchestrate.FanOutFanIn`'s sibling pattern but
distinguishes via iterative re-planning per KC-1; the substrate is
`orchestrate.StateGraph[supervisorState]` with three nodes (`[plan]` /
`[dispatch]` / `[final]`) and a conditional edge on `[plan]` for
dispatch-vs-finish routing. Budget + policy integration is INHERITED
(zero new code in Supervisor) via ctx propagation through the
generateFromPrompt chokepoint shipped in Phase 35-02 and the per-worker
`policy.Wrap` decorator shipped in Phase 36.*

## RESEARCH COMPLETE

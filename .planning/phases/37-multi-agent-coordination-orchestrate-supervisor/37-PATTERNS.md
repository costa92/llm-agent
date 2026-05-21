# Phase 37: Multi-agent coordination (`orchestrate.Supervisor`) — Pattern Map

**Mapped:** 2026-05-21
**Files analyzed:** 6 new / 1 edited (additive doc-comment only)
**Analogs found:** 7 / 7 (all have at least role-match; 5/7 exact)

This phase ships a strict-additive `orchestrate.Supervisor` value type as a
typed `StateGraph[supervisorState]` facade (KC-1). Every planned file has
a close in-tree analog. The planner uses the per-file "Read first" lists
below as the executor's primary source-of-truth: imports, sentinel
shape, builder chains, ctx-propagation idioms, and test layout are all
already established by the v0.6.0/v0.6.1 ships and the locked
`orchestrate/` precedent.

## File Classification

| New / Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `orchestrate/supervisor.go` | orchestrator (typed state-machine facade satisfying `agents.Agent`) | iterative loop / re-planning | `orchestrate/roundrobin.go` (iterative coordinator) + `orchestrate/graph.go` (the substrate) + `orchestrate/fanout.go` (vocabulary cognate) | **exact (composite)** — primary shape from `roundrobin.go`; substrate calls direct from `graph.go`; sentinel + options shape mirrors `fanout.go` |
| `orchestrate/supervisor_test.go` | test (orchestrator unit + behavior) | request-response (table + ctx-cancel) | `orchestrate/roundrobin_test.go` + `orchestrate/fanout_test.go` (stubAgent) + `orchestrate/graph_test.go` (conditional-edge + maxsteps + ctx tests) | **exact** |
| `orchestrate/supervisor_budget_test.go` (or merged into `_test.go`) | integration test (budget + policy compose with Supervisor) | request-response w/ ctx-bound tracker | `agent_chatmodel_test.go` (chokepoint deny tests) + `policy/integration_test.go::TestCompose_BudgetBeatsPolicyAtChokepoint` (compose stack) | **exact** |
| `orchestrate/supervisor_compose_test.go` | integration test (compose-with-StateGraph both directions) | request-response (state-machine inside state-machine) | `orchestrate/graph_test.go::TestStateGraph_*` + `37-RESEARCH.md` Example D | **role-match** (no prior compose test exists; pattern is novel-but-mechanical) |
| `orchestrate/doc.go` (additive paragraph) | doc-comment edit | n/a | itself (`orchestrate/doc.go`) — extend the paradigm list | **exact** |
| `examples/08-supervisor/main.go` | example program (deterministic, 3 demos) | request-response with ScriptedLLM | `examples/06-budget/main.go` + `examples/07-policy/main.go` | **exact** |
| `examples/08-supervisor/main_test.go` | example smoke test (captures stdout, runs main) | test harness | `examples/06-budget/main_test.go` | **exact** |
| `examples/08-supervisor/README.md` | example readme (canonical setup) | docs | `examples/07-policy/README.md` (just shipped) | **exact** |

---

## Pattern Assignments

### `orchestrate/supervisor.go` (orchestrator, iterative loop)

**Primary analog:** `orchestrate/roundrobin.go` — the existing
iterative multi-agent coordinator. Same package, same `agents.Agent`
worker pattern, similar `name + opts` value shape, same
"per-turn-call-an-agent + accumulate Usage" loop. Differs only in:
**Supervisor uses `StateGraph[S]`** instead of a hand-rolled `for turn
:= ...` loop (KC-1 — Supervisor inherits cancellation + MaxSteps from
the substrate), and **dispatches one named worker per round** instead
of rotating a fixed slice.

**Substrate analog:** `orchestrate/graph.go` — the `StateGraph[S]`
builder + `Compile` + `Run(ctx, initial, opts...)` pattern Supervisor
threads three nodes through.

**Vocabulary cognate:** `orchestrate/fanout.go` — distinct shape
(plan-once vs. iterative) but the sentinel family, options struct, and
parser/aggregator function-type pattern come from here.

**Read-first list for executor:**
- `orchestrate/roundrobin.go` (entire 111 LOC — the smallest complete
  coordinator analog)
- `orchestrate/graph.go` (the substrate; especially lines 36-43 builder,
  88-131 Compile, 140-208 Run + MaxSteps)
- `orchestrate/fanout.go` (lines 14-44 vocabulary; 67-77 NewX/Name;
  184-211 worker map lookup; 246-260 sentinel family)
- `agent.go` (lines 12-21 `agents.Agent` interface; 28-33 `StepEvent`;
  58-76 `Step` + `StepKind`; 98-125 `runStreamFromBlocking` — copy
  this 27-line helper into supervisor.go per Decision G option (b))
- `agent_chatmodel.go` (entire 71 LOC — proves ctx → budget
  propagation; Supervisor adds **zero** budget code, just passes ctx
  unchanged)

**Imports pattern** (copy verbatim from `orchestrate/roundrobin.go:1-10`):

```go
package orchestrate

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/costa92/llm-agent"
)
```

Notes: stdlib + the root `agents` package (named `agents` via the module
path's package directive at `agent.go:5`). **No** `policy`/`budget`
import in Supervisor itself — those propagate via ctx through the
chokepoint and the worker's wrapped model. **No** `sync` import in the
core file (Supervisor is per-Run, no shared state; test files may
import `sync/atomic` for counting agents like `roundrobin_test.go:7`).

**Constructor + Name pattern** (copy from `orchestrate/roundrobin.go:44-57` +
`orchestrate/fanout.go:67-77`):

```go
// NewSupervisor constructs a Supervisor.
func NewSupervisor(name string, opts SupervisorOptions) *Supervisor {
	return &Supervisor{name: name, opts: opts}
}

// Name returns the supervisor name used in errors/logs.
func (s *Supervisor) Name() string {
	if s.name == "" {
		return "supervisor"
	}
	return s.name
}
```

**Compile-time `agents.Agent` assertion** (copy from `agent.go` shape,
explicit in 37-RESEARCH Example C):

```go
// Compile-time assertion: *Supervisor satisfies agents.Agent. This is
// the load-bearing property — KC-1 says a Supervisor may be a worker
// of another Supervisor (composition).
var _ agents.Agent = (*Supervisor)(nil)
```

**Sentinel family** (copy structure from `orchestrate/fanout.go:253-260`):

```go
var (
	ErrSupervisorNilPlanner       = errors.New("orchestrate: supervisor requires non-nil planner")
	ErrSupervisorNoWorkers        = errors.New("orchestrate: supervisor requires at least one worker")
	ErrSupervisorNilParseDispatch = errors.New("orchestrate: supervisor requires non-nil ParseDispatch")
	ErrSupervisorNilBuildAggregate = errors.New("orchestrate: supervisor requires non-nil BuildAggregate")
	ErrSupervisorUnknownWorker    = errors.New("orchestrate: supervisor dispatch references unknown worker")
	ErrSupervisorMaxRounds        = errors.New("orchestrate: supervisor max rounds exceeded")
	ErrSupervisorParseDispatch    = errors.New("orchestrate: supervisor parse dispatch failed")
)
```

**StateGraph builder pattern** (copy chain shape from
`orchestrate/graph_test.go:44-66` and `orchestrate/graph.go:36-131`):

```go
// compileGraph builds the 3-node typed StateGraph[supervisorState].
// Per KC-1, this IS the loop substrate — no hand-rolled `for round`.
func (s *Supervisor) compileGraph() (*CompiledGraph[supervisorState], error) {
	g := NewStateGraph[supervisorState]()
	g.AddNode("plan", s.planNode)
	g.AddNode("dispatch", s.dispatchNode)
	g.AddNode("final", s.finalNode)
	g.SetEntry("plan").
		AddConditionalEdge("plan", s.routeFromPlan).
		AddEdge("dispatch", "plan").
		AddEdge("final", NodeEnd)
	return g.Compile()
}
```

Note: `AddConditionalEdge("plan", ...)` is the load-bearing routing
decision. `orchestrate/graph_test.go:44-79` and the public examples in
`graph_test.go:266-871` (ExampleStateGraph_loop, _supportEscalation,
_warrantyEscalation, _humanReviewReseek) are the canonical conditional-
edge precedent — copy idiom from there.

**Worker name lookup** (copy from `orchestrate/fanout.go:184-191`):

```go
worker, ok := s.opts.Workers[d.WorkerName]
if !ok || worker == nil {
	return st, fmt.Errorf("%w: %q (round %d)", ErrSupervisorUnknownWorker, d.WorkerName, st.round)
}
```

**Usage accumulation pattern** (copy idiom from
`orchestrate/fanout.go:246-251` + `orchestrate/roundrobin.go:82-83`):

```go
// addUsage from fanout.go is already exported in-package and reusable.
out.Usage = addUsage(out.Usage, plannerUsage)
out.Usage = addUsage(out.Usage, workerUsage)
```

**Error wrapping pattern** (copy idiom from `orchestrate/roundrobin.go:79`
and `orchestrate/fanout.go:95, 106, 123, 137, 149`):

```go
// per-node error wrap: include supervisor name + round number + node identifier
return st, fmt.Errorf("supervisor %q: planner round %d: %w", s.name, st.round, err)
return st, fmt.Errorf("supervisor %q: worker %q round %d: %w", s.name, d.WorkerName, st.round, err)
return st, fmt.Errorf("supervisor %q: aggregate: %w", s.name, err)
```

**MaxSteps cap pattern** (copy + comment from `orchestrate/graph.go:148, 208`
and Pitfall 2 of 37-RESEARCH):

```go
// MaxRounds × 3 nodes per round + 4 slack (entry [plan], final [plan]
// that routes to [final], the [final] node itself, +1). The substrate's
// defaultMaxSteps=100 (graph.go:148) is too low when MaxRounds>32.
maxSteps := s.opts.MaxRounds*3 + 4
final, err := cg.Run(ctx, supervisorState{input: input}, WithMaxSteps(maxSteps))
```

**Error translation** (the Supervisor surfaces `ErrSupervisorMaxRounds`
NOT `ErrGraphMaxSteps`; see Pitfall 2):

```go
if errors.Is(err, ErrGraphMaxSteps) {
	return agents.Result{}, fmt.Errorf("%w: rounds=%d", ErrSupervisorMaxRounds, s.opts.MaxRounds)
}
```

**RunStream pattern** (copy 27-LOC helper from `agent.go:98-125` into
supervisor.go per Decision G option (b); rename to package-private
`runStreamFromBlocking` already taken in package `agents` — use
`supervisorRunStream` or inline):

```go
// supervisor.go RunStream — reimplements the 27-line helper from
// agent.go:98-125 locally to avoid widening the agents.* exported surface.
// Per Decision G recommendation (b). The shape is identical: spawn a
// goroutine that calls runFn (which calls onStep), pipe StepEvents into
// a buffered channel, emit a Done event on exit.
func (s *Supervisor) RunStream(ctx context.Context, input string) (<-chan agents.StepEvent, error) {
	ch := make(chan agents.StepEvent, 16)
	go func() {
		defer close(ch)
		cb := func(st agents.Step) {
			select {
			case ch <- agents.StepEvent{Step: st}:
			case <-ctx.Done():
			}
		}
		res, err := s.runInternal(ctx, input, cb)
		if err != nil {
			select {
			case ch <- agents.StepEvent{Done: true, Err: err}:
			case <-ctx.Done():
			}
			return
		}
		select {
		case ch <- agents.StepEvent{Done: true, Final: &res}:
		case <-ctx.Done():
		}
	}()
	return ch, nil
}
```

---

### `orchestrate/supervisor_test.go` (test, request-response)

**Primary analog:** `orchestrate/roundrobin_test.go` — the closest
shape. Same package; uses a `countingAgent` stub that implements
`agents.Agent` (Name/Run/RunStream); table-driven scenarios; uses
`atomic.Int32` + `atomic.Value` for race-safe counters; tests
ctx-cancel via pre-cancelled ctx; tests `MaxTurns` graceful stop.

**Vocabulary analog:** `orchestrate/fanout_test.go` — provides the
`stubAgent` test-fixture pattern (`pipeline_test.go:13-45`, available
in-package) that's used across all 5 orchestrate test files.

**State-machine test analog:** `orchestrate/graph_test.go` — the
canonical pattern for `AddConditionalEdge`-driven tests (lines 44-79
TestStateGraph_ConditionalBranching, 102-132
TestStateGraph_ConditionalEdgeTakesPrecedence, 162-172
TestStateGraph_MaxStepsBreaksInfiniteLoop, 252-264
TestStateGraph_ContextCancel).

**Read-first list for executor:**
- `orchestrate/roundrobin_test.go` (entire 137 LOC — the closest test
  shape)
- `orchestrate/pipeline_test.go:13-45` (the in-package `stubAgent`
  struct — reuse directly)
- `orchestrate/fanout_test.go:14-32, 215-237` (parser fixture pattern;
  validation table pattern)
- `orchestrate/graph_test.go:44-79, 102-132, 162-172, 252-264`
  (conditional-edge + MaxSteps + ctx-cancel test pattern)

**stubAgent reuse** (`orchestrate/pipeline_test.go:13-45` — already in-package, no re-declare):

```go
// stubAgent is a minimal agents.Agent test double: returns a fixed
// answer prefixed with the input it received (so tests can assert
// the input was threaded through correctly).
type stubAgent struct {
	name      string
	transform func(input string) string
	err       error
	llmCalls  int
}

func (a *stubAgent) Name() string { return a.name }

func (a *stubAgent) Run(_ context.Context, input string) (agents.Result, error) {
	if a.err != nil {
		return agents.Result{}, a.err
	}
	out := input
	if a.transform != nil {
		out = a.transform(input)
	}
	calls := a.llmCalls
	if calls == 0 {
		calls = 1
	}
	return agents.Result{
		Answer: out,
		Usage:  agents.Usage{LLMCalls: calls, Tokens: len(out)},
	}, nil
}

func (a *stubAgent) RunStream(_ context.Context, _ string) (<-chan agents.StepEvent, error) {
	return nil, errors.New("stubAgent: stream not implemented")
}
```

**Happy-path test pattern** (copy structure from
`orchestrate/fanout_test.go:34-69` and `orchestrate/roundrobin_test.go:31-55`):

```go
func TestSupervisor_HappyPath(t *testing.T) {
	// 2 planner rounds + 1 finish round; 2 specialists
	plannerCalls := 0
	planner := &stubAgent{name: "planner", transform: func(_ string) string {
		plannerCalls++
		switch plannerCalls {
		case 1: return "dispatch to researcher: find facts about X"
		case 2: return "dispatch to summarizer: condense the facts"
		default: return "FINISH"
		}
	}}
	researcher := &stubAgent{name: "researcher", transform: func(s string) string { return "Facts: " + s }}
	summarizer := &stubAgent{name: "summarizer", transform: func(s string) string { return "Summary: " + s }}

	sup := NewSupervisor("research", SupervisorOptions{
		Planner:        planner,
		Workers:        map[string]agents.Agent{"researcher": researcher, "summarizer": summarizer},
		MaxRounds:      5,
		ParseDispatch:  parseDemoDispatch,
		BuildAggregate: joinWorkerResults,
	})

	res, err := sup.Run(context.Background(), "investigate X")
	if err != nil { t.Fatalf("Run: %v", err) }
	if !strings.Contains(res.Answer, "Facts") || !strings.Contains(res.Answer, "Summary") {
		t.Errorf("Answer missing worker outputs: %q", res.Answer)
	}
	if res.Usage.LLMCalls != 5 { t.Errorf("LLMCalls = %d, want 5 (3 planner + 2 worker)", res.Usage.LLMCalls) }
}
```

**Validation table pattern** (copy from
`orchestrate/fanout_test.go:215-237`):

```go
func TestSupervisor_Validation(t *testing.T) {
	good := &stubAgent{name: "g"}
	cases := []struct {
		name string
		opts SupervisorOptions
		want error
	}{
		{"nil planner", SupervisorOptions{Workers: map[string]agents.Agent{"w": good}, ParseDispatch: p, BuildAggregate: a, MaxRounds: 3}, ErrSupervisorNilPlanner},
		{"empty workers", SupervisorOptions{Planner: good, Workers: map[string]agents.Agent{}, ParseDispatch: p, BuildAggregate: a, MaxRounds: 3}, ErrSupervisorNoWorkers},
		{"nil ParseDispatch", SupervisorOptions{Planner: good, Workers: map[string]agents.Agent{"w": good}, BuildAggregate: a, MaxRounds: 3}, ErrSupervisorNilParseDispatch},
		{"nil BuildAggregate", SupervisorOptions{Planner: good, Workers: map[string]agents.Agent{"w": good}, ParseDispatch: p, MaxRounds: 3}, ErrSupervisorNilBuildAggregate},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sup := NewSupervisor("x", tc.opts)
			_, err := sup.Run(context.Background(), "input")
			if !errors.Is(err, tc.want) {
				t.Errorf("err = %v, want errors.Is(..., %v)", err, tc.want)
			}
		})
	}
}
```

**Ctx-cancel test pattern** (copy from `orchestrate/roundrobin_test.go:115-128`
and `orchestrate/graph_test.go:252-264`):

```go
func TestSupervisor_CtxCancel(t *testing.T) {
	sup := NewSupervisor("x", validOpts())
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel
	_, err := sup.Run(ctx, "input")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}
```

**MaxRounds graceful-terminate pattern** (combines
`roundrobin.go:90` Stopped="max_turns" precedent with the
ErrSupervisorMaxRounds wrap from Pitfall 2 of 37-RESEARCH):

```go
func TestSupervisor_MaxRoundsExceeded(t *testing.T) {
	// Planner emits 3 dispatches but MaxRounds=2 → after round 2,
	// the conditional edge routes to [final] (graceful BuildAggregate
	// over 2 results — NOT an ErrGraphMaxSteps).
	// ... assert errors.Is(err, ErrSupervisorMaxRounds) is FALSE on
	// graceful path; assert res.Answer aggregates the 2 collected results.
}
```

---

### `orchestrate/supervisor_budget_test.go` (integration, ctx-bound budget + policy)

**Primary analog:** `agent_chatmodel_test.go` — proves the chokepoint
charges `budget.From(ctx)` pre-call AND post-call. The Supervisor
inherits this verbatim by passing ctx unchanged to workers. The
analog tests `TestGenerateFromPrompt_MaxCalls_PreCallDeny` (lines
101-129) and `TestGenerateFromPrompt_MaxTokens_PostCallDeny_ReturnsBoth`
(lines 145-197) are the shape Supervisor's budget integration test
mirrors at a higher level.

**Compose-stack analog:** `policy/integration_test.go::TestCompose_BudgetBeatsPolicyAtChokepoint`
(lines 425-469) — the canonical pattern for "budget at chokepoint,
policy at decorator, budget fires first because the chokepoint sits
underneath the wrapper stack". The Supervisor's policy-per-worker
test wires the same shape but with the wrapped model attached to a
worker, not directly to a SimpleAgent.

**Read-first list for executor:**
- `agent_chatmodel_test.go` (entire 278 LOC — chokepoint test
  patterns; the file the Supervisor's behavior is downstream of)
- `agent_chatmodel.go` (entire 71 LOC — the chokepoint itself; proves
  Supervisor adds **zero** budget code)
- `policy/integration_test.go:425-469` (compose-stack test pattern;
  budget-beats-policy invariant)
- `policy/integration_test.go:41-145` (in-test observer model pattern
  — Decision G of Phase 36; reuse for the Supervisor's policy test if
  an observer is needed)
- `budget/doc.go:1-46` (Q1/Q2 carry-forward: MaxCalls counts attempts;
  the three Usage types stay distinct)

**Budget propagation test pattern** (copy from
`agent_chatmodel_test.go:101-129` shape; adapt to Supervisor's nested
agent chain):

```go
// TestSupervisor_BudgetPropagatesToWorker proves CC-1's "rounds count
// against Budget.Calls" via INDIRECT counting: each planner call + each
// worker call charges the chokepoint. Supervisor itself adds no
// charges (Decision E).
func TestSupervisor_BudgetPropagatesToWorker(t *testing.T) {
	// MaxCalls=3 → 3 successful chokepoint charges then deny on 4th.
	// 2 planner rounds + 2 worker calls = 4 chokepoint charges → the
	// 4th surfaces ErrCallsExceeded from within agent.Run.
	ctx, tracker := budget.WithBudget(context.Background(), budget.Budget{MaxCalls: 3})
	// ... wire scriptedLLM-backed SimpleAgent for planner + worker
	// ... assert errors.Is(err, budget.ErrCallsExceeded)
	// ... assert errors.Is(err, budget.ErrBudgetExceeded)
	// ... assert tracker.Snapshot().Calls == 3 (cap)
}
```

**Policy-per-worker test pattern** (copy compose stack from
`policy/integration_test.go:425-469`):

```go
// TestSupervisor_PolicyPerWorker proves CC-2 carries through: a
// worker's underlying model wrapped in policy.Wrap fires gates as
// usual. Supervisor is policy-agnostic — it doesn't see the wrap.
func TestSupervisor_PolicyPerWorker(t *testing.T) {
	scripted := llm.NewScriptedLLM( /* multiple responses */ )
	blockingGate := &someGate{ pre: policy.Decision{Action: policy.Block, Reason: "blocked"} }
	wrapped := policy.Wrap(scripted, blockingGate)

	worker := agents.NewSimpleAgent(wrapped, agents.SimpleOptions{Name: "blocked-worker"})
	// ... wire planner that dispatches to "blocked-worker" twice;
	// ... assert second dispatch surfaces policy.ErrBlocked through
	// ...   Supervisor.Run unchanged.
	if !errors.Is(err, policy.ErrBlocked) { /* fail */ }
}
```

**Budget-beats-policy invariant** (verbatim from
`policy/integration_test.go:425-469`; assert holds at Supervisor level
too):

```go
// TestSupervisor_BudgetBeatsPolicy — chokepoint pre-charges Calls
// BEFORE the policy decorator runs; budget exhaustion surfaces as
// budget.ErrCallsExceeded NOT policy.ErrBlocked even when both
// would fire.
```

---

### `orchestrate/supervisor_compose_test.go` (integration, state-machine inside state-machine)

**Primary analog:** `orchestrate/graph_test.go` — the canonical
`NewStateGraph[S]() + AddNode + AddEdge` chain. The compose test
builds an outer `StateGraph[outerState]` whose `supervise` node calls
`sup.Run(ctx, ...)`. Same package, direct access to all symbols.

**No prior compose-direction test** exists in `orchestrate/` — this is
genuinely the first time a Supervisor-shaped type is composed with
the substrate. The pattern is mechanical: build a graph, add a node
that calls a Supervisor, assert end-to-end. The reverse direction
(StateGraph as a worker inside Supervisor) needs a small ~15-LOC
`graphAsAgent` adapter defined in-test.

**Read-first list for executor:**
- `orchestrate/graph_test.go:19-42, 266-305` (linear path +
  ExampleStateGraph_loop — the chain builder shape)
- `37-RESEARCH.md` §"Code Examples" Example D (the worked sketch for
  both compose directions)
- `orchestrate/fanout_test.go:34-69` (TestFanOutFanIn_Success — the
  precedent for "test that an orchestrator inside another container
  works")

**Compose-outer test pattern** (skeleton from Example D of 37-RESEARCH):

```go
func TestSupervisor_InsideStateGraph(t *testing.T) {
	sup := NewSupervisor("inner", /* valid opts */)

	type outerState struct{ input, supRes, final string }
	g := NewStateGraph[outerState]()
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
		AddEdge("postprocess", NodeEnd)
	cg, _ := g.Compile()
	final, err := cg.Run(context.Background(), outerState{input: "hi"})
	// assert err == nil, final.final has expected bracketed shape
}
```

**Compose-inner adapter** (~15 LOC, test-only — copy idiom from
`orchestrate/pipeline_test.go:13-45` stubAgent shape):

```go
// graphAsAgent wraps a *CompiledGraph[S] as an agents.Agent so it
// can be a Supervisor worker. Defined in-test, not exported.
type graphAsAgent struct {
	name string
	cg   *CompiledGraph[wState]
}
func (g *graphAsAgent) Name() string { return g.name }
func (g *graphAsAgent) Run(ctx context.Context, input string) (agents.Result, error) {
	final, err := g.cg.Run(ctx, wState{in: input})
	if err != nil { return agents.Result{}, err }
	return agents.Result{Answer: final.out, Usage: agents.Usage{LLMCalls: 1}}, nil
}
func (g *graphAsAgent) RunStream(_ context.Context, _ string) (<-chan agents.StepEvent, error) {
	return nil, errors.New("graphAsAgent: stream not implemented")
}
```

---

### `orchestrate/doc.go` (additive paragraph)

**Analog:** itself. The existing `orchestrate/doc.go:1-82` already
lists 5 paradigms (Pipeline, FanOutFanIn, RoundRobinChat, RolePlay,
StateGraph[S]); the additive edit adds a 6th bullet for Supervisor —
exactly the shape Phase 36's docstring-additive edit took for the
`policy` cross-reference (see `policy/doc.go:1-40` for Q1-Q5 ratification
voice).

**Read-first list for executor:**
- `orchestrate/doc.go` (entire 82 LOC — the file to edit)
- `policy/doc.go:12-40` (Q1-Q5 ratification voice — the pattern for
  decision-ratification doc comments)

**Edit pattern** (additive only — insert one bullet in the EN list and
one in the 中文 list; preserve existing line numbers as much as
possible):

```go
// In the EN list (after `StateGraph[S]` bullet on line 11):
//   - Supervisor      — iterative planner ↔ worker loop with re-planning (KC-1 v1.2)

// In the 中文 list (after `StateGraph[S]` bullet on line 18):
//   - Supervisor      — planner ↔ worker 迭代循环，支持重新规划（KC-1 v1.2）
```

And add a new "When to use" line in the §"Choosing a paradigm" section:

```go
//   - Planner that re-plans based on worker output → Supervisor
//   - planner 根据 worker 输出重新规划 → Supervisor
```

KC-5: every byte added is doc-comment only; no Go symbol added or
edited.

---

### `examples/08-supervisor/main.go` (example, deterministic)

**Primary analog:** `examples/06-budget/main.go` — the just-shipped
sibling. Same shape: package-level docstring, `func main()` calls 3
`demoX()` functions, helper `countingLLM` / `slowLLM` at bottom, prints
"OK" at end. Uses `examples/scriptedllm` + `examples/scriptedllm.New(...)`
for canonical deterministic mock per CLAUDE.md.

**Secondary analog:** `examples/07-policy/main.go` — even closer
sibling (also v0.6.x). Adds the `errors.As` BlockedError introspection
pattern for the budget-exhausted Supervisor demo's error printing.

**Read-first list for executor:**
- `examples/06-budget/main.go` (entire 216 LOC — closest template;
  3-demo structure)
- `examples/07-policy/main.go` (entire 200 LOC — sibling pattern;
  countingLLM helper)
- `examples/06-budget/main_test.go` (the smoke test pattern;
  pipe-stdout-and-assert)
- `examples/scriptedllm/` (canonical mock package; `scriptedllm.New(...)`
  + `scriptedllm.Text(...)`)

**Docstring header pattern** (copy from `examples/06-budget/main.go:1-22`
or `examples/07-policy/main.go:1-19`):

```go
// Demo 08: Multi-agent coordination via orchestrate.Supervisor.
//
// Wires a planner-emits-Dispatch → worker-runs → planner-observes →
// repeat loop on top of orchestrate.Supervisor (Phase 37, CC-3). The
// loop substrate is StateGraph[supervisorState] per KC-1; budget +
// policy compose underneath via the agent_chatmodel.generateFromPrompt
// chokepoint (Phases 35/36).
//
//   - demoBasic       : 1 planner + 2 specialists × 2 rounds → aggregate.
//   - demoBudget      : Supervisor under Budget{MaxCalls: 3} → the 4th
//                       chokepoint call surfaces ErrCallsExceeded mid-run.
//   - demoComposeWithStateGraph : Supervisor as a node in an outer
//                       StateGraph[outerState] — proves KC-1's "facade
//                       works both directions".
//
// The whole demo is deterministic — the canonical scriptedllm mock
// (per CLAUDE.md) returns pre-recorded responses, no network is touched.
//
// Run:
//
//	cd examples && go run ./08-supervisor
package main
```

**Imports pattern** (copy verbatim shape from
`examples/06-budget/main.go:24-35` + `examples/07-policy/main.go:21-33`):

```go
import (
	"context"
	"errors"
	"fmt"

	agents "github.com/costa92/llm-agent"
	"github.com/costa92/llm-agent/budget"
	"github.com/costa92/llm-agent/examples/scriptedllm"
	"github.com/costa92/llm-agent/llm"
	"github.com/costa92/llm-agent/orchestrate"
)
```

**Main function pattern** (copy from
`examples/06-budget/main.go:37-42`):

```go
func main() {
	demoBasic()
	fmt.Println()
	demoBudget()
	fmt.Println()
	demoComposeWithStateGraph()
	fmt.Println("OK")
}
```

**Demo function shape** (copy from `examples/06-budget/main.go:51-84`):

```go
func demoBasic() {
	fmt.Println("--- Basic: 1 planner + 2 specialists × 2 rounds ---")
	plannerLLM := scriptedllm.New(
		scriptedllm.Text("dispatch to researcher: find facts about X"),
		scriptedllm.Text("dispatch to summarizer: condense the facts"),
		scriptedllm.Text("FINISH"),
	)
	researcherLLM := scriptedllm.New(scriptedllm.Text("Facts: A, B, C."))
	summarizerLLM := scriptedllm.New(scriptedllm.Text("Summary: 3 facts."))
	planner := agents.NewSimpleAgent(plannerLLM, agents.SimpleOptions{Name: "planner"})
	researcher := agents.NewSimpleAgent(researcherLLM, agents.SimpleOptions{Name: "researcher"})
	summarizer := agents.NewSimpleAgent(summarizerLLM, agents.SimpleOptions{Name: "summarizer"})

	sup := orchestrate.NewSupervisor("demo", orchestrate.SupervisorOptions{
		Planner:        planner,
		Workers:        map[string]agents.Agent{"researcher": researcher, "summarizer": summarizer},
		MaxRounds:      5,
		ParseDispatch:  parseDemoDispatch,
		BuildAggregate: joinWorkerResults,
	})
	res, err := sup.Run(context.Background(), "investigate X")
	if err != nil { fmt.Printf("err: %v\n", err); return }
	fmt.Printf("answer: %s\n", res.Answer)
	fmt.Printf("usage : %+v\n", res.Usage)
}
```

**Budget-demo error introspection** (copy from
`examples/06-budget/main.go:65-80`):

```go
ctx, t := budget.WithBudget(context.Background(), budget.Budget{MaxCalls: 3})
// ... run Supervisor ...
fmt.Printf("errors.Is(err, budget.ErrCallsExceeded) = %v\n", errors.Is(err, budget.ErrCallsExceeded))
fmt.Printf("tracker snapshot: %+v\n", t.Snapshot())
```

---

### `examples/08-supervisor/main_test.go` (smoke test)

**Analog:** `examples/06-budget/main_test.go` (entire 67 LOC — copy
verbatim shape). Pipe-stdout-and-assert pattern; uses `mustContain(t,
out, ...fragments)` helper. The fragments asserted are the
deterministic markers the demos emit ("--- Basic:", "FINISH", "OK",
"budget.ErrCallsExceeded", etc.).

**Read-first:** `examples/06-budget/main_test.go` (the verbatim shape).

**Verbatim copy** (sub the markers):

```go
package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestExample_RunsToCompletion(t *testing.T) {
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil { t.Fatalf("os.Pipe: %v", err) }
	os.Stdout = w
	done := make(chan string, 1)
	go func() { b, _ := io.ReadAll(r); done <- string(b) }()
	main()
	_ = w.Close()
	os.Stdout = origStdout
	out := <-done
	mustContain(t, out,
		"--- Basic:",
		"answer:",
		"--- Budget",
		"errors.Is(err, budget.ErrCallsExceeded) = true",
		"--- Compose with StateGraph",
		"OK",
	)
}
func mustContain(t *testing.T, out string, fragments ...string) {
	t.Helper()
	for _, f := range fragments {
		if !strings.Contains(out, f) {
			t.Errorf("missing fragment %q.\nFull output:\n%s", f, out)
		}
	}
}
```

---

### `examples/08-supervisor/README.md` (canonical readme)

**Analog:** `examples/07-policy/README.md` (just shipped — the v0.6.x
voice) and `examples/06-budget/README.md` (the v0.6.0 sibling). Both
≤80 LOC; canonical structure: short intro, runtime invocation, one
section per demo, "composition stack" note at end.

**Read-first:**
- `examples/07-policy/README.md` (target voice + length)
- `examples/06-budget/README.md` (sibling demo's readme)

The readme is README-only — it documents but does not import any
sister-repo code (KC-3 carry-forward from Phase 36). Specifically
calls out:
- `MaxRounds` vs `Budget.MaxCalls` distinction (Decision E of
  37-RESEARCH).
- The composition stack `policy.Wrap(otelmodel.Wrap(provider))` on a
  worker's model fires per-worker without Supervisor knowing.
- The 4-decision graceful behavior at MaxRounds (BuildAggregate over
  results-so-far, not an error).

---

## Shared Patterns (apply across multiple new files)

### Pattern S1: ctx propagation (no detached children)

**Source:** `agent_chatmodel.go:11-54` (the chokepoint) +
`orchestrate/roundrobin.go:67-77` (the per-iteration ctx-cancel check
the substrate inherits) + 35-RESEARCH.md §"Carry-forward notes".

**Apply to:** `orchestrate/supervisor.go` (every node), `orchestrate/supervisor_budget_test.go`.

**Hard rule** (documented in package doc; tested in
`TestSupervisor_BudgetPropagatesToWorker`):

```go
// Workers receive the supervisor's ctx UNCHANGED. Do not use
// context.WithoutCancel, context.Background(), or any detached child.
// The budget tracker (Phase 35) and policy decorator state (Phase 36)
// propagate via ctx. This is the load-bearing invariant for CC-1/CC-2.
res, err := worker.Run(ctx, d.Input) // ← ctx unchanged from Supervisor.Run
```

### Pattern S2: agents.Usage rollup

**Source:** `orchestrate/fanout.go:246-251` + `roundrobin.go:82-83`.

**Apply to:** `orchestrate/supervisor.go` final usage assembly (the
`agents.Result.Usage` returned to caller must sum planner + all
workers).

```go
// addUsage already exists in orchestrate/ — reuse directly.
func addUsage(a, b agents.Usage) agents.Usage {
	return agents.Usage{LLMCalls: a.LLMCalls + b.LLMCalls, Tokens: a.Tokens + b.Tokens}
}
```

### Pattern S3: sentinel-error class per failure mode

**Source:** `orchestrate/fanout.go:253-260` (the established package
convention) + `agent.go:127-136` (the agents-package sentinel set).

**Apply to:** `orchestrate/supervisor.go` (sentinel declarations) + every
test file (use `errors.Is(err, ErrSupervisorXxx)`).

```go
var (
	ErrSupervisorNilPlanner        = errors.New("orchestrate: supervisor requires non-nil planner")
	// ... etc — distinct error per mode so callers discriminate via errors.Is
)
```

### Pattern S4: deterministic ScriptedLLM in examples + tests

**Source:** CLAUDE.md "When the user asks for code" §; canonical mock
at `examples/scriptedllm/` (+ `scriptedllm_test.go` at repo root).
`examples/06-budget/main.go:53-58` and `examples/07-policy/main.go:60-61`
both use the same shape.

**Apply to:** `examples/08-supervisor/main.go`, and any test that needs
a sequence of LLM responses (e.g., the planner across multiple rounds).

```go
plannerLLM := scriptedllm.New(
	scriptedllm.Text("dispatch to researcher: ..."),
	scriptedllm.Text("dispatch to summarizer: ..."),
	scriptedllm.Text("FINISH"),
)
```

### Pattern S5: compile-time interface assertion

**Source:** common Go idiom; `agent.go` does not assert (the file
declares the interface) — but 37-RESEARCH Example C explicitly
mandates it for Supervisor.

**Apply to:** `orchestrate/supervisor.go` (one line, near top).

```go
var _ agents.Agent = (*Supervisor)(nil)
```

This single line IS the proof that Supervisor satisfies `agents.Agent`
and can be a worker of another Supervisor (KC-1's "composition"
guarantee). A separate `TestSupervisor_SatisfiesAgentInterface` test
exists for documentary purposes — the compile-time line is the actual
check.

---

## No Analog Found

No file in Phase 37 lacks an in-tree analog. The compose-direction
tests (`TestSupervisor_InsideStateGraph`, `TestStateGraph_InsideSupervisor`)
are novel in the sense that **no prior orchestrator was composed with
the substrate this way** — but the test scaffolding (StateGraph
builder, stubAgent fixtures, ctx-propagation idioms) is all
established. The novel content is the 4-method `graphAsAgent` adapter
(~15 LOC), defined in-test, and the assertion shape — both worked out
in 37-RESEARCH Example D.

## Metadata

**Analog search scope:**
- `/orchestrate/*.go` (all files; 5 paradigms + termination + doc + graph)
- `/examples/06-budget/*.go`, `/examples/07-policy/*.go` (the two
  closest siblings; just-shipped v0.6.0/v0.6.1)
- `/agent.go`, `/agent_chatmodel.go`, `/simple.go` (the agent-level
  interface + chokepoint + simplest paradigm)
- `/budget/doc.go`, `/policy/doc.go`, `/policy/integration_test.go`
  (the two upstream phases' surfaces)

**Files scanned for analog selection:** ~15 (all read in full or in
relevant sections; no re-reads of identical ranges)

**Pattern extraction date:** 2026-05-21

**Confidence:** HIGH across all 8 files. The closest-analog choices
are forced by KC-1 (Supervisor must reuse StateGraph) and the v0.6.x
sibling examples (06-budget/07-policy) defining the example shape +
test pattern. Every excerpt above is verbatim from the cited file or
documented as a sketch with explicit line citations.

## PATTERN MAPPING COMPLETE

8 files classified (6 new + 1 doc-edit + bundled README); 5 exact analogs (`supervisor.go` ← composite of `roundrobin.go`+`graph.go`+`fanout.go`; `supervisor_test.go` ← `roundrobin_test.go`; `supervisor_budget_test.go` ← `agent_chatmodel_test.go`+`policy/integration_test.go`; `examples/08-supervisor/main.go` ← `examples/06-budget/main.go`; `examples/08-supervisor/main_test.go` ← `examples/06-budget/main_test.go`); 3 role-match (`supervisor_compose_test.go` ← `graph_test.go` patterns, no prior compose direction; `doc.go` ← self-edit; `README.md` ← `examples/07-policy/README.md`).

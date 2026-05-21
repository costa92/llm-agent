package orchestrate

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/costa92/llm-agent"
)

// Supervisor is a thin facade over StateGraph[supervisorState] per KC-1.
// It implements the planner ↔ worker iterative loop: planner emits a
// Dispatch (worker name + sub-input) → worker runs → planner observes
// → repeat or finish. The whole loop is a 3-node typed state machine
// ([plan] → conditional → [dispatch] | [final]; [dispatch] → [plan]).
//
// Q1 (37-RESEARCH §"Open Questions"): hitting MaxRounds is a GRACEFUL
// terminus. The conditional edge on [plan] routes to [final] when
// st.round >= opts.MaxRounds after the just-completed dispatch is
// appended to state.results, and BuildAggregate is called over the
// accumulated results. NOT an error. Mirrors RoundRobinChat's
// "max_turns" stop signal. ErrSupervisorMaxRounds is reserved for
// the unreachable err-translation path that wraps ErrGraphMaxSteps
// (the inner StateGraph cap should never fire when WithMaxSteps
// (MaxRounds*3+4) is set correctly).
//
// Q2: runStreamFromBlocking is REIMPLEMENTED LOCALLY in this file
// (Decision G option b) rather than depending on the agents.-package
// helper. Keeps agents.'s exported surface unchanged (KC-5).
//
// Q3: OnStep is OMITTED from SupervisorOptions in this slice. Callers
// observe per-round events via RunStream.
//
// Q4: Dispatch.Metadata IS included as map[string]any. Zero-LOC cost;
// future seam for planner-to-worker hints. Transparent to Supervisor —
// the planner-to-worker contract is caller-defined.
//
// Q5: No SupervisorAgent alias. The compile-time assertion
// `var _ agents.Agent = (*Supervisor)(nil)` at the bottom of this file
// is the definitive proof that a Supervisor satisfies agents.Agent and
// may therefore be a worker of another Supervisor (KC-1 composability).
//
// Supervisor is safe for concurrent Run calls on the same instance:
// no field on *Supervisor is mutated after construction; Run builds a
// fresh CompiledGraph per invocation and the supervisorState is
// per-Run.
type Supervisor struct {
	name string
	opts SupervisorOptions
}

// SupervisorOptions configures a Supervisor. The 5 fields are locked
// verbatim by KC-1 / CC-3; do not rename or add required fields.
type SupervisorOptions struct {
	// Planner is the agent that, given the user task + accumulated
	// WorkerResults so far, emits a free-text answer that ParseDispatch
	// converts to either a *Dispatch (continue) or nil (finish cleanly).
	Planner agents.Agent

	// Workers is the routing table. Lookup is case-sensitive — convention
	// is lowercase names. ParseDispatch is the normalization seam if
	// callers want to accept variant casings (Pitfall 1).
	Workers map[string]agents.Agent

	// MaxRounds is the hard cap on planner↔worker iterations. Must be > 0.
	// Hitting this cap is a graceful terminus (Q1 / Decision A4), not an
	// error: BuildAggregate is called with whatever results were collected
	// and the run returns a normal agents.Result.
	MaxRounds int

	// ParseDispatch converts the planner's text answer into a structured
	// Dispatch. See DispatchParser doc for the (*Dispatch, error) contract.
	ParseDispatch DispatchParser

	// BuildAggregate is called ONCE at the [final] node (Decision F) to
	// produce the user-facing Result.Answer from the accumulated
	// WorkerResults. NOT called per-round.
	BuildAggregate Aggregator
}

// Dispatch is one unit of delegation emitted by the planner per round:
// route Input to the worker named WorkerName, optionally annotated with
// planner-side Metadata.
//
// Metadata is opaque to Supervisor; planner-to-worker hints. Zero-LOC
// cost; documented as a future extension seam (Q4 / Decision B).
type Dispatch struct {
	WorkerName string
	Input      string
	Metadata   map[string]any
}

// WorkerResult pairs a Dispatch with the agents.Result the named
// worker returned. Decision C: keep the full embedded Result so the
// caller's trace, usage rollup, and nested-Supervisor debugging all
// have the per-worker detail intact.
type WorkerResult struct {
	Dispatch Dispatch
	Result   agents.Result
}

// DispatchParser converts the planner's free-text answer into a
// structured Dispatch.
//
// Contract (Pitfall 3):
//
//   - (*Dispatch, nil)  — continue: route to [dispatch], call the worker.
//   - (nil,       nil)  — terminate cleanly: skip [dispatch], go to [final].
//   - (nil,       err)  — fail: wrapped as ErrSupervisorParseDispatch + err.
type DispatchParser func(plannerAnswer string) (*Dispatch, error)

// Aggregator builds the user-facing final answer from all WorkerResults
// collected during the run. Called ONCE at the [final] node (Decision F)
// — NOT per round. An empty results slice is valid input (clean
// finish before any dispatch).
type Aggregator func(results []WorkerResult) (string, error)

// Sentinel errors. Mirror the orchestrate/fanout.go:253-260 block.
//
// ErrSupervisorMaxRounds is reserved for the err-translation path that
// wraps ErrGraphMaxSteps. That path should be unreachable in practice
// when WithMaxSteps(MaxRounds*3+4) is set correctly (Pitfall 2); see
// translateErr for the wrapping.
var (
	ErrSupervisorNilPlanner        = errors.New("orchestrate: supervisor requires non-nil planner")
	ErrSupervisorNoWorkers         = errors.New("orchestrate: supervisor requires at least one worker")
	ErrSupervisorNilParseDispatch  = errors.New("orchestrate: supervisor requires non-nil ParseDispatch")
	ErrSupervisorNilBuildAggregate = errors.New("orchestrate: supervisor requires non-nil BuildAggregate")
	ErrSupervisorUnknownWorker     = errors.New("orchestrate: supervisor dispatch references unknown worker")
	ErrSupervisorMaxRounds         = errors.New("orchestrate: supervisor max rounds exceeded")
	ErrSupervisorParseDispatch     = errors.New("orchestrate: supervisor parse dispatch failed")
)

// supervisorState is the typed S of StateGraph[supervisorState]. Each
// node is a pure function over this state; nothing is stored on
// *Supervisor itself (so concurrent Run calls do not race).
type supervisorState struct {
	// input is the user-facing task (constant across the run).
	input string
	// round is 1-indexed; incremented at the top of planNode.
	round int
	// lastPlannerAnswer is the most recent planner Result.Answer
	// (referenced by ParseDispatch + buildTrace).
	lastPlannerAnswer string
	// dispatch is the most recent parsed Dispatch; nil signals clean
	// finish (the conditional edge from [plan] routes to [final]).
	dispatch *Dispatch
	// results accumulates per-round worker output. dispatchNode
	// appends; finalNode reads.
	results []WorkerResult
	// plannerUsage accumulates planner-side LLM cost. Worker-side
	// usage lives on each WorkerResult.Result.Usage; aggregateUsage
	// sums both at the end.
	plannerUsage agents.Usage
	// finalAnswer is populated by finalNode (BuildAggregate output).
	finalAnswer string
	// onStep is the optional callback wired by RunStream (Q2 / Decision G
	// option b). The Run path leaves this nil; the streaming variant
	// installs a closure that emits StepAction / StepObservation pairs
	// per round and a terminal StepFinal.
	onStep func(agents.Step)
}

// NewSupervisor constructs a Supervisor. Validation is deferred to Run
// (matching orchestrate/fanout.go:67-77 precedent — same package,
// same vocabulary).
func NewSupervisor(name string, opts SupervisorOptions) *Supervisor {
	return &Supervisor{name: name, opts: opts}
}

// Name implements agents.Agent. Defaults to "supervisor" when the
// constructor was given an empty name (Q5 — no "Agent" suffix).
func (s *Supervisor) Name() string {
	if s.name == "" {
		return "supervisor"
	}
	return s.name
}

// validate enforces the 5 SupervisorOptions invariants. Called once at
// the top of Run / RunStream before the graph is compiled.
func (s *Supervisor) validate() error {
	if s.opts.Planner == nil {
		return ErrSupervisorNilPlanner
	}
	if len(s.opts.Workers) == 0 {
		return ErrSupervisorNoWorkers
	}
	if s.opts.ParseDispatch == nil {
		return ErrSupervisorNilParseDispatch
	}
	if s.opts.BuildAggregate == nil {
		return ErrSupervisorNilBuildAggregate
	}
	if s.opts.MaxRounds <= 0 {
		return fmt.Errorf("orchestrate: supervisor: MaxRounds must be positive (got %d)", s.opts.MaxRounds)
	}
	return nil
}

// compileGraph builds the 3-node StateGraph[supervisorState] facade per
// KC-1. The conditional edge on [plan] is the load-bearing routing
// decision: nil dispatch (clean finish) OR round-cap-hit → [final];
// else [dispatch].
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

// planNode invokes the planner, parses its answer, and stores the
// resulting Dispatch (or nil — finish signal) on state. ctx is passed
// UNCHANGED so the budget tracker propagates (Pitfall 4 hard rule).
func (s *Supervisor) planNode(ctx context.Context, st supervisorState) (supervisorState, error) {
	st.round++
	prompt := s.buildPlannerPrompt(st)
	res, err := s.opts.Planner.Run(ctx, prompt)
	if err != nil {
		return st, fmt.Errorf("supervisor %q: planner round %d: %w", s.Name(), st.round, err)
	}
	st.lastPlannerAnswer = res.Answer
	st.plannerUsage = addUsage(st.plannerUsage, res.Usage)
	d, perr := s.opts.ParseDispatch(res.Answer)
	if perr != nil {
		// Wrap BOTH the sentinel and the underlying err so callers can
		// errors.Is(err, ErrSupervisorParseDispatch) AND see the
		// parser's own message in err.Error().
		return st, fmt.Errorf("supervisor %q: parse dispatch round %d: %w: %v",
			s.Name(), st.round, ErrSupervisorParseDispatch, perr)
	}
	st.dispatch = d // may be nil — that is the clean-finish signal
	return st, nil
}

// dispatchNode looks up the named worker and runs it. ctx UNCHANGED
// (Pitfall 4). The conditional edge guarantees st.dispatch is non-nil
// when this node runs.
func (s *Supervisor) dispatchNode(ctx context.Context, st supervisorState) (supervisorState, error) {
	d := st.dispatch
	worker, ok := s.opts.Workers[d.WorkerName]
	if !ok || worker == nil {
		return st, fmt.Errorf("%w: %q (round %d)", ErrSupervisorUnknownWorker, d.WorkerName, st.round)
	}
	res, err := worker.Run(ctx, d.Input)
	if err != nil {
		return st, fmt.Errorf("supervisor %q: worker %q round %d: %w",
			s.Name(), d.WorkerName, st.round, err)
	}
	wr := WorkerResult{Dispatch: *d, Result: res}
	st.results = append(st.results, wr)
	if st.onStep != nil {
		st.onStep(agents.Step{
			Kind: agents.StepAction,
			Tool: wr.Dispatch.WorkerName,
			Args: wr.Dispatch.Input,
		})
		st.onStep(agents.Step{
			Kind:   agents.StepObservation,
			Result: wr.Result.Answer,
		})
	}
	return st, nil
}

// finalNode calls BuildAggregate ONCE (Decision F) over the accumulated
// results and stores the answer on state.
func (s *Supervisor) finalNode(_ context.Context, st supervisorState) (supervisorState, error) {
	ans, err := s.opts.BuildAggregate(st.results)
	if err != nil {
		return st, fmt.Errorf("supervisor %q: aggregate: %w", s.Name(), err)
	}
	st.finalAnswer = ans
	return st, nil
}

// routeFromPlan implements the conditional edge from [plan]. Order
// matters: nil-dispatch (clean finish) wins over cap-hit so a planner
// round that emits (nil, nil) terminates cleanly without spuriously
// triggering the cap-hit path (Q1 / A4). The cap is based on completed
// dispatches, so MaxRounds N means N worker rounds before the next
// planner pass routes to [final].
func (s *Supervisor) routeFromPlan(st supervisorState) string {
	if st.dispatch == nil {
		return "final"
	}
	if len(st.results) >= s.opts.MaxRounds {
		return "final"
	}
	return "dispatch"
}

// buildPlannerPrompt deterministically stringifies the current state
// into the prompt the planner sees. Format:
//
//	Task: <input>
//
//	Round <N> of up to <MaxRounds>.
//	[Prior results — K items:]
//	  [1] dispatched <worker>(<input>) → <answer>
//	  ...
//	Decide the next dispatch or finish.
//
// For round 1 (no prior results) the "[Prior results …]" block is
// omitted.
func (s *Supervisor) buildPlannerPrompt(st supervisorState) string {
	var b strings.Builder
	b.WriteString("Task: ")
	b.WriteString(st.input)
	b.WriteString("\n\nRound ")
	fmt.Fprintf(&b, "%d of up to %d.\n", st.round, s.opts.MaxRounds)
	if len(st.results) > 0 {
		fmt.Fprintf(&b, "Prior results — %d items:\n", len(st.results))
		for i, wr := range st.results {
			fmt.Fprintf(&b, "  [%d] dispatched %s(%s) → %s\n",
				i+1, wr.Dispatch.WorkerName, wr.Dispatch.Input, wr.Result.Answer)
		}
	}
	b.WriteString("Decide the next dispatch or finish.")
	return b.String()
}

// translateErr maps ErrGraphMaxSteps → ErrSupervisorMaxRounds (Pitfall 2).
// Other errors (ctx.Err, planner err, worker err, parser err, aggregate
// err) flow through unchanged — they already carry the supervisor /
// worker / round breadcrumbs from the node wrappers.
func (s *Supervisor) translateErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrGraphMaxSteps) {
		return fmt.Errorf("%w: rounds=%d", ErrSupervisorMaxRounds, s.opts.MaxRounds)
	}
	return err
}

// buildTrace assembles the agents.Result.Trace from the accumulated
// state. Pattern 3 of 37-RESEARCH: per-round (StepAction +
// StepObservation), optional trailing StepThought with the latest
// planner answer, then a terminal StepFinal.
func (s *Supervisor) buildTrace(st supervisorState) []agents.Step {
	trace := make([]agents.Step, 0, len(st.results)*2+2)
	for _, wr := range st.results {
		trace = append(trace,
			agents.Step{
				Kind: agents.StepAction,
				Tool: wr.Dispatch.WorkerName,
				Args: wr.Dispatch.Input,
			},
			agents.Step{
				Kind:   agents.StepObservation,
				Result: wr.Result.Answer,
			},
		)
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

// aggregateUsage rolls planner-side + per-worker usage into the final
// agents.Usage. Reuses orchestrate/fanout.go's package-local addUsage.
func (s *Supervisor) aggregateUsage(st supervisorState) agents.Usage {
	usage := st.plannerUsage
	for _, wr := range st.results {
		usage = addUsage(usage, wr.Result.Usage)
	}
	return usage
}

// Run implements agents.Agent. Builds the 3-node graph, drives it with
// WithMaxSteps(opts.MaxRounds*3+4), translates ErrGraphMaxSteps →
// ErrSupervisorMaxRounds, and assembles the final agents.Result.
func (s *Supervisor) Run(ctx context.Context, input string) (agents.Result, error) {
	return s.runInternal(ctx, input, nil)
}

// runInternal is the shared core for Run + RunStream. When onStep is
// nil the streaming hooks are no-ops; when non-nil, dispatchNode emits
// StepAction + StepObservation per round and the helper emits a
// terminal StepFinal before returning.
func (s *Supervisor) runInternal(ctx context.Context, input string, onStep func(agents.Step)) (agents.Result, error) {
	if err := s.validate(); err != nil {
		return agents.Result{}, err
	}
	cg, err := s.compileGraph()
	if err != nil {
		return agents.Result{}, fmt.Errorf("supervisor %q: compile: %w", s.Name(), err)
	}
	initial := supervisorState{input: input, onStep: onStep}
	// MaxSteps slack: each round consumes 2 graph steps ([plan] +
	// [dispatch]); the terminating round adds [plan] + [final]; +1
	// slack against off-by-one (Pitfall 2 + A3).
	maxSteps := s.opts.MaxRounds*3 + 4
	final, runErr := cg.Run(ctx, initial, WithMaxSteps(maxSteps))
	if runErr != nil {
		return agents.Result{}, s.translateErr(runErr)
	}
	if onStep != nil {
		onStep(agents.Step{Kind: agents.StepFinal, Content: final.finalAnswer})
	}
	return agents.Result{
		Answer: final.finalAnswer,
		Trace:  s.buildTrace(final),
		Usage:  s.aggregateUsage(final),
	}, nil
}

// RunStream implements agents.Agent. Local reimplementation of
// runStreamFromBlocking (Q2 / Decision G option b): spawns a goroutine
// that drives runInternal, pipes Step→StepEvent into a 16-buffer
// channel, and emits a terminal {Done: true, Final|Err} before
// closing.
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

// Compile-time assertion — KC-1 composability proof. A *Supervisor
// satisfies agents.Agent, so a Supervisor may be a worker (or planner)
// of another Supervisor.
var _ agents.Agent = (*Supervisor)(nil)

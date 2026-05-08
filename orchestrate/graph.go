package orchestrate

import (
	"context"
	"errors"
	"fmt"
)

// Reserved node names used by Compile/Run as graph entry/terminator.
const (
	NodeStart = "__start__"
	NodeEnd   = "__end__"
)

// NodeFunc is a single graph node. Receives current state, returns
// updated state. State is generic so callers get type safety on their
// domain shape (no map[string]any sprawl).
type NodeFunc[S any] func(ctx context.Context, state S) (S, error)

// ConditionFunc is the body of a conditional edge. Returns the NAME
// of the next node (must be either NodeEnd or a registered node name).
type ConditionFunc[S any] func(state S) string

// StateGraph is a typed graph builder. Compile validates
// edges + entry, then returns a CompiledGraph that you Run.
//
// Mirrors LangGraph's StateGraph (Python). Use when you need explicit
// control flow + observable state transitions, not just A→B→C.
type StateGraph[S any] struct {
	nodes      map[string]NodeFunc[S]
	edges      map[string]string // from → to
	condEdges  map[string]ConditionFunc[S]
	entryPoint string
}

// NewStateGraph constructs an empty StateGraph.
func NewStateGraph[S any]() *StateGraph[S] {
	return &StateGraph[S]{
		nodes:     make(map[string]NodeFunc[S]),
		edges:     make(map[string]string),
		condEdges: make(map[string]ConditionFunc[S]),
	}
}

// AddNode registers a node by name. Builder-pattern (returns g) so calls
// chain. Panics if name is reserved (NodeStart / NodeEnd).
func (g *StateGraph[S]) AddNode(name string, fn NodeFunc[S]) *StateGraph[S] {
	if name == NodeStart || name == NodeEnd {
		panic(fmt.Sprintf("orchestrate: node name %q is reserved", name))
	}
	g.nodes[name] = fn
	return g
}

// AddEdge declares an unconditional transition from→to. The "to" can
// be NodeEnd to terminate the graph. A NodeStart→X edge is shorthand
// for SetEntry(X); last write wins if both forms are used.
func (g *StateGraph[S]) AddEdge(from, to string) *StateGraph[S] {
	if from == NodeStart {
		g.entryPoint = to
		return g
	}
	g.edges[from] = to
	return g
}

// AddConditionalEdge declares a conditional transition: when execution
// reaches `from`, the ConditionFunc decides which named node is next.
// Conditional edges take precedence over unconditional edges from the
// same source (so you can register both — the conditional wins).
func (g *StateGraph[S]) AddConditionalEdge(from string, fn ConditionFunc[S]) *StateGraph[S] {
	g.condEdges[from] = fn
	return g
}

// SetEntry marks which node Run starts at. Overwrites any prior entry,
// including one declared via AddEdge(NodeStart, "...").
func (g *StateGraph[S]) SetEntry(name string) *StateGraph[S] {
	g.entryPoint = name
	return g
}

// Compile validates the graph and returns an immutable CompiledGraph.
// Validation: entry must exist, all edge targets must be registered
// nodes (or NodeEnd), condEdges' returned names are NOT pre-validated
// (they're checked at Run time per state). Entry may be declared either
// with SetEntry(name) or AddEdge(NodeStart, name).
func (g *StateGraph[S]) Compile() (*CompiledGraph[S], error) {
	if g.entryPoint == "" {
		return nil, errors.New("orchestrate: graph has no entry point (call SetEntry or AddEdge(NodeStart, ...))")
	}
	if _, ok := g.nodes[g.entryPoint]; !ok {
		return nil, fmt.Errorf("orchestrate: entry %q is not a registered node", g.entryPoint)
	}
	for from, to := range g.edges {
		if from != NodeStart {
			if _, ok := g.nodes[from]; !ok {
				return nil, fmt.Errorf("orchestrate: edge from unknown node %q", from)
			}
		}
		if to != NodeEnd {
			if _, ok := g.nodes[to]; !ok {
				return nil, fmt.Errorf("orchestrate: edge from %q targets unknown node %q", from, to)
			}
		}
	}
	for from := range g.condEdges {
		if _, ok := g.nodes[from]; !ok {
			return nil, fmt.Errorf("orchestrate: conditional edge from unknown node %q", from)
		}
	}

	// Snapshot — copy maps so post-Compile mutations to the builder
	// don't affect the compiled graph.
	cg := &CompiledGraph[S]{
		nodes:      make(map[string]NodeFunc[S], len(g.nodes)),
		edges:      make(map[string]string, len(g.edges)),
		condEdges:  make(map[string]ConditionFunc[S], len(g.condEdges)),
		entryPoint: g.entryPoint,
	}
	for k, v := range g.nodes {
		cg.nodes[k] = v
	}
	for k, v := range g.edges {
		cg.edges[k] = v
	}
	for k, v := range g.condEdges {
		cg.condEdges[k] = v
	}
	return cg, nil
}

// CompiledGraph is the Run-ready, immutable result of Compile.
type CompiledGraph[S any] struct {
	nodes      map[string]NodeFunc[S]
	edges      map[string]string
	condEdges  map[string]ConditionFunc[S]
	entryPoint string
}

// RunOption tunes per-Run behavior.
type RunOption func(*runOptions)

type runOptions struct {
	maxSteps int
}

const defaultMaxSteps = 100

// WithMaxSteps overrides the default 100-step cap. Use a higher value
// for graphs with intentional loops; lower for guard-rails.
func WithMaxSteps(n int) RunOption {
	return func(o *runOptions) { o.maxSteps = n }
}

// Run executes the graph starting at entry. Each iteration: run
// current node → consult conditional edge (if any) else unconditional
// edge → if next == NodeEnd, return state. ErrGraphMaxSteps fires if
// the loop runs longer than MaxSteps (default 100).
func (c *CompiledGraph[S]) Run(ctx context.Context, initial S, opts ...RunOption) (S, error) {
	cfg := runOptions{maxSteps: defaultMaxSteps}
	for _, opt := range opts {
		opt(&cfg)
	}

	state := initial
	current := c.entryPoint

	for step := 0; step < cfg.maxSteps; step++ {
		select {
		case <-ctx.Done():
			return state, ctx.Err()
		default:
		}

		fn, ok := c.nodes[current]
		if !ok {
			return state, fmt.Errorf("orchestrate: node %q not found at step %d", current, step)
		}
		next, err := fn(ctx, state)
		if err != nil {
			return state, fmt.Errorf("orchestrate: node %q: %w", current, err)
		}
		state = next

		// Decide next node
		var dest string
		if cond, ok := c.condEdges[current]; ok {
			dest = cond(state)
		} else if e, ok := c.edges[current]; ok {
			dest = e
		} else {
			return state, fmt.Errorf("orchestrate: node %q has no outgoing edge", current)
		}

		if dest == NodeEnd {
			return state, nil
		}
		if _, ok := c.nodes[dest]; !ok {
			return state, fmt.Errorf("orchestrate: node %q routed to unknown node %q", current, dest)
		}
		current = dest
	}
	return state, ErrGraphMaxSteps
}

// ErrGraphMaxSteps is returned when Run exceeds MaxSteps (default 100).
var ErrGraphMaxSteps = errors.New("orchestrate: graph max steps exceeded")

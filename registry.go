package agents

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/costa92/llm-agent/llm"
)

// Registry holds a name→Tool map. Use one Registry per Agent (or share)
// so test isolation is preserved (no init()-time global singleton).
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry returns a Registry with the given tools batch-registered.
// Panics if duplicate names appear in the variadic args (constructor-time
// safety: prefer panic over silent shadowing).
func NewRegistry(tools ...Tool) *Registry {
	r := &Registry{tools: make(map[string]Tool, len(tools))}
	for _, t := range tools {
		if err := r.Register(t); err != nil {
			panic(fmt.Sprintf("agents.NewRegistry: %v", err))
		}
	}
	return r
}

// Register adds a tool. Returns ErrToolAlreadyRegistered if name collides.
func (r *Registry) Register(t Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tools[t.Name()]; exists {
		return fmt.Errorf("%w: %q", ErrToolAlreadyRegistered, t.Name())
	}
	r.tools[t.Name()] = t
	return nil
}

// Get returns the tool registered under name.
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// List returns all tools sorted by Name (deterministic).
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

// AsLLMTools returns all tools formatted for llm.ToolCaller bindings.
func (r *Registry) AsLLMTools() []llm.Tool {
	tools := r.List()
	out := make([]llm.Tool, len(tools))
	for i, t := range tools {
		out[i] = AsLLMTool(t)
	}
	return out
}

// PromptDescription formats all tools as a "- name: description" list suitable
// for prompt injection. Empty registry returns "(none)\n". Used by ReActAgent
// internally and exposed for external prompt assembly (e.g., Phase 8 PlannerAgent).
func (r *Registry) PromptDescription() string {
	tools := r.List()
	if len(tools) == 0 {
		return "(none)\n"
	}
	var b strings.Builder
	for _, t := range tools {
		fmt.Fprintf(&b, "- %s: %s\n", t.Name(), t.Description())
	}
	return b.String()
}

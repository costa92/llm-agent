package patterns

import (
	"fmt"

	agents "github.com/costa92/llm-agent"
	"github.com/costa92/llm-agent-contract/llm"
)

// ID identifies a productized Agent pattern.
type ID string

const (
	Simple       ID = "simple"
	ReAct        ID = "react"
	FunctionCall ID = "function_call"
	PlanAndSolve ID = "plan_and_solve"
	Reflection   ID = "reflection"
	Workspace    ID = "workspace"
)

// Capability names the behavior a preset exposes.
type Capability string

const (
	CapabilitySingleShot Capability = "single_shot"
	CapabilityTools      Capability = "tools"
	CapabilityPlanning   Capability = "planning"
	CapabilityReflection Capability = "reflection"
	CapabilityWorkspace  Capability = "workspace"
)

// Preset describes one productized pattern.
type Preset struct {
	ID           ID
	Name         string
	Description  string
	Capabilities []Capability
}

// Options configures Build.
type Options struct {
	Name         string
	SystemPrompt string
	Registry     *agents.Registry
	MaxSteps     int
	MaxRounds    int
	MaxParallel  int
	Callback     agents.Callback
}

// Catalog returns the built-in pattern catalog in stable order.
func Catalog() []Preset {
	return []Preset{
		{
			ID:           Simple,
			Name:         "Simple",
			Description:  "Single-shot model call for direct answers.",
			Capabilities: []Capability{CapabilitySingleShot},
		},
		{
			ID:           ReAct,
			Name:         "ReAct",
			Description:  "Thought/action/observation loop over a tool registry.",
			Capabilities: []Capability{CapabilityTools},
		},
		{
			ID:           FunctionCall,
			Name:         "Function Call",
			Description:  "Native tool-call execution using provider tool calling.",
			Capabilities: []Capability{CapabilityTools},
		},
		{
			ID:           PlanAndSolve,
			Name:         "Plan and Solve",
			Description:  "Plan once, execute each step, then synthesize the answer.",
			Capabilities: []Capability{CapabilityPlanning},
		},
		{
			ID:           Reflection,
			Name:         "Reflection",
			Description:  "Generate, critique, and revise until approved or max rounds.",
			Capabilities: []Capability{CapabilityReflection},
		},
		{
			ID:           Workspace,
			Name:         "Workspace",
			Description:  "ReAct-style workspace agent over caller-supplied tools.",
			Capabilities: []Capability{CapabilityTools, CapabilityWorkspace},
		},
	}
}

// Build constructs an Agent for a built-in pattern.
func Build(id ID, model llm.ChatModel, opts Options) (agents.Agent, error) {
	var out agents.Agent
	switch id {
	case Simple:
		out = agents.NewSimpleAgent(model, agents.SimpleOptions{
			Name:         opts.Name,
			SystemPrompt: opts.SystemPrompt,
		})
	case ReAct:
		out = agents.NewReActAgent(model, agents.ReActOptions{
			Name:         opts.Name,
			Registry:     opts.Registry,
			MaxSteps:     opts.MaxSteps,
			SystemPrompt: opts.SystemPrompt,
		})
	case FunctionCall:
		a, err := agents.NewFunctionCallAgent(model, agents.FunctionCallOptions{
			Name:         opts.Name,
			Registry:     opts.Registry,
			SystemPrompt: opts.SystemPrompt,
			MaxParallel:  opts.MaxParallel,
		})
		if err != nil {
			return nil, err
		}
		out = a
	case PlanAndSolve:
		out = agents.NewPlanAndSolveAgent(model, agents.PlanAndSolveOptions{
			Name:     opts.Name,
			MaxSteps: opts.MaxSteps,
		})
	case Reflection:
		out = agents.NewReflectionAgent(model, agents.ReflectionOptions{
			Name:      opts.Name,
			MaxRounds: opts.MaxRounds,
		})
	case Workspace:
		if opts.Registry == nil {
			return nil, fmt.Errorf("patterns: workspace requires a Registry")
		}
		system := opts.SystemPrompt
		if system == "" {
			system = workspaceSystemPrompt
		}
		out = agents.NewReActAgent(model, agents.ReActOptions{
			Name:         defaultName(opts.Name, "workspace"),
			Registry:     opts.Registry,
			MaxSteps:     opts.MaxSteps,
			SystemPrompt: system,
		})
	default:
		return nil, fmt.Errorf("patterns: unknown pattern %q", id)
	}
	if opts.Callback != nil {
		out = agents.WrapAgent(out, opts.Callback)
	}
	return out, nil
}

const workspaceSystemPrompt = `You are a workspace agent. Use the supplied tools only when they are relevant, keep actions bounded to the user's task, and provide a concise final answer.

Format strictly:
  Thought: <one-line reasoning>
  Action: <tool name>
  Args: <JSON args object>
-- OR --
  Thought: <one-line reasoning>
  Final: <answer>

Available tools:
%s
`

func defaultName(got, fallback string) string {
	if got != "" {
		return got
	}
	return fallback
}

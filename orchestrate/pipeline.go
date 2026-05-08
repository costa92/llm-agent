package orchestrate

import (
	"context"
	"errors"
	"fmt"

	"github.com/costa92/llm-agent"
)

// Step is one stage of a Pipeline. Adapt is optional — when nil, the
// next step's input is the previous step's Result.Answer.
type Step struct {
	Name  string
	Agent agents.Agent
	Adapt func(prev agents.Result) string
}

// Pipeline runs Steps in order, threading each Step's Result into the
// next Step's input. Linear A→B→C orchestration with no branching.
//
// Mirrors AgentScope's sequential pipeline (Python). Phase 8's
// Coordinator can use Pipeline for the plan→report ends; the parallel
// summarize×N middle keeps using agents.AsyncRunner.
type Pipeline struct {
	name  string
	steps []Step
}

// PipelineResult carries every step's full Result (for trace inspection)
// plus the final answer + accumulated Usage.
type PipelineResult struct {
	FinalAnswer string
	StepResults []StepResult
	TotalUsage  agents.Usage
}

// StepResult pairs a Step's Name with its agents.Result (full Trace
// preserved so callers can drill into intermediate behavior).
type StepResult struct {
	Step   string
	Result agents.Result
}

// NewPipeline constructs a Pipeline. Steps run in the order passed.
// Empty steps slice is allowed but Run will return an error on call.
func NewPipeline(name string, steps ...Step) *Pipeline {
	return &Pipeline{name: name, steps: steps}
}

// Name returns the pipeline name (used in logs / Trace identifiers).
func (p *Pipeline) Name() string {
	if p.name == "" {
		return "pipeline"
	}
	return p.name
}

// Run executes all Steps sequentially. Returns ErrEmptyPipeline if no
// steps were provided. Step errors abort with the partial result
// captured for inspection.
func (p *Pipeline) Run(ctx context.Context, input string) (PipelineResult, error) {
	if len(p.steps) == 0 {
		return PipelineResult{}, ErrEmptyPipeline
	}

	out := PipelineResult{StepResults: make([]StepResult, 0, len(p.steps))}
	curInput := input
	var lastResult agents.Result

	for i, step := range p.steps {
		if step.Agent == nil {
			return out, fmt.Errorf("pipeline %q: step[%d] %q has nil Agent", p.Name(), i, step.Name)
		}
		res, err := step.Agent.Run(ctx, curInput)
		if err != nil {
			return out, fmt.Errorf("pipeline %q: step %q: %w", p.Name(), step.Name, err)
		}
		out.StepResults = append(out.StepResults, StepResult{Step: step.Name, Result: res})
		out.TotalUsage.LLMCalls += res.Usage.LLMCalls
		out.TotalUsage.Tokens += res.Usage.Tokens
		lastResult = res

		// Prepare next input
		if i < len(p.steps)-1 {
			next := p.steps[i+1]
			if next.Adapt != nil {
				curInput = next.Adapt(res)
			} else {
				curInput = res.Answer
			}
		}
	}

	out.FinalAnswer = lastResult.Answer
	return out, nil
}

// ErrEmptyPipeline is returned by Run when no Steps were registered.
var ErrEmptyPipeline = errors.New("orchestrate: pipeline has no steps")

package agents

import (
	"context"
	"fmt"
	"strings"

	"github.com/costa92/llm-agent/llm"
)

// FunctionCallAgent uses native OpenAI-style function-calling: pkg/llm.Tool +
// resp.ToolCalls instead of prompt-based parsing. Single-turn — emits one LLM
// call, executes returned tool calls in parallel via AsyncRunner, aggregates
// outputs as the answer.
//
// Why single-turn: pkg/llm.Message has no ToolCallID field, so we can't feed
// tool results back to the LLM per OpenAI spec for multi-turn function-calling.
// Multi-turn would require a pkg/llm enhancement (out of scope for this phase).
type FunctionCallAgent struct {
	model llm.ToolCaller
	opts  FunctionCallOptions
}

// FunctionCallOptions configures FunctionCallAgent.
type FunctionCallOptions struct {
	Name         string     // default "function-call"
	Registry     *Registry  // required
	SystemPrompt string     // optional
	MaxParallel  int        // default 4 — caps goroutines spawned per Run
	OnStep       func(Step) // optional
}

// NewFunctionCallAgent constructs a FunctionCallAgent.
func NewFunctionCallAgent(model llm.ChatModel, opts FunctionCallOptions) (*FunctionCallAgent, error) {
	if opts.Name == "" {
		opts.Name = "function-call"
	}
	if opts.MaxParallel == 0 {
		opts.MaxParallel = 4
	}
	tc, ok := nativeToolCaller(model)
	if !ok {
		return nil, toolCapabilityError(model)
	}
	return &FunctionCallAgent{model: tc, opts: opts}, nil
}

// Name implements Agent.
func (a *FunctionCallAgent) Name() string { return a.opts.Name }

// Run executes one round of function-calling.
func (a *FunctionCallAgent) Run(ctx context.Context, input string) (Result, error) {
	return a.runInternal(ctx, input, normalizedOnStep(a.opts.OnStep))
}

// RunStream emits step events through a channel; see Agent interface docs.
func (a *FunctionCallAgent) RunStream(ctx context.Context, input string) (<-chan StepEvent, error) {
	return runStreamFromBlocking(ctx, func(ctx context.Context, cb func(Step)) (Result, error) {
		return a.runInternal(ctx, input, cb)
	})
}

// runInternal assumes onStep is non-nil (caller normalizes via normalizedOnStep).
func (a *FunctionCallAgent) runInternal(ctx context.Context, input string, onStep func(Step)) (Result, error) {
	if strings.TrimSpace(input) == "" {
		return Result{}, ErrEmptyInput
	}
	if a.opts.Registry == nil {
		return Result{}, fmt.Errorf("function-call agent requires a Registry")
	}

	prompt := input
	if a.opts.SystemPrompt != "" {
		prompt = a.opts.SystemPrompt + "\n\n" + input
	}

	toolModel, err := a.model.WithTools(a.opts.Registry.AsLLMTools())
	if err != nil {
		return Result{}, err
	}
	resp, err := generateFromPrompt(ctx, toolModel, "", prompt)
	if err != nil {
		return Result{}, err
	}
	usage := Usage{LLMCalls: 1, Tokens: resp.Usage.TotalTokens}
	trace := []Step{}

	// No tool call → return text directly.
	if len(resp.ToolCalls) == 0 {
		final := Step{Kind: StepFinal, Content: resp.Text}
		onStep(final)
		trace = append(trace, final)
		return Result{Answer: resp.Text, Trace: trace, Usage: usage}, nil
	}

	// Build AsyncRunner Tasks (looking up tools first; unknown tool aborts).
	tasks := make([]Task, 0, len(resp.ToolCalls))
	for _, tc := range resp.ToolCalls {
		tool, ok := a.opts.Registry.Get(tc.Name)
		if !ok {
			return Result{}, fmt.Errorf("%w: %q", ErrToolNotFound, tc.Name)
		}
		tasks = append(tasks, Task{Tool: tool, Args: tc.Arguments})
	}
	runner := NewAsyncRunner(a.opts.MaxParallel)
	results, err := runner.Execute(ctx, tasks)
	if err != nil {
		return Result{}, err // ctx cancel / timeout
	}

	// Fail-fast: any tool execution error aborts (parallel tools may have already run).
	for _, r := range results {
		if r.Err != nil {
			return Result{}, r.Err
		}
	}

	// Aggregate trace + answer.
	var b strings.Builder
	for i, r := range results {
		tc := resp.ToolCalls[i]
		actionStep := Step{Kind: StepAction, Tool: tc.Name, Args: string(tc.Arguments)}
		onStep(actionStep)
		trace = append(trace, actionStep)

		obsStep := Step{Kind: StepObservation, Result: r.Output}
		onStep(obsStep)
		trace = append(trace, obsStep)

		if b.Len() > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "%s: %s", tc.Name, r.Output)
	}
	answer := b.String()

	finalStep := Step{Kind: StepFinal, Content: answer}
	onStep(finalStep)
	trace = append(trace, finalStep)

	return Result{Answer: answer, Trace: trace, Usage: usage}, nil
}

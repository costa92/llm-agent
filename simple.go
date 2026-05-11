package agents

import (
	"context"
	"strings"

	"github.com/costa92/llm-agent/llm"
)

// SimpleAgent forwards user input to an llm.Client in a single call.
// No tools, no loop — the simplest possible Agent.
type SimpleAgent struct {
	model llm.ChatModel
	opts  SimpleOptions
}

// SimpleOptions configures SimpleAgent.
type SimpleOptions struct {
	Name         string     // default "simple"
	SystemPrompt string     // optional, prepended to user input as a system context
	OnStep       func(Step) // optional, called for each trace step (synchronous)
}

// NewSimpleAgent constructs a SimpleAgent.
func NewSimpleAgent(model llm.ChatModel, opts SimpleOptions) *SimpleAgent {
	if opts.Name == "" {
		opts.Name = "simple"
	}
	return &SimpleAgent{model: model, opts: opts}
}

// Name implements Agent.
func (a *SimpleAgent) Name() string { return a.opts.Name }

// Run sends one prompt and returns the LLM's reply as the final answer.
func (a *SimpleAgent) Run(ctx context.Context, input string) (Result, error) {
	return a.runInternal(ctx, input, normalizedOnStep(a.opts.OnStep))
}

// RunStream emits step events through a channel; see Agent interface docs.
func (a *SimpleAgent) RunStream(ctx context.Context, input string) (<-chan StepEvent, error) {
	return runStreamFromBlocking(ctx, func(ctx context.Context, cb func(Step)) (Result, error) {
		return a.runInternal(ctx, input, cb)
	})
}

// runInternal assumes onStep is non-nil (caller normalizes via normalizedOnStep).
func (a *SimpleAgent) runInternal(ctx context.Context, input string, onStep func(Step)) (Result, error) {
	if strings.TrimSpace(input) == "" {
		return Result{}, ErrEmptyInput
	}
	resp, err := generateFromPrompt(ctx, a.model, a.opts.SystemPrompt, input)
	if err != nil {
		return Result{}, err
	}
	final := Step{Kind: StepFinal, Content: resp.Text}
	onStep(final)
	return Result{
		Answer: resp.Text,
		Trace:  []Step{final},
		Usage:  Usage{LLMCalls: 1, Tokens: resp.Usage.TotalTokens},
	}, nil
}

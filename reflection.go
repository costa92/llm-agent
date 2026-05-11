package agents

import (
	"context"
	"fmt"
	"strings"

	"github.com/costa92/llm-agent/llm"
)

// ReflectionAgent: generate → critique → revise → ... up to MaxRounds.
// If the critique contains "APPROVED" (case-insensitive), stops early.
type ReflectionAgent struct {
	model llm.ChatModel
	opts  ReflectionOptions
}

// ReflectionOptions configures ReflectionAgent.
type ReflectionOptions struct {
	Name           string     // default "reflection"
	MaxRounds      int        // default 2
	GenPrompt      string     // default genPromptDefault
	CritiquePrompt string     // default critiquePromptDefault
	RevisePrompt   string     // default revisePromptDefault
	OnStep         func(Step) // optional, called for each trace step (synchronous)
}

const (
	genPromptDefault       = "Task: %s\n\nProduce your best initial answer."
	critiquePromptDefault  = "Task: %s\n\nCurrent draft:\n%s\n\nIf the draft is good, reply exactly: APPROVED. Otherwise critique its weaknesses in 1-3 sentences."
	revisePromptDefault    = "Task: %s\n\nPrevious draft:\n%s\n\nCritique:\n%s\n\nProduce a revised draft addressing the critique."
	approvalSentinelMarker = "APPROVED"
)

// NewReflectionAgent constructs a ReflectionAgent.
func NewReflectionAgent(model llm.ChatModel, opts ReflectionOptions) *ReflectionAgent {
	if opts.Name == "" {
		opts.Name = "reflection"
	}
	if opts.MaxRounds == 0 {
		opts.MaxRounds = 2
	}
	if opts.GenPrompt == "" {
		opts.GenPrompt = genPromptDefault
	}
	if opts.CritiquePrompt == "" {
		opts.CritiquePrompt = critiquePromptDefault
	}
	if opts.RevisePrompt == "" {
		opts.RevisePrompt = revisePromptDefault
	}
	return &ReflectionAgent{model: model, opts: opts}
}

// Name implements Agent.
func (a *ReflectionAgent) Name() string { return a.opts.Name }

// Run executes the gen→critique→revise loop.
func (a *ReflectionAgent) Run(ctx context.Context, input string) (Result, error) {
	return a.runInternal(ctx, input, normalizedOnStep(a.opts.OnStep))
}

// RunStream emits step events through a channel; see Agent interface docs.
func (a *ReflectionAgent) RunStream(ctx context.Context, input string) (<-chan StepEvent, error) {
	return runStreamFromBlocking(ctx, func(ctx context.Context, cb func(Step)) (Result, error) {
		return a.runInternal(ctx, input, cb)
	})
}

// runInternal assumes onStep is non-nil (caller normalizes via normalizedOnStep).
func (a *ReflectionAgent) runInternal(ctx context.Context, input string, onStep func(Step)) (Result, error) {
	if strings.TrimSpace(input) == "" {
		return Result{}, ErrEmptyInput
	}
	trace := make([]Step, 0)
	usage := Usage{}

	// initial gen
	resp, err := generateFromPrompt(ctx, a.model, "", fmt.Sprintf(a.opts.GenPrompt, input))
	if err != nil {
		return Result{}, err
	}
	usage.LLMCalls++
	usage.Tokens += resp.Usage.TotalTokens
	current := resp.Text
	initStep := Step{Kind: StepThought, Content: "initial draft: " + current}
	trace = append(trace, initStep)
	onStep(initStep)

	for round := 0; round < a.opts.MaxRounds; round++ {
		// critique
		critResp, err := generateFromPrompt(ctx, a.model, "", fmt.Sprintf(a.opts.CritiquePrompt, input, current))
		if err != nil {
			return Result{}, err
		}
		usage.LLMCalls++
		usage.Tokens += critResp.Usage.TotalTokens
		critique := critResp.Text
		critStep := Step{Kind: StepReflection, Content: critique}
		trace = append(trace, critStep)
		onStep(critStep)

		if strings.Contains(strings.ToUpper(critique), approvalSentinelMarker) {
			break
		}

		// revise
		revResp, err := generateFromPrompt(ctx, a.model, "", fmt.Sprintf(a.opts.RevisePrompt, input, current, critique))
		if err != nil {
			return Result{}, err
		}
		usage.LLMCalls++
		usage.Tokens += revResp.Usage.TotalTokens
		current = revResp.Text
		revStep := Step{Kind: StepThought, Content: "revised: " + current}
		trace = append(trace, revStep)
		onStep(revStep)
	}
	finalStep := Step{Kind: StepFinal, Content: current}
	trace = append(trace, finalStep)
	onStep(finalStep)
	return Result{Answer: current, Trace: trace, Usage: usage}, nil
}

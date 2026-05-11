package agents

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/costa92/llm-agent/llm"
)

// PlanAndSolveAgent: plan once (LLM emits N steps), then execute each step
// in a single LLM call, finally synthesize a final answer.
type PlanAndSolveAgent struct {
	model llm.ChatModel
	opts  PlanAndSolveOptions
}

// PlanAndSolveOptions configures PlanAndSolveAgent.
type PlanAndSolveOptions struct {
	Name        string     // default "plan-and-solve"
	MaxSteps    int        // default 8 — caps the number of steps the planner may emit
	PlanPrompt  string     // default planPromptDefault
	StepPrompt  string     // default stepPromptDefault
	SynthPrompt string     // default synthPromptDefault
	OnStep      func(Step) // optional, called for each trace step (synchronous)
}

const (
	planPromptDefault  = "Task: %s\n\nProduce a numbered plan starting with 'PLAN:' followed by 1-%d short steps, one per line, prefixed by '<n>. '."
	stepPromptDefault  = "Task: %s\n\nPlan:\n%s\n\nExecute step %d: %s\n\nReturn only the step's result, no preamble."
	synthPromptDefault = "Task: %s\n\nStep results:\n%s\n\nSynthesize a final answer."
)

// NewPlanAndSolveAgent constructs a PlanAndSolveAgent.
func NewPlanAndSolveAgent(model llm.ChatModel, opts PlanAndSolveOptions) *PlanAndSolveAgent {
	if opts.Name == "" {
		opts.Name = "plan-and-solve"
	}
	if opts.MaxSteps == 0 {
		opts.MaxSteps = 8
	}
	if opts.PlanPrompt == "" {
		opts.PlanPrompt = planPromptDefault
	}
	if opts.StepPrompt == "" {
		opts.StepPrompt = stepPromptDefault
	}
	if opts.SynthPrompt == "" {
		opts.SynthPrompt = synthPromptDefault
	}
	return &PlanAndSolveAgent{model: model, opts: opts}
}

// Name implements Agent.
func (a *PlanAndSolveAgent) Name() string { return a.opts.Name }

// Run executes plan → step1..stepN → synthesize.
func (a *PlanAndSolveAgent) Run(ctx context.Context, input string) (Result, error) {
	return a.runInternal(ctx, input, normalizedOnStep(a.opts.OnStep))
}

// RunStream emits step events through a channel; see Agent interface docs.
func (a *PlanAndSolveAgent) RunStream(ctx context.Context, input string) (<-chan StepEvent, error) {
	return runStreamFromBlocking(ctx, func(ctx context.Context, cb func(Step)) (Result, error) {
		return a.runInternal(ctx, input, cb)
	})
}

// runInternal assumes onStep is non-nil (caller normalizes via normalizedOnStep).
func (a *PlanAndSolveAgent) runInternal(ctx context.Context, input string, onStep func(Step)) (Result, error) {
	if strings.TrimSpace(input) == "" {
		return Result{}, ErrEmptyInput
	}
	trace := []Step{}
	usage := Usage{}

	// plan
	planResp, err := generateFromPrompt(ctx, a.model, "", fmt.Sprintf(a.opts.PlanPrompt, input, a.opts.MaxSteps))
	if err != nil {
		return Result{}, err
	}
	usage.LLMCalls++
	usage.Tokens += planResp.Usage.TotalTokens
	steps := parsePlan(planResp.Text)
	if len(steps) == 0 {
		return Result{}, fmt.Errorf("%w: empty plan from %q", ErrPlanningFailed, planResp.Text)
	}
	if len(steps) > a.opts.MaxSteps {
		steps = steps[:a.opts.MaxSteps]
	}
	planStep := Step{Kind: StepPlan, Content: planResp.Text}
	trace = append(trace, planStep)
	onStep(planStep)

	// per-step exec
	results := make([]string, 0, len(steps))
	planText := strings.Join(steps, "\n")
	for i, step := range steps {
		resp, err := generateFromPrompt(ctx, a.model, "", fmt.Sprintf(a.opts.StepPrompt, input, planText, i+1, step))
		if err != nil {
			return Result{}, err
		}
		usage.LLMCalls++
		usage.Tokens += resp.Usage.TotalTokens
		results = append(results, resp.Text)
		thoughtStep := Step{Kind: StepThought, Content: fmt.Sprintf("step %d: %s → %s", i+1, step, resp.Text)}
		trace = append(trace, thoughtStep)
		onStep(thoughtStep)
	}

	// synthesize
	bullets := strings.Builder{}
	for i, r := range results {
		fmt.Fprintf(&bullets, "%d. %s\n", i+1, r)
	}
	synth, err := generateFromPrompt(ctx, a.model, "", fmt.Sprintf(a.opts.SynthPrompt, input, bullets.String()))
	if err != nil {
		return Result{}, err
	}
	usage.LLMCalls++
	usage.Tokens += synth.Usage.TotalTokens

	finalStep := Step{Kind: StepFinal, Content: synth.Text}
	trace = append(trace, finalStep)
	onStep(finalStep)
	return Result{Answer: synth.Text, Trace: trace, Usage: usage}, nil
}

var planLineRE = regexp.MustCompile(`(?m)^\s*\d+\.\s+(.+?)\s*$`)

// parsePlan extracts numbered steps from text. Looks for `<n>. <text>` lines.
// Returns step text in order.
func parsePlan(text string) []string {
	matches := planLineRE.FindAllStringSubmatch(text, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if s := strings.TrimSpace(m[1]); s != "" {
			out = append(out, s)
		}
	}
	return out
}

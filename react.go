package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/costa92/llm-agent-contract/llm"
)

// ReActAgent runs a Thought→Action→Observation loop until the LLM emits a
// "Final:" line or MaxSteps is exceeded.
//
// Output format the LLM is instructed to emit:
//
//	Thought: <reasoning>
//	Action: <tool_name>
//	Args: <json>
//	-- or --
//	Thought: <reasoning>
//	Final: <answer>
type ReActAgent struct {
	model      llm.ChatModel
	toolCaller llm.ToolCaller
	opts       ReActOptions
}

// ReActOptions configures ReActAgent.
type ReActOptions struct {
	Name         string     // default "react"
	Registry     *Registry  // tools the agent can call (nil = no tools, only Final allowed)
	MaxSteps     int        // default 8 — bound on round-trips before ErrMaxStepsExceeded
	SystemPrompt string     // optional override; default = reactSystemPrompt
	OnStep       func(Step) // optional, called for each trace step (synchronous)
}

const reactSystemPrompt = `You are a ReAct agent. On each step you may either call a tool or give a final answer.

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

// NewReActAgent constructs a ReActAgent.
func NewReActAgent(model llm.ChatModel, opts ReActOptions) *ReActAgent {
	if opts.Name == "" {
		opts.Name = "react"
	}
	if opts.MaxSteps == 0 {
		opts.MaxSteps = 8
	}
	if opts.SystemPrompt == "" {
		opts.SystemPrompt = reactSystemPrompt
	}
	tc, _ := nativeToolCaller(model)
	return &ReActAgent{model: model, toolCaller: tc, opts: opts}
}

// Name implements Agent.
func (a *ReActAgent) Name() string { return a.opts.Name }

// Run executes the ReAct loop.
func (a *ReActAgent) Run(ctx context.Context, input string) (Result, error) {
	return a.runInternal(ctx, input, normalizedOnStep(a.opts.OnStep))
}

// RunStream emits step events through a channel; see Agent interface docs.
func (a *ReActAgent) RunStream(ctx context.Context, input string) (<-chan StepEvent, error) {
	return runStreamFromBlocking(ctx, func(ctx context.Context, cb func(Step)) (Result, error) {
		return a.runInternal(ctx, input, cb)
	})
}

// runInternal assumes onStep is non-nil (caller normalizes via normalizedOnStep).
func (a *ReActAgent) runInternal(ctx context.Context, input string, onStep func(Step)) (Result, error) {
	if strings.TrimSpace(input) == "" {
		return Result{}, ErrEmptyInput
	}
	if a.toolCaller != nil && a.opts.Registry != nil {
		return a.runNative(ctx, input, onStep)
	}
	return a.runScratchpad(ctx, input, onStep)
}

func (a *ReActAgent) runScratchpad(ctx context.Context, input string, onStep func(Step)) (Result, error) {
	trace := make([]Step, 0, a.opts.MaxSteps*3)
	usage := Usage{}

	scratchpad := strings.Builder{}
	scratchpad.WriteString(fmt.Sprintf(a.opts.SystemPrompt, a.toolList()))
	scratchpad.WriteString("\nQuestion: ")
	scratchpad.WriteString(input)
	scratchpad.WriteString("\n")

	for step := 0; step < a.opts.MaxSteps; step++ {
		resp, err := generateFromPrompt(ctx, a.model, "", scratchpad.String())
		if err != nil {
			return Result{}, err
		}
		usage.LLMCalls++
		usage.Tokens += resp.Usage.TotalTokens

		thought, action, args, final, perr := parseReAct(resp.Text)
		if perr != nil {
			return Result{}, fmt.Errorf("%w: %v", ErrParseToolCall, perr)
		}
		if thought != "" {
			s := Step{Kind: StepThought, Content: thought}
			trace = append(trace, s)
			onStep(s)
		}
		if final != "" {
			s := Step{Kind: StepFinal, Content: final}
			trace = append(trace, s)
			onStep(s)
			return Result{Answer: final, Trace: trace, Usage: usage}, nil
		}
		// action path
		actionStep := Step{Kind: StepAction, Tool: action, Args: args}
		trace = append(trace, actionStep)
		onStep(actionStep)

		var tool Tool
		var ok bool
		if a.opts.Registry != nil {
			tool, ok = a.opts.Registry.Get(action)
		}
		if !ok {
			return Result{}, fmt.Errorf("%w: %q", ErrToolNotFound, action)
		}
		out, err := tool.Execute(ctx, json.RawMessage(args))
		if err != nil {
			return Result{}, fmt.Errorf("tool %q: %w", action, err)
		}
		obsStep := Step{Kind: StepObservation, Result: out}
		trace = append(trace, obsStep)
		onStep(obsStep)
		scratchpad.WriteString(resp.Text)
		scratchpad.WriteString("\nObservation: ")
		scratchpad.WriteString(out)
		scratchpad.WriteString("\n")
	}
	return Result{}, ErrMaxStepsExceeded
}

func (a *ReActAgent) runNative(ctx context.Context, input string, onStep func(Step)) (Result, error) {
	trace := []Step{}
	toolModel, err := a.toolCaller.WithTools(a.opts.Registry.AsLLMTools())
	if err != nil {
		return Result{}, err
	}
	resp, err := generateFromPrompt(ctx, toolModel, "", input)
	if err != nil {
		return Result{}, err
	}
	usage := Usage{LLMCalls: 1, Tokens: resp.Usage.TotalTokens}

	if len(resp.ToolCalls) == 0 {
		final := Step{Kind: StepFinal, Content: resp.Text}
		trace = append(trace, final)
		onStep(final)
		return Result{Answer: resp.Text, Trace: trace, Usage: usage}, nil
	}

	var outputs []string
	for _, tc := range resp.ToolCalls {
		actionStep := Step{Kind: StepAction, Tool: tc.Name, Args: string(tc.Arguments)}
		trace = append(trace, actionStep)
		onStep(actionStep)

		tool, ok := a.opts.Registry.Get(tc.Name)
		if !ok {
			return Result{}, fmt.Errorf("%w: %q", ErrToolNotFound, tc.Name)
		}
		out, err := tool.Execute(ctx, tc.Arguments)
		if err != nil {
			return Result{}, fmt.Errorf("tool %q: %w", tc.Name, err)
		}
		outputs = append(outputs, out)

		obsStep := Step{Kind: StepObservation, Result: out}
		trace = append(trace, obsStep)
		onStep(obsStep)
	}

	answer := strings.Join(outputs, "\n")
	final := Step{Kind: StepFinal, Content: answer}
	trace = append(trace, final)
	onStep(final)
	return Result{Answer: answer, Trace: trace, Usage: usage}, nil
}

func (a *ReActAgent) toolList() string {
	if a.opts.Registry == nil {
		return "(none)\n"
	}
	return a.opts.Registry.PromptDescription()
}

// parseReAct scans for "Thought:", "Action:", "Args:", "Final:" lines.
// Returns (thought, action, args, final, err). Either action+args or final
// must be present (not both, not neither).
func parseReAct(text string) (thought, action, args, final string, err error) {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, "\r")
		switch {
		case strings.HasPrefix(line, "Thought:"):
			thought = strings.TrimSpace(strings.TrimPrefix(line, "Thought:"))
		case strings.HasPrefix(line, "Action:"):
			action = strings.TrimSpace(strings.TrimPrefix(line, "Action:"))
		case strings.HasPrefix(line, "Args:"):
			args = strings.TrimSpace(strings.TrimPrefix(line, "Args:"))
		case strings.HasPrefix(line, "Final:"):
			final = strings.TrimSpace(strings.TrimPrefix(line, "Final:"))
		}
	}
	if final != "" {
		return
	}
	if action == "" {
		err = fmt.Errorf("missing Action or Final in: %q", text)
		return
	}
	if args == "" {
		args = "{}"
	}
	return
}

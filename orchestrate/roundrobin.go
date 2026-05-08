package orchestrate

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/costa92/llm-agent"
)

// RoundRobinOptions configures a RoundRobinChat. Termination is the
// preferred stop signal; MaxTurns is a hard cap to prevent runaway
// loops if Termination is forgotten.
type RoundRobinOptions struct {
	Termination Termination // optional; default = MaxTurns(MaxTurns) only
	MaxTurns    int         // hard cap; default 20
}

// RoundRobinChat runs N agents who take turns speaking. Each turn:
// the current speaker reads the prior history + a task header and
// produces a Message. Loop continues until Termination signals stop
// or MaxTurns is hit.
//
// Mirrors AutoGen's RoundRobinGroupChat (Python). Useful for "two
// agents critique each other" or "writer + editor + fact-checker"
// emergent patterns.
type RoundRobinChat struct {
	name   string
	agents []agents.Agent
	opts   RoundRobinOptions
}

// ChatResult carries the full conversation history + accumulated usage.
// Stopped distinguishes natural termination from cap / cancel.
type ChatResult struct {
	History []Message
	Usage   agents.Usage
	Stopped string // "termination" | "max_turns" | "ctx_cancel"
}

// NewRoundRobinChat constructs a RoundRobinChat. Empty agents slice is
// allowed but Run returns ErrNoAgents.
func NewRoundRobinChat(name string, ag []agents.Agent, opts RoundRobinOptions) *RoundRobinChat {
	if opts.MaxTurns <= 0 {
		opts.MaxTurns = 20
	}
	return &RoundRobinChat{name: name, agents: ag, opts: opts}
}

// Name returns the chat name (for logs).
func (r *RoundRobinChat) Name() string {
	if r.name == "" {
		return "roundrobin"
	}
	return r.name
}

// Run executes the round-robin loop. task is the initial prompt.
// Returns ErrNoAgents if zero agents were registered.
func (r *RoundRobinChat) Run(ctx context.Context, task string) (ChatResult, error) {
	if len(r.agents) == 0 {
		return ChatResult{}, ErrNoAgents
	}

	res := ChatResult{History: make([]Message, 0, r.opts.MaxTurns)}
	for turn := 0; turn < r.opts.MaxTurns; turn++ {
		select {
		case <-ctx.Done():
			res.Stopped = "ctx_cancel"
			return res, nil
		default:
		}

		speaker := r.agents[turn%len(r.agents)]
		prompt := buildRoundRobinPrompt(task, res.History)
		out, err := speaker.Run(ctx, prompt)
		if err != nil {
			return res, fmt.Errorf("roundrobin %q: %s turn %d: %w", r.Name(), speaker.Name(), turn, err)
		}
		res.History = append(res.History, Message{Speaker: speaker.Name(), Content: out.Answer})
		res.Usage.LLMCalls += out.Usage.LLMCalls
		res.Usage.Tokens += out.Usage.Tokens

		if r.opts.Termination != nil && r.opts.Termination.ShouldStop(res.History) {
			res.Stopped = "termination"
			return res, nil
		}
	}
	res.Stopped = "max_turns"
	return res, nil
}

// buildRoundRobinPrompt formats prior history + task header. Each
// speaker sees: <history\n><blank line>Task: <task>.
func buildRoundRobinPrompt(task string, history []Message) string {
	if len(history) == 0 {
		return "Task: " + task
	}
	var b strings.Builder
	for _, m := range history {
		fmt.Fprintf(&b, "%s: %s\n", m.Speaker, m.Content)
	}
	b.WriteString("\nTask: ")
	b.WriteString(task)
	return b.String()
}

// ErrNoAgents is returned by Run when the agents slice is empty.
var ErrNoAgents = errors.New("orchestrate: roundrobin chat has no agents")

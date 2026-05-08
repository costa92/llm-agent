package orchestrate

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/costa92/llm-agent"
)

// RolePlayOptions configures a RolePlay duo. DoneMarker is the sentinel
// the assistant emits to signal task completion (CAMEL convention:
// "<TASK_DONE>"). MaxTurns is the hard cap on user/assistant exchanges.
type RolePlayOptions struct {
	DoneMarker string // default "<TASK_DONE>"
	MaxTurns   int    // default 30
	InitPrompt string // optional override for the first user prompt; default is CAMEL Inception Prompting
}

// RolePlay runs a CAMEL-style 2-agent dialog: the user agent issues
// instructions, the assistant agent executes. They alternate until
// the assistant emits DoneMarker or MaxTurns is hit.
//
// This is NOT N-agent (use RoundRobinChat for that). RolePlay's value
// is the structured user→assistant→user pattern with explicit "done"
// signaling, useful for task-decomposition-style work.
type RolePlay struct {
	user       agents.Agent
	assistant  agents.Agent
	taskPrompt string
	opts       RolePlayOptions
}

// RolePlayTurn pairs one user message with the assistant's reply.
type RolePlayTurn struct {
	UserMsg      string
	AssistantMsg string
}

// RolePlayResult carries the full dialog + Concluded flag (true when
// DoneMarker hit) + FinalOutput (the assistant's last message).
type RolePlayResult struct {
	Turns       []RolePlayTurn
	Concluded   bool
	FinalOutput string
	Usage       agents.Usage
}

// NewRolePlay constructs a RolePlay duo. Both agents are required.
func NewRolePlay(user, assistant agents.Agent, taskPrompt string, opts RolePlayOptions) *RolePlay {
	if opts.DoneMarker == "" {
		opts.DoneMarker = "<TASK_DONE>"
	}
	if opts.MaxTurns <= 0 {
		opts.MaxTurns = 30
	}
	return &RolePlay{user: user, assistant: assistant, taskPrompt: taskPrompt, opts: opts}
}

// inceptionPrompt is the default first prompt to the user agent
// (CAMEL §3 Inception Prompting). It frames the user as the
// task-issuer and the assistant as the executor.
const inceptionPrompt = `You are the USER in a role-play. Your job is to break the task below into clear instructions for the ASSISTANT.

Issue exactly ONE instruction at a time. Wait for the assistant's reply before continuing.
When the task is fully complete, reply with: <TASK_DONE>

Task:
%s`

// Run executes the role-play loop. Returns ErrNilAgent if either
// agent is nil. ctx-cancel exits gracefully (Concluded=false).
func (r *RolePlay) Run(ctx context.Context) (RolePlayResult, error) {
	if r.user == nil || r.assistant == nil {
		return RolePlayResult{}, ErrNilAgent
	}

	res := RolePlayResult{Turns: make([]RolePlayTurn, 0, r.opts.MaxTurns)}

	// First user prompt: framing + task
	initPrompt := r.opts.InitPrompt
	if initPrompt == "" {
		initPrompt = fmt.Sprintf(inceptionPrompt, r.taskPrompt)
	}
	currentUserInput := initPrompt

	for turn := 0; turn < r.opts.MaxTurns; turn++ {
		select {
		case <-ctx.Done():
			return res, nil
		default:
		}

		// User agent speaks
		userOut, err := r.user.Run(ctx, currentUserInput)
		if err != nil {
			return res, fmt.Errorf("roleplay: user turn %d: %w", turn, err)
		}
		res.Usage.LLMCalls += userOut.Usage.LLMCalls
		res.Usage.Tokens += userOut.Usage.Tokens

		// Check user-side DoneMarker (user can also call task done)
		if strings.Contains(userOut.Answer, r.opts.DoneMarker) {
			res.Turns = append(res.Turns, RolePlayTurn{UserMsg: userOut.Answer})
			res.Concluded = true
			return res, nil
		}

		// Assistant agent replies
		assistantOut, err := r.assistant.Run(ctx, userOut.Answer)
		if err != nil {
			return res, fmt.Errorf("roleplay: assistant turn %d: %w", turn, err)
		}
		res.Usage.LLMCalls += assistantOut.Usage.LLMCalls
		res.Usage.Tokens += assistantOut.Usage.Tokens

		res.Turns = append(res.Turns, RolePlayTurn{
			UserMsg:      userOut.Answer,
			AssistantMsg: assistantOut.Answer,
		})
		res.FinalOutput = assistantOut.Answer

		if strings.Contains(assistantOut.Answer, r.opts.DoneMarker) {
			res.Concluded = true
			return res, nil
		}

		// Next user prompt = assistant's last reply (forms the conversation)
		currentUserInput = assistantOut.Answer
	}
	return res, nil
}

// ErrNilAgent is returned by Run when user or assistant is nil.
var ErrNilAgent = errors.New("orchestrate: roleplay requires non-nil user and assistant agents")

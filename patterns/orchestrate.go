package patterns

import (
	"fmt"
	"strings"

	agents "github.com/costa92/llm-agent"
	"github.com/costa92/llm-agent/orchestrate"
)

// SupervisorOptions configures BuildSupervisor.
type SupervisorOptions struct {
	Name           string
	Planner        agents.Agent
	Workers        map[string]agents.Agent
	MaxRounds      int
	ParseDispatch  orchestrate.DispatchParser
	BuildAggregate orchestrate.Aggregator
}

// BuildSupervisor constructs the productized supervisor pattern.
func BuildSupervisor(opts SupervisorOptions) *orchestrate.Supervisor {
	maxRounds := opts.MaxRounds
	if maxRounds <= 0 {
		maxRounds = 4
	}
	parser := opts.ParseDispatch
	if parser == nil {
		parser = ParseDispatchLine
	}
	aggregate := opts.BuildAggregate
	if aggregate == nil {
		aggregate = JoinWorkerResults
	}
	return orchestrate.NewSupervisor(opts.Name, orchestrate.SupervisorOptions{
		Planner:        opts.Planner,
		Workers:        opts.Workers,
		MaxRounds:      maxRounds,
		ParseDispatch:  parser,
		BuildAggregate: aggregate,
	})
}

// FanOutOptions configures BuildFanOutFanIn.
type FanOutOptions struct {
	Name           string
	Planner        agents.Agent
	Workers        map[string]agents.Agent
	Aggregator     agents.Agent
	MaxConcurrency int
	ParsePlan      orchestrate.PlanParser
	BuildAggregate orchestrate.AggregateInputBuilder
}

// BuildFanOutFanIn constructs the productized fan-out/fan-in pattern.
func BuildFanOutFanIn(opts FanOutOptions) *orchestrate.FanOutFanIn {
	return orchestrate.NewFanOutFanIn(opts.Name, orchestrate.FanOutFanInOptions{
		Planner:        opts.Planner,
		Workers:        opts.Workers,
		Aggregator:     opts.Aggregator,
		ParsePlan:      opts.ParsePlan,
		BuildAggregate: opts.BuildAggregate,
		MaxConcurrency: opts.MaxConcurrency,
	})
}

// RoundRobinOptions configures BuildRoundRobin.
type RoundRobinOptions struct {
	Name        string
	Agents      []agents.Agent
	MaxTurns    int
	Termination orchestrate.Termination
}

// BuildRoundRobin constructs the productized round-robin chat pattern.
func BuildRoundRobin(opts RoundRobinOptions) *orchestrate.RoundRobinChat {
	return orchestrate.NewRoundRobinChat(opts.Name, opts.Agents, orchestrate.RoundRobinOptions{
		MaxTurns:    opts.MaxTurns,
		Termination: opts.Termination,
	})
}

// RolePlayOptions configures BuildRolePlay.
type RolePlayOptions struct {
	User       agents.Agent
	Assistant  agents.Agent
	TaskPrompt string
	DoneMarker string
	MaxTurns   int
	InitPrompt string
}

// BuildRolePlay constructs the productized role-play pattern.
func BuildRolePlay(opts RolePlayOptions) *orchestrate.RolePlay {
	return orchestrate.NewRolePlay(opts.User, opts.Assistant, opts.TaskPrompt, orchestrate.RolePlayOptions{
		DoneMarker: opts.DoneMarker,
		MaxTurns:   opts.MaxTurns,
		InitPrompt: opts.InitPrompt,
	})
}

// ParseDispatchLine parses "dispatch to <worker>: <input>" planner output.
// Empty output or "FINISH" terminates the supervisor cleanly.
func ParseDispatchLine(plannerAnswer string) (*orchestrate.Dispatch, error) {
	trimmed := strings.TrimSpace(plannerAnswer)
	switch {
	case trimmed == "":
		return nil, nil
	case strings.EqualFold(trimmed, "FINISH"):
		return nil, nil
	case strings.HasPrefix(strings.ToLower(trimmed), "dispatch to "):
		rest := strings.TrimSpace(trimmed[len("dispatch to "):])
		parts := strings.SplitN(rest, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid dispatch %q", plannerAnswer)
		}
		worker := strings.TrimSpace(parts[0])
		input := strings.TrimSpace(parts[1])
		if worker == "" || input == "" {
			return nil, fmt.Errorf("invalid dispatch %q", plannerAnswer)
		}
		return &orchestrate.Dispatch{WorkerName: worker, Input: input}, nil
	default:
		return nil, fmt.Errorf("invalid dispatch %q", plannerAnswer)
	}
}

// JoinWorkerResults builds a stable final answer from supervisor worker results.
func JoinWorkerResults(results []orchestrate.WorkerResult) (string, error) {
	if len(results) == 0 {
		return "", nil
	}
	parts := make([]string, 0, len(results))
	for _, wr := range results {
		parts = append(parts, fmt.Sprintf("%s(%s)=%s", wr.Dispatch.WorkerName, wr.Dispatch.Input, wr.Result.Answer))
	}
	return strings.Join(parts, " | "), nil
}

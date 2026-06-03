package agents

// This file re-exports the agent contract surface that now lives in the leaf
// module github.com/costa92/llm-agent-contract/agents. The aliases preserve type
// identity (Go type aliases are the SAME type), so every concrete Agent paradigm,
// the Registry, NewFuncTool/NewAsyncRunner, and all external consumers keep
// compiling against the bare `agents.X` names unchanged. Mirrors
// llm-agent/memory/aliases.go.
//
// Concrete impls/constructors (SimpleAgent/ReAct/Reflection/PlanAndSolve/
// FunctionCall, Registry, NewFuncTool, NewAsyncRunner, AsLLMTool) stay in this
// package — they are NOT re-exported, they are defined here.

import contractagents "github.com/costa92/llm-agent-contract/agents"

// --- interface + data type aliases ---
type (
	Agent       = contractagents.Agent
	Tool        = contractagents.Tool
	ExecuteFunc = contractagents.ExecuteFunc
	Task        = contractagents.Task
	Result      = contractagents.Result
	Usage       = contractagents.Usage
	Step        = contractagents.Step
	StepEvent   = contractagents.StepEvent
	StepKind    = contractagents.StepKind
)

// --- StepKind const re-exports (all 6) ---
const (
	StepThought     = contractagents.StepThought
	StepAction      = contractagents.StepAction
	StepObservation = contractagents.StepObservation
	StepReflection  = contractagents.StepReflection
	StepPlan        = contractagents.StepPlan
	StepFinal       = contractagents.StepFinal
)

// --- sentinel re-exports (identity-preserving for errors.Is across modules) ---
var (
	ErrToolNotFound = contractagents.ErrToolNotFound
	ErrEmptyInput   = contractagents.ErrEmptyInput
)

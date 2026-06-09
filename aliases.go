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
	Agent               = contractagents.Agent
	Tool                = contractagents.Tool
	ExecuteFunc         = contractagents.ExecuteFunc
	Task                = contractagents.Task
	Result              = contractagents.Result
	Usage               = contractagents.Usage
	Step                = contractagents.Step
	StepEvent           = contractagents.StepEvent
	StepKind            = contractagents.StepKind
	RunEvent            = contractagents.RunEvent
	RunEventKind        = contractagents.RunEventKind
	Callback            = contractagents.Callback
	CallbackFunc        = contractagents.CallbackFunc
	RunEventHandler     = contractagents.RunEventHandler
	RunEventHandlerFunc = contractagents.RunEventHandlerFunc
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

// --- RunEventKind const re-exports ---
const (
	RunEventAgentStarted    = contractagents.RunEventAgentStarted
	RunEventAgentStep       = contractagents.RunEventAgentStep
	RunEventModelDelta      = contractagents.RunEventModelDelta
	RunEventModelDone       = contractagents.RunEventModelDone
	RunEventToolCall        = contractagents.RunEventToolCall
	RunEventToolResult      = contractagents.RunEventToolResult
	RunEventHandoff         = contractagents.RunEventHandoff
	RunEventInterrupt       = contractagents.RunEventInterrupt
	RunEventSuspended       = contractagents.RunEventSuspended
	RunEventAgentDone       = contractagents.RunEventAgentDone
	RunEventAgentError      = contractagents.RunEventAgentError
	RunEventFlowStarted     = contractagents.RunEventFlowStarted
	RunEventFlowNodeStarted = contractagents.RunEventFlowNodeStarted
	RunEventFlowNodeDone    = contractagents.RunEventFlowNodeDone
	RunEventFlowNodeSkipped = contractagents.RunEventFlowNodeSkipped
	RunEventFlowDone        = contractagents.RunEventFlowDone
	RunEventFlowError       = contractagents.RunEventFlowError
)

// --- sentinel re-exports (identity-preserving for errors.Is across modules) ---
var (
	ErrToolNotFound = contractagents.ErrToolNotFound
	ErrEmptyInput   = contractagents.ErrEmptyInput
)

// --- helper re-exports ---
var (
	RunEventFromStepEvent   = contractagents.RunEventFromStepEvent
	RunEventFromStreamEvent = contractagents.RunEventFromStreamEvent
	EmitRunEvent            = contractagents.EmitRunEvent
	ChainCallbacks          = contractagents.ChainCallbacks
	NoopCallback            = contractagents.NoopCallback
	MultiRunEventHandler    = contractagents.MultiRunEventHandler
)

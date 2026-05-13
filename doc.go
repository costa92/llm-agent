// Package agents implements five Agent paradigms (Simple, ReAct, Reflection,
// Plan-and-Solve, FunctionCall) and a Tool subsystem (Registry, Chain,
// AsyncRunner) on top of the llm subpackage.
//
// # Portability contract
//
// agents is a stdlib-only Go module. Constraints:
//
//   - Zero third-party Go dependencies
//   - No business vocabulary (tenant, faq, kb, channel, ...)
//   - Sentinel errors only — callers translate to their project taxonomy via errors.Is
//
// # Quick start
//
//	model := llm.NewScriptedLLM(llm.WithResponses(llm.TextResponse("Hello")))
//	agent := agents.NewSimpleAgent(model, agents.SimpleOptions{})
//	res, err := agent.Run(ctx, "Hello")
//
// See example_simple_test.go / example_tool_use_test.go /
// example_multi_agent_test.go for runnable examples covering all five
// agents and the tool subsystem.
//
// # Agents
//
//   - SimpleAgent: single-shot LLM forward.
//   - ReActAgent: Thought → Action → Observation loop, parses LLM output for
//     "Action: <tool>" / "Args: <json>" / "Final: <answer>" lines.
//   - ReflectionAgent: gen → critique → revise loop, exits early on
//     critique containing "APPROVED".
//   - PlanAndSolveAgent: plan once (numbered steps), execute each step,
//     synthesize final answer.
//   - FunctionCallAgent: single-turn native function-calling — uses
//     pkg/llm.Tool / resp.ToolCalls instead of prompt parsing; AsyncRunner
//     for parallel tool execution with bounded parallelism.
//
// # Observability
//
// Every Agent exposes two observation channels:
//
//   - Options.OnStep func(Step): synchronous callback fired at each trace
//     step. Lowest overhead; use for in-process logging.
//   - RunStream(ctx, input) (<-chan StepEvent, error): channel-based
//     streaming for cross-boundary consumers (HTTP SSE, gRPC stream).
//     The channel is closed when the agent finishes; final event has
//     Done=true with either Final or Err set.
//
// # Tools
//
//   - Tool interface: Name / Description / Schema / Execute.
//   - NewFuncTool: wrap a plain function as a Tool without writing a struct.
//   - Registry: name→Tool with sorted List, AsLLMTools, PromptDescription helpers.
//   - Chain: pipes Tools sequentially; itself satisfies Tool.
//   - AsyncRunner: parallel Task execution with ctx cancellation; per-Task
//     errors do not abort siblings.
//
// Built-in tools live in the builtin subpackage (Calculator, MockSearch).
package agents

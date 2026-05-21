// Package orchestrate implements 4 multi-Agent orchestration paradigms on
// top of pkg/llm/agents.Agent.
//
// 中文说明：Package orchestrate 在 pkg/llm/agents.Agent 之上实现多
// Agent 编排能力。
//
//   - Pipeline       — A → B → C linear data-flow (AgentScope-style)
//   - FanOutFanIn    — planner → parallel workers → aggregator
//   - RoundRobinChat — N agents take turns until termination (AutoGen-style)
//   - RolePlay       — user + assistant 2-agent dialog with done-marker (CAMEL-style)
//   - StateGraph[S]  — typed graph with conditional edges + cycles (LangGraph-style, Go generics)
//   - Supervisor     — planner / worker loop on top of StateGraph[S] (KC-1)
//
// 中文说明：
//   - Pipeline       — A → B → C 线性数据流（AgentScope 风格）
//   - FanOutFanIn    — planner → 并行 workers → aggregator
//   - RoundRobinChat — N 个 agent 轮流对话直到终止（AutoGen 风格）
//   - RolePlay       — user + assistant 的双 agent 对话，带 done-marker（CAMEL 风格）
//   - StateGraph[S]  — 带条件边和循环的 typed graph（LangGraph 风格，Go 泛型）
//   - Supervisor     — 建立在 StateGraph[S] 上的 planner / worker 循环（KC-1）
//
// Plus a shared Termination interface + 4 combinators (MaxTurns / TextMatch / And / Or).
//
// # Portability
//
// orchestrate inherits the agents/pkg/llm portability contract — no imports
// from internal/*, no project-specific pkg/*, no business vocabulary. Pure
// framework code reusable in any Go project that pulls in pkg/llm/agents.
//
// 中文说明：orchestrate 继承 agents/pkg/llm 的可移植性约束，不能 import
// internal/*，不能依赖项目专属 pkg/*，不能使用业务词汇。这里的代码应当是纯
// 框架层，可以被任何引入 pkg/llm/agents 的 Go 项目复用。
//
// # Choosing a paradigm
//
//   - Single-Agent loop with retries → don't use orchestrate; use Phase 1's ReAct/Reflection directly
//   - Linear A→B→C with no branching → Pipeline
//   - Planner → parallel specialists → summary → FanOutFanIn
//   - Planner → worker dispatch loop with explicit rounds → Supervisor
//   - "Two writers + an editor" emergent collaboration → RoundRobinChat
//   - Task decomposition with explicit done signal → RolePlay
//   - Explicit branches/loops/observable state → StateGraph
//
// 中文选型：
//   - 单 Agent 重试循环 → 不用 orchestrate，直接用 Phase 1 的 ReAct/Reflection
//   - 线性 A→B→C 且无分支 → Pipeline
//   - planner → 多个 specialist 并行 → summary → FanOutFanIn
//   - planner → 带显式轮次的 worker 分发循环 → Supervisor
//   - 两个 writer + 一个 editor 的协作 → RoundRobinChat
//   - 带明确完成信号的任务分解 → RolePlay
//   - 需要分支 / 循环 / 可观测状态 → StateGraph
//
// # StateGraph guidance
//
// StateGraph is the right fit when a multi-agent workflow needs explicit,
// testable control flow:
//
//   - planner → reviewer → either revise or finish
//   - dispatcher loops over a backlog until no tasks remain
//   - supervisor routes shared state to different specialists
//
// Keep the state shape as your domain struct, put work inside node funcs,
// and use conditional edges only where routing is needed. StateGraph is
// intentionally small: it's a typed orchestration primitive, not a new DSL.
//
// 中文说明：StateGraph 适合需要显式、可测试控制流的多 agent 工作流。
// 状态建议直接使用领域 struct，把工作放到 node 函数里，只在需要路由时使用
// conditional edges。StateGraph 故意保持很小，它是 typed orchestration
// primitive，不是新的 DSL。
//
// Full design notes and Mermaid flow charts live in:
//
// 完整设计说明和 Mermaid 流程图见：
//
//	docs/superpowers/specs/2026-05-06-pkg-llm-agents-multi-agent-orchestration-design.md
//
// Entry can be declared with either SetEntry("node") or AddEdge(NodeStart, "node").
// Use WithMaxSteps on Run when loops are intentional and need a larger guard-rail.
//
// # Integration with Phase 8 (Deep Research)
//
// internal/research.Coordinator hand-rolls plan→summarize×N→report today. It can
// be refactored to use Pipeline for the plan+report ends (the parallel summarize×N
// middle still uses agents.AsyncRunner).
// Migration is optional — direct orchestration is fine for a 3-stage pipeline.
package orchestrate

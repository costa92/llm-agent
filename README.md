# agents — Go LLM agents framework

`agents` 是一个 Go 版 LLM agents 框架，强调 stdlib-only、可组合、可测试。

A standalone, **stdlib-only** Go module providing the building blocks for LLM-driven agents: the 5 classic agent paradigms (Simple, ReAct, Reflection, Plan-and-Solve, FunctionCall), Memory (Working/Episodic/Semantic), RAG, Context engineering, communication protocols (MCP / A2A / ANP), multi-agent orchestration (Pipeline, FanOutFanIn, RoundRobin, RolePlay, StateGraph), Agentic-RL evaluation, and benchmarks (BFCL, GAIA, LLM-as-Judge, Win Rate).
它提供 LLM agents 的基础组件：5 种经典 agent 范式、Memory、RAG、Context 工程、通信协议、多 agent 编排、Agentic-RL 评估和 benchmark。

> **v0.2.0 — 学习 / 原型 stage.** API may break between 0.x releases.
> Wait for **v1.0** for any stability commitment.
> Original design specs(in the parent AICS repo where the framework was incubated):
> [`2026-04-27-pkg-llm-agents-design.md`](https://github.com/costa92/ai-customer-service/blob/main/docs/superpowers/specs/2026-04-27-pkg-llm-agents-design.md)
> · [`2026-05-06-pkg-llm-agents-multi-agent-orchestration-design.md`](https://github.com/costa92/ai-customer-service/blob/main/docs/superpowers/specs/2026-05-06-pkg-llm-agents-multi-agent-orchestration-design.md)

## Install / 安装

```bash
go get github.com/costa92/llm-agent@latest
```

> **Runnable demos:** five end-to-end examples live in
> [`./examples/`](./examples) — each is a standalone `go run .`-able program
> with a deterministic mock LLM, no API key required. See
> [`examples/README.md`](./examples/README.md) for the menu.

## Quick start / 快速开始

````go
package main

import (
	"context"
	"fmt"

	"github.com/costa92/llm-agent"
	"github.com/costa92/llm-agent/builtin"
	"github.com/costa92/llm-agent/llm"
)

func main() {
	model := llm.NewScriptedLLM(
		llm.WithProvider("scripted"),
		llm.WithModel("calculator-demo"),
		llm.WithCapabilities(llm.Capabilities{Tools: true}),
		llm.WithResponses(llm.ToolCallResponse("calculator", `{"expr":"12*8"}`)),
	)

	reg := agents.NewRegistry(builtin.NewCalculator())
	a, _ := agents.NewFunctionCallAgent(model, agents.FunctionCallOptions{Registry: reg})
	res, _ := a.Run(context.Background(), "What is 12 times 8?")
	fmt.Println(res.Answer)
}
````

## Packages / 包结构

| Package | Purpose |
|---|---|
| `agents` | Agent / Tool interface + 5 paradigm constructors (Simple/ReAct/Reflection/PlanAndSolve/FunctionCall) + Chain + Async + Registry |
| `agents/llm` | LLM contract: ChatModel, ToolCaller, Request/Response, StreamReader/Event, Message, Tool, ToolCall, Usage, FinishReason |
| `agents/builtin` | Calculator, MockSearch, NoteTool, TerminalTool |
| `agents/memory` | WorkingMemory, EpisodicMemory, SemanticMemory, Manager, MemoryTool |
| `agents/rag` | Embedder (HashEmbedder), Chunker, InMemoryStore, RAGSystem, MQE, HyDE |
| `agents/context` | GSSC ContextBuilder (Gather → Select → Structure → Compress) |
| `agents/comm` | Envelope + Transport (HTTP, Stdio) |
| `agents/comm/mcp` | Model Context Protocol client/server (toy spec coverage) |
| `agents/comm/a2a` | Agent-to-Agent task lifecycle |
| `agents/comm/anp` | Agent Network Protocol (in-memory registry) |
| `agents/orchestrate` | Pipeline, FanOutFanIn, RoundRobinChat, RolePlay, StateGraph, Termination |
| `agents/rl` | Dataset, Trajectory, Reward, Evaluator, TrainerProxy (no training — Python TRL bridge) |
| `agents/bench` | BFCL, GAIA, LLM-as-Judge, Win Rate, Reporter (mini fixtures only) |

## Choosing orchestration / 编排选型

- Use `Pipeline` for fixed linear handoffs like `research -> summarize -> answer`.
- Use `FanOutFanIn` when one planner should decompose work, several workers can execute in parallel, and one aggregator should combine the results.
- Use `RoundRobinChat` when several agents should converse until a termination rule fires.
- Use `RolePlay` for a strict 2-agent task/delegate loop with a done marker.
- Use `StateGraph` when the workflow must branch, loop, or route through shared typed state.

- `Pipeline` 适合固定的线性交接，例如 `research -> summarize -> answer`。
- `FanOutFanIn` 适合一个 planner 拆任务、多个 worker 并行执行、一个 aggregator 汇总结果。
- `RoundRobinChat` 适合多个 agent 轮流对话，直到终止条件触发。
- `RolePlay` 适合严格的双 agent 任务/委派循环，并且有 done marker。
- `StateGraph` 适合需要分支、循环、或基于共享状态路由的工作流。

Minimal `FanOutFanIn` sketch:

```go
planner := agents.NewSimpleAgent(plannerClient, agents.SimpleOptions{Name: "planner"})
worker := agents.NewSimpleAgent(workerClient, agents.SimpleOptions{Name: "worker"})
aggregator := agents.NewSimpleAgent(aggregatorClient, agents.SimpleOptions{Name: "aggregator"})

flow := orchestrate.NewFanOutFanIn("research-team", orchestrate.FanOutFanInOptions{
    Planner: planner,
    Workers: map[string]agents.Agent{"worker": worker},
    Aggregator: aggregator,
    // Nil ParsePlan defaults to one task per non-empty line in planner output.
})
```

Business-style fit / 业务场景示例：

- support triage: planner splits `policy check` and `risk check`, specialists run in parallel, aggregator drafts the final reply
- incident analysis: planner emits `timeline`, `impact`, `mitigation` tasks, then one summarizer merges the findings
- research brief: planner splits API changes and test changes, then one aggregator turns specialist notes into a release summary
- return policy: planner splits eligibility, logistics, and customer messaging checks, then one aggregator drafts the final reply
- fraud review: planner splits transaction pattern, account history, and policy checks, then one aggregator returns a manual-review decision
- knowledge lookup: planner splits docs, release notes, and FAQ checks, then one aggregator returns one retrieval answer

`StateGraph` is intentionally small. It gives you named nodes, unconditional edges, conditional edges, and a per-run step cap via `WithMaxSteps(...)`. That makes it a good fit for supervisor / specialist workflows, reviewer-revise loops, and backlog-style dispatch, without introducing a separate orchestration DSL.
`StateGraph` 的设计刻意保持很小，只提供命名节点、无条件边、条件边和 `WithMaxSteps(...)` 这类基础能力。它适合 supervisor / specialist 流程、review-revise 循环、backlog 分发等场景，不引入额外的 orchestration DSL。

For the full architecture and flow charts, see [`docs/superpowers/specs/2026-05-06-pkg-llm-agents-multi-agent-orchestration-design.md`](https://github.com/costa92/ai-customer-service/blob/main/docs/superpowers/specs/2026-05-06-pkg-llm-agents-multi-agent-orchestration-design.md).
完整架构图和流程图见 [`docs/superpowers/specs/2026-05-06-pkg-llm-agents-multi-agent-orchestration-design.md`](https://github.com/costa92/ai-customer-service/blob/main/docs/superpowers/specs/2026-05-06-pkg-llm-agents-multi-agent-orchestration-design.md)。

Minimal `StateGraph` sketch:

```go
type ReviewState struct {
    Tasks []string
    Done  []string
}

g := orchestrate.NewStateGraph[ReviewState]()
g.AddNode("dispatch", func(_ context.Context, s ReviewState) (ReviewState, error) {
    s.Done = append(s.Done, s.Tasks[len(s.Done)])
    return s, nil
})
g.AddNode("finalize", func(_ context.Context, s ReviewState) (ReviewState, error) {
    return s, nil
})
g.AddEdge(orchestrate.NodeStart, "dispatch")
g.AddConditionalEdge("dispatch", func(s ReviewState) string {
    if len(s.Done) >= len(s.Tasks) {
        return "finalize"
    }
    return "dispatch"
})
g.AddEdge("finalize", orchestrate.NodeEnd)
```

Business-style fit:

- support escalation: route to `auto-reply`, `request-more-info`, or `handover-human` based on shared state
- compliance review: loop between `draft` and `review` nodes until violations are cleared
- ticket dispatch: classify incoming work into auto-handled vs human-handled routes
- warranty escalation: route eligible cases to repair intake and out-of-warranty cases to a human review queue
- fraud hold: screen a transaction and route high-risk cases to manual review
- knowledge lookup: collect missing product context, search docs, and then answer
- knowledge lookup with clarification: collect product and version first, then search and answer
- human review reseek: route a chargeback case through human review, context collection, and a second search before answering

Scenario sources:

- `Example_pipeline`, `Example_fanOutFanIn`, `Example_fanOutFanIn_supportTriage`, `Example_fanOutFanIn_incidentAnalysis`, `Example_fanOutFanIn_researchBrief`, `Example_fanOutFanIn_returnPolicy`, `Example_fanOutFanIn_fraudReview`, `Example_fanOutFanIn_knowledgeLookup` in [`example_multi_agent_test.go`](./example_multi_agent_test.go)
- `ExampleStateGraph_loop`, `ExampleStateGraph_supportEscalation`, `ExampleStateGraph_ticketDispatch`, `ExampleStateGraph_complianceReview`, `ExampleStateGraph_warrantyEscalation`, `ExampleStateGraph_fraudHold`, `ExampleStateGraph_knowledgeLookup`, `ExampleStateGraph_knowledgeLookup_clarifySearchAnswer`, `ExampleStateGraph_humanReviewReseek` in [`orchestrate/graph_test.go`](./orchestrate/graph_test.go)

Support-escalation style sketch:

```go
type TriageState struct {
    Question     string
    NeedMoreInfo bool
    NeedsHuman   bool
    Reply        string
}

g := orchestrate.NewStateGraph[TriageState]()
g.AddNode("classify", func(_ context.Context, s TriageState) (TriageState, error) {
    if strings.Contains(strings.ToLower(s.Question), "chargeback") {
        s.NeedsHuman = true
    }
    return s, nil
})
g.AddNode("auto-reply", func(_ context.Context, s TriageState) (TriageState, error) {
    s.Reply = "Standard refund flow is available."
    return s, nil
})
g.AddNode("handover-human", func(_ context.Context, s TriageState) (TriageState, error) {
    s.Reply = "Escalating to a human agent."
    return s, nil
})
g.AddEdge(orchestrate.NodeStart, "classify")
g.AddConditionalEdge("classify", func(s TriageState) string {
    if s.NeedsHuman {
        return "handover-human"
    }
    return "auto-reply"
})
g.AddEdge("auto-reply", orchestrate.NodeEnd)
g.AddEdge("handover-human", orchestrate.NodeEnd)
```

## Local development / 本地开发

This is a standalone Go module — clone, edit, test:

```bash
git clone git@github.com:costa92/llm-agent.git
cd llm-agent
go vet ./...
go test ./...
```

To pick up local edits from a downstream consumer (e.g. while iterating on
both this module and a service that imports it), add a `replace` to the
consumer's `go.mod`:

```
replace github.com/costa92/llm-agent => /local/path/to/llm-agent
```

The runnable demos under [`./examples/`](./examples) already do this so they
stay in sync with whatever is checked out in the parent directory.

`go.work` is gitignored. CI and external `go get` callers rely on the `require` directive in the parent module's `go.mod` resolving to the published tag.

## Status & roadmap / 状态与路线图

- ✅ All 9 design phases shipped (see source spec)
- ✅ 12 packages, ~5000 LOC, stdlib only
- ✅ `examples/agents-demo` (in parent AICS repo) end-to-end runnable
- ⏸ v1.0 — pending real-world feedback

## Operational notes

- Multi-repo PR governance design across `llm-agent`, `llm-agent-providers`,
  `llm-agent-otel`, and `llm-agent-customer-support`:
  [`docs/PR-GOVERNANCE-OVERVIEW.md`](./docs/PR-GOVERNANCE-OVERVIEW.md)

## License

MIT — see `LICENSE`.

## Versioning & BC

This module follows [Semantic Versioning 2.0.0](https://semver.org/) with the relaxed `v0.x`
convention:

| Version bump | API compatibility |
|---|---|
| **patch** (0.1.0 → 0.1.1) | BC-compatible — bug fixes, docs, performance |
| **minor** (0.1.0 → 0.1.1) | BC-compatible within a 0.x line |
| **0.x major bump** (0.1 → 0.2) | May break API — check CHANGELOG `### Breaking` section |
| **v1.0.0** | Full stability commitment — not part of Phase R |

**Current line:** `v0.1.x` — `API may break between 0.x major bumps (0.1 → 0.2); minor/patch within a 0.x line are BC-compatible`.

### Breaking changes

When a 0.x major bump introduces breaking changes:
1. A `### Breaking` section appears in `CHANGELOG.md` with the change description.
2. Migration notes are included in the same CHANGELOG entry.
3. A one minor-cycle deprecation warning will appear in the previous release where feasible.

### v1.0.0 trigger

Promotion to `v1.0.0` (full stability, no breaking changes) gates on:
- At least one consumer in production reporting stable usage
- Public API surface reviewed by maintainers (see `OWNERS`)
- Milestone v1.3+ ops trigger (not part of v1.2 scope)

### CHANGELOG format

This project uses [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format.
Sections per release: **Added** · **Changed** · **Deprecated** · **Removed** · **Fixed** · **Security** · **Breaking**.

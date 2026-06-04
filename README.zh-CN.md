[English](./README.md) | [简体中文](./README.zh-CN.md)

# agents — Go LLM agents 框架

`agents` 是一个 Go 版 LLM agents 框架，强调 stdlib-only、可组合、可测试。

一个独立的、**仅标准库（stdlib-only）**的 Go module，提供 LLM 驱动智能体的构建块：5 种经典 agent 范式（Simple、ReAct、Reflection、Plan-and-Solve、FunctionCall）、Memory（Working/Episodic/Semantic）、RAG、Context 工程、通信协议（MCP / A2A / ANP）、多智能体编排（Pipeline、FanOutFanIn、RoundRobin、RolePlay、StateGraph）、Agentic-RL 评估，以及 benchmark（BFCL、GAIA、LLM-as-Judge、Win Rate）。

> **v0.2.0 —— 学习 / 原型阶段。** API 在 0.x 各版本之间可能破坏。
> 任何稳定性承诺请等待 **v1.0**。
> 原始设计 specs（在框架孵化所在的上级 AICS 仓库中）：
> [`2026-04-27-pkg-llm-agents-design.md`](https://github.com/costa92/ai-customer-service/blob/main/docs/superpowers/specs/2026-04-27-pkg-llm-agents-design.md)
> · [`2026-05-06-pkg-llm-agents-multi-agent-orchestration-design.md`](https://github.com/costa92/ai-customer-service/blob/main/docs/superpowers/specs/2026-05-06-pkg-llm-agents-multi-agent-orchestration-design.md)

## Install / 安装

```bash
go get github.com/costa92/llm-agent@latest
```

RAG 已被拆分为一个独立的配套 module：

```bash
go get github.com/costa92/llm-agent-rag@latest
```

参见：

- `github.com/costa92/llm-agent-rag`
- `docs/2026-05-13-rag-sdk-migration-status.md`

> **可运行的演示：** 八个端到端示例位于
> [`./examples/`](./examples) —— 每个都是独立的、可 `go run .` 的程序，
> 带有一个确定性的模拟 LLM，无需 API key。菜单见
> [`examples/README.md`](./examples/README.md)。

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
| `agents` | Agent / Tool 接口 + 5 种范式构造器（Simple/ReAct/Reflection/PlanAndSolve/FunctionCall）+ Chain + Async + Registry |
| `agents/llm` | LLM 契约：ChatModel、ToolCaller、Request/Response、StreamReader/Event、Message、Tool、ToolCall、Usage、FinishReason |
| `agents/builtin` | Calculator、MockSearch、NoteTool、TerminalTool |
| `agents/memory` | WorkingMemory、EpisodicMemory、SemanticMemory、Manager、MemoryTool |
| `agents/rag` | Embedder（HashEmbedder）、Chunker、InMemoryStore、RAGSystem、MQE、HyDE |
| `agents/context` | GSSC ContextBuilder（Gather → Select → Structure → Compress） |
| `agents/comm` | Envelope + Transport（HTTP、Stdio） |
| `agents/comm/mcp` | Model Context Protocol 客户端/服务端（玩具级 spec 覆盖） |
| `agents/comm/a2a` | Agent-to-Agent 任务生命周期 |
| `agents/comm/anp` | Agent Network Protocol（内存版注册表） |
| `agents/orchestrate` | Pipeline、FanOutFanIn、RoundRobinChat、RolePlay、StateGraph、Supervisor、Termination |
| `agents/rl` | Dataset、Trajectory、Reward、Evaluator、TrainerProxy（无训练 —— Python TRL 桥接） |
| `agents/bench` | BFCL、GAIA、LLM-as-Judge、Win Rate、Reporter（仅迷你 fixtures） |

## Choosing orchestration / 编排选型

- `Pipeline` 适合固定的线性交接，例如 `research -> summarize -> answer`。
- `FanOutFanIn` 适合一个 planner 拆任务、多个 worker 并行执行、一个 aggregator 汇总结果。
- `RoundRobinChat` 适合多个 agent 轮流对话，直到终止规则触发。
- `RolePlay` 适合严格的双 agent 任务/委派循环，并且有 done marker。
- `StateGraph` 适合需要分支、循环、或基于共享类型化状态路由的工作流。
- `Supervisor` 适合一个 planner 反复调度 worker、观察结果并循环直到完成的工作流。

`Supervisor` 细节说明见 [`docs/SUPERVISOR.md`](./docs/SUPERVISOR.md)。

最小 `FanOutFanIn` 草图：

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

业务场景示例：

- 客服分诊：planner 拆分 `policy check` 和 `risk check`，专家并行运行，aggregator 起草最终回复
- 事件分析：planner 发出 `timeline`、`impact`、`mitigation` 任务，然后一个 summarizer 合并发现
- 调研简报：planner 拆分 API 变更和测试变更，然后一个 aggregator 把专家笔记变成发布摘要
- 退货策略：planner 拆分资格、物流和客户消息检查，然后一个 aggregator 起草最终回复
- 欺诈复核：planner 拆分交易模式、账户历史和策略检查，然后一个 aggregator 返回一个人工复核决策
- 知识检索：planner 拆分文档、发布说明和 FAQ 检查，然后一个 aggregator 返回一个检索答案

`StateGraph` 的设计刻意保持很小。它给你命名节点、无条件边、条件边，以及通过 `WithMaxSteps(...)` 提供的每次运行步数上限。这使它很适合 supervisor / specialist 工作流、reviewer-revise 循环和 backlog 式分发，而无需引入一个独立的编排 DSL。

完整架构和流程图见 [`docs/superpowers/specs/2026-05-06-pkg-llm-agents-multi-agent-orchestration-design.md`](https://github.com/costa92/ai-customer-service/blob/main/docs/superpowers/specs/2026-05-06-pkg-llm-agents-multi-agent-orchestration-design.md)。

最小 `StateGraph` 草图：

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

业务场景示例：

- 客服升级：基于共享状态路由到 `auto-reply`、`request-more-info` 或 `handover-human`
- 合规复核：在 `draft` 和 `review` 节点间循环，直到违规被清除
- 工单分发：把进来的工作分类为自动处理 vs 人工处理路由
- 保修升级：把符合条件的案例路由到维修受理，把超保案例路由到人工复核队列
- 欺诈冻结：筛查一笔交易并把高风险案例路由到人工复核
- 知识检索：收集缺失的产品上下文、搜索文档，然后回答
- 带澄清的知识检索：先收集产品和版本，然后搜索并回答
- 人工复核再检索：把一个拒付案例路由经过人工复核、上下文收集，并在回答前进行第二次搜索

场景来源：

- [`example_multi_agent_test.go`](./example_multi_agent_test.go) 中的 `Example_pipeline`、`Example_fanOutFanIn`、`Example_fanOutFanIn_supportTriage`、`Example_fanOutFanIn_incidentAnalysis`、`Example_fanOutFanIn_researchBrief`、`Example_fanOutFanIn_returnPolicy`、`Example_fanOutFanIn_fraudReview`、`Example_fanOutFanIn_knowledgeLookup`
- [`orchestrate/graph_test.go`](./orchestrate/graph_test.go) 中的 `ExampleStateGraph_loop`、`ExampleStateGraph_supportEscalation`、`ExampleStateGraph_ticketDispatch`、`ExampleStateGraph_complianceReview`、`ExampleStateGraph_warrantyEscalation`、`ExampleStateGraph_fraudHold`、`ExampleStateGraph_knowledgeLookup`、`ExampleStateGraph_knowledgeLookup_clarifySearchAnswer`、`ExampleStateGraph_humanReviewReseek`

客服升级风格草图：

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

这是一个独立的 Go module —— clone、编辑、测试：

```bash
git clone git@github.com:costa92/llm-agent.git
cd llm-agent
go vet ./...
go test ./...
```

要从下游消费方拾取本地改动（例如在同时迭代本 module 和一个导入它的服务时），在消费方的 `go.mod` 中加一条 `replace`：

```
replace github.com/costa92/llm-agent => /local/path/to/llm-agent
```

[`./examples/`](./examples) 下可运行的演示已经这样做了，以便它们与父目录中检出的任何版本保持同步。

`go.work` 被 gitignore。CI 和外部的 `go get` 调用方依赖父 module `go.mod` 中的 `require` 指令解析到已发布的 tag。

## Status & roadmap / 状态与路线图

- ✅ 全部 9 个设计阶段已交付（见源 spec）
- ✅ 12 个包，约 5000 行代码，仅标准库
- ✅ `examples/agents-demo`（在父 AICS 仓库中）端到端可运行
- ⏸ v1.0 —— 等待真实世界反馈

## Operational notes

- 跨 `llm-agent`、`llm-agent-rag`、`llm-agent-flow`、`llm-agent-providers`、`llm-agent-otel` 和 `llm-agent-customer-support` 的多仓 PR 治理设计，包括最终的 owner 自动合并 + 显式分支清理路径：
  [`docs/PR-GOVERNANCE-OVERVIEW.md`](./docs/PR-GOVERNANCE-OVERVIEW.md)

## License

MIT —— 见 `LICENSE`。

## Versioning & BC

本 module 遵循 [Semantic Versioning 2.0.0](https://semver.org/)，并采用宽松的 `v0.x` 约定：

| Version bump | API compatibility |
|---|---|
| **patch**（0.1.0 → 0.1.1） | BC 兼容 —— bug 修复、文档、性能 |
| **minor**（0.1.0 → 0.1.1） | 在一条 0.x 线内 BC 兼容 |
| **0.x major bump**（0.1 → 0.2） | 可能破坏 API —— 检查 CHANGELOG 的 `### Breaking` 小节 |
| **v1.0.0** | 完全稳定承诺 —— 不属于 Phase R |

**当前线：** `v0.1.x` —— `API may break between 0.x major bumps (0.1 → 0.2); minor/patch within a 0.x line are BC-compatible`。

### Breaking changes

当一次 0.x major 提升引入破坏性变更时：
1. `CHANGELOG.md` 中会出现一个带变更说明的 `### Breaking` 小节。
2. 迁移说明包含在同一条 CHANGELOG 条目中。
3. 在可行时，会在上一个版本中出现一个为期一个 minor 周期的弃用警告。

### v1.0.0 trigger

提升到 `v1.0.0`（完全稳定，无破坏性变更）以以下为门槛：
- 至少有一个生产中的消费方报告稳定使用
- 公共 API 面经维护者评审（见 `OWNERS`）
- Milestone v1.3+ 运维触发（不属于 v1.2 范围）

### CHANGELOG format

本项目使用 [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) 格式。
每个版本的小节：**Added** · **Changed** · **Deprecated** · **Removed** · **Fixed** · **Security** · **Breaking**。

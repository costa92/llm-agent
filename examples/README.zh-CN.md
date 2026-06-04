[English](./README.md) | [简体中文](./README.zh-CN.md)

# examples — runnable demos for `github.com/costa92/llm-agent`

每个子目录都是一个独立的 `package main`，你可以 `go run`。演示 01–08 不需要 API key —— 它们插入一个确定性的 `scriptedllm` 客户端，使输出可在离线时复现；把它替换成真实的 `llm.ChatModel`（OpenAI-compatible / DeepSeek / Ollama / Anthropic / MiniMax / …），同一份演示代码就能在生产中继续工作。[`09-ollama/`](./09-ollama) 正展示了对一个运行中的本地 Ollama 模型做这种替换。

| Demo | Surface | What it shows |
|---|---|---|
| [`01-simple-agent/`](./01-simple-agent) | `agents.SimpleAgent` | 单次 LLM 前向 —— 翻译 / 摘要 / 单轮问答 |
| [`02-tool-use/`](./02-tool-use) | `agents.FunctionCallAgent` + `agents.Registry` + `builtin.Calculator` | 原生工具调用：LLM 发出 `ToolCall`，agent 执行，结果成为 `Answer` |
| [`03-pipeline/`](./03-pipeline) | `orchestrate.Pipeline` | 线性交接 `research → summarize → answer`，附带完整的逐步轨迹 |
| [`04-state-graph/`](./04-state-graph) | `orchestrate.StateGraph` | 带条件边 + 一个循环的分支工作流 —— 客服分诊 |
| [`05-fanout/`](./05-fanout) | `pkg/fanout.Run` | 有界并行的任务执行器，带保序的 `Result[T].Index` |
| [`06-budget/`](./06-budget) | `budget.WithBudget` + `agents.SimpleAgent` | 预算 / 取消上下文 —— `MaxCalls` 调用前拒绝、`MaxTokens` 调用后拒绝、`MaxWall` ctx-deadline（附带一个确定性的 `main_test.go`） |
| [`07-policy/`](./07-policy) | `policy.Wrap` + `agents.SimpleAgent` | 安全中间件 —— PII 脱敏、提示词注入拦截，以及最大输入长度强制 |
| [`08-supervisor/`](./08-supervisor) | `orchestrate.Supervisor` | 迭代式 planner/worker 循环，带 dispatch 解析、聚合和预算传播 |
| [`09-ollama/`](./09-ollama) | `agents.SimpleAgent` + `llm-agent-providers/ollama` | **真实提供方** —— 把 `scriptedllm` 换成一个运行中的本地 Ollama 模型 + 原始 token 流式。它自己的 module；需要 `ollama serve`（见其 [README](./09-ollama)） |
| [`10-ollama-tools/`](./10-ollama-tools) | `agents.FunctionCallAgent` + `builtin.Calculator` + `llm-agent-providers/ollama` | **真实提供方 + 工具** —— 对一个自己决定调用工具的真实模型运行 demo 02。需要一个具备工具能力的模型（`llama3.1` / `qwen2.5-coder` / `qwen3-coder`）；自己的 module（见其 [README](./10-ollama-tools)） |

演示 01–08 插入确定性的 `scriptedllm` 模拟，无需 API key 即可离线运行。**演示 09 和 10 是例外**：它们与一个运行中的本地 Ollama 服务器通信，活在它们各自的 Go module 中以把沉重的提供方依赖挡在其他演示之外，且不属于下面的全部运行循环。

共享辅助：[`scriptedllm/`](./scriptedllm) —— 一个约 60 行的确定性模拟 `llm.ChatModel`。被演示 01-03 和 06-07 使用；演示 04-05 完全不触碰 LLM（StateGraph 运行纯节点函数；fanout 与提供方无关）。

## Run

```bash
# from the repo root
cd examples/01-simple-agent && go run .
cd examples/02-tool-use && go run .
cd examples/03-pipeline && go run .
cd examples/04-state-graph && go run .
cd examples/05-fanout && go run .
cd examples/06-budget && go run .
cd examples/07-policy && go run .
cd examples && go run ./08-supervisor
```

或者一次性全部运行：

```bash
cd examples
for d in 01-* 02-* 03-* 04-* 05-* 06-* 07-*; do
  echo "=== $d ==="; (cd "$d" && go run .)
done
echo "=== 08-supervisor ==="; go run ./08-supervisor
```

## Module layout

`examples/` 是它自己的 Go module（`github.com/costa92/llm-agent/examples`），带有 `replace github.com/costa92/llm-agent => ../`，因此：

- 仓内开发会即时拾取对框架的本地改动
- 父级 module 导出的 API 面保持干净 —— 当下游消费方 `go get github.com/costa92/llm-agent` 时不会被拉入 examples
- 每个演示都能通过一次 `go.mod` 改写（去掉 `replace`、锚定一个版本）复制进你自己的项目

## Where to find more

- **Godoc Examples**（在 pkg.go.dev 上渲染的 `func ExampleXxx` 形式）活在它们所记录的包中 —— 它们被刻意 _不_ 移到这里，从而 pkg.go.dev 能在每个 API 旁继续展示内联代码片段。见 https://pkg.go.dev/github.com/costa92/llm-agent 上的包级页面。
- **Architecture & design** —— 父级项目的 specs，位于 https://github.com/costa92/ai-customer-service/tree/main/docs/superpowers/specs。

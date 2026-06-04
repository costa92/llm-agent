[English](./README.md) | [简体中文](./README.zh-CN.md)

# Demo 10: Ollama + tools — native function-calling against a live model

这是把 [demo 02](../02-tool-use)（`FunctionCallAgent` + `Registry` + `builtin.Calculator`）换成**真实的 Ollama 后端**而非脚本化模拟。模型自己决定调用 `calculator` 工具；agent 执行它并把结果折叠进答案。与 demo 02 的唯一改动是客户端构造函数 —— 这正是 `llm.ChatModel` / `ToolCaller` 接缝在发挥作用。

## The tool-capable-model gotcha

原生工具调用需要一个 **Ollama 提供方**为其附带解析器的模型。截至 `llm-agent-providers` v0.3.0，那仅有 **`llama3.1`、`qwen2.5-coder` 和 `qwen3-coder`** 这些家族。

这与服务器 `GET /api/tags` 中的 `"tools"` 能力标志**不**是一回事 —— 像 `gemma4` 这样的模型可能向服务器上报 `"tools"`，却仍被提供方拒绝（因为它没有相应的解析器）。所以本演示通过询问提供方（`Info().Capabilities.Tools`）来选择模型，这与 agent 在构造时强制的信号相同。

**`llama3.1` 是最可靠的选择**：它返回原生的 `message.tool_calls`，提供方直接映射。`qwen*-coder` 家族把调用作为文本返回，提供方用启发式方法解析 —— 小变体（例如 `qwen2.5-coder:3b`）偶尔会发出格式错误的参数，于是运行会以一个 tool-args 错误干净地失败；重跑一次或换一个更大的模型即可。（解析带 markdown 围栏的 ```json 工具调用需要带 qwen-fence 修复的 `llm-agent-providers`；伞形构建已包含它。）

## Prerequisites

```sh
ollama serve
ollama pull llama3.1     # most reliable; or qwen2.5-coder / qwen3-coder
```

如果你已拉取的模型都不属于那些家族，该演示会精确打印这一点并告诉你该拉取什么 —— 它不会崩溃。

## Run

```sh
cd examples/10-ollama-tools
go run .
```

当 `OLLAMA_MODEL` 未设置时，该演示会询问提供方你已拉取的模型中哪些合格，并选择第一个。显式覆盖：

```sh
OLLAMA_HOST=http://localhost:11434 OLLAMA_MODEL=qwen2.5-coder go run .
```

## What it shows

1. **真实的工具调用** —— 真实模型发出一个 `ToolCall`，`FunctionCallAgent` 运行 `builtin.Calculator`，结果成为答案。没有脚本。
2. **能力感知的模型选择** —— 按提供方权威的 `Info().Capabilities.Tools` 来挑选模型，并附带可操作的错误（无论是自动选择时，还是当 `OLLAMA_MODEL` 强制指定了一个不受支持的模型时）。

选择逻辑（`pickToolModel` / `providerSupportsTools` / `normalizeHost`）在 `main_test.go` 中有确定性的、基于 `httptest` 的测试，它们演练真实的提供方策略并完全离线运行：

```sh
GOWORK=off go test ./...
```

## Why its own Go module

与 [demo 09](../09-ollama) 的理由相同：一个**独立 module**把那棵很重的提供方依赖树（Ollama / OpenAI / Anthropic SDK）挡在默认离线的父级 `examples` module 之外。不属于「全部运行」循环；在 CI 中用 `GOWORK=off` 构建。

[English](./README.md) | [简体中文](./README.zh-CN.md)

# Demo 09: Ollama — a real provider behind the `llm.ChatModel` seam

本目录下其他每个演示都插入确定性的 `scriptedllm` 模拟以便离线运行。**这一个不同**：它把一个真实的 [`llm-agent-providers/ollama`](../../../llm-agent-providers/ollama) 客户端接入 [demo 01](../01-simple-agent) 中那个完全相同的 `SimpleAgent` —— 只有构造函数那一行变了。这正是 `llm.ChatModel` 接缝的回报：模拟与生产共享同一份 agent 代码。

`*ollama.Ollama` 实现了 `llm.ChatModel`（`Generate` / `Stream` / `Info`），因此它能直接落入五种 agent 范式中的任何一种 —— 无需适配器。

## Prerequisites

本演示与一个**本地 Ollama 服务器**通信（它不像其他演示那样可离线复现）。你需要运行该服务器，且**至少有一个 chat 模型** —— 任意一个都行：

```sh
ollama serve            # start the server (default http://localhost:11434)
ollama pull llama3.2    # ...or any chat model you already have
```

## Run

```sh
cd examples/09-ollama
go run .
```

模型**不是硬编码的**：当 `OLLAMA_MODEL` 未设置时，该演示会查询 `GET /api/tags` 并自动选择你已拉取的第一个具备 chat 能力的模型（仅嵌入的模型会被跳过），因此 `go run .` 能对你机器上现有的任何模型工作。可通过环境变量显式覆盖 host/model：

```sh
OLLAMA_HOST=http://localhost:11434 OLLAMA_MODEL=qwen2.5 go run .
```

如果服务器不可达，或没有具备 chat 能力的模型，该演示会打印一条简短、可操作的提示并退出 —— 它不会 panic。

模型解析逻辑（`pickModel` / `chatCapable` / `normalizeHost`）由 `main_test.go` 中确定性的、基于 `httptest` 的测试覆盖，这些测试完全离线运行：

```sh
GOWORK=off go test ./...
```

## What it shows

1. **Agent 接线** —— `ollama.New(ollama.WithModel(...))` → `agents.NewSimpleAgent(client, ...)`。形状与 demo 01 相同，但后端是真实的。
2. **原始 token 流式** —— 迭代 `client.Stream(ctx, req)` 并在 `EventTextDelta` 文本到达时打印，这是 Ollama 在交互式 UX 上的强项。
3. **模型发现** —— 从 `GET /api/tags` 自动选择一个具备 chat 能力的模型，使该演示对你已拉取的任何模型都能运行，并在没有合适模型时给出可操作的错误。

## Why its own Go module

本目录是一个**独立 module**（`go.mod` + `go.sum`），与父级 `github.com/costa92/llm-agent/examples` module 分离。Ollama 适配器拖入了一棵很重的依赖树（Ollama / OpenAI / Anthropic SDK）；把它隔离在此处可以让父级 examples module **保持离线且无依赖** —— 这是它的定义性属性。其他演示绝不为此付出代价。

它被刻意**排除**在父级 [`README.md`](../README.zh-CN.md) 的「全部运行」循环之外，也**不**在 `go.work` 中（CI 用 `GOWORK=off` 构建它）。

```sh
# build/vet it on its own (no server needed for these):
cd examples/09-ollama
GOWORK=off go build .
GOWORK=off go vet .
```

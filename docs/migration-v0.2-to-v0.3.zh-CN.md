[English](./migration-v0.2-to-v0.3.md) | [简体中文](./migration-v0.2-to-v0.3.zh-CN.md)

# Migrating from v0.2 to v0.3

本文档现在是 `v0.3` 过渡的历史背景。`v0.4` 线已移除已弃用的兼容性接口面，因此对任何仍在使用旧 API 的调用方来说，迁移现在是强制的。

## Removed surface → current surface

| Removed symbol | Use instead |
|---|---|
| `llm.Client` | `llm.ChatModel` |
| `llm.LegacyClient` | `llm.ChatModel` |
| `llm.Client.Generate(ctx, GenerateRequest)` | `llm.ChatModel.Generate(ctx, Request)` |
| `llm.Client.GenerateStream(ctx, GenerateRequest)` | `llm.ChatModel.Stream(ctx, Request)` |
| `llm.GenerateRequest` | `llm.Request` |
| `llm.GenerateResponse` | `llm.Response` |
| `llm.StreamChunk` | `llm.StreamEvent` |
| `llm.StreamUsage` | `llm.Usage` |

## Current example: Simple paradigm

```go
package main

import (
    "context"
    "fmt"

    agents "github.com/costa92/llm-agent"
    "github.com/costa92/llm-agent/llm"
)

func main() {
    model := llm.NewScriptedLLM(
        llm.WithProvider("scripted"),
        llm.WithModel("test-1"),
        llm.WithResponses(llm.TextResponse("The capital of France is Paris.")),
    )

    // accepts llm.ChatModel post-Phase 3 (CORE-10)
    agent := agents.NewSimpleAgent(model, agents.SimpleOptions{
        Name:         "geography",
        SystemPrompt: "You are a helpful geography assistant.",
    })
    res, err := agent.Run(context.Background(), "What is the capital of France?")
    if err != nil {
        fmt.Println("error:", err)
        return
    }
    fmt.Println(res.Answer)
}
```

## Capability detection

依赖能力的代码路径的规范惯用法是类型断言**加上**一个 `Capabilities` 运行时检查 —— 两者都必需，因为 Go 类型可能实现某个接口（例如 Ollama 总是实现 `ToolCaller`），而所绑定的模型实际上并不支持该特性（`llama2` 返回 `Capabilities.Tools == false`）。

```go
if tc, ok := model.(llm.ToolCaller); ok && model.Info().Capabilities.Tools {
    bound, err := tc.WithTools(tools)
    if err != nil { return err }
    return bound.Generate(ctx, req)
}
// Fall back: scratchpad templating, or return ErrCapabilityNotSupported.
return model.Generate(ctx, scratchpadReq(req))
```

这是整个仓库中使用的基线惯用法。

## Streaming

v0.3 的流式契约使用 `llm.StreamReader`（迭代器风格：`Next + Close`），并发出一个带 `Kind` 枚举（`EventTextDelta` / `EventToolCallStart` / `EventToolCallArgsDelta` / `EventToolCallEnd` / `EventThinkingDelta` / `EventDone`）的类型化 `llm.StreamEvent` 联合。adapter（Phase 2）发出它们的**原生**粒度；想要一个扁平 `Response` 的消费方可以调用 `llm.AccumulateStream(sr)`。

## When to migrate

- 在消费 `v0.4.x` 线之前迁移。
- 如果你的代码仍提及上表中任何被移除的符号，先更新它；兼容层已不复存在。

完整的时间线 + 每个 Deprecated 符号 → 目标版本的映射，住在仓库根的 [`DEPRECATIONS.md`](../DEPRECATIONS.md) 里。

## Notes on shared / unchanged types

这些类型在当前 API 中保持相同的公共形状：

- `llm.Tool` —— 未变。相同的 `Name` / `Description` / `Parameters` 字段。
- `llm.Message` —— 未变。相同的 `Role` / `Content` 字段。
- `llm.FinishReason` + 6 个常量 —— 未变。
- `llm.ToolCall` —— 增加一个可选的 `ID string` 字段（被 Phase 3 以 `(message_id, tool_use_id)` 为 key 的工具去重层使用）。

共享减少了变动，并避免了并行的类型系统。

---

Last updated: 2026-05-13（Phase 7 `v0.4` 弃用移除）。

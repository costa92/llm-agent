# Migrating from v0.2 to v0.3

This document is now historical context for the `v0.3` transition. The `v0.4`
line has removed the deprecated compatibility surface, so migration is now
mandatory for any caller still using the old API.

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

The canonical idiom for capability-dependent code paths is type assertion
PLUS a `Capabilities` runtime check — both are required because the Go type may
implement an interface (e.g., Ollama always implements `ToolCaller`) while the
bound model does not actually support the feature (`llama2` returns
`Capabilities.Tools == false`).

```go
if tc, ok := model.(llm.ToolCaller); ok && model.Info().Capabilities.Tools {
    bound, err := tc.WithTools(tools)
    if err != nil { return err }
    return bound.Generate(ctx, req)
}
// Fall back: scratchpad templating, or return ErrCapabilityNotSupported.
return model.Generate(ctx, scratchpadReq(req))
```

This is the baseline idiom used throughout the repo.

## Streaming

The v0.3 streaming contract uses `llm.StreamReader` (iterator-style: `Next + Close`)
and emits a typed `llm.StreamEvent` union with a `Kind` enum (`EventTextDelta` /
`EventToolCallStart` / `EventToolCallArgsDelta` / `EventToolCallEnd` /
`EventThinkingDelta` / `EventDone`). Adapters (Phase 2) emit their NATIVE
granularity; consumers that want a flat `Response` can call
`llm.AccumulateStream(sr)`.

## When to migrate

- Migrate before consuming the `v0.4.x` line.
- If your code still mentions any removed symbol in the table above, update it
  first; the compatibility layer no longer exists.

The full timeline + every Deprecated symbol → target version mapping lives in
[`DEPRECATIONS.md`](../DEPRECATIONS.md) at the repo root.

## Notes on shared / unchanged types

These types keep the same public shape in the current API:

- `llm.Tool` — unchanged. Same `Name` / `Description` / `Parameters` fields.
- `llm.Message` — unchanged. Same `Role` / `Content` fields.
- `llm.FinishReason` + 6 constants — unchanged.
- `llm.ToolCall` — adds an optional `ID string` field (used by Phase 3's tool
  dedupe layer keyed by `(message_id, tool_use_id)`).

Sharing reduces churn and avoids parallel type systems.

---

Last updated: 2026-05-13 (Phase 7 `v0.4` deprecation removal).

# Migrating from v0.2 to v0.3

The v0.3 release adds capability-aware interfaces (`llm.ChatModel`, `llm.ToolCaller`,
`llm.Embedder`, `llm.StructuredOutputs`) alongside the existing `llm.Client` (now
`llm.LegacyClient`). The old surface remains callable through v0.3.x; **it will be
removed in v0.4.0**.

> **You do not have to migrate immediately.** `type Client = LegacyClient` is an
> alias — every existing caller compiles unchanged in v0.3.0. Migration becomes
> mandatory only at the v0.4.0 cut (one minor cycle later, per the project's
> dual-track BC policy).

## Quick reference: type renames

| v0.2 | v0.3 (new code) | v0.3 (legacy callers — alias) |
|---|---|---|
| `llm.Client` | `llm.ChatModel` | `llm.LegacyClient` (or `llm.Client`, alias) |
| `llm.Client.Generate(ctx, GenerateRequest) (GenerateResponse, error)` | `llm.ChatModel.Generate(ctx, Request) (Response, error)` | unchanged via alias |
| `llm.Client.GenerateStream(ctx, GenerateRequest) (<-chan StreamChunk, error)` | `llm.ChatModel.Stream(ctx, Request) (StreamReader, error)` | unchanged via alias |
| `llm.GenerateRequest` | `llm.Request` | unchanged via alias |
| `llm.GenerateResponse` | `llm.Response` | unchanged via alias |
| `llm.StreamChunk` (single channel value) | `llm.StreamEvent` (typed union with `Kind`) | unchanged via alias |
| `llm.StreamUsage` | `llm.Usage` (+ new `Source` field) | unchanged via alias |
| `llm.Tool` | `llm.Tool` (unchanged shape) | unchanged |
| `llm.ToolCall` (Name + Arguments) | `llm.ToolCall` (now: ID + Name + Arguments) | unchanged shape; `ID` is `omitempty` (back-compat) |
| `llm.Message` | `llm.Message` (unchanged shape) | unchanged |
| `llm.FinishReason` + 6 const | unchanged (alias to underlying type) | unchanged |
| (no equivalent) | `llm.ProviderInfo` returned by `Info()` | n/a — new in v0.3 |
| (no equivalent) | `llm.Capabilities` (struct on `ProviderInfo`) | n/a — new in v0.3 |
| (no equivalent) | `llm.ToolCaller`, `llm.Embedder`, `llm.StructuredOutputs` | n/a — new in v0.3 |
| (no equivalent) | `llm.ErrCapabilityNotSupported` | n/a — new in v0.3 |
| `scriptedllm_test.go` `scriptedLLM` (test helper) | `llm.ScriptedLLM` (production code; full-capability mock) | shim retained until Phase 3 |

## Worked example: Simple paradigm

The Simple paradigm is the smallest agent — it just forwards every prompt to the
LLM and returns the text. Both versions below are runnable.

### v0.2 (current — still works in v0.3.0 via alias)

```go
package main

import (
    "context"
    "fmt"

    agents "github.com/costa92/llm-agent"
)

func main() {
    // newScriptedLLM here returns the v0.2 llm.Client contract.
    // In production, replace with a real llm.Client implementation.
    client := newScriptedLLM(
        textResp("The capital of France is Paris."),
    )

    agent := agents.NewSimpleAgent(client, agents.SimpleOptions{
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

### v0.3 transitional — no source change required

```go
// SAME CODE WORKS. llm.Client is now an alias for llm.LegacyClient,
// and the scriptedLLM test helper still returns the legacy contract.
// Compiles unchanged in v0.3.0.
package main

import (
    "context"
    "fmt"

    agents "github.com/costa92/llm-agent"
)

func main() {
    client := newScriptedLLM(
        textResp("The capital of France is Paris."),
    )
    agent := agents.NewSimpleAgent(client, agents.SimpleOptions{
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

### v0.3 idiomatic — uses the new ChatModel directly

This shape becomes available after Phase 3 lands `agents.NewSimpleAgent(llm.ChatModel)`
(CORE-10 in the roadmap). Until then, stay on the legacy path above.

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

## Capability detection (forward-looking)

In v0.3 the canonical idiom for capability-dependent code paths is type assertion
PLUS a `Capabilities` runtime check — both are required because the Go type may
implement an interface (e.g., Ollama always implements `ToolCaller`) while the
bound model does not actually support the feature (`llama2` returns
`Capabilities.Tools == false`). This is the K2 keystone of v0.3.

```go
if tc, ok := model.(llm.ToolCaller); ok && model.Info().Capabilities.Tools {
    bound, err := tc.WithTools(tools)
    if err != nil { return err }
    return bound.Generate(ctx, req)
}
// Fall back: scratchpad templating, or return ErrCapabilityNotSupported.
return model.Generate(ctx, scratchpadReq(req))
```

Phase 3 (CORE-10) refactors `agents.NewReActAgent`, `agents.NewFunctionCallAgent`,
etc., to use this idiom. v0.2 callers don't need to do anything.

## Streaming

The v0.3 streaming contract uses `llm.StreamReader` (iterator-style: `Next + Close`)
and emits a typed `llm.StreamEvent` union with a `Kind` enum (`EventTextDelta` /
`EventToolCallStart` / `EventToolCallArgsDelta` / `EventToolCallEnd` /
`EventThinkingDelta` / `EventDone`). Adapters (Phase 2) emit their NATIVE
granularity; consumers that want a flat `Response` can call
`llm.AccumulateStream(sr)`.

The v0.2 `<-chan StreamChunk` shape remains via `LegacyClient.GenerateStream`.

## When to migrate

- **v0.3.0 (now):** new provider adapter authors target `llm.ChatModel` directly.
  Existing callers do nothing — alias preserves compilation.
- **v0.3.x (Phase 3 / CORE-10):** internal agents migrate to `llm.ChatModel`.
  Examples + tests follow.
- **v0.4.0 (one minor cycle after v0.3.0 ships):** `llm.Client`,
  `llm.LegacyClient`, `llm.GenerateRequest`, `llm.GenerateResponse`,
  `llm.StreamChunk`, `llm.StreamUsage` removed. All remaining callers MUST
  migrate before this tag.

The full timeline + every Deprecated symbol → target version mapping lives in
[`DEPRECATIONS.md`](../DEPRECATIONS.md) at the repo root.

## Notes on shared / unchanged types

These types KEEP THEIR v0.2 SHAPE in v0.3 — both the legacy and new surfaces use
the same declarations from `llm/types.go`:

- `llm.Tool` — unchanged. Same `Name` / `Description` / `Parameters` fields.
- `llm.Message` — unchanged. Same `Role` / `Content` fields.
- `llm.FinishReason` + 6 constants — unchanged. The new `FinishReason` is a Go
  type alias to the legacy underlying type, so `llm.FinishReasonStop` is the same
  value in both surfaces.
- `llm.ToolCall` — adds an optional `ID string` field (used by Phase 3's tool
  dedupe layer keyed by `(message_id, tool_use_id)`). The field is `omitempty`,
  so v0.2 callers continue to work; only the LLM-side populates `ID`.

Sharing reduces churn and avoids two parallel type systems. If a future v0.4
release evolves any of these shapes, that's the migration window — it doesn't
affect v0.3.x.

---

Last updated: 2026-05-10 (Phase 0 of the v0.3 milestone).

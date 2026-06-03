# Demo 10: Ollama + tools — native function-calling against a live model

This is [demo 02](../02-tool-use) (`FunctionCallAgent` + `Registry` +
`builtin.Calculator`) with a **real Ollama backend** instead of the scripted
mock. The model itself decides to call the `calculator` tool; the agent
executes it and folds the result into the answer. The only change from demo
02 is the client constructor — that is the `llm.ChatModel` / `ToolCaller`
seam doing its job.

## The tool-capable-model gotcha

Native tool-calling needs a model the **Ollama provider** ships a parser for.
As of `llm-agent-providers` v0.3.0 that is only the **`llama3.1`,
`qwen2.5-coder`, and `qwen3-coder`** families.

This is **not** the same as the server's `GET /api/tags` `"tools"` capability
flag — a model like `gemma4` can report `"tools"` to the server yet still be
rejected by the provider (which has no parser for it). So this demo selects
models by asking the provider (`Info().Capabilities.Tools`), the same signal
the agent enforces at construction time.

**`llama3.1` is the most reliable choice**: it returns native
`message.tool_calls`, which the provider maps directly. The `qwen*-coder`
families return the call as text the provider parses heuristically — small
variants (e.g. `qwen2.5-coder:3b`) occasionally emit malformed arguments and
the run fails cleanly with a tool-args error; just re-run or use a larger
model. (Parsing markdown-fenced ```json tool calls needs `llm-agent-providers`
with the qwen-fence fix; the umbrella build already has it.)

## Prerequisites

```sh
ollama serve
ollama pull llama3.1     # most reliable; or qwen2.5-coder / qwen3-coder
```

If none of your pulled models are in those families, the demo prints exactly
that and tells you what to pull — it does not crash.

## Run

```sh
cd examples/10-ollama-tools
go run .
```

With `OLLAMA_MODEL` unset the demo asks the provider which of your pulled
models qualifies and picks the first. Override explicitly:

```sh
OLLAMA_HOST=http://localhost:11434 OLLAMA_MODEL=qwen2.5-coder go run .
```

## What it shows

1. **Real tool-calling** — the live model emits a `ToolCall`, `FunctionCallAgent` runs `builtin.Calculator`, and the result becomes the answer. No scripting.
2. **Capability-aware model selection** — picking a model by the provider's authoritative `Info().Capabilities.Tools`, with an actionable error (both at auto-select and when `OLLAMA_MODEL` forces an unsupported model).

The selection logic (`pickToolModel` / `providerSupportsTools` /
`normalizeHost`) has deterministic `httptest`-backed tests in `main_test.go`
that exercise the real provider strategy and run fully offline:

```sh
GOWORK=off go test ./...
```

## Why its own Go module

Same rationale as [demo 09](../09-ollama): a **standalone module** keeps the
heavy provider dependency tree (Ollama / OpenAI / Anthropic SDKs) off the
offline-by-default parent `examples` module. Not part of the run-all loop;
built in CI with `GOWORK=off`.

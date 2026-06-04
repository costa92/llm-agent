[English](./README.md) | [简体中文](./README.zh-CN.md)

# Demo 09: Ollama — a real provider behind the `llm.ChatModel` seam

Every other demo in this directory plugs the deterministic `scriptedllm`
mock so it runs offline. **This one is different**: it wires a live
[`llm-agent-providers/ollama`](../../../llm-agent-providers/ollama) client
into the exact `SimpleAgent` from [demo 01](../01-simple-agent) — only the
constructor line changes. That is the payoff of the `llm.ChatModel` seam:
mock and production share the same agent code.

`*ollama.Ollama` implements `llm.ChatModel` (`Generate` / `Stream` /
`Info`), so it drops straight into any of the five agent paradigms — no
adapter.

## Prerequisites

This demo talks to a **local Ollama server** (it is NOT offline-reproducible
like the others). You need the server running with **at least one chat
model** — any will do:

```sh
ollama serve            # start the server (default http://localhost:11434)
ollama pull llama3.2    # ...or any chat model you already have
```

## Run

```sh
cd examples/09-ollama
go run .
```

The model is **not hardcoded**: with `OLLAMA_MODEL` unset the demo queries
`GET /api/tags` and auto-selects the first chat-capable model you have
pulled (embedding-only models are skipped), so `go run .` works against
whatever is on your machine. Override host/model explicitly via env:

```sh
OLLAMA_HOST=http://localhost:11434 OLLAMA_MODEL=qwen2.5 go run .
```

If the server is unreachable, or has no chat-capable model, the demo prints
a short actionable hint and exits — it does not panic.

The model-resolution logic (`pickModel` / `chatCapable` / `normalizeHost`)
is covered by deterministic `httptest`-backed tests in `main_test.go` that
run fully offline:

```sh
GOWORK=off go test ./...
```

## What it shows

1. **Agent wiring** — `ollama.New(ollama.WithModel(...))` → `agents.NewSimpleAgent(client, ...)`. Same shape as demo 01, real backend.
2. **Raw token streaming** — iterating `client.Stream(ctx, req)` and printing `EventTextDelta` text as it arrives, Ollama's strength for interactive UX.
3. **Model discovery** — auto-selecting a chat-capable model from `GET /api/tags` so the demo runs against whatever you have pulled, with an actionable error when nothing fits.

## Why its own Go module

This directory is a **standalone module** (`go.mod` + `go.sum`), separate
from the parent `github.com/costa92/llm-agent/examples` module. The Ollama
adapter drags in a heavy dependency tree (Ollama / OpenAI / Anthropic
SDKs); isolating it here keeps the parent examples module **offline and
dependency-free** — its defining property. The other demos never pay for
this.

It is intentionally **not** part of the "run them all" loop in the parent
[`README.md`](../README.md), and **not** in `go.work` (CI builds it with
`GOWORK=off`).

```sh
# build/vet it on its own (no server needed for these):
cd examples/09-ollama
GOWORK=off go build .
GOWORK=off go vet .
```

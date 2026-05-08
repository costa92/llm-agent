# examples — runnable demos for `github.com/costa92/llm-agent`

Each subdirectory is a standalone `package main` you can `go run` without
an API key — every demo plugs a deterministic `scriptedllm` client so the
output is reproducible offline. Replace it with a real `llm.Client`
(OpenAI-compatible / Ollama / Anthropic / …) and the same demo code keeps
working in production.

| Demo | Surface | What it shows |
|---|---|---|
| [`01-simple-agent/`](./01-simple-agent) | `agents.SimpleAgent` | Single-shot LLM forward — translation / summarization / single-turn Q&A |
| [`02-tool-use/`](./02-tool-use) | `agents.FunctionCallAgent` + `agents.Registry` + `builtin.Calculator` | Native tool-calling: LLM emits `ToolCall`, agent executes, result becomes `Answer` |
| [`03-pipeline/`](./03-pipeline) | `orchestrate.Pipeline` | Linear handoff `research → summarize → answer` with full per-step trail |
| [`04-state-graph/`](./04-state-graph) | `orchestrate.StateGraph` | Branching workflow with conditional edges + a loop — customer-service triage |
| [`05-fanout/`](./05-fanout) | `pkg/fanout.Run` | Bounded-parallelism task runner with order-preserving `Result[T].Index` |

Shared helper: [`scriptedllm/`](./scriptedllm) — a ~60-line deterministic
mock `llm.Client`. Used by demos 01-03; demos 04-05 don't touch the LLM at
all (StateGraph runs pure node funcs; fanout is provider-agnostic).

## Run

```bash
# from the repo root
cd examples/01-simple-agent && go run .
cd examples/02-tool-use && go run .
cd examples/03-pipeline && go run .
cd examples/04-state-graph && go run .
cd examples/05-fanout && go run .
```

Or run them all in one go:

```bash
cd examples
for d in 01-* 02-* 03-* 04-* 05-*; do
  echo "=== $d ==="; (cd "$d" && go run .)
done
```

## Module layout

`examples/` is its own Go module (`github.com/costa92/llm-agent/examples`)
with `replace github.com/costa92/llm-agent => ../`, so:

- in-repo development picks up local edits to the framework instantly
- the parent module's exported API surface stays clean — examples are not
  pulled in when downstream consumers `go get github.com/costa92/llm-agent`
- each demo can be copied into your own project with one `go.mod` rewrite
  (drop the `replace`, pin a version)

## Where to find more

- **Godoc Examples** (the `func ExampleXxx` form rendered on pkg.go.dev) live
  in the package they document — they are intentionally _not_ moved here, so
  pkg.go.dev keeps showing inline code snippets next to each API. See the
  package-level pages on https://pkg.go.dev/github.com/costa92/llm-agent.
- **Architecture & design** — the parent project's specs at
  https://github.com/costa92/ai-customer-service/tree/main/docs/superpowers/specs.

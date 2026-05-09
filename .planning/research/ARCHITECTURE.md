# Architecture Research

**Domain:** Go LLM agent framework — provider abstraction evolution + OTel observability + multi-repo deployment
**Researched:** 2026-05-10
**Confidence:** HIGH (Eino interfaces, OTel semconv, Anthropic streaming spec, openai-go API verified against current docs/source); MEDIUM on reference-service shape (no single canonical pattern; synthesis from go-llm/ADK/LangGraph patterns)

## Standard Architecture

### Multi-Repo Layered View

```
┌──────────────────────────────────────────────────────────────────────────┐
│ Repo 4: llm-agent-customer-support  (deployable reference service)       │
│   HTTP handlers · session store · agent factories · docker-compose       │
│   imports → llm-agent + llm-agent-providers + llm-agent-otel             │
└────────────────────────────────┬─────────────────────────────────────────┘
                                 │ depends on (all 3 below)
              ┌──────────────────┼──────────────────┐
              ▼                  ▼                  ▼
┌─────────────────────┐ ┌──────────────────┐ ┌────────────────────────────┐
│ Repo 2: providers   │ │ Repo 3: otel     │ │ Repo 1: llm-agent (core)   │
│  llm-agent-providers│ │ llm-agent-otel   │ │  ── stdlib-only ──         │
│                     │ │                  │ │                            │
│  openai/  → llm.v2  │ │ traces / metrics │ │ llm/        Client (deprc) │
│  anthropic/         │ │ slog bridge      │ │ llm/v2/     ChatModel +    │
│  ollama/            │ │ wraps llm.v2.    │ │             ToolCaller +   │
│                     │ │ ChatModel +      │ │             Embedder       │
│  imports llm-agent  │ │ agent.Agent      │ │ agent/  paradigms          │
│  ONLY (not otel)    │ │                  │ │ orchestrate/ graphs        │
│                     │ │ imports          │ │ rag/ memory/ context/ ...  │
│                     │ │ llm-agent ONLY   │ │                            │
└─────────────────────┘ └──────────────────┘ └────────────────────────────┘
              ▲                  ▲                       ▲
              │                  │                       │
              └─── Both depend on llm-agent's interface package ───┘
                   Neither depends on the other (siblings)
```

**Critical rule:** providers and otel are SIBLINGS. They both depend on the core's stable `llm/v2` interface package. They never depend on each other. The reference service is the ONLY place where the three converge.

This shape is the same one openai-go, anthropic-sdk-go, and `go.opentelemetry.io/contrib/instrumentation/...` all converge on: provider SDKs and instrumentation packages depend on a common interface, never on each other.

### Component Responsibilities (core repo)

| Package | Owns | Boundary Rule |
|---|---|---|
| `llm/` | Legacy `Client`, `Generate*Request/Response`, `Message`, `Tool`, `ToolCall`, `StreamChunk`, `FinishReason` | Frozen at v0.2.x shape. Marked `// Deprecated:` after v0.3 ships. Stays callable for one minor cycle. |
| `llm/v2/` | `ChatModel` (base), `ToolCaller`, `Embedder`, `StructuredOutputs`, `ProviderInfo` (capability struct), `StreamEvent` (typed union) | New canonical contract. The seam every external repo binds to. Stdlib-only. **No provider implementations live here.** |
| `agent/` (root pkg today) | `Agent`, `Step`, `Result`, 5 paradigm constructors (Simple/ReAct/Reflection/PlanSolve/FunctionCall) | Consumes `llm/v2.ChatModel`. Probes capabilities via type assertion + `ProviderInfo`. Falls back to prompt templates when capability is missing. |
| `orchestrate/` | Pipeline / FanOutFanIn / RoundRobin / RolePlay / StateGraph / Termination | Untouched by this milestone — talks to `agent.Agent`, not to LLM clients. |
| `builtin/`, `memory/`, `rag/`, `context/`, `comm/`, `bench/`, `rl/` | Existing — unchanged | RAG `Embedder` (currently HashEmbedder) gains an "I can wrap an `llm/v2.Embedder`" constructor. |

### Component Responsibilities (sister repos)

| Package | Owns | Imports |
|---|---|---|
| `llm-agent-providers/openai` | OpenAI adapter implementing `ChatModel + ToolCaller + Embedder + StructuredOutputs` over `github.com/openai/openai-go/v3` | `llm/v2` |
| `llm-agent-providers/anthropic` | Anthropic adapter implementing `ChatModel + ToolCaller` (no Embedder — Anthropic has none); uses `Accumulate()` for streaming over `github.com/anthropics/anthropic-sdk-go` | `llm/v2` |
| `llm-agent-providers/ollama` | Ollama adapter implementing `ChatModel + ToolCaller (model-dependent) + Embedder`; hand-rolled HTTP | `llm/v2` |
| `llm-agent-providers/internal/contract` | httptest contract suite — runs the same scenarios against all three adapters | `llm/v2`, all three adapters |
| `llm-agent-otel` | `otelmodel.Wrap(llm/v2.ChatModel) llm/v2.ChatModel`, `otelagent.Wrap(agent.Agent) agent.Agent`; emits `gen_ai.*` semconv attributes | `llm/v2` + core agent pkg + `go.opentelemetry.io/otel` |
| `llm-agent-customer-support` | HTTP `/chat`, `/chat/stream` (SSE), session storage, agent factories, docker-compose, optional Helm | All three sister modules + `llm-agent` core |

## Capability Negotiation Pattern

This is the central design question for the milestone. Three patterns were considered.

### Recommended: Small Interfaces + Type Assertion + Provider-Side Metadata

```go
// llm/v2/client.go
package llmv2

// ChatModel is the base contract every provider implements. Mirrors
// eino.BaseChatModel and genkit.DefineModel's generation handler.
type ChatModel interface {
    Generate(ctx context.Context, req Request) (Response, error)
    Stream(ctx context.Context, req Request) (StreamReader, error)
    Info() ProviderInfo // SEE BELOW: cheap struct, not a method per capability
}

// ToolCaller is the capability for native tool/function-calling.
// Providers that don't support tools simply don't implement it.
// Agents type-assert to detect.
type ToolCaller interface {
    ChatModel
    // WithTools returns a NEW model bound to these tools — pure, safe to
    // call concurrently. (Eino learned this the hard way: BindTools
    // mutated state, was deprecated, replaced by WithTools.)
    WithTools(tools []Tool) (ToolCaller, error)
}

// Embedder is the capability for vector embeddings.
type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    EmbedDimensions() int
}

// StructuredOutputs is the capability for native JSON-schema constrained
// generation (OpenAI response_format, Anthropic tool-as-output trick).
type StructuredOutputs interface {
    GenerateStructured(ctx context.Context, req Request, schema json.RawMessage) (Response, error)
}

// ProviderInfo is a cheap, copy-by-value capability advertisement.
// Mirrors genkit's ai.ModelInfo.Supports and serves as a HINT for
// callers that want to choose between a fast path and a fallback
// without doing a type assertion. Type assertions are still authoritative.
type ProviderInfo struct {
    Name              string // "openai", "anthropic", "ollama"
    Model             string // "gpt-4o-2026-01", "claude-3-5-sonnet", "llama3.1"
    SupportsTools     bool
    SupportsStreaming bool
    SupportsEmbedding bool
    SupportsStructured bool
    SupportsSystemRole bool
    SupportsMedia     bool
    MaxContextTokens  int
}
```

**How an agent uses this — ReAct pressure test:**

```go
// react.go — refactored
func (a *ReActAgent) step(ctx context.Context, model llmv2.ChatModel, req llmv2.Request) (...) {
    // Fast path: provider has native tool calls.
    if tc, ok := model.(llmv2.ToolCaller); ok && a.opts.Registry != nil {
        bound, err := tc.WithTools(a.opts.Registry.AsLLMTools())
        if err != nil {
            return nil, err
        }
        resp, err := bound.Generate(ctx, req)
        // resp.ToolCalls populated by provider
        return resp, err
    }
    // Fallback: prompt-template tool list, parse Action/Args from text.
    req.Prompt = a.injectToolList(req.Prompt) // existing scratchpad path
    return model.Generate(ctx, req)
}
```

**Why this shape:**

1. **Go-idiomatic.** "The bigger the interface, the weaker the abstraction." Small interfaces that consumers compose via type assertion is exactly the `io.Reader` / `io.WriterTo` / `io.ReaderFrom` model.
2. **Eino converged on this** — `BaseChatModel` is small, `ToolCallingChatModel` extends it. Their first attempt (single `ChatModel` with `BindTools`) was deprecated.
3. **Genkit converged on this** — `ai.ModelSupports` struct + the model handler is one function. The struct is the hint; the handler is the truth.
4. **Langchaingo took the other road** — single fat `Model` interface with `GenerateContent` + `WithTools` as a `CallOption`. Works, but they leak provider-specific options through `WithMetadata` / `WithJSONMode` / etc. — a soft-typed escape hatch for capabilities they didn't carve out as types. We avoid that.
5. **Five paradigms × three providers = 15 negotiation sites.** Both type assertion AND `ProviderInfo` cost is paid once per agent step, not per call. `ProviderInfo()` lets agents log/route without a type-assert; type assertion confirms before commit.

### Alternatives Considered

**Alternative A: Single fat `Client` with optional fields (current `llm.Client` shape).**
Today's `llm.Client.Generate` takes `Tools` in the request and providers "ignore" them. Rejected because: agents can't tell the difference between "provider ignored Tools" and "provider returned no tool calls because none were needed"; there's no way to surface "this provider doesn't do embeddings" without runtime errors; can't add embeddings without breaking BC.

**Alternative B: Capability bitmask method `Caps() Capability`.**
```go
type Capability uint32
const (
    CapTools Capability = 1 << iota
    CapEmbedding
    CapStreaming
)
```
Rejected because: bitmask hides the actual method signature, callers still need a way to invoke the embedding method, which means they still need a type assertion or a fat interface with optional methods. Bitmask just adds bookkeeping without removing the assertion. Genkit's `ModelSupports` struct (which is what `ProviderInfo` mirrors) is the same idea but with named fields, which is more readable.

## Data Flow

### Single Request, Streaming Path (OpenAI tools example)

```
HTTP /chat/stream  (refsvc handler, refsvc repo)
  │
  │ ① POST {messages, session_id}
  ▼
session.Load(session_id)  → []Message (history)
  │
  ▼
agent := factory.NewReActAgent(model, registry)   ← model is otel-wrapped
  │
  │ ② agent.RunStream(ctx, input) <-chan StepEvent
  ▼
ReActAgent.runInternal
  │
  │ ③ type-assert: model.(llmv2.ToolCaller) → bound, err := tc.WithTools(tools)
  │
  │ ④ bound.Stream(ctx, req) → StreamReader
  ▼
otelmodel wrapper                                  ← starts span "chat openai/gpt-4o"
  │                                                  attrs: gen_ai.system="openai",
  │                                                         gen_ai.request.model=...
  ▼
openai-go v3.Chat.Completions.NewStreaming         ← provider repo
  │
  │ ⑤ for stream.Next() { ev := stream.Current(); … }
  │     OpenAI emits ChoiceDelta with .ToolCalls[i].Function.Arguments delta strings
  ▼
adapter normalizes to llmv2.StreamEvent           ← typed union, see below
  │   - StreamEvent{Kind: TextDelta, Text: "..."}
  │   - StreamEvent{Kind: ToolCallStart, ToolCall: ToolCall{Name: "search"}}
  │   - StreamEvent{Kind: ToolCallArgsDelta, ToolCall: ToolCall{Args: "{\"q\":\"go"}}
  │   - StreamEvent{Kind: ToolCallStart, …}        ← Anthropic content_block_start
  │   - StreamEvent{Kind: Done, FinishReason: "tool_calls"}
  ▼
agent loop accumulates by ToolCall.Index/ID, executes tool, feeds back
  │
  │ ⑥ Step events emitted via runStreamFromBlocking(ctx, …)
  ▼
SSE writer (refsvc) writes StepEvent → text/event-stream
  │   span ends on channel close; gen_ai.usage.{input,output}_tokens recorded
  ▼
client
```

### The Streaming + Tool Calls Abstraction (concrete, not handwave)

The hard problem the question flags: **OpenAI emits incremental delta chunks (`tool_calls[i].function.arguments` is a partial string at each chunk)**, **Anthropic emits content blocks (`content_block_start{type:"tool_use", id, name}` then a stream of `content_block_delta{delta: {type:"input_json_delta", partial_json: "..."}}` then `content_block_stop`)**.

The current `llm.StreamChunk` has a single optional `*ToolCall` field, which is lowest-common-denominator and forces every adapter to fully accumulate before emitting — losing streaming UX for tool calls.

**Solution: typed event union with stable per-tool-call indexing.**

```go
// llm/v2/stream.go
type StreamEvent struct {
    Kind     StreamEventKind // see below
    Text     string          // TextDelta only
    ToolCall *ToolCallDelta  // ToolCall* kinds only
    Usage    *Usage          // Done only (when provider sends it)
    FinishReason FinishReason // Done only
}

type StreamEventKind uint8
const (
    EventTextDelta StreamEventKind = iota
    EventToolCallStart       // name + index/id known; args not yet
    EventToolCallArgsDelta   // partial JSON string for this tool call
    EventToolCallEnd         // this tool call is fully streamed
    EventThinkingDelta       // optional; Anthropic + reasoning models
    EventDone                // terminal; usage + finish_reason
)

type ToolCallDelta struct {
    Index     int    // stable across deltas — same tool call across chunks
    ID        string // OpenAI tool_call_id / Anthropic content_block id
    Name      string // populated on EventToolCallStart only
    ArgsDelta string // partial JSON string (raw concatenation builds final args)
}

type StreamReader interface {
    Next() (StreamEvent, error) // io.EOF when stream ends
    Close() error
}
```

**Adapter responsibility (per provider):**
- OpenAI: buffer `tool_calls[i]` first appearance into `EventToolCallStart`; subsequent `arguments` deltas → `EventToolCallArgsDelta` keyed by `i`; `finish_reason` → `EventDone`. Index = OpenAI's `i`.
- Anthropic: `content_block_start{type:"tool_use"}` → `EventToolCallStart`; `content_block_delta{input_json_delta}` → `EventToolCallArgsDelta`; `content_block_stop` for a tool block → `EventToolCallEnd`; `message_stop` → `EventDone`. Index = Anthropic's `index`.
- Ollama: simpler — full tool call in one chunk; emit `Start` + `ArgsDelta` (full string) + `End` back-to-back.

**Why this works without lowest-common-denominator:**
- OpenAI users get true incremental UX (can render `searching for "go cap..."` mid-call).
- Anthropic users get true incremental UX (mirrors Anthropic's `Accumulate()` model).
- Ollama users get the same API even though their wire format is simpler.
- Agents that don't care just call a helper `event.AccumulateToToolCall(&buf)` and ignore intermediate kinds.

## Architectural Patterns

### Pattern 1: Decorator Wrapping for Observability (NOT hooks)

**What:** OTel instrumentation is a `Wrap(ChatModel) ChatModel` and `Wrap(Agent) Agent` — the wrapper implements the same interface, calls the inner, and emits spans/metrics around the call.

**When to use:** Always. This is the only pattern that composes with capability interfaces correctly.

**Trade-offs:**
- (+) Zero coupling between core and otel. Core knows nothing about OTel.
- (+) Users opt in by wrapping; opt out by not wrapping.
- (+) Composes with other wrappers (retry, rate limit, cache).
- (−) Wrapper must re-implement every capability interface it wants to expose. Solution: `otelmodel.Wrap` returns a value that implements all known capabilities IF the inner does, using type assertion + struct embedding. Same trick `otelhttp.NewTransport` uses for `http.RoundTripper`.

**Why not hooks/callbacks:** Hooks (e.g., `agent.OnLLMCall(func(...))`) put observability in the core repo's API surface, which violates the stdlib-only contract. They also don't compose: if user wants both retry + observability, they'd register two hooks with no defined ordering. Decorator chain has explicit ordering.

**Example:**
```go
// llm-agent-otel/model.go
package otelmodel

func Wrap(inner llmv2.ChatModel, tp trace.TracerProvider, mp metric.MeterProvider) llmv2.ChatModel {
    base := &wrapper{inner: inner, tracer: tp.Tracer("llm-agent-otel"), meter: mp.Meter(...)}
    // If inner supports tools, return a value that ALSO implements ToolCaller.
    if tc, ok := inner.(llmv2.ToolCaller); ok {
        return &toolCallerWrapper{wrapper: base, inner: tc}
    }
    return base
}

func (w *wrapper) Generate(ctx context.Context, req llmv2.Request) (llmv2.Response, error) {
    ctx, span := w.tracer.Start(ctx, "chat "+w.inner.Info().Model,
        trace.WithAttributes(
            attribute.String("gen_ai.system", w.inner.Info().Name),
            attribute.String("gen_ai.request.model", w.inner.Info().Model),
        ))
    defer span.End()
    resp, err := w.inner.Generate(ctx, req)
    if err != nil {
        span.RecordError(err)
        return resp, err
    }
    span.SetAttributes(
        attribute.Int("gen_ai.usage.input_tokens", resp.Usage.PromptTokens),
        attribute.Int("gen_ai.usage.output_tokens", resp.Usage.CompletionTokens),
    )
    return resp, nil
}
```

### Pattern 2: Functional `WithTools` (immutable)

**What:** Tool binding returns a new model, never mutates the receiver.

**When to use:** Always for capability-extending operations.

**Trade-offs:** Slightly more allocation per request. Eliminates a class of concurrency bugs that Eino's `BindTools` shipped with.

### Pattern 3: Provider-Specific Adapter, Not Generic HTTP Client

**What:** Each provider lives in its own subpackage, imports the official SDK (`openai-go`, `anthropic-sdk-go`), and translates SDK types to/from `llm/v2` types. No "generic HTTP" layer.

**When to use:** When a usable official SDK exists (true for OpenAI, Anthropic). Hand-roll only when no SDK exists or the SDK is unmaintained (Ollama).

**Trade-offs:** Translation code duplicated per provider. But we get bug-fixes and new model support for free, vs. perpetually tracking wire format changes ourselves.

## Project Structure

### Core repo (`llm-agent`)

```
llm-agent/
├── go.mod                  # stdlib-only, no go.sum
├── llm/                    # legacy contract — DEPRECATED after v0.3
│   ├── client.go           # Client (deprecated, kept for one cycle)
│   └── doc.go
├── llm/v2/                 # NEW canonical contract
│   ├── client.go           # ChatModel, ToolCaller, Embedder, StructuredOutputs
│   ├── stream.go           # StreamReader, StreamEvent (typed union)
│   ├── info.go             # ProviderInfo struct
│   ├── types.go            # Message, Tool, ToolCall, Request, Response, Usage
│   └── doc.go              # capability-negotiation guide for provider authors
├── agent.go                # Agent + paradigm constructors (refactored to use v2)
├── react.go                # capability-aware: WithTools fast path + scratchpad fallback
├── function_call.go        # requires ToolCaller (fail-fast at construction)
├── ...rest unchanged: orchestrate/, memory/, rag/, context/, comm/, ...
├── examples/               # add: examples-v2/ that uses llm/v2 directly
└── docs/
    └── PROVIDER_AUTHORING.md  # contract + wire-format expectations
```

### Provider repo (`llm-agent-providers`)

```
llm-agent-providers/
├── go.mod                  # require: llm-agent v0.3.x; openai-go; anthropic-sdk-go
├── openai/
│   ├── model.go            # implements ChatModel + ToolCaller + Embedder + StructuredOutputs
│   ├── stream.go           # OpenAI delta → llm/v2.StreamEvent translation
│   └── model_test.go
├── anthropic/
│   ├── model.go            # implements ChatModel + ToolCaller (no Embedder)
│   ├── stream.go           # content_block_delta → llm/v2.StreamEvent
│   └── model_test.go
├── ollama/
│   ├── model.go            # ChatModel + ToolCaller (model-conditional) + Embedder
│   ├── http.go             # hand-rolled HTTP, no SDK
│   └── model_test.go
└── internal/contract/      # httptest-driven cross-provider conformance suite
    ├── streaming_test.go   # all three exit-loop on iteration
    ├── tools_test.go       # tool_call args round-trip identical
    └── fixtures/           # canned httptest responses per provider
```

### OTel repo (`llm-agent-otel`)

```
llm-agent-otel/
├── go.mod                  # require: llm-agent v0.3.x; go.opentelemetry.io/otel; semconv
├── model.go                # otelmodel.Wrap(ChatModel) — gen_ai.* span attrs
├── agent.go                # otelagent.Wrap(Agent) — agent.run / step.thought / step.action spans
├── slog.go                 # slog handler that bridges to OTel logs
├── metrics.go              # token / latency / cost histograms
└── compose/                # docker-compose with otel-collector + Tempo + Prometheus + Grafana
```

### Ref-service repo (`llm-agent-customer-support`)

```
llm-agent-customer-support/
├── go.mod                  # all three sister modules + chi/echo + sqlite/postgres
├── cmd/server/main.go      # boot order: load config → init otel → factories → http server
├── internal/
│   ├── http/               # /chat (one-shot), /chat/stream (SSE)
│   ├── session/            # session-id → []Message store (sqlite for dev, postgres for prod)
│   ├── factory/            # NewAgent(provider, paradigm) — combines providers + otel + core
│   └── kb/                 # RAG knowledge base wired to llm/v2.Embedder
├── deploy/
│   ├── docker-compose.yml  # service + ollama + otel-collector + grafana
│   └── helm/               # optional Helm chart
└── README.md               # PROVIDER=openai|anthropic|ollama selection
```

## Multi-Repo Dependency Graph

```
                  ┌────────────────────────┐
                  │ llm-agent (core, v0.3) │
                  │  llm/v2/ ChatModel,…   │
                  └─────┬──────────┬───────┘
                        │          │
        depends on      │          │      depends on
                        ▼          ▼
   ┌──────────────────────┐    ┌─────────────────┐
   │ llm-agent-providers  │    │ llm-agent-otel  │
   │       v0.1.x         │    │     v0.1.x      │
   └──────────┬───────────┘    └────────┬────────┘
              │                         │
              │   depends on both       │
              ▼                         ▼
            ┌────────────────────────────┐
            │ llm-agent-customer-support │
            │           v0.1.x           │
            └────────────────────────────┘
```

**Acyclicity proof:**
- `llm-agent` imports nothing in this graph (stdlib only).
- `llm-agent-providers` imports `llm-agent` only. Does NOT import `llm-agent-otel`. (If you want providers + otel, the consumer wraps the provider — that's the decorator pattern's whole point.)
- `llm-agent-otel` imports `llm-agent` only. Does NOT import `llm-agent-providers`. (It wraps any `ChatModel`, regardless of provider.)
- `llm-agent-customer-support` imports all three. It's a leaf.

**Why this is stable under version bumps:**
- Bumping `llm-agent` minor version (e.g., v0.3.0 → v0.3.1) is BC-safe per project policy. Providers and otel don't need to bump.
- Bumping `llm-agent` 0.x major (v0.3 → v0.4) breaks BC. Providers and otel each bump their `require` line and re-tag (their own v0.1 → v0.2). Refsvc updates all three lines together.
- `go.work` at workspace level handles local cross-repo iteration. Workspace-mode is dev-only; published modules use `require` directives, so production consumers see the resolved tag.

**The dependency direction is the same one `go.opentelemetry.io/contrib/instrumentation/...` packages use** — instrumentation packages depend on the library being instrumented + OTel API, never on each other. We're applying that pattern to LLM provider adapters too.

## Suggested Build Order

The dependency graph dictates the order. Each phase produces an independently testable artifact.

```
Phase 1: llm/v2/ scaffolding in core repo
  └─ ChatModel, ToolCaller, Embedder, StructuredOutputs interfaces
  └─ ProviderInfo struct
  └─ StreamReader + typed StreamEvent union
  └─ Message/Tool/ToolCall/Request/Response/Usage types
  └─ A scripted/mock implementation in tests
  └─ DOES NOT TOUCH agent paradigms yet
  └─ Deliverable: a new package that compiles + has 100% type-level test coverage
  └─ Repo: llm-agent

Phase 2: agent refactor — capability-aware paradigms
  └─ Refactor ReAct, FunctionCall, Reflection, PlanSolve, Simple to consume ChatModel
  └─ Capability negotiation: type assertion + ProviderInfo
  └─ ScriptedLLM updated to implement new interfaces
  └─ ALL existing examples in /examples/ migrated to llm/v2
  └─ Old llm.Client retained, marked Deprecated; existing /examples/* keep working
  └─ Deliverable: green CI on core repo with both old + new contract callable
  └─ Repo: llm-agent

Phase 3: First provider — OpenAI (highest market share, well-documented, full-feature)
  └─ NEW REPO: llm-agent-providers
  └─ openai/ subpackage implements all four capabilities
  └─ httptest-driven contract tests (no real API calls in PR CI)
  └─ Deliverable: openai.New(...) returns a usable model; cross-provider contract suite has its first member
  └─ Repo: llm-agent-providers v0.1.0
  └─ Validates: that llm/v2 can actually express OpenAI's streaming + tool semantics

Phase 4: OTel core (parallelizable with Phase 5 once Phase 3 lands)
  └─ NEW REPO: llm-agent-otel
  └─ otelmodel.Wrap + otelagent.Wrap
  └─ gen_ai.* semantic-convention attributes
  └─ slog bridge
  └─ docker-compose verification harness (collector + grafana)
  └─ Deliverable: wrapping the OpenAI provider produces correct spans
  └─ Repo: llm-agent-otel v0.1.0

Phase 5: Anthropic provider — pressure-tests the streaming union
  └─ Anthropic content_block_delta → llm/v2.StreamEvent translation
  └─ Cross-provider contract suite gets second member; identical scenarios MUST pass against both
  └─ Deliverable: Anthropic adapter; abstraction holes (if any) found and patched in llm/v2
  └─ Repo: llm-agent-providers (bump to v0.2.0 if llm/v2 needed any tweaks; otherwise v0.1.x)

Phase 6: Ollama provider — pressure-tests the "no SDK" path + Embedder + capability conditionality
  └─ Hand-rolled HTTP, no SDK
  └─ Tool support is per-model (Llama 3.1 yes, older models no) — first real test of ProviderInfo.SupportsTools
  └─ Embedder for RAG
  └─ Deliverable: third provider; cross-provider contract suite complete
  └─ Repo: llm-agent-providers v0.1.x (or v0.2.x if shape evolved)

Phase 7: Reference service — proves the whole stack
  └─ NEW REPO: llm-agent-customer-support
  └─ HTTP /chat + /chat/stream (SSE)
  └─ Session storage
  └─ Customer-support multi-agent (RAG + StateGraph + tools)
  └─ docker-compose: refsvc + ollama + otel-collector + grafana
  └─ Provider switch via PROVIDER env var
  └─ Deliverable: a single command brings up the whole thing; end-to-end trace visible in Grafana
  └─ Repo: llm-agent-customer-support v0.1.0

Phase 8: Old contract removal (one minor cycle later)
  └─ Delete llm/Client; users have had one minor cycle to migrate
  └─ Deliverable: llm-agent v0.4.0 — only llm/v2 remains
  └─ Repo: llm-agent
```

**Why OpenAI before Anthropic:** Phase 3 validates the abstraction; Phase 5 stress-tests it. If Phase 5 forces an llm/v2 change, only OpenAI needs re-validation, not three providers.

**Why OTel core can run parallel with Anthropic:** OTel only needs `llm/v2.ChatModel` (frozen at end of Phase 1) + the OpenAI provider (frozen at end of Phase 3). It doesn't depend on Phase 5.

**Why ref service is last:** It exercises every preceding phase. Building it earlier means rebuilding it as the abstractions shift.

## Reference Service Architecture (Phase 7)

The question asks "what shape does a deployable LLM service take in 2026?" The honest answer is "there's no single canonical shape." But the consensus across go-llm, ADK Go, and the LangGraph-server pattern is:

```
HTTP Handler (per-request, stateless)
  │
  ├─ ① extract session_id from request
  ├─ ② session.Load(session_id) → []Message  (sqlite/postgres)
  ├─ ③ build a context.Context with timeout + trace
  ├─ ④ NEW agent value PER REQUEST — agents are cheap (just pointer to model + opts)
  ├─ ⑤ agent.RunStream(ctx, input) → <-chan StepEvent
  ├─ ⑥ for ev := range chan { sse.Write(ev) }
  └─ ⑦ session.Save(session_id, append(history, ev.Final))
```

**Per-request agent construction (recommended):**
- Agents are stateless value objects today (`agent.go: ReActAgent{client, opts}`). Construction is cheap.
- Per-request construction means request-scoped tools (e.g., a `database` tool with this user's connection), request-scoped registries (RBAC: this user can call X, not Y), and clean ctx.Done() teardown.
- Goroutine cost is negligible (~2KB stack); 1000 concurrent agents = 2MB.

**NOT a long-running agent process per session:**
- Tempting (skip session-load on each turn) but couples request-handling to a stateful actor model.
- Forces sticky sessions in load balancers, complicates HA, and conflicts with horizontal scaling.
- Only useful if the agent has expensive in-memory state (a vectorized cache, a fine-tuned model). Customer support has neither.

**Worker pool: only for tool execution, not agents.**
- Inside a single agent run, tool calls fan out via `AsyncRunner(MaxParallel)` (already exists in core). Worker pool = bounded parallelism for the tool layer, not the request layer.

**Session storage:** sqlite for dev / single-instance, postgres for HA. The `History []Message` field on `llm/v2.Request` is the sole serialized state per session.

## Scaling Considerations

| Scale | Architecture Adjustments |
|---|---|
| 0–100 concurrent users | Single binary + sqlite. No worker pool needed. docker-compose deployment. |
| 100–10k concurrent users | Postgres for sessions; OTel collector → managed APM. Token-cost metrics drive provider routing. |
| 10k+ users | Read-replica postgres for session reads; per-provider rate limiter (provider quotas, not service capacity, are usually the bottleneck); circuit breaker on each provider with fallback chain (e.g., openai → anthropic on quota error). All these live in the consumer service, NOT in `llm-agent` core. |

### Scaling Priorities

1. **First bottleneck:** Provider rate limits — OpenAI's RPM/TPM cap, not your CPU. Mitigate with per-provider rate limiter in refsvc + multi-provider routing.
2. **Second bottleneck:** Session DB write contention — every turn writes new history. Append-only schema with per-session partition.
3. **Third bottleneck:** Token cost — measure with `gen_ai.usage.*_tokens` metrics, decide which traffic gets premium models.

## Anti-Patterns

### Anti-Pattern 1: Providers depending on the OTel module

**What people do:** Add OTel imports inside the OpenAI provider so spans are emitted "automatically."
**Why it's wrong:** Forces every provider consumer to take an OTel dependency. Couples the two evolution timelines. Users who want a different observability stack (Datadog SDK, Sentry, custom slog handler) can't opt out.
**Do this instead:** Decorator pattern. `otelmodel.Wrap(openai.New(...))`. Composes cleanly; opt-in.

### Anti-Pattern 2: Lowest-common-denominator stream chunks

**What people do:** Define `StreamChunk` as `{Text string; ToolCalls []ToolCall}` and tell adapters "fully accumulate tool calls before emitting."
**Why it's wrong:** Loses streaming UX for tool calls (user sees nothing until full args parsed); buffer pressure; wastes Anthropic's content-block model.
**Do this instead:** Typed event union (`StreamEvent.Kind`) with stable per-tool-call indexing. Adapters emit their native granularity; consumers that don't care use a helper to accumulate.

### Anti-Pattern 3: Bitmask capabilities replacing type assertion

**What people do:** Single fat interface + `Caps() Capability` bitmask.
**Why it's wrong:** Doesn't remove the need to call the capability method; just adds a check. Still need an embedded interface or method-on-interface to actually invoke. Method ends up returning `(result, ErrNotSupported)` — same as type assertion failure but worse (compile-time → runtime).
**Do this instead:** Small interfaces + type assertion + `ProviderInfo` struct as a hint.

### Anti-Pattern 4: `BindTools(tools)` mutating the receiver

**What people do:** Method on the model that stores tools as a field on the receiver.
**Why it's wrong:** Two requests on the same model concurrently? Last-writer-wins on the tools field. Eino shipped this and had to deprecate it.
**Do this instead:** `WithTools(tools) ChatModel` — pure function, returns a new value. No shared mutable state.

### Anti-Pattern 5: Long-running agent process per session

**What people do:** Spawn one goroutine per session that owns the agent + session state for the session's lifetime.
**Why it's wrong:** Sticky-session routing required; restart kills in-flight requests; horizontal scaling breaks; goroutine leak risk if termination logic is buggy.
**Do this instead:** Per-request agent construction; session state in DB; agents are values, not actors.

## Integration Points

### External Services (refsvc)

| Service | Integration Pattern | Notes |
|---|---|---|
| OpenAI / Anthropic / Ollama | `llm/v2.ChatModel` adapters in providers repo | Set via `PROVIDER` env; one adapter per provider. |
| OTel collector | `otelmodel.Wrap` + `otelagent.Wrap`; OTLP/gRPC exporter | Defaults to `otel-collector:4317` in compose. |
| Postgres / sqlite | session storage, knowledge-base persistence | Wrapped behind `session.Store` interface; sqlite default. |
| Grafana / Tempo / Prometheus | OTel collector exports to these | Provided in `deploy/docker-compose.yml`. |

### Internal Boundaries

| Boundary | Communication | Notes |
|---|---|---|
| `llm-agent` ↔ `llm-agent-providers` | type-level: providers implement `llm/v2.ChatModel` (et al.) | Stable across `llm-agent` minor versions. Provider-specific options live as `func(*opts)` in each provider package — never leak into core. |
| `llm-agent` ↔ `llm-agent-otel` | type-level: otel wrappers consume + return `llm/v2.ChatModel` and `agent.Agent` | Same stability contract. |
| `llm-agent-providers` ↔ `llm-agent-otel` | NONE — siblings, never import each other | Composition happens in the leaf consumer (refsvc). |
| Refsvc handler ↔ `agent.Agent` | per-request construction; SSE for streaming | Goroutine + channel; existing `RunStream` returns `<-chan StepEvent` already. |
| Refsvc ↔ session DB | per-request load/save; `[]Message` is the unit | History stored as JSON blob keyed by session_id. |

## Sources

### Primary (Context7 / official docs)

- [tmc/langchaingo Model interface (Context7)](https://github.com/tmc/langchaingo/blob/main/docs/docs/modules/model_io/models/index.mdx) — `Model.GenerateContent` (single fat interface; `WithTools` is a CallOption — alternative path we did not take)
- [cloudwego/eino BaseChatModel + ToolCallingChatModel](https://github.com/cloudwego/eino/blob/main/components/model/interface.go) — small-interface + WithTools pattern (recommended); deprecated `BindTools` lesson learned
- [Eino ChatModel User Guide](https://www.cloudwego.io/docs/eino/core_modules/components/chat_model_guide/) — ToolCalls in stream chunks
- [Genkit Go ai.ModelSupports / DefineModel](https://genkit.dev/go/docs/plugin-authoring-models) — capability-struct (ProviderInfo) pattern
- [OpenTelemetry Semantic Conventions for GenAI Spans](https://opentelemetry.io/docs/specs/semconv/gen-ai/gen-ai-spans/) — `gen_ai.system`, `gen_ai.request.model`, `gen_ai.usage.input_tokens`, span name format
- [OpenTelemetry GenAI Agent + Framework Spans](https://opentelemetry.io/docs/specs/semconv/gen-ai/gen-ai-agent-spans/) — agent-step instrumentation pattern
- [OpenAI Go SDK official](https://github.com/openai/openai-go) — `Chat.Completions.NewStreaming` iterator + `.Accumulate()`
- [Anthropic Streaming Messages spec](https://docs.anthropic.com/en/api/messages-streaming) — `content_block_delta` / `input_json_delta` model
- [Anthropic SDK Go streaming](https://deepwiki.com/anthropics/anthropic-sdk-go/4.3-message-batch-service) — `Accumulate()` method behavior
- [Streaming Tool Calls: Parse Anthropic SSE](https://dev.to/gabrielanhaia/streaming-tool-calls-parse-anthropic-sse-without-loading-the-whole-message-2on) — concrete delta semantics (input_json_delta partial JSON)
- [Go workspaces tutorial](https://go.dev/doc/tutorial/workspaces) — `go.work` for multi-repo dev
- [OpenTelemetry Go instrumentation libraries](https://opentelemetry.io/docs/languages/go/libraries/) — naming conventions, wrapper pattern
- [otelhttp wrapper pattern](https://oneuptime.com/blog/post/2026-02-06-instrument-go-net-http-otelhttp-opentelemetry/view) — decorator-style instrumentation, our model

### Secondary (synthesis sources, MEDIUM confidence)

- [Effective Go on small interfaces](https://go.dev/doc/effective_go) — "the bigger the interface, the weaker the abstraction"
- [Building real-time AI APIs with Go](https://www.dsinnovators.com/blog/golang/ai-apis-golang-concurrency-llm-2026/) — per-request goroutine pattern
- [Stateful Continuation for AI Agents (InfoQ)](https://www.infoq.com/articles/ai-agent-transport-layer/) — stateless HTTP vs stateful actors trade-off
- [How to Implement SSE in Go (freeCodeCamp)](https://www.freecodecamp.org/news/how-to-implement-server-sent-events-in-go/) — SSE handler shape
- [Comparing streaming response structures across LLM APIs](https://medium.com/percolation-labs/comparing-the-streaming-response-structure-for-different-llm-apis-2b8645028b41) — confirms OpenAI delta vs Anthropic content-block divergence

### Local code references

- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/llm/client.go` — current `Client`, `Tool`, `ToolCall`, `StreamChunk` shapes
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/agent.go` — `Agent`, `RunStream`, `runStreamFromBlocking`, `Step` shape (preserved)
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/react.go` — capability negotiation pressure test target
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/function_call.go` — already requires native tool calls; clean target for `ToolCaller` requirement
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/orchestrate/graph.go` — orchestration unaffected by this milestone (operates on `agent.Agent`, not LLM clients)

---
*Architecture research for: Go LLM agent framework, milestone v0.3 — provider abstraction + OTel + multi-repo*
*Researched: 2026-05-10*

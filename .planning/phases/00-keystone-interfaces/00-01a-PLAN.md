---
phase: 00-keystone-interfaces
plan: 01a
type: execute
wave: 1
depends_on: []
files_modified:
  - llm/chatmodel.go
  - llm/capabilities.go
  - llm/stream.go
  - llm/info.go
  - llm/types.go
  - llm/errors.go
  - llm/legacy.go
  - llm/client.go
autonomous: true
requirements:
  - CORE-01
  - CORE-02
  - CORE-03
  - CORE-04
  - CORE-05
  - CORE-06
  - CORE-08
tags:
  - llm
  - interfaces
  - capability-negotiation
  - go-stdlib-only

must_haves:
  truths:
    - "go build ./... succeeds in stdlib-only core repo (no go.sum, no new deps) — every existing agent paradigm compiles unchanged via the type Client = LegacyClient alias"
    - "godoc lists ChatModel, ToolCaller, Embedder, StructuredOutputs, StreamReader, StreamEvent, ProviderInfo, Capabilities as exported types in the llm package"
    - "LegacyClient is callable; type Client = LegacyClient alias preserves source compatibility for every existing v0.2 caller"
    - "// Deprecated: godoc comments target v0.4.0 on LegacyClient, Client alias, GenerateRequest, GenerateResponse, StreamChunk, StreamUsage"
    - "go vet ./... and go build ./... pass across the whole repo"
    - "go.mod has no require block — stdlib-only invariant intact"
  artifacts:
    - path: "llm/chatmodel.go"
      provides: "ChatModel interface (Generate + Stream + Info)"
      contains: "type ChatModel interface"
    - path: "llm/capabilities.go"
      provides: "ToolCaller, Embedder, StructuredOutputs capability interfaces"
      contains: "type ToolCaller interface"
    - path: "llm/stream.go"
      provides: "StreamReader iterator + StreamEvent typed union + StreamEventKind enum + ToolCallDelta + AccumulateStream helper"
      contains: "type StreamReader interface"
    - path: "llm/info.go"
      provides: "ProviderInfo + Capabilities struct (D-02)"
      contains: "type Capabilities struct"
    - path: "llm/types.go"
      provides: "Request, Response, Message, Tool, ToolCall (with ID), Vector, Usage, UsageSource, FinishReason alias"
      contains: "type Request struct"
    - path: "llm/errors.go"
      provides: "ErrCapabilityNotSupported, ErrScriptExhausted sentinels"
      contains: "ErrCapabilityNotSupported"
    - path: "llm/legacy.go"
      provides: "LegacyClient (renamed from Client) + Client alias + companion v0.2 types with // Deprecated: godoc"
      contains: "type LegacyClient interface"
  key_links:
    - from: "llm/legacy.go"
      to: "llm/types.go (FinishReason alias)"
      via: "type FinishReason = legacyFinishReason"
      pattern: "type FinishReason = legacyFinishReason"
    - from: "llm/types.go"
      to: "llm/legacy.go (legacyFinishReason underlying type)"
      via: "alias declaration"
      pattern: "= legacyFinishReason"
    - from: "existing agent paradigms (simple.go, react.go, etc.)"
      to: "llm/legacy.go (Client = LegacyClient alias)"
      via: "alias preserves call-site compatibility"
      pattern: "type Client = LegacyClient"
---

<objective>
Lock the **contract surface** of the v0.3 `llm/` reboot per D-01 (CONTEXT.md). This plan ratifies the K1/K2/K3 keystones at the type level: capability interfaces (`ChatModel`, `ToolCaller`, `Embedder`, `StructuredOutputs`), typed `StreamEvent` union with stable per-tool-call `Index`, per-(provider × model) `ProviderInfo` shape (D-02), shared `types.go` (Request/Response/Tool/ToolCall-with-ID/Message/Vector/Usage/UsageSource/FinishReason alias), sentinel errors, and the `LegacyClient` rename with `// Deprecated:` godoc + `Client = LegacyClient` alias.

Existing agent paradigms (`simple.go`, `react.go`, etc.) MUST continue to compile unchanged via the alias — this plan is purely additive at every existing call site.

Purpose: Plan 00-01b (Wave 1, sequenced after 01a) builds the ScriptedLLM v2 + ChatOnlyMock + doc.go + tests on top of THIS plan's contract surface. Splitting the original 7-task plan into 01a (contract) + 01b (mocks/tests) keeps each plan within ~50% context budget per the planner's anti-shallow rules and clarifies the dependency: 01b's compile-time `var _ ChatModel = (*ScriptedLLM)(nil)` cannot exist until 01a's `ChatModel` is declared.

Output: 7 new/rewritten files in `llm/` (chatmodel.go, capabilities.go, stream.go, info.go, types.go, errors.go, legacy.go). The current `llm/client.go` is renamed (via `git mv`) to `llm/legacy.go`.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/ROADMAP.md
@.planning/STATE.md
@.planning/REQUIREMENTS.md
@.planning/phases/00-keystone-interfaces/00-CONTEXT.md
@.planning/phases/00-keystone-interfaces/00-RESEARCH.md
@.planning/phases/00-keystone-interfaces/00-PATTERNS.md
@.planning/phases/00-keystone-interfaces/00-VALIDATION.md
@CLAUDE.md

# Existing source-of-truth files being rebooted / referenced
@llm/client.go
@llm/doc.go
@agent.go
@tool.go

<interfaces>
<!-- Existing interfaces / types in `llm/client.go` (lines from current file) that this plan REPLACES (verbatim move to legacy.go). -->

From llm/client.go (current — to be relocated to llm/legacy.go with deprecation comments):
```go
type Client interface {
    Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error)
    GenerateStream(ctx context.Context, req GenerateRequest) (<-chan StreamChunk, error)
}
type GenerateRequest struct {
    Prompt  string         `json:"prompt"`
    Context map[string]any `json:"context,omitempty"`
    Tools   []Tool         `json:"tools,omitempty"`
    History []Message      `json:"history,omitempty"`
}
type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}
type GenerateResponse struct {
    Text         string         `json:"text"`
    FinishReason FinishReason   `json:"finish_reason,omitempty"`
    Provider     string         `json:"provider"`
    Model        string         `json:"model,omitempty"`
    UsageToken   int            `json:"usage_token,omitempty"`
    Metadata     map[string]any `json:"metadata,omitempty"`
    ToolCalls    []ToolCall     `json:"tool_calls,omitempty"`
}
type FinishReason string
const (
    FinishReasonStop FinishReason = "stop"
    FinishReasonLength FinishReason = "length"
    FinishReasonContentFilter FinishReason = "content_filter"
    FinishReasonToolCalls FinishReason = "tool_calls"
    FinishReasonFunctionCall FinishReason = "function_call"
    FinishReasonUnknown FinishReason = "unknown"
)
type StreamChunk struct {
    Text     string       `json:"text"`
    Done     bool         `json:"done"`
    Usage    *StreamUsage `json:"usage,omitempty"`
    ToolCall *ToolCall    `json:"tool_call,omitempty"`
}
type StreamUsage struct {
    PromptTokens     int `json:"prompt_tokens,omitempty"`
    CompletionTokens int `json:"completion_tokens,omitempty"`
    TotalTokens      int `json:"total_tokens,omitempty"`
}
type Tool struct {
    Name        string          `json:"name"`
    Description string          `json:"description"`
    Parameters  json.RawMessage `json:"parameters"`
}
type ToolCall struct {  // legacy shape — NO ID field; new types.go ToolCall ADDS ID
    Name      string          `json:"name"`
    Arguments json.RawMessage `json:"arguments"`
}
```

Existing callers that must continue to compile via `type Client = LegacyClient` alias (verified by grep — see RESEARCH.md §"Internal llm.Client user inventory"):
- `simple.go`, `react.go`, `function_call.go`, `reflection.go`, `plan_solve.go` — agent paradigms
- `tool.go`, `registry.go` — agent tool dispatch
- `rag/rag.go`, `bench/judge.go`, `bench/winrate.go`, `context/builder.go`, `rl/trainer_proxy.go`
- `examples/scriptedllm/scriptedllm.go` (separate Go module under `examples/`)
- All `*_test.go` test files in repo root that use `scriptedLLM`, `newScriptedLLM`, `textResp`

Phase 3 is the migration phase (CORE-10) — DO NOT touch any of these files in this plan beyond what is explicitly listed in `files_modified`.
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Rename llm/client.go to llm/legacy.go and add // Deprecated: comments + Client alias + extract FinishReason underlying type</name>

  <files>llm/legacy.go (NEW — content lifted from current llm/client.go), llm/client.go (DELETED via rename)</files>

  <read_first>
    - llm/client.go (current — entire file; this becomes legacy.go verbatim with rename + deprecation comments + alias)
    - llm/doc.go (current — leave for now; plan 00-01b Task 5 replaces it)
    - .planning/phases/00-keystone-interfaces/00-PATTERNS.md §"`llm/legacy.go` (renamed from `client.go`)" lines 477-510 — exact deprecation comment template
    - .planning/phases/00-keystone-interfaces/00-RESEARCH.md §"`llm/legacy.go`" lines 587-633 — full file body + FinishReason alias decision
    - .planning/phases/00-keystone-interfaces/00-CONTEXT.md §"Specifics" line 167 — exact Deprecated: comment string
  </read_first>

  <behavior>
    - Test 1: `var _ llm.Client = (llm.LegacyClient)(nil)` compiles (alias resolves both directions)
    - Test 2: `var _ llm.LegacyClient = (llm.Client)(nil)` compiles (symmetric reverse)
    - Test 3: `string(llm.FinishReasonStop) == "stop"` (constants stay public + named)
    - Test 4: `grep -c '^// Deprecated:' llm/legacy.go` returns >= 5 (one per: LegacyClient, Client alias, GenerateRequest, GenerateResponse, StreamChunk; StreamUsage and the legacy ToolCall optional)
    - Test 5: existing `simple.go`, `react.go`, etc. compile unchanged (asserted by `go build ./...`)
  </behavior>

  <action>
Step 1: Move `llm/client.go` to `llm/legacy.go` (use `git mv` so history follows). Open the new file in edit mode.

Step 2: Make these edits IN ORDER:

(a) Replace the existing package-doc-less header (current line 1 starts with `// Package llm owns the LLM-provider contract...`) with a NEW comment-less header (the package doc is moving to `llm/doc.go` — landed in plan 00-01b Task 5). The file should open with `package llm` directly. Keep the existing `import ( "context"; "encoding/json" )` block.

(b) Rename the `Client` interface to `LegacyClient`, prepend godoc + Deprecated comment EXACTLY as below, and ADD a `Client = LegacyClient` alias right after:

```go
// LegacyClient is the v0.2 LLM contract — superseded by ChatModel.
//
// Deprecated: Use llm.ChatModel instead. LegacyClient will be removed in v0.4.0. See docs/migration-v0.2-to-v0.3.md.
type LegacyClient interface {
	Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error)
	GenerateStream(ctx context.Context, req GenerateRequest) (<-chan StreamChunk, error)
}

// Client is an alias for LegacyClient retained for v0.2 source compatibility.
//
// Deprecated: Use llm.ChatModel instead. Client will be removed in v0.4.0. See docs/migration-v0.2-to-v0.3.md.
type Client = LegacyClient
```

(c) Move the `Tool`, `ToolCall`, and `Message` type declarations OUT of `legacy.go` — they belong in `llm/types.go` (Task 3) which CREATES new versions (Tool unchanged shape; ToolCall ADDS `ID string`; Message unchanged shape). DELETE these three type declarations from `legacy.go`. The `import "encoding/json"` line is no longer needed in legacy.go after removing `Tool` and `ToolCall` (which used `json.RawMessage`); REMOVE that import. Keep `import "context"`.

(d) Add `// Deprecated: Use llm.Request instead.` (separated from the existing godoc by `//\n//`) above `GenerateRequest`. Same pattern for `GenerateResponse` (`Use llm.Response instead.`), `StreamChunk` (`Use llm.StreamEvent instead.`), `StreamUsage` (`Use llm.Usage instead.`).

(e) Rename the public `FinishReason` string type to a private alias-target `legacyFinishReason`:
```go
// legacyFinishReason is the underlying string type for FinishReason. The
// public FinishReason name is declared in types.go as `type FinishReason = legacyFinishReason`
// so legacy callers and new code see the same type.
type legacyFinishReason string
```
Then update the `const ( FinishReasonStop FinishReason = "stop" ... )` block — KEEP using the public alias name `FinishReason` (the alias declaration in types.go makes `FinishReason` refer to `legacyFinishReason` so this still compiles). The constants stay public and named identically.

Final shape of `llm/legacy.go`: package declaration, single import (`context`), `LegacyClient` interface (deprecated), `Client = LegacyClient` alias (deprecated), `GenerateRequest` struct (deprecated), `GenerateResponse` struct (deprecated), `legacyFinishReason` private type, public `FinishReason` constant block, `StreamChunk` struct (deprecated), `StreamUsage` struct (deprecated). NOTHING ELSE in this file. Total ≈ 80 LOC.

The `Tool`, `ToolCall`, `Message` types previously co-located here are deliberately moved to `types.go` so the new `ChatModel` types live in one consistent file. The package declaration `package llm` means callers writing `llm.Tool` still resolve correctly because `types.go` (Task 3) declares them in the same package.

This is a verbatim file move plus 5 narrow textual changes. NO behavioural change. Per RESEARCH.md A8, the alias preserves source compatibility for every existing caller verified in the grep audit.
  </action>

  <verify>
    <automated>
      cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent && \
      test ! -f llm/client.go && \
      test -f llm/legacy.go && \
      grep -c '^// Deprecated:' llm/legacy.go | awk '$1 >= 5 {exit 0} {exit 1}' && \
      grep -q '^type LegacyClient interface' llm/legacy.go && \
      grep -q '^type Client = LegacyClient' llm/legacy.go && \
      grep -q '^type legacyFinishReason string' llm/legacy.go
    </automated>
  </verify>

  <done>
    - llm/client.go no longer exists (renamed via `git mv`, history preserved).
    - llm/legacy.go exists with LegacyClient, Client alias, GenerateRequest, GenerateResponse, StreamChunk, StreamUsage, and a private `legacyFinishReason string` type + public `FinishReason*` constants.
    - At least 5 `// Deprecated:` comments are present (one per superseded public symbol). Each names `v0.4.0` as removal target and links `docs/migration-v0.2-to-v0.3.md`.
    - The Tool, ToolCall, Message types are NOT in legacy.go (they move to types.go in Task 3).
    - `go vet ./llm/...` passes; `go build ./...` does NOT yet pass (Tool/ToolCall/Message are referenced by other repo files and live in the future types.go — this is intentional; Task 3 closes the gap. The Wave-level acceptance is `go build ./...` green AFTER all 3 tasks land, not after Task 1 alone).
  </done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Create the new ChatModel + capability interfaces + ProviderInfo + StreamEvent + sentinel errors (the K1/K2/K3 contract)</name>

  <files>llm/chatmodel.go (NEW), llm/capabilities.go (NEW), llm/info.go (NEW), llm/stream.go (NEW), llm/errors.go (NEW)</files>

  <read_first>
    - llm/legacy.go (after Task 1 — the deprecated baseline)
    - .planning/phases/00-keystone-interfaces/00-RESEARCH.md §"Concrete Go Type Definitions for `llm/` Reboot" lines 311-585 — every type body, exact field names, decision rationale (Info() on ChatModel; Embedder NOT embedding ChatModel; WithSchema returning ChatModel; ArgsDelta naming)
    - .planning/phases/00-keystone-interfaces/00-PATTERNS.md §"`llm/chatmodel.go`" lines 73-118 + §"`llm/capabilities.go`" lines 124-146 + §"`llm/stream.go`" lines 152-202 + §"`llm/info.go`" lines 209-244 + §"`llm/errors.go`" lines 283-316 — godoc style, json tag conventions, sentinel-error block format
    - .planning/phases/00-keystone-interfaces/00-CONTEXT.md §"D-02" lines 39-55 — Capabilities exact shape; Tools, Embeddings, StructuredOutputs, PromptCaching as bool fields
    - tool.go (current) — multi-method interface with multi-paragraph godoc analog (lines 16-21)
    - agent.go (current) — sentinel-error block analog (lines 127-136)
  </read_first>

  <behavior>
    - Test 1: After plan 00-01b Task 6 lands ScriptedLLM, `var _ ChatModel = (*ScriptedLLM)(nil)` compiles (interface declared here)
    - Test 2: `ProviderInfo{Provider: "p", Model: "m", Capabilities: Capabilities{Tools: true}}` JSON-round-trips with `{"provider":"p","model":"m","capabilities":{"tools":true,"embeddings":false,"structured_outputs":false,"prompt_caching":false}}` (snake_case keys, all bool fields rendered)
    - Test 3: `StreamEventKind` has 6 variants (`EventTextDelta`, `EventToolCallStart`, `EventToolCallArgsDelta`, `EventToolCallEnd`, `EventThinkingDelta`, `EventDone`); iota assignment makes them 0..5
    - Test 4: `errors.Is(fmt.Errorf("anthropic: embeddings: %w", llm.ErrCapabilityNotSupported), llm.ErrCapabilityNotSupported)` is true
    - Test 5: `errors.Is(fmt.Errorf("script: %w", llm.ErrScriptExhausted), llm.ErrScriptExhausted)` is true
  </behavior>

  <action>
Create FIVE new files in `llm/`. Use the EXACT bodies as specified in the original 00-01-PLAN (this is now identified as 00-01a Task 2; the bodies were preserved verbatim from the unsplit plan). They ratify D-02 and embed every field naming decision from RESEARCH.md §"Concrete Go Type Definitions" verbatim.

---

### File 1: `llm/chatmodel.go`

```go
package llm

import "context"

// ChatModel is the base contract every provider implements. It is the
// smallest possible interface: Generate (one-shot), Stream (iterator),
// Info (per-(provider × model) identity).
//
// Capabilities beyond plain text generation are expressed as separate
// interfaces (ToolCaller, Embedder, StructuredOutputs); callers detect
// them via type assertion. ProviderInfo.Capabilities is the runtime
// signal for per-(provider × model) variation that type assertion
// cannot see — see doc.go for the canonical negotiation idiom.
//
// All implementations MUST be safe for concurrent use; concurrent
// Generate / Stream calls on the same value are part of the contract.
type ChatModel interface {
	Generate(ctx context.Context, req Request) (Response, error)
	Stream(ctx context.Context, req Request) (StreamReader, error)
	Info() ProviderInfo
}
```

---

### File 2: `llm/capabilities.go`

```go
package llm

import "context"

// ToolCaller is the capability for native tool/function-calling.
// WithTools is IMMUTABLE: it returns a new ToolCaller bound to the
// given tools; the receiver is unchanged. This rejects Eino's
// deprecated BindTools mutation pattern (concurrent calls on the same
// model with different tool sets would otherwise race).
//
// Implementations MUST satisfy ChatModel — a tool-bound model is still
// a ChatModel that can Generate / Stream.
type ToolCaller interface {
	ChatModel
	WithTools(tools []Tool) (ToolCaller, error)
}

// Embedder is the capability for vector embeddings. Returns vectors in
// input order with len(vectors) == len(texts). Providers without
// embedding endpoints (Anthropic, in v0.3) do NOT implement this
// interface; callers detect via type assertion AND consult
// Capabilities.Embeddings on the bound ProviderInfo.
//
// Embedder deliberately does NOT embed ChatModel. A pure embedding-only
// adapter (e.g., a future voyageai adapter) might implement Embedder
// without ChatModel — orthogonality preserves that option.
type Embedder interface {
	Embed(ctx context.Context, texts []string) (vectors []Vector, usage Usage, err error)
	EmbedDimensions() int
}

// StructuredOutputs is the capability for JSON-schema-constrained
// generation (OpenAI response_format, Anthropic tool-as-output trick).
//
// WithSchema is IMMUTABLE — like WithTools — and returns ChatModel
// (NOT StructuredOutputs): re-applying a schema is meaningless, so the
// return type signals that the value is now schema-bound and a second
// WithSchema call is not the intended call shape.
type StructuredOutputs interface {
	ChatModel
	WithSchema(schema []byte) (ChatModel, error)
}
```

---

### File 3: `llm/info.go` (D-02 ratification)

```go
package llm

// ProviderInfo describes a bound provider+model combination. Returned
// by ChatModel.Info(). Capabilities reflect THIS bound model, not the
// provider type generically (Pitfall 6). Provider instances bind a
// model at construction time — `openai.New(openai.WithModel("gpt-4o"))`
// — so Info() is constant for the lifetime of the value.
type ProviderInfo struct {
	Provider     string       `json:"provider"`     // "openai", "anthropic", "ollama"
	Model        string       `json:"model"`        // "gpt-4o-mini", "claude-3-5-haiku", "llama3.1:8b"
	Capabilities Capabilities `json:"capabilities"`
}

// Capabilities is a value type — JSON-serializable for OTel attribute
// emission (gen_ai.provider.capabilities.* in Phase 5). Per D-02, this
// is a struct (NOT methods, NOT a bitmask): self-documenting in test
// failures, extensible with non-bool fields later (e.g.,
// MaxToolsPerCall int) without breaking JSON consumers.
//
// Type assertion remains the PRIMARY signal at compile time
// (`if tc, ok := model.(ToolCaller); ok { ... }`); Capabilities is the
// RUNTIME signal for per-(provider × model) variation that type
// assertion cannot see — Ollama's Go type implements ToolCaller, but
// for `llama2` the Capabilities.Tools bool is false.
type Capabilities struct {
	Tools             bool `json:"tools"`               // Native function-calling supported by the bound model
	Embeddings        bool `json:"embeddings"`          // Embed() returns vectors (NOT ErrCapabilityNotSupported)
	StructuredOutputs bool `json:"structured_outputs"`  // WithSchema() applies a JSON schema constraint
	PromptCaching     bool `json:"prompt_caching"`      // Anthropic explicit / OpenAI auto (consumed Phase 5+)
}
```

---

### File 4: `llm/stream.go` (K1 ratification)

```go
package llm

// StreamReader is the iterator-style interface for streaming responses.
// Next returns io.EOF (from package "io") when the stream ends cleanly,
// or ctx.Err() when the underlying context is cancelled. Close is
// idempotent and MUST be called by every consumer (typically via
// `defer sr.Close()`) to prevent goroutine leaks (Pitfall 3).
//
// Iterator (rather than <-chan StreamEvent) is chosen for: explicit
// cancellation semantics, single-call error propagation, prevention of
// producer-goroutine leaks when consumers break out early, and
// composability with the K4 retry state machine (Phase 2).
type StreamReader interface {
	Next() (StreamEvent, error)
	Close() error
}

// StreamEventKind enumerates the typed-union variants. Adapters emit
// their NATIVE granularity (OpenAI per-index deltas, Anthropic per-
// content-block deltas, Ollama whole-tool-call). Consumers that don't
// care about granularity use AccumulateStream below.
type StreamEventKind uint8

const (
	EventTextDelta         StreamEventKind = iota // adapter emitted text
	EventToolCallStart                            // tool_call begins; ToolCall.{Index, ID, Name} known
	EventToolCallArgsDelta                        // partial args JSON for an in-flight tool_call
	EventToolCallEnd                              // tool_call complete; consumer may dispatch
	EventThinkingDelta                            // reasoning models / Anthropic thinking blocks
	EventDone                                     // terminal; Usage + FinishReason populated
)

// StreamEvent is the typed union. Field population is gated by Kind:
//
//	Kind = EventTextDelta:         Text != ""
//	Kind = EventToolCallStart:     ToolCall != nil; ToolCall.{Index, ID, Name} populated
//	Kind = EventToolCallArgsDelta: ToolCall != nil; ToolCall.{Index, ArgsDelta} populated
//	Kind = EventToolCallEnd:       ToolCall != nil; ToolCall.Index populated
//	Kind = EventThinkingDelta:     Text != ""
//	Kind = EventDone:               Usage != nil; FinishReason != ""
type StreamEvent struct {
	Kind         StreamEventKind
	Text         string         // EventTextDelta, EventThinkingDelta
	ToolCall     *ToolCallDelta // EventToolCall* kinds
	Usage        *Usage         // EventDone (when provider reports it)
	FinishReason FinishReason   // EventDone
}

// ToolCallDelta carries per-tool-call streaming state.
//
// Index is the STABLE per-tool-call key: across all chunks for a single
// tool call, Index is identical. The agent-layer accumulator joins by
// Index, NOT by Name (Pitfall 1: "OpenAI streaming tool_calls — losing
// chunks because you keyed by name instead of index").
//
// ID is the provider-side identifier (OpenAI tool_call_id, Anthropic
// content_block id) — used by the agent dedupe layer (Phase 3) keyed
// by (message_id, tool_use_id).
//
// Name is populated ONCE on the EventToolCallStart event for that Index.
// ArgsDelta is the partial JSON string; concatenation across chunks
// for a given Index yields the final arguments JSON (matches OpenAI's
// function.arguments delta string and Anthropic's
// input_json_delta.partial_json).
type ToolCallDelta struct {
	Index     int    // stable across chunks for a single tool call
	ID        string // provider-assigned ID; empty until provider emits it
	Name      string // populated on EventToolCallStart
	ArgsDelta string // partial JSON; concat all deltas for this Index to get final args
}

// AccumulateStream is a convenience for consumers that don't care about
// streaming granularity — drains sr to completion and returns the
// equivalent non-streaming Response. Closes sr on exit (caller need not
// defer Close when using this helper).
func AccumulateStream(sr StreamReader) (Response, error) {
	defer sr.Close()
	var out Response
	for {
		ev, err := sr.Next()
		if err != nil {
			// io.EOF is reported via the standard sentinel — caller can
			// distinguish via errors.Is(err, io.EOF). We surface it as
			// nil so the typical caller treats clean termination as
			// success.
			if isEOF(err) {
				return out, nil
			}
			return out, err
		}
		switch ev.Kind {
		case EventTextDelta:
			out.Text += ev.Text
		case EventToolCallStart, EventToolCallArgsDelta, EventToolCallEnd:
			// Accumulator concatenates deltas keyed by ToolCall.Index.
			// For Phase 0 we only need the helper to compile and pass
			// trivial smoke tests; per-Index fan-in is built up
			// when Phase 2 lands the streaming adapters. Until then,
			// store the latest ToolCall snapshot so out.ToolCalls
			// reflects something coherent in the helper-only path.
			if ev.ToolCall != nil {
				out.ToolCalls = appendToolCallDelta(out.ToolCalls, ev.ToolCall)
			}
		case EventThinkingDelta:
			// Drop thinking deltas in the accumulator helper — the
			// non-streaming Response shape has no thinking field. Phase
			// 5 OTel exporter captures these on spans separately.
		case EventDone:
			if ev.Usage != nil {
				out.Usage = *ev.Usage
			}
			out.FinishReason = ev.FinishReason
		}
	}
}

// isEOF is a small indirection so stream.go does not import "io"
// directly at this layer (the SR implementations import it). EOF
// semantics are detected by the sentinel returned by Next, which is
// always io.EOF when the stream ends cleanly.
func isEOF(err error) bool {
	return err != nil && err.Error() == "EOF"
}

// appendToolCallDelta merges a delta into the accumulated ToolCalls
// slice keyed by Index. A new Index appends; an existing Index extends
// Arguments. NOTE: this helper exists so tests of AccumulateStream
// compile and round-trip; it is NOT the production accumulator that
// Phase 2 will write.
func appendToolCallDelta(existing []ToolCall, d *ToolCallDelta) []ToolCall {
	for i := range existing {
		if existing[i].ID != "" && existing[i].ID == d.ID {
			existing[i].Arguments = append(existing[i].Arguments, []byte(d.ArgsDelta)...)
			return existing
		}
	}
	if d.Name == "" {
		return existing
	}
	return append(existing, ToolCall{
		ID:        d.ID,
		Name:      d.Name,
		Arguments: []byte(d.ArgsDelta),
	})
}
```

NOTE on `isEOF`: the indirection avoids importing `io` in `stream.go` and keeps the file's import set empty. SR implementations (in `scripted.go`, plan 00-01b) import `io` to emit `io.EOF` from their own Next; the matcher here just compares the error string. This is intentional minimalism for the Phase 0 helper. Phase 2's real accumulator imports `io` and uses `errors.Is(err, io.EOF)` directly.

---

### File 5: `llm/errors.go`

```go
package llm

import "errors"

// Sentinel errors for the llm package. Callers detect via errors.Is.
// Both sentinels MUST survive `fmt.Errorf("...: %w", sentinel)` wrapping.
var (
	// ErrCapabilityNotSupported is returned by methods on capability
	// interfaces when the bound model does not actually support the
	// capability — even though the Go type implements the interface.
	//
	// Canonical wrap pattern:
	//   return nil, fmt.Errorf("anthropic: embeddings: %w", llm.ErrCapabilityNotSupported)
	//
	// Callers detect with errors.Is(err, llm.ErrCapabilityNotSupported).
	ErrCapabilityNotSupported = errors.New("llm: capability not supported by bound model")

	// ErrScriptExhausted is returned by ScriptedLLM when the script runs
	// out of pre-recorded responses. Test code matches with errors.Is.
	ErrScriptExhausted = errors.New("llm: scripted llm: script exhausted")
)
```

---

ALL files use stdlib-only imports (`context` only). No new dependencies are added; the core repo's `go.mod` MUST stay at:
```
module github.com/costa92/llm-agent

go 1.26.0
```
No `require` block. Verify by running `go mod tidy` and confirming no diff.
  </action>

  <verify>
    <automated>
      cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent && \
      test -f llm/chatmodel.go && test -f llm/capabilities.go && test -f llm/info.go && test -f llm/stream.go && test -f llm/errors.go && \
      go vet ./llm/... && \
      grep -c '^type Capabilities struct' llm/info.go | awk '$1 == 1 {exit 0} {exit 1}' && \
      grep -c 'EventTextDelta\|EventToolCallStart\|EventToolCallArgsDelta\|EventToolCallEnd\|EventThinkingDelta\|EventDone' llm/stream.go | awk '$1 >= 6 {exit 0} {exit 1}' && \
      grep -q 'ErrCapabilityNotSupported = errors.New' llm/errors.go && \
      grep -q 'ErrScriptExhausted = errors.New' llm/errors.go && \
      ! grep -q '^require' go.mod
    </automated>
  </verify>

  <done>
    - 5 new files exist in `llm/` with the bodies above; `go vet ./llm/...` passes.
    - `Capabilities` has exactly 4 bool fields with snake_case JSON tags: `tools`, `embeddings`, `structured_outputs`, `prompt_caching`.
    - `StreamEventKind` has 6 enum variants; `ToolCallDelta` has Index/ID/Name/ArgsDelta in that order.
    - `ToolCaller` and `StructuredOutputs` BOTH embed `ChatModel`; `Embedder` does NOT embed it.
    - Sentinel errors compile, are wrappable, and `errors.Is` round-trips.
    - `go.mod` has no `require` block — stdlib-only invariant intact.
  </done>
</task>

<task type="auto" tdd="true">
  <name>Task 3: Create llm/types.go (Request, Response, Message, Tool, ToolCall with new ID field, Vector, Usage + UsageSource, FinishReason alias)</name>

  <files>llm/types.go (NEW)</files>

  <read_first>
    - llm/legacy.go (after Task 1 — note the Tool/ToolCall/Message types were REMOVED there; types.go re-introduces them in the same package)
    - .planning/phases/00-keystone-interfaces/00-RESEARCH.md §"`llm/types.go`" lines 487-563 — exact field bodies; decision on shared types between LegacyClient and ChatModel; ToolCall.ID addition rationale (Pitfall 4 dedupe key)
    - .planning/phases/00-keystone-interfaces/00-PATTERNS.md §"`llm/types.go`" lines 246-280 — json.RawMessage convention, multi-paragraph godoc style
    - .planning/phases/00-keystone-interfaces/00-CONTEXT.md §"Q1 in research" — types are SHARED between LegacyClient and ChatModel (Q1 RESOLVED in RESEARCH.md: share)
    - llm/client.go (now legacy.go after Task 1) — original Tool/ToolCall/Message shapes, must remain backward-compatible at the field-name level
  </read_first>

  <behavior>
    - Test 1: `llm.Tool{Name: "x", Description: "y", Parameters: json.RawMessage(`{}`)}` JSON-serialises with `name`, `description`, `parameters` keys (snake_case)
    - Test 2: `llm.ToolCall{ID: "id1", Name: "calc", Arguments: json.RawMessage(`{"a":2}`)}` JSON-serialises with `id`, `name`, `arguments` keys; the new ID field is the dedupe key for Pitfall 4
    - Test 3: `var fr llm.FinishReason = llm.FinishReasonStop; string(fr) == "stop"` — FinishReason alias resolves to legacyFinishReason underlying type
    - Test 4: `usage := llm.Usage{Source: llm.UsageReported, InputTokens: 10}` — UsageSource has Reported/Estimated/Unknown constants
    - Test 5: `llm.Vector` is `[]float32`
    - Test 6: existing `simple.go` (which uses `llm.Tool`, `llm.Message`, `llm.GenerateResponse.ToolCalls` of type `[]llm.ToolCall`) compiles unchanged (the new `ToolCall` adds an `ID` field but does not remove any field, so v0.2 callers that don't construct ToolCalls explicitly continue to work)
  </behavior>

  <action>
Create `llm/types.go` with the EXACT body below. This file is in the SAME package as `legacy.go`, so the types declared here are accessible to both legacy callers (via field-shape compatibility) and new code (via the ChatModel interface).

```go
package llm

import "encoding/json"

// Request is the new-surface request type. Replaces GenerateRequest at
// the new-interface (ChatModel) layer. LegacyClient continues to use
// GenerateRequest (defined in legacy.go).
//
// Why a separate type: the v0.2 GenerateRequest used Prompt+History;
// the v0.3 surface is messages-only with SystemPrompt lifted out so
// Anthropic's top-level system parameter has a clean home and OpenAI's
// system-role message can be derived from it.
type Request struct {
	Messages        []Message      `json:"messages"`                    // multi-turn dialog (preferred over Prompt)
	SystemPrompt    string         `json:"system_prompt,omitempty"`     // lifted out of Messages for Anthropic top-level system
	MaxOutputTokens int            `json:"max_output_tokens,omitempty"` // 0 = use provider default
	Temperature     *float32       `json:"temperature,omitempty"`       // pointer: nil = use provider default
	Metadata        map[string]any `json:"metadata,omitempty"`          // provider-specific extras (rare; prefer typed)
}

// Response is the new-surface response type. Replaces GenerateResponse
// at the ChatModel layer.
type Response struct {
	Text         string       `json:"text"`
	FinishReason FinishReason `json:"finish_reason,omitempty"`
	Provider     string       `json:"provider"`
	Model        string       `json:"model,omitempty"`
	Usage        Usage        `json:"usage"`
	ToolCalls    []ToolCall   `json:"tool_calls,omitempty"`
}

// Message is a single turn in a conversation. Reused unchanged from
// the v0.2 surface — same Role/Content shape. System messages are
// lifted to Request.SystemPrompt before sending to providers; the
// "system" role string remains valid for callers that prefer
// embedding it in Messages (LegacyClient flow).
type Message struct {
	Role    string `json:"role"`    // "user", "assistant", "tool", "system"
	Content string `json:"content"`
}

// Tool declares a function the model may call. Parameters is a raw
// JSON Schema document — this package doesn't validate it (the
// upstream provider does) so callers can use whatever schema dialect
// their provider expects.
//
// Shape unchanged from v0.2; shared between LegacyClient and ChatModel
// surfaces deliberately (the field names haven't needed to evolve in
// 6 months of v0.2; sharing avoids two parallel type systems).
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ToolCall is what the model returns when it decides to invoke a Tool.
// The ID field is NEW vs. v0.2's ToolCall — it's the provider-assigned
// identifier (OpenAI tool_call_id, Anthropic content_block.id) that
// the agent dedupe layer (Phase 3) uses as one half of the
// (message_id, tool_use_id) dedupe key (Pitfall 4).
//
// v0.2 callers that READ ToolCalls keep working because the model is
// the only producer of ToolCall values — v0.2 did not construct
// ToolCalls explicitly, so adding an ID field is back-compat.
type ToolCall struct {
	ID        string          `json:"id,omitempty"` // provider-assigned; NEW vs v0.2
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// Vector is one embedding. Length matches Embedder.EmbedDimensions().
type Vector []float32

// Usage carries token accounting for one request. Source distinguishes
// reported (provider returned actual counts), estimated (computed from
// tokenizer), and unknown (mid-stream abort, no usage available).
//
// Source != "" is an invariant after Phase 2 lands (K4); for Phase 0
// the Source field exists but defaults to UsageUnknown when the zero
// value is used.
type Usage struct {
	InputTokens  int         `json:"input_tokens"`
	OutputTokens int         `json:"output_tokens"`
	TotalTokens  int         `json:"total_tokens,omitempty"`
	Source       UsageSource `json:"source,omitempty"`
}

// UsageSource enumerates the provenance of token counts in a Usage.
// Reported = provider returned actual counts; Estimated = computed
// from a tokenizer; Unknown = mid-stream abort, no usage available
// (NOT zero-tokens — Pitfall 5).
type UsageSource string

const (
	UsageReported  UsageSource = "reported"
	UsageEstimated UsageSource = "estimated"
	UsageUnknown   UsageSource = "unknown"
)

// FinishReason is an alias for the underlying legacyFinishReason
// string type defined in legacy.go. The alias means LegacyClient and
// ChatModel callers see the same FinishReason name and the same
// constant set (FinishReasonStop, FinishReasonLength, etc.) — type
// identity is preserved across the v0.2 / v0.3 surfaces.
type FinishReason = legacyFinishReason
```

After creating `types.go`, verify the whole repo compiles:
```
go build ./...
```
This MUST succeed: existing `simple.go`, `react.go`, `function_call.go`, `reflection.go`, `plan_solve.go`, `tool.go`, `registry.go`, `rag/rag.go`, `bench/judge.go`, `bench/winrate.go`, `context/builder.go`, `rl/trainer_proxy.go` reference `llm.Tool`, `llm.Message`, `llm.GenerateRequest`, `llm.GenerateResponse`, `llm.Client` — all those names are now resolvable via the combination of (a) `Client = LegacyClient` alias in legacy.go, (b) `Tool`/`Message` re-declared in types.go in the same package, (c) `GenerateRequest`/`GenerateResponse` in legacy.go, (d) `FinishReason = legacyFinishReason` alias in types.go.

NOTE on ToolCall: existing repo files that use `llm.ToolCall` consume it as the `[]ToolCall` slice on `GenerateResponse.ToolCalls`. Adding an `ID` field is BACK-COMPAT because (a) field is `omitempty`, (b) callers read .Name and .Arguments, never construct ToolCall directly (only the LLM model returns them). Verified by grep: no `llm.ToolCall{...}` literal appears in any non-test file outside `llm/` itself.

NOTE on imports: `types.go` imports only `encoding/json`. No third-party deps. `go.mod` stays unchanged.
  </action>

  <verify>
    <automated>
      cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent && \
      test -f llm/types.go && \
      go vet ./... && \
      go build ./... && \
      grep -q '^type FinishReason = legacyFinishReason' llm/types.go && \
      grep -q '^type Vector \[\]float32' llm/types.go && \
      grep -q '^type UsageSource string' llm/types.go && \
      grep -q 'ID        string          `json:"id,omitempty"`' llm/types.go && \
      ! grep -q '^require' go.mod
    </automated>
  </verify>

  <done>
    - `llm/types.go` exists with Request, Response, Message, Tool, ToolCall (with ID), Vector, Usage, UsageSource (+ 3 constants), FinishReason alias.
    - `go build ./...` succeeds for the whole repo (existing agent paradigms compile via the alias chain Client→LegacyClient + types-in-same-package).
    - JSON tag conventions match existing repo style (snake_case keys, `,omitempty` on optional fields, `json.RawMessage` for opaque-pass-through JSON).
    - `go.mod` still has no `require` block.
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| public Go API surface (`llm/` exports) | Untrusted downstream code (sister-repo adapters, third-party adapters, customer-support service) consumes these types. The boundary is the moment a downstream caller reaches into `llm.*` symbols — what we publish here is what they bind to. |
| `LegacyClient` rename | An accidental re-export or rename that drops a previously-exported field/method silently breaks downstream callers' compile (good — visible) or worse, silently changes runtime behaviour (bad — invisible). |
| `Capabilities` struct shape | Becomes a stable JSON shape consumed by Phase 5 OTel attribute emission. Adding a non-bool field later is BC-safe; reordering or renaming existing fields is breaking. |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-00-01a-01 | Tampering | `llm/legacy.go` rename | mitigate | `type Client = LegacyClient` alias keeps every existing caller compiling. Compile-time `var _ Client = (LegacyClient)(nil)` and reverse asserts the alias is symmetric (asserted at file-scope in plan 00-01b's llm_test.go). Any future PR that breaks the alias fails CI on the `go build ./...` step before reaching review. |
| T-00-01a-02 | Tampering | `llm/types.go` ToolCall.ID addition | accept | New field is `omitempty` and only populated by the LLM-side; v0.2 callers never construct ToolCall literals (verified via grep — no `llm.ToolCall{...}` outside `llm/` itself). Adding a field is back-compat by Go semantics. Acceptable. |
| T-00-01a-03 | Information Disclosure | `Capabilities` JSON serialisation | mitigate | Capabilities is a value type with explicit `json:"..."` tags; no Go reflection-based field name leakage. The 4 keys (`tools`, `embeddings`, `structured_outputs`, `prompt_caching`) are the contract. `TestProviderInfo_JSONRoundTrip` (plan 00-01b Task 6) asserts the exact key ordering in marshalled output. |
| T-00-01a-04 | Repudiation / Architectural | `// Deprecated:` comments not honored | mitigate | Each Deprecated symbol uses the EXACT `// Deprecated:` keyword (capital D, trailing colon) recognised by `gopls`/`staticcheck`/IDE tools so warnings surface to every caller at hover time. `grep -c '^// Deprecated:' llm/legacy.go` ≥ 5 is asserted in Task 1's automated verify. Plan 00-02 separately maintains the DEPRECATIONS.md table forcing a pre-release sweep. |
</threat_model>

<verification>
- `go vet ./... && go build ./...` green at the whole-repo level — every existing caller (`simple.go`, `react.go`, etc.) compiles unchanged via the `Client = LegacyClient` alias.
- `cd examples && go vet ./... && go build ./...` green — the `examples/` Go module references `llm.Client` and continues to resolve via the alias.
- `cat go.mod | grep -c '^require '` returns 0 — stdlib-only invariant preserved.
- `grep -c '^// Deprecated:' llm/legacy.go` returns ≥ 5 — every superseded public symbol carries the exact `// Deprecated:` keyword.
- (Note: `go test ./... -count=1 -race` ratifies in plan 00-01b once `llm/llm_test.go` lands; this plan only ratifies compile-time invariants.)
</verification>

<success_criteria>
1. The 7 new/rewritten files in `llm/` exist with the bodies specified in tasks 1-3 (chatmodel.go, capabilities.go, stream.go, info.go, types.go, errors.go, legacy.go via rename of client.go).
2. The whole repo compiles: `go vet ./... && go build ./...`.
3. `go.mod` still has no `require` block — the core repo's stdlib-only invariant is intact.
4. The `Capabilities` struct ratifies D-02 verbatim (4 bool fields with snake_case JSON tags); the `StreamEvent` typed union ratifies K1 (Kind enum with 6 variants + per-tool-call Index field).
5. Every existing agent paradigm (simple.go, react.go, function_call.go, reflection.go, plan_solve.go) compiles with zero diff to its source — confirmed by `go build ./...`.
6. `// Deprecated:` godoc on LegacyClient, Client alias, GenerateRequest, GenerateResponse, StreamChunk, StreamUsage names `v0.4.0` as the removal target and links `docs/migration-v0.2-to-v0.3.md`.
</success_criteria>

<output>
After completion, create `.planning/phases/00-keystone-interfaces/00-01a-SUMMARY.md` capturing:
- The exact file list created/modified
- Final LOC counts per file
- Confirmation that `go.mod` was not modified (stdlib-only intact)
- Any deviation from the planned field names with rationale (none expected)
- Note: ScriptedLLM v2 + ChatOnlyMock + doc.go + tests + scriptedllm_test.go shim all land in plan 00-01b; this plan only ratifies the contract surface.
</output>

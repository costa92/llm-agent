---
phase: 00-keystone-interfaces
plan: 01b
type: execute
wave: 1
depends_on:
  - 01a
files_modified:
  - llm/scripted.go
  - llm/chat_only_mock.go
  - llm/doc.go
  - llm/llm_test.go
  - scriptedllm_test.go
autonomous: true
requirements:
  - CORE-07
  - CORE-09
tags:
  - llm
  - mocks
  - testing
  - capability-degradation
  - go-stdlib-only

must_haves:
  truths:
    - "ScriptedLLM type-satisfies all 4 capability interfaces (ChatModel, ToolCaller, Embedder, StructuredOutputs) at compile time"
    - "ChatOnlyMock type-satisfies ChatModel and explicitly does NOT satisfy ToolCaller, Embedder, or StructuredOutputs (asserted at runtime via reflect-style type assertion)"
    - "go test ./llm/... -count=1 -race is green; all 8 tests in llm_test.go pass"
    - "go test ./... -count=1 -race is green for the entire repo (existing agent paradigm tests continue to pass via the scriptedllm_test.go shim)"
    - "llm/doc.go contains the package overview + canonical capability-negotiation idiom (type assertion + Capabilities check)"
    - "go.mod has no require block — stdlib-only invariant intact"
  artifacts:
    - path: "llm/scripted.go"
      provides: "ScriptedLLM v2 full-capability mock with functional options"
      contains: "type ScriptedLLM struct"
    - path: "llm/chat_only_mock.go"
      provides: "ChatOnlyMock implementing ONLY ChatModel"
      contains: "type ChatOnlyMock struct"
    - path: "llm/doc.go"
      provides: "Package overview + canonical capability-negotiation idiom for adapter authors"
      contains: "package llm"
    - path: "llm/llm_test.go"
      provides: "Interface-satisfaction tests, ChatOnlyMock negative assertions, idempotent Close test, alias roundtrip test"
      contains: "TestChatOnlyMockExcludesCapabilities"
    - path: "scriptedllm_test.go"
      provides: "Thin shim — scriptedLLM is a v0.2-shape wrapper around the v0.3 surface so existing agent paradigm tests still compile"
      contains: "errScriptExhausted = llm.ErrScriptExhausted"
  key_links:
    - from: "llm/scripted.go"
      to: "llm/chatmodel.go, llm/capabilities.go (from plan 00-01a)"
      via: "compile-time interface assertions"
      pattern: "var\\s*\\(\\s*_\\s+ChatModel.*=.*\\(\\*ScriptedLLM\\)\\(nil\\)"
    - from: "llm/chat_only_mock.go"
      to: "llm/chatmodel.go (from plan 00-01a)"
      via: "compile-time ChatModel assertion ONLY (no others)"
      pattern: "var _ ChatModel = \\(\\*ChatOnlyMock\\)\\(nil\\)"
    - from: "scriptedllm_test.go (root)"
      to: "llm/errors.go (from plan 00-01a)"
      via: "errScriptExhausted = llm.ErrScriptExhausted (sentinel re-export)"
      pattern: "errScriptExhausted = llm\\.ErrScriptExhausted"
---

<objective>
Land the test ergonomics on top of plan 00-01a's contract surface: `ScriptedLLM v2` (full-capability deterministic mock per D-03), `ChatOnlyMock` (capability-degraded mock for fallback testing), the package overview in `llm/doc.go`, the new `llm/llm_test.go` exercising interface-satisfaction + ChatOnlyMock negative assertions + ScriptedLLM happy paths + sentinel `errors.Is` round-trips, and the rewritten `scriptedllm_test.go` shim that keeps every existing agent paradigm test compiling against the v0.2 `llm.Client` (via the LegacyClient alias from plan 00-01a).

Purpose: Plan 00-01a ratified the **contract** (ChatModel + ToolCaller + Embedder + StructuredOutputs + StreamEvent + ProviderInfo + Capabilities + types.go + errors.go + legacy.go). This plan ratifies the **mocks + tests + docs** that prove the contract is satisfiable AND that existing agent paradigms continue to work unchanged. After both plans land, the Phase 0 K1/K2/K3 deliverable is complete on the core-repo side.

Output: 4 new files (scripted.go, chat_only_mock.go, doc.go, llm_test.go) + 1 rewritten file (scriptedllm_test.go shim). All tests pass under `-race`. `go.mod` unchanged.
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

# Plan 00-01a deliverables (contract surface this plan builds on)
@llm/chatmodel.go
@llm/capabilities.go
@llm/info.go
@llm/stream.go
@llm/types.go
@llm/errors.go
@llm/legacy.go

# Existing test helper being rewritten as a shim
@scriptedllm_test.go
@examples/scriptedllm/scriptedllm.go
@agent_test.go

<interfaces>
<!-- Interfaces from plan 00-01a that this plan binds against. The mocks below MUST satisfy these contracts at compile time. -->

From plan 00-01a / llm/chatmodel.go:
```go
type ChatModel interface {
    Generate(ctx context.Context, req Request) (Response, error)
    Stream(ctx context.Context, req Request) (StreamReader, error)
    Info() ProviderInfo
}
```

From plan 00-01a / llm/capabilities.go:
```go
type ToolCaller interface {
    ChatModel
    WithTools(tools []Tool) (ToolCaller, error)
}
type Embedder interface {
    Embed(ctx context.Context, texts []string) (vectors []Vector, usage Usage, err error)
    EmbedDimensions() int
}
type StructuredOutputs interface {
    ChatModel
    WithSchema(schema []byte) (ChatModel, error)
}
```

From plan 00-01a / llm/errors.go:
```go
var ErrCapabilityNotSupported = errors.New("...")
var ErrScriptExhausted = errors.New("...")
```

From plan 00-01a / llm/legacy.go:
```go
type LegacyClient interface { ... }      // v0.2 contract
type Client = LegacyClient                 // alias
```

The `scriptedllm_test.go` shim must satisfy `llm.Client` (= `llm.LegacyClient`) at compile time. New test code (in `llm/llm_test.go`) exercises the v0.3 surface.
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 4: Create llm/scripted.go (ScriptedLLM v2 with full capabilities + functional options) and llm/chat_only_mock.go</name>

  <files>llm/scripted.go (NEW), llm/chat_only_mock.go (NEW)</files>

  <read_first>
    - llm/chatmodel.go, llm/capabilities.go, llm/info.go, llm/stream.go, llm/types.go, llm/errors.go (all created in plan 00-01a — Task 4 uses them)
    - scriptedllm_test.go (current — design template; D-03 promotes a v2 of this to non-test code)
    - examples/scriptedllm/scriptedllm.go (existing public-facing variant — pattern for functional helpers `Text(s)`, `ToolCall(name, args)`)
    - .planning/phases/00-keystone-interfaces/00-RESEARCH.md §"`llm/scripted.go`" lines 635-713 — full type body, options, helper functions, var-_ assertions
    - .planning/phases/00-keystone-interfaces/00-RESEARCH.md §"`llm/chat_only_mock.go`" lines 715-744 — full type body, ONE var-_ assertion
    - .planning/phases/00-keystone-interfaces/00-PATTERNS.md §"`llm/scripted.go`" lines 320-396 — sync.Mutex+cursor pattern, functional helpers Text/ToolCall, compile-time assertions in production file (not test file)
  </read_first>

  <behavior>
    - Test 1: `llm.NewScriptedLLM(llm.WithProvider("p"), llm.WithModel("m"), llm.WithResponses(llm.TextResponse("hello")))` returns a *ScriptedLLM whose first Generate call returns the "hello" Response
    - Test 2: After exhausting the script, the next Generate returns `(Response{}, ErrScriptExhausted)` — and `errors.Is(err, llm.ErrScriptExhausted)` is true
    - Test 3: Stream synthesises an EventTextDelta + EventDone sequence from the current script Response; AccumulateStream over it returns the same Text the Response carries
    - Test 4: WithTools returns a NEW *ScriptedLLM (immutability — Pattern 2), and calling Info().Capabilities.Tools on the returned ToolCaller returns true
    - Test 5: Embed returns vectors with length equal to len(texts); EmbedDimensions returns a deterministic value (e.g., 4 by default, or whatever WithEmbedDimensions sets)
    - Test 6: WithSchema returns a *ScriptedLLM (which IS a ChatModel — schema is honored as a no-op for Phase 0)
    - Test 7: Compile-time `var _ ChatModel = (*ScriptedLLM)(nil)` AND `var _ ToolCaller = (*ScriptedLLM)(nil)` AND `var _ Embedder = (*ScriptedLLM)(nil)` AND `var _ StructuredOutputs = (*ScriptedLLM)(nil)` all hold
    - Test 8: ChatOnlyMock — `var _ ChatModel = (*ChatOnlyMock)(nil)` holds; runtime `_, ok := m.(ToolCaller); ok` is FALSE; same for Embedder, StructuredOutputs
  </behavior>

  <action>
Create TWO files. The full bodies are below (faithful to RESEARCH.md §"`llm/scripted.go`" and §"`llm/chat_only_mock.go`" with stream Next/Close implementations filled in).

---

### File 1: `llm/scripted.go`

```go
package llm

import (
	"context"
	"fmt"
	"io"
	"sync"
)

// ScriptedLLM is a deterministic full-capability mock. It implements
// ChatModel + ToolCaller + Embedder + StructuredOutputs and is used
// across the umbrella as the canonical reference: agent unit tests
// (this repo), conformance baseline (sister repos, Phase 1), example
// programs.
//
// Construction is via functional options:
//
//	m := llm.NewScriptedLLM(
//	    llm.WithProvider("scripted"),
//	    llm.WithModel("test-1"),
//	    llm.WithCapabilities(llm.Capabilities{Tools: true, Embeddings: true}),
//	    llm.WithResponses(
//	        llm.TextResponse("hello"),
//	        llm.ToolCallResponse("calc", `{"a":2,"b":3}`),
//	    ),
//	)
//
// Concurrent-safe: the cursor is protected by sync.Mutex.
type ScriptedLLM struct {
	mu        sync.Mutex
	provider  string
	model     string
	caps      Capabilities
	cursor    int
	resps     []Response
	embeds    [][]Vector // per-call batch responses for Embed
	embedDim  int
	tools     []Tool // bound by WithTools (returns new ScriptedLLM)
}

// Compile-time interface satisfaction. Placed in production code (not
// only in tests) so capability claims are part of the published API
// surface visible via godoc.
var (
	_ ChatModel         = (*ScriptedLLM)(nil)
	_ ToolCaller        = (*ScriptedLLM)(nil)
	_ Embedder          = (*ScriptedLLM)(nil)
	_ StructuredOutputs = (*ScriptedLLM)(nil)
)

// NewScriptedLLM constructs a ScriptedLLM with functional options.
// Default Capabilities are ALL TRUE (full-capability default; for
// capability-degradation testing use ChatOnlyMock instead).
func NewScriptedLLM(opts ...ScriptedOption) *ScriptedLLM {
	s := &ScriptedLLM{
		provider: "scripted",
		model:    "test",
		caps:     Capabilities{Tools: true, Embeddings: true, StructuredOutputs: true, PromptCaching: false},
		embedDim: 4,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Generate returns the next scripted Response or ErrScriptExhausted.
func (s *ScriptedLLM) Generate(_ context.Context, _ Request) (Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cursor >= len(s.resps) {
		s.cursor++
		return Response{}, fmt.Errorf("scripted: %w", ErrScriptExhausted)
	}
	r := s.resps[s.cursor]
	s.cursor++
	return r, nil
}

// Stream synthesises a streaming view of the next scripted Response.
// Emits EventTextDelta (if Text != "") then EventDone with Usage and
// FinishReason populated from the Response.
func (s *ScriptedLLM) Stream(_ context.Context, _ Request) (StreamReader, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cursor >= len(s.resps) {
		s.cursor++
		return nil, fmt.Errorf("scripted: %w", ErrScriptExhausted)
	}
	r := s.resps[s.cursor]
	s.cursor++
	return newScriptedStream(r), nil
}

// Info returns the bound provider/model + Capabilities.
func (s *ScriptedLLM) Info() ProviderInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	return ProviderInfo{Provider: s.provider, Model: s.model, Capabilities: s.caps}
}

// WithTools returns a NEW *ScriptedLLM with tools bound (immutable —
// Pattern 2). The receiver is unchanged; safe to call concurrently.
func (s *ScriptedLLM) WithTools(tools []Tool) (ToolCaller, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *s
	cp.mu = sync.Mutex{}
	cp.tools = append([]Tool(nil), tools...)
	return &cp, nil
}

// Embed returns deterministic per-text vectors of EmbedDimensions
// length. If WithEmbeds was used to script per-call vectors, those are
// returned in cursor order (independent cursor from Generate).
func (s *ScriptedLLM) Embed(_ context.Context, texts []string) ([]Vector, Usage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Vector, len(texts))
	for i := range texts {
		out[i] = make(Vector, s.embedDim)
		// Deterministic content: fill with float32(i+1)/10 so vectors
		// differ between texts and tests can assert ordering.
		for j := range out[i] {
			out[i][j] = float32(i+1) / 10
		}
	}
	usage := Usage{InputTokens: len(texts), OutputTokens: 0, TotalTokens: len(texts), Source: UsageReported}
	return out, usage, nil
}

// EmbedDimensions returns the bound vector dimension. Defaults to 4
// (small for fast tests) unless overridden via WithEmbedDimensions.
func (s *ScriptedLLM) EmbedDimensions() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.embedDim
}

// WithSchema is honored as a no-op (the mock does not validate JSON
// schemas) but returns a NEW *ScriptedLLM (immutable). Returning
// ChatModel matches the StructuredOutputs interface signature.
func (s *ScriptedLLM) WithSchema(_ []byte) (ChatModel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *s
	cp.mu = sync.Mutex{}
	return &cp, nil
}

// ScriptedOption configures a ScriptedLLM at construction time.
type ScriptedOption func(*ScriptedLLM)

// WithProvider sets the Provider field returned by Info().
func WithProvider(p string) ScriptedOption { return func(s *ScriptedLLM) { s.provider = p } }

// WithModel sets the Model field returned by Info().
func WithModel(m string) ScriptedOption { return func(s *ScriptedLLM) { s.model = m } }

// WithCapabilities sets the Capabilities returned by Info().
func WithCapabilities(c Capabilities) ScriptedOption { return func(s *ScriptedLLM) { s.caps = c } }

// WithResponses appends scripted Responses; Generate/Stream consume in order.
func WithResponses(rs ...Response) ScriptedOption {
	return func(s *ScriptedLLM) { s.resps = append(s.resps, rs...) }
}

// WithEmbedDimensions overrides the EmbedDimensions return value.
func WithEmbedDimensions(d int) ScriptedOption {
	return func(s *ScriptedLLM) { s.embedDim = d }
}

// TextResponse is a convenience constructor for plain-text responses
// ending in FinishReasonStop.
func TextResponse(text string) Response {
	return Response{Text: text, FinishReason: FinishReasonStop, Provider: "scripted"}
}

// ToolCallResponse builds a tool-call response (FinishReasonToolCalls)
// for the given tool name and JSON arguments string.
func ToolCallResponse(name, argsJSON string) Response {
	return Response{
		FinishReason: FinishReasonToolCalls,
		Provider:     "scripted",
		ToolCalls:    []ToolCall{{Name: name, Arguments: []byte(argsJSON)}},
	}
}

// scriptedStream is a tiny StreamReader that emits one EventTextDelta
// (if Text != "") followed by EventDone, then io.EOF on subsequent
// Next calls. Close is idempotent.
type scriptedStream struct {
	mu     sync.Mutex
	r      Response
	step   int
	closed bool
}

func newScriptedStream(r Response) StreamReader {
	return &scriptedStream{r: r}
}

func (s *scriptedStream) Next() (StreamEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return StreamEvent{}, io.EOF
	}
	switch s.step {
	case 0:
		s.step++
		if s.r.Text != "" {
			return StreamEvent{Kind: EventTextDelta, Text: s.r.Text}, nil
		}
		// fall through to Done if no text
		fallthrough
	case 1:
		s.step = 2
		usage := s.r.Usage
		return StreamEvent{Kind: EventDone, Usage: &usage, FinishReason: s.r.FinishReason}, nil
	default:
		return StreamEvent{}, io.EOF
	}
}

func (s *scriptedStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}
```

---

### File 2: `llm/chat_only_mock.go`

```go
package llm

import "context"

// ChatOnlyMock implements ONLY ChatModel — no ToolCaller, no Embedder,
// no StructuredOutputs. Used in agent tests (Phase 3) to verify
// graceful capability degradation: ReAct falls back to scratchpad
// templating when model.(ToolCaller) fails. Phase 0 lands the type so
// downstream tests have a canonical capability-degraded mock.
type ChatOnlyMock struct {
	Provider string
	Model    string
	Resp     Response
}

// Compile-time: ChatModel ONLY — explicitly NOT ToolCaller / Embedder /
// StructuredOutputs (negative assertions in llm/llm_test.go).
var _ ChatModel = (*ChatOnlyMock)(nil)

func (m *ChatOnlyMock) Generate(_ context.Context, _ Request) (Response, error) {
	return m.Resp, nil
}

func (m *ChatOnlyMock) Stream(_ context.Context, _ Request) (StreamReader, error) {
	return newScriptedStream(m.Resp), nil
}

func (m *ChatOnlyMock) Info() ProviderInfo {
	return ProviderInfo{
		Provider:     m.Provider,
		Model:        m.Model,
		Capabilities: Capabilities{}, // ALL false — that's the point
	}
}
```

This is the minimal correct file. `chat_only_mock.go` delegates Stream to `newScriptedStream` from `scripted.go` (same package), so neither `io` nor `sync` is imported here directly.
  </action>

  <verify>
    <automated>
      cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent && \
      test -f llm/scripted.go && test -f llm/chat_only_mock.go && \
      go vet ./llm/... && \
      go build ./... && \
      grep -q '_ ChatModel\s*=\s*(\*ScriptedLLM)(nil)' llm/scripted.go && \
      grep -q '_ ToolCaller\s*=\s*(\*ScriptedLLM)(nil)' llm/scripted.go && \
      grep -q '_ Embedder\s*=\s*(\*ScriptedLLM)(nil)' llm/scripted.go && \
      grep -q '_ StructuredOutputs\s*=\s*(\*ScriptedLLM)(nil)' llm/scripted.go && \
      grep -q '_ ChatModel\s*=\s*(\*ChatOnlyMock)(nil)' llm/chat_only_mock.go && \
      ! grep -q '_ ToolCaller\s*=\s*(\*ChatOnlyMock)(nil)' llm/chat_only_mock.go && \
      ! grep -q '^require' go.mod
    </automated>
  </verify>

  <done>
    - `llm/scripted.go` compiles, satisfies all 4 capability interfaces at compile time.
    - `llm/chat_only_mock.go` compiles, satisfies ONLY ChatModel at compile time.
    - WithTools and WithSchema return NEW values (immutability — Pattern 2 verified by inspection of receiver-copy logic).
    - `Embed` returns deterministic per-text vectors of EmbedDimensions length.
    - The synthesized `scriptedStream` emits EventTextDelta then EventDone then io.EOF; Close is idempotent.
    - `go vet` passes; `go build ./...` passes; `go.mod` still has no require block.
  </done>
</task>

<task type="auto" tdd="true">
  <name>Task 5: Replace llm/doc.go with new package overview + canonical capability-negotiation idiom</name>

  <files>llm/doc.go (REPLACED)</files>

  <read_first>
    - llm/doc.go (current — 18-line existing file; entirely replaced)
    - llm/chatmodel.go, llm/capabilities.go, llm/info.go (plan 00-01a — what doc.go documents)
    - .planning/phases/00-keystone-interfaces/00-PATTERNS.md §"`llm/doc.go`" lines 408-440 — package-doc style: comment block precedes package declaration directly, bullet list 4-space indent, package as final line, no imports
    - .planning/phases/00-keystone-interfaces/00-RESEARCH.md §"Pattern 1" lines 223-240 — canonical caller idiom (type assertion + Capabilities check both)
  </read_first>

  <behavior>
    - Test 1: `go doc github.com/costa92/llm-agent/llm` displays the package overview with all new types listed and the capability-negotiation idiom shown
    - Test 2: `go vet ./llm/...` passes (no syntax errors, no unused imports — doc.go has zero imports)
    - Test 3: The file ends with `package llm` (no code follows)
    - Test 4: The file mentions ChatModel, ToolCaller, Embedder, StructuredOutputs, StreamEvent, StreamReader, ProviderInfo, Capabilities, LegacyClient (deprecation note)
  </behavior>

  <action>
Replace the entire contents of `llm/doc.go` with the body below. The file MUST end with `package llm` on the final line and have no imports or other code.

```go
// Package llm owns the capability-aware LLM-provider contract for the
// agents framework.
//
// The contract is intentionally narrow — only the types an Agent or
// Tool implementation needs to call a model:
//
//   - ChatModel          base interface (Generate + Stream + Info)
//   - ToolCaller         capability: native function-calling
//                        (WithTools is immutable; returns a new value)
//   - Embedder           capability: vector embeddings (does NOT embed
//                        ChatModel — orthogonal to chat)
//   - StructuredOutputs  capability: JSON-schema-constrained output
//   - StreamReader       iterator-style streaming (Next + Close)
//   - StreamEvent        typed union (TextDelta / ToolCall* / Done)
//   - ProviderInfo       bound provider+model identity returned by Info()
//   - Capabilities       per-(provider × model) feature struct
//                        (Tools / Embeddings / StructuredOutputs /
//                        PromptCaching as bool fields; JSON-serializable
//                        for OTel attribute emission)
//   - Tool / ToolCall    function-call schema + invocation
//   - Message            single conversation turn
//   - Request / Response chat-layer request/response (NEW in v0.3)
//   - Vector / Usage / UsageSource embeddings + token accounting
//   - FinishReason + 6 const  OpenAI-compatible stop reasons (shared
//                        between LegacyClient and ChatModel surfaces)
//   - LegacyClient       v0.2 contract retained for source compatibility;
//                        Deprecated, removal target v0.4.0
//   - ScriptedLLM        full-capability deterministic mock (NEW in v0.3)
//   - ChatOnlyMock       ChatModel-only mock (capability-degradation tests)
//
// # Capability negotiation
//
// Callers detect capabilities via type assertion AND consult
// ProviderInfo.Capabilities. The two checks together are the canonical
// idiom — type assertion is the compile-time signal, Capabilities is
// the runtime signal for per-(provider × model) variation that type
// assertion cannot see (Ollama's Go type implements ToolCaller, but
// for `llama2` Capabilities.Tools is false):
//
//	if tc, ok := model.(llm.ToolCaller); ok && model.Info().Capabilities.Tools {
//	    bound, err := tc.WithTools(tools)
//	    if err != nil { return err }
//	    return bound.Generate(ctx, req)
//	}
//	// Fall back to scratchpad templating
//	return model.Generate(ctx, scratchpadReq(req))
//
// # Streaming
//
// StreamReader is iterator-style (Next/Close) rather than channel-
// based. Consumers MUST defer sr.Close() to prevent goroutine leaks.
// AccumulateStream is a convenience for consumers that want a flat
// Response from a stream.
//
// # Deprecation
//
// LegacyClient (the v0.2 Client interface, renamed) and its companion
// types (GenerateRequest, GenerateResponse, StreamChunk, StreamUsage)
// remain callable through the v0.3.x cycle and will be removed in
// v0.4.0. New code MUST use ChatModel and the new Request/Response/
// StreamReader/StreamEvent types. See docs/migration-v0.2-to-v0.3.md.
package llm
```

This file has zero imports and ends with `package llm` as the final line. It is purely documentation.
  </action>

  <verify>
    <automated>
      cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent && \
      tail -1 llm/doc.go | grep -q '^package llm$' && \
      go vet ./llm/... && \
      grep -q 'ChatModel' llm/doc.go && grep -q 'ToolCaller' llm/doc.go && \
      grep -q 'Embedder' llm/doc.go && grep -q 'StructuredOutputs' llm/doc.go && \
      grep -q 'StreamReader' llm/doc.go && grep -q 'StreamEvent' llm/doc.go && \
      grep -q 'ProviderInfo' llm/doc.go && grep -q 'Capabilities' llm/doc.go && \
      grep -q 'LegacyClient' llm/doc.go && grep -q 'capability negotiation' llm/doc.go && \
      ! grep -q '^import' llm/doc.go
    </automated>
  </verify>

  <done>
    - `llm/doc.go` is a documentation-only file (package declaration as last line, no imports, no code).
    - All new public types are mentioned in the bulleted list.
    - The canonical capability-negotiation idiom (type assertion + Capabilities check) is shown verbatim per RESEARCH.md Pattern 1.
    - `go vet ./llm/...` passes; `go doc github.com/costa92/llm-agent/llm` renders the new content.
  </done>
</task>

<task type="auto" tdd="true">
  <name>Task 6: Create llm/llm_test.go (interface satisfaction tests, ChatOnlyMock negative assertions, StreamReader idempotent close, alias roundtrip, sentinel errors.Is, ScriptedLLM happy paths)</name>

  <files>llm/llm_test.go (NEW)</files>

  <read_first>
    - llm/scripted.go, llm/chat_only_mock.go (Task 4), llm/legacy.go, llm/types.go, llm/errors.go (plan 00-01a)
    - agent_test.go (current — analog test pattern: compile-time `var _` block, internal test package, table-test loop with `errors.Is`)
    - .planning/phases/00-keystone-interfaces/00-RESEARCH.md §"`llm/llm_test.go`" lines 746-786 — every test name + behaviour
    - .planning/phases/00-keystone-interfaces/00-PATTERNS.md §"`llm/llm_test.go`" lines 442-475 — internal-test-package convention, naming style TestSubject_Behavior
  </read_first>

  <behavior>
    - Test 1: TestChatOnlyMockExcludesCapabilities — runtime assertions that ChatOnlyMock does NOT satisfy ToolCaller, Embedder, StructuredOutputs
    - Test 2: TestScriptedLLM_Capabilities — happy-path Generate, Stream→AccumulateStream, WithTools immutability, Embed dimensions, WithSchema returns ChatModel
    - Test 3: TestStreamReaderClosesIdempotent — sr.Close() called twice does not panic and returns nil
    - Test 4: TestLegacyClientAlias — `var _ Client = (LegacyClient)(nil)` and reverse compile (this is a compile-time test; the `_test.go` file existing is sufficient)
    - Test 5: TestSentinelErrors_ErrorsIs — table test asserting `errors.Is(fmt.Errorf("...: %w", sentinel), sentinel)` for ErrCapabilityNotSupported and ErrScriptExhausted
    - Test 6: TestStreamEventKind_Variants — assert const ordering: EventTextDelta == 0, EventDone == 5
    - Test 7: TestProviderInfo_JSONRoundTrip — marshal+unmarshal preserves all fields including all 4 Capabilities bool fields with snake_case keys
    - Test 8: TestToolCallerImmutable — `tc.WithTools(toolsA)` and `tc.WithTools(toolsB)` produce two values that do not share state; concurrent invocation does not race (-race flag)
  </behavior>

  <action>
Create `llm/llm_test.go` with all 8 tests. Use the internal test package (`package llm`, same as production) per the existing repo convention.

```go
package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
)

// ----- Compile-time alias roundtrip tests (TestLegacyClientAlias) -----
// These assertions live at file scope so they are checked at every build.
var (
	_ Client       = (LegacyClient)(nil)
	_ LegacyClient = (Client)(nil)
)

func TestLegacyClientAlias(t *testing.T) {
	t.Helper()
	// File-scope assertions above are the actual proof. This test
	// exists so the symbol shows up in `go test -v` output.
	t.Log("Client and LegacyClient are aliases — compile-time satisfied")
}

// ----- ChatOnlyMock negative capability assertions -----
func TestChatOnlyMockExcludesCapabilities(t *testing.T) {
	var m ChatModel = &ChatOnlyMock{Provider: "test", Model: "m"}
	if _, ok := m.(ToolCaller); ok {
		t.Fatal("ChatOnlyMock must not implement ToolCaller")
	}
	if _, ok := m.(Embedder); ok {
		t.Fatal("ChatOnlyMock must not implement Embedder")
	}
	if _, ok := m.(StructuredOutputs); ok {
		t.Fatal("ChatOnlyMock must not implement StructuredOutputs")
	}
	info := m.Info()
	if info.Capabilities.Tools || info.Capabilities.Embeddings ||
		info.Capabilities.StructuredOutputs || info.Capabilities.PromptCaching {
		t.Errorf("ChatOnlyMock.Info().Capabilities = %+v, want all-false", info.Capabilities)
	}
}

// ----- ScriptedLLM happy paths -----
func TestScriptedLLM_Capabilities(t *testing.T) {
	ctx := context.Background()
	m := NewScriptedLLM(
		WithProvider("scripted"),
		WithModel("test-1"),
		WithResponses(TextResponse("hello"), TextResponse("world")),
	)

	// Generate path
	r, err := m.Generate(ctx, Request{})
	if err != nil {
		t.Fatalf("Generate#1: %v", err)
	}
	if r.Text != "hello" {
		t.Errorf("Generate#1 Text = %q, want %q", r.Text, "hello")
	}

	// Stream path
	sr, err := m.Stream(ctx, Request{})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	t.Cleanup(func() { _ = sr.Close() })
	resp, err := AccumulateStream(sr)
	if err != nil {
		t.Fatalf("AccumulateStream: %v", err)
	}
	if resp.Text != "world" {
		t.Errorf("AccumulateStream Text = %q, want %q", resp.Text, "world")
	}

	// Exhaustion
	_, err = m.Generate(ctx, Request{})
	if !errors.Is(err, ErrScriptExhausted) {
		t.Errorf("expected ErrScriptExhausted, got %v", err)
	}

	// Embed
	em := NewScriptedLLM(WithEmbedDimensions(8))
	vecs, usage, err := em.Embed(ctx, []string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vecs) != 3 {
		t.Errorf("Embed len = %d, want 3", len(vecs))
	}
	if em.EmbedDimensions() != 8 {
		t.Errorf("EmbedDimensions = %d, want 8", em.EmbedDimensions())
	}
	if len(vecs[0]) != 8 {
		t.Errorf("Embed[0] dim = %d, want 8", len(vecs[0]))
	}
	if usage.Source != UsageReported {
		t.Errorf("Embed Usage.Source = %q, want %q", usage.Source, UsageReported)
	}

	// WithSchema (returns ChatModel — schema-bound)
	sm := NewScriptedLLM(WithResponses(TextResponse("schema")))
	bound, err := sm.WithSchema([]byte(`{"type":"object"}`))
	if err != nil {
		t.Fatalf("WithSchema: %v", err)
	}
	if _, ok := bound.(ChatModel); !ok {
		t.Fatal("WithSchema must return a ChatModel")
	}
}

// ----- ToolCaller immutability + concurrency -----
func TestToolCallerImmutable(t *testing.T) {
	ctx := context.Background()
	base := NewScriptedLLM(WithResponses(TextResponse("x"), TextResponse("y")))

	toolsA := []Tool{{Name: "a", Description: "A", Parameters: json.RawMessage(`{}`)}}
	toolsB := []Tool{{Name: "b", Description: "B", Parameters: json.RawMessage(`{}`)}}

	a, err := base.WithTools(toolsA)
	if err != nil {
		t.Fatalf("WithTools(A): %v", err)
	}
	b, err := base.WithTools(toolsB)
	if err != nil {
		t.Fatalf("WithTools(B): %v", err)
	}
	if a == b {
		t.Fatal("WithTools must return distinct values")
	}

	// Concurrent Generate calls must not race (-race flag asserts this).
	var wg sync.WaitGroup
	wg.Add(2)
	errCh := make(chan error, 2)
	go func() {
		defer wg.Done()
		_, err := a.Generate(ctx, Request{})
		errCh <- err
	}()
	go func() {
		defer wg.Done()
		_, err := b.Generate(ctx, Request{})
		errCh <- err
	}()
	wg.Wait()
	close(errCh)
	for e := range errCh {
		if e != nil {
			t.Errorf("concurrent Generate err: %v", e)
		}
	}
}

// ----- StreamReader idempotent close -----
func TestStreamReaderClosesIdempotent(t *testing.T) {
	ctx := context.Background()
	m := NewScriptedLLM(WithResponses(TextResponse("hi")))
	sr, err := m.Stream(ctx, Request{})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if err := sr.Close(); err != nil {
		t.Errorf("Close#1: %v", err)
	}
	// MUST NOT panic on second Close
	if err := sr.Close(); err != nil {
		t.Errorf("Close#2: %v", err)
	}
	// After close, Next returns io.EOF
	_, err = sr.Next()
	if !errors.Is(err, io.EOF) {
		t.Errorf("Next after Close = %v, want io.EOF", err)
	}
}

// ----- Sentinel errors.Is round-trip -----
func TestSentinelErrors_ErrorsIs(t *testing.T) {
	cases := []struct {
		name string
		s    error
	}{
		{"ErrCapabilityNotSupported", ErrCapabilityNotSupported},
		{"ErrScriptExhausted", ErrScriptExhausted},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wrapped := fmt.Errorf("wrap: %w", c.s)
			if !errors.Is(wrapped, c.s) {
				t.Errorf("errors.Is(wrapped, %s) = false, want true", c.name)
			}
		})
	}
}

// ----- StreamEventKind variant ordering -----
func TestStreamEventKind_Variants(t *testing.T) {
	cases := []struct {
		k    StreamEventKind
		want uint8
	}{
		{EventTextDelta, 0},
		{EventToolCallStart, 1},
		{EventToolCallArgsDelta, 2},
		{EventToolCallEnd, 3},
		{EventThinkingDelta, 4},
		{EventDone, 5},
	}
	for _, c := range cases {
		if uint8(c.k) != c.want {
			t.Errorf("kind = %d, want %d", c.k, c.want)
		}
	}
}

// ----- ProviderInfo JSON round-trip (Capabilities serialisation) -----
func TestProviderInfo_JSONRoundTrip(t *testing.T) {
	in := ProviderInfo{
		Provider: "openai",
		Model:    "gpt-4o-mini",
		Capabilities: Capabilities{
			Tools: true, Embeddings: true, StructuredOutputs: false, PromptCaching: false,
		},
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := `{"provider":"openai","model":"gpt-4o-mini","capabilities":{"tools":true,"embeddings":true,"structured_outputs":false,"prompt_caching":false}}`
	if string(b) != want {
		t.Errorf("Marshal:\n got  %s\n want %s", b, want)
	}

	var out ProviderInfo
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if out != in {
		t.Errorf("round-trip:\n got  %+v\n want %+v", out, in)
	}
}
```

After writing the file, run `go test ./llm/... -count=1 -race` and ensure all 8 tests pass with the race detector enabled.
  </action>

  <verify>
    <automated>
      cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent && \
      test -f llm/llm_test.go && \
      go vet ./llm/... && \
      go test ./llm/... -count=1 -race && \
      grep -q 'TestChatOnlyMockExcludesCapabilities' llm/llm_test.go && \
      grep -q 'TestScriptedLLM_Capabilities' llm/llm_test.go && \
      grep -q 'TestStreamReaderClosesIdempotent' llm/llm_test.go && \
      grep -q 'TestLegacyClientAlias' llm/llm_test.go && \
      grep -q 'TestSentinelErrors_ErrorsIs' llm/llm_test.go && \
      grep -q 'TestStreamEventKind_Variants' llm/llm_test.go && \
      grep -q 'TestProviderInfo_JSONRoundTrip' llm/llm_test.go && \
      grep -q 'TestToolCallerImmutable' llm/llm_test.go
    </automated>
  </verify>

  <done>
    - All 8 tests pass under `-race`.
    - File-scope `var _ Client = (LegacyClient)(nil)` and `var _ LegacyClient = (Client)(nil)` enforce alias roundtrip at every build.
    - ChatOnlyMock negative-assertion test is the planner-mandated guard rail (prevents a future refactor from silently making ChatOnlyMock a ToolCaller).
    - ProviderInfo JSON round-trip asserts the exact snake_case key ordering for OTel emission compatibility.
  </done>
</task>

<task type="auto" tdd="true">
  <name>Task 7: Convert scriptedllm_test.go to a thin shim that delegates to the v0.2 contract (preserves existing agent paradigm tests)</name>

  <files>scriptedllm_test.go (REWRITTEN — root of repo, package agents)</files>

  <read_first>
    - scriptedllm_test.go (current — entire file is the source of truth for the helpers existing tests use)
    - simple_test.go, react_test.go, function_call_test.go, reflection_test.go, plan_solve_test.go, example_simple_test.go, example_tool_use_test.go, example_multi_agent_test.go (ALL existing tests that use scriptedLLM/newScriptedLLM/textResp — these continue to compile after Task 7)
    - llm/scripted.go (Task 4 — new ScriptedLLM v2 with options)
    - llm/legacy.go (plan 00-01a — GenerateResponse remains the legacy response shape)
    - .planning/phases/00-keystone-interfaces/00-RESEARCH.md §"Test-helper migration" lines 1219-1253 — exact shim shape
  </read_first>

  <behavior>
    - Test 1: `go test ./... -count=1 -race` is green for the WHOLE repo (every existing agent paradigm test continues to pass)
    - Test 2: `scriptedLLM` is callable from existing tests as `newScriptedLLM(textResp("hello"))` and supports `Generate(ctx, llm.GenerateRequest{...})` returning a `llm.GenerateResponse` (legacy shape) — i.e., the shim implements `llm.LegacyClient`, NOT the new `llm.ChatModel`. Phase 3 will migrate agent paradigms to ChatModel.
    - Test 3: `errScriptExhausted = llm.ErrScriptExhausted` so existing test code matching with `errors.Is(err, errScriptExhausted)` keeps working
    - Test 4: `callCount()` method preserved
    - Test 5: `textResp(s)` returns a `llm.GenerateResponse` (legacy type)
  </behavior>

  <action>
The current `scriptedllm_test.go` (in the REPO ROOT, package `agents`) is a TEST helper that implements the v0.2 `llm.Client` interface. The new `llm.ScriptedLLM` (in `llm/scripted.go`) implements the v0.3 `llm.ChatModel` interface. These are DIFFERENT contracts (v0.2 uses `GenerateRequest`/`GenerateResponse`, v0.3 uses `Request`/`Response`).

The shim must KEEP the old contract working — agent paradigms still call the v0.2 `llm.Client` (via the `llm.Client = LegacyClient` alias) and Phase 0 doesn't migrate them. So the shim is a small wrapper around the v0.2 contract that re-exports the new sentinel error.

Replace the entire body of `scriptedllm_test.go` with:

```go
package agents

import (
	"context"
	"fmt"
	"sync"

	"github.com/costa92/llm-agent/llm"
)

// scriptedLLM is a test helper that returns pre-set GenerateResponse
// values in order on each Generate call. After the script is exhausted
// it returns errScriptExhausted. Concurrent-safe via mu.
//
// As of v0.3 (Phase 0), this type is a thin shim that re-uses the
// canonical sentinel error from the new llm package. It still satisfies
// the v0.2 llm.Client (= LegacyClient) interface so existing agent
// paradigm tests in this package continue to compile unchanged. Phase 3
// will migrate agent paradigms to llm.ChatModel and this shim will go
// away.
//
// Deprecated: New tests should use llm.NewScriptedLLM directly. Retained
// until Phase 3 refactors agent paradigms.
type scriptedLLM struct {
	mu    sync.Mutex
	calls int
	resps []llm.GenerateResponse
}

// errScriptExhausted aliases llm.ErrScriptExhausted so existing tests
// matching with errors.Is(err, errScriptExhausted) continue to work.
var errScriptExhausted = llm.ErrScriptExhausted

func newScriptedLLM(resps ...llm.GenerateResponse) *scriptedLLM {
	return &scriptedLLM{resps: resps}
}

// Generate returns the next scripted GenerateResponse or wraps
// errScriptExhausted when the script is exhausted.
func (s *scriptedLLM) Generate(_ context.Context, _ llm.GenerateRequest) (llm.GenerateResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.calls >= len(s.resps) {
		s.calls++
		return llm.GenerateResponse{}, fmt.Errorf("scriptedLLM: %w", errScriptExhausted)
	}
	r := s.resps[s.calls]
	s.calls++
	return r, nil
}

// GenerateStream returns an error — streaming was not supported by the
// v0.2 helper either. Phase 2 streaming work uses llm.ScriptedLLM
// directly via the v0.3 surface.
func (s *scriptedLLM) GenerateStream(_ context.Context, _ llm.GenerateRequest) (<-chan llm.StreamChunk, error) {
	return nil, fmt.Errorf("scriptedLLM: streaming not supported (use llm.ScriptedLLM via v0.3 surface)")
}

// callCount returns how many times Generate was invoked. Preserved for
// backwards-compat with existing tests that assert on call count.
func (s *scriptedLLM) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

// textResp builds a single text-only GenerateResponse (v0.2 shape).
func textResp(text string) llm.GenerateResponse {
	return llm.GenerateResponse{
		Text:         text,
		FinishReason: llm.FinishReasonStop,
		Provider:     "scripted",
	}
}

// Compile-time: scriptedLLM satisfies the v0.2 LegacyClient (= Client) contract.
var _ llm.Client = (*scriptedLLM)(nil)
```

NOTE: We chose to KEEP the test-helper as a stand-alone wrapper (not a type alias to `*llm.ScriptedLLM`) because the v0.2 / v0.3 method signatures differ (v0.2 returns `<-chan StreamChunk`, v0.3 returns `StreamReader`). A wrapper type with bespoke method bodies is the cleanest way to preserve back-compat without changing every existing agent test file. The shim is < 80 LOC and has no behaviour change vs. the current helper.

After this edit:
- `go test ./... -count=1 -race` MUST pass (every existing agent paradigm test stays green).
- Phase 3 (CORE-10) will replace this shim with direct use of `llm.NewScriptedLLM`.
  </action>

  <verify>
    <automated>
      cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent && \
      go vet ./... && \
      go build ./... && \
      go test ./... -count=1 -race && \
      grep -q 'errScriptExhausted = llm.ErrScriptExhausted' scriptedllm_test.go && \
      grep -q 'var _ llm.Client = (\*scriptedLLM)(nil)' scriptedllm_test.go && \
      grep -q 'github.com/costa92/llm-agent/llm' scriptedllm_test.go && \
      ! grep -q '^require' go.mod
    </automated>
  </verify>

  <done>
    - `scriptedllm_test.go` is rewritten as a thin v0.2 shim that uses `llm.ErrScriptExhausted` from the new package and otherwise preserves the existing helper API (`newScriptedLLM`, `textResp`, `callCount`).
    - `go test ./... -count=1 -race` passes for the entire repo (Phase 0 has zero behavioural diff to agent paradigm tests).
    - `var _ llm.Client = (*scriptedLLM)(nil)` ensures the shim satisfies the legacy contract at compile time.
    - `go.mod` still has no `require` block — stdlib-only invariant intact.
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| `ScriptedLLM` published as production code | Promoted from `_test.go` to package-level so sister-repo conformance suites (Phase 1) can import it. The boundary is the moment a downstream caller `import "github.com/costa92/llm-agent/llm"` reaches into ScriptedLLM — what we publish here is what they bind to. |
| `ChatOnlyMock` capability shape | The negative-capability assertion (does NOT implement ToolCaller / Embedder / StructuredOutputs) is what makes capability-degradation tests meaningful. A future refactor that silently makes ChatOnlyMock a ToolCaller would invalidate every Phase 3 fallback test. |
| `scriptedllm_test.go` shim | A v0.2-shape wrapper kept callable so agent paradigm tests (simple_test.go, react_test.go, …) compile unchanged. The shim's correctness is what gates Phase 0 from breaking the existing test suite. |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-00-01b-01 | Tampering | Future refactor silently adds ToolCaller/Embedder/StructuredOutputs to ChatOnlyMock | mitigate | `TestChatOnlyMockExcludesCapabilities` (Task 6) is a runtime guard — ANY method addition that makes ChatOnlyMock satisfy a capability interface fails this test. The test is the planner-mandated guardrail per CONTEXT.md / D-03. |
| T-00-01b-02 | Denial of Service | `ScriptedLLM` mutex held across long calls | accept | The mock holds `sync.Mutex` only for cursor advancement (no I/O under lock); concurrent Generate calls serialise but each is fast. `TestToolCallerImmutable` runs concurrent Generate under `-race` and passes. Acceptable for a deterministic test mock. |
| T-00-01b-03 | Tampering | `scriptedStream.Close()` non-idempotent | mitigate | `TestStreamReaderClosesIdempotent` asserts second `Close()` returns nil and that subsequent `Next()` returns `io.EOF`. The internal `closed` bool guards against double-state-mutation. |
| T-00-01b-04 | Repudiation | `scriptedllm_test.go` shim drops a v0.2 method or changes its signature | mitigate | Compile-time `var _ llm.Client = (*scriptedLLM)(nil)` in the shim asserts the v0.2 contract is satisfied. Any signature drift fails the build immediately. |
| T-00-01b-05 | Information Disclosure | `llm/doc.go` leaks internal API names | accept | doc.go documents only EXPORTED symbols — same surface as `go doc ./llm/`. No internal naming is leaked. Public-by-design. |
</threat_model>

<verification>
- `go vet ./... && go build ./...` green at the whole-repo level (uses plan 00-01a's contract surface).
- `go test ./... -count=1 -race` green — all 8 new `llm/llm_test.go` tests pass under the race detector AND every existing agent paradigm test continues to pass.
- `cd examples && go vet ./... && go build ./...` green — the `examples/` Go module references `llm.Client` and continues to resolve via the alias.
- `cat go.mod | grep -c '^require '` returns 0 — stdlib-only invariant preserved.
- `go doc github.com/costa92/llm-agent/llm` displays the new package overview with all 12+ exported types and the canonical capability-negotiation idiom.
</verification>

<success_criteria>
1. The 4 new files (scripted.go, chat_only_mock.go, doc.go, llm_test.go) plus the rewritten scriptedllm_test.go shim exist with the bodies specified in tasks 4-7.
2. The whole repo compiles + tests green: `go vet ./... && go build ./... && go test ./... -count=1 -race`.
3. `go.mod` still has no `require` block — the core repo's stdlib-only invariant is intact.
4. ScriptedLLM type-satisfies all 4 capability interfaces at compile time; ChatOnlyMock type-satisfies only ChatModel and is asserted to NOT implement the others at runtime.
5. Every existing agent paradigm test passes with zero diff to its source — confirmed by `go test ./... -count=1 -race` over the unchanged `simple_test.go`, `react_test.go`, `function_call_test.go`, `reflection_test.go`, `plan_solve_test.go`, `example_*_test.go` files.
6. `llm/doc.go` shows the canonical capability-negotiation idiom (type assertion + Capabilities runtime check).
</success_criteria>

<output>
After completion, create `.planning/phases/00-keystone-interfaces/00-01b-SUMMARY.md` capturing:
- The exact file list created/modified
- Final LOC counts per file
- Confirmation that `go.mod` was not modified (stdlib-only intact)
- Any deviation from the planned field names with rationale (none expected)
- The `go doc ./llm/...` output as a baseline snapshot pointer for Pitfall 22 (saved to `docs/api-snapshot.txt` by plan 00-05's final task at phase exit, not within this plan)
</output>

package policy

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/costa92/llm-agent/llm"
)

// --- test doubles ---------------------------------------------------

// countingChatModel records Generate/Stream invocations and the last
// Request it received. Forwards to an inner llm.ChatModel.
type countingChatModel struct {
	inner         llm.ChatModel
	generateCalls int64 // atomic
	streamCalls   int64 // atomic
	mu            sync.Mutex
	lastReq       llm.Request
}

func (c *countingChatModel) Generate(ctx context.Context, req llm.Request) (llm.Response, error) {
	atomic.AddInt64(&c.generateCalls, 1)
	c.mu.Lock()
	c.lastReq = req
	c.mu.Unlock()
	return c.inner.Generate(ctx, req)
}

func (c *countingChatModel) Stream(ctx context.Context, req llm.Request) (llm.StreamReader, error) {
	atomic.AddInt64(&c.streamCalls, 1)
	c.mu.Lock()
	c.lastReq = req
	c.mu.Unlock()
	return c.inner.Stream(ctx, req)
}

func (c *countingChatModel) Info() llm.ProviderInfo { return c.inner.Info() }

func (c *countingChatModel) Generated() int64 { return atomic.LoadInt64(&c.generateCalls) }
func (c *countingChatModel) Streamed() int64  { return atomic.LoadInt64(&c.streamCalls) }
func (c *countingChatModel) LastReq() llm.Request {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastReq
}

var _ llm.ChatModel = (*countingChatModel)(nil)

// testGate returns the configured Decision for each event kind. Allow
// (zero-value) is the default for kinds that don't have a configured
// decision. invocations counts every Inspect call across kinds.
type testGate struct {
	name        string
	pre         Decision
	post        Decision
	preStream   Decision
	streamDelta Decision
	postStream  Decision
	invocations int64 // atomic
}

func (g *testGate) Inspect(_ context.Context, ev Event) Decision {
	atomic.AddInt64(&g.invocations, 1)
	switch ev.Kind {
	case PreGenerate:
		return g.pre
	case PostGenerate:
		return g.post
	case PreStream:
		return g.preStream
	case StreamDelta:
		return g.streamDelta
	case PostStream:
		return g.postStream
	}
	return Decision{}
}

func (g *testGate) Name() string {
	if g.name == "" {
		return "testGate"
	}
	return g.name
}

func (g *testGate) Invocations() int64 { return atomic.LoadInt64(&g.invocations) }

// scriptedStreamChat is an in-test ChatModel whose Stream returns a
// scriptedEventStream emitting the configured sequence of events.
// Used for streaming tests that need precise control over the inner
// stream's events (the canonical ScriptedLLM is response-driven, not
// event-driven).
type scriptedStreamChat struct {
	provider string
	model    string
	events   []llm.StreamEvent // emitted in order; followed by io.EOF
	emitEOF  bool              // if true, terminate with io.EOF instead of EventDone in last position
	caps     llm.Capabilities

	// inner Next() counter (records calls reaching the inner stream)
	streamNextCalls int64 // atomic
}

func (s *scriptedStreamChat) Generate(_ context.Context, _ llm.Request) (llm.Response, error) {
	return llm.Response{Provider: s.provider, Model: s.model}, nil
}

func (s *scriptedStreamChat) Stream(_ context.Context, _ llm.Request) (llm.StreamReader, error) {
	return &scriptedEventStream{
		events:  append([]llm.StreamEvent(nil), s.events...),
		emitEOF: s.emitEOF,
		owner:   s,
	}, nil
}

func (s *scriptedStreamChat) Info() llm.ProviderInfo {
	return llm.ProviderInfo{Provider: s.provider, Model: s.model, Capabilities: s.caps}
}

type scriptedEventStream struct {
	mu      sync.Mutex
	events  []llm.StreamEvent
	emitEOF bool
	idx     int
	closed  bool
	owner   *scriptedStreamChat
}

func (s *scriptedEventStream) Next() (llm.StreamEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.owner != nil {
		atomic.AddInt64(&s.owner.streamNextCalls, 1)
	}
	if s.closed {
		return llm.StreamEvent{}, io.EOF
	}
	if s.idx >= len(s.events) {
		if s.emitEOF {
			return llm.StreamEvent{}, io.EOF
		}
		return llm.StreamEvent{}, io.EOF
	}
	ev := s.events[s.idx]
	s.idx++
	return ev, nil
}

func (s *scriptedEventStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

// --- TestWrap_PreservesCapabilities ----------------------------------
//
// Table-driven over the 2³ capability combinations: build a
// ScriptedLLM with the matching Capabilities, wrap, assert the
// type-assertions on the wrapped value match the expected combo.

func TestWrap_PreservesCapabilities(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name              string
		tools             bool
		embeds            bool
		structuredOutputs bool
	}{
		{"none", false, false, false},
		{"tools", true, false, false},
		{"embeds", false, true, false},
		{"schema", false, false, true},
		{"tools+embeds", true, true, false},
		{"tools+schema", true, false, true},
		{"embeds+schema", false, true, true},
		{"tools+embeds+schema", true, true, true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var inner llm.ChatModel
			full := llm.NewScriptedLLM(
				llm.WithProvider("scripted"),
				llm.WithModel("test"),
				llm.WithCapabilities(llm.Capabilities{Tools: tc.tools, Embeddings: tc.embeds, StructuredOutputs: tc.structuredOutputs}),
				llm.WithResponses(llm.TextResponse("ok")),
			)
			// Project the ScriptedLLM down to the requested capability
			// set by hiding the unwanted interfaces behind a narrow
			// type. Use a per-combination concrete type so the
			// type-switch in WrapConfig picks the right wrapper.
			inner = projectCaps(full, tc.tools, tc.embeds, tc.structuredOutputs)

			wrapped := Wrap(inner)

			_, isTool := wrapped.(llm.ToolCaller)
			_, isEmb := wrapped.(llm.Embedder)
			_, isSO := wrapped.(llm.StructuredOutputs)

			if isTool != tc.tools {
				t.Fatalf("ToolCaller: got %v, want %v", isTool, tc.tools)
			}
			if isEmb != tc.embeds {
				t.Fatalf("Embedder: got %v, want %v", isEmb, tc.embeds)
			}
			if isSO != tc.structuredOutputs {
				t.Fatalf("StructuredOutputs: got %v, want %v", isSO, tc.structuredOutputs)
			}
		})
	}
}

// projectCaps wraps a fully-capable inner LLM in one of 8 narrow types
// that exposes only the requested subset of capability interfaces.
// Needed because *llm.ScriptedLLM always implements all 3 optional
// interfaces — the type-switch in WrapConfig can't pick a narrower
// wrapper otherwise.
func projectCaps(full *llm.ScriptedLLM, tools, embeds, so bool) llm.ChatModel {
	switch {
	case tools && embeds && so:
		return capProjector{inner: full, fullTool: full, fullEmb: full, fullSO: full, tools: true, embeds: true, so: true}
	case tools && embeds:
		return capProjectorTE{inner: full, fullTool: full, fullEmb: full}
	case tools && so:
		return capProjectorTS{inner: full, fullTool: full, fullSO: full}
	case embeds && so:
		return capProjectorES{inner: full, fullEmb: full, fullSO: full}
	case tools:
		return capProjectorT{inner: full, fullTool: full}
	case embeds:
		return capProjectorE{inner: full, fullEmb: full}
	case so:
		return capProjectorS{inner: full, fullSO: full}
	default:
		return capProjectorNone{inner: full}
	}
}

// 8 narrow capability projector types. Each exposes ChatModel plus
// only the requested optional interfaces. Used by
// TestWrap_PreservesCapabilities.

type capProjectorNone struct{ inner llm.ChatModel }

func (p capProjectorNone) Generate(ctx context.Context, req llm.Request) (llm.Response, error) {
	return p.inner.Generate(ctx, req)
}
func (p capProjectorNone) Stream(ctx context.Context, req llm.Request) (llm.StreamReader, error) {
	return p.inner.Stream(ctx, req)
}
func (p capProjectorNone) Info() llm.ProviderInfo { return p.inner.Info() }

type capProjectorT struct {
	inner    llm.ChatModel
	fullTool llm.ToolCaller
}

func (p capProjectorT) Generate(ctx context.Context, req llm.Request) (llm.Response, error) {
	return p.inner.Generate(ctx, req)
}
func (p capProjectorT) Stream(ctx context.Context, req llm.Request) (llm.StreamReader, error) {
	return p.inner.Stream(ctx, req)
}
func (p capProjectorT) Info() llm.ProviderInfo { return p.inner.Info() }
func (p capProjectorT) WithTools(tools []llm.Tool) (llm.ToolCaller, error) {
	return p.fullTool.WithTools(tools)
}

type capProjectorE struct {
	inner   llm.ChatModel
	fullEmb llm.Embedder
}

func (p capProjectorE) Generate(ctx context.Context, req llm.Request) (llm.Response, error) {
	return p.inner.Generate(ctx, req)
}
func (p capProjectorE) Stream(ctx context.Context, req llm.Request) (llm.StreamReader, error) {
	return p.inner.Stream(ctx, req)
}
func (p capProjectorE) Info() llm.ProviderInfo { return p.inner.Info() }
func (p capProjectorE) Embed(ctx context.Context, texts []string) ([]llm.Vector, llm.Usage, error) {
	return p.fullEmb.Embed(ctx, texts)
}
func (p capProjectorE) EmbedDimensions() int { return p.fullEmb.EmbedDimensions() }

type capProjectorS struct {
	inner  llm.ChatModel
	fullSO llm.StructuredOutputs
}

func (p capProjectorS) Generate(ctx context.Context, req llm.Request) (llm.Response, error) {
	return p.inner.Generate(ctx, req)
}
func (p capProjectorS) Stream(ctx context.Context, req llm.Request) (llm.StreamReader, error) {
	return p.inner.Stream(ctx, req)
}
func (p capProjectorS) Info() llm.ProviderInfo { return p.inner.Info() }
func (p capProjectorS) WithSchema(schema []byte) (llm.ChatModel, error) {
	return p.fullSO.WithSchema(schema)
}

type capProjectorTE struct {
	inner    llm.ChatModel
	fullTool llm.ToolCaller
	fullEmb  llm.Embedder
}

func (p capProjectorTE) Generate(ctx context.Context, req llm.Request) (llm.Response, error) {
	return p.inner.Generate(ctx, req)
}
func (p capProjectorTE) Stream(ctx context.Context, req llm.Request) (llm.StreamReader, error) {
	return p.inner.Stream(ctx, req)
}
func (p capProjectorTE) Info() llm.ProviderInfo { return p.inner.Info() }
func (p capProjectorTE) WithTools(tools []llm.Tool) (llm.ToolCaller, error) {
	return p.fullTool.WithTools(tools)
}
func (p capProjectorTE) Embed(ctx context.Context, texts []string) ([]llm.Vector, llm.Usage, error) {
	return p.fullEmb.Embed(ctx, texts)
}
func (p capProjectorTE) EmbedDimensions() int { return p.fullEmb.EmbedDimensions() }

type capProjectorTS struct {
	inner    llm.ChatModel
	fullTool llm.ToolCaller
	fullSO   llm.StructuredOutputs
}

func (p capProjectorTS) Generate(ctx context.Context, req llm.Request) (llm.Response, error) {
	return p.inner.Generate(ctx, req)
}
func (p capProjectorTS) Stream(ctx context.Context, req llm.Request) (llm.StreamReader, error) {
	return p.inner.Stream(ctx, req)
}
func (p capProjectorTS) Info() llm.ProviderInfo { return p.inner.Info() }
func (p capProjectorTS) WithTools(tools []llm.Tool) (llm.ToolCaller, error) {
	return p.fullTool.WithTools(tools)
}
func (p capProjectorTS) WithSchema(schema []byte) (llm.ChatModel, error) {
	return p.fullSO.WithSchema(schema)
}

type capProjectorES struct {
	inner   llm.ChatModel
	fullEmb llm.Embedder
	fullSO  llm.StructuredOutputs
}

func (p capProjectorES) Generate(ctx context.Context, req llm.Request) (llm.Response, error) {
	return p.inner.Generate(ctx, req)
}
func (p capProjectorES) Stream(ctx context.Context, req llm.Request) (llm.StreamReader, error) {
	return p.inner.Stream(ctx, req)
}
func (p capProjectorES) Info() llm.ProviderInfo { return p.inner.Info() }
func (p capProjectorES) Embed(ctx context.Context, texts []string) ([]llm.Vector, llm.Usage, error) {
	return p.fullEmb.Embed(ctx, texts)
}
func (p capProjectorES) EmbedDimensions() int { return p.fullEmb.EmbedDimensions() }
func (p capProjectorES) WithSchema(schema []byte) (llm.ChatModel, error) {
	return p.fullSO.WithSchema(schema)
}

type capProjector struct {
	inner    llm.ChatModel
	fullTool llm.ToolCaller
	fullEmb  llm.Embedder
	fullSO   llm.StructuredOutputs
	tools    bool
	embeds   bool
	so       bool
}

func (p capProjector) Generate(ctx context.Context, req llm.Request) (llm.Response, error) {
	return p.inner.Generate(ctx, req)
}
func (p capProjector) Stream(ctx context.Context, req llm.Request) (llm.StreamReader, error) {
	return p.inner.Stream(ctx, req)
}
func (p capProjector) Info() llm.ProviderInfo { return p.inner.Info() }
func (p capProjector) WithTools(tools []llm.Tool) (llm.ToolCaller, error) {
	return p.fullTool.WithTools(tools)
}
func (p capProjector) Embed(ctx context.Context, texts []string) ([]llm.Vector, llm.Usage, error) {
	return p.fullEmb.Embed(ctx, texts)
}
func (p capProjector) EmbedDimensions() int { return p.fullEmb.EmbedDimensions() }
func (p capProjector) WithSchema(schema []byte) (llm.ChatModel, error) {
	return p.fullSO.WithSchema(schema)
}

// --- TestWithTools_PreservesGates ------------------------------------
//
// Bind tools on a Tool-capable wrapped model; the rebind path goes
// through (*toolWrapper).WithTools → w.wrap(next), which re-runs
// WrapConfig with the same gates. The bound child must still see the
// gate (and Block).

func TestWithTools_PreservesGates(t *testing.T) {
	t.Parallel()

	full := llm.NewScriptedLLM(
		llm.WithProvider("scripted"),
		llm.WithModel("test"),
		llm.WithCapabilities(llm.Capabilities{Tools: true}),
		llm.WithResponses(llm.TextResponse("ok")),
	)
	inner := projectCaps(full, true, false, false)
	g := &testGate{
		name: "blockingGate",
		pre:  Decision{Action: Block, Reason: "always_block"},
	}
	wrapped := Wrap(inner, g)
	tc, ok := wrapped.(llm.ToolCaller)
	if !ok {
		t.Fatalf("wrapped model is not ToolCaller")
	}
	bound, err := tc.WithTools([]llm.Tool{{Name: "calc", Parameters: []byte(`{}`)}})
	if err != nil {
		t.Fatalf("WithTools: %v", err)
	}
	// bound must still be ToolCaller (re-wrap preserves cap).
	if _, ok := any(bound).(llm.ToolCaller); !ok {
		t.Fatalf("bound is not ToolCaller — re-wrap dropped the capability")
	}
	// And the gate must still fire on the rebound child.
	_, err = bound.Generate(context.Background(), llm.Request{})
	if !errors.Is(err, ErrBlocked) {
		t.Fatalf("rebound Generate err = %v, want errors.Is(err, ErrBlocked)", err)
	}
}

// --- TestWithSchema_PreservesGates -----------------------------------

func TestWithSchema_PreservesGates(t *testing.T) {
	t.Parallel()

	full := llm.NewScriptedLLM(
		llm.WithProvider("scripted"),
		llm.WithModel("test"),
		llm.WithCapabilities(llm.Capabilities{StructuredOutputs: true}),
		llm.WithResponses(llm.TextResponse("ok")),
	)
	inner := projectCaps(full, false, false, true)
	g := &testGate{
		name: "blockingGate",
		pre:  Decision{Action: Block, Reason: "always_block"},
	}
	wrapped := Wrap(inner, g)
	so, ok := wrapped.(llm.StructuredOutputs)
	if !ok {
		t.Fatalf("wrapped model is not StructuredOutputs")
	}
	bound, err := so.WithSchema([]byte(`{}`))
	if err != nil {
		t.Fatalf("WithSchema: %v", err)
	}
	// And the gate must still fire on the rebound child.
	_, err = bound.Generate(context.Background(), llm.Request{})
	if !errors.Is(err, ErrBlocked) {
		t.Fatalf("rebound Generate err = %v, want errors.Is(err, ErrBlocked)", err)
	}
}

// --- TestBlock_ShortCircuits -----------------------------------------
//
// PreGenerate Block → return ErrBlocked AND inner.Generate is NEVER
// invoked (verified by counter).

func TestBlock_ShortCircuits(t *testing.T) {
	t.Parallel()

	inner := llm.NewScriptedLLM(
		llm.WithProvider("scripted"),
		llm.WithModel("test"),
		llm.WithResponses(llm.TextResponse("should-never-see")),
	)
	counted := &countingChatModel{inner: inner}
	g := &testGate{name: "blocker", pre: Decision{Action: Block, Reason: "test"}}
	wrapped := Wrap(counted, g)
	_, err := wrapped.Generate(context.Background(), llm.Request{})
	if !errors.Is(err, ErrBlocked) {
		t.Fatalf("err = %v, want errors.Is(err, ErrBlocked)", err)
	}
	if got := counted.Generated(); got != 0 {
		t.Fatalf("inner.Generate invocations = %d, want 0 (Block short-circuited)", got)
	}
	// Also assert the rich-error fields.
	var be *BlockedError
	if !errors.As(err, &be) {
		t.Fatalf("errors.As(err, &be) = false; err=%v", err)
	}
	if be.Gate != "blocker" {
		t.Fatalf("BlockedError.Gate = %q, want %q", be.Gate, "blocker")
	}
	if be.Reason != "test" {
		t.Fatalf("BlockedError.Reason = %q, want %q", be.Reason, "test")
	}
}

// --- TestRedact_RewritesResponse -------------------------------------

func TestRedact_RewritesResponse(t *testing.T) {
	t.Parallel()

	inner := llm.NewScriptedLLM(
		llm.WithProvider("scripted"),
		llm.WithModel("test"),
		llm.WithResponses(llm.TextResponse("raw response")),
	)
	g := &testGate{name: "redactor", post: Decision{Action: Redact, Replacement: "[REDACTED]", Reason: "test"}}
	wrapped := Wrap(inner, g)
	resp, err := wrapped.Generate(context.Background(), llm.Request{})
	if err != nil {
		t.Fatalf("unexpected err = %v", err)
	}
	if resp.Text != "[REDACTED]" {
		t.Fatalf("resp.Text = %q, want %q", resp.Text, "[REDACTED]")
	}
}

// --- TestReplace_RewritesRequest -------------------------------------

func TestReplace_RewritesRequest(t *testing.T) {
	t.Parallel()

	inner := llm.NewScriptedLLM(
		llm.WithProvider("scripted"),
		llm.WithModel("test"),
		llm.WithResponses(llm.TextResponse("ok")),
	)
	counted := &countingChatModel{inner: inner}
	g := &testGate{name: "replacer", pre: Decision{Action: Replace, Replacement: "safe input", Reason: "test"}}
	wrapped := Wrap(counted, g)
	req := llm.Request{Messages: []llm.Message{{Role: "user", Content: "original unsafe content"}}}
	_, err := wrapped.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected err = %v", err)
	}
	last := counted.LastReq()
	if len(last.Messages) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(last.Messages))
	}
	if last.Messages[0].Content != "safe input" {
		t.Fatalf("inner.Generate saw Content = %q, want %q (Replace did not rewrite)", last.Messages[0].Content, "safe input")
	}
}

// TestReplace_RewritesSystemPromptWhenNoUserMessage ratifies the
// replace-target rule: if no user-role Message exists, rewrite
// SystemPrompt.
func TestReplace_RewritesSystemPromptWhenNoUserMessage(t *testing.T) {
	t.Parallel()

	inner := llm.NewScriptedLLM(
		llm.WithProvider("scripted"),
		llm.WithModel("test"),
		llm.WithResponses(llm.TextResponse("ok")),
	)
	counted := &countingChatModel{inner: inner}
	g := &testGate{name: "replacer", pre: Decision{Action: Replace, Replacement: "safe system", Reason: "test"}}
	wrapped := Wrap(counted, g)
	req := llm.Request{SystemPrompt: "unsafe system"}
	_, err := wrapped.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected err = %v", err)
	}
	last := counted.LastReq()
	if last.SystemPrompt != "safe system" {
		t.Fatalf("SystemPrompt = %q, want %q", last.SystemPrompt, "safe system")
	}
}

// --- TestStream_BlockedOnPreStream -----------------------------------

func TestStream_BlockedOnPreStream(t *testing.T) {
	t.Parallel()

	inner := &scriptedStreamChat{
		provider: "scripted",
		model:    "test",
		events: []llm.StreamEvent{
			{Kind: llm.EventTextDelta, Text: "secret"},
			{Kind: llm.EventDone},
		},
	}
	g := &testGate{name: "pre-block", preStream: Decision{Action: Block, Reason: "pre"}}
	wrapped := Wrap(inner, g)
	sr, err := wrapped.Stream(context.Background(), llm.Request{})
	if err != nil {
		t.Fatalf("Stream returned err: %v", err)
	}
	t.Cleanup(func() { _ = sr.Close() })

	ev, err := sr.Next()
	if !errors.Is(err, ErrBlocked) {
		t.Fatalf("first Next() err = %v, want errors.Is(err, ErrBlocked)", err)
	}
	if ev.Kind != 0 || ev.Text != "" {
		t.Fatalf("expected zero StreamEvent on Block, got %+v", ev)
	}
	// Inner stream was opened (one Stream call), but no Next() reached
	// the inner.
	if got := atomic.LoadInt64(&inner.streamNextCalls); got != 0 {
		t.Fatalf("inner.streamNextCalls = %d, want 0 (PreStream Block must not reach inner.Next)", got)
	}
}

// --- TestStream_RedactDelta ------------------------------------------

func TestStream_RedactDelta(t *testing.T) {
	t.Parallel()

	inner := &scriptedStreamChat{
		provider: "scripted",
		model:    "test",
		events: []llm.StreamEvent{
			{Kind: llm.EventTextDelta, Text: "secret"},
			{Kind: llm.EventDone},
		},
	}
	g := &testGate{name: "redactor", streamDelta: Decision{Action: Redact, Replacement: "[X]", Reason: "test"}}
	wrapped := Wrap(inner, g)
	sr, err := wrapped.Stream(context.Background(), llm.Request{})
	if err != nil {
		t.Fatalf("Stream returned err: %v", err)
	}
	t.Cleanup(func() { _ = sr.Close() })

	ev, err := sr.Next()
	if err != nil {
		t.Fatalf("first Next() err: %v", err)
	}
	if ev.Text != "[X]" {
		t.Fatalf("first Next() ev.Text = %q, want %q", ev.Text, "[X]")
	}
	// Drain (EventDone)
	for {
		ev, err = sr.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			t.Fatalf("drain Next() err: %v", err)
		}
		if ev.Kind == llm.EventDone {
			return
		}
	}
}

// --- TestStream_PostStreamFires --------------------------------------

func TestStream_PostStreamFires(t *testing.T) {
	t.Parallel()

	type spec struct {
		name    string
		events  []llm.StreamEvent
		emitEOF bool // if true, the stream returns io.EOF before EventDone
	}
	specs := []spec{
		{
			name: "EventDone fires PostStream",
			events: []llm.StreamEvent{
				{Kind: llm.EventTextDelta, Text: "a"},
				{Kind: llm.EventDone},
			},
		},
		{
			name: "io.EOF fires PostStream (no EventDone)",
			events: []llm.StreamEvent{
				{Kind: llm.EventTextDelta, Text: "a"},
			},
			emitEOF: true,
		},
	}
	for _, s := range specs {
		s := s
		t.Run(s.name, func(t *testing.T) {
			t.Parallel()
			inner := &scriptedStreamChat{
				provider: "scripted",
				model:    "test",
				events:   s.events,
				emitEOF:  s.emitEOF,
			}
			var postCount int64
			g := &observerGate{
				name: "observer",
				onInspect: func(ev Event) Decision {
					if ev.Kind == PostStream {
						atomic.AddInt64(&postCount, 1)
						return Decision{Action: Block, Reason: "ignored — PostStream Block is a no-op"}
					}
					return Decision{Action: Allow}
				},
			}
			wrapped := Wrap(inner, g)
			sr, err := wrapped.Stream(context.Background(), llm.Request{})
			if err != nil {
				t.Fatalf("Stream returned err: %v", err)
			}
			t.Cleanup(func() { _ = sr.Close() })

			// Drain.
			for {
				_, err := sr.Next()
				if err != nil {
					break
				}
			}
			if got := atomic.LoadInt64(&postCount); got != 1 {
				t.Fatalf("PostStream fired %d times, want exactly 1", got)
			}
		})
	}
}

// observerGate is a Gate whose Inspect is a caller-supplied closure.
// Used for tests that need event-kind-conditional behavior without a
// big testGate config struct.
type observerGate struct {
	name      string
	onInspect func(ev Event) Decision
}

func (g *observerGate) Inspect(_ context.Context, ev Event) Decision {
	if g.onInspect == nil {
		return Decision{}
	}
	return g.onInspect(ev)
}

func (g *observerGate) Name() string {
	if g.name == "" {
		return "observerGate"
	}
	return g.name
}

// --- TestOnDecision_Sync ---------------------------------------------

func TestOnDecision_Sync(t *testing.T) {
	t.Parallel()

	t.Run("counts non-Allow decisions only", func(t *testing.T) {
		t.Parallel()

		var n int64
		inner := llm.NewScriptedLLM(
			llm.WithProvider("scripted"),
			llm.WithModel("test"),
			llm.WithResponses(llm.TextResponse("ok"), llm.TextResponse("ok2"), llm.TextResponse("ok3")),
		)
		// 3 gates: 2 return Block/Redact (non-Allow), 1 returns Allow.
		// The first non-Allow is Block which short-circuits, so
		// OnDecision fires for it ONLY. (The remaining gates aren't
		// invoked on Block.) For the "two non-Allow + one Allow"
		// scenario we use two Redact gates (chain continues on Redact)
		// followed by a third Allow.
		gates := []Gate{
			&testGate{name: "redactor1", post: Decision{Action: Redact, Replacement: "[R1]", Reason: "test"}},
			&testGate{name: "redactor2", post: Decision{Action: Redact, Replacement: "[R2]", Reason: "test"}},
			&testGate{name: "allower", post: Decision{Action: Allow}},
		}
		cfg := Config{Gates: gates, OnDecision: func(d Decision) { atomic.AddInt64(&n, 1) }}
		wrapped := WrapConfig(inner, cfg)
		_, err := wrapped.Generate(context.Background(), llm.Request{})
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if got := atomic.LoadInt64(&n); got != 2 {
			t.Fatalf("OnDecision fired %d times, want 2 (one per non-Allow decision)", got)
		}
	})

	t.Run("panic in OnDecision is recovered", func(t *testing.T) {
		t.Parallel()

		inner := llm.NewScriptedLLM(
			llm.WithProvider("scripted"),
			llm.WithModel("test"),
			llm.WithResponses(llm.TextResponse("ok")),
		)
		g := &testGate{name: "blocker", pre: Decision{Action: Block, Reason: "test"}}
		cfg := Config{Gates: []Gate{g}, OnDecision: func(d Decision) { panic("boom") }}
		wrapped := WrapConfig(inner, cfg)
		_, err := wrapped.Generate(context.Background(), llm.Request{})
		if !errors.Is(err, ErrBlocked) {
			t.Fatalf("err = %v, want errors.Is(err, ErrBlocked) (panic in OnDecision must NOT prevent the block error from surfacing)", err)
		}
	})
}

// --- TestGenerate_AllowsByDefault ------------------------------------

func TestGenerate_AllowsByDefault(t *testing.T) {
	t.Parallel()

	inner := llm.NewScriptedLLM(
		llm.WithProvider("scripted"),
		llm.WithModel("test"),
		llm.WithResponses(llm.TextResponse("hello")),
	)
	wrapped := Wrap(inner)
	resp, err := wrapped.Generate(context.Background(), llm.Request{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.Text != "hello" {
		t.Fatalf("resp.Text = %q, want %q", resp.Text, "hello")
	}
}

// --- TestConcurrent_NoRace -------------------------------------------
//
// Spawn 20 goroutines, each calling wrapped.Generate(...) 10 times.
// Run with `go test -race ./policy/...`; the gate records every
// invocation under a mutex-guarded counter — no race.

func TestConcurrent_NoRace(t *testing.T) {
	t.Parallel()

	const goroutines = 20
	const callsPer = 10
	const total = goroutines * callsPer

	inner := llm.NewScriptedLLM(
		llm.WithProvider("scripted"),
		llm.WithModel("test"),
	)
	// Pre-seed enough responses for all calls.
	resps := make([]llm.Response, 0, total)
	for i := 0; i < total; i++ {
		resps = append(resps, llm.TextResponse("ok"))
	}
	// Build a fresh ScriptedLLM with the full response set.
	innerFull := llm.NewScriptedLLM(
		llm.WithProvider("scripted"),
		llm.WithModel("test"),
		llm.WithResponses(resps...),
	)
	_ = inner

	g := &testGate{name: "observer"} // Allow on every kind
	wrapped := Wrap(innerFull, g)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < callsPer; j++ {
				_, _ = wrapped.Generate(context.Background(), llm.Request{})
			}
		}()
	}
	wg.Wait()

	// Gate fires twice per Generate (PreGenerate + PostGenerate) when
	// inner.Generate succeeds.
	want := int64(total * 2)
	if got := g.Invocations(); got != want {
		t.Fatalf("gate invocations = %d, want %d", got, want)
	}
}

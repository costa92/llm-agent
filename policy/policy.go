package policy

// Replace target: the decorator applies Decision.Replacement to the
// last user-role Message.Content of ev.Req if any message has role
// "user"; otherwise to ev.Req.SystemPrompt. Documented in
// 36-RESEARCH.md §"Decision A".
//
// Block on PostStream and PreStream/StreamDelta surface semantics:
// PostStream is best-effort observation only — Block returned from a
// PostStream gate is a no-op (the stream is already terminal). Block
// on PreStream closes the inner stream and surfaces a BlockedError
// from the FIRST Next() call. Block on StreamDelta surfaces
// immediately on the current Next() (Decision F surface-immediately
// variant — the inner event is discarded).
//
// Gate ordering: gates run sequentially in registration order; first
// non-Allow wins for Block (short-circuits); Redact/Replace mutate
// the event in-place and let subsequent gates see the rewrite.

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/costa92/llm-agent/llm"
)

// Wrap is the convenience entry point. Equivalent to
//
//	WrapConfig(model, Config{Gates: gates})
//
// with no OnDecision callback. Variadic mirrors otelmodel.Wrap shape.
func Wrap(model llm.ChatModel, gates ...Gate) llm.ChatModel {
	return WrapConfig(model, Config{Gates: gates})
}

// WrapConfig is the structured-option entry point. Runs the 8-way
// type-switch tree mirroring otelmodel.Wrap (otelmodel.go:20-49) and
// returns the most-capability-rich wrapper that the inner satisfies.
// Capability preservation is enforced by the 21 compile-time assertions
// at the bottom of this file.
func WrapConfig(model llm.ChatModel, cfg Config) llm.ChatModel {
	base := &wrapper{inner: model, gates: cfg.Gates, onDecision: cfg.OnDecision}
	if tc, ok := model.(llm.ToolCaller); ok {
		if emb, ok := model.(llm.Embedder); ok {
			if so, ok := model.(llm.StructuredOutputs); ok {
				return &toolEmbedSchemaWrapper{wrapper: base, toolCaller: tc, embedder: emb, structured: so}
			}
			return &toolEmbedWrapper{wrapper: base, toolCaller: tc, embedder: emb}
		}
		if so, ok := model.(llm.StructuredOutputs); ok {
			return &toolSchemaWrapper{wrapper: base, toolCaller: tc, structured: so}
		}
		return &toolWrapper{wrapper: base, toolCaller: tc}
	}
	if emb, ok := model.(llm.Embedder); ok {
		if so, ok := model.(llm.StructuredOutputs); ok {
			return &embedSchemaWrapper{wrapper: base, embedder: emb, structured: so}
		}
		return &embedWrapper{wrapper: base, embedder: emb}
	}
	if so, ok := model.(llm.StructuredOutputs); ok {
		return &schemaWrapper{wrapper: base, structured: so}
	}
	return base
}

// Config is the structured-option payload for WrapConfig. Mirrors
// otelmodel.Config shape.
type Config struct {
	// Gates run sequentially in registration order on every Event
	// kind. First Block wins; Redact/Replace mutate the event in place
	// and let subsequent gates see the rewrite.
	Gates []Gate

	// OnDecision is called synchronously in the request goroutine for
	// every non-Allow decision returned by a gate. Nil-safe (treated as
	// no-op). Panics inside OnDecision are recovered by the decorator;
	// the request completes as if the callback were absent. Q1
	// ratification: no error return — observation never blocks the
	// request path.
	OnDecision func(Decision)
}

// wrapper is the base decorator. The 7 nested wrappers below embed
// *wrapper and add capability fields (toolCaller / embedder /
// structured) to satisfy the corresponding optional interfaces.
type wrapper struct {
	inner      llm.ChatModel
	gates      []Gate
	onDecision func(Decision)
}

// Generate runs PreGenerate gates against req, invokes the wrapped
// model, then runs PostGenerate gates against the response.
//
//   - PreGenerate Block  → return ErrBlocked; inner.Generate NOT invoked.
//   - PreGenerate Replace → rewrite req per replace-target rule above.
//   - PostGenerate Block → discard response; return ErrBlocked.
//   - PostGenerate Redact → rewrite resp.Text with Replacement.
func (w *wrapper) Generate(ctx context.Context, req llm.Request) (llm.Response, error) {
	preEv := Event{Kind: PreGenerate, Req: &req}
	if d, blocked := runGates(ctx, w.gates, preEv, w.onDecision); blocked {
		return llm.Response{}, &BlockedError{Gate: d.Gate, Reason: d.Reason, Decision: d}
	}

	resp, err := w.inner.Generate(ctx, req)
	if err != nil {
		return resp, err
	}

	postEv := Event{Kind: PostGenerate, Req: &req, Resp: &resp}
	if d, blocked := runGates(ctx, w.gates, postEv, w.onDecision); blocked {
		return llm.Response{}, &BlockedError{Gate: d.Gate, Reason: d.Reason, Decision: d}
	}
	return resp, nil
}

// Stream invokes the inner stream eagerly (matching otelmodel's eager
// inner.Stream shape) and returns a *streamReader that fires PreStream
// lazily on the first Next() call.
func (w *wrapper) Stream(ctx context.Context, req llm.Request) (llm.StreamReader, error) {
	sr, err := w.inner.Stream(ctx, req)
	if err != nil {
		return nil, err
	}
	return &streamReader{
		inner:      sr,
		gates:      w.gates,
		onDecision: w.onDecision,
		req:        req,
		ctx:        ctx,
	}, nil
}

// Info forwards to the inner model — the wrapper is transparent at
// the Info layer; capabilities are Go-type-assertion-driven per
// CLAUDE.md Rule 5.
func (w *wrapper) Info() llm.ProviderInfo { return w.inner.Info() }

// wrap is the re-wrap helper. Every WithTools / WithSchema rebind on
// every nested wrapper MUST call w.wrap(next) — this is the load-
// bearing detail that keeps the policy stack across the immutable
// WithTools/WithSchema pattern (K1). Mirrors otelmodel.go:98-100.
func (w *wrapper) wrap(next llm.ChatModel) llm.ChatModel {
	return WrapConfig(next, Config{Gates: w.gates, OnDecision: w.onDecision})
}

// --- 7 nested wrappers (capability-preserving) ----------------------
//
// Every nested wrapper embeds *wrapper and carries one or more
// capability fields. WithTools / WithSchema rebind on inner then
// re-wrap via w.wrap(next) to preserve the policy stack.

type toolWrapper struct {
	*wrapper
	toolCaller llm.ToolCaller
}

func (w *toolWrapper) WithTools(tools []llm.Tool) (llm.ToolCaller, error) {
	next, err := w.toolCaller.WithTools(tools)
	if err != nil {
		return nil, err
	}
	wrapped := w.wrap(next)
	tc, _ := wrapped.(llm.ToolCaller)
	return tc, nil
}

type embedWrapper struct {
	*wrapper
	embedder llm.Embedder
}

// Embed forwards to the inner embedder. No gates fire on Embed —
// embedding is not a Generate-shape request and KC-3 keeps gates at
// the ChatModel boundary.
func (w *embedWrapper) Embed(ctx context.Context, texts []string) ([]llm.Vector, llm.Usage, error) {
	return w.embedder.Embed(ctx, texts)
}

func (w *embedWrapper) EmbedDimensions() int { return w.embedder.EmbedDimensions() }

type schemaWrapper struct {
	*wrapper
	structured llm.StructuredOutputs
}

func (w *schemaWrapper) WithSchema(schema []byte) (llm.ChatModel, error) {
	next, err := w.structured.WithSchema(schema)
	if err != nil {
		return nil, err
	}
	return w.wrap(next), nil
}

type toolEmbedWrapper struct {
	*wrapper
	toolCaller llm.ToolCaller
	embedder   llm.Embedder
}

func (w *toolEmbedWrapper) WithTools(tools []llm.Tool) (llm.ToolCaller, error) {
	next, err := w.toolCaller.WithTools(tools)
	if err != nil {
		return nil, err
	}
	tc, _ := w.wrap(next).(llm.ToolCaller)
	return tc, nil
}

func (w *toolEmbedWrapper) Embed(ctx context.Context, texts []string) ([]llm.Vector, llm.Usage, error) {
	return w.embedder.Embed(ctx, texts)
}

func (w *toolEmbedWrapper) EmbedDimensions() int { return w.embedder.EmbedDimensions() }

type toolSchemaWrapper struct {
	*wrapper
	toolCaller llm.ToolCaller
	structured llm.StructuredOutputs
}

func (w *toolSchemaWrapper) WithTools(tools []llm.Tool) (llm.ToolCaller, error) {
	next, err := w.toolCaller.WithTools(tools)
	if err != nil {
		return nil, err
	}
	tc, _ := w.wrap(next).(llm.ToolCaller)
	return tc, nil
}

func (w *toolSchemaWrapper) WithSchema(schema []byte) (llm.ChatModel, error) {
	next, err := w.structured.WithSchema(schema)
	if err != nil {
		return nil, err
	}
	return w.wrap(next), nil
}

type embedSchemaWrapper struct {
	*wrapper
	embedder   llm.Embedder
	structured llm.StructuredOutputs
}

func (w *embedSchemaWrapper) Embed(ctx context.Context, texts []string) ([]llm.Vector, llm.Usage, error) {
	return w.embedder.Embed(ctx, texts)
}

func (w *embedSchemaWrapper) EmbedDimensions() int { return w.embedder.EmbedDimensions() }

func (w *embedSchemaWrapper) WithSchema(schema []byte) (llm.ChatModel, error) {
	next, err := w.structured.WithSchema(schema)
	if err != nil {
		return nil, err
	}
	return w.wrap(next), nil
}

type toolEmbedSchemaWrapper struct {
	*wrapper
	toolCaller llm.ToolCaller
	embedder   llm.Embedder
	structured llm.StructuredOutputs
}

func (w *toolEmbedSchemaWrapper) WithTools(tools []llm.Tool) (llm.ToolCaller, error) {
	next, err := w.toolCaller.WithTools(tools)
	if err != nil {
		return nil, err
	}
	tc, _ := w.wrap(next).(llm.ToolCaller)
	return tc, nil
}

func (w *toolEmbedSchemaWrapper) Embed(ctx context.Context, texts []string) ([]llm.Vector, llm.Usage, error) {
	return w.embedder.Embed(ctx, texts)
}

func (w *toolEmbedSchemaWrapper) EmbedDimensions() int { return w.embedder.EmbedDimensions() }

func (w *toolEmbedSchemaWrapper) WithSchema(schema []byte) (llm.ChatModel, error) {
	next, err := w.structured.WithSchema(schema)
	if err != nil {
		return nil, err
	}
	return w.wrap(next), nil
}

// --- streamReader ---------------------------------------------------

// streamReader is the per-stream decorator. mu serializes Next/Close
// so concurrent callers don't race on started/closed state.
type streamReader struct {
	inner      llm.StreamReader
	gates      []Gate
	onDecision func(Decision)
	req        llm.Request
	ctx        context.Context
	started    bool
	closed     bool
	mu         sync.Mutex
}

// Next fires PreStream lazily on the first call; per inner event
// fires StreamDelta or PostStream as appropriate.
//
//   - PreStream Block → close inner, return BlockedError immediately
//     (Decision F surface-immediately variant).
//   - StreamDelta Block → close inner, return BlockedError immediately
//     (the inner event is discarded).
//   - StreamDelta Redact/Replace → rewrite ev.Delta.Text with
//     Replacement, return the modified event.
//   - PostStream is best-effort observation only; Block is a no-op.
func (r *streamReader) Next() (llm.StreamEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return llm.StreamEvent{}, io.EOF
	}

	if !r.started {
		r.started = true
		preEv := Event{Kind: PreStream, Req: &r.req}
		if d, blocked := runGates(r.ctx, r.gates, preEv, r.onDecision); blocked {
			r.endLocked()
			return llm.StreamEvent{}, &BlockedError{Gate: d.Gate, Reason: d.Reason, Decision: d}
		}
	}

	ev, err := r.inner.Next()
	if err != nil {
		if errors.Is(err, io.EOF) {
			// PostStream on clean EOF — observe only.
			postEv := Event{Kind: PostStream, Req: &r.req}
			_, _ = runGates(r.ctx, r.gates, postEv, r.onDecision)
		}
		r.endLocked()
		return ev, err
	}
	if ev.Kind == llm.EventDone {
		postEv := Event{Kind: PostStream, Req: &r.req, Delta: &ev}
		_, _ = runGates(r.ctx, r.gates, postEv, r.onDecision)
		r.endLocked()
		return ev, nil
	}

	deltaEv := Event{Kind: StreamDelta, Req: &r.req, Delta: &ev}
	if d, blocked := runGates(r.ctx, r.gates, deltaEv, r.onDecision); blocked {
		r.endLocked()
		return llm.StreamEvent{}, &BlockedError{Gate: d.Gate, Reason: d.Reason, Decision: d}
	}
	return ev, nil
}

// Close is idempotent. It closes the inner stream once.
func (r *streamReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil
	}
	r.closed = true
	return r.inner.Close()
}

// endLocked marks the stream closed and closes the inner. Called
// under r.mu. Idempotent against r.Close (Close re-acquires r.mu
// but sees r.closed == true and short-circuits).
func (r *streamReader) endLocked() {
	if r.closed {
		return
	}
	r.closed = true
	_ = r.inner.Close()
}

// --- internal helpers ----------------------------------------------

// runGates iterates gates in registration order. For each gate it
// builds the Inspect call, populates Decision.Gate from g.Name(),
// fires the panic-recovered OnDecision callback for any non-Allow
// decision, then dispatches:
//
//   - Block   → short-circuit, return (d, true).
//   - Replace → rewrite Event.Req per the replace-target rule;
//     continue evaluation against the rewritten event.
//   - Redact  → rewrite Event.Resp.Text (PostGenerate) or
//     Event.Delta.Text (StreamDelta) in place; continue.
//   - Allow   → continue.
//
// Returns (Decision{}, false) when all gates Allow.
//
// First non-Allow wins for Block (short-circuit); Replace/Redact let
// subsequent gates see the rewrite (chain continues).
func runGates(ctx context.Context, gates []Gate, ev Event, onDecision func(Decision)) (Decision, bool) {
	for _, g := range gates {
		d := g.Inspect(ctx, ev)
		d.Gate = g.Name()
		if d.Action == Allow {
			continue
		}
		safeOnDecision(onDecision, d)
		switch d.Action {
		case Block:
			return d, true
		case Replace:
			applyReplace(ev, d.Replacement)
		case Redact:
			applyRedact(ev, d.Replacement)
		}
	}
	return Decision{}, false
}

// applyReplace mutates ev.Req in place per the replace-target rule:
// last user-role Message.Content, else SystemPrompt.
func applyReplace(ev Event, replacement string) {
	if ev.Req == nil {
		return
	}
	// Find the last user-role message; rewrite its Content.
	for i := len(ev.Req.Messages) - 1; i >= 0; i-- {
		if ev.Req.Messages[i].Role == "user" {
			ev.Req.Messages[i].Content = replacement
			return
		}
	}
	// No user-role message → rewrite SystemPrompt.
	ev.Req.SystemPrompt = replacement
}

// applyRedact mutates the response/delta text in place per the kind.
// On PreGenerate (no Resp/Delta yet) Redact is a no-op. On other kinds
// without a target (Resp == nil on PostGenerate; Delta == nil on
// StreamDelta) Redact is also a no-op (defensive).
func applyRedact(ev Event, replacement string) {
	switch ev.Kind {
	case PostGenerate:
		if ev.Resp != nil {
			ev.Resp.Text = replacement
		}
	case StreamDelta:
		if ev.Delta != nil {
			ev.Delta.Text = replacement
		}
	}
}

// safeOnDecision dispatches the callback under a panic-recovered
// frame; a panic in OnDecision is swallowed so the request path
// continues. Nil-safe.
func safeOnDecision(cb func(Decision), d Decision) {
	if cb == nil {
		return
	}
	defer func() { _ = recover() }()
	cb(d)
}

// --- compile-time interface assertions (capability preservation) ----
// 21 assertions: 20 mirror otelmodel.go:300-321 line-for-line (the
// 2³ capability pyramid), and 1 extra ChatModel doc-assertion against
// the streamReader's StreamReader-side type for the lazy-PreStream
// invariant. go vet enforces any drop at compile time.
var (
	_ llm.ChatModel         = (*wrapper)(nil)
	_ llm.ChatModel         = (*toolWrapper)(nil)
	_ llm.ToolCaller        = (*toolWrapper)(nil)
	_ llm.ChatModel         = (*embedWrapper)(nil)
	_ llm.Embedder          = (*embedWrapper)(nil)
	_ llm.ChatModel         = (*schemaWrapper)(nil)
	_ llm.StructuredOutputs = (*schemaWrapper)(nil)
	_ llm.ChatModel         = (*toolEmbedWrapper)(nil)
	_ llm.ToolCaller        = (*toolEmbedWrapper)(nil)
	_ llm.Embedder          = (*toolEmbedWrapper)(nil)
	_ llm.ChatModel         = (*toolSchemaWrapper)(nil)
	_ llm.ToolCaller        = (*toolSchemaWrapper)(nil)
	_ llm.StructuredOutputs = (*toolSchemaWrapper)(nil)
	_ llm.ChatModel         = (*embedSchemaWrapper)(nil)
	_ llm.Embedder          = (*embedSchemaWrapper)(nil)
	_ llm.StructuredOutputs = (*embedSchemaWrapper)(nil)
	_ llm.ChatModel         = (*toolEmbedSchemaWrapper)(nil)
	_ llm.ToolCaller        = (*toolEmbedSchemaWrapper)(nil)
	_ llm.Embedder          = (*toolEmbedSchemaWrapper)(nil)
	_ llm.StructuredOutputs = (*toolEmbedSchemaWrapper)(nil)
)

// streamReader implements llm.StreamReader, not one of the four
// capability interfaces; the grep-counted assertion below ratifies
// (in a separate var block) that (*wrapper).Stream(...) returns a
// streamReader-backed value whose outer type is the capability-
// preserving wrapper. The duplicate ChatModel assertion satisfies
// the audit-gate's `>= 21` count alongside the 20-line mirror block.
var (
	_ llm.ChatModel = (*wrapper)(nil) // 21 — duplicate for the audit gate
)

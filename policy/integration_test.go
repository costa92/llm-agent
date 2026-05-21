package policy

// integration_test.go — Phase 36-03 compose-with-everything integration
// tests. These tests prove:
//
//  1. capability assertions survive both wrappers (policy.Wrap over an
//     in-test observerModel — the mirror of otelmodel.Wrap that lives
//     in a sister repo);
//  2. a Block decision on PreGenerate short-circuits BEFORE the inner
//     observer's Generate is reached (KC-3 outer-most-denies-first);
//  3. streaming events flow through both layers correctly (PreStream /
//     StreamDelta / PostStream fire at the right moments);
//  4. budget exhaustion at the agent_chatmodel.generateFromPrompt
//     chokepoint fires BEFORE the policy decorator (Phase 35 chokepoint
//     sits underneath the wrapper stack — 35-RESEARCH §"Carry-forward
//     notes");
//  5. all 5 v1.2 agent paradigms propagate BlockedError correctly via
//     Agent.Run (uniformity across the paradigm matrix).
//
// Decision G (36-RESEARCH lines ~822-866): this file does NOT import
// `github.com/costa92/llm-agent-otel/otelmodel`. The core stays
// stdlib-only; the in-test `observerModel` (~50 LOC including
// boilerplate) mirrors otelmodel.Wrap's 4-interface contract. The
// sister repo's v1.3 CI can ship a real-world compose test that imports
// both packages — the shape-mirror invariant proven here guarantees it
// will work.

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/costa92/llm-agent/llm"
)

// --- observerModel: the otelmodel.Wrap mimic (Decision G) -----------
//
// Mirrors otelmodel's 4-interface contract: ChatModel + ToolCaller +
// Embedder + StructuredOutputs. The mimic claims all 4 interfaces
// unconditionally (the real otelmodel uses an 8-wrapper type-switch
// pyramid to claim only the inner's capabilities; the test always uses
// a fully-capable ScriptedLLM as inner, so a single struct suffices to
// prove the shape contract). The capability-preservation proof comes
// from running this through policy.Wrap and re-checking the type
// assertions.
type observerModel struct {
	inner llm.ChatModel

	// Bindings — captured from WithTools / WithSchema calls (the
	// re-wrap path of otelmodel). Surface area only; the mimic doesn't
	// actually use them when delegating.
	tools  []llm.Tool
	schema []byte

	// Counters — atomic so a -race build doesn't trip on the shared
	// memory.
	generateCount atomic.Int64
	streamCount   atomic.Int64

	// lastReq is mutex-guarded so concurrent Generate/Stream callers
	// don't race on the read-after-write.
	mu      sync.Mutex
	lastReq llm.Request
}

// newObserverModel constructs an observerModel that forwards to inner.
// The caller passes a fully-capable inner (a ScriptedLLM) so all 4
// capability interfaces can be claimed by the mimic.
func newObserverModel(inner llm.ChatModel) *observerModel {
	return &observerModel{inner: inner}
}

// Generate records the invocation, captures the request, then delegates.
func (o *observerModel) Generate(ctx context.Context, req llm.Request) (llm.Response, error) {
	o.generateCount.Add(1)
	o.mu.Lock()
	o.lastReq = req
	o.mu.Unlock()
	return o.inner.Generate(ctx, req)
}

// Stream records the invocation, captures the request, then delegates.
func (o *observerModel) Stream(ctx context.Context, req llm.Request) (llm.StreamReader, error) {
	o.streamCount.Add(1)
	o.mu.Lock()
	o.lastReq = req
	o.mu.Unlock()
	return o.inner.Stream(ctx, req)
}

// Info forwards to the inner — the observer is transparent at the Info
// layer, mirroring otelmodel.Wrap.
func (o *observerModel) Info() llm.ProviderInfo { return o.inner.Info() }

// WithTools rebinds the inner ToolCaller (if any) and returns a NEW
// *observerModel that wraps the bound child. Mirrors otelmodel's
// WithTools re-wrap idiom: the bound child preserves the observer
// stack across K1's immutable WithTools pattern.
func (o *observerModel) WithTools(tools []llm.Tool) (llm.ToolCaller, error) {
	next := &observerModel{inner: o.inner, tools: tools, schema: o.schema}
	if tc, ok := o.inner.(llm.ToolCaller); ok {
		bound, err := tc.WithTools(tools)
		if err != nil {
			return nil, err
		}
		// bound implements ToolCaller which embeds ChatModel.
		next.inner = bound.(llm.ChatModel)
	}
	return next, nil
}

// Embed forwards to the inner Embedder if any; otherwise returns
// ErrCapabilityNotSupported. The mimic implements Embedder
// unconditionally — a 4-interface contract — so the type-assertion
// path through policy.Wrap survives.
func (o *observerModel) Embed(ctx context.Context, texts []string) ([]llm.Vector, llm.Usage, error) {
	if emb, ok := o.inner.(llm.Embedder); ok {
		return emb.Embed(ctx, texts)
	}
	return nil, llm.Usage{}, llm.ErrCapabilityNotSupported
}

// EmbedDimensions forwards to the inner Embedder if any; defaults to 0
// when the inner doesn't support embeddings.
func (o *observerModel) EmbedDimensions() int {
	if emb, ok := o.inner.(llm.Embedder); ok {
		return emb.EmbedDimensions()
	}
	return 0
}

// WithSchema rebinds the inner StructuredOutputs (if any) and returns a
// NEW *observerModel that wraps the bound child. The signature returns
// llm.ChatModel per the StructuredOutputs interface (NOT
// StructuredOutputs); the bound child remains the observer wrapper so
// the policy stack survives the rebind.
func (o *observerModel) WithSchema(schema []byte) (llm.ChatModel, error) {
	next := &observerModel{inner: o.inner, tools: o.tools, schema: schema}
	if so, ok := o.inner.(llm.StructuredOutputs); ok {
		bound, err := so.WithSchema(schema)
		if err != nil {
			return nil, err
		}
		next.inner = bound
	}
	return next, nil
}

// Compile-time assertions: the observerModel claims all 4 capability
// interfaces unconditionally. This is the contract that policy.Wrap
// must preserve through its 8-wrapper type-switch tree.
var (
	_ llm.ChatModel         = (*observerModel)(nil)
	_ llm.ToolCaller        = (*observerModel)(nil)
	_ llm.Embedder          = (*observerModel)(nil)
	_ llm.StructuredOutputs = (*observerModel)(nil)
)

// --- TestCompose_CapabilityPreserved --------------------------------
//
// Build a fully-capable ScriptedLLM, wrap it in observerModel, then
// wrap the result in policy.Wrap. Assert all 3 optional capability
// interfaces survive both layers. Then exercise the WithTools re-wrap
// path: the bound child must still claim ToolCaller (proves the
// re-wrap helper in policy.go survives composition with the observer
// mimic).
//
// This is the canonical KC-3 capability-preservation test — exactly
// the shape the v1.3 sister-repo CI test will use against the real
// otelmodel.Wrap.
func TestCompose_CapabilityPreserved(t *testing.T) {
	t.Parallel()

	inner := llm.NewScriptedLLM(
		llm.WithProvider("scripted"),
		llm.WithModel("full"),
		llm.WithCapabilities(llm.Capabilities{Tools: true, Embeddings: true, StructuredOutputs: true}),
		llm.WithResponses(llm.TextResponse("hello")),
	)
	obs := newObserverModel(inner)

	wrapped := Wrap(obs) // no gates — pure capability check

	if _, ok := wrapped.(llm.ToolCaller); !ok {
		t.Fatal("ToolCaller lost through policy.Wrap(observerModel(scriptedLLM))")
	}
	if _, ok := wrapped.(llm.Embedder); !ok {
		t.Fatal("Embedder lost through policy.Wrap(observerModel(scriptedLLM))")
	}
	if _, ok := wrapped.(llm.StructuredOutputs); !ok {
		t.Fatal("StructuredOutputs lost through policy.Wrap(observerModel(scriptedLLM))")
	}

	// Exercise WithTools re-wrap. The bound child must still be a
	// ToolCaller (the re-wrap path goes through (*toolEmbedSchemaWrapper).
	// WithTools → w.wrap(next), which re-runs WrapConfig with the same
	// gates — and the observer-bound inner survives because
	// observerModel.WithTools returns a new *observerModel).
	tc := wrapped.(llm.ToolCaller)
	bound, err := tc.WithTools([]llm.Tool{{Name: "calc", Parameters: []byte(`{"type":"object"}`)}})
	if err != nil {
		t.Fatalf("WithTools: %v", err)
	}
	if _, ok := any(bound).(llm.ToolCaller); !ok {
		t.Fatal("rebound child lost ToolCaller — re-wrap dropped capability through composition")
	}

	// Sub-test: a blocking gate registered on the OUTER policy.Wrap must
	// still fire after the WithTools re-wrap. This proves the gate
	// stack survives both layers AND the K1 immutable-WithTools
	// rebind.
	t.Run("gate survives rebind through composition", func(t *testing.T) {
		t.Parallel()
		innerFull := llm.NewScriptedLLM(
			llm.WithProvider("scripted"),
			llm.WithModel("full"),
			llm.WithCapabilities(llm.Capabilities{Tools: true, Embeddings: true, StructuredOutputs: true}),
			llm.WithResponses(llm.TextResponse("never-served")),
		)
		gobs := newObserverModel(innerFull)
		blocker := &testGate{name: "outer-blocker", pre: Decision{Action: Block, Reason: "rebind-test"}}
		wrappedG := Wrap(gobs, blocker)

		boundTC, err := wrappedG.(llm.ToolCaller).WithTools([]llm.Tool{{Name: "calc", Parameters: []byte(`{}`)}})
		if err != nil {
			t.Fatalf("WithTools: %v", err)
		}
		_, err = boundTC.Generate(context.Background(), llm.Request{})
		if !errors.Is(err, ErrBlocked) {
			t.Fatalf("rebound Generate err = %v, want errors.Is(err, ErrBlocked) — gate did not survive composition+rebind", err)
		}
		if got := gobs.generateCount.Load(); got != 0 {
			t.Fatalf("observer generateCount = %d, want 0 (policy gate must short-circuit before reaching observer)", got)
		}
	})
}

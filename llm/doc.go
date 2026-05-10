// Package llm owns the capability-aware LLM-provider contract for the
// agents framework.
//
// The contract is intentionally narrow — only the types an Agent or
// Tool implementation needs to call a model:
//
//   - ChatModel          base interface (Generate + Stream + Info)
//   - ToolCaller         capability: native function-calling
//     (WithTools is immutable; returns a new value)
//   - Embedder           capability: vector embeddings (does NOT embed
//     ChatModel — orthogonal to chat)
//   - StructuredOutputs  capability: JSON-schema-constrained output
//   - StreamReader       iterator-style streaming (Next + Close)
//   - StreamEvent        typed union (TextDelta / ToolCall* / Done)
//   - ProviderInfo       bound provider+model identity returned by Info()
//   - Capabilities       per-(provider × model) feature struct
//     (Tools / Embeddings / StructuredOutputs /
//     PromptCaching as bool fields; JSON-serializable
//     for OTel attribute emission)
//   - Tool / ToolCall    function-call schema + invocation
//   - Message            single conversation turn
//   - Request / Response chat-layer request/response (NEW in v0.3)
//   - Vector / Usage / UsageSource embeddings + token accounting
//   - FinishReason + 6 const  OpenAI-compatible stop reasons (shared
//     between LegacyClient and ChatModel surfaces)
//   - LegacyClient       v0.2 contract retained for source compatibility;
//     Deprecated, removal target v0.4.0
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

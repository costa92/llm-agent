// Package agentstest provides shared test helpers for code that
// interacts with [agents.Tool]. It is analogous in spirit to net/http's
// httptest: a small, focused set of fakes that other modules in the
// ecosystem can import from their *_test.go files instead of each
// re-inventing the same stub locally.
//
// The package is stdlib-only and imports only the parent `agents`
// package. Sister repos (llm-agent-rag, llm-agent-flow,
// llm-agent-customer-support, ...) may import it from test files
// without any new runtime dependencies.
//
// Two primitives are provided:
//
//   - [StubTool] / [NewStubTool] / [NewErrorTool] — agents.Tool
//     implementations with configurable Name/Description/Schema and
//     either a fixed-output or function-driven Execute.
//
//   - [RecordingTool] — decorator that wraps another agents.Tool and
//     records every Execute invocation (args, output, error) for later
//     assertion. Safe for concurrent use.
//
// Bridging to flow.Tool (the narrower interface in llm-agent-flow):
// wrap a StubTool with flow.FromAgentTool() at the call site — the
// agentstest package intentionally has no flow dependency to keep
// the dependency direction clean.
package agentstest

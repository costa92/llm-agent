// Package llm holds the LLM-provider contract used by the agents
// framework. It is the only package outside agents/* that an Agent or
// Tool implementation depends on at the type level.
//
// The contract is intentionally narrow:
//
//   - Client    one-shot Generate + token-streaming GenerateStream
//   - Tool      function-call schema (JSON Schema parameters)
//   - ToolCall  function-call invocation returned by the model
//   - Message   single conversation turn
//   - StreamChunk / StreamUsage  streaming primitives
//   - FinishReason + 6 const  OpenAI-compatible stop reasons
//
// Provider implementations (HTTP, Ollama, etc.) live in the parent
// AICS repo's pkg/llm package, which type-aliases everything here.
// External users plug their own provider by implementing Client and
// passing it to any Agent constructor.
package llm

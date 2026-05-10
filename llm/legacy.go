package llm

import "context"

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

// GenerateRequest is the v0.2 request type passed to LegacyClient.Generate.
//
// Deprecated: Use llm.Request instead. GenerateRequest will be removed in v0.4.0. See docs/migration-v0.2-to-v0.3.md.
type GenerateRequest struct {
	Prompt  string         `json:"prompt"`
	Context map[string]any `json:"context,omitempty"`
	// Tools enables function calling — providers that don't support
	// tools simply ignore the field.
	Tools []Tool `json:"tools,omitempty"`
	// History threads prior turns into the messages array so
	// multi-turn dialog is possible without forcing the caller to
	// hand-craft a single mega-prompt.
	History []Message `json:"history,omitempty"`
}

// GenerateResponse is the v0.2 response type returned by LegacyClient.Generate.
//
// Deprecated: Use llm.Response instead. GenerateResponse will be removed in v0.4.0. See docs/migration-v0.2-to-v0.3.md.
type GenerateResponse struct {
	Text         string         `json:"text"`
	FinishReason FinishReason   `json:"finish_reason,omitempty"`
	Provider     string         `json:"provider"`
	Model        string         `json:"model,omitempty"`
	UsageToken   int            `json:"usage_token,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	// ToolCalls are populated when the model decides to invoke one or
	// more registered Tools. Callers route them to executors and feed
	// results back via History on the next turn.
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// legacyFinishReason is the underlying string type for FinishReason. The
// public FinishReason name is declared in types.go as `type FinishReason = legacyFinishReason`
// so legacy callers and new code see the same type.
type legacyFinishReason string

// FinishReason constants mirror the OpenAI /v1/chat/completions stop_reason field
// so that providers that surface this can pass it through without conversion.
const (
	FinishReasonStop          FinishReason = "stop"
	FinishReasonLength        FinishReason = "length"
	FinishReasonContentFilter FinishReason = "content_filter"
	FinishReasonToolCalls     FinishReason = "tool_calls"
	FinishReasonFunctionCall  FinishReason = "function_call"
	FinishReasonUnknown       FinishReason = "unknown"
)

// StreamChunk is a v0.2 streaming primitive returned over <-chan StreamChunk.
//
// Deprecated: Use llm.StreamEvent instead. StreamChunk will be removed in v0.4.0. See docs/migration-v0.2-to-v0.3.md.
type StreamChunk struct {
	Text  string       `json:"text"`
	Done  bool         `json:"done"`
	Usage *StreamUsage `json:"usage,omitempty"`
	// ToolCall is set on stream chunks when the model emits a function
	// call delta. Done==true with ToolCall set marks the end of a tool
	// invocation; subsequent calls are a new chunk sequence.
	ToolCall *ToolCall `json:"tool_call,omitempty"`
}

// StreamUsage carries token counts for a streaming response.
//
// Deprecated: Use llm.Usage instead. StreamUsage will be removed in v0.4.0. See docs/migration-v0.2-to-v0.3.md.
type StreamUsage struct {
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

// Package llm owns the LLM-provider contract for the agents framework.
// It is intentionally narrow: only the types an Agent needs to call a
// model. Provider implementations (HTTP, Ollama, Volcano, MiniMax,
// mock) live outside this module — anything satisfying Client works.
package llm

import (
	"context"
	"encoding/json"
)

// Client is the portable seam between business code and LLM providers.
// Generate is one-shot; GenerateStream streams tokens over <-chan StreamChunk.
type Client interface {
	Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error)
	GenerateStream(ctx context.Context, req GenerateRequest) (<-chan StreamChunk, error)
}

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

// Message represents a single turn in a conversation. Role is one of
// "user" / "assistant" (provider-specific extras like "system" / "tool"
// land in Metadata-shaped extensions if a provider ever needs them).
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
	// ToolCalls are populated when the model decides to invoke one or
	// more registered Tools. Callers route them to executors and feed
	// results back via History on the next turn.
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// FinishReason mirrors the OpenAI /v1/chat/completions stop_reason field so
// that providers that surface this can pass it through without conversion.
type FinishReason string

const (
	FinishReasonStop          FinishReason = "stop"
	FinishReasonLength        FinishReason = "length"
	FinishReasonContentFilter FinishReason = "content_filter"
	FinishReasonToolCalls     FinishReason = "tool_calls"
	FinishReasonFunctionCall  FinishReason = "function_call"
	FinishReasonUnknown       FinishReason = "unknown"
)

type StreamChunk struct {
	Text  string       `json:"text"`
	Done  bool         `json:"done"`
	Usage *StreamUsage `json:"usage,omitempty"`
	// ToolCall is set on stream chunks when the model emits a function
	// call delta. Done==true with ToolCall set marks the end of a tool
	// invocation; subsequent calls are a new chunk sequence.
	ToolCall *ToolCall `json:"tool_call,omitempty"`
}

type StreamUsage struct {
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

// Tool declares a function the model may call. Parameters is a raw
// JSON Schema document — this package doesn't validate it (the
// upstream provider does) so callers can use whatever schema dialect
// their provider expects.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ToolCall is what the model returns when it decides to invoke a Tool.
// Arguments is raw JSON because the model fills it per the Tool's
// Parameters schema.
type ToolCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// Package scriptedllm provides a minimal, deterministic llm.Client used by
// the examples in github.com/costa92/llm-agent/examples. It returns
// pre-recorded responses in order so demos can run without an API key and
// produce reproducible output.
//
// Real applications should plug a production llm.Client (OpenAI-compatible,
// Ollama, Anthropic, etc.) at the same boundary.
package scriptedllm

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/costa92/llm-agent/llm"
)

// New returns an llm.Client that yields the given responses in order. When
// the script runs out, subsequent calls return ErrScriptExhausted.
func New(responses ...llm.GenerateResponse) llm.Client {
	return &client{responses: responses}
}

// Text is a convenience constructor for plain-text responses ending in
// FinishReasonStop.
func Text(s string) llm.GenerateResponse {
	return llm.GenerateResponse{Text: s, FinishReason: llm.FinishReasonStop}
}

// ToolCall builds a tool-call response (FinishReasonToolCalls) for the given
// tool name and JSON arguments string.
func ToolCall(name, argsJSON string) llm.GenerateResponse {
	return llm.GenerateResponse{
		FinishReason: llm.FinishReasonToolCalls,
		ToolCalls: []llm.ToolCall{{
			Name:      name,
			Arguments: json.RawMessage(argsJSON),
		}},
	}
}

// ErrScriptExhausted is returned when the script has no more responses.
var ErrScriptExhausted = errors.New("scriptedllm: no more responses")

type client struct {
	responses []llm.GenerateResponse
	cursor    int
}

func (c *client) Generate(_ context.Context, _ llm.GenerateRequest) (llm.GenerateResponse, error) {
	if c.cursor >= len(c.responses) {
		return llm.GenerateResponse{}, ErrScriptExhausted
	}
	resp := c.responses[c.cursor]
	c.cursor++
	return resp, nil
}

func (c *client) GenerateStream(_ context.Context, _ llm.GenerateRequest) (<-chan llm.StreamChunk, error) {
	ch := make(chan llm.StreamChunk)
	close(ch)
	return ch, nil
}

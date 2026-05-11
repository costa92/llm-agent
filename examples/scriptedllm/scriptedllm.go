// Package scriptedllm provides a minimal, deterministic llm.ChatModel used by
// the examples in github.com/costa92/llm-agent/examples. It returns
// pre-recorded responses in order so demos can run without an API key and
// produce reproducible output.
//
// Real applications should plug a production llm.ChatModel (OpenAI-compatible,
// Ollama, Anthropic, etc.) at the same boundary.
package scriptedllm

import (
	"encoding/json"

	"github.com/costa92/llm-agent/llm"
)

// New returns an llm.ChatModel that yields the given responses in order.
func New(responses ...llm.GenerateResponse) llm.ChatModel {
	out := make([]llm.Response, 0, len(responses))
	for _, resp := range responses {
		out = append(out, llm.Response{
			Text:         resp.Text,
			FinishReason: resp.FinishReason,
			Provider:     resp.Provider,
			Model:        resp.Model,
			Usage: llm.Usage{
				TotalTokens: resp.UsageToken,
				Source:      llm.UsageReported,
			},
			ToolCalls: resp.ToolCalls,
		})
	}
	return llm.NewScriptedLLM(llm.WithResponses(out...))
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

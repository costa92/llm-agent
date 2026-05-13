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
func New(responses ...llm.Response) llm.ChatModel {
	return llm.NewScriptedLLM(llm.WithResponses(responses...))
}

// Text is a convenience constructor for plain-text responses ending in
// FinishReasonStop.
func Text(s string) llm.Response {
	return llm.Response{
		Text:         s,
		FinishReason: llm.FinishReasonStop,
		Provider:     "scripted",
		Usage:        llm.Usage{Source: llm.UsageReported},
	}
}

// ToolCall builds a tool-call response (FinishReasonToolCalls) for the given
// tool name and JSON arguments string.
func ToolCall(name, argsJSON string) llm.Response {
	return llm.Response{
		FinishReason: llm.FinishReasonToolCalls,
		Provider:     "scripted",
		Usage:        llm.Usage{Source: llm.UsageReported},
		ToolCalls: []llm.ToolCall{{
			Name:      name,
			Arguments: json.RawMessage(argsJSON),
		}},
	}
}

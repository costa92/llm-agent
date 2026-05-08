package agents_test

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/costa92/llm-agent"
	"github.com/costa92/llm-agent/builtin"
	"github.com/costa92/llm-agent/llm"
)

// toolCallResp builds a GenerateResponse that instructs the agent to call the
// named tool with the given JSON arguments. Used only in example_tool_use_test.go
// to keep Output deterministic without a real LLM.
func toolCallResp(toolName string, argsJSON string) llm.GenerateResponse {
	return llm.GenerateResponse{
		FinishReason: llm.FinishReasonToolCalls,
		Provider:     "scripted",
		ToolCalls: []llm.ToolCall{
			{Name: toolName, Arguments: json.RawMessage(argsJSON)},
		},
	}
}

// toolUseScriptedLLM is a minimal llm.Client stub for the tool-use example.
// It returns pre-set GenerateResponse values in order on each Generate call.
type toolUseScriptedLLM struct {
	calls int
	resps []llm.GenerateResponse
}

func (s *toolUseScriptedLLM) Generate(_ context.Context, _ llm.GenerateRequest) (llm.GenerateResponse, error) {
	if s.calls >= len(s.resps) {
		return llm.GenerateResponse{}, fmt.Errorf("scripted LLM: script exhausted")
	}
	r := s.resps[s.calls]
	s.calls++
	return r, nil
}

func (s *toolUseScriptedLLM) GenerateStream(_ context.Context, _ llm.GenerateRequest) (<-chan llm.StreamChunk, error) {
	return nil, fmt.Errorf("scripted LLM: streaming not supported")
}

// ExampleFunctionCallAgent demonstrates FunctionCallAgent using the tool subsystem.
// The agent receives a scripted LLM response containing a ToolCall for "calculator",
// looks up the tool in the Registry, executes it, and aggregates the result as Answer.
//
// In production, replace toolUseScriptedLLM with an OpenAI-compatible llm.Client that
// returns real ToolCalls when the model decides to invoke a registered tool.
func ExampleFunctionCallAgent() {
	// Register the built-in Calculator tool.
	reg := agents.NewRegistry(builtin.NewCalculator())

	// Scripted LLM: returns a ToolCall for "calculator" with expr "3*7".
	// FunctionCallAgent executes the tool and uses the output as its Answer.
	client := &toolUseScriptedLLM{
		resps: []llm.GenerateResponse{
			toolCallResp("calculator", `{"expr":"3*7"}`),
		},
	}

	agent := agents.NewFunctionCallAgent(client, agents.FunctionCallOptions{
		Name:     "math-agent",
		Registry: reg,
	})

	res, err := agent.Run(context.Background(), "What is 3 times 7?")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(res.Answer)
	// Output:
	// calculator: 21
}

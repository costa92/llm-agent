package agents_test

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/costa92/llm-agent"
	"github.com/costa92/llm-agent/builtin"
	"github.com/costa92/llm-agent/llm"
)

// toolCallResp builds an llm.Response that instructs the agent to call the
// named tool with the given JSON arguments. Used only in example_tool_use_test.go
// to keep Output deterministic without a real LLM.
func toolCallResp(toolName string, argsJSON string) llm.Response {
	return llm.Response{
		FinishReason: llm.FinishReasonToolCalls,
		Provider:     "scripted",
		Usage:        llm.Usage{Source: llm.UsageReported},
		ToolCalls: []llm.ToolCall{
			{Name: toolName, Arguments: json.RawMessage(argsJSON)},
		},
	}
}

// toolUseScriptedLLM is a minimal llm.ChatModel + llm.ToolCaller stub for the
// tool-use example. It returns pre-set responses in order on each Generate call.
type toolUseScriptedLLM struct {
	calls int
	resps []llm.Response
}

func (s *toolUseScriptedLLM) Generate(_ context.Context, _ llm.Request) (llm.Response, error) {
	if s.calls >= len(s.resps) {
		return llm.Response{}, fmt.Errorf("scripted LLM: script exhausted")
	}
	r := s.resps[s.calls]
	s.calls++
	return r, nil
}

func (s *toolUseScriptedLLM) Stream(ctx context.Context, req llm.Request) (llm.StreamReader, error) {
	resp, err := s.Generate(ctx, req)
	if err != nil {
		return nil, err
	}
	return llm.NewScriptedLLM(llm.WithResponses(resp)).Stream(ctx, llm.Request{})
}

func (s *toolUseScriptedLLM) Info() llm.ProviderInfo {
	return llm.ProviderInfo{
		Provider: "scripted",
		Model:    "example",
		Capabilities: llm.Capabilities{
			Tools: true,
		},
	}
}

func (s *toolUseScriptedLLM) WithTools(_ []llm.Tool) (llm.ToolCaller, error) {
	return s, nil
}

// ExampleFunctionCallAgent demonstrates FunctionCallAgent using the tool subsystem.
// The agent receives a scripted LLM response containing a ToolCall for "calculator",
// looks up the tool in the Registry, executes it, and aggregates the result as Answer.
//
// In production, replace toolUseScriptedLLM with an OpenAI-compatible llm.ChatModel that
// returns real ToolCalls when the model decides to invoke a registered tool.
func ExampleFunctionCallAgent() {
	// Register the built-in Calculator tool.
	reg := agents.NewRegistry(builtin.NewCalculator())

	// Scripted LLM: returns a ToolCall for "calculator" with expr "3*7".
	// FunctionCallAgent executes the tool and uses the output as its Answer.
	client := &toolUseScriptedLLM{
		resps: []llm.Response{
			{
				FinishReason: llm.FinishReasonToolCalls,
				Provider:     "scripted",
				ToolCalls: []llm.ToolCall{
					{Name: "calculator", Arguments: json.RawMessage(`{"expr":"3*7"}`)},
				},
			},
		},
	}

	agent, err := agents.NewFunctionCallAgent(client, agents.FunctionCallOptions{
		Name:     "math-agent",
		Registry: reg,
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	res, err := agent.Run(context.Background(), "What is 3 times 7?")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(res.Answer)
	// Output:
	// calculator: 21
}

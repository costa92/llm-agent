package agents

import (
	"context"
	"fmt"
)

// ExampleSimpleAgent demonstrates using SimpleAgent for a single-shot LLM forward pass.
// SimpleAgent wraps an llm.ChatModel and calls Generate once; the response text is
// returned as Result.Answer with no tool calls or multi-step reasoning.
//
// In production, replace newScriptedLLM with a real llm.ChatModel (e.g. OpenAI-compatible).
func ExampleSimpleAgent() {
	// Use a deterministic scripted LLM for testable output.
	// In production: plug your own llm.ChatModel implementation.
	client := newScriptedLLM(
		textResp("The capital of France is Paris."),
	)

	agent := NewSimpleAgent(client, SimpleOptions{
		Name:         "geography",
		SystemPrompt: "You are a helpful geography assistant.",
	})

	res, err := agent.Run(context.Background(), "What is the capital of France?")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(res.Answer)
	// Output:
	// The capital of France is Paris.
}

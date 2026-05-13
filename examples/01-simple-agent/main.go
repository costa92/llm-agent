// Demo 01: SimpleAgent — single-shot LLM forward pass.
//
// SimpleAgent is the smallest paradigm: one Generate call, response.Text
// becomes Result.Answer. Use it for translation, summarization, single-turn
// Q&A — anything where one prompt → one answer is enough.
//
// Run:
//
//	cd examples/01-simple-agent && go run .
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/costa92/llm-agent"
	"github.com/costa92/llm-agent/examples/scriptedllm"
)

func main() {
	// Plug a deterministic mock LLM so the demo runs offline.
	// In production, replace with an OpenAI-compatible / DeepSeek / Ollama / Anthropic / MiniMax client.
	client := scriptedllm.New(
		scriptedllm.Text("The capital of France is Paris."),
	)

	agent := agents.NewSimpleAgent(client, agents.SimpleOptions{
		Name:         "geography",
		SystemPrompt: "You are a helpful geography assistant. Answer in one sentence.",
	})

	res, err := agent.Run(context.Background(), "What is the capital of France?")
	if err != nil {
		log.Fatalf("agent run failed: %v", err)
	}

	fmt.Println("Q: What is the capital of France?")
	fmt.Println("A:", res.Answer)
	fmt.Printf("(trace steps: %d, llm calls: %d, tokens: %d)\n",
		len(res.Trace), res.Usage.LLMCalls, res.Usage.Tokens)
}

// Demo 02: FunctionCallAgent with tool registry.
//
// FunctionCallAgent uses the LLM's native tool-call API instead of prompt
// parsing. The agent advertises registered tools via llm.Tool, executes any
// ToolCalls returned by the LLM, and aggregates the result into Result.Answer.
//
// This demo registers the built-in Calculator and scripts the LLM to invoke
// it with `3*7`.
//
// Run:
//
//	cd examples/02-tool-use && go run .
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/costa92/llm-agent"
	"github.com/costa92/llm-agent/builtin"
	"github.com/costa92/llm-agent/examples/scriptedllm"
)

func main() {
	// Register tools the LLM can call.
	reg := agents.NewRegistry(builtin.NewCalculator())

	// Script the LLM to invoke calculator(3*7). In production the LLM
	// chooses tools and arguments on its own.
	client := scriptedllm.New(
		scriptedllm.ToolCall("calculator", `{"expr":"3*7"}`),
	)

	agent, err := agents.NewFunctionCallAgent(client, agents.FunctionCallOptions{
		Name:     "math-agent",
		Registry: reg,
	})
	if err != nil {
		log.Fatalf("agent construction failed: %v", err)
	}

	res, err := agent.Run(context.Background(), "What is 3 times 7?")
	if err != nil {
		log.Fatalf("agent run failed: %v", err)
	}

	names := make([]string, 0, len(reg.List()))
	for _, t := range reg.List() {
		names = append(names, t.Name())
	}

	fmt.Println("Q: What is 3 times 7?")
	fmt.Println("A:", res.Answer)
	fmt.Printf("(tools registered: %v)\n", names)
}

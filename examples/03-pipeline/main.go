// Demo 03: Pipeline — sequential multi-agent handoff.
//
// orchestrate.Pipeline threads each step's Result.Answer into the next step's
// input. Use it for fixed linear flows like research → summarize → answer or
// classify → enrich → reply.
//
// This demo uses 3 SimpleAgents wired into a 3-step pipeline; each agent has
// its own scripted LLM so the run is deterministic and offline.
//
// Run:
//
//	cd examples/03-pipeline && go run .
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/costa92/llm-agent"
	"github.com/costa92/llm-agent/examples/scriptedllm"
	"github.com/costa92/llm-agent/orchestrate"
)

func main() {
	research := agents.NewSimpleAgent(
		scriptedllm.New(scriptedllm.Text(
			"Three notable facts about the Eiffel Tower:\n"+
				"1. Built 1887-1889 by Gustave Eiffel for the World's Fair.\n"+
				"2. 330 metres tall, was the world's tallest until 1930.\n"+
				"3. Receives ~7 million visitors per year.",
		)),
		agents.SimpleOptions{Name: "researcher", SystemPrompt: "Surface 3 key facts."},
	)

	summarize := agents.NewSimpleAgent(
		scriptedllm.New(scriptedllm.Text(
			"Eiffel Tower: 330m iron lattice tower built 1887-1889 by Gustave Eiffel; once the world's tallest; ~7M visitors annually.",
		)),
		agents.SimpleOptions{Name: "summarizer", SystemPrompt: "Compress to one sentence."},
	)

	answer := agents.NewSimpleAgent(
		scriptedllm.New(scriptedllm.Text(
			"The Eiffel Tower is a 330-metre iron lattice tower in Paris, completed in 1889 by Gustave Eiffel for the World's Fair. It was the tallest structure in the world until 1930 and now welcomes about 7 million visitors a year.",
		)),
		agents.SimpleOptions{Name: "answerer", SystemPrompt: "Reply in friendly travel-guide voice."},
	)

	pipeline := orchestrate.NewPipeline("eiffel-tour",
		orchestrate.Step{Name: "research", Agent: research},
		orchestrate.Step{Name: "summarize", Agent: summarize},
		orchestrate.Step{Name: "answer", Agent: answer},
	)

	res, err := pipeline.Run(context.Background(), "Tell me about the Eiffel Tower.")
	if err != nil {
		log.Fatalf("pipeline failed: %v", err)
	}

	fmt.Println("=== final answer ===")
	fmt.Println(res.FinalAnswer)
	fmt.Println()
	fmt.Println("=== per-step trail ===")
	for _, sr := range res.StepResults {
		fmt.Printf("- %-10s -> %.80q...\n", sr.Step, sr.Result.Answer)
	}
}

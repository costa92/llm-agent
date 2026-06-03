// Demo 09: Ollama — wiring a REAL provider into an agent.
//
// Every other demo plugs the deterministic scriptedllm mock so it runs
// offline. This one swaps in a live llm-agent-providers/ollama client to
// show the payoff of the llm.ChatModel seam: the SAME SimpleAgent code from
// demo 01 now drives a local Ollama model, unchanged — only the constructor
// line differs.
//
// The model is NOT hardcoded: with OLLAMA_MODEL unset the demo asks the
// server which models are pulled and picks a chat-capable one, so `go run .`
// works against whatever you happen to have locally.
//
// Prerequisites (a local Ollama server with at least one chat model):
//
//	ollama serve            # start the server (default http://localhost:11434)
//	ollama pull llama3.2    # ...or any chat model you like
//
// Run:
//
//	cd examples/09-ollama && go run .
//
// Override host/model via env:
//
//	OLLAMA_HOST=http://localhost:11434 OLLAMA_MODEL=qwen2.5 go run .
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	agents "github.com/costa92/llm-agent"
	"github.com/costa92/llm-agent-contract/llm"
	"github.com/costa92/llm-agent-providers/ollama"
)

const defaultHost = "http://localhost:11434"

func main() {
	ctx := context.Background()
	host := normalizeHost(os.Getenv("OLLAMA_HOST"))

	// Resolve a model before constructing the client. This call also
	// doubles as the "is Ollama reachable?" probe, so its error message is
	// the actionable one.
	model, err := pickModel(ctx, host)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	client, err := ollama.New(
		ollama.WithBaseURL(host),
		ollama.WithModel(model),
		ollama.WithTimeout(60*time.Second),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to build ollama client:", err)
		os.Exit(1)
	}

	// *ollama.Ollama satisfies llm.ChatModel, so it drops straight into any
	// agent paradigm. This is the exact SimpleAgent wiring from demo 01.
	agent := agents.NewSimpleAgent(client, agents.SimpleOptions{
		Name:         "ollama-geography",
		SystemPrompt: "You are a helpful geography assistant. Answer in one sentence.",
	})

	const question = "What is the capital of France?"
	res, err := agent.Run(ctx, question)
	if err != nil {
		fmt.Fprintf(os.Stderr, "agent run failed with model %q: %v\n", model, err)
		os.Exit(1)
	}

	info := client.Info()
	fmt.Println("Q:", question)
	fmt.Println("A:", res.Answer)
	fmt.Printf("(provider: %s/%s, llm calls: %d, tokens: %d)\n\n",
		info.Provider, info.Model, res.Usage.LLMCalls, res.Usage.Tokens)

	// Bonus: raw token streaming straight off the ChatModel seam. The agent
	// paradigms above buffer the whole reply; here we print tokens as they
	// arrive — Ollama's strength for interactive UX.
	fmt.Println("Streaming a short story:")
	if err := streamStory(ctx, client); err != nil {
		fmt.Fprintln(os.Stderr, "\nstreaming failed:", err)
		os.Exit(1)
	}
}

// pickModel returns OLLAMA_MODEL when set, otherwise auto-selects the first
// chat-capable model the server reports. The error is actionable: it tells
// the user whether Ollama is unreachable or simply has no chat model.
func pickModel(ctx context.Context, host string) (string, error) {
	if m := os.Getenv("OLLAMA_MODEL"); m != "" {
		return m, nil
	}
	models, err := listModels(ctx, host)
	if err != nil {
		return "", fmt.Errorf("cannot reach Ollama at %s: %w\n"+
			"start it with `ollama serve`, then `ollama pull llama3.2` "+
			"(or set OLLAMA_MODEL)", host, err)
	}
	for _, m := range models {
		if m.chatCapable() {
			fmt.Printf("auto-selected model %q (set OLLAMA_MODEL to override)\n\n", m.Name)
			return m.Name, nil
		}
	}
	return "", fmt.Errorf("Ollama at %s has no chat-capable model — pull one with "+
		"`ollama pull llama3.2`, or set OLLAMA_MODEL", host)
}

// ollamaModel is one entry of the GET /api/tags catalog.
type ollamaModel struct {
	Name         string   `json:"name"`
	Capabilities []string `json:"capabilities"`
}

// chatCapable reports whether the model can drive a chat/generate request.
// Modern Ollama advertises a "completion" capability (embedding-only models
// expose "embedding" instead); older servers omit the field, so we fall
// back to a name heuristic that skips obvious embedders.
func (m ollamaModel) chatCapable() bool {
	for _, c := range m.Capabilities {
		if c == "completion" {
			return true
		}
	}
	return len(m.Capabilities) == 0 &&
		!strings.Contains(m.Name, "embed") &&
		!strings.Contains(m.Name, "m3e")
}

// listModels fetches the server's model catalog via GET /api/tags.
func listModels(ctx context.Context, host string) ([]ollamaModel, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, host+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET /api/tags returned %s", resp.Status)
	}
	var body struct {
		Models []ollamaModel `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	return body.Models, nil
}

// streamStory iterates the StreamReader and prints text deltas live.
// Next returns io.EOF at a clean end; Close must always run (Pitfall 3).
func streamStory(ctx context.Context, model llm.ChatModel) error {
	sr, err := model.Stream(ctx, llm.Request{
		Messages: []llm.Message{
			{Role: "user", Content: "Write a two-sentence story about a curious robot."},
		},
	})
	if err != nil {
		return err
	}
	defer sr.Close()

	for {
		ev, err := sr.Next()
		if err == io.EOF {
			fmt.Println()
			return nil
		}
		if err != nil {
			return err
		}
		if ev.Kind == llm.EventTextDelta {
			fmt.Print(ev.Text)
		}
	}
}

// normalizeHost mirrors ollama.New's host resolution so the catalog probe
// and the client target the same URL: default when empty, add a scheme when
// missing, drop any trailing slash.
func normalizeHost(h string) string {
	if h == "" {
		h = defaultHost
	}
	if !strings.Contains(h, "://") {
		h = "http://" + h
	}
	return strings.TrimRight(h, "/")
}

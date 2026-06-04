// Demo 10: Ollama + tools — native function-calling against a live model.
//
// This is demo 02 (FunctionCallAgent + Registry + builtin.Calculator) with a
// REAL Ollama backend instead of the scripted mock: the model itself decides
// to call the calculator tool, and the agent executes it and folds the result
// into the answer. The only change from demo 02 is the client constructor.
//
// Native tool-calling needs a model the Ollama PROVIDER wires tool parsing
// for. As of llm-agent-providers v0.3.0 that is the llama3.1, qwen2.5-coder,
// and qwen3-coder families — NOT every model the server reports as
// "tools"-capable. With OLLAMA_MODEL unset the demo asks the provider which
// of your pulled models qualifies and picks the first.
//
// Prerequisites:
//
//	ollama serve
//	ollama pull llama3.1     # ...or qwen2.5-coder / qwen3-coder
//
// Run:
//
//	cd examples/10-ollama-tools && go run .
//
// Override host/model via env:
//
//	OLLAMA_HOST=http://localhost:11434 OLLAMA_MODEL=qwen2.5-coder go run .
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	agents "github.com/costa92/llm-agent"
	"github.com/costa92/llm-agent-contract/llm"
	"github.com/costa92/llm-agent-providers/ollama"
	"github.com/costa92/llm-agent-builtin"
)

const defaultHost = "http://localhost:11434"

func main() {
	ctx := context.Background()
	host := normalizeHost(os.Getenv("OLLAMA_HOST"))

	model, err := pickToolModel(ctx, host)
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

	// Register the tools the model may call — same Registry as demo 02. The
	// only difference from the mock demo is the live client above.
	reg := agents.NewRegistry(builtin.NewCalculator())

	agent, err := agents.NewFunctionCallAgent(client, agents.FunctionCallOptions{
		Name:         "ollama-math",
		Registry:     reg,
		SystemPrompt: "Use the calculator tool for any arithmetic.",
	})
	if err != nil {
		// Construction fails when the bound model lacks tool support (the
		// provider wraps llm.ErrCapabilityNotSupported) — typically because
		// OLLAMA_MODEL forced an unsupported model.
		exitToolHint(model, err)
	}

	const question = "What is 347 times 29?"
	res, err := agent.Run(ctx, question)
	if err != nil {
		exitToolHint(model, err)
	}

	names := make([]string, 0, len(reg.List()))
	for _, t := range reg.List() {
		names = append(names, t.Name())
	}

	info := client.Info()
	fmt.Println("Q:", question)
	fmt.Println("A:", res.Answer)
	fmt.Printf("(provider: %s/%s, tools registered: %v, llm calls: %d, tokens: %d)\n",
		info.Provider, info.Model, names, res.Usage.LLMCalls, res.Usage.Tokens)
}

// pickToolModel returns OLLAMA_MODEL when set, otherwise auto-selects the
// first pulled model the provider can do tool-calling with.
func pickToolModel(ctx context.Context, host string) (string, error) {
	if m := os.Getenv("OLLAMA_MODEL"); m != "" {
		return m, nil
	}
	models, err := listModels(ctx, host)
	if err != nil {
		return "", fmt.Errorf("cannot reach Ollama at %s: %w\n"+
			"start it with `ollama serve`, then `ollama pull llama3.1` "+
			"(or set OLLAMA_MODEL)", host, err)
	}
	for _, m := range models {
		if providerSupportsTools(host, m.Name) {
			fmt.Printf("auto-selected tools-capable model %q (set OLLAMA_MODEL to override)\n\n", m.Name)
			return m.Name, nil
		}
	}
	return "", fmt.Errorf("none of the %d locally-pulled model(s) support tool-calling via the "+
		"ollama provider — pull one it recognizes with `ollama pull llama3.1` "+
		"(or qwen2.5-coder / qwen3-coder), or set OLLAMA_MODEL", len(models))
}

// providerSupportsTools asks the ollama provider whether it would wire native
// tool-calling for the given model. This is the AUTHORITATIVE signal the agent
// itself uses, and it can disagree with the server's /api/tags "tools" flag:
// the provider only attaches tools for the model families it ships parsers for
// (llama3.1, qwen2.5-coder, qwen3-coder). Constructing the client is local
// (no network) — Info().Capabilities.Tools is derived from the model name.
func providerSupportsTools(host, model string) bool {
	c, err := ollama.New(ollama.WithBaseURL(host), ollama.WithModel(model))
	if err != nil {
		return false
	}
	return c.Info().Capabilities.Tools
}

// exitToolHint prints an actionable message and exits. Capability errors get a
// tool-specific hint; anything else is surfaced verbatim.
func exitToolHint(model string, err error) {
	if errors.Is(err, llm.ErrCapabilityNotSupported) {
		fmt.Fprintf(os.Stderr, "model %q does not support native tool-calling via the ollama "+
			"provider — pick a model it recognizes (`ollama pull llama3.1`, qwen2.5-coder, "+
			"qwen3-coder) or set OLLAMA_MODEL.\n", model)
	} else {
		fmt.Fprintf(os.Stderr, "agent failed with model %q: %v\n", model, err)
	}
	os.Exit(1)
}

// ollamaModel is one entry of the GET /api/tags catalog.
type ollamaModel struct {
	Name string `json:"name"`
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

// normalizeHost mirrors ollama.New's host resolution so the catalog probe and
// the client target the same URL: default when empty, add a scheme when
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

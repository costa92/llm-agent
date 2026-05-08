package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Chain pipes Tools sequentially: each tool's string output becomes the
// next tool's args (wrapped as {"input": "..."}). Chain itself satisfies
// Tool, so it can be registered or nested.
type Chain struct {
	name  string
	tools []Tool
}

// NewChain constructs a Chain. The first tool receives the original args;
// subsequent tools receive {"input": <prev_output>}.
func NewChain(name string, tools ...Tool) *Chain {
	return &Chain{name: name, tools: tools}
}

// Name implements Tool.
func (c *Chain) Name() string { return c.name }

// Description implements Tool.
func (c *Chain) Description() string {
	if len(c.tools) == 0 {
		return "empty chain"
	}
	names := make([]string, len(c.tools))
	for i, t := range c.tools {
		names[i] = t.Name()
	}
	return fmt.Sprintf("chain of %d tools: %s", len(c.tools), strings.Join(names, " → "))
}

// Schema implements Tool — delegates to the first tool's schema.
func (c *Chain) Schema() json.RawMessage {
	if len(c.tools) == 0 {
		return json.RawMessage(`{"type":"object"}`)
	}
	return c.tools[0].Schema()
}

// Execute runs each tool in sequence, piping output → next args as
// {"input": <prev>}.
func (c *Chain) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if len(c.tools) == 0 {
		return "", errors.New("chain: no tools")
	}
	currentArgs := args
	var out string
	for i, t := range c.tools {
		var err error
		out, err = t.Execute(ctx, currentArgs)
		if err != nil {
			return "", fmt.Errorf("chain[%d] %s: %w", i, t.Name(), err)
		}
		// next iteration: wrap output as {"input": out}
		nextArgs, _ := json.Marshal(map[string]string{"input": out})
		currentArgs = nextArgs
	}
	return out, nil
}

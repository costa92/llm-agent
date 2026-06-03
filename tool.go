package agents

import (
	"context"
	"encoding/json"

	"github.com/costa92/llm-agent-contract/llm"
)

// Tool and ExecuteFunc moved to the leaf contract
// github.com/costa92/llm-agent-contract/agents and are re-exported via aliases.go.
// The AsLLMTool bridge, NewFuncTool constructor, and funcTool impl stay here.

// AsLLMTool translates an agents.Tool into llm.Tool so it can be passed
// to a tool-capable llm.ChatModel.
func AsLLMTool(t Tool) llm.Tool {
	return llm.Tool{
		Name:        t.Name(),
		Description: t.Description(),
		Parameters:  t.Schema(),
	}
}

// NewFuncTool wraps a plain function as a Tool — saves writing a struct
// with Name/Description/Schema/Execute methods when the tool is trivial.
//
//	tool := agents.NewFuncTool(
//	    "weather",
//	    "Get weather for a city",
//	    json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}}}`),
//	    func(ctx context.Context, args json.RawMessage) (string, error) {
//	        var p struct{ City string }
//	        json.Unmarshal(args, &p)
//	        return "sunny in " + p.City, nil
//	    },
//	)
func NewFuncTool(name, description string, schema json.RawMessage, fn ExecuteFunc) Tool {
	return &funcTool{name: name, description: description, schema: schema, fn: fn}
}

type funcTool struct {
	name        string
	description string
	schema      json.RawMessage
	fn          ExecuteFunc
}

func (t *funcTool) Name() string            { return t.name }
func (t *funcTool) Description() string     { return t.description }
func (t *funcTool) Schema() json.RawMessage { return t.schema }
func (t *funcTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	return t.fn(ctx, args)
}

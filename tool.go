package agents

import (
	"context"
	"encoding/json"

	"github.com/costa92/llm-agent-contract/llm"
)

// Tool is a capability unit an Agent may invoke.
//
// Description is shown to the LLM (it decides whether to call); Schema describes
// the parameters as raw JSON Schema (we don't validate it — upstream provider does);
// Execute does the work and returns a string suitable for either prompt-injection
// (ReActAgent's Observation) or aggregation (FunctionCallAgent's answer).
type Tool interface {
	Name() string
	Description() string
	Schema() json.RawMessage
	Execute(ctx context.Context, args json.RawMessage) (string, error)
}

// AsLLMTool translates an agents.Tool into llm.Tool so it can be passed
// to a tool-capable llm.ChatModel.
func AsLLMTool(t Tool) llm.Tool {
	return llm.Tool{
		Name:        t.Name(),
		Description: t.Description(),
		Parameters:  t.Schema(),
	}
}

// ExecuteFunc is the signature used when wrapping a plain function as a Tool.
type ExecuteFunc func(ctx context.Context, args json.RawMessage) (string, error)

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

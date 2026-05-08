package agents

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type fixedTool struct {
	name, desc string
	schema     json.RawMessage
}

func (f fixedTool) Name() string                                                 { return f.name }
func (f fixedTool) Description() string                                          { return f.desc }
func (f fixedTool) Schema() json.RawMessage                                      { return f.schema }
func (f fixedTool) Execute(_ context.Context, _ json.RawMessage) (string, error) { return "ok", nil }

func TestAsLLMTool_Translates(t *testing.T) {
	src := fixedTool{
		name:   "lookup",
		desc:   "lookup something",
		schema: json.RawMessage(`{"type":"object","properties":{"q":{"type":"string"}}}`),
	}
	got := AsLLMTool(src)
	if got.Name != "lookup" {
		t.Errorf("Name = %q, want lookup", got.Name)
	}
	if got.Description != "lookup something" {
		t.Errorf("Description = %q", got.Description)
	}
	if string(got.Parameters) != string(src.schema) {
		t.Errorf("Parameters not preserved")
	}
}

func TestNewFuncTool(t *testing.T) {
	called := false
	tool := NewFuncTool(
		"upper-fn",
		"convert to uppercase",
		json.RawMessage(`{"type":"object","properties":{"s":{"type":"string"}}}`),
		func(_ context.Context, args json.RawMessage) (string, error) {
			called = true
			var p struct{ S string }
			if err := json.Unmarshal(args, &p); err != nil {
				return "", err
			}
			return strings.ToUpper(p.S), nil
		},
	)
	if tool.Name() != "upper-fn" {
		t.Errorf("Name = %q", tool.Name())
	}
	if tool.Description() != "convert to uppercase" {
		t.Errorf("Description = %q", tool.Description())
	}
	out, err := tool.Execute(context.Background(), json.RawMessage(`{"s":"hello"}`))
	if err != nil || out != "HELLO" {
		t.Errorf("Execute = (%q, %v)", out, err)
	}
	if !called {
		t.Error("ExecuteFunc not invoked")
	}
}

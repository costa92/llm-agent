package agents

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// upperTool: takes {"input":"..."} and returns ALL CAPS.
type upperTool struct{}

func (upperTool) Name() string            { return "upper" }
func (upperTool) Description() string     { return "uppercase" }
func (upperTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (upperTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p struct{ Input string }
	_ = json.Unmarshal(args, &p)
	return strings.ToUpper(p.Input), nil
}

// reverseTool: takes {"input":"..."} and returns reversed.
type reverseTool struct{}

func (reverseTool) Name() string            { return "reverse" }
func (reverseTool) Description() string     { return "reverse" }
func (reverseTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (reverseTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p struct{ Input string }
	_ = json.Unmarshal(args, &p)
	r := []rune(p.Input)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r), nil
}

func TestChain_PipesOutputToNext(t *testing.T) {
	c := NewChain("up_then_rev", upperTool{}, reverseTool{})
	args, _ := json.Marshal(map[string]string{"input": "hello"})
	out, err := c.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if out != "OLLEH" {
		t.Errorf("out = %q, want OLLEH", out)
	}
}

func TestChain_SatisfiesTool(t *testing.T) {
	var _ Tool = NewChain("x", upperTool{})
}

func TestChain_EmptyChain(t *testing.T) {
	c := NewChain("empty")
	_, err := c.Execute(context.Background(), json.RawMessage(`{}`))
	if err == nil {
		t.Error("want err for empty chain")
	}
}

package rag

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/costa92/llm-agent/llm"
)

func TestRAGTool_AddTextAndSearch(t *testing.T) {
	r := New(Options{})
	tool := AsTool(r)
	ctx := context.Background()

	addOut, err := tool.Execute(ctx, []byte(`{"action":"add_text","text":"go modules manage dependencies"}`))
	if err != nil {
		t.Fatalf("add_text: %v", err)
	}
	var addRes struct {
		Count int      `json:"count"`
		IDs   []string `json:"ids"`
	}
	_ = json.Unmarshal([]byte(addOut), &addRes)
	if addRes.Count == 0 {
		t.Fatal("add_text returned 0 chunks")
	}

	searchOut, err := tool.Execute(ctx, []byte(`{"action":"search","query":"go modules","top_k":3}`))
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if !strings.Contains(searchOut, "go modules") {
		t.Errorf("search output missing content: %s", searchOut)
	}
}

func TestRAGTool_Ask(t *testing.T) {
	r := New(Options{LLM: newScripted(llm.Response{Text: "Modules manage Go dependencies."})})
	tool := AsTool(r)
	ctx := context.Background()
	_, _ = tool.Execute(ctx, []byte(`{"action":"add_text","text":"go modules ship with go.mod"}`))

	out, err := tool.Execute(ctx, []byte(`{"action":"ask","question":"what are go modules?"}`))
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if !strings.Contains(out, "Modules") {
		t.Errorf("answer missing: %s", out)
	}
}

func TestRAGTool_Remove(t *testing.T) {
	r := New(Options{})
	tool := AsTool(r)
	ctx := context.Background()
	addOut, _ := tool.Execute(ctx, []byte(`{"action":"add_text","text":"x"}`))
	var addRes struct {
		IDs []string `json:"ids"`
	}
	_ = json.Unmarshal([]byte(addOut), &addRes)
	id := addRes.IDs[0]

	_, err := tool.Execute(ctx, []byte(`{"action":"remove","id":"`+id+`"}`))
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if r.Stats().Count != 0 {
		t.Errorf("after remove, Count = %d, want 0", r.Stats().Count)
	}
}

func TestRAGTool_Stats(t *testing.T) {
	r := New(Options{})
	tool := AsTool(r)
	ctx := context.Background()
	_, _ = tool.Execute(ctx, []byte(`{"action":"add_text","text":"hello"}`))

	out, err := tool.Execute(ctx, []byte(`{"action":"stats"}`))
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if !strings.Contains(out, `"Count"`) {
		t.Errorf("stats output missing Count: %s", out)
	}
}

func TestRAGTool_BadActions(t *testing.T) {
	tool := AsTool(New(Options{}))
	if _, err := tool.Execute(context.Background(), []byte(`{"action":"explode"}`)); err == nil {
		t.Error("expected error for unknown action")
	}
	if _, err := tool.Execute(context.Background(), []byte(`{}`)); err == nil {
		t.Error("expected error for empty action")
	}
	if _, err := tool.Execute(context.Background(), []byte(`{"action":"add_text"}`)); err == nil {
		t.Error("expected error for add_text without text")
	}
	if _, err := tool.Execute(context.Background(), []byte(`{"action":"search"}`)); err == nil {
		t.Error("expected error for search without query")
	}
}

func TestRAGTool_SchemaIsValidJSON(t *testing.T) {
	tool := AsTool(New(Options{}))
	var v map[string]any
	if err := json.Unmarshal(tool.Schema(), &v); err != nil {
		t.Errorf("schema not valid JSON: %v", err)
	}
}

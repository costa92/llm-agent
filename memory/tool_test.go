package memory

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func newToolMgr(t *testing.T) *Manager {
	t.Helper()
	return newManager(t)
}

func TestTool_AddSearchRoundTrip(t *testing.T) {
	tool := AsTool(newToolMgr(t))
	ctx := context.Background()

	addOut, err := tool.Execute(ctx, []byte(`{
		"action":"add","kind":"working","content":"go modules history","importance":0.7
	}`))
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	var addRes struct{ ID string }
	_ = json.Unmarshal([]byte(addOut), &addRes)
	if addRes.ID == "" {
		t.Fatal("add returned no id")
	}

	searchOut, err := tool.Execute(ctx, []byte(`{
		"action":"search","kind":"working","query":"go modules","top_k":1
	}`))
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if !strings.Contains(searchOut, addRes.ID) {
		t.Errorf("search did not return added id: %s", searchOut)
	}
}

func TestTool_GetUpdateRemove(t *testing.T) {
	tool := AsTool(newToolMgr(t))
	ctx := context.Background()

	addOut, _ := tool.Execute(ctx, []byte(`{"action":"add","kind":"semantic","content":"original","tags":["t1"],"importance":0.5}`))
	var addRes struct{ ID string }
	_ = json.Unmarshal([]byte(addOut), &addRes)
	id := addRes.ID

	// Get
	getOut, err := tool.Execute(ctx, []byte(`{"action":"get","kind":"semantic","id":"`+id+`"}`))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.Contains(getOut, "original") {
		t.Errorf("get missing content: %s", getOut)
	}

	// Update
	_, err = tool.Execute(ctx, []byte(`{"action":"update","kind":"semantic","id":"`+id+`","content":"updated"}`))
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	getOut, _ = tool.Execute(ctx, []byte(`{"action":"get","kind":"semantic","id":"`+id+`"}`))
	if !strings.Contains(getOut, "updated") {
		t.Errorf("after update, content not changed: %s", getOut)
	}

	// Remove
	_, err = tool.Execute(ctx, []byte(`{"action":"remove","kind":"semantic","id":"`+id+`"}`))
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	_, err = tool.Execute(ctx, []byte(`{"action":"get","kind":"semantic","id":"`+id+`"}`))
	if err == nil {
		t.Error("get after remove should error")
	}
}

func TestTool_SearchAllWhenKindOmitted(t *testing.T) {
	tool := AsTool(newToolMgr(t))
	ctx := context.Background()
	_, _ = tool.Execute(ctx, []byte(`{"action":"add","kind":"working","content":"alpha","importance":0.5}`))
	_, _ = tool.Execute(ctx, []byte(`{"action":"add","kind":"episodic","content":"alpha event","importance":0.5}`))

	out, err := tool.Execute(ctx, []byte(`{"action":"search","query":"alpha","top_k":3}`))
	if err != nil {
		t.Fatalf("search-all: %v", err)
	}
	if !strings.Contains(out, "working") || !strings.Contains(out, "episodic") {
		t.Errorf("search-all output should mention both kinds: %s", out)
	}
}

func TestTool_ConsolidateAndStats(t *testing.T) {
	tool := AsTool(newToolMgr(t))
	ctx := context.Background()
	_, _ = tool.Execute(ctx, []byte(`{"action":"add","kind":"working","content":"important","importance":0.9}`))
	_, _ = tool.Execute(ctx, []byte(`{"action":"add","kind":"working","content":"trivial","importance":0.1}`))

	out, err := tool.Execute(ctx, []byte(`{"action":"consolidate","threshold":0.7}`))
	if err != nil {
		t.Fatalf("consolidate: %v", err)
	}
	if !strings.Contains(out, `"consolidated":1`) {
		t.Errorf("expected consolidated:1, got %s", out)
	}

	statsOut, err := tool.Execute(ctx, []byte(`{"action":"stats"}`))
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if !strings.Contains(statsOut, "working") || !strings.Contains(statsOut, "episodic") {
		t.Errorf("stats missing kinds: %s", statsOut)
	}
}

func TestTool_ForgetByImportance(t *testing.T) {
	tool := AsTool(newToolMgr(t))
	ctx := context.Background()
	_, _ = tool.Execute(ctx, []byte(`{"action":"add","kind":"episodic","content":"keep","importance":0.9}`))
	_, _ = tool.Execute(ctx, []byte(`{"action":"add","kind":"episodic","content":"toss","importance":0.1}`))

	out, err := tool.Execute(ctx, []byte(`{"action":"forget","kind":"episodic","strategy":"importance","threshold":0.5}`))
	if err != nil {
		t.Fatalf("forget: %v", err)
	}
	if !strings.Contains(out, `"forgot":1`) {
		t.Errorf("expected forgot:1, got %s", out)
	}
}

func TestTool_BadActionErrors(t *testing.T) {
	tool := AsTool(newToolMgr(t))
	if _, err := tool.Execute(context.Background(), []byte(`{"action":"explode"}`)); err == nil {
		t.Error("expected error for unknown action")
	}
	if _, err := tool.Execute(context.Background(), []byte(`{}`)); err == nil {
		t.Error("expected error for empty action")
	}
	if _, err := tool.Execute(context.Background(), []byte(`not json`)); err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestTool_AddRequiresContent(t *testing.T) {
	tool := AsTool(newToolMgr(t))
	if _, err := tool.Execute(context.Background(), []byte(`{"action":"add","kind":"working"}`)); err == nil {
		t.Error("expected error when content missing")
	}
}

func TestTool_SchemaIsValidJSON(t *testing.T) {
	tool := AsTool(newToolMgr(t))
	var v map[string]any
	if err := json.Unmarshal(tool.Schema(), &v); err != nil {
		t.Errorf("schema not valid JSON: %v", err)
	}
}

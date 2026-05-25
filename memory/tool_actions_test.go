package memory

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// --- list action ----------------------------------------------------------

func TestTool_ListAction_NoKindFansOut(t *testing.T) {
	tool := AsTool(newToolMgr(t))
	ctx := context.Background()
	if _, err := tool.Execute(ctx, []byte(`{"action":"add","kind":"working","content":"w","importance":0.5}`)); err != nil {
		t.Fatalf("add working: %v", err)
	}
	if _, err := tool.Execute(ctx, []byte(`{"action":"add","kind":"episodic","content":"e","importance":0.5}`)); err != nil {
		t.Fatalf("add episodic: %v", err)
	}
	if _, err := tool.Execute(ctx, []byte(`{"action":"add","kind":"semantic","content":"s","importance":0.5}`)); err != nil {
		t.Fatalf("add semantic: %v", err)
	}
	out, err := tool.Execute(ctx, []byte(`{"action":"list","page_size":10}`))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, k := range []string{"working", "episodic", "semantic"} {
		if !strings.Contains(out, k) {
			t.Errorf("list output missing kind %q: %s", k, out)
		}
	}
}

func TestTool_ListAction_PerKind(t *testing.T) {
	tool := AsTool(newToolMgr(t))
	ctx := context.Background()
	if _, err := tool.Execute(ctx, []byte(`{"action":"add","kind":"working","content":"w1","importance":0.5}`)); err != nil {
		t.Fatalf("add: %v", err)
	}
	out, err := tool.Execute(ctx, []byte(`{"action":"list","kind":"working","page_size":10}`))
	if err != nil {
		t.Fatalf("list working: %v", err)
	}
	var page ListPage
	if err := json.Unmarshal([]byte(out), &page); err != nil {
		t.Fatalf("unmarshal page: %v; raw=%s", err, out)
	}
	if len(page.Items) != 1 || page.Items[0].Content != "w1" {
		t.Errorf("page items = %+v, want [w1]", page.Items)
	}
}

func TestTool_ListAction_FilterByPinnedOnly(t *testing.T) {
	tool := AsTool(newToolMgr(t))
	ctx := context.Background()
	// Add a plain item.
	if _, err := tool.Execute(ctx, []byte(`{"action":"add","kind":"working","content":"plain","importance":0.5}`)); err != nil {
		t.Fatalf("add plain: %v", err)
	}
	// Add a pinned item via add + pin.
	addOut, _ := tool.Execute(ctx, []byte(`{"action":"add","kind":"working","content":"pinned","importance":0.5}`))
	var addRes struct{ ID string }
	_ = json.Unmarshal([]byte(addOut), &addRes)
	if _, err := tool.Execute(ctx, []byte(`{"action":"pin","kind":"working","id":"`+addRes.ID+`"}`)); err != nil {
		t.Fatalf("pin: %v", err)
	}

	out, err := tool.Execute(ctx, []byte(`{"action":"list","kind":"working","page_size":10,"filter":{"pinned_only":true}}`))
	if err != nil {
		t.Fatalf("list pinned: %v", err)
	}
	if !strings.Contains(out, "pinned") {
		t.Errorf("output missing pinned item: %s", out)
	}
	if strings.Contains(out, `"Content":"plain"`) || strings.Contains(out, `"content":"plain"`) {
		t.Errorf("plain item leaked: %s", out)
	}
}

func TestTool_ListAction_Pagination(t *testing.T) {
	tool := AsTool(newToolMgr(t))
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		_, _ = tool.Execute(ctx, []byte(`{"action":"add","kind":"working","content":"x","importance":0.5}`))
	}
	out, err := tool.Execute(ctx, []byte(`{"action":"list","kind":"working","page_size":1}`))
	if err != nil {
		t.Fatalf("list page1: %v", err)
	}
	var page ListPage
	if err := json.Unmarshal([]byte(out), &page); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("page1 items = %d, want 1", len(page.Items))
	}
	if page.NextCursor == "" {
		t.Fatal("expected NextCursor on page1")
	}
	// page 2 via cursor
	req := map[string]any{
		"action":    "list",
		"kind":      "working",
		"page_size": 1,
		"cursor":    page.NextCursor,
	}
	body, _ := json.Marshal(req)
	out2, err := tool.Execute(ctx, body)
	if err != nil {
		t.Fatalf("list page2: %v", err)
	}
	var page2 ListPage
	if err := json.Unmarshal([]byte(out2), &page2); err != nil {
		t.Fatalf("unmarshal page2: %v", err)
	}
	if len(page2.Items) != 1 {
		t.Errorf("page2 items = %d, want 1", len(page2.Items))
	}
	if page.Items[0].ID == page2.Items[0].ID {
		t.Error("cursor pagination returned the same id twice")
	}
}

// --- pin / unpin ----------------------------------------------------------

func TestTool_PinAction(t *testing.T) {
	mgr := newToolMgr(t)
	tool := AsTool(mgr)
	ctx := context.Background()
	addOut, _ := tool.Execute(ctx, []byte(`{"action":"add","kind":"working","content":"x","importance":0.5}`))
	var addRes struct{ ID string }
	_ = json.Unmarshal([]byte(addOut), &addRes)

	if _, err := tool.Execute(ctx, []byte(`{"action":"pin","kind":"working","id":"`+addRes.ID+`"}`)); err != nil {
		t.Fatalf("pin: %v", err)
	}
	got, err := mgr.Get(ctx, KindWorking, addRes.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !IsPinned(got) {
		t.Error("item should be pinned")
	}
}

func TestTool_UnpinAction(t *testing.T) {
	mgr := newToolMgr(t)
	tool := AsTool(mgr)
	ctx := context.Background()
	addOut, _ := tool.Execute(ctx, []byte(`{"action":"add","kind":"working","content":"x","importance":0.5}`))
	var addRes struct{ ID string }
	_ = json.Unmarshal([]byte(addOut), &addRes)
	_, _ = tool.Execute(ctx, []byte(`{"action":"pin","kind":"working","id":"`+addRes.ID+`"}`))

	if _, err := tool.Execute(ctx, []byte(`{"action":"unpin","kind":"working","id":"`+addRes.ID+`"}`)); err != nil {
		t.Fatalf("unpin: %v", err)
	}
	got, _ := mgr.Get(ctx, KindWorking, addRes.ID)
	if IsPinned(got) {
		t.Error("item should not be pinned after unpin")
	}
}

func TestTool_PinRequiresID(t *testing.T) {
	tool := AsTool(newToolMgr(t))
	if _, err := tool.Execute(context.Background(), []byte(`{"action":"pin","kind":"working"}`)); err == nil {
		t.Error("expected error when id missing")
	}
}

// --- disable / enable -----------------------------------------------------

func TestTool_DisableAction(t *testing.T) {
	mgr := newToolMgr(t)
	tool := AsTool(mgr)
	ctx := context.Background()
	addOut, _ := tool.Execute(ctx, []byte(`{"action":"add","kind":"working","content":"x","importance":0.5}`))
	var addRes struct{ ID string }
	_ = json.Unmarshal([]byte(addOut), &addRes)

	if _, err := tool.Execute(ctx, []byte(`{"action":"disable","kind":"working","id":"`+addRes.ID+`"}`)); err != nil {
		t.Fatalf("disable: %v", err)
	}
	got, _ := mgr.Get(ctx, KindWorking, addRes.ID)
	if !IsDisabled(got) {
		t.Error("item should be disabled")
	}
}

func TestTool_EnableAction(t *testing.T) {
	mgr := newToolMgr(t)
	tool := AsTool(mgr)
	ctx := context.Background()
	addOut, _ := tool.Execute(ctx, []byte(`{"action":"add","kind":"working","content":"x","importance":0.5}`))
	var addRes struct{ ID string }
	_ = json.Unmarshal([]byte(addOut), &addRes)
	_, _ = tool.Execute(ctx, []byte(`{"action":"disable","kind":"working","id":"`+addRes.ID+`"}`))

	if _, err := tool.Execute(ctx, []byte(`{"action":"enable","kind":"working","id":"`+addRes.ID+`"}`)); err != nil {
		t.Fatalf("enable: %v", err)
	}
	got, _ := mgr.Get(ctx, KindWorking, addRes.ID)
	if IsDisabled(got) {
		t.Error("item should not be disabled after enable")
	}
}

// --- schema -------------------------------------------------------------

func TestTool_SchemaIncludesNewActions(t *testing.T) {
	tool := AsTool(newToolMgr(t))
	schema := string(tool.Schema())
	for _, a := range []string{"list", "pin", "unpin", "disable", "enable"} {
		if !strings.Contains(schema, `"`+a+`"`) {
			t.Errorf("schema missing action %q: %s", a, schema)
		}
	}
	// new top-level fields
	for _, f := range []string{"filter", "page_size", "cursor", "cursors"} {
		if !strings.Contains(schema, `"`+f+`"`) {
			t.Errorf("schema missing field %q: %s", f, schema)
		}
	}
}

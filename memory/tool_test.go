package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// fakeManager is a minimal, map-backed implementation of the contract
// Manager interface. It exists solely to drive AsTool's action routing,
// argument parsing, and error paths — it is NOT a faithful re-implementation
// of the real engine's scoring or lifecycle semantics. The engine now lives
// in llm-agent-memory/v2; these tests must not depend on it.
type fakeManager struct {
	// items[kind][id] = item
	items map[Kind]map[string]MemoryItem
	seq   int
	// store gates export/import: nil ⇒ "no SnapshotStore configured".
	store map[string]map[Kind]Snapshot
}

func newFakeManager() *fakeManager {
	return &fakeManager{
		items: map[Kind]map[string]MemoryItem{
			KindWorking:  {},
			KindEpisodic: {},
			KindSemantic: {},
		},
	}
}

// withStore enables export/import by attaching an in-memory snapshot store.
func (f *fakeManager) withStore() *fakeManager {
	f.store = map[string]map[Kind]Snapshot{}
	return f
}

func (f *fakeManager) HasKind(kind Kind) bool {
	_, ok := f.items[kind]
	return ok
}

func (f *fakeManager) Add(ctx context.Context, kind Kind, item MemoryItem) (string, error) {
	bucket, ok := f.items[kind]
	if !ok {
		return "", ErrKindDisabled
	}
	f.seq++
	id := strconv.Itoa(f.seq)
	item.ID = id
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	bucket[id] = item
	return id, nil
}

func (f *fakeManager) Get(ctx context.Context, kind Kind, id string) (MemoryItem, error) {
	bucket, ok := f.items[kind]
	if !ok {
		return MemoryItem{}, ErrKindDisabled
	}
	it, ok := bucket[id]
	if !ok {
		return MemoryItem{}, ErrNotFound
	}
	return it, nil
}

func (f *fakeManager) Update(ctx context.Context, kind Kind, id string, fn func(*MemoryItem)) error {
	bucket, ok := f.items[kind]
	if !ok {
		return ErrKindDisabled
	}
	it, ok := bucket[id]
	if !ok {
		return ErrNotFound
	}
	fn(&it)
	bucket[id] = it
	return nil
}

func (f *fakeManager) Remove(ctx context.Context, kind Kind, id string) error {
	bucket, ok := f.items[kind]
	if !ok {
		return ErrKindDisabled
	}
	if _, ok := bucket[id]; !ok {
		return ErrNotFound
	}
	delete(bucket, id)
	return nil
}

// Search does a trivial substring match on Content; Score is constant.
func (f *fakeManager) Search(ctx context.Context, kind Kind, query string, topK int) ([]SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, ErrEmptyQuery
	}
	bucket, ok := f.items[kind]
	if !ok {
		return nil, ErrKindDisabled
	}
	var out []SearchResult
	for _, it := range bucket {
		if IsDisabled(it) {
			continue
		}
		if strings.Contains(it.Content, query) {
			out = append(out, SearchResult{Item: it, Score: 1})
		}
	}
	if topK > 0 && len(out) > topK {
		out = out[:topK]
	}
	return out, nil
}

func (f *fakeManager) StatsAll() map[Kind]Stats {
	out := make(map[Kind]Stats, len(f.items))
	for k, bucket := range f.items {
		out[k] = Stats{Count: len(bucket)}
	}
	return out
}

func (f *fakeManager) SearchAll(ctx context.Context, query string, topK int) (map[Kind][]SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, ErrEmptyQuery
	}
	out := make(map[Kind][]SearchResult, len(f.items))
	for k := range f.items {
		res, err := f.Search(ctx, k, query, topK)
		if err != nil {
			return nil, err
		}
		out[k] = res
	}
	return out, nil
}

func (f *fakeManager) ListAll(ctx context.Context, filter ListFilter, pageSize int, cursors map[Kind]string) (map[Kind]ListPage, error) {
	out := make(map[Kind]ListPage, len(f.items))
	for k := range f.items {
		mem, _ := f.Lookup(k)
		page, err := mem.(Lister).List(ctx, filter, pageSize, cursors[k])
		if err != nil {
			return nil, err
		}
		out[k] = page
	}
	return out, nil
}

// Consolidate promotes Working items at/above the threshold into Episodic.
func (f *fakeManager) Consolidate(ctx context.Context, opts ConsolidateOptions) (int, error) {
	thresh := opts.Threshold
	if thresh == 0 {
		thresh = 0.7
	}
	n := 0
	for id, it := range f.items[KindWorking] {
		if it.Importance >= thresh {
			delete(f.items[KindWorking], id)
			f.seq++
			newID := strconv.Itoa(f.seq)
			it.ID = newID
			f.items[KindEpisodic][newID] = it
			n++
		}
	}
	return n, nil
}

// Forget evicts items below the importance threshold (only strategy needed
// by the tool tests).
func (f *fakeManager) Forget(ctx context.Context, kind Kind, opts ForgetOptions) (int, error) {
	bucket, ok := f.items[kind]
	if !ok {
		return 0, ErrKindDisabled
	}
	n := 0
	for id, it := range bucket {
		if IsPinned(it) {
			continue
		}
		if opts.Strategy == ForgetByImportance && it.Importance < opts.Threshold {
			delete(bucket, id)
			n++
		}
	}
	return n, nil
}

func (f *fakeManager) ExportAll(ctx context.Context, persistKey string) (map[Kind]Snapshot, error) {
	if persistKey != "" && f.store == nil {
		return nil, ErrSnapshotStoreNotConfigured
	}
	out := make(map[Kind]Snapshot, len(f.items))
	for k, bucket := range f.items {
		snap := Snapshot{Version: SnapshotVersion, Kind: k}
		for _, it := range bucket {
			snap.Items = append(snap.Items, SnapshotItem{Item: it})
		}
		out[k] = snap
	}
	if persistKey != "" {
		f.store[persistKey] = out
	}
	return out, nil
}

func (f *fakeManager) ImportAll(ctx context.Context, snaps map[Kind]Snapshot, persistKey string, mode ImportMode) (map[Kind]ImportReport, error) {
	if snaps == nil {
		if f.store == nil {
			return nil, ErrSnapshotStoreNotConfigured
		}
		snaps = f.store[persistKey]
	}
	out := make(map[Kind]ImportReport, len(snaps))
	for k, snap := range snaps {
		bucket, ok := f.items[k]
		if !ok {
			continue
		}
		var rpt ImportReport
		for _, si := range snap.Items {
			if _, exists := bucket[si.Item.ID]; exists {
				if mode == ImportMerge {
					rpt.Skipped++
					continue
				}
				rpt.Replaced++
			} else {
				rpt.Loaded++
			}
			bucket[si.Item.ID] = si.Item
		}
		out[k] = rpt
	}
	return out, nil
}

func (f *fakeManager) Lookup(kind Kind) (Memory, error) {
	bucket, ok := f.items[kind]
	if !ok {
		return nil, ErrKindDisabled
	}
	return &fakeMemory{kind: kind, bucket: bucket, mgr: f}, nil
}

// fakeMemory is the per-kind Memory view returned by Lookup. It implements
// Lister so the tool's per-kind "list" action works.
type fakeMemory struct {
	kind   Kind
	bucket map[string]MemoryItem
	mgr    *fakeManager
}

func (m *fakeMemory) Type() Kind { return m.kind }
func (m *fakeMemory) Add(ctx context.Context, item MemoryItem) (string, error) {
	return m.mgr.Add(ctx, m.kind, item)
}
func (m *fakeMemory) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	return m.mgr.Search(ctx, m.kind, query, topK)
}
func (m *fakeMemory) Get(ctx context.Context, id string) (MemoryItem, error) {
	return m.mgr.Get(ctx, m.kind, id)
}
func (m *fakeMemory) Update(ctx context.Context, id string, fn func(*MemoryItem)) error {
	return m.mgr.Update(ctx, m.kind, id, fn)
}
func (m *fakeMemory) Remove(ctx context.Context, id string) error {
	return m.mgr.Remove(ctx, m.kind, id)
}
func (m *fakeMemory) Stats() Stats { return Stats{Count: len(m.bucket)} }

// List applies the subset of ListFilter the tool tests exercise
// (PinnedOnly, IncludeDisabled) and paginates by an integer offset cursor
// over a deterministic (CreatedAt DESC, ID ASC) order.
func (m *fakeMemory) List(ctx context.Context, filter ListFilter, pageSize int, cursor string) (ListPage, error) {
	items := make([]MemoryItem, 0, len(m.bucket))
	for _, it := range m.bucket {
		if !filter.IncludeDisabled && IsDisabled(it) {
			continue
		}
		if filter.PinnedOnly && !IsPinned(it) {
			continue
		}
		items = append(items, it)
	}
	sort.Slice(items, func(i, j int) bool {
		if !items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].CreatedAt.After(items[j].CreatedAt)
		}
		return items[i].ID < items[j].ID
	})

	start := 0
	if cursor != "" {
		n, err := strconv.Atoi(cursor)
		if err != nil {
			return ListPage{}, fmt.Errorf("memory: bad cursor %q", cursor)
		}
		start = n
	}
	if start > len(items) {
		start = len(items)
	}
	end := len(items)
	next := ""
	if pageSize > 0 && start+pageSize < len(items) {
		end = start + pageSize
		next = strconv.Itoa(end)
	} else if pageSize > 0 {
		end = start + pageSize
		if end > len(items) {
			end = len(items)
		}
	}
	return ListPage{Items: items[start:end], NextCursor: next}, nil
}

func newToolMgr(t *testing.T) Manager {
	t.Helper()
	return newFakeManager()
}

// --- core CRUD + search ----------------------------------------------------

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

// --- error paths -----------------------------------------------------------

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

func TestTool_ForgetRequiresKind(t *testing.T) {
	tool := AsTool(newToolMgr(t))
	if _, err := tool.Execute(context.Background(), []byte(`{"action":"forget","strategy":"importance"}`)); err == nil {
		t.Error("expected error when kind missing for forget")
	}
}

// --- schema ----------------------------------------------------------------

func TestTool_SchemaIsValidJSON(t *testing.T) {
	tool := AsTool(newToolMgr(t))
	var v map[string]any
	if err := json.Unmarshal(tool.Schema(), &v); err != nil {
		t.Errorf("schema not valid JSON: %v", err)
	}
}

func TestTool_SchemaIncludesActions(t *testing.T) {
	tool := AsTool(newToolMgr(t))
	schema := string(tool.Schema())
	for _, a := range []string{"add", "search", "get", "update", "remove", "consolidate", "forget", "stats", "list", "pin", "unpin", "disable", "enable", "export", "import"} {
		if !strings.Contains(schema, `"`+a+`"`) {
			t.Errorf("schema missing action %q: %s", a, schema)
		}
	}
	for _, f := range []string{"filter", "page_size", "cursor", "cursors", "snapshot_key", "import_mode"} {
		if !strings.Contains(schema, `"`+f+`"`) {
			t.Errorf("schema missing field %q: %s", f, schema)
		}
	}
}

// --- list action -----------------------------------------------------------

func TestTool_ListAction_NoKindFansOut(t *testing.T) {
	tool := AsTool(newToolMgr(t))
	ctx := context.Background()
	_, _ = tool.Execute(ctx, []byte(`{"action":"add","kind":"working","content":"w","importance":0.5}`))
	_, _ = tool.Execute(ctx, []byte(`{"action":"add","kind":"episodic","content":"e","importance":0.5}`))
	_, _ = tool.Execute(ctx, []byte(`{"action":"add","kind":"semantic","content":"s","importance":0.5}`))

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
	_, _ = tool.Execute(ctx, []byte(`{"action":"add","kind":"working","content":"w1","importance":0.5}`))

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
	_, _ = tool.Execute(ctx, []byte(`{"action":"add","kind":"working","content":"plain","importance":0.5}`))
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
		// distinct CreatedAt so ordering is deterministic
		_, _ = tool.Execute(ctx, []byte(`{"action":"add","kind":"working","content":"x","importance":0.5}`))
		time.Sleep(time.Millisecond)
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

// --- pin / unpin -----------------------------------------------------------

func TestTool_PinAction(t *testing.T) {
	mgr := newFakeManager()
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
	mgr := newFakeManager()
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

// --- disable / enable ------------------------------------------------------

func TestTool_DisableAction(t *testing.T) {
	mgr := newFakeManager()
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
	mgr := newFakeManager()
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

// --- export / import -------------------------------------------------------

func TestTool_ExportAction_NoStore(t *testing.T) {
	tool := AsTool(newFakeManager()) // no store
	_, err := tool.Execute(context.Background(), []byte(`{"action":"export","snapshot_key":"k1"}`))
	if err == nil {
		t.Error("expected error when snapshot_key set without SnapshotStore")
	}
}

func TestTool_ExportAction_Inline(t *testing.T) {
	tool := AsTool(newFakeManager())
	ctx := context.Background()
	_, _ = tool.Execute(ctx, []byte(`{"action":"add","kind":"working","content":"alpha","importance":0.5}`))

	out, err := tool.Execute(ctx, []byte(`{"action":"export"}`))
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if !strings.Contains(out, `"working"`) {
		t.Errorf("export output missing working snapshot: %s", out)
	}
	if !strings.Contains(out, `"alpha"`) {
		t.Errorf("export output missing item content: %s", out)
	}
}

func TestTool_ImportAction_DefaultMergeMode(t *testing.T) {
	// src exports to a shared store; dst imports via the tool (default merge).
	src := newFakeManager().withStore()
	ctx := context.Background()
	_, _ = src.Add(ctx, KindWorking, MemoryItem{Content: "alpha"})
	if _, err := src.ExportAll(ctx, "k1"); err != nil {
		t.Fatalf("src ExportAll: %v", err)
	}

	dst := newFakeManager().withStore()
	dst.store = src.store // share the snapshot store
	tool := AsTool(dst)
	out, err := tool.Execute(ctx, []byte(`{"action":"import","snapshot_key":"k1"}`))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if !strings.Contains(out, `"loaded":1`) {
		t.Errorf("expected loaded:1 in report, got %s", out)
	}
	if dst.StatsAll()[KindWorking].Count != 1 {
		t.Errorf("dst working count = %d, want 1", dst.StatsAll()[KindWorking].Count)
	}
}

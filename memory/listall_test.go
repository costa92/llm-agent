package memory

import (
	"context"
	"testing"
)

func TestManager_ListAllFansOut(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	if _, err := mgr.Add(ctx, KindWorking, MemoryItem{Content: "w", Importance: 0.5}); err != nil {
		t.Fatalf("Add working: %v", err)
	}
	if _, err := mgr.Add(ctx, KindEpisodic, MemoryItem{Content: "e", Importance: 0.5}); err != nil {
		t.Fatalf("Add episodic: %v", err)
	}
	if _, err := mgr.Add(ctx, KindSemantic, MemoryItem{Content: "s", Importance: 0.5}); err != nil {
		t.Fatalf("Add semantic: %v", err)
	}

	pages, err := mgr.ListAll(ctx, ListFilter{}, 10, nil)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	for _, kind := range []Kind{KindWorking, KindEpisodic, KindSemantic} {
		page, has := pages[kind]
		if !has {
			t.Errorf("missing kind %s", kind)
			continue
		}
		if len(page.Items) != 1 {
			t.Errorf("kind %s: got %d items, want 1", kind, len(page.Items))
		}
	}
}

func TestManager_ListAllSkipsDisabledKinds(t *testing.T) {
	mgr, _ := NewManager(ManagerOptions{Working: newWorking(t)}) // only working
	ctx := context.Background()
	_, _ = mgr.Add(ctx, KindWorking, MemoryItem{Content: "w", Importance: 0.5})

	pages, err := mgr.ListAll(ctx, ListFilter{}, 10, nil)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if _, has := pages[KindEpisodic]; has {
		t.Error("episodic should be absent (disabled)")
	}
	if _, has := pages[KindSemantic]; has {
		t.Error("semantic should be absent (disabled)")
	}
	if len(pages[KindWorking].Items) != 1 {
		t.Errorf("working items = %d, want 1", len(pages[KindWorking].Items))
	}
}

func TestManager_ListAllAppliesPerKindCursor(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	// Add 3 items into working; pageSize=1 so we need two pages.
	for i := 0; i < 3; i++ {
		_, _ = mgr.Add(ctx, KindWorking, MemoryItem{Content: "w", Importance: 0.5})
	}
	page1, err := mgr.ListAll(ctx, ListFilter{}, 1, nil)
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if len(page1[KindWorking].Items) != 1 {
		t.Fatalf("page1 working items = %d, want 1", len(page1[KindWorking].Items))
	}
	if page1[KindWorking].NextCursor == "" {
		t.Fatal("expected NextCursor on first page")
	}
	cursors := map[Kind]string{KindWorking: page1[KindWorking].NextCursor}
	page2, err := mgr.ListAll(ctx, ListFilter{}, 1, cursors)
	if err != nil {
		t.Fatalf("page2: %v", err)
	}
	if len(page2[KindWorking].Items) != 1 {
		t.Errorf("page2 working items = %d, want 1", len(page2[KindWorking].Items))
	}
	if page1[KindWorking].Items[0].ID == page2[KindWorking].Items[0].ID {
		t.Error("cursor pagination returned the same item twice")
	}
}

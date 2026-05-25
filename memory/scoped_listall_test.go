package memory

import (
	"context"
	"testing"
)

func TestScopedManager_ListAllAppliesCtxScope(t *testing.T) {
	sm := newScopedManager(t)
	aliceCtx := WithScope(context.Background(), Scope{User: "alice"})
	bobCtx := WithScope(context.Background(), Scope{User: "bob"})

	if _, err := sm.Add(aliceCtx, KindWorking, MemoryItem{Content: "alice w", Importance: 0.5}); err != nil {
		t.Fatalf("alice Add: %v", err)
	}
	if _, err := sm.Add(bobCtx, KindWorking, MemoryItem{Content: "bob w", Importance: 0.5}); err != nil {
		t.Fatalf("bob Add: %v", err)
	}

	pages, err := sm.ListAll(aliceCtx, ListFilter{}, 10, nil)
	if err != nil {
		t.Fatalf("alice ListAll: %v", err)
	}
	w := pages[KindWorking]
	if len(w.Items) != 1 {
		t.Fatalf("alice working items = %d, want 1", len(w.Items))
	}
	if got := readScope(w.Items[0]); got.User != "alice" {
		t.Errorf("alice item scope.user = %q, want alice", got.User)
	}
}

func TestScopedManager_ListAllCtxOverridesFilterScope(t *testing.T) {
	sm := newScopedManager(t)
	aliceCtx := WithScope(context.Background(), Scope{User: "alice"})
	bobCtx := WithScope(context.Background(), Scope{User: "bob"})

	if _, err := sm.Add(aliceCtx, KindWorking, MemoryItem{Content: "alice w", Importance: 0.5}); err != nil {
		t.Fatalf("alice Add: %v", err)
	}
	if _, err := sm.Add(bobCtx, KindWorking, MemoryItem{Content: "bob w", Importance: 0.5}); err != nil {
		t.Fatalf("bob Add: %v", err)
	}

	// Filter says bob, ctx says alice → alice wins.
	pages, err := sm.ListAll(aliceCtx, ListFilter{Scope: Scope{User: "bob"}}, 10, nil)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	w := pages[KindWorking]
	if len(w.Items) != 1 {
		t.Fatalf("items = %d, want 1 (alice's only)", len(w.Items))
	}
	if got := readScope(w.Items[0]); got.User != "alice" {
		t.Errorf("scope.user = %q, want alice (ctx overrides filter)", got.User)
	}
}

func TestScopedManager_ListAllZeroCtxScopeHonorsFilter(t *testing.T) {
	sm := newScopedManager(t)
	aliceCtx := WithScope(context.Background(), Scope{User: "alice"})
	bobCtx := WithScope(context.Background(), Scope{User: "bob"})

	if _, err := sm.Add(aliceCtx, KindWorking, MemoryItem{Content: "alice w", Importance: 0.5}); err != nil {
		t.Fatalf("alice Add: %v", err)
	}
	if _, err := sm.Add(bobCtx, KindWorking, MemoryItem{Content: "bob w", Importance: 0.5}); err != nil {
		t.Fatalf("bob Add: %v", err)
	}

	// Zero ctx scope; filter says bob → filter wins (returns bob's item).
	pages, err := sm.ListAll(context.Background(), ListFilter{Scope: Scope{User: "bob"}}, 10, nil)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	w := pages[KindWorking]
	if len(w.Items) != 1 {
		t.Fatalf("items = %d, want 1 (bob's only)", len(w.Items))
	}
	if got := readScope(w.Items[0]); got.User != "bob" {
		t.Errorf("scope.user = %q, want bob (filter honored when ctx is zero)", got.User)
	}
}

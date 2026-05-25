package memory

import (
	"context"
	"errors"
	"testing"
)

func newScopedManager(t *testing.T) *ScopedManager {
	t.Helper()
	sm, err := NewScopedManager(newManager(t))
	if err != nil {
		t.Fatalf("NewScopedManager: %v", err)
	}
	return sm
}

func TestScopedManager_NewRejectsNil(t *testing.T) {
	_, err := NewScopedManager(nil)
	if !errors.Is(err, ErrManagerRequired) {
		t.Errorf("err = %v, want ErrManagerRequired", err)
	}
}

func TestScopedManager_InnerExposesUnderlying(t *testing.T) {
	inner := newManager(t)
	sm, err := NewScopedManager(inner)
	if err != nil {
		t.Fatalf("NewScopedManager: %v", err)
	}
	if sm.Inner() != inner {
		t.Error("Inner() should return the wrapped *Manager")
	}
}

// Ensure the bare wrapper still pipes basic Add/Get through (no scope
// asserted yet) — this catches accidental signature drift.
func TestScopedManager_PassThroughWithZeroScope(t *testing.T) {
	sm := newScopedManager(t)
	ctx := context.Background()
	id, err := sm.Add(ctx, KindWorking, MemoryItem{Content: "hello", Importance: 0.5})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	got, err := sm.Get(ctx, KindWorking, id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Content != "hello" {
		t.Errorf("Content = %q, want hello", got.Content)
	}
}

func TestScopedManager_AddStampsScope(t *testing.T) {
	sm := newScopedManager(t)
	ctx := WithScope(context.Background(), Scope{User: "alice"})
	id, err := sm.Add(ctx, KindWorking, MemoryItem{Content: "alice fact", Importance: 0.5})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	// Read back through the raw Manager so this assertion does NOT depend
	// on Get-side scope filtering (added in a later commit).
	got, err := sm.Inner().Get(ctx, KindWorking, id)
	if err != nil {
		t.Fatalf("Inner.Get: %v", err)
	}
	raw, ok := got.Metadata[metaKeyScope]
	if !ok {
		t.Fatalf("Metadata[%q] not set", metaKeyScope)
	}
	m, ok := raw.(map[string]string)
	if !ok {
		t.Fatalf("Metadata[%q] type = %T, want map[string]string", metaKeyScope, raw)
	}
	if m["user"] != "alice" {
		t.Errorf("user = %q, want alice", m["user"])
	}
}

func TestScopedManager_AddZeroScopeNoStamp(t *testing.T) {
	sm := newScopedManager(t)
	id, err := sm.Add(context.Background(), KindWorking, MemoryItem{Content: "unscoped", Importance: 0.5})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	got, err := sm.Inner().Get(context.Background(), KindWorking, id)
	if err != nil {
		t.Fatalf("Inner.Get: %v", err)
	}
	if _, has := got.Metadata[metaKeyScope]; has {
		t.Errorf("zero-scope Add must not stamp Metadata[%q]", metaKeyScope)
	}
}

func TestScopedManager_GetEnforcesScope(t *testing.T) {
	sm := newScopedManager(t)
	aliceCtx := WithScope(context.Background(), Scope{User: "alice"})
	bobCtx := WithScope(context.Background(), Scope{User: "bob"})

	id, err := sm.Add(aliceCtx, KindWorking, MemoryItem{Content: "alice fact", Importance: 0.5})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	// alice can see her own item.
	if _, err := sm.Get(aliceCtx, KindWorking, id); err != nil {
		t.Errorf("alice Get: err = %v, want nil", err)
	}
	// bob cannot — returns ErrNotFound, not some leakier error.
	_, err = sm.Get(bobCtx, KindWorking, id)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("bob Get: err = %v, want ErrNotFound", err)
	}
}

func TestScopedManager_GetWildcardSeesAll(t *testing.T) {
	sm := newScopedManager(t)
	aliceCtx := WithScope(context.Background(), Scope{User: "alice"})
	id, err := sm.Add(aliceCtx, KindWorking, MemoryItem{Content: "alice fact", Importance: 0.5})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	// zero ctx scope = wildcard.
	got, err := sm.Get(context.Background(), KindWorking, id)
	if err != nil {
		t.Fatalf("wildcard Get: %v", err)
	}
	if got.Content != "alice fact" {
		t.Errorf("Content = %q", got.Content)
	}
}

func TestScopedManager_UpdateEnforcesScope(t *testing.T) {
	sm := newScopedManager(t)
	aliceCtx := WithScope(context.Background(), Scope{User: "alice"})
	bobCtx := WithScope(context.Background(), Scope{User: "bob"})

	id, err := sm.Add(aliceCtx, KindWorking, MemoryItem{Content: "alice fact", Importance: 0.5})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	err = sm.Update(bobCtx, KindWorking, id, func(it *MemoryItem) { it.Content = "hacked" })
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("cross-scope Update: err = %v, want ErrNotFound", err)
	}
	// confirm content unchanged via the raw inner Manager.
	got, _ := sm.Inner().Get(context.Background(), KindWorking, id)
	if got.Content != "alice fact" {
		t.Errorf("Content tampered: %q", got.Content)
	}
	// same-scope Update still works.
	if err := sm.Update(aliceCtx, KindWorking, id, func(it *MemoryItem) { it.Importance = 0.9 }); err != nil {
		t.Errorf("alice Update: %v", err)
	}
}

func TestScopedManager_SearchFilters(t *testing.T) {
	sm := newScopedManager(t)
	aliceCtx := WithScope(context.Background(), Scope{User: "alice"})
	bobCtx := WithScope(context.Background(), Scope{User: "bob"})

	if _, err := sm.Add(aliceCtx, KindWorking, MemoryItem{Content: "shared topic", Importance: 0.5}); err != nil {
		t.Fatalf("alice Add: %v", err)
	}
	if _, err := sm.Add(bobCtx, KindWorking, MemoryItem{Content: "shared topic", Importance: 0.5}); err != nil {
		t.Fatalf("bob Add: %v", err)
	}

	aliceRes, err := sm.Search(aliceCtx, KindWorking, "shared topic", 10)
	if err != nil {
		t.Fatalf("alice Search: %v", err)
	}
	if len(aliceRes) != 1 {
		t.Fatalf("alice got %d results, want 1", len(aliceRes))
	}
	if got := readScope(aliceRes[0].Item); got.User != "alice" {
		t.Errorf("alice result scope.user = %q, want alice", got.User)
	}

	bobRes, err := sm.Search(bobCtx, KindWorking, "shared topic", 10)
	if err != nil {
		t.Fatalf("bob Search: %v", err)
	}
	if len(bobRes) != 1 {
		t.Fatalf("bob got %d results, want 1", len(bobRes))
	}
	if got := readScope(bobRes[0].Item); got.User != "bob" {
		t.Errorf("bob result scope.user = %q, want bob", got.User)
	}
}

func TestScopedManager_SearchAllFilters(t *testing.T) {
	sm := newScopedManager(t)
	aliceCtx := WithScope(context.Background(), Scope{User: "alice"})
	bobCtx := WithScope(context.Background(), Scope{User: "bob"})

	for _, kind := range []Kind{KindWorking, KindEpisodic, KindSemantic} {
		if _, err := sm.Add(aliceCtx, kind, MemoryItem{Content: "topic alpha", Importance: 0.5, Tags: []string{"x"}}); err != nil {
			t.Fatalf("alice Add %s: %v", kind, err)
		}
		if _, err := sm.Add(bobCtx, kind, MemoryItem{Content: "topic alpha", Importance: 0.5, Tags: []string{"x"}}); err != nil {
			t.Fatalf("bob Add %s: %v", kind, err)
		}
	}

	out, err := sm.SearchAll(aliceCtx, "topic alpha", 10)
	if err != nil {
		t.Fatalf("alice SearchAll: %v", err)
	}
	totalAlice := 0
	for kind, res := range out {
		if len(res) != 1 {
			t.Errorf("kind %s: got %d results, want 1", kind, len(res))
		}
		for _, r := range res {
			if got := readScope(r.Item); got.User != "alice" {
				t.Errorf("kind %s: result scope.user = %q, want alice", kind, got.User)
			}
		}
		totalAlice += len(res)
	}
	if totalAlice != 3 {
		t.Errorf("alice SearchAll total = %d, want 3", totalAlice)
	}
}

// TestScopedManager_ConsolidateForgetStatsBypass locks in the v0.7
// known limitation: Consolidate / Forget / StatsAll operate on the
// entire inner Manager without scope filtering. When a future PR adds
// scope-aware variants, this test should be updated to reflect the new
// contract.
func TestScopedManager_ConsolidateForgetStatsBypass(t *testing.T) {
	sm := newScopedManager(t)
	aliceCtx := WithScope(context.Background(), Scope{User: "alice"})
	bobCtx := WithScope(context.Background(), Scope{User: "bob"})

	// alice: high-importance item
	if _, err := sm.Add(aliceCtx, KindWorking, MemoryItem{Content: "alice big", Importance: 0.9}); err != nil {
		t.Fatalf("alice Add: %v", err)
	}
	// bob: high-importance item too
	if _, err := sm.Add(bobCtx, KindWorking, MemoryItem{Content: "bob big", Importance: 0.9}); err != nil {
		t.Fatalf("bob Add: %v", err)
	}

	// Consolidate (called via the scoped manager, in alice's ctx) must
	// promote BOTH items — v0.7 limit: no scope filter on Consolidate.
	n, err := sm.Consolidate(aliceCtx, ConsolidateOptions{Threshold: 0.7})
	if err != nil {
		t.Fatalf("Consolidate: %v", err)
	}
	if n != 2 {
		t.Errorf("Consolidate promoted %d, want 2 (both alice and bob's items)", n)
	}

	// StatsAll must show counts for the whole Manager regardless of ctx
	// scope.
	stats := sm.StatsAll()
	if stats[KindWorking].Count != 2 {
		t.Errorf("StatsAll working Count = %d, want 2 (bypasses scope)", stats[KindWorking].Count)
	}
	if stats[KindEpisodic].Count != 2 {
		t.Errorf("StatsAll episodic Count = %d, want 2 (bypasses scope)", stats[KindEpisodic].Count)
	}

	// Forget by importance with threshold 0.5: nothing qualifies for
	// removal because both items are at 0.9. So we drop the threshold
	// past 0.9 — Forget must touch BOTH items regardless of ctx scope.
	removed, err := sm.Forget(aliceCtx, KindWorking, ForgetOptions{Strategy: ForgetByImportance, Threshold: 1.0})
	if err != nil {
		t.Fatalf("Forget: %v", err)
	}
	if removed != 2 {
		t.Errorf("Forget removed %d, want 2 (bypasses scope)", removed)
	}
}

func TestScopedManager_WildcardSearchSeesAll(t *testing.T) {
	sm := newScopedManager(t)
	aliceCtx := WithScope(context.Background(), Scope{User: "alice"})
	bobCtx := WithScope(context.Background(), Scope{User: "bob"})
	if _, err := sm.Add(aliceCtx, KindWorking, MemoryItem{Content: "shared topic", Importance: 0.5}); err != nil {
		t.Fatalf("alice Add: %v", err)
	}
	if _, err := sm.Add(bobCtx, KindWorking, MemoryItem{Content: "shared topic", Importance: 0.5}); err != nil {
		t.Fatalf("bob Add: %v", err)
	}
	// zero ctx scope = wildcard = sees both.
	res, err := sm.Search(context.Background(), KindWorking, "shared topic", 10)
	if err != nil {
		t.Fatalf("wildcard Search: %v", err)
	}
	if len(res) != 2 {
		t.Errorf("wildcard got %d results, want 2", len(res))
	}
}

func TestScopedManager_RemoveEnforcesScope(t *testing.T) {
	sm := newScopedManager(t)
	aliceCtx := WithScope(context.Background(), Scope{User: "alice"})
	bobCtx := WithScope(context.Background(), Scope{User: "bob"})

	id, err := sm.Add(aliceCtx, KindWorking, MemoryItem{Content: "alice fact", Importance: 0.5})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := sm.Remove(bobCtx, KindWorking, id); !errors.Is(err, ErrNotFound) {
		t.Errorf("cross-scope Remove: err = %v, want ErrNotFound", err)
	}
	// item still present.
	if _, err := sm.Inner().Get(context.Background(), KindWorking, id); err != nil {
		t.Errorf("item was incorrectly removed: %v", err)
	}
	// same-scope Remove succeeds.
	if err := sm.Remove(aliceCtx, KindWorking, id); err != nil {
		t.Errorf("alice Remove: %v", err)
	}
	if _, err := sm.Inner().Get(context.Background(), KindWorking, id); !errors.Is(err, ErrNotFound) {
		t.Errorf("post-remove Get: err = %v, want ErrNotFound", err)
	}
}

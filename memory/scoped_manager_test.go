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

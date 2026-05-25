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

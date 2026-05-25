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

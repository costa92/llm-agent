package memory

import (
	"context"
)

// ScopedManager wraps a *Manager and applies scope-based filtering on
// every operation. Scope is read from ctx via ScopeFrom; items are
// stamped with scope on Add and filtered on Get / Search / SearchAll /
// Update / Remove.
//
// LIMITATIONS (v0.7):
//   - Consolidate / Forget / StatsAll do NOT honor scope; they operate
//     on the entire underlying Manager. This is a deliberate v0.7 scope
//     limit because these operations bypass the Memory abstraction to
//     access the underlying scoredStore directly. Future work (v2) may
//     add scope-aware variants.
//
// All methods are pass-through to the inner Manager except for the
// filtering / stamping logic noted on each method. ScopedManager is
// goroutine-safe iff the inner *Manager is.
type ScopedManager struct {
	inner *Manager
}

// NewScopedManager wraps an existing *Manager. Returns ErrManagerRequired
// if inner is nil.
func NewScopedManager(inner *Manager) (*ScopedManager, error) {
	if inner == nil {
		return nil, ErrManagerRequired
	}
	return &ScopedManager{inner: inner}, nil
}

// Inner returns the underlying *Manager. Callers that need to invoke
// non-scope-aware operations (Consolidate / Forget / StatsAll) on the
// raw Manager can do so via this accessor — but those operations are
// also exposed directly on ScopedManager as pass-throughs.
func (sm *ScopedManager) Inner() *Manager { return sm.inner }

// Add forwards to the inner Manager. (Scope stamping is added in a
// subsequent commit.)
func (sm *ScopedManager) Add(ctx context.Context, kind Kind, item MemoryItem) (string, error) {
	return sm.inner.Add(ctx, kind, item)
}

// Get forwards to the inner Manager. (Scope enforcement is added in a
// subsequent commit.)
func (sm *ScopedManager) Get(ctx context.Context, kind Kind, id string) (MemoryItem, error) {
	return sm.inner.Get(ctx, kind, id)
}

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

// Add stamps the ctx scope into item.Metadata (under metaKeyScope) and
// forwards to the inner Manager. A zero-value ctx scope is a no-op —
// Metadata is left untouched so unscoped callers see no change.
func (sm *ScopedManager) Add(ctx context.Context, kind Kind, item MemoryItem) (string, error) {
	stampScope(&item, ScopeFrom(ctx))
	return sm.inner.Add(ctx, kind, item)
}

// Get fetches the item from the inner Manager then enforces the ctx
// scope. Returns ErrNotFound if the item exists but lives in a
// different scope — this avoids leaking cross-scope ID existence.
func (sm *ScopedManager) Get(ctx context.Context, kind Kind, id string) (MemoryItem, error) {
	it, err := sm.inner.Get(ctx, kind, id)
	if err != nil {
		return MemoryItem{}, err
	}
	if !ScopeFrom(ctx).Matches(readScope(it)) {
		return MemoryItem{}, ErrNotFound
	}
	return it, nil
}

// Update verifies the item's scope matches the ctx scope before
// mutating. Returns ErrNotFound on mismatch (same leak-avoidance
// rationale as Get).
func (sm *ScopedManager) Update(ctx context.Context, kind Kind, id string, fn func(*MemoryItem)) error {
	it, err := sm.inner.Get(ctx, kind, id)
	if err != nil {
		return err
	}
	if !ScopeFrom(ctx).Matches(readScope(it)) {
		return ErrNotFound
	}
	return sm.inner.Update(ctx, kind, id, fn)
}

// Remove verifies the item's scope matches the ctx scope before
// deleting. Returns ErrNotFound on mismatch.
func (sm *ScopedManager) Remove(ctx context.Context, kind Kind, id string) error {
	it, err := sm.inner.Get(ctx, kind, id)
	if err != nil {
		return err
	}
	if !ScopeFrom(ctx).Matches(readScope(it)) {
		return ErrNotFound
	}
	return sm.inner.Remove(ctx, kind, id)
}

// Search forwards to the inner Manager then drops results whose stored
// scope does not match the ctx scope. A zero ctx scope (wildcard)
// returns the inner results verbatim.
func (sm *ScopedManager) Search(ctx context.Context, kind Kind, query string, topK int) ([]SearchResult, error) {
	raw, err := sm.inner.Search(ctx, kind, query, topK)
	if err != nil {
		return nil, err
	}
	return filterByScope(raw, ScopeFrom(ctx)), nil
}

// SearchAll fans out to the inner Manager and applies per-result scope
// filtering on each kind.
func (sm *ScopedManager) SearchAll(ctx context.Context, query string, topK int) (map[Kind][]SearchResult, error) {
	raw, err := sm.inner.SearchAll(ctx, query, topK)
	if err != nil {
		return nil, err
	}
	s := ScopeFrom(ctx)
	out := make(map[Kind][]SearchResult, len(raw))
	for kind, results := range raw {
		out[kind] = filterByScope(results, s)
	}
	return out, nil
}

// Consolidate forwards to the inner Manager. NO scope filtering — see
// the type-level LIMITATIONS note. This passes items from every scope
// (including legacy unscoped data) through the consolidation rule.
func (sm *ScopedManager) Consolidate(ctx context.Context, opts ConsolidateOptions) (int, error) {
	return sm.inner.Consolidate(ctx, opts)
}

// Forget forwards to the inner Manager. NO scope filtering — see the
// type-level LIMITATIONS note.
func (sm *ScopedManager) Forget(ctx context.Context, kind Kind, opts ForgetOptions) (int, error) {
	return sm.inner.Forget(ctx, kind, opts)
}

// StatsAll forwards to the inner Manager. NO scope filtering — counts
// include items from every scope.
func (sm *ScopedManager) StatsAll() map[Kind]Stats {
	return sm.inner.StatsAll()
}

// filterByScope drops results whose stored scope does not Match s.
// A zero-value s short-circuits to the input slice (wildcard).
func filterByScope(results []SearchResult, s Scope) []SearchResult {
	if s.IsZero() {
		return results
	}
	out := make([]SearchResult, 0, len(results))
	for _, r := range results {
		if s.Matches(readScope(r.Item)) {
			out = append(out, r)
		}
	}
	return out
}

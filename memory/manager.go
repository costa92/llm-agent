package memory

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"
)

// Manager coordinates the 3 Memory types under one façade. nil entries
// are valid — callers can opt out of any kind by passing nil for that
// field on construction.
type Manager struct {
	working  *WorkingMemory
	episodic *EpisodicMemory
	semantic *SemanticMemory
}

// ManagerOptions allows the caller to specify which memories are
// active. Pass nil for any field to disable that kind. At least one
// memory must be non-nil; otherwise NewManager returns an error.
type ManagerOptions struct {
	Working  *WorkingMemory
	Episodic *EpisodicMemory
	Semantic *SemanticMemory
}

// NewManager constructs a Manager. Returns ErrNoMemories if all three
// fields are nil.
func NewManager(opts ManagerOptions) (*Manager, error) {
	if opts.Working == nil && opts.Episodic == nil && opts.Semantic == nil {
		return nil, ErrNoMemories
	}
	return &Manager{
		working:  opts.Working,
		episodic: opts.Episodic,
		semantic: opts.Semantic,
	}, nil
}

// Add routes an item to the named memory.
func (m *Manager) Add(ctx context.Context, kind Kind, item MemoryItem) (string, error) {
	mem, err := m.lookup(kind)
	if err != nil {
		return "", err
	}
	return mem.Add(ctx, item)
}

// Get fetches an item from the named memory.
func (m *Manager) Get(ctx context.Context, kind Kind, id string) (MemoryItem, error) {
	mem, err := m.lookup(kind)
	if err != nil {
		return MemoryItem{}, err
	}
	return mem.Get(ctx, id)
}

// Update mutates an item in the named memory.
func (m *Manager) Update(ctx context.Context, kind Kind, id string, fn func(*MemoryItem)) error {
	mem, err := m.lookup(kind)
	if err != nil {
		return err
	}
	return mem.Update(ctx, id, fn)
}

// Remove deletes an item from the named memory.
func (m *Manager) Remove(ctx context.Context, kind Kind, id string) error {
	mem, err := m.lookup(kind)
	if err != nil {
		return err
	}
	return mem.Remove(ctx, id)
}

// Search runs a search against one named memory.
func (m *Manager) Search(ctx context.Context, kind Kind, query string, topK int) ([]SearchResult, error) {
	mem, err := m.lookup(kind)
	if err != nil {
		return nil, err
	}
	return mem.Search(ctx, query, topK)
}

// SearchAll fans out the query to every active memory and returns the
// per-kind result lists. Per-kind topK is honored (not a global cap).
func (m *Manager) SearchAll(ctx context.Context, query string, topK int) (map[Kind][]SearchResult, error) {
	out := make(map[Kind][]SearchResult, 3)
	for _, kind := range []Kind{KindWorking, KindEpisodic, KindSemantic} {
		mem, err := m.lookup(kind)
		if errors.Is(err, ErrKindDisabled) {
			continue
		}
		if err != nil {
			return out, err
		}
		res, err := mem.Search(ctx, query, topK)
		if err != nil {
			return out, fmt.Errorf("memory: search %s: %w", kind, err)
		}
		out[kind] = res
	}
	return out, nil
}

// ListAll fans out List across active memory kinds. Returns per-kind
// page. pageSize applies PER kind (not as a global cap). cursors is a
// per-kind map: kinds with no entry start from the beginning. Kinds
// that are disabled on this Manager are omitted from the result map.
//
// If a kind's underlying Memory does not implement Lister (foreign
// implementations supplied by callers) it is silently skipped. The
// three bundled types all satisfy Lister.
func (m *Manager) ListAll(ctx context.Context, filter ListFilter, pageSize int, cursors map[Kind]string) (map[Kind]ListPage, error) {
	out := make(map[Kind]ListPage, 3)
	for _, kind := range []Kind{KindWorking, KindEpisodic, KindSemantic} {
		mem, err := m.lookup(kind)
		if errors.Is(err, ErrKindDisabled) {
			continue
		}
		if err != nil {
			return out, err
		}
		lister, ok := mem.(Lister)
		if !ok {
			continue
		}
		cursor := ""
		if cursors != nil {
			cursor = cursors[kind]
		}
		page, err := lister.List(ctx, filter, pageSize, cursor)
		if err != nil {
			return out, fmt.Errorf("memory: list %s: %w", kind, err)
		}
		out[kind] = page
	}
	return out, nil
}

// StatsAll returns Stats for every active memory.
func (m *Manager) StatsAll() map[Kind]Stats {
	out := make(map[Kind]Stats, 3)
	if m.working != nil {
		out[KindWorking] = m.working.Stats()
	}
	if m.episodic != nil {
		out[KindEpisodic] = m.episodic.Stats()
	}
	if m.semantic != nil {
		out[KindSemantic] = m.semantic.Stats()
	}
	return out
}

// --- Consolidate ----------------------------------------------------------

// ConsolidateOptions tunes Consolidate. Threshold is the minimum
// importance to promote (default 0.7). MinAge optionally requires
// items to have been around at least this long before promotion
// (default 0 = any age qualifies).
type ConsolidateOptions struct {
	Threshold float64
	MinAge    time.Duration
}

// Consolidate copies items from working memory whose Importance ≥
// Threshold to episodic memory. Source items are NOT deleted. Returns
// the count of items copied.
//
// Requires both Working and Episodic to be active; otherwise returns
// ErrConsolidateUnavailable.
func (m *Manager) Consolidate(ctx context.Context, opts ConsolidateOptions) (int, error) {
	if m.working == nil || m.episodic == nil {
		return 0, ErrConsolidateUnavailable
	}
	if opts.Threshold <= 0 {
		opts.Threshold = 0.7
	}
	items, _ := m.working.store.snapshot()
	now := time.Now()
	count := 0
	for _, it := range items {
		if it.Importance < opts.Threshold {
			continue
		}
		if opts.MinAge > 0 && now.Sub(it.CreatedAt) < opts.MinAge {
			continue
		}
		clone := it
		clone.ID = "" // let episodic re-generate
		if _, err := m.episodic.Add(ctx, clone); err != nil {
			return count, fmt.Errorf("memory: consolidate: %w", err)
		}
		count++
	}
	return count, nil
}

// --- Forget ---------------------------------------------------------------

// ForgetStrategy picks the rule used by Forget.
type ForgetStrategy string

const (
	ForgetByImportance ForgetStrategy = "importance"
	ForgetByAge        ForgetStrategy = "age"
	ForgetByCapacity   ForgetStrategy = "capacity"
)

// ForgetOptions tunes Forget. Threshold is the cutoff for the
// "importance" strategy; MaxAge for "age"; Keep is the cap retained
// after "capacity" eviction.
type ForgetOptions struct {
	Strategy  ForgetStrategy
	Threshold float64       // for importance
	MaxAge    time.Duration // for age
	Keep      int           // for capacity (number of items to KEEP)
}

// Forget applies the chosen strategy to one named memory and returns
// the count deleted.
func (m *Manager) Forget(ctx context.Context, kind Kind, opts ForgetOptions) (int, error) {
	mem, err := m.lookup(kind)
	if err != nil {
		return 0, err
	}
	store := storeOf(mem)
	if store == nil {
		return 0, fmt.Errorf("memory: forget unsupported on %s", kind)
	}
	items, _ := store.snapshot()

	switch opts.Strategy {
	case ForgetByImportance:
		threshold := opts.Threshold
		count := 0
		for id, it := range items {
			if it.Importance < threshold {
				if IsPinned(it) {
					continue
				}
				if err := mem.Remove(ctx, id); err == nil {
					count++
				}
			}
		}
		return count, nil

	case ForgetByAge:
		if opts.MaxAge <= 0 {
			return 0, fmt.Errorf("memory: forget by age requires MaxAge > 0")
		}
		now := time.Now()
		count := 0
		for id, it := range items {
			if now.Sub(it.CreatedAt) > opts.MaxAge {
				if IsPinned(it) {
					continue
				}
				if err := mem.Remove(ctx, id); err == nil {
					count++
				}
			}
		}
		return count, nil

	case ForgetByCapacity:
		if opts.Keep <= 0 {
			return 0, nil
		}
		// Sort by importance ascending; evict the lowest-importance items
		// first. Pinned items are excluded from the candidate list, so
		// they neither count toward the eviction budget nor get removed.
		type pair struct {
			id  string
			imp float64
		}
		all := make([]pair, 0, len(items))
		for id, it := range items {
			if IsPinned(it) {
				continue
			}
			all = append(all, pair{id, it.Importance})
		}
		if len(all) <= opts.Keep {
			return 0, nil
		}
		sort.Slice(all, func(i, j int) bool { return all[i].imp < all[j].imp })
		toEvict := len(all) - opts.Keep
		count := 0
		for i := 0; i < toEvict; i++ {
			if err := mem.Remove(ctx, all[i].id); err == nil {
				count++
			}
		}
		return count, nil

	default:
		return 0, fmt.Errorf("memory: unknown forget strategy %q", opts.Strategy)
	}
}

// --- internals ------------------------------------------------------------

func (m *Manager) lookup(kind Kind) (Memory, error) {
	switch kind {
	case KindWorking:
		if m.working == nil {
			return nil, ErrKindDisabled
		}
		return m.working, nil
	case KindEpisodic:
		if m.episodic == nil {
			return nil, ErrKindDisabled
		}
		return m.episodic, nil
	case KindSemantic:
		if m.semantic == nil {
			return nil, ErrKindDisabled
		}
		return m.semantic, nil
	default:
		return nil, fmt.Errorf("memory: unknown kind %q", kind)
	}
}

// storeOf returns the underlying scoredStore behind a Memory, or nil if
// the Memory implementation is foreign to this package. Used by Forget
// to cheaply enumerate items by importance/age without re-implementing
// per-type traversal.
func storeOf(mem Memory) *scoredStore {
	switch m := mem.(type) {
	case *WorkingMemory:
		return m.store
	case *EpisodicMemory:
		return m.store
	case *SemanticMemory:
		return m.store
	default:
		return nil
	}
}

// --- sentinel errors specific to Manager ----------------------------------

// ErrNoMemories is returned by NewManager when all 3 memory fields are nil.
var ErrNoMemories = errors.New("memory: manager requires at least one active memory")

// ErrKindDisabled is returned when an operation targets a memory kind
// that wasn't activated on this Manager.
var ErrKindDisabled = errors.New("memory: kind disabled on this manager")

// ErrConsolidateUnavailable is returned by Consolidate when either
// working or episodic is nil.
var ErrConsolidateUnavailable = errors.New("memory: consolidate requires both working and episodic memories")

// ErrManagerRequired is returned by NewScopedManager when the inner
// *Manager is nil.
var ErrManagerRequired = errors.New("memory: manager required")

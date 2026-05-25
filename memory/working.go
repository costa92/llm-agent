package memory

import (
	"context"
	"sort"
	"strings"
	"time"
)

// WorkingMemory holds the most recent / most active items. Capacity is
// enforced by evicting the lowest-scoring item on Add. Score formula
// per spec §6.3:
//
//	(vec×0.7 + keyword×0.3) × time_decay × (0.8 + importance×0.4)
//
// time_decay defaults to exp(-age/Decay) with Decay=24h.
type WorkingMemory struct {
	store *scoredStore
	opts  WorkingOptions
}

// WorkingOptions configures a WorkingMemory.
type WorkingOptions struct {
	Capacity int           // 0 → defaults to 50
	Decay    time.Duration // 0 → defaults to 24h

	// SavedBoost is a multiplicative score factor applied at Search time
	// to items where IsPinned(it) || GetSource(it) == SourceUserSaved.
	// Non-positive (including the zero value) is treated as 1.0
	// (no-op), preserving pre-v0.7 scoring behavior for callers that
	// leave it unset. Typical values: 1.5–3.0.
	SavedBoost float64
}

// NewWorking constructs a WorkingMemory. Returns ErrEmbedderRequired
// if e is nil.
func NewWorking(e Embedder, opts WorkingOptions) (*WorkingMemory, error) {
	if e == nil {
		return nil, ErrEmbedderRequired
	}
	if opts.Capacity <= 0 {
		opts.Capacity = 50
	}
	if opts.Decay <= 0 {
		opts.Decay = 24 * time.Hour
	}
	return &WorkingMemory{store: newScoredStore("wrk", e), opts: opts}, nil
}

// Type implements Memory.
func (w *WorkingMemory) Type() Kind { return KindWorking }

// Add inserts item. If at capacity, evicts the lowest-scoring item
// against the new item's content as the eviction probe.
func (w *WorkingMemory) Add(ctx context.Context, item MemoryItem) (string, error) {
	id, err := w.store.add(ctx, item)
	if err != nil {
		return "", err
	}
	w.evictIfOverCapacity(ctx, item.Content)
	return id, nil
}

// Search returns the topK items ranked by the working-memory composite.
func (w *WorkingMemory) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, ErrEmptyQuery
	}
	if topK <= 0 {
		topK = 5
	}
	qv, err := queryEmbedding(ctx, w.store.embedder, query)
	if err != nil {
		return nil, err
	}
	items, vecs := w.store.snapshot()
	out := make([]SearchResult, 0, len(items))
	for id, it := range items {
		if IsDisabled(it) {
			continue
		}
		score := w.score(query, qv, it, vecs[id])
		out = append(out, SearchResult{Item: it, Score: score})
	}
	sortDesc(out)
	if len(out) > topK {
		out = out[:topK]
	}
	return out, nil
}

// Get implements Memory.
func (w *WorkingMemory) Get(_ context.Context, id string) (MemoryItem, error) {
	return w.store.get(id)
}

// Update implements Memory.
func (w *WorkingMemory) Update(ctx context.Context, id string, fn func(*MemoryItem)) error {
	return w.store.update(ctx, id, fn)
}

// Remove implements Memory.
func (w *WorkingMemory) Remove(_ context.Context, id string) error {
	return w.store.remove(id)
}

// Stats implements Memory.
func (w *WorkingMemory) Stats() Stats {
	return w.store.stats(w.opts.Capacity)
}

// score is the working-memory composite per spec §6.3, with an
// optional SavedBoost factor applied for pinned / user-saved items
// (see savedBoostMultiplier).
func (w *WorkingMemory) score(query string, qv []float32, it MemoryItem, iv []float32) float64 {
	vec := vectorScore(qv, iv)
	kw := keywordScore(query, it.Content)
	decay := timeDecay(it.CreatedAt, w.opts.Decay)
	return (vec*0.7 + kw*0.3) * decay * importanceMultiplier(it.Importance) *
		savedBoostMultiplier(it, w.opts.SavedBoost)
}

// evictIfOverCapacity removes the lowest-scoring item against probe text.
// Cheap O(n); capacity is small by design.
func (w *WorkingMemory) evictIfOverCapacity(ctx context.Context, probe string) {
	if w.opts.Capacity <= 0 {
		return
	}
	w.store.mu.Lock()
	if len(w.store.items) <= w.opts.Capacity {
		w.store.mu.Unlock()
		return
	}
	w.store.mu.Unlock()

	qv, err := queryEmbedding(ctx, w.store.embedder, probe)
	if err != nil {
		// On embedder failure, fall back to evicting oldest.
		w.evictOldest()
		return
	}
	items, vecs := w.store.snapshot()
	type pair struct {
		id    string
		score float64
	}
	scores := make([]pair, 0, len(items))
	for id, it := range items {
		s := w.score(probe, qv, it, vecs[id])
		scores = append(scores, pair{id, s})
	}
	sort.Slice(scores, func(i, j int) bool { return scores[i].score < scores[j].score })
	_ = w.store.remove(scores[0].id)
}

func (w *WorkingMemory) evictOldest() {
	items, _ := w.store.snapshot()
	var (
		oldestID string
		oldest   = time.Now()
	)
	for id, it := range items {
		if it.CreatedAt.Before(oldest) {
			oldest = it.CreatedAt
			oldestID = id
		}
	}
	if oldestID != "" {
		_ = w.store.remove(oldestID)
	}
}

// sortDesc sorts results by Score descending (highest first).
func sortDesc(out []SearchResult) {
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
}

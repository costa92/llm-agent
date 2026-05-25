package memory

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// EpisodicMemory holds long-term events with NO capacity cap. Score
// formula per spec §6.3:
//
//	(vec×0.8 + recency×0.2) × (0.8 + importance×0.4)
//
// recency = exp(-age_days/RecencyHalfLifeDays); default RecencyHalfLifeDays=30.
type EpisodicMemory struct {
	store *scoredStore
	opts  EpisodicOptions
}

// EpisodicOptions configures an EpisodicMemory.
type EpisodicOptions struct {
	RecencyHalfLifeDays float64 // 0 → defaults to 30

	// SavedBoost is a multiplicative score factor applied at Search time
	// to items where IsPinned(it) || GetSource(it) == SourceUserSaved.
	// Non-positive (including the zero value) is treated as 1.0
	// (no-op), preserving pre-v0.7 scoring behavior for callers that
	// leave it unset.
	SavedBoost float64
}

// NewEpisodic constructs an EpisodicMemory.
func NewEpisodic(e Embedder, opts EpisodicOptions) (*EpisodicMemory, error) {
	if e == nil {
		return nil, ErrEmbedderRequired
	}
	if opts.RecencyHalfLifeDays <= 0 {
		opts.RecencyHalfLifeDays = 30
	}
	return &EpisodicMemory{store: newScoredStore("epi", e), opts: opts}, nil
}

func (m *EpisodicMemory) Type() Kind { return KindEpisodic }

func (m *EpisodicMemory) Add(ctx context.Context, item MemoryItem) (string, error) {
	return m.store.add(ctx, item)
}

func (m *EpisodicMemory) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, ErrEmptyQuery
	}
	if topK <= 0 {
		topK = 5
	}
	qv, err := queryEmbedding(ctx, m.store.embedder, query)
	if err != nil {
		return nil, err
	}
	items, vecs := m.store.snapshot()
	out := make([]SearchResult, 0, len(items))
	halfLife := time.Duration(m.opts.RecencyHalfLifeDays * 24 * float64(time.Hour))
	for id, it := range items {
		if IsDisabled(it) {
			continue
		}
		vec := vectorScore(qv, vecs[id])
		recency := timeDecay(it.CreatedAt, halfLife)
		score := (vec*0.8 + recency*0.2) * importanceMultiplier(it.Importance) *
			savedBoostMultiplier(it, m.opts.SavedBoost)
		out = append(out, SearchResult{Item: it, Score: score})
	}
	sortDesc(out)
	if len(out) > topK {
		out = out[:topK]
	}
	return out, nil
}

func (m *EpisodicMemory) Get(_ context.Context, id string) (MemoryItem, error) {
	return m.store.get(id)
}

func (m *EpisodicMemory) Update(ctx context.Context, id string, fn func(*MemoryItem)) error {
	return m.store.update(ctx, id, fn)
}

func (m *EpisodicMemory) Remove(_ context.Context, id string) error {
	return m.store.remove(id)
}

// Stats implements Memory. Capacity field is 0 (unlimited).
func (m *EpisodicMemory) Stats() Stats {
	return m.store.stats(0)
}

// List implements Lister. ctx is ignored (no I/O is performed).
func (m *EpisodicMemory) List(_ context.Context, filter ListFilter, pageSize int, cursor string) (ListPage, error) {
	return listFromStore(m.store, filter, pageSize, cursor)
}

// Export implements Exporter. Vectors are inlined so the receiver can reuse
// them without re-embedding.
func (m *EpisodicMemory) Export(_ context.Context) (Snapshot, error) {
	return exportFromStore(m.store, KindEpisodic), nil
}

// Import implements Importer. Returns ErrSnapshotKindMismatch when
// snap.Kind != KindEpisodic; ErrSnapshotVersionMismatch when the version is
// unknown.
func (m *EpisodicMemory) Import(_ context.Context, snap Snapshot, mode ImportMode) (ImportReport, error) {
	if snap.Kind != KindEpisodic {
		return ImportReport{}, fmt.Errorf("%w: got %s, want episodic", ErrSnapshotKindMismatch, snap.Kind)
	}
	return importIntoStore(m.store, snap, mode)
}

// RestoreEpisodic constructs an EpisodicMemory and immediately imports the
// given snapshot using ImportReplace mode. See RestoreWorking for the
// rationale on embedder reuse.
func RestoreEpisodic(e Embedder, snap Snapshot, opts EpisodicOptions) (*EpisodicMemory, error) {
	if e == nil {
		return nil, ErrEmbedderRequired
	}
	m, err := NewEpisodic(e, opts)
	if err != nil {
		return nil, err
	}
	if _, err := m.Import(context.Background(), snap, ImportReplace); err != nil {
		return nil, err
	}
	return m, nil
}

package memory

import (
	"context"
	"strings"
	"time"

	"github.com/costa92/llm-agent/rag"
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
}

// NewEpisodic constructs an EpisodicMemory.
func NewEpisodic(e rag.Embedder, opts EpisodicOptions) (*EpisodicMemory, error) {
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
		vec := vectorScore(qv, vecs[id])
		recency := timeDecay(it.CreatedAt, halfLife)
		score := (vec*0.8 + recency*0.2) * importanceMultiplier(it.Importance)
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

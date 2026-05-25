package memory

import (
	"context"
	"strings"
)

// SemanticMemory is K-V style with tag-aware ranking. Score formula
// per spec §6.3:
//
//	(vec×0.7 + tag_overlap×0.3) × (0.8 + importance×0.4)
//
// Search supports tag-pre-filter: prefix the query with "tag:foo,bar "
// to restrict to items containing any of those tags. Otherwise all
// items are considered.
type SemanticMemory struct {
	store *scoredStore
	opts  SemanticOptions
}

// SemanticOptions configures a SemanticMemory.
type SemanticOptions struct {
	// SavedBoost is a multiplicative score factor applied at Search time
	// to items where IsPinned(it) || GetSource(it) == SourceUserSaved.
	// Non-positive (including the zero value) is treated as 1.0
	// (no-op), preserving pre-v0.7 scoring behavior for callers that
	// leave it unset.
	SavedBoost float64
}

// NewSemantic constructs a SemanticMemory.
func NewSemantic(e Embedder, opts SemanticOptions) (*SemanticMemory, error) {
	if e == nil {
		return nil, ErrEmbedderRequired
	}
	return &SemanticMemory{store: newScoredStore("sem", e), opts: opts}, nil
}

func (m *SemanticMemory) Type() Kind { return KindSemantic }

func (m *SemanticMemory) Add(ctx context.Context, item MemoryItem) (string, error) {
	return m.store.add(ctx, item)
}

func (m *SemanticMemory) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, ErrEmptyQuery
	}
	if topK <= 0 {
		topK = 5
	}
	queryText, queryTags := parseTagPrefix(query)
	qv, err := queryEmbedding(ctx, m.store.embedder, queryText)
	if err != nil {
		return nil, err
	}
	items, vecs := m.store.snapshot()
	out := make([]SearchResult, 0, len(items))
	for id, it := range items {
		if IsDisabled(it) {
			continue
		}
		if len(queryTags) > 0 && !anyTagMatches(queryTags, it.Tags) {
			continue
		}
		vec := vectorScore(qv, vecs[id])
		tag := tagOverlap(queryTags, it.Tags)
		score := (vec*0.7 + tag*0.3) * importanceMultiplier(it.Importance) *
			savedBoostMultiplier(it, m.opts.SavedBoost)
		out = append(out, SearchResult{Item: it, Score: score})
	}
	sortDesc(out)
	if len(out) > topK {
		out = out[:topK]
	}
	return out, nil
}

func (m *SemanticMemory) Get(_ context.Context, id string) (MemoryItem, error) {
	return m.store.get(id)
}

func (m *SemanticMemory) Update(ctx context.Context, id string, fn func(*MemoryItem)) error {
	return m.store.update(ctx, id, fn)
}

func (m *SemanticMemory) Remove(_ context.Context, id string) error {
	return m.store.remove(id)
}

func (m *SemanticMemory) Stats() Stats {
	return m.store.stats(0)
}

// parseTagPrefix splits "tag:a,b real query" → ("real query", ["a","b"]).
// Returns (query, nil) if no prefix.
func parseTagPrefix(query string) (string, []string) {
	const prefix = "tag:"
	q := strings.TrimSpace(query)
	if !strings.HasPrefix(q, prefix) {
		return query, nil
	}
	rest := q[len(prefix):]
	sp := strings.IndexByte(rest, ' ')
	if sp < 0 {
		return "", splitCSV(rest)
	}
	tagsCSV := rest[:sp]
	body := strings.TrimSpace(rest[sp+1:])
	return body, splitCSV(tagsCSV)
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func anyTagMatches(want, have []string) bool {
	wantLower := make(map[string]bool, len(want))
	for _, t := range want {
		wantLower[strings.ToLower(t)] = true
	}
	for _, t := range have {
		if wantLower[strings.ToLower(t)] {
			return true
		}
	}
	return false
}

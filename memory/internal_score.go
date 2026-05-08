package memory

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/costa92/llm-agent/rag"
)

// scoredStore is the shared in-memory engine all 3 memory types share:
// a map[id]MemoryItem + a parallel map[id][]float32 of cached embeddings,
// plus a monotonic ID generator. Each Memory wraps it with type-specific
// scoring.
type scoredStore struct {
	mu       sync.Mutex
	items    map[string]MemoryItem
	vectors  map[string][]float32
	embedder rag.Embedder
	prefix   string
	seq      int
}

func newScoredStore(prefix string, e rag.Embedder) *scoredStore {
	return &scoredStore{
		items:    make(map[string]MemoryItem),
		vectors:  make(map[string][]float32),
		embedder: e,
		prefix:   prefix,
	}
}

// add inserts (generating ID if blank), embeds the content, returns ID.
func (s *scoredStore) add(ctx context.Context, item MemoryItem) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if item.ID == "" {
		s.seq++
		item.ID = fmt.Sprintf("%s_%d_%d", s.prefix, time.Now().UnixNano(), s.seq)
	}
	now := time.Now().UTC()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.AccessedAt = now
	if item.Importance < 0 {
		item.Importance = 0
	}
	if item.Importance > 1 {
		item.Importance = 1
	}
	vec, err := s.embedder.Embed(ctx, item.Content)
	if err != nil {
		return "", fmt.Errorf("memory: embed: %w", err)
	}
	s.items[item.ID] = item
	s.vectors[item.ID] = vec
	return item.ID, nil
}

func (s *scoredStore) get(id string) (MemoryItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	if !ok {
		return MemoryItem{}, ErrNotFound
	}
	item.AccessedAt = time.Now().UTC()
	s.items[id] = item
	return item, nil
}

// update applies fn under lock; if fn changes Content, the embedding
// is re-computed.
func (s *scoredStore) update(ctx context.Context, id string, fn func(*MemoryItem)) error {
	s.mu.Lock()
	item, ok := s.items[id]
	if !ok {
		s.mu.Unlock()
		return ErrNotFound
	}
	prevContent := item.Content
	fn(&item)
	item.AccessedAt = time.Now().UTC()
	if item.Importance < 0 {
		item.Importance = 0
	}
	if item.Importance > 1 {
		item.Importance = 1
	}
	contentChanged := item.Content != prevContent
	s.items[id] = item
	s.mu.Unlock()

	if contentChanged {
		vec, err := s.embedder.Embed(ctx, item.Content)
		if err != nil {
			return fmt.Errorf("memory: re-embed: %w", err)
		}
		s.mu.Lock()
		s.vectors[id] = vec
		s.mu.Unlock()
	}
	return nil
}

func (s *scoredStore) remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return ErrNotFound
	}
	delete(s.items, id)
	delete(s.vectors, id)
	return nil
}

// snapshot returns deep copies of the items + vectors for read-only
// operations like Search / Stats / Forget. Callers can iterate freely.
func (s *scoredStore) snapshot() (map[string]MemoryItem, map[string][]float32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	itemsCopy := make(map[string]MemoryItem, len(s.items))
	vecsCopy := make(map[string][]float32, len(s.vectors))
	for k, v := range s.items {
		itemsCopy[k] = v
	}
	for k, v := range s.vectors {
		cp := make([]float32, len(v))
		copy(cp, v)
		vecsCopy[k] = cp
	}
	return itemsCopy, vecsCopy
}

func (s *scoredStore) stats(capacity int) Stats {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := Stats{Count: len(s.items), Capacity: capacity}
	if len(s.items) == 0 {
		return out
	}
	var importanceSum float64
	oldest := time.Now()
	for _, it := range s.items {
		importanceSum += it.Importance
		if it.CreatedAt.Before(oldest) {
			oldest = it.CreatedAt
		}
	}
	out.AvgImportance = importanceSum / float64(len(s.items))
	out.OldestAge = time.Since(oldest)
	return out
}

// --- shared scoring helpers ------------------------------------------------

// keywordScore returns the fraction of distinct query tokens present in
// content (∈ [0,1]). Simple substring match — no TF-IDF.
func keywordScore(query, content string) float64 {
	q := strings.ToLower(query)
	c := strings.ToLower(content)
	tokens := splitTokens(q)
	if len(tokens) == 0 {
		return 0
	}
	hits := 0
	seen := map[string]bool{}
	for _, t := range tokens {
		if seen[t] {
			continue
		}
		seen[t] = true
		if strings.Contains(c, t) {
			hits++
		}
	}
	return float64(hits) / float64(len(seen))
}

// tagOverlap returns |query ∩ item| / |query|. 0 if query is empty.
func tagOverlap(queryTags, itemTags []string) float64 {
	if len(queryTags) == 0 {
		return 0
	}
	set := make(map[string]bool, len(itemTags))
	for _, t := range itemTags {
		set[strings.ToLower(t)] = true
	}
	hits := 0
	for _, t := range queryTags {
		if set[strings.ToLower(t)] {
			hits++
		}
	}
	return float64(hits) / float64(len(queryTags))
}

// importanceMultiplier maps importance ∈ [0,1] → [0.8, 1.2]. Boosts
// items with importance ≥ 0.5; lightly demotes items < 0.5.
func importanceMultiplier(imp float64) float64 {
	return 0.8 + imp*0.4
}

// timeDecay returns exp(-age/halfLife). At age=0 → 1; at age=halfLife → ~0.37.
func timeDecay(createdAt time.Time, halfLife time.Duration) float64 {
	age := time.Since(createdAt)
	if age <= 0 || halfLife <= 0 {
		return 1
	}
	return math.Exp(-float64(age) / float64(halfLife))
}

func splitTokens(s string) []string {
	out := make([]string, 0, len(s)/4)
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			cur.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return out
}

// queryEmbedding embeds query text once for use across one Search call.
func queryEmbedding(ctx context.Context, e rag.Embedder, query string) ([]float32, error) {
	return e.Embed(ctx, query)
}

// vectorScore returns cosine similarity, but 0 for any nil/missing vector.
func vectorScore(qv, iv []float32) float64 {
	if len(qv) == 0 || len(iv) == 0 {
		return 0
	}
	return rag.CosineSimilarity(qv, iv)
}

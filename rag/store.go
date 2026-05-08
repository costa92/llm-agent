package rag

import (
	"context"
	"errors"
	"sort"
	"sync"
)

// Document is the unit of vector storage. Vector is filled by the
// caller (RAGSystem) before Upsert.
type Document struct {
	ID       string
	Content  string
	Vector   []float32
	Metadata map[string]any
}

// SearchHit is one ranked result from a VectorStore.Search call.
// Score is cosine similarity in [0, 1].
type SearchHit struct {
	Doc   Document
	Score float64
}

// VectorStore is the abstract vector index. InMemoryStore is the
// shipped fallback; production usage swaps in pgvector / Qdrant /
// Milvus backends behind the same interface.
type VectorStore interface {
	Upsert(ctx context.Context, doc Document) error
	Search(ctx context.Context, query []float32, topK int) ([]SearchHit, error)
	Get(ctx context.Context, id string) (Document, error)
	Remove(ctx context.Context, id string) error
	Stats() StoreStats
}

// StoreStats summarizes a store's contents.
type StoreStats struct {
	Count int
	Dim   int
}

// ErrStoreNotFound is returned by Get / Remove when id is absent.
var ErrStoreNotFound = errors.New("rag: document not found")

// ErrDimMismatch is returned by Upsert/Search when the vector
// dimension differs from the store's configured Dim.
var ErrDimMismatch = errors.New("rag: vector dimension mismatch")

// InMemoryStore is a slice-backed brute-force store. O(N) per Search
// — fine for ≤ a few thousand chunks. Goroutine-safe via RWMutex.
type InMemoryStore struct {
	mu  sync.RWMutex
	dim int
	all map[string]Document
}

// NewInMemoryStore constructs an InMemoryStore for vectors of dimension dim.
func NewInMemoryStore(dim int) *InMemoryStore {
	if dim <= 0 {
		dim = 32
	}
	return &InMemoryStore{dim: dim, all: make(map[string]Document)}
}

// Upsert inserts or replaces the doc by ID. Returns ErrDimMismatch if
// the vector length doesn't match the configured dim.
func (s *InMemoryStore) Upsert(_ context.Context, doc Document) error {
	if len(doc.Vector) != s.dim {
		return ErrDimMismatch
	}
	if doc.ID == "" {
		return errors.New("rag: document ID is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.all[doc.ID] = doc
	return nil
}

// Search returns the topK most similar documents to query by cosine.
func (s *InMemoryStore) Search(_ context.Context, query []float32, topK int) ([]SearchHit, error) {
	if len(query) != s.dim {
		return nil, ErrDimMismatch
	}
	if topK <= 0 {
		topK = 5
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	hits := make([]SearchHit, 0, len(s.all))
	for _, d := range s.all {
		hits = append(hits, SearchHit{Doc: d, Score: CosineSimilarity(query, d.Vector)})
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	if len(hits) > topK {
		hits = hits[:topK]
	}
	return hits, nil
}

// Get returns one document by ID. ErrStoreNotFound when absent.
func (s *InMemoryStore) Get(_ context.Context, id string) (Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.all[id]
	if !ok {
		return Document{}, ErrStoreNotFound
	}
	return d, nil
}

// Remove deletes one document. ErrStoreNotFound when absent.
func (s *InMemoryStore) Remove(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.all[id]; !ok {
		return ErrStoreNotFound
	}
	delete(s.all, id)
	return nil
}

// Stats implements VectorStore.
func (s *InMemoryStore) Stats() StoreStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return StoreStats{Count: len(s.all), Dim: s.dim}
}

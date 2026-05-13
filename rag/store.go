package rag

import (
	"context"
	"errors"

	ragembed "github.com/costa92/llm-agent-rag/embed"
	ragstore "github.com/costa92/llm-agent-rag/store"
)

// Document is the unit of vector storage. Vector is filled by the caller before Upsert.
type Document struct {
	ID       string
	Content  string
	Vector   []float32
	Metadata map[string]any
}

// SearchHit is one ranked result from a VectorStore.Search call.
type SearchHit struct {
	Doc   Document
	Score float64
}

// VectorStore is the abstract vector index.
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

var ErrStoreNotFound = errors.New("rag: document not found")

var ErrDimMismatch = errors.New("rag: vector dimension mismatch")

// InMemoryStore is the compatibility wrapper over the standalone SDK store.
type InMemoryStore struct {
	inner *ragstore.InMemoryStore
}

// NewInMemoryStore constructs an InMemoryStore for vectors of dimension dim.
func NewInMemoryStore(dim int) *InMemoryStore {
	return &InMemoryStore{inner: ragstore.NewInMemoryStore(dim)}
}

// Upsert inserts or replaces the doc by ID.
func (s *InMemoryStore) Upsert(ctx context.Context, doc Document) error {
	if doc.ID == "" {
		return errors.New("rag: document ID is required")
	}
	err := s.inner.Upsert(ctx, []ragstore.StoredChunk{{
		ID:       doc.ID,
		Content:  doc.Content,
		Vector:   ragembed.Vector(doc.Vector),
		Metadata: doc.Metadata,
	}})
	return mapStoreErr(err)
}

// Search returns the topK most similar documents to query by cosine.
func (s *InMemoryStore) Search(ctx context.Context, query []float32, topK int) ([]SearchHit, error) {
	hits, err := s.inner.Search(ctx, ragstore.Query{
		Vector: ragembed.Vector(query),
		TopK:   topK,
	})
	if err != nil {
		return nil, mapStoreErr(err)
	}
	out := make([]SearchHit, 0, len(hits))
	for _, hit := range hits {
		out = append(out, SearchHit{
			Doc: Document{
				ID:       hit.Chunk.ID,
				Content:  hit.Chunk.Content,
				Vector:   []float32(hit.Chunk.Vector),
				Metadata: hit.Chunk.Metadata,
			},
			Score: hit.Score,
		})
	}
	return out, nil
}

// Get returns one document by ID.
func (s *InMemoryStore) Get(ctx context.Context, id string) (Document, error) {
	chunk, err := s.inner.Get(ctx, id)
	if err != nil {
		return Document{}, mapStoreErr(err)
	}
	return Document{
		ID:       chunk.ID,
		Content:  chunk.Content,
		Vector:   []float32(chunk.Vector),
		Metadata: chunk.Metadata,
	}, nil
}

// Remove deletes one document.
func (s *InMemoryStore) Remove(ctx context.Context, id string) error {
	return mapStoreErr(s.inner.Remove(ctx, id))
}

// Stats implements VectorStore.
func (s *InMemoryStore) Stats() StoreStats {
	stats, _ := s.inner.Stats(context.Background(), "")
	return StoreStats{Count: stats.Count, Dim: stats.Dim}
}

func mapStoreErr(err error) error {
	switch {
	case errors.Is(err, ragstore.ErrNotFound):
		return ErrStoreNotFound
	case errors.Is(err, ragstore.ErrDimensionMismatch):
		return ErrDimMismatch
	default:
		return err
	}
}

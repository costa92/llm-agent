package rag

import (
	"context"
	"errors"
	"testing"
)

func TestInMemoryStore_UpsertSearchRoundTrip(t *testing.T) {
	s := NewInMemoryStore(4)
	docs := []Document{
		{ID: "a", Content: "alpha", Vector: []float32{1, 0, 0, 0}},
		{ID: "b", Content: "beta", Vector: []float32{0, 1, 0, 0}},
		{ID: "c", Content: "gamma", Vector: []float32{0, 0, 1, 0}},
	}
	for _, d := range docs {
		if err := s.Upsert(context.Background(), d); err != nil {
			t.Fatalf("Upsert %s: %v", d.ID, err)
		}
	}
	if s.Stats().Count != 3 {
		t.Errorf("Count = %d, want 3", s.Stats().Count)
	}
	hits, err := s.Search(context.Background(), []float32{1, 0, 0, 0}, 2)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) != 2 {
		t.Fatalf("got %d hits, want 2", len(hits))
	}
	if hits[0].Doc.ID != "a" {
		t.Errorf("top hit = %q, want a", hits[0].Doc.ID)
	}
}

func TestInMemoryStore_DimMismatchErrors(t *testing.T) {
	s := NewInMemoryStore(4)
	err := s.Upsert(context.Background(), Document{ID: "x", Vector: []float32{1, 0}})
	if !errors.Is(err, ErrDimMismatch) {
		t.Errorf("Upsert err = %v, want ErrDimMismatch", err)
	}
	_, err = s.Search(context.Background(), []float32{1, 0}, 1)
	if !errors.Is(err, ErrDimMismatch) {
		t.Errorf("Search err = %v, want ErrDimMismatch", err)
	}
}

func TestInMemoryStore_RequireID(t *testing.T) {
	s := NewInMemoryStore(4)
	err := s.Upsert(context.Background(), Document{Vector: []float32{1, 0, 0, 0}})
	if err == nil {
		t.Error("expected error when ID is empty")
	}
}

func TestInMemoryStore_GetRemove(t *testing.T) {
	s := NewInMemoryStore(2)
	_ = s.Upsert(context.Background(), Document{ID: "x", Content: "X", Vector: []float32{1, 0}})
	got, err := s.Get(context.Background(), "x")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Content != "X" {
		t.Errorf("Content = %q", got.Content)
	}
	if err := s.Remove(context.Background(), "x"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := s.Get(context.Background(), "x"); !errors.Is(err, ErrStoreNotFound) {
		t.Errorf("Get after remove err = %v, want ErrStoreNotFound", err)
	}
}

func TestInMemoryStore_UpsertReplaces(t *testing.T) {
	s := NewInMemoryStore(2)
	_ = s.Upsert(context.Background(), Document{ID: "x", Content: "v1", Vector: []float32{1, 0}})
	_ = s.Upsert(context.Background(), Document{ID: "x", Content: "v2", Vector: []float32{0, 1}})
	got, _ := s.Get(context.Background(), "x")
	if got.Content != "v2" {
		t.Errorf("Content = %q, want v2", got.Content)
	}
	if s.Stats().Count != 1 {
		t.Errorf("Count = %d, want 1 (Upsert should replace, not append)", s.Stats().Count)
	}
}

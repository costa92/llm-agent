package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/costa92/llm-agent/rag"
)

func newWorking(t *testing.T) *WorkingMemory {
	t.Helper()
	w, err := NewWorking(rag.NewHashEmbedder(64), WorkingOptions{Capacity: 5, Decay: 24 * time.Hour})
	if err != nil {
		t.Fatalf("NewWorking: %v", err)
	}
	return w
}

func newEpisodic(t *testing.T) *EpisodicMemory {
	t.Helper()
	m, err := NewEpisodic(rag.NewHashEmbedder(64), EpisodicOptions{})
	if err != nil {
		t.Fatalf("NewEpisodic: %v", err)
	}
	return m
}

func newSemantic(t *testing.T) *SemanticMemory {
	t.Helper()
	m, err := NewSemantic(rag.NewHashEmbedder(64), SemanticOptions{})
	if err != nil {
		t.Fatalf("NewSemantic: %v", err)
	}
	return m
}

// --- shared interface conformance tests -----------------------------------

func TestMemory_AllTypesSatisfyInterface(t *testing.T) {
	var _ Memory = newWorking(t)
	var _ Memory = newEpisodic(t)
	var _ Memory = newSemantic(t)
}

func TestMemory_TypeReturnsCorrectKind(t *testing.T) {
	if k := newWorking(t).Type(); k != KindWorking {
		t.Errorf("working Type = %q, want %q", k, KindWorking)
	}
	if k := newEpisodic(t).Type(); k != KindEpisodic {
		t.Errorf("episodic Type = %q, want %q", k, KindEpisodic)
	}
	if k := newSemantic(t).Type(); k != KindSemantic {
		t.Errorf("semantic Type = %q, want %q", k, KindSemantic)
	}
}

func TestMemory_ConstructorRejectsNilEmbedder(t *testing.T) {
	if _, err := NewWorking(nil, WorkingOptions{}); !errors.Is(err, ErrEmbedderRequired) {
		t.Errorf("NewWorking(nil) err = %v, want ErrEmbedderRequired", err)
	}
	if _, err := NewEpisodic(nil, EpisodicOptions{}); !errors.Is(err, ErrEmbedderRequired) {
		t.Errorf("NewEpisodic(nil) err = %v, want ErrEmbedderRequired", err)
	}
	if _, err := NewSemantic(nil, SemanticOptions{}); !errors.Is(err, ErrEmbedderRequired) {
		t.Errorf("NewSemantic(nil) err = %v, want ErrEmbedderRequired", err)
	}
}

// --- working memory behavior ----------------------------------------------

func TestWorking_AddAndGet(t *testing.T) {
	w := newWorking(t)
	id, err := w.Add(context.Background(), MemoryItem{Content: "go modules", Importance: 0.8})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	got, err := w.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Content != "go modules" {
		t.Errorf("Content = %q", got.Content)
	}
	if got.AccessedAt.IsZero() {
		t.Error("AccessedAt not set on Get")
	}
}

func TestWorking_GetNotFound(t *testing.T) {
	w := newWorking(t)
	_, err := w.Get(context.Background(), "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestWorking_SearchEmptyQueryErrors(t *testing.T) {
	w := newWorking(t)
	if _, err := w.Search(context.Background(), "  ", 5); !errors.Is(err, ErrEmptyQuery) {
		t.Errorf("err = %v, want ErrEmptyQuery", err)
	}
}

func TestWorking_SearchRanksMostRelevantFirst(t *testing.T) {
	w := newWorking(t)
	ctx := context.Background()
	_, _ = w.Add(ctx, MemoryItem{Content: "go modules dependency management", Importance: 0.5})
	_, _ = w.Add(ctx, MemoryItem{Content: "rust ownership and borrowing", Importance: 0.5})
	_, _ = w.Add(ctx, MemoryItem{Content: "python async programming", Importance: 0.5})

	results, err := w.Search(ctx, "go modules", 3)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("no results")
	}
	if results[0].Item.Content != "go modules dependency management" {
		t.Errorf("top result = %q, want go modules first", results[0].Item.Content)
	}
}

func TestWorking_CapacityEnforced(t *testing.T) {
	w := newWorking(t) // capacity 5
	ctx := context.Background()
	for i := 0; i < 8; i++ {
		_, err := w.Add(ctx, MemoryItem{Content: "item " + string(rune('a'+i)), Importance: 0.5})
		if err != nil {
			t.Fatalf("Add %d: %v", i, err)
		}
	}
	if w.Stats().Count > 5 {
		t.Errorf("Count = %d, want <= 5 (capacity enforced)", w.Stats().Count)
	}
}

func TestWorking_UpdateChangesContentAndReembeds(t *testing.T) {
	w := newWorking(t)
	ctx := context.Background()
	id, _ := w.Add(ctx, MemoryItem{Content: "first", Importance: 0.5})
	if err := w.Update(ctx, id, func(it *MemoryItem) { it.Content = "updated content" }); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := w.Get(ctx, id)
	if got.Content != "updated content" {
		t.Errorf("Content = %q", got.Content)
	}
	// Search by new content should find it.
	res, _ := w.Search(ctx, "updated", 1)
	if len(res) != 1 || res[0].Item.ID != id {
		t.Errorf("re-embed broken: %v", res)
	}
}

func TestWorking_RemoveDeletes(t *testing.T) {
	w := newWorking(t)
	ctx := context.Background()
	id, _ := w.Add(ctx, MemoryItem{Content: "x", Importance: 0.5})
	if err := w.Remove(ctx, id); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	_, err := w.Get(ctx, id)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("after remove, Get err = %v, want ErrNotFound", err)
	}
}

// --- episodic memory ------------------------------------------------------

func TestEpisodic_NoCapacityLimit(t *testing.T) {
	m := newEpisodic(t)
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		_, _ = m.Add(ctx, MemoryItem{Content: "ev " + string(rune('a'+i%26)), Importance: 0.5})
	}
	if m.Stats().Count != 100 {
		t.Errorf("Count = %d, want 100 (no cap)", m.Stats().Count)
	}
	if m.Stats().Capacity != 0 {
		t.Errorf("Capacity = %d, want 0 (unlimited)", m.Stats().Capacity)
	}
}

func TestEpisodic_RecencyBoostsRecentItems(t *testing.T) {
	m := newEpisodic(t)
	ctx := context.Background()
	// Old item (manually backdated)
	oldID, _ := m.Add(ctx, MemoryItem{Content: "go programming language", Importance: 0.5})
	_ = m.Update(ctx, oldID, func(it *MemoryItem) {
		it.CreatedAt = time.Now().Add(-90 * 24 * time.Hour) // 3 months ago
	})
	// Recent identical-topic item
	_, _ = m.Add(ctx, MemoryItem{Content: "go programming language", Importance: 0.5})

	results, _ := m.Search(ctx, "go programming", 2)
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	// Recent item should outscore the old one
	if results[0].Item.ID == oldID {
		t.Errorf("old item ranked first; recency boost not applied")
	}
}

// --- semantic memory ------------------------------------------------------

func TestSemantic_TagOverlapBoostsScore(t *testing.T) {
	m := newSemantic(t)
	ctx := context.Background()
	_, _ = m.Add(ctx, MemoryItem{Content: "general go info", Tags: []string{"language"}, Importance: 0.5})
	_, _ = m.Add(ctx, MemoryItem{Content: "general go info", Tags: []string{"go", "modules"}, Importance: 0.5})

	// Tag-prefixed query → tag-overlap weighting kicks in
	results, _ := m.Search(ctx, "tag:go,modules go info", 2)
	if len(results) == 0 {
		t.Fatal("no results")
	}
	// Item with go+modules tags should rank first due to overlap
	if !contains(results[0].Item.Tags, "modules") {
		t.Errorf("top result tags = %v, want item with 'modules' tag", results[0].Item.Tags)
	}
}

func TestSemantic_TagPrefixFilters(t *testing.T) {
	m := newSemantic(t)
	ctx := context.Background()
	_, _ = m.Add(ctx, MemoryItem{Content: "x", Tags: []string{"a"}, Importance: 0.5})
	_, _ = m.Add(ctx, MemoryItem{Content: "x", Tags: []string{"b"}, Importance: 0.5})
	_, _ = m.Add(ctx, MemoryItem{Content: "x", Tags: []string{"c"}, Importance: 0.5})

	results, _ := m.Search(ctx, "tag:a x", 5)
	for _, r := range results {
		if !contains(r.Item.Tags, "a") {
			t.Errorf("filtered result has tags %v, want only 'a'", r.Item.Tags)
		}
	}
}

func TestSemantic_NoTagPrefixSearchesAll(t *testing.T) {
	m := newSemantic(t)
	ctx := context.Background()
	_, _ = m.Add(ctx, MemoryItem{Content: "x", Tags: []string{"a"}, Importance: 0.5})
	_, _ = m.Add(ctx, MemoryItem{Content: "x", Tags: []string{"b"}, Importance: 0.5})
	results, _ := m.Search(ctx, "x", 5)
	if len(results) != 2 {
		t.Errorf("got %d results, want 2 (no filter)", len(results))
	}
}

// --- shared helpers tests -------------------------------------------------

func TestImportanceClampedOnAddAndUpdate(t *testing.T) {
	w := newWorking(t)
	ctx := context.Background()
	id, _ := w.Add(ctx, MemoryItem{Content: "x", Importance: 1.5})
	got, _ := w.Get(ctx, id)
	if got.Importance != 1 {
		t.Errorf("Importance = %f, want clamped to 1", got.Importance)
	}
	_ = w.Update(ctx, id, func(it *MemoryItem) { it.Importance = -0.5 })
	got, _ = w.Get(ctx, id)
	if got.Importance != 0 {
		t.Errorf("after Update, Importance = %f, want clamped to 0", got.Importance)
	}
}

func TestStats_AvgImportanceAndOldestAge(t *testing.T) {
	w := newWorking(t)
	ctx := context.Background()
	_, _ = w.Add(ctx, MemoryItem{Content: "a", Importance: 0.2})
	_, _ = w.Add(ctx, MemoryItem{Content: "b", Importance: 0.8})
	s := w.Stats()
	if s.Count != 2 {
		t.Errorf("Count = %d, want 2", s.Count)
	}
	if s.AvgImportance < 0.49 || s.AvgImportance > 0.51 {
		t.Errorf("AvgImportance = %f, want ~0.5", s.AvgImportance)
	}
	if s.OldestAge < 0 {
		t.Errorf("OldestAge negative: %v", s.OldestAge)
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

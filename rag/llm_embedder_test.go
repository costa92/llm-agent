package rag

import (
	"context"
	"testing"

	"github.com/costa92/llm-agent/llm"
)

type llmEmbedderAdapter struct {
	inner llm.Embedder
	dim   int
}

func (a llmEmbedderAdapter) Embed(ctx context.Context, text string) ([]float32, error) {
	vectors, _, err := a.inner.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 {
		return nil, nil
	}
	return vectors[0], nil
}

func (a llmEmbedderAdapter) Dimension() int { return a.dim }

func TestRAGSystem_WorksWithLLMEmbedderAdapter(t *testing.T) {
	model := llm.NewScriptedLLM(
		llm.WithEmbedDimensions(8),
	)
	adapter := llmEmbedderAdapter{
		inner: model,
		dim:   model.EmbedDimensions(),
	}

	r := New(Options{
		Embedder: adapter,
		Store:    NewInMemoryStore(adapter.Dimension()),
	})
	ctx := context.Background()
	_, err := r.AddText(ctx, "go modules manage dependencies via go.mod files", nil)
	if err != nil {
		t.Fatalf("AddText: %v", err)
	}
	hits, err := r.Search(ctx, "go modules", 3, SearchOptions{})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("no hits")
	}
}

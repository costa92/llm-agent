package rag

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/costa92/llm-agent/llm"
)

// scriptedLLM is a multi-call mock returning preset responses by index.
type scriptedLLM struct {
	mu    sync.Mutex
	calls int
	resps []llm.GenerateResponse
}

func newScripted(rs ...llm.GenerateResponse) *scriptedLLM {
	return &scriptedLLM{resps: rs}
}
func (s *scriptedLLM) Generate(_ context.Context, _ llm.GenerateRequest) (llm.GenerateResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.calls >= len(s.resps) {
		s.calls++
		return llm.GenerateResponse{}, errors.New("scripted exhausted")
	}
	r := s.resps[s.calls]
	s.calls++
	return r, nil
}
func (s *scriptedLLM) GenerateStream(_ context.Context, _ llm.GenerateRequest) (<-chan llm.StreamChunk, error) {
	return nil, errors.New("nope")
}

func TestRAGSystem_AddAndSearch(t *testing.T) {
	r := New(Options{})
	ctx := context.Background()
	ids, err := r.AddText(ctx, "go modules manage dependencies for go projects via go.mod files", nil)
	if err != nil {
		t.Fatalf("AddText: %v", err)
	}
	if len(ids) == 0 {
		t.Fatal("no chunks generated")
	}
	hits, err := r.Search(ctx, "go modules", 5, SearchOptions{})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("no hits")
	}
	if !strings.Contains(hits[0].Doc.Content, "go modules") {
		t.Errorf("top hit content = %q", hits[0].Doc.Content)
	}
}

func TestRAGSystem_AddTextChunksLongInput(t *testing.T) {
	r := New(Options{MaxChunkChars: 100})
	ctx := context.Background()
	long := strings.Repeat("alpha beta gamma delta epsilon. ", 20) // ~640 chars
	ids, err := r.AddText(ctx, long, nil)
	if err != nil {
		t.Fatalf("AddText: %v", err)
	}
	if len(ids) < 2 {
		t.Errorf("got %d chunks, want >=2", len(ids))
	}
}

func TestRAGSystem_SearchEmptyQueryErrors(t *testing.T) {
	r := New(Options{})
	if _, err := r.Search(context.Background(), "  ", 5, SearchOptions{}); !errors.Is(err, ErrEmptyQuery) {
		t.Errorf("err = %v, want ErrEmptyQuery", err)
	}
}

func TestRAGSystem_AskRequiresLLM(t *testing.T) {
	r := New(Options{}) // no LLM
	if _, err := r.Ask(context.Background(), "anything", SearchOptions{}); !errors.Is(err, ErrLLMRequired) {
		t.Errorf("err = %v, want ErrLLMRequired", err)
	}
}

func TestRAGSystem_AskHappyPath(t *testing.T) {
	r := New(Options{LLM: newScripted(llm.GenerateResponse{Text: "Modules manage deps."})})
	ctx := context.Background()
	_, _ = r.AddText(ctx, "go modules manage dependencies via go.mod files", nil)
	out, err := r.Ask(ctx, "what are go modules?", SearchOptions{})
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if !strings.Contains(out, "Modules") {
		t.Errorf("answer = %q", out)
	}
}

func TestRAGSystem_MQERequiresLLM(t *testing.T) {
	r := New(Options{}) // no LLM
	_, err := r.Search(context.Background(), "x", 5, SearchOptions{EnableMQE: true})
	if !errors.Is(err, ErrLLMRequired) {
		t.Errorf("err = %v, want ErrLLMRequired", err)
	}
}

func TestRAGSystem_MQEExpandsAndDedupes(t *testing.T) {
	r := New(Options{
		LLM: newScripted(llm.GenerateResponse{Text: "go modules\nGo modules\ngo dependency mgmt\ngolang module system"}),
	})
	expansions, err := r.mqeExpand(context.Background(), "go modules", 3)
	if err != nil {
		t.Fatalf("mqeExpand: %v", err)
	}
	if len(expansions) < 2 {
		t.Errorf("got %d expansions, want >=2 (after dedup)", len(expansions))
	}
	for _, e := range expansions {
		if strings.EqualFold(e, "go modules") {
			t.Errorf("expansion %q duplicates original query", e)
		}
	}
}

func TestRAGSystem_HyDEGeneratesContext(t *testing.T) {
	r := New(Options{
		LLM: newScripted(llm.GenerateResponse{Text: "Go modules manage Go dependencies via go.mod."}),
	})
	hypo, err := r.hydeGenerate(context.Background(), "what are go modules?")
	if err != nil {
		t.Fatalf("hyde: %v", err)
	}
	if !strings.Contains(hypo, "modules") {
		t.Errorf("hypo = %q", hypo)
	}
}

func TestRAGSystem_SearchWithMQEMergesResults(t *testing.T) {
	r := New(Options{
		LLM: newScripted(llm.GenerateResponse{Text: "go dependency\nmodule system"}),
	})
	ctx := context.Background()
	_, _ = r.AddText(ctx, "go modules manage dependencies", nil)
	_, _ = r.AddText(ctx, "module system in golang", nil)
	hits, err := r.Search(ctx, "go modules", 5, SearchOptions{EnableMQE: true, MQECount: 2})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("no hits with MQE")
	}
}

func TestRAGSystem_RemoveAndStats(t *testing.T) {
	r := New(Options{})
	ctx := context.Background()
	ids, _ := r.AddText(ctx, "test content", nil)
	if r.Stats().Count != len(ids) {
		t.Errorf("Count = %d, want %d", r.Stats().Count, len(ids))
	}
	if err := r.Remove(ctx, ids[0]); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if r.Stats().Count != len(ids)-1 {
		t.Errorf("after remove, Count = %d", r.Stats().Count)
	}
}

package rag

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/costa92/llm-agent/llm"
)

// RAGSystem is the orchestration façade: chunk → embed → upsert on
// AddText; embed query → store.Search on Search; Search → render
// prompt → llm.Generate on Ask.
type RAGSystem struct {
	chunker  Chunker
	embedder Embedder
	store    VectorStore
	llm      llm.ChatModel
	maxChunk int
	seq      int
	mu       sync.Mutex
}

// Options configures a RAGSystem. All zero-values get safe defaults.
type Options struct {
	Chunker       Chunker
	Embedder      Embedder
	Store         VectorStore
	LLM           llm.ChatModel
	MaxChunkChars int
}

// SearchOptions tunes per-call retrieval.
type SearchOptions struct {
	EnableMQE               bool // multi-query expansion via LLM
	EnableHyDE              bool // hypothetical document embedding via LLM
	MQECount                int  // # rewrites; default 3
	CandidatePoolMultiplier int  // initial pool size = topK * multiplier; default 4
}

// New constructs a RAGSystem with sensible defaults.
func New(opts Options) *RAGSystem {
	embedder := opts.Embedder
	if embedder == nil {
		embedder = NewHashEmbedder(32)
	}
	store := opts.Store
	if store == nil {
		store = NewInMemoryStore(embedder.Dimension())
	}
	chunker := opts.Chunker
	if chunker == nil {
		chunker = CharChunker{Overlap: 50}
	}
	maxChunk := opts.MaxChunkChars
	if maxChunk <= 0 {
		maxChunk = 500
	}
	return &RAGSystem{
		chunker:  chunker,
		embedder: embedder,
		store:    store,
		llm:      opts.LLM,
		maxChunk: maxChunk,
	}
}

// AddText chunks text → embeds each chunk → upserts. Returns the new
// chunk IDs in chunk order.
func (r *RAGSystem) AddText(ctx context.Context, text string, meta map[string]any) ([]string, error) {
	chunks := r.chunker.Chunk(text, r.maxChunk)
	if len(chunks) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(chunks))
	for i, c := range chunks {
		vec, err := r.embedder.Embed(ctx, c)
		if err != nil {
			return nil, fmt.Errorf("rag: embed chunk %d: %w", i, err)
		}
		id := r.nextID()
		md := copyMeta(meta)
		md["chunk_index"] = i
		md["chunk_total"] = len(chunks)
		if err := r.store.Upsert(ctx, Document{ID: id, Content: c, Vector: vec, Metadata: md}); err != nil {
			return nil, fmt.Errorf("rag: upsert chunk %d: %w", i, err)
		}
		out = append(out, id)
	}
	return out, nil
}

// Search runs the configured retrieval pipeline. With both MQE and
// HyDE off, it's a single embed + store.Search. With either on,
// queries are expanded/rewritten via LLM (requires r.llm) and
// results are merged + de-duped + re-ranked.
func (r *RAGSystem) Search(ctx context.Context, query string, topK int, opts SearchOptions) ([]SearchHit, error) {
	if strings.TrimSpace(query) == "" {
		return nil, ErrEmptyQuery
	}
	if topK <= 0 {
		topK = 5
	}
	pool := topK
	if opts.CandidatePoolMultiplier > 1 {
		pool = topK * opts.CandidatePoolMultiplier
	}

	queries := []string{query}
	if opts.EnableMQE {
		if r.llm == nil {
			return nil, ErrLLMRequired
		}
		count := opts.MQECount
		if count <= 0 {
			count = 3
		}
		expansions, err := r.mqeExpand(ctx, query, count)
		if err != nil {
			return nil, fmt.Errorf("rag: MQE: %w", err)
		}
		queries = append(queries, expansions...)
	}
	if opts.EnableHyDE {
		if r.llm == nil {
			return nil, ErrLLMRequired
		}
		hypo, err := r.hydeGenerate(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("rag: HyDE: %w", err)
		}
		queries = append(queries, hypo)
	}

	merged := make(map[string]SearchHit, pool)
	for _, q := range queries {
		qv, err := r.embedder.Embed(ctx, q)
		if err != nil {
			return nil, fmt.Errorf("rag: embed query: %w", err)
		}
		hits, err := r.store.Search(ctx, qv, pool)
		if err != nil {
			return nil, fmt.Errorf("rag: store search: %w", err)
		}
		for _, h := range hits {
			if prev, ok := merged[h.Doc.ID]; !ok || h.Score > prev.Score {
				merged[h.Doc.ID] = h
			}
		}
	}

	out := make([]SearchHit, 0, len(merged))
	for _, h := range merged {
		out = append(out, h)
	}
	sortHitsDesc(out)
	if len(out) > topK {
		out = out[:topK]
	}
	return out, nil
}

// Ask runs Search and stuffs the hits into a context-aware prompt for
// the configured LLM. ErrLLMRequired if no LLM was wired.
func (r *RAGSystem) Ask(ctx context.Context, question string, opts SearchOptions) (string, error) {
	if r.llm == nil {
		return "", ErrLLMRequired
	}
	hits, err := r.Search(ctx, question, 5, opts)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("Use the context below to answer the question. Cite chunk IDs in [brackets] when you rely on them.\n\nContext:\n")
	for _, h := range hits {
		fmt.Fprintf(&b, "[%s] %s\n\n", h.Doc.ID, h.Doc.Content)
	}
	fmt.Fprintf(&b, "Question: %s", question)

	resp, err := r.llm.Generate(ctx, llm.Request{
		Messages: []llm.Message{{Role: "user", Content: b.String()}},
	})
	if err != nil {
		return "", fmt.Errorf("rag: llm: %w", err)
	}
	return resp.Text, nil
}

// Remove deletes one chunk by ID.
func (r *RAGSystem) Remove(ctx context.Context, id string) error {
	return r.store.Remove(ctx, id)
}

// Stats returns the underlying store stats.
func (r *RAGSystem) Stats() StoreStats {
	return r.store.Stats()
}

// nextID generates monotonic chunk IDs (no time component, no rand —
// keeps tests deterministic across runs).
func (r *RAGSystem) nextID() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	return fmt.Sprintf("chunk_%d", r.seq)
}

func copyMeta(in map[string]any) map[string]any {
	out := make(map[string]any, len(in)+2)
	for k, v := range in {
		out[k] = v
	}
	return out
}

func sortHitsDesc(hits []SearchHit) {
	// Insertion sort — small N (≤ pool*queries), no need for sort.Slice.
	for i := 1; i < len(hits); i++ {
		for j := i; j > 0 && hits[j].Score > hits[j-1].Score; j-- {
			hits[j], hits[j-1] = hits[j-1], hits[j]
		}
	}
}

// ErrEmptyQuery is returned by Search when query is whitespace-only.
var ErrEmptyQuery = errors.New("rag: query is required")

// ErrLLMRequired is returned by Ask / MQE / HyDE paths when no LLM
// client was configured.
var ErrLLMRequired = errors.New("rag: llm.ChatModel required for this operation")

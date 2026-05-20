package rag

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	ragembed "github.com/costa92/llm-agent-rag/embed"
	raggenerate "github.com/costa92/llm-agent-rag/generate"
	ragingest "github.com/costa92/llm-agent-rag/ingest"
	ragprompt "github.com/costa92/llm-agent-rag/prompt"
	ragcore "github.com/costa92/llm-agent-rag/rag"
	ragstore "github.com/costa92/llm-agent-rag/store"

	"github.com/costa92/llm-agent/llm"
)

// RAGSystem preserves the historical llm-agent rag API while delegating
// base import / retrieve / ask orchestration to the standalone SDK.
type RAGSystem struct {
	chunker  Chunker
	embedder Embedder
	store    VectorStore
	llm      llm.ChatModel
	maxChunk int
	seq      int

	core *ragcore.System

	mu sync.Mutex
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
	EnableMQE               bool
	EnableHyDE              bool
	MQECount                int
	EnableRerank            bool
	CandidatePoolMultiplier int
	Filters                 map[string]any
	SecurityFilters         map[string]any
}

const namespaceMetadataKey = "__rag_namespace"

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

	r := &RAGSystem{
		chunker:  chunker,
		embedder: embedder,
		store:    store,
		llm:      opts.LLM,
		maxChunk: maxChunk,
	}
	r.core = ragcore.New(ragcore.Options{
		Splitter: splitterAdapter{inner: chunker},
		Embedder: embedderAdapter{inner: embedder},
		Store:    newStoreAdapter(store),
		Model:    modelAdapter{inner: opts.LLM},
		Template: ragprompt.DefaultQATemplate{},
		MaxChars: maxChunk,
	})
	return r
}

// AddText chunks text → embeds each chunk → upserts. Returns the new chunk IDs.
func (r *RAGSystem) AddText(ctx context.Context, text string, meta map[string]any) ([]string, error) {
	docID := r.nextDocID()
	namespace := namespaceFromMetadata(meta)
	res, err := r.core.Import(ctx, []ragingest.Document{{
		ID:       docID,
		Content:  text,
		Metadata: meta,
	}}, ragingest.ImportOptions{
		Namespace: namespace,
		MaxChars:  r.maxChunk,
	})
	if err != nil {
		return nil, err
	}
	return res.ChunkIDs, nil
}

// Search runs the configured retrieval pipeline.
func (r *RAGSystem) Search(ctx context.Context, query string, topK int, opts SearchOptions) ([]SearchHit, error) {
	return r.searchWithNamespace(ctx, query, topK, "", opts)
}

func (r *RAGSystem) searchWithNamespace(ctx context.Context, query string, topK int, namespace string, opts SearchOptions) ([]SearchHit, error) {
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
	hits, err := r.core.Retrieve(ctx, query, ragcore.SearchOptions{
		TopK:            pool,
		Namespace:       namespace,
		Filters:         opts.Filters,
		SecurityFilters: opts.SecurityFilters,
		EnableMQE:       opts.EnableMQE,
		EnableHyDE:      opts.EnableHyDE,
		MQECount:        opts.MQECount,
	})
	if err != nil {
		if errors.Is(err, ragcore.ErrModelRequired) {
			return nil, ErrLLMRequired
		}
		return nil, fmt.Errorf("rag: store search: %w", err)
	}
	out := make([]SearchHit, 0, len(hits))
	for _, h := range hits {
		out = append(out, fromStoreHit(h))
	}
	return out, nil
}

// Ask runs Search and sends the composed prompt to the configured LLM.
func (r *RAGSystem) Ask(ctx context.Context, question string, opts SearchOptions) (string, error) {
	return r.askWithNamespace(ctx, question, "", opts)
}

func (r *RAGSystem) askWithNamespace(ctx context.Context, question string, namespace string, opts SearchOptions) (string, error) {
	if r.llm == nil {
		return "", ErrLLMRequired
	}
	answer, err := r.core.Ask(ctx, question, ragcore.AskOptions{
		Search: ragcore.SearchOptions{
			TopK:            5,
			Namespace:       namespace,
			Filters:         opts.Filters,
			SecurityFilters: opts.SecurityFilters,
			EnableMQE:       opts.EnableMQE,
			EnableHyDE:      opts.EnableHyDE,
			MQECount:        opts.MQECount,
			EnableRerank:    opts.EnableRerank,
		},
	})
	if err != nil {
		if errors.Is(err, ragcore.ErrModelRequired) {
			return "", ErrLLMRequired
		}
		return "", err
	}
	return answer.Text, nil
}

// Remove deletes one chunk by ID.
func (r *RAGSystem) Remove(ctx context.Context, id string) error {
	return r.store.Remove(ctx, id)
}

// Stats returns the underlying store stats.
func (r *RAGSystem) Stats() StoreStats {
	return r.store.Stats()
}

func (r *RAGSystem) nextDocID() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	return fmt.Sprintf("chunk_%d", r.seq)
}

// ErrEmptyQuery is returned by Search when query is whitespace-only.
var ErrEmptyQuery = errors.New("rag: query is required")

// ErrLLMRequired is returned by Ask / MQE / HyDE paths when no LLM client was configured.
var ErrLLMRequired = errors.New("rag: llm.ChatModel required for this operation")

type splitterAdapter struct {
	inner Chunker
}

func (a splitterAdapter) Split(doc ragingest.Document, maxChars int) []ragingest.Chunk {
	parts := a.inner.Chunk(doc.Content, maxChars)
	out := make([]ragingest.Chunk, 0, len(parts))
	for i, part := range parts {
		md := copyMeta(doc.Metadata)
		md["chunk_index"] = i
		md["chunk_total"] = len(parts)
		out = append(out, ragingest.Chunk{
			ID:       fmt.Sprintf("%s:%d", doc.ID, i),
			DocID:    doc.ID,
			Index:    i,
			Total:    len(parts),
			Title:    doc.Title,
			Content:  part,
			Metadata: md,
		})
	}
	return out
}

type embedderAdapter struct {
	inner Embedder
}

func (a embedderAdapter) Embed(ctx context.Context, text string) (ragembed.Vector, error) {
	v, err := a.inner.Embed(ctx, text)
	return ragembed.Vector(v), err
}

func (a embedderAdapter) Dimension() int {
	return a.inner.Dimension()
}

// lister is the optional enumeration capability a VectorStore may implement
// so the facade can list stored documents via a real list route instead of a
// similarity search. The default *InMemoryStore satisfies it via
// ListDocuments. A custom VectorStore that does not implement it falls back
// to the storeAdapter-maintained id index (see idIndex below).
type lister interface {
	ListDocuments(ctx context.Context) ([]Document, error)
}

// idIndex tracks the IDs the storeAdapter has observed through Upsert/Remove.
// It is the enumeration fallback for a custom VectorStore that does not
// implement the optional lister capability — storeAdapter.List walks the
// index and fetches each document via VectorStore.Get, never a nil-vector
// search. The storeAdapter is passed by value into the SDK, so the index is
// held behind a pointer to keep Upsert/Remove mutations visible to List.
type idIndex struct {
	mu  sync.Mutex
	ids map[string]struct{}
}

func (x *idIndex) add(id string) {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.ids[id] = struct{}{}
}

func (x *idIndex) remove(id string) {
	x.mu.Lock()
	defer x.mu.Unlock()
	delete(x.ids, id)
}

func (x *idIndex) snapshot() []string {
	x.mu.Lock()
	defer x.mu.Unlock()
	out := make([]string, 0, len(x.ids))
	for id := range x.ids {
		out = append(out, id)
	}
	return out
}

type storeAdapter struct {
	inner VectorStore
	index *idIndex
}

func newStoreAdapter(store VectorStore) storeAdapter {
	return storeAdapter{
		inner: store,
		index: &idIndex{ids: make(map[string]struct{})},
	}
}

func (a storeAdapter) Upsert(ctx context.Context, chunks []ragstore.StoredChunk) error {
	for _, chunk := range chunks {
		md := copyMeta(chunk.Metadata)
		if chunk.Namespace != "" {
			md[namespaceMetadataKey] = chunk.Namespace
		}
		if err := a.inner.Upsert(ctx, Document{
			ID:       chunk.ID,
			Content:  chunk.Content,
			Vector:   chunk.Vector,
			Metadata: md,
		}); err != nil {
			return err
		}
		a.index.add(chunk.ID)
	}
	return nil
}

func (a storeAdapter) Search(ctx context.Context, q ragstore.Query) ([]ragstore.Hit, error) {
	topK := q.TopK
	if topK <= 0 {
		topK = 5
	}
	pool := topK
	if q.Namespace != "" || len(q.Filters) > 0 {
		pool = a.inner.Stats().Count
		if pool == 0 {
			return nil, nil
		}
	}
	if len(q.SecurityFilters) > 0 {
		pool = a.inner.Stats().Count
		if pool == 0 {
			return nil, nil
		}
	}
	hits, err := a.inner.Search(ctx, q.Vector, pool)
	if err != nil {
		return nil, err
	}
	out := make([]ragstore.Hit, 0, len(hits))
	for _, hit := range hits {
		if q.Namespace != "" && namespaceFromMetadata(hit.Doc.Metadata) != q.Namespace {
			continue
		}
		if !metadataMatchesFilters(hit.Doc.Metadata, q.Filters) {
			continue
		}
		if !metadataMatchesFilters(hit.Doc.Metadata, q.SecurityFilters) {
			continue
		}
		out = append(out, ragstore.Hit{
			Chunk: ragstore.StoredChunk{
				ID:        hit.Doc.ID,
				Namespace: namespaceFromMetadata(hit.Doc.Metadata),
				Content:   hit.Doc.Content,
				Vector:    hit.Doc.Vector,
				Metadata:  hit.Doc.Metadata,
			},
			Score: hit.Score,
		})
	}
	if len(out) > topK {
		out = out[:topK]
	}
	return out, nil
}

// List enumerates stored chunks via a real list route — never a nil-vector
// similarity search. When the inner VectorStore implements the optional
// lister capability (the default *InMemoryStore does, via ListDocuments) it
// is used directly; otherwise the storeAdapter-maintained id index is walked
// and each document fetched via VectorStore.Get. Namespace/filter/security
// scoping is then applied with the facade's metadata-based matching.
func (a storeAdapter) List(ctx context.Context, namespace string, filters ragstore.Filter, securityFilters ragstore.Filter) ([]ragstore.StoredChunk, error) {
	docs, err := a.enumerate(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ragstore.StoredChunk, 0, len(docs))
	for _, doc := range docs {
		if namespace != "" && namespaceFromMetadata(doc.Metadata) != namespace {
			continue
		}
		if !metadataMatchesFilters(doc.Metadata, filters) {
			continue
		}
		if !metadataMatchesFilters(doc.Metadata, securityFilters) {
			continue
		}
		out = append(out, ragstore.StoredChunk{
			ID:        doc.ID,
			Namespace: namespaceFromMetadata(doc.Metadata),
			Content:   doc.Content,
			Vector:    doc.Vector,
			Metadata:  doc.Metadata,
		})
	}
	return out, nil
}

// enumerate returns every stored document without ever issuing a similarity
// search: the optional lister capability when present, the id-index fallback
// otherwise.
func (a storeAdapter) enumerate(ctx context.Context) ([]Document, error) {
	if l, ok := a.inner.(lister); ok {
		return l.ListDocuments(ctx)
	}
	ids := a.index.snapshot()
	out := make([]Document, 0, len(ids))
	for _, id := range ids {
		doc, err := a.inner.Get(ctx, id)
		if err != nil {
			if errors.Is(err, ErrStoreNotFound) {
				// The index may lag a store mutated outside the facade;
				// skip a vanished id rather than fail the whole enumeration.
				a.index.remove(id)
				continue
			}
			return nil, err
		}
		out = append(out, doc)
	}
	return out, nil
}

func (a storeAdapter) Get(ctx context.Context, id string) (ragstore.StoredChunk, error) {
	doc, err := a.inner.Get(ctx, id)
	if err != nil {
		return ragstore.StoredChunk{}, err
	}
	return ragstore.StoredChunk{
		ID:        doc.ID,
		Namespace: namespaceFromMetadata(doc.Metadata),
		Content:   doc.Content,
		Vector:    doc.Vector,
		Metadata:  doc.Metadata,
	}, nil
}

func (a storeAdapter) Remove(ctx context.Context, id string) error {
	if err := a.inner.Remove(ctx, id); err != nil {
		return err
	}
	a.index.remove(id)
	return nil
}

func (a storeAdapter) RemoveByFilter(ctx context.Context, namespace string, filters ragstore.Filter) (int, error) {
	chunks, err := a.List(ctx, namespace, filters, nil)
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, chunk := range chunks {
		if err := a.Remove(ctx, chunk.ID); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

func (a storeAdapter) Stats(_ context.Context, _ string) (ragstore.Stats, error) {
	stats := a.inner.Stats()
	return ragstore.Stats{Count: stats.Count, Dim: stats.Dim}, nil
}

type modelAdapter struct {
	inner llm.ChatModel
}

func (a modelAdapter) Generate(ctx context.Context, req raggenerate.Request) (raggenerate.Response, error) {
	if a.inner == nil {
		return raggenerate.Response{}, ragcore.ErrModelRequired
	}
	msgs := make([]llm.Message, 0, len(req.Messages))
	for _, msg := range req.Messages {
		msgs = append(msgs, llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	resp, err := a.inner.Generate(ctx, llm.Request{
		SystemPrompt: req.SystemPrompt,
		Messages:     msgs,
		Metadata:     req.Metadata,
	})
	if err != nil {
		return raggenerate.Response{}, err
	}
	return raggenerate.Response{Text: resp.Text}, nil
}

func fromStoreHit(hit ragstore.Hit) SearchHit {
	return SearchHit{
		Doc: Document{
			ID:       hit.Chunk.ID,
			Content:  hit.Chunk.Content,
			Vector:   hit.Chunk.Vector,
			Metadata: hit.Chunk.Metadata,
		},
		Score: hit.Score,
	}
}

func namespaceFromMetadata(meta map[string]any) string {
	if meta == nil {
		return ""
	}
	raw, ok := meta[namespaceMetadataKey]
	if !ok {
		return ""
	}
	s, _ := raw.(string)
	return s
}

func metadataMatchesFilters(meta map[string]any, filters map[string]any) bool {
	if len(filters) == 0 {
		return true
	}
	for k, want := range filters {
		got, ok := meta[k]
		if !ok || !reflect.DeepEqual(got, want) {
			return false
		}
	}
	return true
}

func copyMeta(in map[string]any) map[string]any {
	out := make(map[string]any, len(in)+2)
	for k, v := range in {
		out[k] = v
	}
	return out
}

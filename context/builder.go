package context

import (
	stdctx "context"
	"math"

	"github.com/costa92/llm-agent/llm"
	"github.com/costa92/llm-agent/memory"
)

// Builder runs the GSSC pipeline. Construct via New + functional Options.
type Builder struct {
	cfg      Config
	counter  TokenCounter
	llm      llm.ChatModel
	embedder llm.Embedder
}

// Option mutates a Builder in New. Functional options keep New signature
// stable as we add knobs.
type Option func(*Builder)

// WithTokenCounter swaps the default SimpleCounter.
func WithTokenCounter(c TokenCounter) Option {
	return func(b *Builder) { b.counter = c }
}

// WithLLM enables the Compress phase's optional LLM-backed summarization.
func WithLLM(c llm.ChatModel) Option {
	return func(b *Builder) { b.llm = c }
}

// WithEmbedder swaps Jaccard relevance for embedding cosine similarity
// during Select. The query is embedded once per Build call.
func WithEmbedder(e llm.Embedder) Option {
	return func(b *Builder) { b.embedder = e }
}

// New constructs a Builder with cfg defaults applied.
func New(cfg Config, opts ...Option) *Builder {
	b := &Builder{cfg: cfg.applyDefaults(), counter: SimpleCounter{}}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// BuildInput is the multi-source assembly of a single Build call.
type BuildInput struct {
	UserQuery    string
	SystemPrompt string
	History      []llm.Message
	MemoryHits   []memory.SearchResult
	RAGHits      []EvidenceHit
	Custom       []Packet
}

// EvidenceHit is the minimal external-knowledge shape context assembly
// needs. Callers adapt store/rag-specific hit types into this struct.
type EvidenceHit struct {
	ID       string
	Content  string
	Score    float64
	Metadata map[string]any
}

// BuildOutput is the result of Build. Prompt is ready to feed
// llm.Generate. Selected is the actual kept set (audit trail).
type BuildOutput struct {
	Prompt       string
	Selected     []Packet
	UsedTokens   int
	DroppedCount int
}

// Build runs Gather → Select → Structure → Compress and returns the
// prompt + audit info. Background context is used internally; pass a
// real ctx via BuildCtx if you need cancellation propagation.
func (b *Builder) Build(input BuildInput) BuildOutput {
	return b.BuildCtx(stdctx.Background(), input)
}

// BuildCtx is the ctx-aware variant. Cancellation only matters when
// EnableCompress=true + Builder has an LLM (the Compress LLM call
// honors ctx) or when WithEmbedder is in use (the embedder may call
// network providers).
func (b *Builder) BuildCtx(ctx stdctx.Context, input BuildInput) BuildOutput {
	// Gather
	packets := gather(input, b.counter)

	// Pick relevance fn
	scoreFn := relevanceFn(jaccardRelevance)
	if b.embedder != nil && input.UserQuery != "" {
		if qv, err := embedOne(ctx, b.embedder, input.UserQuery); err == nil {
			scoreFn = embedderRelevance(b.embedder, qv)
		}
	}

	// Select
	kept, dropped := selectPackets(ctx, input.UserQuery, packets, b.cfg, scoreFn)

	// Structure
	prompt := structurePackets(input.UserQuery, kept)

	// Compress (if needed)
	prompt = compress(ctx, prompt, b.counter, b.cfg, b.llm)

	return BuildOutput{
		Prompt:       prompt,
		Selected:     kept,
		UsedTokens:   b.counter.Count(prompt),
		DroppedCount: dropped,
	}
}

func embedOne(ctx stdctx.Context, e llm.Embedder, text string) ([]float32, error) {
	vectors, _, err := e.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 {
		return nil, nil
	}
	return vectors[0], nil
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		af := float64(a[i])
		bf := float64(b[i])
		dot += af * bf
		normA += af * af
		normB += bf * bf
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

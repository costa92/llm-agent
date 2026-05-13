package rag

import (
	"context"
	"hash/fnv"
	"math"
	"strings"

	ragembed "github.com/costa92/llm-agent-rag/embed"
)

// Embedder converts text into a fixed-dimension vector for similarity search.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	Dimension() int
}

// HashEmbedder is the compatibility wrapper over the standalone SDK embedder.
type HashEmbedder struct {
	Dim   int
	inner *ragembed.HashEmbedder
}

// NewHashEmbedder constructs a HashEmbedder. dim ≤ 0 → defaults to 32.
func NewHashEmbedder(dim int) *HashEmbedder {
	inner := ragembed.NewHashEmbedder(dim)
	return &HashEmbedder{
		Dim:   inner.Dimension(),
		inner: inner,
	}
}

// Dimension implements Embedder.
func (h *HashEmbedder) Dimension() int { return h.inner.Dimension() }

// Embed implements Embedder.
func (h *HashEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	v, err := h.inner.Embed(ctx, text)
	return []float32(v), err
}

// CosineSimilarity delegates to the standalone SDK implementation.
func CosineSimilarity(a, b []float32) float64 {
	return ragembed.CosineSimilarity(ragembed.Vector(a), ragembed.Vector(b))
}

// tokenize / bucketFor / normalize remain for compatibility with local tests.
func tokenize(text string) []string {
	text = strings.ToLower(text)
	out := make([]string, 0, len(text)/4)
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			cur.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return out
}

func bucketFor(tok string, dim int) int {
	h := fnv.New64a()
	_, _ = h.Write([]byte(tok))
	return int(h.Sum64() % uint64(dim))
}

func normalize(v []float32) {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	if sum == 0 {
		return
	}
	mag := float32(math.Sqrt(sum))
	for i := range v {
		v[i] /= mag
	}
}

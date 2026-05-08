package rag

import (
	"context"
	"hash/fnv"
	"math"
	"strings"
)

// Embedder converts text into a fixed-dimension vector for similarity
// search. Implementations may call out to a model (Ollama / OpenAI /
// DashScope) — the interface stays portable.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	Dimension() int
}

// HashEmbedder is the zero-dependency fallback: tokenize, hash each
// token via FNV-64 modulo Dim into a bucket, accumulate counts, then
// L2-normalize. Deterministic + free; semantic quality is poor (no
// synonym awareness) but it's sufficient for learning + tests.
//
// Default Dim is 32. Set Dim explicitly via NewHashEmbedder(N) for
// larger vectors. Larger Dim → fewer collisions but no semantic gain.
type HashEmbedder struct {
	Dim int
}

// NewHashEmbedder constructs a HashEmbedder. dim ≤ 0 → defaults to 32.
func NewHashEmbedder(dim int) *HashEmbedder {
	if dim <= 0 {
		dim = 32
	}
	return &HashEmbedder{Dim: dim}
}

// Dimension implements Embedder.
func (h *HashEmbedder) Dimension() int { return h.Dim }

// Embed implements Embedder. ctx is accepted for interface compliance
// but unused (Hash embedding is local + fast).
func (h *HashEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	v := make([]float32, h.Dim)
	for _, tok := range tokenize(text) {
		idx := bucketFor(tok, h.Dim)
		v[idx]++
	}
	normalize(v)
	return v, nil
}

// CosineSimilarity returns the cosine similarity of two equal-length
// L2-normalized vectors. For non-normalized vectors it still returns
// the dot product divided by the magnitude product (clamped [0, 1]).
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	sim := dot / (math.Sqrt(na) * math.Sqrt(nb))
	if sim < 0 {
		return 0 // negative cosine → treat as no similarity for ranking
	}
	if sim > 1 {
		return 1
	}
	return sim
}

// tokenize lower-cases + splits on non-letter/digit runs. Stable order
// across runs (no map iteration).
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

package rag

import (
	"context"
	"math"
	"testing"
)

func TestHashEmbedder_DefaultDim(t *testing.T) {
	e := NewHashEmbedder(0)
	if e.Dimension() != 32 {
		t.Errorf("default Dim = %d, want 32", e.Dimension())
	}
	e2 := NewHashEmbedder(64)
	if e2.Dimension() != 64 {
		t.Errorf("custom Dim = %d, want 64", e2.Dimension())
	}
}

func TestHashEmbedder_DeterministicOutput(t *testing.T) {
	e := NewHashEmbedder(32)
	v1, _ := e.Embed(context.Background(), "hello world")
	v2, _ := e.Embed(context.Background(), "hello world")
	if len(v1) != 32 || len(v2) != 32 {
		t.Fatalf("got Dim %d/%d, want 32/32", len(v1), len(v2))
	}
	for i := range v1 {
		if v1[i] != v2[i] {
			t.Errorf("non-deterministic at index %d: %f vs %f", i, v1[i], v2[i])
		}
	}
}

func TestHashEmbedder_NormalizedToUnitLength(t *testing.T) {
	e := NewHashEmbedder(16)
	v, _ := e.Embed(context.Background(), "the quick brown fox jumps over the lazy dog")
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	if math.Abs(sum-1.0) > 1e-5 {
		t.Errorf("L2 norm² = %f, want ~1.0 (unit vector)", sum)
	}
}

func TestHashEmbedder_EmptyTextZeroVector(t *testing.T) {
	e := NewHashEmbedder(8)
	v, _ := e.Embed(context.Background(), "   ")
	for i, x := range v {
		if x != 0 {
			t.Errorf("v[%d] = %f, want 0 for empty input", i, x)
		}
	}
}

func TestCosineSimilarity_IdenticalIs1(t *testing.T) {
	e := NewHashEmbedder(32)
	v, _ := e.Embed(context.Background(), "go modules dependency management")
	if sim := CosineSimilarity(v, v); math.Abs(sim-1.0) > 1e-5 {
		t.Errorf("self-similarity = %f, want 1.0", sim)
	}
}

func TestCosineSimilarity_OrthogonalIs0(t *testing.T) {
	a := []float32{1, 0, 0, 0}
	b := []float32{0, 1, 0, 0}
	if sim := CosineSimilarity(a, b); sim != 0 {
		t.Errorf("orthogonal sim = %f, want 0", sim)
	}
}

func TestCosineSimilarity_LengthMismatchIs0(t *testing.T) {
	a := []float32{1, 0}
	b := []float32{1, 0, 0}
	if sim := CosineSimilarity(a, b); sim != 0 {
		t.Errorf("mismatch sim = %f, want 0", sim)
	}
}

func TestCosineSimilarity_RelatedTextsScoreHigher(t *testing.T) {
	e := NewHashEmbedder(64)
	a, _ := e.Embed(context.Background(), "go modules import path")
	b, _ := e.Embed(context.Background(), "go modules dependency import")
	c, _ := e.Embed(context.Background(), "completely unrelated chocolate cake recipe")

	simAB := CosineSimilarity(a, b)
	simAC := CosineSimilarity(a, c)
	if simAB <= simAC {
		t.Errorf("similar pair sim=%f should beat unrelated pair sim=%f", simAB, simAC)
	}
}

func TestTokenize_LowerAndSplit(t *testing.T) {
	got := tokenize("Hello, World! Go-Modules 1.21")
	want := []string{"hello", "world", "go", "modules", "1", "21"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("tok[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

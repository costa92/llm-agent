package rag

import (
	"strings"
	"testing"
)

func TestCharChunker_ShortTextReturnsAsOne(t *testing.T) {
	c := CharChunker{}
	got := c.Chunk("hello world", 100)
	if len(got) != 1 || got[0] != "hello world" {
		t.Errorf("got %v", got)
	}
}

func TestCharChunker_PrefersParagraphBreaks(t *testing.T) {
	text := strings.Repeat("a", 480) + "\n\n" + strings.Repeat("b", 200)
	c := CharChunker{}
	got := c.Chunk(text, 500)
	if len(got) < 2 {
		t.Fatalf("got %d chunks, want >=2", len(got))
	}
	// First chunk should end before the paragraph break (no leading b's).
	if strings.Contains(got[0], "b") {
		t.Errorf("first chunk contains b's; paragraph split missed: %q", got[0][:50])
	}
}

func TestCharChunker_HardSplitWhenNoBreak(t *testing.T) {
	text := strings.Repeat("a", 1500)
	c := CharChunker{}
	got := c.Chunk(text, 500)
	if len(got) < 2 {
		t.Fatalf("got %d chunks, want >=2", len(got))
	}
	for i, ch := range got {
		// Allow up to 20% slack from soft cap
		if len(ch) > 600 {
			t.Errorf("chunk[%d] = %d chars, exceeds 600 limit", i, len(ch))
		}
	}
}

func TestCharChunker_OverlapCarriesContext(t *testing.T) {
	text := "alpha beta gamma delta epsilon zeta eta theta iota kappa lambda mu nu xi omicron pi rho sigma tau upsilon phi chi psi omega"
	c := CharChunker{Overlap: 10}
	got := c.Chunk(text, 50)
	if len(got) < 2 {
		t.Fatalf("got %d chunks, want >=2", len(got))
	}
	// Some characters from the end of chunk[0] should reappear at start of chunk[1].
	tail := got[0]
	if len(tail) > 10 {
		tail = tail[len(tail)-10:]
	}
	if !strings.Contains(got[1], strings.Fields(tail)[0]) {
		t.Errorf("overlap not carried: end-of-0 %q vs start-of-1 %q", tail, got[1][:20])
	}
}

func TestCharChunker_EmptyInput(t *testing.T) {
	c := CharChunker{}
	if got := c.Chunk("", 500); len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
	if got := c.Chunk("   ", 500); len(got) != 0 {
		t.Errorf("whitespace-only got %v, want empty", got)
	}
}

func TestCharChunker_DefaultMaxChars(t *testing.T) {
	c := CharChunker{}
	got := c.Chunk(strings.Repeat("x", 1200), 0)
	if len(got) < 2 {
		t.Errorf("got %d chunks; default 500 should split 1200-char input", len(got))
	}
}

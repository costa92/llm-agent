package rag

import "strings"

// Chunker breaks long text into a slice of chunks for indexing.
// Implementations may be character/sentence/structure aware. maxChars
// is a soft upper bound on each chunk size.
type Chunker interface {
	Chunk(text string, maxChars int) []string
}

// CharChunker is a character-budget chunker that prefers paragraph
// breaks (\n\n) when one exists within the soft max. Optional Overlap
// (chars) carries trailing context into the next chunk for retrieval
// continuity.
//
// Soft max: if no paragraph break is found inside [maxChars,
// maxChars*1.2], the chunker still cuts at maxChars to keep memory
// bounded.
type CharChunker struct {
	Overlap int // chars carried into the next chunk; default 0
}

// Chunk implements Chunker.
func (c CharChunker) Chunk(text string, maxChars int) []string {
	if maxChars <= 0 {
		maxChars = 500
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if len(text) <= maxChars {
		return []string{text}
	}

	overlap := c.Overlap
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= maxChars {
		overlap = maxChars / 2 // never let overlap eat the whole window
	}

	out := make([]string, 0, len(text)/maxChars+1)
	for start := 0; start < len(text); {
		hardEnd := start + maxChars + maxChars/5 // soft cap +20%
		if hardEnd >= len(text) {
			out = append(out, strings.TrimSpace(text[start:]))
			break
		}
		// Search [start, hardEnd] for the LAST paragraph break — prefer
		// a clean semantic split anywhere in the soft-cap window.
		breakAt := -1
		if idx := strings.LastIndex(text[start:hardEnd], "\n\n"); idx >= 0 {
			breakAt = start + idx
		}
		if breakAt < 0 {
			// No paragraph break — cut at maxChars on the last whitespace.
			end := start + maxChars
			if ws := strings.LastIndexByte(text[start:end], ' '); ws > 0 {
				breakAt = start + ws
			} else {
				breakAt = end
			}
		}
		nextStart := breakAt
		// Skip the "\n\n" delimiter so the next chunk starts cleanly.
		if breakAt < len(text)-1 && text[breakAt] == '\n' && text[breakAt+1] == '\n' {
			nextStart = breakAt + 2
		}
		out = append(out, strings.TrimSpace(text[start:breakAt]))
		next := nextStart - overlap
		if next <= start {
			next = start + 1 // avoid infinite loop on degenerate input
		}
		start = next
	}
	// Filter empty trims.
	cleaned := out[:0]
	for _, c := range out {
		if c != "" {
			cleaned = append(cleaned, c)
		}
	}
	return cleaned
}

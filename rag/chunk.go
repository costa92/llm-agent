package rag

import ragingest "github.com/costa92/llm-agent-rag/ingest"

// Chunker breaks long text into a slice of chunks for indexing.
type Chunker interface {
	Chunk(text string, maxChars int) []string
}

// CharChunker is the compatibility wrapper over the standalone SDK splitter.
type CharChunker struct {
	Overlap int
}

// Chunk implements Chunker.
func (c CharChunker) Chunk(text string, maxChars int) []string {
	splitter := ragingest.CharSplitter{Overlap: c.Overlap}
	chunks := splitter.Split(ragingest.Document{
		ID:      "doc",
		Content: text,
	}, maxChars)
	out := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		out = append(out, chunk.Content)
	}
	return out
}

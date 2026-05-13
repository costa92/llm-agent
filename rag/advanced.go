package rag

import (
	"context"
	"fmt"
	"strings"

	"github.com/costa92/llm-agent/llm"
)

// mqeExpand asks the LLM to rewrite query into n semantically
// equivalent variants. One LLM call returns N expansions, parsed by
// splitting on newlines. Empty / dup lines are discarded.
func (r *RAGSystem) mqeExpand(ctx context.Context, query string, n int) ([]string, error) {
	prompt := fmt.Sprintf(`Rewrite the user's search query into %d semantically-equivalent alternatives.
Each alternative on its own line, no numbering, no commentary.

Query: %s`, n, query)

	resp, err := r.llm.Generate(ctx, llm.Request{
		Messages: []llm.Message{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{strings.ToLower(query): true}
	out := make([]string, 0, n)
	for _, line := range strings.Split(resp.Text, "\n") {
		line = strings.TrimSpace(line)
		// Strip common numbering prefixes (1., -, *).
		line = strings.TrimLeft(line, "0123456789.-* \t")
		if line == "" {
			continue
		}
		key := strings.ToLower(line)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, line)
		if len(out) >= n {
			break
		}
	}
	return out, nil
}

// hydeGenerate asks the LLM to write a hypothetical short answer that
// the user is searching for. The hypo answer's embedding tends to be
// closer to the actual relevant docs than the raw query embedding —
// especially for under-specified queries.
func (r *RAGSystem) hydeGenerate(ctx context.Context, query string) (string, error) {
	prompt := fmt.Sprintf(`Write a short hypothetical answer (2-3 sentences) that would directly answer the question below. Do not say "I don't know" — invent plausible-sounding content; this output is used to find similar real documents, not shown to the user.

Question: %s`, query)

	resp, err := r.llm.Generate(ctx, llm.Request{
		Messages: []llm.Message{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Text), nil
}

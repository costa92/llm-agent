package context

import (
	stdctx "context"
	"fmt"
	"strings"

	"github.com/costa92/llm-agent/llm"
)

// structurePackets renders kept packets into the canonical 5-section
// prompt layout per spec §8.3 (Structure phase):
//
//	[Role & Policies]   ← System packets
//	[Task]              ← UserQuery
//	[Evidence]          ← RAG hits
//	[Context]           ← Memory hits + Custom packets
//	[History]           ← Conversation history
//
// Sections are emitted only when they have content.
func structurePackets(userQuery string, kept []Packet) string {
	groups := groupBySource(kept)

	var b strings.Builder
	if pkts := groups[SourceSystem]; len(pkts) > 0 {
		b.WriteString("[Role & Policies]\n")
		for _, p := range pkts {
			b.WriteString(p.Content)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("[Task]\n")
	b.WriteString(userQuery)
	b.WriteString("\n\n")
	if pkts := groups[SourceRAG]; len(pkts) > 0 {
		b.WriteString("[Evidence]\n")
		for _, p := range pkts {
			fmt.Fprintf(&b, "- %s\n", p.Content)
		}
		b.WriteString("\n")
	}
	if memPkts, custom := groups[SourceMemory], groups[SourceCustom]; len(memPkts)+len(custom) > 0 {
		b.WriteString("[Context]\n")
		for _, p := range memPkts {
			fmt.Fprintf(&b, "- %s\n", p.Content)
		}
		for _, p := range custom {
			fmt.Fprintf(&b, "- %s\n", p.Content)
		}
		b.WriteString("\n")
	}
	if pkts := groups[SourceConversation]; len(pkts) > 0 {
		b.WriteString("[History]\n")
		for _, p := range pkts {
			b.WriteString(p.Content)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func groupBySource(packets []Packet) map[Source][]Packet {
	out := make(map[Source][]Packet, 5)
	for _, p := range packets {
		out[p.Source] = append(out[p.Source], p)
	}
	return out
}

// compress applies fallback truncation when the structured prompt still
// exceeds budget after Select. If a llm.Client is configured AND
// EnableCompress=true, the Evidence + Context sections (in that order)
// are summarized via the LLM. Otherwise sections are hard-truncated.
func compress(ctx stdctx.Context, prompt string, counter TokenCounter, cfg Config, client llm.Client) string {
	tokens := counter.Count(prompt)
	if tokens <= cfg.MaxTokens {
		return prompt
	}

	if client != nil && cfg.EnableCompress {
		if shrunk, err := llmSummarize(ctx, prompt, counter, cfg, client); err == nil {
			return shrunk
		}
		// Fall through to hard truncation on summarize failure.
	}

	return hardTruncate(prompt, counter, cfg.MaxTokens)
}

// llmSummarize asks the LLM to produce a tight summary of the prompt
// that fits the budget. Returns the summary or an error.
func llmSummarize(ctx stdctx.Context, prompt string, counter TokenCounter, cfg Config, client llm.Client) (string, error) {
	target := int(float64(cfg.MaxTokens) * 0.8)
	req := llm.GenerateRequest{Prompt: fmt.Sprintf(
		`Compress the prompt below to ~%d tokens. Preserve sections and key facts; drop filler.

PROMPT:
%s`, target, prompt)}
	resp, err := client.Generate(ctx, req)
	if err != nil {
		return "", err
	}
	out := strings.TrimSpace(resp.Text)
	// Sanity check: if LLM returned something larger or equal, fall back.
	if counter.Count(out) >= counter.Count(prompt) {
		return "", fmt.Errorf("compress: summarization did not shrink prompt")
	}
	return out, nil
}

// hardTruncate slices the prompt to fit MaxTokens. Coarse — just drops
// the tail. Adds a "[truncated]" marker.
func hardTruncate(prompt string, counter TokenCounter, maxTokens int) string {
	// Binary-shrink by char until under budget.
	lo, hi := 0, len(prompt)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if counter.Count(prompt[:mid]) > maxTokens {
			hi = mid - 1
		} else {
			lo = mid
		}
	}
	return strings.TrimRight(prompt[:lo], " \n\t") + "\n\n[truncated]"
}

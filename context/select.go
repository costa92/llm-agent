package context

import (
	stdctx "context"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/costa92/llm-agent-contract/llm"
)

// relevanceFn computes packet relevance for a query. Two implementations:
//   - jaccardRelevance — token-set overlap, no embedder required
//   - embedderRelevance — cosine similarity via llm.Embedder
type relevanceFn func(ctx stdctx.Context, query string, packet Packet) float64

// jaccardRelevance returns |query ∩ content| / |query ∪ content| over
// lower-cased ASCII/digit token sets. Stateless.
func jaccardRelevance(_ stdctx.Context, query string, packet Packet) float64 {
	q := tokenSet(query)
	c := tokenSet(packet.Content)
	if len(q) == 0 || len(c) == 0 {
		return 0
	}
	inter := 0
	for tok := range q {
		if c[tok] {
			inter++
		}
	}
	union := len(q) + len(c) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// embedderRelevance returns cosine similarity. Computed lazily — the
// query is embedded once per Build call and reused via closure.
func embedderRelevance(e llm.Embedder, queryVec []float32) relevanceFn {
	return func(ctx stdctx.Context, _ string, packet Packet) float64 {
		v, err := embedOne(ctx, e, packet.Content)
		if err != nil {
			return 0
		}
		return cosineSimilarity(queryVec, v)
	}
}

// recencyScore returns exp(-age_hours/24). Age=0 → 1; age=24h → ~0.37;
// age=72h → ~0.05. Items with zero Timestamp get recency = 1 (i.e.
// system / current-turn packets are treated as "fresh").
func recencyScore(ts time.Time) float64 {
	if ts.IsZero() {
		return 1
	}
	hours := time.Since(ts).Hours()
	if hours < 0 {
		return 1
	}
	return math.Exp(-hours / 24.0)
}

// tokenSet lowercases and tokenizes on non-alnum runs into a set.
func tokenSet(text string) map[string]bool {
	out := make(map[string]bool, len(text)/4)
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			out[cur.String()] = true
			cur.Reset()
		}
	}
	for _, r := range strings.ToLower(text) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			cur.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return out
}

// gather converts BuildInput sources into a uniform []Packet, computes
// TokenCount on each. Order: SystemPrompt → History → MemoryHits →
// RAGHits → Custom.
func gather(input BuildInput, counter TokenCounter) []Packet {
	out := make([]Packet, 0, 8+len(input.History)+len(input.MemoryHits)+len(input.RAGHits)+len(input.Custom))
	now := time.Now()

	if input.SystemPrompt != "" {
		out = append(out, Packet{
			Content:    input.SystemPrompt,
			Source:     SourceSystem,
			Timestamp:  now,
			TokenCount: counter.Count(input.SystemPrompt),
		})
	}
	for _, m := range input.History {
		body := m.Role + ": " + m.Content
		out = append(out, Packet{
			Content:    body,
			Source:     SourceConversation,
			Timestamp:  now,
			TokenCount: counter.Count(body),
		})
	}
	for _, h := range input.MemoryHits {
		out = append(out, Packet{
			Content:    h.Item.Content,
			Source:     SourceMemory,
			Timestamp:  h.Item.CreatedAt,
			TokenCount: counter.Count(h.Item.Content),
			Metadata:   map[string]any{"score": h.Score, "id": h.Item.ID, "tags": h.Item.Tags},
		})
	}
	for _, h := range input.RAGHits {
		meta := map[string]any{"score": h.Score, "id": h.ID}
		for k, v := range h.Metadata {
			meta[k] = v
		}
		out = append(out, Packet{
			Content:    h.Content,
			Source:     SourceRAG,
			Timestamp:  now,
			TokenCount: counter.Count(h.Content),
			Metadata:   meta,
		})
	}
	for _, p := range input.Custom {
		if p.TokenCount == 0 {
			p.TokenCount = counter.Count(p.Content)
		}
		if p.Source == "" {
			p.Source = SourceCustom
		}
		if p.Timestamp.IsZero() {
			p.Timestamp = now
		}
		out = append(out, p)
	}
	return out
}

// selectPackets scores + filters + greedy-fills under budget. Returns
// the kept packets (in original order — Structure phase regroups by
// Source).
func selectPackets(ctx stdctx.Context, query string, packets []Packet, cfg Config, score relevanceFn) ([]Packet, int) {
	// 1. Score every packet
	for i := range packets {
		// System packets always rank high — they're scaffolding, not evidence.
		if packets[i].Source == SourceSystem {
			packets[i].Relevance = 1
			continue
		}
		packets[i].Relevance = score(ctx, query, packets[i])
	}

	// 2. Filter under MinRelevance (system always survives)
	filtered := packets[:0]
	for _, p := range packets {
		if p.Source == SourceSystem || p.Relevance >= cfg.MinRelevance {
			filtered = append(filtered, p)
		}
	}

	// 3. Sort by composite score (recency × Wᵣ + relevance × Wₗ).
	type ranked struct {
		idx   int
		score float64
		pkt   Packet
	}
	ranks := make([]ranked, len(filtered))
	for i, p := range filtered {
		s := recencyScore(p.Timestamp)*cfg.RecencyWeight + p.Relevance*cfg.RelevanceWeight
		// System gets a flat 2.0 to guarantee it stays at the top.
		if p.Source == SourceSystem {
			s = 2.0
		}
		ranks[i] = ranked{idx: i, score: s, pkt: p}
	}
	sort.SliceStable(ranks, func(i, j int) bool { return ranks[i].score > ranks[j].score })

	// 4. Greedy fill under budget
	budget := int(float64(cfg.MaxTokens) * (1.0 - cfg.ReserveRatio))
	if budget < 0 {
		budget = 0
	}
	used := 0
	keptIdx := make(map[int]bool, len(ranks))
	for _, r := range ranks {
		if used+r.pkt.TokenCount > budget {
			continue
		}
		used += r.pkt.TokenCount
		keptIdx[r.idx] = true
	}

	// 5. Restore original order
	kept := make([]Packet, 0, len(keptIdx))
	for i, p := range filtered {
		if keptIdx[i] {
			kept = append(kept, p)
		}
	}
	dropped := len(packets) - len(kept)
	return kept, dropped
}

package bench

import (
	"context"
	"iter"
	"sort"
	"strings"

	"github.com/costa92/llm-agent"
	"github.com/costa92/llm-agent/rl"
)

// GAIASample is one GAIA-style sample with a difficulty Level (1-3).
type GAIASample struct {
	rl.Sample
	Level int
}

// gaiaDataset wraps GAIASamples behind rl.Dataset.
type gaiaDataset struct {
	name    string
	samples []GAIASample
}

// NewGAIADataset wraps the provided samples in an rl.Dataset.
func NewGAIADataset(name string, samples []GAIASample) rl.Dataset {
	return &gaiaDataset{name: name, samples: samples}
}

// MiniGAIA returns the built-in 3-sample fixture (one per level).
func MiniGAIA() rl.Dataset {
	return NewGAIADataset("gaia-mini", MiniGAIASamples())
}

func (d *gaiaDataset) Name() string { return d.name }
func (d *gaiaDataset) Len() int     { return len(d.samples) }
func (d *gaiaDataset) Iter(ctx context.Context) iter.Seq[rl.Sample] {
	return func(yield func(rl.Sample) bool) {
		for _, s := range d.samples {
			select {
			case <-ctx.Done():
				return
			default:
			}
			if !yield(s.Sample) {
				return
			}
		}
	}
}

// SamplesIter exposes GAIASamples (with Level) for evaluators.
func (d *gaiaDataset) SamplesIter(ctx context.Context) iter.Seq[GAIASample] {
	return func(yield func(GAIASample) bool) {
		for _, s := range d.samples {
			select {
			case <-ctx.Done():
				return
			default:
			}
			if !yield(s) {
				return
			}
		}
	}
}

// GAIAEvaluator computes per-level accuracy + drop rates.
type GAIAEvaluator struct {
	agent agents.Agent
	opts  rl.EvaluatorOptions
}

// NewGAIAEvaluator constructs a GAIA evaluator.
func NewGAIAEvaluator(agent agents.Agent, opts rl.EvaluatorOptions) *GAIAEvaluator {
	if opts.Concurrency <= 0 {
		opts.Concurrency = 1
	}
	return &GAIAEvaluator{agent: agent, opts: opts}
}

// GAIAMetrics extends rl.Metrics with GAIA-specific fields.
type GAIAMetrics struct {
	rl.Metrics
	ExactMatchRate float64
	PerLevel       map[int]float64
	DropRate12     float64 // (Acc_L1 - Acc_L2) / Acc_L1
	DropRate23     float64
}

// Run evaluates the GAIA dataset using Quasi-Exact-Match scoring.
func (g *GAIAEvaluator) Run(ctx context.Context, ds rl.Dataset) (GAIAMetrics, []rl.Trajectory, error) {
	gds, isGAIA := ds.(*gaiaDataset)
	if !isGAIA {
		ev := rl.NewEvaluator(g.agent, noopReward{}, g.opts)
		base, trajs, err := ev.Run(ctx, ds)
		return GAIAMetrics{Metrics: base}, trajs, err
	}

	trajs := make([]rl.Trajectory, 0, gds.Len())
	perLevelHits := make(map[int]int)
	perLevelTotal := make(map[int]int)
	overallHits := 0
	for s := range gds.SamplesIter(ctx) {
		select {
		case <-ctx.Done():
			return GAIAMetrics{}, trajs, ctx.Err()
		default:
		}
		traj := runForSample(ctx, g.agent, s.Sample, g.opts.Timeout)
		if quasiExactMatch(traj.Answer, s.GroundTruth) {
			traj.Reward = 1
			overallHits++
			perLevelHits[s.Level]++
		}
		perLevelTotal[s.Level]++
		trajs = append(trajs, traj)
	}

	out := GAIAMetrics{
		Metrics:  rl.Metrics{SampleCount: len(trajs)},
		PerLevel: make(map[int]float64),
	}
	if len(trajs) > 0 {
		out.ExactMatchRate = float64(overallHits) / float64(len(trajs))
		out.Accuracy = out.ExactMatchRate
		out.PassAtK = out.ExactMatchRate
	}
	for lvl, total := range perLevelTotal {
		if total > 0 {
			out.PerLevel[lvl] = float64(perLevelHits[lvl]) / float64(total)
		}
	}
	if a1, ok := out.PerLevel[1]; ok && a1 > 0 {
		if a2, ok := out.PerLevel[2]; ok {
			out.DropRate12 = (a1 - a2) / a1
		}
	}
	if a2, ok := out.PerLevel[2]; ok && a2 > 0 {
		if a3, ok := out.PerLevel[3]; ok {
			out.DropRate23 = (a2 - a3) / a2
		}
	}
	return out, trajs, nil
}

// quasiExactMatch normalizes both sides then compares. Handles digits
// (commas + periods normalized), text (lowercased + articles stripped
// + punctuation collapsed + whitespace normalized), lists (split-by-
// comma + element-wise normalize).
func quasiExactMatch(pred, truth string) bool {
	pn := normalizeQEM(pred)
	tn := normalizeQEM(truth)
	if pn == tn {
		return true
	}
	// List comparison: split + compare as multisets
	pl := splitList(pn)
	tl := splitList(tn)
	if len(pl) != len(tl) {
		return false
	}
	sort.Strings(pl)
	sort.Strings(tl)
	for i := range pl {
		if pl[i] != tl[i] {
			return false
		}
	}
	return true
}

// normalizeQEM lowercases, removes thousands-separators, drops articles
// + punctuation, collapses ws. Comma deletion happens BEFORE punct
// stripping so "1,234" → "1234" (not "1 234"). Trailing period
// stripped at the end so sentence-style answers ("paris.") match the
// raw token form ("paris").
func normalizeQEM(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, ",", "") // remove thousands separators first
	s = stripPunct(s)
	s = stripArticles(s)
	s = collapseWS(s)
	s = strings.TrimSpace(s)
	s = strings.TrimRight(s, ".")
	return s
}

func stripPunct(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == ' ' || r == ':' {
			b.WriteRune(r)
		} else {
			b.WriteByte(' ')
		}
	}
	return b.String()
}

func stripArticles(s string) string {
	tokens := strings.Fields(s)
	out := tokens[:0]
	for _, t := range tokens {
		switch t {
		case "a", "an", "the":
			continue
		}
		out = append(out, t)
	}
	return strings.Join(out, " ")
}

func collapseWS(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func splitList(s string) []string {
	parts := strings.Split(s, ":")
	if len(parts) > 1 {
		// "4: jupiter, saturn, ..." → drop "4"
		s = parts[len(parts)-1]
	}
	out := strings.Split(s, " ")
	cleaned := out[:0]
	for _, p := range out {
		p = strings.TrimSpace(p)
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}
	return cleaned
}

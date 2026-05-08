package bench

import (
	"context"
	stderrors "errors"
	"fmt"
	"iter"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/costa92/llm-agent"
	"github.com/costa92/llm-agent/rl"
)

// FunctionCall represents one parsed function invocation. Args is
// keyed by argument name; values are string / number / bool — no
// nested objects (BFCL fixture style).
type FunctionCall struct {
	Name string
	Args map[string]any
}

// BFCLSample carries one BFCL row + the expected function calls.
// Embeds rl.Sample so it works in any rl.Dataset.
type BFCLSample struct {
	rl.Sample
	Category      string
	ExpectedCalls []FunctionCall
}

// rlSample constructs an embedded rl.Sample. Internal helper for
// fixtures.go — keeps the file readable.
func rlSample(id, question, groundTruthCalls string) rl.Sample {
	return rl.Sample{
		ID:          id,
		Prompt:      question,
		GroundTruth: groundTruthCalls,
	}
}

// bfclDataset wraps a slice of BFCLSamples behind the rl.Dataset
// interface. Categories survive via embedding.
type bfclDataset struct {
	name    string
	samples []BFCLSample
}

// NewBFCLDataset wraps the provided samples in an rl.Dataset.
func NewBFCLDataset(name string, samples []BFCLSample) rl.Dataset {
	return &bfclDataset{name: name, samples: samples}
}

// MiniBFCL returns the built-in 5-sample fixture as an rl.Dataset.
func MiniBFCL() rl.Dataset {
	return NewBFCLDataset("bfcl-mini", MiniBFCLSamples())
}

// LoadBFCLFromJSONL is reserved for users who download real BFCL data.
// Not implemented in Phase 7 (would require BFCL's exact JSON schema);
// returns ErrBFCLLoaderNotImplemented until someone wires it.
func LoadBFCLFromJSONL(_ string) (rl.Dataset, error) {
	return nil, ErrBFCLLoaderNotImplemented
}

// ErrBFCLLoaderNotImplemented is returned by LoadBFCLFromJSONL.
var ErrBFCLLoaderNotImplemented = stderrors.New("bench: real BFCL loader is out of scope for Phase 7; build your own LineParser via rl.NewJSONLDataset")

func (d *bfclDataset) Name() string { return d.name }
func (d *bfclDataset) Len() int     { return len(d.samples) }
func (d *bfclDataset) Iter(ctx context.Context) iter.Seq[rl.Sample] {
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

// SamplesIter exposes BFCLSamples (with Category + ExpectedCalls)
// for evaluators that need the BFCL-specific fields.
func (d *bfclDataset) SamplesIter(ctx context.Context) iter.Seq[BFCLSample] {
	return func(yield func(BFCLSample) bool) {
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

// BFCLEvaluator wraps rl.Evaluator with BFCL-specific match logic.
type BFCLEvaluator struct {
	agent agents.Agent
	opts  rl.EvaluatorOptions
}

// NewBFCLEvaluator constructs a BFCL evaluator around an agent.
func NewBFCLEvaluator(agent agents.Agent, opts rl.EvaluatorOptions) *BFCLEvaluator {
	if opts.Concurrency <= 0 {
		opts.Concurrency = 1
	}
	return &BFCLEvaluator{agent: agent, opts: opts}
}

// BFCLMetrics extends rl.Metrics with BFCL-specific accuracy fields.
type BFCLMetrics struct {
	rl.Metrics
	OverallAccuracy  float64
	PerCategory      map[string]float64
	WeightedAccuracy float64 // weighted by category sample count
}

// Run evaluates the BFCL dataset. ds may be a *bfclDataset (uses
// per-sample Category info) or a generic rl.Dataset (Category-aware
// fields will be empty).
func (b *BFCLEvaluator) Run(ctx context.Context, ds rl.Dataset) (BFCLMetrics, []rl.Trajectory, error) {
	bds, isBFCL := ds.(*bfclDataset)
	if !isBFCL {
		// Fallback: just delegate to rl.Evaluator with no-op reward.
		ev := rl.NewEvaluator(b.agent, noopReward{}, b.opts)
		base, trajs, err := ev.Run(ctx, ds)
		return BFCLMetrics{Metrics: base}, trajs, err
	}

	// BFCL-aware path: agent runs each sample, we parse its answer and
	// match against ExpectedCalls.
	trajs := make([]rl.Trajectory, 0, bds.Len())
	perCatHits := make(map[string]int)
	perCatTotal := make(map[string]int)
	for s := range bds.SamplesIter(ctx) {
		select {
		case <-ctx.Done():
			return BFCLMetrics{}, trajs, ctx.Err()
		default:
		}
		traj := runForSample(ctx, b.agent, s.Sample, b.opts.Timeout)
		// Score: match parsed calls against expected.
		matched := matchExpected(traj.Answer, s.ExpectedCalls)
		if matched {
			traj.Reward = 1
			perCatHits[s.Category]++
		}
		perCatTotal[s.Category]++
		trajs = append(trajs, traj)
	}

	overall := 0
	for _, h := range perCatHits {
		overall += h
	}
	totalSamples := 0
	for _, t := range perCatTotal {
		totalSamples += t
	}
	out := BFCLMetrics{
		Metrics:     rl.Metrics{SampleCount: len(trajs)},
		PerCategory: make(map[string]float64),
	}
	if totalSamples > 0 {
		out.OverallAccuracy = float64(overall) / float64(totalSamples)
		out.Accuracy = out.OverallAccuracy
		out.PassAtK = out.OverallAccuracy
	}
	var weightedSum float64
	for cat, total := range perCatTotal {
		if total == 0 {
			continue
		}
		acc := float64(perCatHits[cat]) / float64(total)
		out.PerCategory[cat] = acc
		weightedSum += acc * float64(total)
	}
	if totalSamples > 0 {
		out.WeightedAccuracy = weightedSum / float64(totalSamples)
	}
	return out, trajs, nil
}

// matchExpected parses the predicted answer into FunctionCalls and
// matches them against the expected list (order-insensitive). Empty
// expected with empty parsed = match (irrelevance category).
func matchExpected(predAnswer string, expected []FunctionCall) bool {
	parsed := parseAllCalls(predAnswer)
	if len(expected) == 0 {
		return len(parsed) == 0
	}
	if len(parsed) != len(expected) {
		return false
	}
	// Order-insensitive: pair greedy on Name match
	used := make([]bool, len(parsed))
	for _, want := range expected {
		matchedIdx := -1
		for i, got := range parsed {
			if used[i] {
				continue
			}
			if matchFunctionCalls(got, want) {
				matchedIdx = i
				break
			}
		}
		if matchedIdx == -1 {
			return false
		}
		used[matchedIdx] = true
	}
	return true
}

// parseAllCalls splits predAnswer on commas at top level (no nested
// parens) and parses each chunk as one function call.
func parseAllCalls(s string) []FunctionCall {
	out := make([]FunctionCall, 0, 2)
	for _, chunk := range splitTopLevel(s, ',') {
		fc, err := parseSimpleCall(strings.TrimSpace(chunk))
		if err == nil && fc.Name != "" {
			out = append(out, fc)
		}
	}
	return out
}

// splitTopLevel splits s on sep at paren depth 0.
func splitTopLevel(s string, sep rune) []string {
	out := make([]string, 0, 4)
	depth := 0
	start := 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
		case sep:
			if depth == 0 {
				out = append(out, s[start:i])
				start = i + 1
			}
		}
	}
	out = append(out, s[start:])
	return out
}

// parseSimpleCall parses a single function call expression of the
// form: name(arg1=val1, arg2=val2). Values may be quoted strings,
// numbers, or true/false. No nested calls or expression args.
func parseSimpleCall(s string) (FunctionCall, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return FunctionCall{}, stderrors.New("bench: empty call expression")
	}
	open := strings.IndexByte(s, '(')
	if open < 0 || !strings.HasSuffix(s, ")") {
		return FunctionCall{}, fmt.Errorf("bench: malformed call %q", s)
	}
	name := strings.TrimSpace(s[:open])
	body := s[open+1 : len(s)-1]
	args := map[string]any{}
	if strings.TrimSpace(body) == "" {
		return FunctionCall{Name: name, Args: args}, nil
	}
	for _, part := range splitTopLevel(body, ',') {
		part = strings.TrimSpace(part)
		eq := strings.IndexByte(part, '=')
		if eq < 0 {
			return FunctionCall{}, fmt.Errorf("bench: arg without '=': %q", part)
		}
		key := strings.TrimSpace(part[:eq])
		raw := strings.TrimSpace(part[eq+1:])
		args[key] = parseLiteral(raw)
	}
	return FunctionCall{Name: name, Args: args}, nil
}

// parseLiteral coerces a token into string, int, float, bool.
// Quoted strings (single or double) lose their quotes.
func parseLiteral(s string) any {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"') {
			return s[1 : len(s)-1]
		}
	}
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return s
}

// matchFunctionCalls compares two FunctionCalls. Names must match
// exactly (case-sensitive); Args must match as a set (every key in
// pred is in truth with == values, and same len).
func matchFunctionCalls(pred, truth FunctionCall) bool {
	if pred.Name != truth.Name {
		return false
	}
	if len(pred.Args) != len(truth.Args) {
		return false
	}
	for k, v := range truth.Args {
		got, ok := pred.Args[k]
		if !ok {
			return false
		}
		if !literalEqual(got, v) {
			return false
		}
	}
	return true
}

func literalEqual(a, b any) bool {
	// Coerce all numeric types to float64 for comparison (so 25 == 25.0).
	if af, aok := toFloat(a); aok {
		if bf, bok := toFloat(b); bok {
			return af == bf
		}
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case float64:
		return x, true
	}
	return 0, false
}

// runForSample executes the agent on one sample under per-sample
// timeout (if non-zero). Errors are captured on the Trajectory rather
// than returned so partial-failure runs still produce metrics.
func runForSample(ctx context.Context, agent agents.Agent, s rl.Sample, timeout time.Duration) rl.Trajectory {
	runCtx := ctx
	cancel := func() {}
	if timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()
	start := time.Now()
	res, err := agent.Run(runCtx, s.Prompt)
	return rl.Trajectory{
		SampleID: s.ID,
		Prompt:   s.Prompt,
		Steps:    res.Trace,
		Answer:   res.Answer,
		Latency:  time.Since(start).Seconds(),
		Err:      err,
	}
}

// --- noopReward ----------------------------------------------------------

type noopReward struct{}

func (noopReward) Name() string { return "noop" }
func (noopReward) Score(_ context.Context, _ rl.Trajectory, _ rl.Sample) (float64, error) {
	return 0, nil
}

// sortedKeys returns map keys deterministically.
func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

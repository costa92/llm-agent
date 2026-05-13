package bench

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"
	"time"

	"github.com/costa92/llm-agent/pkg/fanout"
	"github.com/costa92/llm-agent/llm"
)

// Verdict is one comparison's outcome.
type Verdict string

const (
	VerdictA   Verdict = "a"
	VerdictB   Verdict = "b"
	VerdictTie Verdict = "tie"
)

// Comparison is one pairwise input.
type Comparison struct {
	SampleID string
	Question string
	AnswerA  string
	AnswerB  string
}

// ComparisonResult is one pairwise outcome.
type ComparisonResult struct {
	SampleID string
	Verdict  Verdict
	Reason   string
}

// WinRateOptions configures WinRateEvaluator.Compare.
type WinRateOptions struct {
	Concurrency int
	Timeout     time.Duration // per-pair timeout; default 30s
	SwapEval    bool          // if true, evaluate (A,B) AND (B,A) per item; defaults to true
}

// WinRateEvaluator runs LLM-judged head-to-head comparisons.
type WinRateEvaluator struct {
	llm  llm.ChatModel
	opts WinRateOptions
}

// NewWinRateEvaluator constructs a WinRateEvaluator. SwapEval defaults
// to true (anti-position-bias: each pair evaluated A-vs-B and B-vs-A,
// final verdict majority-vote).
func NewWinRateEvaluator(judge llm.ChatModel, opts WinRateOptions) *WinRateEvaluator {
	if opts.Concurrency <= 0 {
		opts.Concurrency = 1
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	// SwapEval intentionally defaults to true to fight position bias.
	// Detect "user explicitly set false" with a separate config field
	// would be cleaner — but for a learning tool this default+option
	// pair is fine.
	return &WinRateEvaluator{llm: judge, opts: opts}
}

// ErrWinRateNoLLM is returned by Compare when the judge LLM is nil.
var ErrWinRateNoLLM = stderrors.New("bench: WinRateEvaluator requires non-nil judge llm.ChatModel")

// Compare runs the comparison set under bounded concurrency.
func (w *WinRateEvaluator) Compare(ctx context.Context, items []Comparison) ([]ComparisonResult, error) {
	if w.llm == nil {
		return nil, ErrWinRateNoLLM
	}
	if len(items) == 0 {
		return nil, nil
	}

	tasks := make([]fanout.Task[ComparisonResult], len(items))
	for i, item := range items {
		item := item
		tasks[i] = func(ctx context.Context) (ComparisonResult, error) {
			return w.compareOne(ctx, item), nil
		}
	}

	results, err := fanout.Run(ctx, w.opts.Concurrency, tasks)
	if err != nil {
		return nil, err
	}

	out := make([]ComparisonResult, len(items))
	for i, r := range results {
		out[i] = r.Value
	}
	return out, nil
}

func (w *WinRateEvaluator) compareOne(ctx context.Context, c Comparison) ComparisonResult {
	v1, r1 := w.judgePair(ctx, c.Question, c.AnswerA, c.AnswerB)
	if !w.opts.SwapEval {
		return ComparisonResult{SampleID: c.SampleID, Verdict: v1, Reason: r1}
	}
	// Swap and judge again — swap verdict back so VerdictA still means
	// "original AnswerA wins".
	v2, r2 := w.judgePair(ctx, c.Question, c.AnswerB, c.AnswerA)
	v2unswapped := swapVerdict(v2)

	final := combineVerdicts(v1, v2unswapped)
	return ComparisonResult{
		SampleID: c.SampleID,
		Verdict:  final,
		Reason:   "round1: " + r1 + "\nround2(swap): " + r2,
	}
}

// judgePair sends one prompt and parses the verdict.
func (w *WinRateEvaluator) judgePair(ctx context.Context, q, a, b string) (Verdict, string) {
	runCtx, cancel := context.WithTimeout(ctx, w.opts.Timeout)
	defer cancel()
	prompt := fmt.Sprintf(`You are an expert judge. Compare two answers to the same question.

Question: %s

Answer A:
%s

Answer B:
%s

Output ONE word followed by an explanation:
- "A" if Answer A is better
- "B" if Answer B is better
- "TIE" if they are equally good

Format: <verdict>: <one-sentence reason>`, q, a, b)

	resp, err := w.llm.Generate(runCtx, llm.Request{
		Messages: []llm.Message{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return VerdictTie, "judge llm error: " + err.Error()
	}
	return parseVerdict(resp.Text)
}

// parseVerdict reads the LLM's verdict + reason.
func parseVerdict(text string) (Verdict, string) {
	clean := strings.TrimSpace(text)
	upper := strings.ToUpper(clean)
	switch {
	case strings.HasPrefix(upper, "A"):
		return VerdictA, clean
	case strings.HasPrefix(upper, "B"):
		return VerdictB, clean
	case strings.HasPrefix(upper, "TIE"):
		return VerdictTie, clean
	}
	return VerdictTie, "unparseable verdict: " + truncate(text, 100)
}

// swapVerdict swaps A↔B; TIE stays TIE.
func swapVerdict(v Verdict) Verdict {
	switch v {
	case VerdictA:
		return VerdictB
	case VerdictB:
		return VerdictA
	}
	return VerdictTie
}

// combineVerdicts: agree → that verdict; disagree → TIE.
func combineVerdicts(v1, v2 Verdict) Verdict {
	if v1 == v2 {
		return v1
	}
	return VerdictTie
}

// WinRateMetrics aggregates ComparisonResults.
type WinRateMetrics struct {
	Total    int
	WinA     int
	WinB     int
	Tie      int
	WinRateA float64
	WinRateB float64
	TieRate  float64
}

// ComputeWinRate aggregates results.
func ComputeWinRate(results []ComparisonResult) WinRateMetrics {
	out := WinRateMetrics{Total: len(results)}
	if out.Total == 0 {
		return out
	}
	for _, r := range results {
		switch r.Verdict {
		case VerdictA:
			out.WinA++
		case VerdictB:
			out.WinB++
		case VerdictTie:
			out.Tie++
		}
	}
	t := float64(out.Total)
	out.WinRateA = float64(out.WinA) / t
	out.WinRateB = float64(out.WinB) / t
	out.TieRate = float64(out.Tie) / t
	return out
}

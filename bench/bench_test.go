package bench

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/costa92/llm-agent/llm"
	"github.com/costa92/llm-agent"
	"github.com/costa92/llm-agent/rl"
)

// --- BFCL parser ---------------------------------------------------------

func TestParseSimpleCall_Plain(t *testing.T) {
	fc, err := parseSimpleCall(`get_weather(city='Beijing', units='c')`)
	if err != nil {
		t.Fatalf("parseSimpleCall: %v", err)
	}
	if fc.Name != "get_weather" {
		t.Errorf("Name = %q", fc.Name)
	}
	if fc.Args["city"] != "Beijing" || fc.Args["units"] != "c" {
		t.Errorf("Args = %+v", fc.Args)
	}
}

func TestParseSimpleCall_NoArgs(t *testing.T) {
	fc, err := parseSimpleCall(`get_time()`)
	if err != nil {
		t.Fatalf("parseSimpleCall: %v", err)
	}
	if fc.Name != "get_time" || len(fc.Args) != 0 {
		t.Errorf("got %+v", fc)
	}
}

func TestParseSimpleCall_NumericAndBool(t *testing.T) {
	fc, _ := parseSimpleCall(`compute(x=42, y=3.14, ok=true)`)
	if fc.Args["x"].(int64) != 42 {
		t.Errorf("x = %v (%T)", fc.Args["x"], fc.Args["x"])
	}
	if fc.Args["y"].(float64) != 3.14 {
		t.Errorf("y = %v", fc.Args["y"])
	}
	if fc.Args["ok"].(bool) != true {
		t.Errorf("ok = %v", fc.Args["ok"])
	}
}

func TestParseSimpleCall_Malformed(t *testing.T) {
	if _, err := parseSimpleCall(`broken`); err == nil {
		t.Error("expected error for missing parens")
	}
}

func TestMatchFunctionCalls_StrictNameAndArgs(t *testing.T) {
	a := FunctionCall{Name: "f", Args: map[string]any{"x": "1", "y": "2"}}
	b := FunctionCall{Name: "f", Args: map[string]any{"x": "1", "y": "2"}}
	if !matchFunctionCalls(a, b) {
		t.Error("identical calls should match")
	}
	c := FunctionCall{Name: "g", Args: map[string]any{"x": "1", "y": "2"}}
	if matchFunctionCalls(a, c) {
		t.Error("name mismatch should not match")
	}
	d := FunctionCall{Name: "f", Args: map[string]any{"x": "1"}}
	if matchFunctionCalls(a, d) {
		t.Error("arg-count mismatch should not match")
	}
}

func TestMatchFunctionCalls_NumericCoercion(t *testing.T) {
	a := FunctionCall{Name: "f", Args: map[string]any{"n": int64(42)}}
	b := FunctionCall{Name: "f", Args: map[string]any{"n": float64(42)}}
	if !matchFunctionCalls(a, b) {
		t.Error("int64 42 should match float64 42")
	}
}

// --- BFCL Evaluator -------------------------------------------------------

type bfclStubAgent struct {
	answers map[string]string
}

func (a *bfclStubAgent) Name() string { return "stub" }
func (a *bfclStubAgent) Run(_ context.Context, prompt string) (agents.Result, error) {
	for prefix, ans := range a.answers {
		if strings.Contains(prompt, prefix) {
			return agents.Result{Answer: ans}, nil
		}
	}
	return agents.Result{Answer: ""}, nil
}
func (a *bfclStubAgent) RunStream(_ context.Context, _ string) (<-chan agents.StepEvent, error) {
	return nil, errors.New("nope")
}

func TestBFCLEvaluator_AccuracyAcrossCategories(t *testing.T) {
	stub := &bfclStubAgent{answers: map[string]string{
		"Beijing":      `get_weather(city='Beijing', units='c')`, // hit
		"25 * 4":       `calculate(expression='25*4')`,            // hit
		"go modules":   `search(query='wrong')`,                    // miss
		"a joke":       `tell_joke()`,                              // miss (irrelevance)
		"alice":        `send_email(to='alice@example.com', subject='hi')`, // hit
	}}
	ev := NewBFCLEvaluator(stub, rl.EvaluatorOptions{Timeout: 2 * time.Second})
	m, _, err := ev.Run(context.Background(), MiniBFCL())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if m.SampleCount != 5 {
		t.Errorf("SampleCount = %d, want 5", m.SampleCount)
	}
	if m.OverallAccuracy < 0.5 || m.OverallAccuracy > 0.7 {
		t.Errorf("OverallAccuracy = %f, want ~0.6 (3/5)", m.OverallAccuracy)
	}
	if _, ok := m.PerCategory["simple"]; !ok {
		t.Errorf("missing 'simple' category in %+v", m.PerCategory)
	}
}

// --- GAIA QEM -------------------------------------------------------------

func TestQuasiExactMatch_Basic(t *testing.T) {
	if !quasiExactMatch("Paris", "paris") {
		t.Error("case-insensitive match should pass")
	}
	// QEM normalizes punctuation/articles in BOTH sides — symmetric.
	if !quasiExactMatch("The Paris.", "the paris") {
		t.Error("article + punctuation strip should match symmetrically")
	}
	if quasiExactMatch("London", "Paris") {
		t.Error("different cities should not match")
	}
}

func TestQuasiExactMatch_NumberWithComma(t *testing.T) {
	if !quasiExactMatch("1,234", "1234") {
		t.Error("comma normalization should match")
	}
}

func TestQuasiExactMatch_ListAsMultiset(t *testing.T) {
	if !quasiExactMatch("4: Jupiter, Saturn, Uranus, Neptune", "4: neptune jupiter saturn uranus") {
		t.Error("list multiset comparison should match")
	}
}

// --- GAIA Evaluator -------------------------------------------------------

type gaiaStubAgent struct {
	answers map[string]string
}

func (a *gaiaStubAgent) Name() string { return "stub" }
func (a *gaiaStubAgent) Run(_ context.Context, prompt string) (agents.Result, error) {
	for prefix, ans := range a.answers {
		if strings.Contains(prompt, prefix) {
			return agents.Result{Answer: ans}, nil
		}
	}
	return agents.Result{Answer: ""}, nil
}
func (a *gaiaStubAgent) RunStream(_ context.Context, _ string) (<-chan agents.StepEvent, error) {
	return nil, errors.New("nope")
}

func TestGAIAEvaluator_PerLevelAccuracy(t *testing.T) {
	stub := &gaiaStubAgent{answers: map[string]string{
		"capital of France": "Paris",                                    // L1 hit
		"gas giants":        "4: Jupiter, Saturn, Uranus, Neptune",       // L2 hit
		"product":           "wrong",                                     // L3 miss
	}}
	ev := NewGAIAEvaluator(stub, rl.EvaluatorOptions{Timeout: 2 * time.Second})
	m, _, err := ev.Run(context.Background(), MiniGAIA())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if m.SampleCount != 3 {
		t.Errorf("SampleCount = %d, want 3", m.SampleCount)
	}
	if m.PerLevel[1] != 1 {
		t.Errorf("L1 acc = %f, want 1", m.PerLevel[1])
	}
	if m.PerLevel[3] != 0 {
		t.Errorf("L3 acc = %f, want 0", m.PerLevel[3])
	}
	// DropRate12 = (1 - 1) / 1 = 0
	if m.DropRate12 != 0 {
		t.Errorf("DropRate12 = %f, want 0", m.DropRate12)
	}
}

// --- Judge ----------------------------------------------------------------

type fakeLLM struct {
	mu      sync.Mutex
	calls   int
	respond func(int) string
	err     error
}

func (f *fakeLLM) Generate(_ context.Context, _ llm.Request) (llm.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.err != nil {
		return llm.Response{}, f.err
	}
	return llm.Response{Text: f.respond(f.calls)}, nil
}
func (f *fakeLLM) Stream(_ context.Context, _ llm.Request) (llm.StreamReader, error) {
	return nil, errors.New("nope")
}
func (f *fakeLLM) Info() llm.ProviderInfo { return llm.ProviderInfo{} }

func TestJudge_RequiresLLM(t *testing.T) {
	j := NewJudge(nil, []JudgeRubric{{Name: "x", Scale: 5}}, JudgeOptions{})
	_, err := j.Evaluate(context.Background(), []JudgeItem{{SampleID: "1"}})
	if !errors.Is(err, ErrJudgeNoLLM) {
		t.Errorf("err = %v, want ErrJudgeNoLLM", err)
	}
}

func TestJudge_ParsesScores(t *testing.T) {
	stub := &fakeLLM{respond: func(_ int) string {
		return "```json\n{\"scores\":{\"correctness\":4,\"clarity\":5},\"reasoning\":\"good\"}\n```"
	}}
	j := NewJudge(stub, []JudgeRubric{{Name: "correctness", Scale: 5}, {Name: "clarity", Scale: 5}}, JudgeOptions{})
	scores, err := j.Evaluate(context.Background(), []JudgeItem{
		{SampleID: "s1", Question: "q", Answer: "a"},
	})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(scores) != 1 {
		t.Fatalf("got %d scores, want 1", len(scores))
	}
	if scores[0].Scores["correctness"] != 4 || scores[0].Scores["clarity"] != 5 {
		t.Errorf("scores = %+v", scores[0].Scores)
	}
	if scores[0].AvgScore != 4.5 {
		t.Errorf("avg = %f, want 4.5", scores[0].AvgScore)
	}
}

func TestComputeJudgeMetrics(t *testing.T) {
	scores := []JudgeScore{
		{Scores: map[string]int{"a": 5, "b": 5}, AvgScore: 5},
		{Scores: map[string]int{"a": 3, "b": 4}, AvgScore: 3.5},
		{Scores: map[string]int{"a": 1, "b": 2}, AvgScore: 1.5},
	}
	m := ComputeJudgeMetrics(scores, 5)
	if m.Total != 3 {
		t.Errorf("Total = %d", m.Total)
	}
	if m.PassRate < 0.6 || m.PassRate > 0.7 {
		t.Errorf("PassRate = %f, want 0.667 (2/3 ≥ 3.5)", m.PassRate)
	}
	if m.PerRubric["a"] != 3 {
		t.Errorf("PerRubric[a] = %f, want 3", m.PerRubric["a"])
	}
}

// --- WinRate --------------------------------------------------------------

func TestWinRate_RequiresLLM(t *testing.T) {
	w := NewWinRateEvaluator(nil, WinRateOptions{})
	_, err := w.Compare(context.Background(), []Comparison{{SampleID: "x"}})
	if !errors.Is(err, ErrWinRateNoLLM) {
		t.Errorf("err = %v, want ErrWinRateNoLLM", err)
	}
}

func TestWinRate_AnswerAWinsBothRounds(t *testing.T) {
	stub := &fakeLLM{respond: func(call int) string {
		// In round 2, A and B are swapped — we want the same OBJECTIVE answer
		// to win, which means the LLM picks the swapped position. So both
		// rounds need to return the position of the original A:
		// round 1: A is in slot A → "A: better"
		// round 2: A is in slot B → "B: better"
		if call%2 == 1 {
			return "A: better"
		}
		return "B: better"
	}}
	w := NewWinRateEvaluator(stub, WinRateOptions{SwapEval: true})
	results, err := w.Compare(context.Background(), []Comparison{
		{SampleID: "1", Question: "q", AnswerA: "alpha", AnswerB: "beta"},
	})
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if len(results) != 1 || results[0].Verdict != VerdictA {
		t.Errorf("verdict = %v, want A (consistent across swap)", results[0].Verdict)
	}
}

func TestWinRate_DisagreementBecomesTie(t *testing.T) {
	stub := &fakeLLM{respond: func(_ int) string { return "A: better" }} // always picks slot A
	w := NewWinRateEvaluator(stub, WinRateOptions{SwapEval: true})
	results, _ := w.Compare(context.Background(), []Comparison{
		{SampleID: "1", Question: "q", AnswerA: "alpha", AnswerB: "beta"},
	})
	// Round 1: VerdictA (alpha wins)
	// Round 2 (swap): VerdictA = beta wins; unswapped = VerdictB
	// Combined: A vs B → TIE
	if results[0].Verdict != VerdictTie {
		t.Errorf("verdict = %v, want TIE (positional bias detected)", results[0].Verdict)
	}
}

func TestComputeWinRate(t *testing.T) {
	results := []ComparisonResult{
		{Verdict: VerdictA},
		{Verdict: VerdictA},
		{Verdict: VerdictB},
		{Verdict: VerdictTie},
	}
	m := ComputeWinRate(results)
	if m.Total != 4 || m.WinA != 2 || m.WinB != 1 || m.Tie != 1 {
		t.Errorf("counts wrong: %+v", m)
	}
	if m.WinRateA != 0.5 {
		t.Errorf("WinRateA = %f, want 0.5", m.WinRateA)
	}
}

// --- Reporter -------------------------------------------------------------

func TestReporter_WriteAccuracyBar(t *testing.T) {
	var buf bytes.Buffer
	r := NewReporter("t", &buf)
	r.WriteAccuracyBar("acc", 0.5, 10)
	out := buf.String()
	if !strings.Contains(out, "█████░░░░░") || !strings.Contains(out, "50.00%") {
		t.Errorf("bar output unexpected: %q", out)
	}
}

func TestReporter_WriteTable(t *testing.T) {
	var buf bytes.Buffer
	r := NewReporter("t", &buf)
	r.WriteTable([]string{"A", "B"}, [][]string{{"x", "1"}, {"y", "2"}})
	out := buf.String()
	if !strings.Contains(out, "| A | B |") || !strings.Contains(out, "| --- | --- |") {
		t.Errorf("table format wrong: %q", out)
	}
}

func TestReporter_WriteJSON(t *testing.T) {
	var buf bytes.Buffer
	r := NewReporter("t", &buf)
	if err := r.WriteJSON("payload", map[string]int{"a": 1}); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	if !strings.Contains(buf.String(), "```json") || !strings.Contains(buf.String(), `"a": 1`) {
		t.Errorf("json block wrong: %q", buf.String())
	}
}

func TestExportJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := ExportJSON(&buf, map[string]string{"k": "v"}); err != nil {
		t.Fatalf("ExportJSON: %v", err)
	}
	if !strings.Contains(buf.String(), `"k": "v"`) {
		t.Errorf("json missing: %q", buf.String())
	}
}

func TestRenderBFCLReport(t *testing.T) {
	var buf bytes.Buffer
	m := BFCLMetrics{
		Metrics:          rl.Metrics{SampleCount: 5},
		OverallAccuracy:  0.6,
		WeightedAccuracy: 0.6,
		PerCategory:      map[string]float64{"simple": 1.0, "irrelevance": 0.0},
	}
	RenderBFCLReport(&buf, "test-agent", m, []rl.Trajectory{{SampleID: "x", Reward: 0, Answer: "wrong"}})
	out := buf.String()
	for _, want := range []string{"BFCL Evaluation Report", "test-agent", "60.00%", "Failed samples"} {
		if !strings.Contains(out, want) {
			t.Errorf("BFCL report missing %q\n%s", want, out)
		}
	}
}

// ctxAwareFakeLLM 是 fakeLLM 的 ctx-aware 变体:Generate 在 ctx done 时返回 ctx.Err。
// 用来测试 Judge.Evaluate / WinRateEvaluator.Compare 在 outer ctx cancel 时的行为。
type ctxAwareFakeLLM struct{}

func (f *ctxAwareFakeLLM) Generate(ctx context.Context, _ llm.Request) (llm.Response, error) {
	select {
	case <-ctx.Done():
		return llm.Response{}, ctx.Err()
	default:
	}
	return llm.Response{Text: `{"scores":{"x":1},"reasoning":"ok"}`}, nil
}
func (f *ctxAwareFakeLLM) Stream(_ context.Context, _ llm.Request) (llm.StreamReader, error) {
	return nil, errors.New("ctxAwareFakeLLM: stream not implemented")
}
func (f *ctxAwareFakeLLM) Info() llm.ProviderInfo { return llm.ProviderInfo{} }

func TestJudgeEvaluate_OuterCtxCancelReturnsErr(t *testing.T) {
	stub := &ctxAwareFakeLLM{}
	j := NewJudge(stub, []JudgeRubric{{Name: "x", Scale: 5}}, JudgeOptions{Concurrency: 2})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 已取消

	scores, err := j.Evaluate(ctx, []JudgeItem{
		{SampleID: "s1", Question: "q", Answer: "a"},
		{SampleID: "s2", Question: "q", Answer: "a"},
		{SampleID: "s3", Question: "q", Answer: "a"},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want errors.Is context.Canceled", err)
	}
	if scores != nil {
		t.Errorf("scores = %v, want nil (Evaluate must discard partial)", scores)
	}
}

func TestWinRateCompare_OuterCtxCancelReturnsErr(t *testing.T) {
	stub := &ctxAwareFakeLLM{}
	w := NewWinRateEvaluator(stub, WinRateOptions{Concurrency: 2})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 已取消

	results, err := w.Compare(ctx, []Comparison{
		{SampleID: "p1", Question: "q", AnswerA: "a", AnswerB: "b"},
		{SampleID: "p2", Question: "q", AnswerA: "a", AnswerB: "b"},
		{SampleID: "p3", Question: "q", AnswerA: "a", AnswerB: "b"},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want errors.Is context.Canceled", err)
	}
	if results != nil {
		t.Errorf("results = %v, want nil (Compare must discard partial)", results)
	}
}
